package httpapi

import (
	"github.com/gofiber/fiber/v3"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/engine"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/repository"
)

// Server agrupa as dependências dos handlers HTTP (o motor de regras e o
// repositório). Receber as dependências por injeção — em vez de variáveis
// globais — deixa os handlers testáveis e as dependências explícitas.
type Server struct {
	engine *engine.Engine
	repo   *repository.EvaluationRepository
}

// NewServer cria o servidor HTTP com suas dependências.
func NewServer(eng *engine.Engine, repo *repository.EvaluationRepository) *Server {
	return &Server{engine: eng, repo: repo}
}

// Register adiciona todas as rotas da API à instância do Fiber.
// Manter o registro em um único ponto facilita enxergar o contrato,
// versionado sob /api/v1.
func (s *Server) Register(app *fiber.App) {
	v1 := app.Group("/api/v1")

	v1.Get("/health", s.health)
	v1.Post("/evaluate", s.evaluate)
	v1.Get("/evaluations", s.listEvaluations)
	v1.Get("/evaluations/:id", s.getEvaluation)
}
