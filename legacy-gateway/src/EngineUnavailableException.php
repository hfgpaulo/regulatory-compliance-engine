<?php

declare(strict_types=1);

namespace LegacyGateway;

use RuntimeException;

/**
 * O motor não pôde ser alcançado a tempo (fora do ar, conexão recusada ou
 * timeout). O gateway responde 503: o problema é transitório.
 */
final class EngineUnavailableException extends RuntimeException
{
}
