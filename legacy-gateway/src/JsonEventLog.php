<?php

declare(strict_types=1);

namespace LegacyGateway;

/**
 * Escreve cada evento como uma linha JSON pura no stderr do processo, que no
 * container vai para o docker logs. Não usa error_log: sob o Apache, ele
 * prefixa a linha ("[data] [php:notice] [pid ...]") e escapa as aspas, e o
 * agregador deixaria de conseguir ler o JSON.
 */
final class JsonEventLog implements EventLog
{
    /** Falhas de integração saem como ERROR (alertáveis); rejeições são fluxo normal. */
    private const ERROR_EVENTS = ['motor_indisponivel', 'falha_no_motor', 'gravacao_legado_falhou'];

    public function record(string $event, array $context): void
    {
        $level = in_array($event, self::ERROR_EVENTS, true) ? 'ERROR' : 'INFO';

        $line = json_encode(
            ['time' => gmdate('Y-m-d\TH:i:s\Z'), 'level' => $level, 'evento' => $event, 'servico' => 'legacy-gateway'] + $context,
            JSON_THROW_ON_ERROR | JSON_UNESCAPED_UNICODE,
        );

        file_put_contents('php://stderr', $line . "\n");
    }
}
