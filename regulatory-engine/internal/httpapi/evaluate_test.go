package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/engine"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
)

// fakeStore implementa EvaluationStore em memória: os handlers dependem da
// interface, então o teste roda sem MongoDB. Save simula o repositório real
// atribuindo um id quando ele vem vazio.
type fakeStore struct {
	saved   map[string]*model.Evaluation
	saveErr error
}

func newFakeStore() *fakeStore { return &fakeStore{saved: map[string]*model.Evaluation{}} }

func (f *fakeStore) Save(_ context.Context, eval *model.Evaluation) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	if eval.ID == "" {
		eval.ID = "generated-id"
	}
	f.saved[eval.ID] = eval
	return nil
}

func (f *fakeStore) FindByID(_ context.Context, id string) (*model.Evaluation, error) {
	return f.saved[id], nil // nil quando não existe: o handler traduz para 404
}

func (f *fakeStore) List(_ context.Context, _ int64) ([]model.Evaluation, error) {
	out := make([]model.Evaluation, 0, len(f.saved))
	for _, e := range f.saved {
		out = append(out, *e)
	}
	return out, nil
}

// newTestApp monta um app Fiber real com o store falso, para exercitar as
// rotas de ponta a ponta via app.Test (sem abrir porta de rede).
func newTestApp(store EvaluationStore) *fiber.App {
	eng := engine.New() // sem regras: operação sempre conforme, basta para o teste HTTP
	app := fiber.New()
	NewServer(eng, store).Register(app)
	return app
}

func TestEvaluate_Returns201AndPersists(t *testing.T) {
	store := newFakeStore()
	app := newTestApp(store)

	body := `{
		"product": {"type": "personal_loan", "origin": "US", "origin_currency": "USD"},
		"operation": {"amount": "1000.00", "currency": "USD", "method": "international_transfer",
			"counterparty": {"name": "John Doe", "pep": false}}
	}`
	req := httptest.NewRequest("POST", "/api/v1/evaluate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var created model.Evaluation
	raw, _ := io.ReadAll(resp.Body)
	require.NoError(t, json.Unmarshal(raw, &created))
	assert.NotEmpty(t, created.ID, "o recurso criado deve vir com id")
	assert.Len(t, store.saved, 1, "a avaliacao deve ter sido persistida")
}

func TestEvaluate_Returns400OnInvalidBody(t *testing.T) {
	app := newTestApp(newFakeStore())

	req := httptest.NewRequest("POST", "/api/v1/evaluate", strings.NewReader("{ not json"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetEvaluation_Returns404WhenMissing(t *testing.T) {
	app := newTestApp(newFakeStore())

	req := httptest.NewRequest("GET", "/api/v1/evaluations/does-not-exist", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}
