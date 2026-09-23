package engine

import (
	"fmt"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
)

// Rule é a abstração que toda regra regulatória implementa. O motor não sabe
// quais regras existem — só conhece esta interface. Adicionar uma nova regra
// (PLD, LGPD, etc.) não exige mudar o motor.
type Rule interface {
	// Code identifica a regra (ex.: "iof.international_transfer").
	Code() string
	// Evaluate aplica a regra à operação. Retorna nil quando a regra não se
	// aplica àquele caso (ex.: a regra de IOF não vale para operações que não
	// sejam transferência internacional).
	Evaluate(req model.EvaluationRequest) (*Outcome, error)
}

// Outcome é o que uma regra produz quando se aplica: o resultado por domínio
// e, opcionalmente, o texto da adaptação exigida.
type Outcome struct {
	Result     model.Result
	Adaptation string // vazio quando a regra não exige nenhuma adaptação
}

// Engine roda um conjunto de regras e agrega o GapReport.
type Engine struct {
	rules []Rule
}

// New cria um motor com as regras informadas. A ordem é preservada na saída.
func New(rules ...Rule) *Engine {
	return &Engine{rules: rules}
}

// Evaluate roda todas as regras aplicáveis e monta o veredito consolidado.
//
// Regras que retornam nil (não se aplicam) são ignoradas. A operação é
// considerada conforme quando nenhuma regra aplicável exigiu ação.
func (e *Engine) Evaluate(req model.EvaluationRequest) (model.GapReport, error) {
	report := model.GapReport{
		Compliant:           true,
		Results:             []model.Result{},
		RequiredAdaptations: []string{},
	}

	for _, rule := range e.rules {
		outcome, err := rule.Evaluate(req)
		if err != nil {
			return model.GapReport{}, fmt.Errorf("regra %s falhou: %w", rule.Code(), err)
		}
		if outcome == nil {
			continue // regra não se aplica a esta operação
		}

		report.Results = append(report.Results, outcome.Result)

		// Qualquer resultado que não seja "compliant" torna a operação não-conforme.
		if outcome.Result.Status != model.StatusCompliant {
			report.Compliant = false
		}
		if outcome.Adaptation != "" {
			report.RequiredAdaptations = append(report.RequiredAdaptations, outcome.Adaptation)
		}
	}

	return report, nil
}
