<?php

declare(strict_types=1);

namespace LegacyGateway\Tests;

use LegacyGateway\ProposalRecord;
use LegacyGateway\ProposalRepository;
use RuntimeException;

/**
 * Banco legado em memória. Com $failWith, simula o MySQL fora do ar.
 */
final class InMemoryProposalRepository implements ProposalRepository
{
    /** @var list<ProposalRecord> */
    public array $saved = [];

    public function __construct(private readonly ?string $failWith = null)
    {
    }

    public function save(ProposalRecord $record): void
    {
        if ($this->failWith !== null) {
            throw new RuntimeException($this->failWith);
        }
        $this->saved[] = $record;
    }
}
