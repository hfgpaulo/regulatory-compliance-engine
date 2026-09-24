package model

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/money"
)

// validRequest devolve uma requisição válida; cada caso de teste altera um
// único campo para isolar a regra de validação exercitada.
func validRequest() EvaluationRequest {
	return EvaluationRequest{
		Product: Product{Type: ProductPersonalLoan, Origin: "US", OriginCurrency: "USD"},
		Operation: Operation{
			Amount:       money.New(decimal.RequireFromString("1000.00")),
			Currency:     "USD",
			Method:       MethodInternationalTransfer,
			Counterparty: Counterparty{Name: "John Doe"},
		},
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(r *EvaluationRequest)
		wantField string
	}{
		{"valida", func(r *EvaluationRequest) {}, ""},
		{"valor com zeros extras e aceito", func(r *EvaluationRequest) {
			r.Operation.Amount = money.New(decimal.RequireFromString("1000.000"))
		}, ""},
		{"tipo de produto desconhecido", func(r *EvaluationRequest) { r.Product.Type = "mortgage" }, "product.type"},
		{"origem vazia", func(r *EvaluationRequest) { r.Product.Origin = "" }, "product.origin"},
		{"origem fora do formato", func(r *EvaluationRequest) { r.Product.Origin = "usa" }, "product.origin"},
		{"moeda de origem fora do formato", func(r *EvaluationRequest) { r.Product.OriginCurrency = "us$" }, "product.origin_currency"},
		{"valor zero", func(r *EvaluationRequest) { r.Operation.Amount = money.New(decimal.Zero) }, "operation.amount"},
		{"valor negativo", func(r *EvaluationRequest) {
			r.Operation.Amount = money.New(decimal.RequireFromString("-10.00"))
		}, "operation.amount"},
		{"valor com mais de 2 casas", func(r *EvaluationRequest) {
			r.Operation.Amount = money.New(decimal.RequireFromString("1000.005"))
		}, "operation.amount"},
		{"moeda vazia", func(r *EvaluationRequest) { r.Operation.Currency = "" }, "operation.currency"},
		{"metodo desconhecido", func(r *EvaluationRequest) { r.Operation.Method = "pix" }, "operation.method"},
		{"contraparte em branco", func(r *EvaluationRequest) { r.Operation.Counterparty.Name = "   " }, "operation.counterparty.name"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validRequest()
			tc.mutate(&req)

			err := req.Validate()
			if tc.wantField == "" {
				assert.NoError(t, err)
				return
			}

			var verr *ValidationError
			require.ErrorAs(t, err, &verr)
			require.Len(t, verr.Fields, 1)
			assert.Equal(t, tc.wantField, verr.Fields[0].Field)
		})
	}
}

func TestValidate_EmptyRequestReportsAllFields(t *testing.T) {
	var verr *ValidationError
	require.ErrorAs(t, EvaluationRequest{}.Validate(), &verr)

	fields := make([]string, len(verr.Fields))
	for i, f := range verr.Fields {
		fields[i] = f.Field
	}
	assert.ElementsMatch(t, []string{
		"product.type", "product.origin", "product.origin_currency",
		"operation.amount", "operation.currency", "operation.method",
		"operation.counterparty.name",
	}, fields)
}
