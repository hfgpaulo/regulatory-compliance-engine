<?php

declare(strict_types=1);

namespace LegacyGateway\Tests;

use InvalidArgumentException;
use LegacyGateway\Cents;
use PHPUnit\Framework\Attributes\DataProvider;
use PHPUnit\Framework\TestCase;

final class CentsTest extends TestCase
{
    /**
     * @return iterable<string, array{int, string}>
     */
    public static function pairs(): iterable
    {
        yield 'zero' => [0, '0.00'];
        yield 'um centavo' => [1, '0.01'];
        yield 'dez centavos' => [10, '0.10'];
        yield 'um real' => [100, '1.00'];
        yield 'teto do COAF em reais' => [5000000, '50000.00'];
        yield 'C05 em dolares' => [7500000, '75000.00'];
        yield 'valor alto sem perda de precisao' => [123456789012, '1234567890.12'];
    }

    #[DataProvider('pairs')]
    public function testToDecimal(int $cents, string $decimal): void
    {
        self::assertSame($decimal, Cents::toDecimal($cents));
    }

    #[DataProvider('pairs')]
    public function testFromDecimal(int $cents, string $decimal): void
    {
        self::assertSame($cents, Cents::fromDecimal($decimal));
    }

    public function testToDecimalRejectsNegative(): void
    {
        $this->expectException(InvalidArgumentException::class);
        Cents::toDecimal(-1);
    }

    /**
     * @return iterable<string, array{string}>
     */
    public static function invalidDecimals(): iterable
    {
        yield 'sem casas' => ['1425'];
        yield 'uma casa' => ['1425.0'];
        yield 'tres casas' => ['1425.000'];
        yield 'virgula' => ['1425,00'];
        yield 'negativo' => ['-1.00'];
        yield 'vazio' => [''];
    }

    #[DataProvider('invalidDecimals')]
    public function testFromDecimalRejectsInvalid(string $decimal): void
    {
        $this->expectException(InvalidArgumentException::class);
        Cents::fromDecimal($decimal);
    }
}
