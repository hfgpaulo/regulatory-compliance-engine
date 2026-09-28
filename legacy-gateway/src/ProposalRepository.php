<?php

declare(strict_types=1);

namespace LegacyGateway;

/**
 * Persistência das propostas no banco legado. O handler depende desta
 * interface, não do PDO: os testes usam uma implementação em memória.
 */
interface ProposalRepository
{
    public function save(ProposalRecord $record): void;
}
