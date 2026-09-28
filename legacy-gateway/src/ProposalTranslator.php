<?php

declare(strict_types=1);

namespace LegacyGateway;

/**
 * Camada anticorrupção: traduz entre o formato do legado (campos em
 * português, enums em maiúsculas, centavos inteiros, pep "S"/"N") e o
 * contrato do motor.
 *
 * Valida só o que ele próprio traduz (tipos e enums do legado). As regras de
 * negócio (ISO, moeda suportada, contraparte obrigatória) ficam no motor, a
 * única fonte de verdade; os erros do motor voltam com os nomes do legado.
 */
final class ProposalTranslator
{
    private const PRODUCT_TYPES = ['EMPRESTIMO_PESSOAL' => 'personal_loan'];

    private const METHODS = ['TRANSFERENCIA_INTERNACIONAL' => 'international_transfer'];

    private const PEP_FLAGS = ['S' => true, 'N' => false];

    /** Nome do campo no motor => nome do campo no legado. */
    private const FIELD_NAMES = [
        'product.type' => 'produto.tipo',
        'product.origin' => 'produto.pais_origem',
        'product.origin_currency' => 'produto.moeda_origem',
        'operation.amount' => 'operacao.valor_centavos',
        'operation.currency' => 'operacao.moeda',
        'operation.method' => 'operacao.modalidade',
        'operation.counterparty.name' => 'operacao.contraparte.nome',
        'operation.counterparty.pep' => 'operacao.contraparte.pep',
    ];

    public const SITUATION_APPROVED = 'APROVADA';
    public const SITUATION_ADAPTATION = 'PENDENTE_ADAPTACAO';
    public const SITUATION_REPORT_COAF = 'COMUNICAR_COAF';

    /**
     * @return array<string, mixed> requisição no contrato do motor
     *
     * @throws InvalidProposalException
     */
    public function toEngineRequest(mixed $proposal): array
    {
        if (!is_array($proposal)) {
            throw new InvalidProposalException([['campo' => 'proposta', 'mensagem' => 'deve ser um objeto JSON']]);
        }

        $errors = [];

        $type = self::at($proposal, 'produto', 'tipo');
        if (!is_string($type) || !isset(self::PRODUCT_TYPES[$type])) {
            $errors[] = ['campo' => 'produto.tipo', 'mensagem' => 'tipo de produto desconhecido (aceito: EMPRESTIMO_PESSOAL)'];
        }

        $cents = self::at($proposal, 'operacao', 'valor_centavos');
        if (!is_int($cents) || $cents <= 0) {
            $errors[] = ['campo' => 'operacao.valor_centavos', 'mensagem' => 'deve ser inteiro positivo, em centavos'];
        }

        $method = self::at($proposal, 'operacao', 'modalidade');
        if (!is_string($method) || !isset(self::METHODS[$method])) {
            $errors[] = ['campo' => 'operacao.modalidade', 'mensagem' => 'modalidade desconhecida (aceita: TRANSFERENCIA_INTERNACIONAL)'];
        }

        $pep = self::at($proposal, 'operacao', 'contraparte', 'pep');
        if (!is_string($pep) || !isset(self::PEP_FLAGS[$pep])) {
            $errors[] = ['campo' => 'operacao.contraparte.pep', 'mensagem' => 'deve ser "S" ou "N"'];
        }

        $texts = [];
        foreach (['produto.pais_origem', 'produto.moeda_origem', 'operacao.moeda', 'operacao.contraparte.nome'] as $field) {
            $value = self::at($proposal, ...explode('.', $field));
            if ($value !== null && !is_string($value)) {
                $errors[] = ['campo' => $field, 'mensagem' => 'deve ser texto'];
            }
            // Ausente vira "": o motor decide se é obrigatório e o erro volta traduzido.
            $texts[$field] = is_string($value) ? $value : '';
        }

        if ($errors !== [] || !is_string($type) || !is_int($cents) || !is_string($method) || !is_string($pep)) {
            throw new InvalidProposalException($errors);
        }

        return [
            'product' => [
                'type' => self::PRODUCT_TYPES[$type],
                'origin' => $texts['produto.pais_origem'],
                'origin_currency' => $texts['produto.moeda_origem'],
            ],
            'operation' => [
                'amount' => Cents::toDecimal($cents),
                'currency' => $texts['operacao.moeda'],
                'method' => self::METHODS[$method],
                'counterparty' => [
                    'name' => $texts['operacao.contraparte.nome'],
                    'pep' => self::PEP_FLAGS[$pep],
                ],
            ],
        ];
    }

    /**
     * @param array<string, mixed> $evaluation avaliação devolvida pelo motor
     *
     * @return array<string, mixed> resposta no formato do legado
     */
    public function toLegacyResponse(array $evaluation): array
    {
        $results = self::at($evaluation, 'report', 'results');
        $results = is_array($results) ? $results : [];
        $adaptations = self::at($evaluation, 'report', 'required_adaptations');

        return [
            'protocolo' => $evaluation['id'] ?? null,
            'situacao' => self::situation($results),
            'iof_centavos' => self::iofCents($results),
            'exigencias' => is_array($adaptations) ? array_values($adaptations) : [],
            'versao_regras' => $evaluation['rules_version'] ?? null,
        ];
    }

    /**
     * Traduz os campos de um 400 do motor para o vocabulário do legado.
     * Um campo sem tradução conhecida mantém o nome do motor (melhor que sumir).
     *
     * @param list<array{field: string, message: string}> $details
     *
     * @return list<array{campo: string, mensagem: string}>
     */
    public function toLegacyFields(array $details): array
    {
        return array_map(
            static fn (array $detail): array => [
                'campo' => self::FIELD_NAMES[$detail['field']] ?? $detail['field'],
                'mensagem' => $detail['message'],
            ],
            $details,
        );
    }

    /**
     * Precedência: comunicação ao COAF pesa mais que adaptação, que pesa mais
     * que aprovação. O legado vê uma única situação, a mais grave.
     *
     * @param array<mixed> $results
     */
    private static function situation(array $results): string
    {
        $statuses = array_map(static fn (mixed $result): mixed => is_array($result) ? ($result['status'] ?? null) : null, $results);

        if (in_array('reportable', $statuses, true)) {
            return self::SITUATION_REPORT_COAF;
        }
        if (in_array('adaptation_required', $statuses, true)) {
            return self::SITUATION_ADAPTATION;
        }

        return self::SITUATION_APPROVED;
    }

    /**
     * @param array<mixed> $results
     */
    private static function iofCents(array $results): int
    {
        foreach ($results as $result) {
            if (is_array($result) && ($result['domain'] ?? null) === 'IOF' && is_string($result['calculated_amount'] ?? null)) {
                return Cents::fromDecimal($result['calculated_amount']);
            }
        }

        return 0;
    }

    /**
     * Lê um valor aninhado sem avisos quando algum nível não existe.
     *
     * @param array<mixed> $data
     */
    private static function at(array $data, string ...$path): mixed
    {
        $current = $data;
        foreach ($path as $key) {
            if (!is_array($current) || !array_key_exists($key, $current)) {
                return null;
            }
            $current = $current[$key];
        }

        return $current;
    }
}
