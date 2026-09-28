<?php

declare(strict_types=1);

use LegacyGateway\Database;

require __DIR__ . '/../vendor/autoload.php';

// Roda antes do Apache subir (ver Dockerfile). Falhar aqui impede o container
// de subir: sem a tabela, o gateway não teria onde registrar as propostas.
Database::migrate(Database::connectFromEnv());

fwrite(STDOUT, "legacy-gateway: schema do banco legado verificado\n");
