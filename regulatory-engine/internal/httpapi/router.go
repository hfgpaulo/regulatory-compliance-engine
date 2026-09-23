package httpapi

import "github.com/gofiber/fiber/v3"

// RegisterRoutes adiciona todas as rotas da API à instância do Fiber.
// Manter o registro de rotas em um único ponto facilita enxergar o
// contrato da API e versioná-lo (tudo abaixo de /api/v1).
func RegisterRoutes(app *fiber.App) {
	v1 := app.Group("/api/v1")

	v1.Get("/health", health)
	v1.Post("/evaluate", evaluate)
}
