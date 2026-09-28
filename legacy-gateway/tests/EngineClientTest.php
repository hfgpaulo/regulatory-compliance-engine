<?php

declare(strict_types=1);

namespace LegacyGateway\Tests;

use GuzzleHttp\Exception\ConnectException;
use GuzzleHttp\Psr7\Request;
use GuzzleHttp\Psr7\Response;
use LegacyGateway\EngineFailureException;
use LegacyGateway\EngineUnavailableException;
use LegacyGateway\EngineValidationException;
use PHPUnit\Framework\TestCase;

final class EngineClientTest extends TestCase
{
    public function testReturnsEvaluationAndSendsTheRequestToTheEngine(): void
    {
        $history = FakeEngine::history();
        $client = FakeEngine::client([FakeEngine::json(201, FakeEngine::evaluationAboveThreshold())], $history);

        $evaluation = $client->evaluate(['product' => ['type' => 'personal_loan']]);

        self::assertSame('6aba9559710a953f1bbaea79', $evaluation['id']);
        self::assertCount(1, $history);
        $transaction = $history[0];
        self::assertIsArray($transaction);
        $sent = $transaction['request'] ?? null;
        self::assertInstanceOf(Request::class, $sent);
        self::assertSame('POST', $sent->getMethod());
        self::assertSame('/api/v1/evaluate', $sent->getUri()->getPath());
        self::assertSame('{"product":{"type":"personal_loan"}}', (string) $sent->getBody());
    }

    public function testEngineValidationErrorKeepsTheFields(): void
    {
        $client = FakeEngine::client([FakeEngine::json(400, [
            'error' => 'requisicao invalida',
            'details' => [['field' => 'operation.currency', 'message' => 'moeda nao suportada (aceita: USD)']],
        ])]);

        try {
            $client->evaluate([]);
            self::fail('deveria lancar EngineValidationException');
        } catch (EngineValidationException $e) {
            self::assertSame([['field' => 'operation.currency', 'message' => 'moeda nao suportada (aceita: USD)']], $e->details);
        }
    }

    public function testBadRequestWithoutDetailsIsAnIntegrationFailure(): void
    {
        $client = FakeEngine::client([FakeEngine::json(400, ['error' => 'corpo da requisicao invalido'])]);

        $this->expectException(EngineFailureException::class);
        $client->evaluate([]);
    }

    public function testServerErrorIsAFailure(): void
    {
        $client = FakeEngine::client([FakeEngine::json(500, ['error' => 'falha ao gravar a avaliacao'])]);

        $this->expectException(EngineFailureException::class);
        $client->evaluate([]);
    }

    public function testNonJsonResponseIsAFailure(): void
    {
        $client = FakeEngine::client([new Response(201, [], 'nao e json')]);

        $this->expectException(EngineFailureException::class);
        $client->evaluate([]);
    }

    public function testConnectionErrorOrTimeoutMeansUnavailable(): void
    {
        $client = FakeEngine::client([
            new ConnectException('cURL error 28: Operation timed out', new Request('POST', 'api/v1/evaluate')),
        ]);

        $this->expectException(EngineUnavailableException::class);
        $client->evaluate([]);
    }
}
