package httpapi

import "github.com/gofiber/fiber/v3"

// health responde o healthcheck da aplicação.
// É consumido por orquestradores (Docker/Kubernetes) e por monitoramento
// para saber, de forma barata, se o serviço está no ar e respondendo.
func (s *Server) health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "regulatory-engine",
	})
}
