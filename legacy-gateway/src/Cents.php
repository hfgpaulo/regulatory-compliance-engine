<?php

declare(strict_types=1);

namespace LegacyGateway;

use InvalidArgumentException;

/**
 * Converte entre centavos inteiros (formato do legado) e decimal em string
 * com 2 casas (formato do motor). Nunca usa float: só aritmética inteira e de
 * string, para não perder precisão em valores monetários.
 */
final class Cents
{
    public static function toDecimal(int $cents): string
    {
        if ($cents < 0) {
            throw new InvalidArgumentException('valor em centavos nao pode ser negativo');
        }

        return sprintf('%d.%02d', intdiv($cents, 100), $cents % 100);
    }

    public static function fromDecimal(string $amount): int
    {
        if (preg_match('/^(\d+)\.(\d{2})$/', $amount, $parts) !== 1) {
            throw new InvalidArgumentException(sprintf('valor decimal invalido: "%s"', $amount));
        }

        return (int) $parts[1] * 100 + (int) $parts[2];
    }
}
