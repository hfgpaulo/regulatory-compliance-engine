<?php

declare(strict_types=1);

use LegacyGateway\AppFactory;
use LegacyGateway\Database;
use LegacyGateway\EngineClient;
use LegacyGateway\JsonEventLog;
use LegacyGateway\PdoProposalRepository;

require __DIR__ . '/../vendor/autoload.php';

$engineUrl = getenv('ENGINE_URL') ?: 'http://localhost:3000';

AppFactory::create(
    EngineClient::fromBaseUrl($engineUrl),
    new PdoProposalRepository(Database::connectFromEnv(...)),
    new JsonEventLog(),
)->run();
