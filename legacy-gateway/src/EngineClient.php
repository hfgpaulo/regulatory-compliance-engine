<?php

declare(strict_types=1);

namespace LegacyGateway;

use GuzzleHttp\Client;
use GuzzleHttp\ClientInterface;
use GuzzleHttp\Exception\ConnectException;
use GuzzleHttp\Exception\GuzzleException;

/**
 * Cliente HTTP do regulatory-engine. Converte as respostas do motor em
 * retorno ou em exceções do domínio do gateway, para o handler não conhecer
 * Guzzle nem códigos HTTP do motor.
 */
final class EngineClient
{
    /** Prazo para abrir a conexão com o motor. */
    private const CONNECT_TIMEOUT_SECONDS = 2;

    /** Prazo total da chamada; acima do storeTimeout (5s) do motor, para não cortar uma gravação lenta. */
    private const TIMEOUT_SECONDS = 8;

    public function __construct(private readonly ClientInterface $http)
    {
    }

    public static function fromBaseUrl(string $baseUrl): self
    {
        return new self(new Client([
            'base_uri' => rtrim($baseUrl, '/') . '/',
            'connect_timeout' => self::CONNECT_TIMEOUT_SECONDS,
            'timeout' => self::TIMEOUT_SECONDS,
            // O status é tratado aqui; o Guzzle não deve lançar exceção por 4xx/5xx.
            'http_errors' => false,
        ]));
    }

    /**
     * @param array<string, mixed> $request requisição no contrato do motor
     *
     * @return array<string, mixed> avaliação criada
     *
     * @throws EngineValidationException  motor respondeu 400 com campos inválidos
     * @throws EngineUnavailableException motor fora do ar ou timeout
     * @throws EngineFailureException     motor respondeu erro ou fora do contrato
     */
    public function evaluate(array $request): array
    {
        try {
            $response = $this->http->request('POST', 'api/v1/evaluate', ['json' => $request]);
        } catch (ConnectException $e) {
            // Inclui conexão recusada, DNS e timeout: o motor não respondeu a tempo.
            throw new EngineUnavailableException('motor de conformidade indisponivel', previous: $e);
        } catch (GuzzleException $e) {
            throw new EngineFailureException('falha na chamada ao motor: ' . $e->getMessage(), previous: $e);
        }

        $status = $response->getStatusCode();
        $body = json_decode((string) $response->getBody(), true);

        if ($status === 201 && is_array($body)) {
            return $body;
        }

        if ($status === 400 && is_array($body) && is_array($body['details'] ?? null)) {
            throw new EngineValidationException(self::details($body['details']));
        }

        // Inclui 400 sem "details" (corpo que o gateway montou e o motor não
        // entendeu): é defeito de integração, não erro da proposta.
        throw new EngineFailureException(sprintf('motor respondeu status %d fora do contrato', $status));
    }

    /**
     * @param array<mixed> $details
     *
     * @return list<array{field: string, message: string}>
     */
    private static function details(array $details): array
    {
        $normalized = [];
        foreach ($details as $detail) {
            if (is_array($detail) && is_string($detail['field'] ?? null) && is_string($detail['message'] ?? null)) {
                $normalized[] = ['field' => $detail['field'], 'message' => $detail['message']];
            }
        }

        return $normalized;
    }
}
