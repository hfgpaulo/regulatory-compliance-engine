package rules_test

import (
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/money"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/rules"
)

// Cenários de aceitação: o motor montado exatamente como em produção
// (rules.NewEngine) sobre o config/rules.json versionado. Se a parametrização
// mudar, estes testes dizem quais vereditos mudaram. Os IDs (C01...) são os
// mesmos de docs/cenarios_de_teste.md e docs/cenarios_de_teste.http.
//
// Parametrização assumida: IOF 0,38%, câmbio USD->BRL 5,00, teto de
// comunicação ao COAF R$ 50.000,00, sancionados "Ivan Petrov" e "Mara Costa".

type expectedResult struct {
	domain         model.Domain
	status         model.Status
	amount         string // vazio quando o resultado não calcula valor
	detailContains string
}

func request(amount, counterparty string, pep bool) model.EvaluationRequest {
	return model.EvaluationRequest{
		Product: model.Product{Type: model.ProductPersonalLoan, Origin: "US", OriginCurrency: "USD"},
		Operation: model.Operation{
			Amount:       money.New(decimal.RequireFromString(amount)),
			Currency:     model.CurrencyUSD,
			Method:       model.MethodInternationalTransfer,
			Counterparty: model.Counterparty{Name: counterparty, PEP: pep},
		},
	}
}

func iof(amount string) expectedResult {
	return expectedResult{domain: model.DomainIOF, status: model.StatusAdaptationRequired, amount: amount, detailContains: "IOF de 0.38%"}
}

func threshold(amountBRL string) expectedResult {
	return expectedResult{domain: model.DomainPLDCOAF, status: model.StatusReportable, amount: amountBRL, detailContains: "teto de comunicacao"}
}

func screening(reason string) expectedResult {
	return expectedResult{domain: model.DomainPLDCOAF, status: model.StatusReportable, detailContains: reason}
}

func TestAcceptanceScenarios(t *testing.T) {
	params, err := rules.Load(filepath.Join("..", "..", "config", "rules.json"))
	require.NoError(t, err)
	eng := rules.NewEngine(params)

	cases := []struct {
		id   string
		name string
		req  model.EvaluationRequest
		want []expectedResult
	}{
		{"C01", "transferencia simples, contraparte sem risco", request("1000.00", "John Doe", false),
			[]expectedResult{iof("19.00")}},
		{"C02", "arredondamento do IOF (23,45683 -> 23,46)", request("1234.57", "John Doe", false),
			[]expectedResult{iof("23.46")}},
		{"C03", "logo abaixo do teto do COAF (R$ 49.999,95)", request("9999.99", "John Doe", false),
			[]expectedResult{iof("190.00")}},
		{"C04", "exatamente no teto do COAF (R$ 50.000,00)", request("10000.00", "John Doe", false),
			[]expectedResult{iof("190.00"), threshold("50000.00")}},
		{"C05", "acima do teto do COAF (R$ 375.000,00)", request("75000.00", "John Doe", false),
			[]expectedResult{iof("1425.00"), threshold("375000.00")}},
		{"C06", "contraparte PEP", request("1000.00", "John Doe", true),
			[]expectedResult{iof("19.00"), screening("pessoa exposta politicamente")}},
		{"C07", "contraparte sancionada", request("1000.00", "Ivan Petrov", false),
			[]expectedResult{iof("19.00"), screening("consta em lista de sancionados")}},
		{"C08", "sancionada com caixa e espacos diferentes", request("1000.00", "  ivan PETROV ", false),
			[]expectedResult{iof("19.00"), screening("consta em lista de sancionados")}},
		{"C09", "contraparte PEP e sancionada", request("1000.00", "Mara Costa", true),
			[]expectedResult{iof("19.00"), screening("e PEP e consta em lista de sancionados")}},
		{"C10", "todos os dominios disparam juntos", request("75000.00", "Mara Costa", true),
			[]expectedResult{iof("1425.00"), threshold("375000.00"), screening("e PEP e consta em lista de sancionados")}},
	}

	for _, tc := range cases {
		t.Run(tc.id+" "+tc.name, func(t *testing.T) {
			require.NoError(t, tc.req.Validate(), "o cenario deve ser uma requisicao valida")

			report, err := eng.Evaluate(tc.req)
			require.NoError(t, err)

			// Toda transferência internacional exige adaptação de IOF, então
			// nenhum cenário válido hoje sai conforme.
			assert.False(t, report.Compliant)
			assert.Len(t, report.RequiredAdaptations, len(tc.want))
			require.Len(t, report.Results, len(tc.want))

			for i, want := range tc.want {
				got := report.Results[i]
				assert.Equal(t, want.domain, got.Domain, "resultado %d: dominio", i)
				assert.Equal(t, want.status, got.Status, "resultado %d: status", i)
				assert.Contains(t, got.Detail, want.detailContains, "resultado %d: detalhe", i)
				if want.amount == "" {
					assert.Nil(t, got.CalculatedAmount, "resultado %d: nao deveria calcular valor", i)
				} else {
					require.NotNil(t, got.CalculatedAmount, "resultado %d: deveria calcular valor", i)
					assert.Equal(t, want.amount, got.CalculatedAmount.StringFixed(2), "resultado %d: valor", i)
				}
			}
		})
	}
}
