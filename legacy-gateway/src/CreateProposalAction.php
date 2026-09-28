<?php

declare(strict_types=1);

namespace LegacyGateway;

use JsonException;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;

/**
 * POST /propostas: recebe a proposta no formato legado, traduz, chama o motor
 * e devolve a resposta no formato legado. Só orquestra; a tradução fica no
 * ProposalTranslator e a integração HTTP no EngineClient.
 */
final class CreateProposalAction
{
    public function __construct(
        private readonly EngineClient $engine,
        private readonly ProposalTranslator $translator,
    ) {
    }

    public function __invoke(ServerRequestInterface $request, ResponseInterface $response): ResponseInterface
    {
        try {
            $proposal = json_decode((string) $request->getBody(), true, flags: JSON_THROW_ON_ERROR);
        } catch (JsonException) {
            return self::json($response, 400, ['erro' => 'corpo da requisicao nao e JSON valido']);
        }

        try {
            $evaluation = $this->engine->evaluate($this->translator->toEngineRequest($proposal));
        } catch (InvalidProposalException $e) {
            return self::json($response, 422, ['erro' => 'proposta invalida', 'campos' => $e->fields]);
        } catch (EngineValidationException $e) {
            return self::json($response, 422, ['erro' => 'proposta invalida', 'campos' => $this->translator->toLegacyFields($e->details)]);
        } catch (EngineUnavailableException $e) {
            error_log('legacy-gateway: ' . $e->getMessage());

            return self::json($response, 503, ['erro' => 'motor de conformidade indisponivel']);
        } catch (EngineFailureException $e) {
            error_log('legacy-gateway: ' . $e->getMessage());

            return self::json($response, 502, ['erro' => 'falha no motor de conformidade']);
        }

        return self::json($response, 201, $this->translator->toLegacyResponse($evaluation));
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
