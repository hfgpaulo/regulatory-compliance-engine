<?php

declare(strict_types=1);

namespace LegacyGateway\Tests;

use GuzzleHttp\Client;
use GuzzleHttp\Handler\MockHandler;
use GuzzleHttp\HandlerStack;
use GuzzleHttp\Middleware;
use GuzzleHttp\Psr7\Response;
use LegacyGateway\EngineClient;
use ArrayObject;
use Psr\Http\Message\ResponseInterface;
use Throwable;

/**
 * Simula o regulatory-engine com o MockHandler do Guzzle: cada chamada consome
 * a próxima resposta (ou exceção) da fila, sem rede. O histórico guarda as
 * requisições enviadas, para conferir o que o gateway mandou ao motor.
 */
final class FakeEngine
{
    /**
     * @param list<ResponseInterface|Throwable>                  $queue
     * @param ArrayObject<int, array<mixed>>|null                   $history
     */
    public static function client(array $queue, ?ArrayObject $history = null): EngineClient
    {
        $stack = HandlerStack::create(new MockHandler($queue));
        $container = $history ?? new ArrayObject();
        $stack->push(Middleware::history($container));

        return new EngineClient(new Client([
            'handler' => $stack,
            'base_uri' => 'http://engine.test/',
            'http_errors' => false,
        ]));
    }

    /**
     * Histórico vazio para passar ao client(); cada item tem 'request' e 'response'.
     *
     * @return ArrayObject<int, array<mixed>>
     */
    public static function history(): ArrayObject
    {
        return new ArrayObject();
    }

    /**
     * @param array<string, mixed> $body
     */
    public static function json(int $status, array $body): Response
    {
        return new Response($status, ['Content-Type' => 'application/json'], json_encode($body, JSON_THROW_ON_ERROR));
    }

    /**
     * Avaliação como o motor devolve no cenário C05 (US$ 75.000,00): IOF e
     * comunicação ao COAF.
     *
     * @return array<string, mixed>
     */
    public static function evaluationAboveThreshold(): array
    {
        return [
            'id' => '6aba9559710a953f1bbaea79',
            'created_at' => '2026-09-28T16:27:05.889Z',
            'rules_version' => 'sha256:cab7586760c88b7afad9779d7451e187abf1e2dc5fdb3fac2c9cf30a752dba4f',
            'request' => [],
            'report' => [
                'compliant' => false,
                'results' => [
                    ['domain' => 'IOF', 'status' => 'adaptation_required', 'detail' => 'IOF', 'calculated_amount' => '1425.00', 'reference' => 'rule:iof.international_transfer'],
                    ['domain' => 'PLD_COAF', 'status' => 'reportable', 'detail' => 'teto', 'calculated_amount' => '375000.00', 'reference' => 'rule:pld.reporting_threshold'],
                ],
                'required_adaptations' => [
                    'Incluir calculo e retencao de IOF no fluxo de entrada.',
                    'Gerar comunicacao ao COAF para a operacao.',
                ],
            ],
        ];
    }
}
