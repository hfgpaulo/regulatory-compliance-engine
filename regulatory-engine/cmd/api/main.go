package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/config"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/database"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/httpapi"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/repository"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/rules"
)

// shutdownTimeout é o prazo para as requisições em andamento terminarem após
// o sinal de encerramento. Precisa ser menor que a janela do orquestrador
// antes do SIGKILL (stop_grace_period no compose).
const shutdownTimeout = 10 * time.Second

// main é o ponto de entrada da API. Sua única responsabilidade é "montar" a
// aplicação: configurar o log, carregar a configuração, conectar ao banco,
// montar o motor, criar o servidor, começar a ouvir e encerrar com graça. A
// lógica de negócio mora nos pacotes internos, não aqui.
func main() {
	// Subcomando usado pelo HEALTHCHECK do container (a imagem não tem curl).
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck("http://127.0.0.1:" + config.Load().Port + "/api/v1/ready"))
	}

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
	slog.Info("conectado ao MongoDB", "db", db.Name())

	// Carrega a parametrização regulatória e monta o motor com suas regras.
	params, err := rules.Load(cfg.RulesPath)
	if err != nil {
		slog.Error("nao foi possivel carregar as regras", "err", err)
		os.Exit(1)
	}
	eng := rules.NewEngine(params)
	slog.Info("motor de regras carregado", "path", cfg.RulesPath, "rules_version", params.Version)

	// Repositório de avaliações (persistência no MongoDB).
	evaluationRepo := repository.NewEvaluationRepository(db)

	// Índices no boot, também fail-fast: sem eles as consultas degradam em silêncio.
	indexCtx, cancelIndex := context.WithTimeout(context.Background(), 10*time.Second)
	err = evaluationRepo.EnsureIndexes(indexCtx)
	cancelIndex()
	if err != nil {
		slog.Error("nao foi possivel criar os indices", "err", err)
		os.Exit(1)
	}

	app := fiber.New(fiber.Config{
		AppName: "regulatory-engine",
	})

	// Injeta o motor, a versão das regras, o repositório e o ping do banco (readiness) no servidor.
	pingDB := httpapi.PingerFunc(func(ctx context.Context) error {
		return client.Ping(ctx, readpref.Primary())
	})
	server := httpapi.NewServer(eng, params.Version, evaluationRepo, pingDB)
	server.Register(app)

	// SIGTERM (docker stop, orquestradores) e SIGINT (Ctrl+C) disparam o
	// encerramento gracioso.
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	address := ":" + cfg.Port
	slog.Info("regulatory-engine ouvindo", "address", address, "env", cfg.AppEnv)

	// Listen roda em goroutine para o main poder esperar o sinal. Não se usa o
	// GracefulContext do Fiber porque nele o Listen retorna assim que o
	// listener fecha, antes de as requisições em andamento terminarem — e o
	// banco seria desconectado no meio delas.
	listenErr := make(chan error, 1)
	go func() { listenErr <- app.Listen(address) }()

	select {
	case err := <-listenErr:
		slog.Error("falha ao iniciar o servidor", "err", err)
		disconnect(client)
		os.Exit(1)
	case <-signalCtx.Done():
	}

	slog.Info("sinal de encerramento recebido; aguardando requisicoes em andamento", "timeout", shutdownTimeout.String())
	// Para de aceitar conexões e bloqueia até as requisições terminarem (ou o prazo vencer).
	if err := app.ShutdownWithTimeout(shutdownTimeout); err != nil {
		slog.Error("encerramento do servidor excedeu o prazo", "err", err)
	}
	// Só agora, sem requisições em andamento, o banco é desconectado.
	disconnect(client)
	slog.Info("regulatory-engine encerrado")
}

// disconnect fecha a conexão com o MongoDB com prazo próprio.
func disconnect(client *mongo.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Disconnect(ctx); err != nil {
		slog.Error("falha ao desconectar do MongoDB", "err", err)
	}
}
