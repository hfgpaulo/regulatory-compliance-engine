<?php

declare(strict_types=1);

namespace LegacyGateway;

use JsonException;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Throwable;

/**
 * POST /propostas: recebe a proposta no formato legado, traduz, chama o motor,
 * registra a proposta aceita no banco legado e devolve a resposta no formato
 * legado. Só orquestra; a tradução fica no ProposalTranslator, a integração
 * HTTP no EngineClient e a gravação no ProposalRepository.
 *
 * Aceitas vão para o banco legado (e a avaliação completa fica no motor);
 * rejeições e falhas vão para o log, com o motivo e sem dados pessoais.
 */
final class CreateProposalAction
{
    public function __construct(
        private readonly EngineClient $engine,
        private readonly ProposalTranslator $translator,
        private readonly ProposalRepository $proposals,
        private readonly EventLog $log,
    ) {
    }

    public function __invoke(ServerRequestInterface $request, ResponseInterface $response): ResponseInterface
    {
        try {
            $proposal = json_decode((string) $request->getBody(), true, flags: JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            $this->rejected('gateway', [['campo' => 'proposta', 'mensagem' => 'corpo da requisicao nao e JSON valido']]);

            return self::json($response, 400, ['erro' => 'corpo da requisicao nao e JSON valido']);
        }

        try {
            $evaluation = $this->engine->evaluate($this->translator->toEngineRequest($proposal));
        } catch (InvalidProposalException $e) {
            $this->rejected('gateway', $e->fields);

            return self::json($response, 422, ['erro' => 'proposta invalida', 'campos' => $e->fields]);
        } catch (EngineValidationException $e) {
            $fields = $this->translator->toLegacyFields($e->details);
            $this->rejected('motor', $fields);

            return self::json($response, 422, ['erro' => 'proposta invalida', 'campos' => $fields]);
        } catch (EngineUnavailableException $e) {
            $this->log->record('motor_indisponivel', ['motivo' => $e->getMessage()]);

            return self::json($response, 503, ['erro' => 'motor de conformidade indisponivel']);
        } catch (EngineFailureException $e) {
            $this->log->record('falha_no_motor', ['motivo' => $e->getMessage()]);

            return self::json($response, 502, ['erro' => 'falha no motor de conformidade']);
        }

        $legacyResponse = $this->translator->toLegacyResponse($evaluation);

        // toEngineRequest só retorna para um array; a checagem é para o tipo estático.
        if (is_array($proposal)) {
            try {
                $this->proposals->save($this->translator->toRecord($proposal, $legacyResponse));
            } catch (Throwable $e) {
                // O motor já gravou a avaliação (fonte da verdade). Responder erro
                // levaria o cliente a reenviar e o motor criaria uma segunda
                // avaliação. A linha legada é reconciliável pelo protocolo.
                $this->log->record('gravacao_legado_falhou', [
                    'protocolo' => $legacyResponse['protocolo'],
                    'motivo' => $e->getMessage(),
                ]);
            }
        }

        return self::json($response, 201, $legacyResponse);
    }

    /**
     * Registra a rejeição só com campos e mensagens: a proposta tem dado
     * pessoal (nome da contraparte), que não deve ir para log (LGPD).
     *
     * @param list<array{campo: string, mensagem: string}> $fields
     */
    private function rejected(string $origin, array $fields): void
    {
        $this->log->record('proposta_rejeitada', ['origem' => $origin, 'campos' => $fields]);
    }

    /**
     * @param array<string, mixed> $payload
     */
    private static function json(ResponseInterface $response, int $status, array $payload): ResponseInterface
    {
        $response->getBody()->write(json_encode($payload, JSON_THROW_ON_ERROR | JSON_UNESCAPED_UNICODE));

        return $response->withStatus($status)->withHeader('Content-Type', 'application/json');
    }
}
