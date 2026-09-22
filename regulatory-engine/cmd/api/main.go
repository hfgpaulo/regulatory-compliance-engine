package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/config"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/database"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/httpapi"
)

// main é o ponto de entrada da API. Sua única responsabilidade é "montar" a
// aplicação: carregar a configuração, conectar ao banco, criar o servidor,
// registrar as rotas e começar a ouvir. A lógica de negócio mora nos pacotes
// internos, não aqui.
func main() {
	cfg := config.Load()

	// Conexão com o MongoDB (fail-fast: sem banco, o serviço não sobe).
	client, db, err := database.Connect(context.Background(), cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("nao foi possivel conectar ao banco: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(ctx)
	}()
	log.Printf("conectado ao MongoDB (db=%s)", db.Name())

	app := fiber.New(fiber.Config{
		AppName: "regulatory-engine",
	})

	// Log de requisições. Por enquanto no formato padrão do Fiber;
	// os logs estruturados em JSON entram no bloco de observabilidade.
	app.Use(logger.New())

	// Registro de todas as rotas da API.
	httpapi.RegisterRoutes(app)

	address := ":" + cfg.Port
	log.Printf("regulatory-engine ouvindo em %s (env=%s)", address, cfg.AppEnv)

	if err := app.Listen(address); err != nil {
		log.Fatalf("falha ao iniciar o servidor: %v", err)
	}
}
