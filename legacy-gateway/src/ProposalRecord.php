<?php

declare(strict_types=1);

namespace LegacyGateway;

/**
 * Linha da tabela `propostas` do banco legado: o que o legado guarda de uma
 * proposta aceita pelo motor. A avaliação completa (auditoria) fica no motor;
 * o `protocolo` liga os dois bancos.
 */
final readonly class ProposalRecord
{
    public function __construct(
        public string $protocolo,
        public string $tipoProduto,
        public string $modalidade,
        public int $valorCentavos,
        public string $moeda,
        public string $contraparteNome,
        public string $contrapartePep,
        public string $situacao,
        public int $iofCentavos,
        public string $versaoRegras,
    ) {
    }
}
