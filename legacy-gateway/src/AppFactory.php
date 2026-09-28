<?php

declare(strict_types=1);

namespace LegacyGateway;

use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Slim\App;
use Slim\Factory\AppFactory as SlimAppFactory;

/**
 * Monta a aplicação Slim. É o único ponto de montagem: o public/index.php e
 * os testes usam a mesma função, então o teste exercita o app de produção.
 */
final class AppFactory
{
    /**
     * @return App<ContainerInterface|null>
     */
    public static function create(): App
    {
        $app = SlimAppFactory::create();
        $app->addRoutingMiddleware();
        // Não expõe detalhes de erro ao cliente; registra no log do servidor.
        $app->addErrorMiddleware(false, true, true);

        $app->get('/health', static function (ServerRequestInterface $request, ResponseInterface $response): ResponseInterface {
            $response->getBody()->write(json_encode(
                ['status' => 'ok', 'service' => 'legacy-gateway'],
                JSON_THROW_ON_ERROR,
            ));

            return $response->withHeader('Content-Type', 'application/json');
        });

        return $app;
    }
}
