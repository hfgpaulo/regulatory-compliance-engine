package httpapi

import (
	"github.com/gofiber/fiber/v3"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
)

// evaluate recebe um produto/operação e devolve o gap report produzido pelo
// motor de regras. A lógica de negócio mora nas regras (pacote rules) e no
// motor (pacote engine); o handler só cuida do HTTP: desserializa, chama o
// motor e serializa a resposta.
func (s *Server) evaluate(c fiber.Ctx) error {
	var req model.EvaluationRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "corpo da requisicao invalido",
		})
	}

	report, err := s.engine.Evaluate(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "falha ao avaliar a operacao",
		})
	}

	return c.JSON(report)
}
