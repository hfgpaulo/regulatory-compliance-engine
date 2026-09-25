package httpapi

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/engine"
)

func TestReady(t *testing.T) {
	cases := []struct {
		name       string
		db         Pinger
		wantStatus int
	}{
		{"banco acessivel", dbUp, fiber.StatusOK},
		{"banco indisponivel", PingerFunc(func(context.Context) error { return errors.New("sem conexao") }), fiber.StatusServiceUnavailable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			NewServer(engine.New(), newFakeStore(), tc.db).Register(app)

			resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/ready", nil))
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}

// TestHealth_DoesNotDependOnDatabase garante a separação liveness/readiness:
// com o banco fora, o processo continua "vivo" (não deve ser reiniciado).
func TestHealth_DoesNotDependOnDatabase(t *testing.T) {
	app := fiber.New()
	dbDown := PingerFunc(func(context.Context) error { return errors.New("sem conexao") })
	NewServer(engine.New(), newFakeStore(), dbDown).Register(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/health", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
