package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/config"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/database"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/engine"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/httpapi"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/repository"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/rules"
)

// main é o ponto de entrada da API. Sua única responsabilidade é "montar" a
// aplicação: configurar o log, carregar a configuração, conectar ao banco,
// montar o motor, criar o servidor e começar a ouvir. A lógica de negócio
// mora nos pacotes internos, não aqui.
func main() {
	// Log estruturado em JSON para toda a aplicação.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()

	// Conexão com o MongoDB (fail-fast: sem banco, o serviço não sobe).
	client, db, err := database.Connect(context.Background(), cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		slog.Error("nao foi possivel conectar ao banco", "err", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(ctx)
	}()
	slog.Info("conectado ao MongoDB", "db", db.Name())

	// Carrega a parametrização regulatória e monta o motor com suas regras.
	params, err := rules.Load(cfg.RulesPath)
	if err != nil {
		slog.Error("nao foi possivel carregar as regras", "err", err)
		os.Exit(1)
	}
	eng := engine.New(
		rules.NewIOFRule(params.IOF.InternationalTransfer.Rate, params.FX.USDBRL),
		rules.NewPLDThresholdRule(params.PLD.ReportingThreshold.Amount, params.FX.USDBRL),
		rules.NewPLDScreeningRule(params.PLD.SanctionedNames),
	)
	slog.Info("motor de regras carregado", "path", cfg.RulesPath)

	// Repositório de avaliações (persistência no MongoDB).
	evaluationRepo := repository.NewEvaluationRepository(db)

	app := fiber.New(fiber.Config{
		AppName: "regulatory-engine",
	})

	// Injeta o motor e o repositório no servidor e registra as rotas.
	server := httpapi.NewServer(eng, evaluationRepo)
	server.Register(app)

	address := ":" + cfg.Port
	slog.Info("regulatory-engine ouvindo", "address", address, "env", cfg.AppEnv)

	if err := app.Listen(address); err != nil {
		slog.Error("falha ao iniciar o servidor", "err", err)
		os.Exit(1)
	}
}
