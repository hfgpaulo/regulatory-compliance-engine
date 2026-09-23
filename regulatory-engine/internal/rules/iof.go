package rules

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/engine"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/money"
)

// IOFRule avalia a incidência de IOF na entrada de recursos por transferência
// internacional (o caso central da tropicalização EUA -> BR). A regra recebe
// seus parâmetros na construção (injeção de dependência): não conhece arquivo
// de config, apenas aplica a lógica sobre os valores que recebeu.
type IOFRule struct {
	iofRate decimal.Decimal // alíquota de IOF (ex.: 0.0038)
	usdBRL  decimal.Decimal // câmbio USD->BRL usado na conversão
}

// NewIOFRule constrói a regra de IOF com a alíquota e o câmbio parametrizados.
func NewIOFRule(iofRate, usdBRL decimal.Decimal) *IOFRule {
	return &IOFRule{iofRate: iofRate, usdBRL: usdBRL}
}

// Code identifica a regra.
func (r *IOFRule) Code() string { return "iof.international_transfer" }

// Evaluate aplica a regra de IOF. Só incide sobre transferências internacionais;
// nos demais casos retorna nil (não se aplica).
func (r *IOFRule) Evaluate(req model.EvaluationRequest) (*engine.Outcome, error) {
	if req.Operation.Method != model.MethodInternationalTransfer {
		return nil, nil
	}

	// Assume-se origem em USD (caso de tropicalização). Converte o valor para
	// BRL e aplica a alíquota de IOF. Tudo em decimal, sem float.
	amountBRL := req.Operation.Amount.Mul(r.usdBRL)
	iof := money.New(amountBRL.Mul(r.iofRate).Round(2))

	percent := r.iofRate.Mul(decimal.NewFromInt(100))
	detail := fmt.Sprintf(
		"Entrada convertida de USD para BRL a %s; IOF de %s%% aplicavel (produto original nao previa tributacao).",
		r.usdBRL.StringFixed(2), percent.String(),
	)

	return &engine.Outcome{
		Result: model.Result{
			Domain:           model.DomainIOF,
			Status:           model.StatusAdaptationRequired,
			Detail:           detail,
			CalculatedAmount: &iof,
			Reference:        "rule:" + r.Code(),
		},
		Adaptation: "Incluir calculo e retencao de IOF no fluxo de entrada.",
	}, nil
}
