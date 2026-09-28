package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/database"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/money"
)

// Testes de integração contra um MongoDB real. Rodam só quando MONGO_TEST_URI
// está definida (no CI, por um service container; localmente, com o Mongo do
// compose) e se pulam caso contrário, para o go test comum seguir sem Docker.
// Cada teste usa um banco com nome único, apagado ao final.

func newTestRepository(t *testing.T) *EvaluationRepository {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI nao definida: teste de integracao pulado")
	}

	ctx := context.Background()
	dbName := fmt.Sprintf("regulatory_test_%d", time.Now().UnixNano())
	client, db, err := database.Connect(ctx, uri, dbName)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})
	return NewEvaluationRepository(db)
}

func amount(s string) *money.Money {
	m := money.New(decimal.RequireFromString(s))
	return &m
}

func sampleEvaluation() model.Evaluation {
	return model.Evaluation{
		RulesVersion: "sha256:abc123",
		Request: model.EvaluationRequest{
			Product: model.Product{Type: model.ProductPersonalLoan, Origin: "US", OriginCurrency: "USD"},
			Operation: model.Operation{
				Amount:       *amount("75000.00"),
				Currency:     model.CurrencyUSD,
				Method:       model.MethodInternationalTransfer,
				Counterparty: model.Counterparty{Name: "Mara Costa", PEP: true},
			},
		},
		Report: model.GapReport{
			Compliant: false,
			Results: []model.Result{
				{Domain: model.DomainIOF, Status: model.StatusAdaptationRequired, Detail: "IOF", CalculatedAmount: amount("1425.00"), Reference: "rule:iof"},
				{Domain: model.DomainPLDCOAF, Status: model.StatusReportable, Detail: "Screening", Reference: "rule:pld"},
			},
			RequiredAdaptations: []string{"reter IOF", "comunicar COAF"},
		},
	}
}

// TestSaveAndFindByID_RoundTrip garante que o que o POST devolve é idêntico ao
// que um GET posterior lê do banco — inclusive created_at (precisão de ms),
// valores Decimal128, resultado sem valor calculado e rules_version.
func TestSaveAndFindByID_RoundTrip(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	saved := sampleEvaluation()
	require.NoError(t, repo.Save(ctx, &saved))
	assert.NotEmpty(t, saved.ID, "Save deve gerar o id")
	assert.False(t, saved.CreatedAt.IsZero(), "Save deve preencher created_at")

	found, err := repo.FindByID(ctx, saved.ID)
	require.NoError(t, err)
	require.NotNil(t, found)

	want, err := json.Marshal(saved)
	require.NoError(t, err)
	got, err := json.Marshal(found)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
}

func TestFindByID_ReturnsNilWhenMissing(t *testing.T) {
	repo := newTestRepository(t)

	found, err := repo.FindByID(context.Background(), "nao-existe")
	require.NoError(t, err)
	assert.Nil(t, found, "id inexistente deve voltar nil, que o handler traduz para 404")
}

func TestList_NewestFirstAndLimited(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	for i, id := range []string{"oldest", "middle", "newest"} {
		eval := sampleEvaluation()
		eval.ID = id
		eval.CreatedAt = base.Add(time.Duration(i) * time.Hour)
		require.NoError(t, repo.Save(ctx, &eval))
	}

	list, err := repo.List(ctx, 2)
	require.NoError(t, err)
	require.Len(t, list, 2, "o limite deve ser respeitado")
	assert.Equal(t, "newest", list[0].ID)
	assert.Equal(t, "middle", list[1].ID)
}

func TestEnsureIndexes_IsIdempotent(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	require.NoError(t, repo.EnsureIndexes(ctx))
	require.NoError(t, repo.EnsureIndexes(ctx), "rodar de novo (como a cada boot) nao pode falhar")

	cursor, err := repo.col.Indexes().List(ctx)
	require.NoError(t, err)
	var indexes []bson.M
	require.NoError(t, cursor.All(ctx, &indexes))

	names := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		names = append(names, idx["name"].(string))
	}
	assert.Contains(t, names, "created_at_desc")
}
