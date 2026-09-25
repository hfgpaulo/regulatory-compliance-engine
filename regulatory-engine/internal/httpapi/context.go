package httpapi

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
)

// storeTimeout é o prazo máximo de cada chamada ao store feita por um handler.
const storeTimeout = 5 * time.Second

// newRequestContext deriva o contexto das chamadas ao store a partir do
// contexto da requisição, com o prazo de storeTimeout. Derivar (em vez de
// partir de context.Background) propaga o que os middlewares anexarem
// (request-id, tracing) até o banco.
//
// Não cancela se o cliente desconectar: o fasthttp, base do Fiber, não
// sinaliza desconexão durante o handler. O prazo é o limite efetivo.
func newRequestContext(c fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Context(), storeTimeout)
}
