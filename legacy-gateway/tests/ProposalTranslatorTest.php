<?php

declare(strict_types=1);

namespace LegacyGateway\Tests;

use LegacyGateway\InvalidProposalException;
use LegacyGateway\ProposalTranslator;
use PHPUnit\Framework\Attributes\DataProvider;
use PHPUnit\Framework\TestCase;

final class ProposalTranslatorTest extends TestCase
{
    /**
     * @return array<string, mixed>
     */
    public static function validProposal(): array
    {
        return [
            'produto' => ['tipo' => 'EMPRESTIMO_PESSOAL', 'pais_origem' => 'US', 'moeda_origem' => 'USD'],
            'operacao' => [
                'valor_centavos' => 7500000,
                'moeda' => 'USD',
                'modalidade' => 'TRANSFERENCIA_INTERNACIONAL',
                'contraparte' => ['nome' => 'John Doe', 'pep' => 'S'],
            ],
        ];
    }

    public function testTranslatesLegacyProposalToEngineRequest(): void
    {
        $request = (new ProposalTranslator())->toEngineRequest(self::validProposal());

        self::assertSame([
            'product' => ['type' => 'personal_loan', 'origin' => 'US', 'origin_currency' => 'USD'],
            'operation' => [
                'amount' => '75000.00',
                'currency' => 'USD',
                'method' => 'international_transfer',
                'counterparty' => ['name' => 'John Doe', 'pep' => true],
            ],
        ], $request);
    }

    public function testMissingTextFieldsAreDelegatedToTheEngine(): void
    {
        $proposal = self::validProposal();
        unset($proposal['operacao']['contraparte']['nome']);

        $request = (new ProposalTranslator())->toEngineRequest($proposal);

        self::assertSame('', $request['operation']['counterparty']['name'], 'o motor valida e o erro volta traduzido');
    }

    /**
     * @return iterable<string, array{callable(array<string, mixed>): array<string, mixed>, string}>
     */
    public static function invalidProposals(): iterable
    {
        yield 'tipo desconhecido' => [static function (array $p): array { $p['produto']['tipo'] = 'FINANCIAMENTO'; return $p; }, 'produto.tipo'];
        yield 'valor com decimais (float)' => [static function (array $p): array { $p['operacao']['valor_centavos'] = 75000.5; return $p; }, 'operacao.valor_centavos'];
        yield 'valor em texto' => [static function (array $p): array { $p['operacao']['valor_centavos'] = '7500000'; return $p; }, 'operacao.valor_centavos'];
        yield 'valor zero' => [static function (array $p): array { $p['operacao']['valor_centavos'] = 0; return $p; }, 'operacao.valor_centavos'];
        yield 'modalidade desconhecida' => [static function (array $p): array { $p['operacao']['modalidade'] = 'PIX'; return $p; }, 'operacao.modalidade'];
        yield 'pep booleano' => [static function (array $p): array { $p['operacao']['contraparte']['pep'] = true; return $p; }, 'operacao.contraparte.pep'];
        yield 'pep minusculo' => [static function (array $p): array { $p['operacao']['contraparte']['pep'] = 's'; return $p; }, 'operacao.contraparte.pep'];
        yield 'moeda numerica' => [static function (array $p): array { $p['operacao']['moeda'] = 840; return $p; }, 'operacao.moeda'];
    }

    /**
     * @param callable(array<string, mixed>): array<string, mixed> $mutate
     */
    #[DataProvider('invalidProposals')]
    public function testRejectsInvalidLegacyFormat(callable $mutate, string $field): void
    {
        try {
            (new ProposalTranslator())->toEngineRequest($mutate(self::validProposal()));
            self::fail('deveria rejeitar a proposta');
        } catch (InvalidProposalException $e) {
            self::assertSame([$field], array_column($e->fields, 'campo'));
        }
    }

    public function testRejectsNonObjectBody(): void
    {
        $this->expectException(InvalidProposalException::class);
        (new ProposalTranslator())->toEngineRequest('texto');
    }

    public function testTranslatesEvaluationToLegacyResponse(): void
    {
        $evaluation = FakeEngine::evaluationAboveThreshold();

        $response = (new ProposalTranslator())->toLegacyResponse($evaluation);

        self::assertSame([
            'protocolo' => '6aba9559710a953f1bbaea79',
            'situacao' => 'COMUNICAR_COAF',
            'iof_centavos' => 142500,
            'exigencias' => [
                'Incluir calculo e retencao de IOF no fluxo de entrada.',
                'Gerar comunicacao ao COAF para a operacao.',
            ],
            'versao_regras' => 'sha256:cab7586760c88b7afad9779d7451e187abf1e2dc5fdb3fac2c9cf30a752dba4f',
        ], $response);
    }

    /**
     * @return iterable<string, array{list<string>, string}>
     */
    public static function situations(): iterable
    {
        yield 'nenhum resultado' => [[], 'APROVADA'];
        yield 'so adaptacao' => [['adaptation_required'], 'PENDENTE_ADAPTACAO'];
        yield 'so comunicacao' => [['reportable'], 'COMUNICAR_COAF'];
        yield 'comunicacao prevalece sobre adaptacao' => [['adaptation_required', 'reportable'], 'COMUNICAR_COAF'];
    }

    /**
     * @param list<string> $statuses
     */
    #[DataProvider('situations')]
    public function testSituationPrecedence(array $statuses, string $expected): void
    {
        $results = array_map(static fn (string $status): array => ['domain' => 'PLD_COAF', 'status' => $status], $statuses);

        $response = (new ProposalTranslator())->toLegacyResponse(['id' => 'x', 'report' => ['results' => $results]]);

        self::assertSame($expected, $response['situacao']);
        self::assertSame(0, $response['iof_centavos'], 'sem resultado de IOF, o valor e zero');
    }

    public function testTranslatesEngineFieldNamesToLegacy(): void
    {
        $fields = (new ProposalTranslator())->toLegacyFields([
            ['field' => 'operation.currency', 'message' => 'moeda nao suportada (aceita: USD)'],
            ['field' => 'operation.counterparty.name', 'message' => 'obrigatorio'],
            ['field' => 'campo.novo', 'message' => 'x'],
        ]);

        self::assertSame([
            ['campo' => 'operacao.moeda', 'mensagem' => 'moeda nao suportada (aceita: USD)'],
            ['campo' => 'operacao.contraparte.nome', 'mensagem' => 'obrigatorio'],
            ['campo' => 'campo.novo', 'mensagem' => 'x'],
        ], $fields);
    }
}
