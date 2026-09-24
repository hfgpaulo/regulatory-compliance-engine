package httpapi

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
)

// requestLogger registra cada requisição em log estruturado (JSON), com
// método, rota, status e latência. Um agregador (Loki, ELK, CloudWatch) consegue filtrar por campo.
func requestLogger(c fiber.Ctx) error {
	start := time.Now()

	err := c.Next() // processa a requisição

	slog.Info("request",
		"method", c.Method(),
		"path", c.Path(),
		"status", c.Response().StatusCode(),
		"latency_ms", time.Since(start).Milliseconds(),
	)
	return err
}
