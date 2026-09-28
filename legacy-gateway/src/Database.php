<?php

declare(strict_types=1);

namespace LegacyGateway;

use PDO;
use RuntimeException;

/**
 * Conexão com o MySQL legado e o schema que o gateway precisa. O código é
 * dono do schema (como o EnsureIndexes do motor): a migração é idempotente e
 * roda a cada subida do container, inclusive sobre um volume já existente.
 */
final class Database
{
    public static function connectFromEnv(): PDO
    {
        $dsn = getenv('DB_DSN');
        if ($dsn === false || $dsn === '') {
            throw new RuntimeException('DB_DSN nao definida');
        }

        return new PDO($dsn, getenv('DB_USER') ?: null, getenv('DB_PASSWORD') ?: null, [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_EMULATE_PREPARES => false,
            PDO::ATTR_TIMEOUT => 2,
        ]);
    }

    public static function migrate(PDO $pdo): void
    {
        // Estilo legado: nomes em português e dinheiro em centavos inteiros (sem float).
        $pdo->exec(
            'CREATE TABLE IF NOT EXISTS propostas (
                id               BIGINT AUTO_INCREMENT PRIMARY KEY,
                protocolo        CHAR(24)     NOT NULL,
                tipo_produto     VARCHAR(40)  NOT NULL,
                modalidade       VARCHAR(40)  NOT NULL,
                valor_centavos   BIGINT       NOT NULL,
                moeda            CHAR(3)      NOT NULL,
                contraparte_nome VARCHAR(200) NOT NULL,
                contraparte_pep  CHAR(1)      NOT NULL,
                situacao         VARCHAR(30)  NOT NULL,
                iof_centavos     BIGINT       NOT NULL,
                versao_regras    VARCHAR(80)  NOT NULL,
                criado_em        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
                UNIQUE KEY uk_propostas_protocolo (protocolo)
            ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4',
        );
    }
}
