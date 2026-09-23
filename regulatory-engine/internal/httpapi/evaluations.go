package httpapi

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
)

// listEvaluations devolve as avaliações mais recentes.
func (s *Server) listEvaluations(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	evaluations, err := s.repo.List(ctx, 50)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "falha ao listar avaliacoes",
		})
	}
	return c.JSON(evaluations)
}

// getEvaluation devolve uma avaliação pelo id, ou 404 quando não existe.
func (s *Server) getEvaluation(c fiber.Ctx) error {
	id := c.Params("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	evaluation, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "falha ao buscar a avaliacao",
		})
	}
	if evaluation == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "avaliacao nao encontrada",
		})
	}
	return c.JSON(evaluation)
}
