<?php

declare(strict_types=1);

namespace LegacyGateway;

use RuntimeException;

/**
 * O motor rejeitou a requisição (400) apontando campos inválidos, ainda com
 * os nomes do contrato do motor (ex.: "operation.amount").
 */
final class EngineValidationException extends RuntimeException
{
    /**
     * @param list<array{field: string, message: string}> $details
     */
    public function __construct(public readonly array $details)
    {
        parent::__construct('motor rejeitou a requisicao');
    }
}
