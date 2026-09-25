package rules

import "github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/engine"

// NewEngine monta o motor com todas as regras a partir da parametrização.
// É o único ponto de montagem: o main e os testes de aceitação usam a mesma
// função, então o teste valida exatamente o motor que roda em produção.
// A ordem das regras define a ordem dos resultados no relatório.
func NewEngine(p Parameters) *engine.Engine {
	return engine.New(
		NewIOFRule(p.IOF.InternationalTransfer.Rate, p.FX.USDBRL),
		NewPLDThresholdRule(p.PLD.ReportingThreshold.Amount, p.FX.USDBRL),
		NewPLDScreeningRule(p.PLD.SanctionedNames),
	)
}
