package httpapi

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
)

// readyTimeout limita o ping do readiness: um banco lento já conta como
// indisponível para receber tráfego.
const readyTimeout = 2 * time.Second

// health é o liveness: responde se o processo está no ar, sem tocar em
// dependências. Um orquestrador reinicia o processo quando ele falha, então
// não pode depender do banco (reiniciar não conserta o banco).
func (s *Server) health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "regulatory-engine",
	})
}

// ready é o readiness: responde se o serviço consegue atender agora, o que
// exige o banco acessível. Falhar aqui tira o serviço do tráfego sem reiniciá-lo.
func (s *Server) ready(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), readyTimeout)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		slog.Warn("readiness: banco indisponivel", "err", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":  "unavailable",
			"service": "regulatory-engine",
		})
	}
	return c.JSON(fiber.Map{
		"status":  "ready",
		"service": "regulatory-engine",
	})
}
