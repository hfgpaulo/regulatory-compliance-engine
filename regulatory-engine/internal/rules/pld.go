package rules

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/engine"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/money"
)

// Este arquivo concentra as regras do domínio PLD/COAF (Prevenção à Lavagem
// de Dinheiro). São duas regras independentes — limite de comunicação e
// screening — cada uma com responsabilidade única, plugadas no mesmo motor.

// -----------------------------------------------------------------------------
// Regra 1: limite de comunicação ao COAF
// -----------------------------------------------------------------------------

// PLDThresholdRule marca a operação como reportável ao COAF quando o valor,
// convertido para BRL, atinge o teto definido em norma.
type PLDThresholdRule struct {
	thresholdBRL decimal.Decimal
	usdBRL       decimal.Decimal
}

// NewPLDThresholdRule constrói a regra com o teto (em BRL) e o câmbio.
func NewPLDThresholdRule(thresholdBRL, usdBRL decimal.Decimal) *PLDThresholdRule {
	return &PLDThresholdRule{thresholdBRL: thresholdBRL, usdBRL: usdBRL}
}

// Code identifica a regra.
func (r *PLDThresholdRule) Code() string { return "pld.reporting_threshold" }

// Evaluate compara o valor (convertido para BRL) com o teto de comunicação.
func (r *PLDThresholdRule) Evaluate(req model.EvaluationRequest) (*engine.Outcome, error) {
	amountBRL := req.Operation.Amount.Mul(r.usdBRL)

	if amountBRL.LessThan(r.thresholdBRL) {
		return nil, nil // abaixo do teto: não há comunicação obrigatória
	}

	value := money.New(amountBRL.Round(2))
	detail := fmt.Sprintf(
		"Valor de %s BRL atinge o teto de comunicacao (%s BRL); operacao reportavel ao COAF.",
		value.StringFixed(2), r.thresholdBRL.StringFixed(2),
	)

	return &engine.Outcome{
		Result: model.Result{
			Domain:           model.DomainPLDCOAF,
			Status:           model.StatusReportable,
			Detail:           detail,
			CalculatedAmount: &value,
			Reference:        "rule:" + r.Code(),
		},
		Adaptation: "Gerar comunicacao ao COAF para a operacao.",
	}, nil
}

// -----------------------------------------------------------------------------
// Regra 2: screening de PEP / sancionados
// -----------------------------------------------------------------------------

// PLDScreeningRule sinaliza a operação quando a contraparte é PEP (pessoa
// exposta politicamente) ou consta em uma lista de sancionados.
//
// A lista chega por injeção (hoje, uma lista mock da parametrização). Trocar
// a fonte por uma lista oficial (ex.: OFAC, COAF) muda só quem monta a regra,
// não a regra em si.
type PLDScreeningRule struct {
	sanctioned map[string]bool
}

// NewPLDScreeningRule normaliza a lista de nomes para busca sem depender de
// caixa (maiúsculas/minúsculas) ou espaços nas pontas.
func NewPLDScreeningRule(sanctionedNames []string) *PLDScreeningRule {
	set := make(map[string]bool, len(sanctionedNames))
	for _, name := range sanctionedNames {
		set[normalizeName(name)] = true
	}
	return &PLDScreeningRule{sanctioned: set}
}

// Code identifica a regra.
func (r *PLDScreeningRule) Code() string { return "pld.pep_screening" }

// Evaluate verifica se a contraparte é PEP ou consta na lista de sancionados.
func (r *PLDScreeningRule) Evaluate(req model.EvaluationRequest) (*engine.Outcome, error) {
	cp := req.Operation.Counterparty
	sanctioned := r.sanctioned[normalizeName(cp.Name)]

	if !cp.PEP && !sanctioned {
		return nil, nil // contraparte sem sinal de risco
	}

	var reason string
	switch {
	case cp.PEP && sanctioned:
		reason = "contraparte e PEP e consta em lista de sancionados"
	case cp.PEP:
		reason = "contraparte e pessoa exposta politicamente (PEP)"
	default:
		reason = "contraparte consta em lista de sancionados"
	}

	return &engine.Outcome{
		Result: model.Result{
			Domain:    model.DomainPLDCOAF,
			Status:    model.StatusReportable,
			Detail:    "Screening PLD: " + reason + ".",
			Reference: "rule:" + r.Code(),
		},
		Adaptation: "Aplicar diligencia reforcada e avaliar comunicacao ao COAF.",
	}, nil
}

// normalizeName padroniza um nome para comparação (minúsculas, sem espaços nas pontas).
func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
