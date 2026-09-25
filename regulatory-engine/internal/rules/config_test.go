package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

// rulesFields são os valores parametrizáveis que os testes variam.
type rulesFields struct {
	rate, usdBRL, thresholdAmount, thresholdCurrency string
}

func validFields() rulesFields {
	return rulesFields{rate: "0.0038", usdBRL: "5.00", thresholdAmount: "50000", thresholdCurrency: "BRL"}
}

func rulesJSON(f rulesFields) string {
	return fmt.Sprintf(`{
		"iof": {"international_transfer": {"rate": %q}},
		"fx": {"usd_brl": %q},
		"pld": {
			"reporting_threshold": {"amount": %q, "currency": %q},
			"sanctioned_names": ["Ivan Petrov"]
		}
	}`, f.rate, f.usdBRL, f.thresholdAmount, f.thresholdCurrency)
}

func TestLoad_ValidFile(t *testing.T) {
	params, err := Load(writeRules(t, rulesJSON(validFields())))
	require.NoError(t, err)
	assert.Equal(t, "50000", params.PLD.ReportingThreshold.Amount.String())
}

func TestLoad_RejectsInvalidParameters(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(f *rulesFields)
		wantField string
	}{
		{"teto em outra moeda", func(f *rulesFields) { f.thresholdCurrency = "USD" }, "pld.reporting_threshold.currency"},
		{"aliquota zero", func(f *rulesFields) { f.rate = "0" }, "iof.international_transfer.rate"},
		{"aliquota negativa", func(f *rulesFields) { f.rate = "-0.0038" }, "iof.international_transfer.rate"},
		{"aliquota de 100%", func(f *rulesFields) { f.rate = "1" }, "iof.international_transfer.rate"},
		{"cambio zero", func(f *rulesFields) { f.usdBRL = "0" }, "fx.usd_brl"},
		{"cambio negativo", func(f *rulesFields) { f.usdBRL = "-5.00" }, "fx.usd_brl"},
		{"teto zero", func(f *rulesFields) { f.thresholdAmount = "0" }, "pld.reporting_threshold.amount"},
		{"teto negativo", func(f *rulesFields) { f.thresholdAmount = "-1" }, "pld.reporting_threshold.amount"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := validFields()
			tc.mutate(&f)

			_, err := Load(writeRules(t, rulesJSON(f)))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantField)
		})
	}
}

// TestLoad_MissingKeysAreRejected cobre o caso que motiva o "maior que zero":
// chave ausente (ou com nome digitado errado) vira zero no decimal.
func TestLoad_MissingKeysAreRejected(t *testing.T) {
	_, err := Load(writeRules(t, `{"pld": {"reporting_threshold": {"currency": "BRL"}}}`))
	require.Error(t, err)
	for _, field := range []string{"iof.international_transfer.rate", "fx.usd_brl", "pld.reporting_threshold.amount"} {
		assert.Contains(t, err.Error(), field, "todos os problemas devem ser reportados de uma vez")
	}
}

// TestLoad_VersionIsSHA256OfFileBytes garante que a versão é o hash dos
// bytes do arquivo (conferível com sha256sum) e que muda quando o arquivo muda.
func TestLoad_VersionIsSHA256OfFileBytes(t *testing.T) {
	content := rulesJSON(validFields())
	params, err := Load(writeRules(t, content))
	require.NoError(t, err)

	sum := sha256.Sum256([]byte(content))
	assert.Equal(t, "sha256:"+hex.EncodeToString(sum[:]), params.Version)

	changed := validFields()
	changed.rate = "0.0050"
	other, err := Load(writeRules(t, rulesJSON(changed)))
	require.NoError(t, err)
	assert.NotEqual(t, params.Version, other.Version, "arquivo diferente deve gerar versao diferente")
}

// TestLoad_ShippedRulesFile garante que o rules.json versionado continua
// válido: uma parametrização quebrada falha no CI, não no boot em produção.
func TestLoad_ShippedRulesFile(t *testing.T) {
	_, err := Load(filepath.Join("..", "..", "config", "rules.json"))
	require.NoError(t, err)
}
