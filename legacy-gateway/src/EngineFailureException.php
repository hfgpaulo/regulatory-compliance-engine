<?php

declare(strict_types=1);

namespace LegacyGateway;

use RuntimeException;

/**
 * O motor respondeu, mas com erro (5xx) ou algo fora do contrato. O gateway
 * responde 502: a falha está no serviço de trás, não na proposta.
 */
final class EngineFailureException extends RuntimeException
{
}
