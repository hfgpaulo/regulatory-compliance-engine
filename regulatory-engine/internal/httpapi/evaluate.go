package httpapi

import (
	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
)

// evaluate recebe um produto/operação e devolve o gap report.
//
// STUB: por enquanto o veredito é fixo, apenas para materializar o contrato
// da API. A lógica real (motor de regras + domínios PLD/IOF) entra nos
// próximos blocos, sem alterar este contrato de entrada e saída.
func evaluate(c fiber.Ctx) error {
	var req model.EvaluationRequest

	// Bind().Body desserializa o corpo JSON na struct de entrada.
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "corpo da requisicao invalido",
		})
	}

	report := exemploGapReport()
	return c.JSON(report)
}

// exemploGapReport devolve um veredito de exemplo, fixo, usado enquanto o
// motor de regras não existe. Serve para o contrato ficar testável desde já.
func exemploGapReport() model.GapReport {
	iof := decimal.NewFromFloat(288.75)

	return model.GapReport{
		Compliant: false,
		Results: []model.Result{
			{
				Domain:           model.DomainIOF,
				Status:           model.StatusAdaptationRequired,
				Detail:           "Operação sujeita a IOF na entrada; produto original não previa tributação.",
				CalculatedAmount: &iof,
				Reference:        "rule:iof.international_transfer",
			},
			{
				Domain:    model.DomainPLDCOAF,
				Status:    model.StatusReportable,
				Detail:    "Valor acima do teto de comunicação automática ao COAF.",
				Reference: "rule:pld.reporting_threshold",
			},
		},
		RequiredAdaptations: []string{
			"Incluir cálculo e retenção de IOF no fluxo de entrada.",
			"Gerar comunicação ao COAF para a operação.",
		},
	}
}
