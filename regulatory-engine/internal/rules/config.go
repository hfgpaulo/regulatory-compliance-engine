package rules

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/shopspring/decimal"
)

// Parameters é a parametrização regulatória carregada do rules.json.
// Concentra os valores que a norma define (alíquotas, limites, câmbio) fora
// do código — assim, quando a norma muda, ajusta-se o arquivo, não o binário.
type Parameters struct {
	IOF IOFParams `json:"iof"`
	FX  FXParams  `json:"fx"`
	PLD PLDParams `json:"pld"`
}

// PLDParams agrupa a parametrização de Prevenção à Lavagem de Dinheiro.
type PLDParams struct {
	ReportingThreshold Threshold `json:"reporting_threshold"` // teto de comunicação ao COAF
	SanctionedNames    []string  `json:"sanctioned_names"`    // lista mock de sancionados
}

// Threshold é um limite monetário com a moeda em que foi definido.
type Threshold struct {
	Amount   decimal.Decimal `json:"amount"`
	Currency string          `json:"currency"`
}

// IOFParams agrupa a parametrização do IOF por tipo de operação.
type IOFParams struct {
	InternationalTransfer IOFRate `json:"international_transfer"`
}

// IOFRate é a alíquota de IOF (ex.: "0.0038" = 0,38%).
type IOFRate struct {
	Rate decimal.Decimal `json:"rate"`
}

// FXParams agrupa as taxas de câmbio usadas na conversão.
type FXParams struct {
	USDBRL decimal.Decimal `json:"usd_brl"` // quantos reais vale 1 dólar
}

// Load lê e interpreta o rules.json no caminho informado.
func Load(path string) (Parameters, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Parameters{}, fmt.Errorf("erro ao ler %s: %w", path, err)
	}

	var params Parameters
	if err := json.Unmarshal(data, &params); err != nil {
		return Parameters{}, fmt.Errorf("erro ao interpretar %s: %w", path, err)
	}
	return params, nil
}
