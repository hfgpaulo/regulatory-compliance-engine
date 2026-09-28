<?php

declare(strict_types=1);

namespace LegacyGateway\Tests;

use GuzzleHttp\Exception\ConnectException;
use GuzzleHttp\Psr7\Request;
use LegacyGateway\AppFactory;
use LegacyGateway\EngineClient;
use PHPUnit\Framework\TestCase;
use Psr\Http\Message\ResponseInterface;
use Slim\Psr7\Factory\ServerRequestFactory;
use Slim\Psr7\Factory\StreamFactory;

/**
 * Exercita POST /propostas pelo app Slim real (AppFactory), com o motor
 * simulado. Cobre o caminho feliz e cada tradução de erro.
 */
final class CreateProposalActionTest extends TestCase
{
    public function testCreatesProposalAndRespondsInLegacyFormat(): void
    {
        $history = FakeEngine::history();
        $engine = FakeEngine::client([FakeEngine::json(201, FakeEngine::evaluationAboveThreshold())], $history);

        $response = self::post($engine, json_encode(ProposalTranslatorTest::validProposal(), JSON_THROW_ON_ERROR));

        self::assertSame(201, $response->getStatusCode());
        self::assertSame('application/json', $response->getHeaderLine('Content-Type'));
        $body = self::body($response);
        self::assertSame('6aba9559710a953f1bbaea79', $body['protocolo']);
        self::assertSame('COMUNICAR_COAF', $body['situacao']);
        self::assertSame(142500, $body['iof_centavos']);
        self::assertCount(1, $history, 'o motor deve ser chamado uma vez');
    }

    public function testInvalidLegacyFormatReturns422WithoutCallingTheEngine(): void
    {
        $history = FakeEngine::history();
        $engine = FakeEngine::client([], $history);
        $proposal = ProposalTranslatorTest::validProposal();
        $proposal['operacao']['contraparte']['pep'] = 'talvez';

        $response = self::post($engine, json_encode($proposal, JSON_THROW_ON_ERROR));

        self::assertSame(422, $response->getStatusCode());
        self::assertSame(
            ['erro' => 'proposta invalida', 'campos' => [['campo' => 'operacao.contraparte.pep', 'mensagem' => 'deve ser "S" ou "N"']]],
            self::body($response),
        );
        self::assertCount(0, $history, 'proposta invalida no formato legado nao chega ao motor');
    }

    public function testEngineValidationErrorReturns422WithLegacyFieldNames(): void
    {
        $engine = FakeEngine::client([FakeEngine::json(400, [
            'error' => 'requisicao invalida',
            'details' => [['field' => 'operation.currency', 'message' => 'moeda nao suportada (aceita: USD)']],
        ])]);
        $proposal = ProposalTranslatorTest::validProposal();
        $proposal['operacao']['moeda'] = 'BRL';

        $response = self::post($engine, json_encode($proposal, JSON_THROW_ON_ERROR));

        self::assertSame(422, $response->getStatusCode());
        self::assertSame(
            ['erro' => 'proposta invalida', 'campos' => [['campo' => 'operacao.moeda', 'mensagem' => 'moeda nao suportada (aceita: USD)']]],
            self::body($response),
        );
    }

    public function testMalformedJsonReturns400(): void
    {
        $response = self::post(FakeEngine::client([]), '{ nao e json');

        self::assertSame(400, $response->getStatusCode());
    }

    public function testEngineUnavailableReturns503(): void
    {
        $engine = FakeEngine::client([new ConnectException('connection refused', new Request('POST', 'api/v1/evaluate'))]);

        $response = self::post($engine, json_encode(ProposalTranslatorTest::validProposal(), JSON_THROW_ON_ERROR));

        self::assertSame(503, $response->getStatusCode());
        self::assertSame(['erro' => 'motor de conformidade indisponivel'], self::body($response));
    }

    public function testEngineFailureReturns502WithoutLeakingDetails(): void
    {
        $engine = FakeEngine::client([FakeEngine::json(500, ['error' => 'falha ao gravar a avaliacao'])]);

        $response = self::post($engine, json_encode(ProposalTranslatorTest::validProposal(), JSON_THROW_ON_ERROR));

        self::assertSame(502, $response->getStatusCode());
        self::assertSame(['erro' => 'falha no motor de conformidade'], self::body($response));
    }

    private static function post(EngineClient $engine, string $body): ResponseInterface
    {
        $request = (new ServerRequestFactory())
            ->createServerRequest('POST', '/propostas')
            ->withHeader('Content-Type', 'application/json')
            ->withBody((new StreamFactory())->createStream($body));

        return AppFactory::create($engine)->handle($request);
    }

    /**
     * @return array<string, mixed>
     */
    private static function body(ResponseInterface $response): array
    {
        $decoded = json_decode((string) $response->getBody(), true, flags: JSON_THROW_ON_ERROR);
        self::assertIsArray($decoded);

        return $decoded;
    }
}
