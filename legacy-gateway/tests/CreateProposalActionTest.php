<?php

declare(strict_types=1);

namespace LegacyGateway\Tests;

use GuzzleHttp\Exception\ConnectException;
use GuzzleHttp\Psr7\Request;
use LegacyGateway\AppFactory;
use LegacyGateway\EngineClient;
use LegacyGateway\ProposalRecord;
use PHPUnit\Framework\TestCase;
use Psr\Http\Message\ResponseInterface;
use Slim\Psr7\Factory\ServerRequestFactory;
use Slim\Psr7\Factory\StreamFactory;

/**
 * Exercita POST /propostas pelo app Slim real (AppFactory), com o motor, o
 * banco legado e o log simulados. Cobre o caminho feliz, cada tradução de
 * erro, o que vai para o banco legado e o que vai para o log.
 */
final class CreateProposalActionTest extends TestCase
{
    private InMemoryProposalRepository $proposals;

    private InMemoryEventLog $log;

    protected function setUp(): void
    {
        $this->proposals = new InMemoryProposalRepository();
        $this->log = new InMemoryEventLog();
    }

    public function testCreatesProposalAndRespondsInLegacyFormat(): void
    {
        $history = FakeEngine::history();
        $engine = FakeEngine::client([FakeEngine::json(201, FakeEngine::evaluationAboveThreshold())], $history);

        $response = $this->post($engine, self::validBody());

        self::assertSame(201, $response->getStatusCode());
        self::assertSame('application/json', $response->getHeaderLine('Content-Type'));
        $body = self::body($response);
        self::assertSame('6aba9559710a953f1bbaea79', $body['protocolo']);
        self::assertSame('COMUNICAR_COAF', $body['situacao']);
        self::assertSame(142500, $body['iof_centavos']);
        self::assertCount(1, $history, 'o motor deve ser chamado uma vez');
        self::assertSame([], $this->log->events, 'proposta aceita nao gera evento de log');
    }

    public function testAcceptedProposalIsSavedInLegacyVocabulary(): void
    {
        $engine = FakeEngine::client([FakeEngine::json(201, FakeEngine::evaluationAboveThreshold())]);

        $this->post($engine, self::validBody());

        self::assertEquals([new ProposalRecord(
            protocolo: '6aba9559710a953f1bbaea79',
            tipoProduto: 'EMPRESTIMO_PESSOAL',
            modalidade: 'TRANSFERENCIA_INTERNACIONAL',
            valorCentavos: 7500000,
            moeda: 'USD',
            contraparteNome: 'John Doe',
            contrapartePep: 'S',
            situacao: 'COMUNICAR_COAF',
            iofCentavos: 142500,
            versaoRegras: 'sha256:cab7586760c88b7afad9779d7451e187abf1e2dc5fdb3fac2c9cf30a752dba4f',
        )], $this->proposals->saved);
    }

    public function testLegacyDatabaseFailureStillReturns201AndIsLogged(): void
    {
        $this->proposals = new InMemoryProposalRepository(failWith: 'SQLSTATE[HY000] [2002] Connection refused');
        $engine = FakeEngine::client([FakeEngine::json(201, FakeEngine::evaluationAboveThreshold())]);

        $response = $this->post($engine, self::validBody());

        self::assertSame(201, $response->getStatusCode(), 'o motor ja gravou a avaliacao; erro levaria a reenvio e avaliacao duplicada');
        self::assertSame(['gravacao_legado_falhou'], $this->log->names());
        self::assertSame('6aba9559710a953f1bbaea79', $this->log->events[0]['context']['protocolo'], 'o protocolo permite reconciliar');
    }

    public function testInvalidLegacyFormatReturns422WithoutCallingTheEngine(): void
    {
        $history = FakeEngine::history();
        $engine = FakeEngine::client([], $history);
        $proposal = ProposalTranslatorTest::validProposal();
        $proposal['operacao']['contraparte']['pep'] = 'talvez';

        $response = $this->post($engine, json_encode($proposal, JSON_THROW_ON_ERROR));

        self::assertSame(422, $response->getStatusCode());
        self::assertSame(
            ['erro' => 'proposta invalida', 'campos' => [['campo' => 'operacao.contraparte.pep', 'mensagem' => 'deve ser "S" ou "N"']]],
            self::body($response),
        );
        self::assertCount(0, $history, 'proposta invalida no formato legado nao chega ao motor');
        self::assertSame([], $this->proposals->saved, 'rejeitada nao vai para o banco legado');
        self::assertSame(['proposta_rejeitada'], $this->log->names());
        self::assertSame('gateway', $this->log->events[0]['context']['origem']);
    }

    public function testEngineValidationErrorReturns422WithLegacyFieldNames(): void
    {
        $engine = FakeEngine::client([FakeEngine::json(400, [
            'error' => 'requisicao invalida',
            'details' => [['field' => 'operation.currency', 'message' => 'moeda nao suportada (aceita: USD)']],
        ])]);
        $proposal = ProposalTranslatorTest::validProposal();
        $proposal['operacao']['moeda'] = 'BRL';

        $response = $this->post($engine, json_encode($proposal, JSON_THROW_ON_ERROR));

        self::assertSame(422, $response->getStatusCode());
        self::assertSame(
            ['erro' => 'proposta invalida', 'campos' => [['campo' => 'operacao.moeda', 'mensagem' => 'moeda nao suportada (aceita: USD)']]],
            self::body($response),
        );
        self::assertSame([], $this->proposals->saved);
        self::assertSame('motor', $this->log->events[0]['context']['origem']);
    }

    public function testRejectionLogCarriesNoPersonalData(): void
    {
        $engine = FakeEngine::client([FakeEngine::json(400, [
            'error' => 'requisicao invalida',
            'details' => [['field' => 'operation.currency', 'message' => 'moeda nao suportada (aceita: USD)']],
        ])]);

        $this->post($engine, self::validBody());

        $logged = json_encode($this->log->events, JSON_THROW_ON_ERROR);
        self::assertStringNotContainsString('John Doe', $logged, 'nome da contraparte e dado pessoal (LGPD)');
        self::assertStringNotContainsString('7500000', $logged, 'o log leva o motivo, nao os dados da proposta');
    }

    public function testMalformedJsonReturns400AndIsLogged(): void
    {
        $response = $this->post(FakeEngine::client([]), '{ nao e json');

        self::assertSame(400, $response->getStatusCode());
        self::assertSame(['proposta_rejeitada'], $this->log->names());
    }

    public function testEngineUnavailableReturns503(): void
    {
        $engine = FakeEngine::client([new ConnectException('connection refused', new Request('POST', 'api/v1/evaluate'))]);

        $response = $this->post($engine, self::validBody());

        self::assertSame(503, $response->getStatusCode());
        self::assertSame(['erro' => 'motor de conformidade indisponivel'], self::body($response));
        self::assertSame([], $this->proposals->saved);
        self::assertSame(['motor_indisponivel'], $this->log->names());
    }

    public function testEngineFailureReturns502WithoutLeakingDetails(): void
    {
        $engine = FakeEngine::client([FakeEngine::json(500, ['error' => 'falha ao gravar a avaliacao'])]);

        $response = $this->post($engine, self::validBody());

        self::assertSame(502, $response->getStatusCode());
        self::assertSame(['erro' => 'falha no motor de conformidade'], self::body($response));
        self::assertSame(['falha_no_motor'], $this->log->names());
    }

    private function post(EngineClient $engine, string $body): ResponseInterface
    {
        $request = (new ServerRequestFactory())
            ->createServerRequest('POST', '/propostas')
            ->withHeader('Content-Type', 'application/json')
            ->withBody((new StreamFactory())->createStream($body));

        return AppFactory::create($engine, $this->proposals, $this->log)->handle($request);
    }

    private static function validBody(): string
    {
        return json_encode(ProposalTranslatorTest::validProposal(), JSON_THROW_ON_ERROR);
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
