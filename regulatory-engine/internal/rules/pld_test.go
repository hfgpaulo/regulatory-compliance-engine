package rules

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/money"
)

func TestPLDThresholdRule(t *testing.T) {
	// teto de 50.000 BRL; câmbio 5,00 (então 10.000 USD já atinge o teto)
	rule := NewPLDThresholdRule(decimal.RequireFromString("50000"), decimal.RequireFromString("5.00"))

	cases := []struct {
		name        string
		amount      string
		wantApplies bool
		wantBRL     string
	}{
		{"acima do teto", "75000", true, "375000.00"},
		{"exatamente no teto", "10000", true, "50000.00"},
		{"abaixo do teto", "5000", false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := model.EvaluationRequest{Operation: model.Operation{Amount: money.New(decimal.RequireFromString(tc.amount))}}
			out, err := rule.Evaluate(req)
			require.NoError(t, err)

			if !tc.wantApplies {
				assert.Nil(t, out)
				return
			}
			require.NotNil(t, out)
			assert.Equal(t, model.DomainPLDCOAF, out.Result.Domain)
			assert.Equal(t, model.StatusReportable, out.Result.Status)
			require.NotNil(t, out.Result.CalculatedAmount)
			assert.Equal(t, tc.wantBRL, out.Result.CalculatedAmount.StringFixed(2))
		})
	}
}

func TestPLDScreeningRule(t *testing.T) {
	rule := NewPLDScreeningRule([]string{"Ivan Petrov", "Mara Costa"})

	cases := []struct {
		name        string
		partyName   string
		pep         bool
		wantApplies bool
	}{
		{"sancionado", "Ivan Petrov", false, true},
		{"sancionado com caixa/espaco diferentes", "  ivan petrov ", false, true},
		{"pep", "Fulano de Tal", true, true},
		{"limpo", "John Doe", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := model.EvaluationRequest{Operation: model.Operation{
				Counterparty: model.Counterparty{Name: tc.partyName, PEP: tc.pep},
			}}
			out, err := rule.Evaluate(req)
			require.NoError(t, err)

			if !tc.wantApplies {
				assert.Nil(t, out)
				return
			}
			require.NotNil(t, out)
			assert.Equal(t, model.DomainPLDCOAF, out.Result.Domain)
			assert.Equal(t, model.StatusReportable, out.Result.Status)
			assert.NotEmpty(t, out.Adaptation)
		})
	}
}
