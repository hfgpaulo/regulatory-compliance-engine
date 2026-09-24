package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeRules grava um rules.json temporário e devolve o caminho.
func writeRules(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rules.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func rulesJSON(thresholdCurrency string) string {
	return `{
		"iof": {"international_transfer": {"rate": "0.0038"}},
		"fx": {"usd_brl": "5.00"},
		"pld": {
			"reporting_threshold": {"amount": "50000", "currency": "` + thresholdCurrency + `"},
			"sanctioned_names": ["Ivan Petrov"]
		}
	}`
}

func TestLoad_ValidFile(t *testing.T) {
	params, err := Load(writeRules(t, rulesJSON("BRL")))
	require.NoError(t, err)
	assert.Equal(t, "50000", params.PLD.ReportingThreshold.Amount.String())
}

func TestLoad_RejectsThresholdInOtherCurrency(t *testing.T) {
	_, err := Load(writeRules(t, rulesJSON("USD")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pld.reporting_threshold.currency")
}

// TestLoad_ShippedRulesFile garante que o rules.json versionado continua
// válido: uma parametrização quebrada falha no CI, não no boot em produção.
func TestLoad_ShippedRulesFile(t *testing.T) {
	_, err := Load(filepath.Join("..", "..", "config", "rules.json"))
	require.NoError(t, err)
}
