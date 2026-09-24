package httpapi

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
)

// evaluate recebe um produto/operação, roda o motor de regras, persiste a
// avaliação e devolve o recurso criado (id + data + request + report).
func (s *Server) evaluate(c fiber.Ctx) error {
	var req model.EvaluationRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "corpo da requisicao invalido",
		})
	}

	// Valida antes do motor: entrada incompleta não pode virar "conforme".
	var verr *model.ValidationError
	if errors.As(req.Validate(), &verr) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "requisicao invalida",
			"details": verr.Fields,
		})
	}

	report, err := s.engine.Evaluate(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "falha ao avaliar a operacao",
		})
	}

	evaluation := model.Evaluation{Request: req, Report: report}

	// Deriva do contexto da requisição para propagar o que os middlewares
	// anexarem (request-id, tracing). Não cancela se o cliente desconectar:
	// o fasthttp não sinaliza desconexão durante o handler.
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()
	if err := s.store.Save(ctx, &evaluation); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "falha ao gravar a avaliacao",
		})
	}

	// 201 Created: o corpo é o recurso recém-criado.
	return c.Status(fiber.StatusCreated).JSON(evaluation)
}
