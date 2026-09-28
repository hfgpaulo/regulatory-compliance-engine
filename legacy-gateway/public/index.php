<?php

declare(strict_types=1);

use LegacyGateway\AppFactory;
use LegacyGateway\EngineClient;

require __DIR__ . '/../vendor/autoload.php';

$engineUrl = getenv('ENGINE_URL') ?: 'http://localhost:3000';

AppFactory::create(EngineClient::fromBaseUrl($engineUrl))->run();
