package engine

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
)

// fakeRule é uma regra controlável: o teste decide o que ela devolve, sem
// depender das regras reais (IOF/PLD). Assim testamos apenas a agregação.
type fakeRule struct {
	code    string
	outcome *Outcome
	err     error
}

func (f fakeRule) Code() string { return f.code }
func (f fakeRule) Evaluate(model.EvaluationRequest) (*Outcome, error) {
	return f.outcome, f.err
}

func adaptation(domain model.Domain) *Outcome {
	return &Outcome{
		Result:     model.Result{Domain: domain, Status: model.StatusAdaptationRequired},
		Adaptation: "adaptar " + string(domain),
	}
}

func TestEngine_AggregatesApplicableRules(t *testing.T) {
	eng := New(
		fakeRule{code: "a", outcome: adaptation(model.DomainIOF)},
		fakeRule{code: "b", outcome: nil}, // não se aplica: deve ser ignorada
		fakeRule{code: "c", outcome: adaptation(model.DomainPLDCOAF)},
	)

	report, err := eng.Evaluate(model.EvaluationRequest{})
	require.NoError(t, err)

	assert.False(t, report.Compliant, "qualquer status != compliant torna a operacao nao-conforme")
	assert.Len(t, report.Results, 2, "regra que retorna nil nao entra no relatorio")
	assert.Len(t, report.RequiredAdaptations, 2)
}

func TestEngine_CompliantWhenNoRuleApplies(t *testing.T) {
	eng := New(
		fakeRule{code: "a", outcome: nil},
		fakeRule{code: "b", outcome: nil},
	)

	report, err := eng.Evaluate(model.EvaluationRequest{})
	require.NoError(t, err)

	assert.True(t, report.Compliant)
	assert.Empty(t, report.Results)
	assert.Empty(t, report.RequiredAdaptations)
}

func TestEngine_CompliantResultDoesNotFlagNonCompliant(t *testing.T) {
	eng := New(fakeRule{code: "a", outcome: &Outcome{
		Result: model.Result{Domain: model.DomainIOF, Status: model.StatusCompliant},
	}})

	report, err := eng.Evaluate(model.EvaluationRequest{})
	require.NoError(t, err)

	assert.True(t, report.Compliant)
	assert.Len(t, report.Results, 1)
	assert.Empty(t, report.RequiredAdaptations, "resultado sem adaptacao nao adiciona texto")
}

func TestEngine_PropagatesRuleError(t *testing.T) {
	boom := errors.New("boom")
	eng := New(fakeRule{code: "a", err: boom})

	_, err := eng.Evaluate(model.EvaluationRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, boom, "erro da regra deve ser envelopado com %%w")
}
