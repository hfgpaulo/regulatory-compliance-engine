<?php

declare(strict_types=1);

namespace LegacyGateway\Tests;

use LegacyGateway\EventLog;

/**
 * Log de eventos em memória, para os testes conferirem o que foi registrado.
 */
final class InMemoryEventLog implements EventLog
{
    /** @var list<array{event: string, context: array<string, mixed>}> */
    public array $events = [];

    public function record(string $event, array $context): void
    {
        $this->events[] = ['event' => $event, 'context' => $context];
    }

    /**
     * @return list<string>
     */
    public function names(): array
    {
        return array_column($this->events, 'event');
    }
}
