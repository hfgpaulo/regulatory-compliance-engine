<?php

declare(strict_types=1);

namespace LegacyGateway;

use RuntimeException;

/**
 * Proposta inválida, com os campos no vocabulário do legado. Vem da validação
 * do próprio gateway ou de um 400 do motor com os nomes já traduzidos.
 */
final class InvalidProposalException extends RuntimeException
{
    /**
     * @param list<array{campo: string, mensagem: string}> $fields
     */
    public function __construct(public readonly array $fields)
    {
        parent::__construct('proposta invalida');
    }
}
