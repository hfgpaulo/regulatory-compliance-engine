<?php

declare(strict_types=1);

namespace LegacyGateway;

use Closure;
use PDO;

/**
 * Grava as propostas no MySQL legado com prepared statements.
 *
 * A conexão é aberta só na primeira gravação (preguiçosa): o /health e as
 * requisições rejeitadas não dependem do banco legado, e um MySQL fora do ar
 * não impede o fluxo principal (proposta -> motor), que é a fonte da verdade.
 */
final class PdoProposalRepository implements ProposalRepository
{
    private ?PDO $pdo = null;

    /**
     * @param Closure(): PDO $connect
     */
    public function __construct(private readonly Closure $connect)
    {
    }

    public function save(ProposalRecord $record): void
    {
        $this->pdo ??= ($this->connect)();

        $statement = $this->pdo->prepare(
            'INSERT INTO propostas
                (protocolo, tipo_produto, modalidade, valor_centavos, moeda, contraparte_nome,
                 contraparte_pep, situacao, iof_centavos, versao_regras)
             VALUES
                (:protocolo, :tipo_produto, :modalidade, :valor_centavos, :moeda, :contraparte_nome,
                 :contraparte_pep, :situacao, :iof_centavos, :versao_regras)',
        );

        $statement->execute([
            'protocolo' => $record->protocolo,
            'tipo_produto' => $record->tipoProduto,
            'modalidade' => $record->modalidade,
            'valor_centavos' => $record->valorCentavos,
            'moeda' => $record->moeda,
            'contraparte_nome' => $record->contraparteNome,
            'contraparte_pep' => $record->contrapartePep,
            'situacao' => $record->situacao,
            'iof_centavos' => $record->iofCentavos,
            'versao_regras' => $record->versaoRegras,
        ]);
    }
}
