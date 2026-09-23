package httpapi

import (
	"github.com/gofiber/fiber/v3"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/engine"
)

// Server agrupa as dependências dos handlers HTTP (por enquanto, o motor de
// regras). Receber as dependências por injeção — em vez de variáveis globais —
// deixa os handlers testáveis e as dependências explícitas.
type Server struct {
	engine *engine.Engine
}

// NewServer cria o servidor HTTP com suas dependências.
func NewServer(eng *engine.Engine) *Server {
	return &Server{engine: eng}
}

// Register adiciona todas as rotas da API à instância do Fiber.
// Manter o registro em um único ponto facilita enxergar o contrato,
// versionado sob /api/v1.
func (s *Server) Register(app *fiber.App) {
	v1 := app.Group("/api/v1")

	v1.Get("/health", s.health)
	v1.Post("/evaluate", s.evaluate)
}
