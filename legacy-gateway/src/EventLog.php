<?php

declare(strict_types=1);

namespace LegacyGateway;

/**
 * Registro de eventos do gateway (rejeições e falhas). Como interface, os
 * testes conferem o que foi registrado — em especial, que nenhum dado pessoal
 * da proposta vai para o log.
 */
interface EventLog
{
    /**
     * @param array<string, mixed> $context
     */
    public function record(string $event, array $context): void;
}
