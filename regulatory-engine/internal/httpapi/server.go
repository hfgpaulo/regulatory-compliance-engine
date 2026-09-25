package httpapi

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/engine"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
)

// EvaluationStore é o contrato de persistência de que os handlers precisam.
// Depender de uma interface (definida aqui, onde é consumida) — e não do
// repositório concreto — deixa os handlers testáveis com um "fake" em memória.
type EvaluationStore interface {
	Save(ctx context.Context, eval *model.Evaluation) error
	FindByID(ctx context.Context, id string) (*model.Evaluation, error)
	List(ctx context.Context, limit int64) ([]model.Evaluation, error)
}

// Pinger verifica se uma dependência externa (o banco) está acessível.
// É o que o readiness consulta; como interface, o teste usa um fake.
type Pinger interface {
	Ping(ctx context.Context) error
}

// PingerFunc adapta uma função comum a Pinger, no estilo de http.HandlerFunc.
type PingerFunc func(ctx context.Context) error

// Ping chama a própria função.
func (f PingerFunc) Ping(ctx context.Context) error { return f(ctx) }

// Server agrupa as dependências dos handlers HTTP (motor de regras, versão da
// parametrização que o alimenta, store e verificação do banco para o readiness).
type Server struct {
	engine       *engine.Engine
	rulesVersion string
	store        EvaluationStore
	db           Pinger
}

// NewServer cria o servidor HTTP com suas dependências.
func NewServer(eng *engine.Engine, rulesVersion string, store EvaluationStore, db Pinger) *Server {
	return &Server{engine: eng, rulesVersion: rulesVersion, store: store, db: db}
}

// Register aplica o middleware de log e registra as rotas da API.
func (s *Server) Register(app *fiber.App) {
	app.Use(requestLogger)

	v1 := app.Group("/api/v1")

	v1.Get("/health", s.health)
	v1.Get("/ready", s.ready)
	v1.Post("/evaluate", s.evaluate)
	v1.Get("/evaluations", s.listEvaluations)
	v1.Get("/evaluations/:id", s.getEvaluation)
}
