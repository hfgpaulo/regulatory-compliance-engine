package rules

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/money"
)

func opRequest(method model.OperationMethod, amount string) model.EvaluationRequest {
	return model.EvaluationRequest{
		Operation: model.Operation{
			Amount: money.New(decimal.RequireFromString(amount)),
			Method: method,
		},
	}
}

func TestIOFRule(t *testing.T) {
	rule := NewIOFRule(decimal.RequireFromString("0.0038"), decimal.RequireFromString("5.00"))

	cases := []struct {
		name        string
		method      model.OperationMethod
		amount      string
		wantApplies bool
		wantIOF     string // 75000 USD * 5,00 * 0,0038 = 1425,00 BRL
	}{
		{"transferencia internacional aplica", model.MethodInternationalTransfer, "75000", true, "1425.00"},
		{"outro metodo nao aplica", model.OperationMethod("pix"), "75000", false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := rule.Evaluate(opRequest(tc.method, tc.amount))
			require.NoError(t, err)

			if !tc.wantApplies {
				assert.Nil(t, out)
				return
			}

			require.NotNil(t, out)
			assert.Equal(t, model.DomainIOF, out.Result.Domain)
			assert.Equal(t, model.StatusAdaptationRequired, out.Result.Status)
			require.NotNil(t, out.Result.CalculatedAmount)
			assert.Equal(t, tc.wantIOF, out.Result.CalculatedAmount.StringFixed(2))
			assert.NotEmpty(t, out.Adaptation)
		})
	}
}
