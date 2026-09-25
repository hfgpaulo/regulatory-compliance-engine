package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/shopspring/decimal"

	"github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/model"
)

// Parameters é a parametrização regulatória carregada do rules.json.
// Concentra os valores que a norma define (alíquotas, limites, câmbio) fora
// do código — assim, quando a norma muda, ajusta-se o arquivo, não o binário.
type Parameters struct {
	IOF IOFParams `json:"iof"`
	FX  FXParams  `json:"fx"`
	PLD PLDParams `json:"pld"`

	// Version identifica o arquivo carregado: "sha256:" + hash dos bytes.
	// Não vem do JSON; é calculado no Load e conferível com sha256sum.
	Version string `json:"-"`
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
	if err := params.Validate(); err != nil {
		return Parameters{}, fmt.Errorf("parametrizacao invalida em %s: %w", path, err)
	}

	sum := sha256.Sum256(data)
	params.Version = "sha256:" + hex.EncodeToString(sum[:])
	return params, nil
}

// Validate confere se a parametrização é coerente com o que as regras
// assumem. Acumula todos os problemas para o boot falhar mostrando de uma
// vez tudo o que precisa ser corrigido.
func (p Parameters) Validate() error {
	var errs []error

	// Campo ausente no JSON vira zero, então "maior que zero" também pega
	// chave esquecida ou digitada errado. Alíquota zero real se representa
	// desligando a regra, não com rate 0.
	if rate := p.IOF.InternationalTransfer.Rate; !rate.IsPositive() || rate.GreaterThanOrEqual(decimal.NewFromInt(1)) {
		errs = append(errs, fmt.Errorf("iof.international_transfer.rate: deve ser maior que 0 e menor que 1, recebido %s", rate))
	}
	if fx := p.FX.USDBRL; !fx.IsPositive() {
		errs = append(errs, fmt.Errorf("fx.usd_brl: deve ser maior que 0, recebido %s", fx))
	}
	if amount := p.PLD.ReportingThreshold.Amount; !amount.IsPositive() {
		errs = append(errs, fmt.Errorf("pld.reporting_threshold.amount: deve ser maior que 0, recebido %s", amount))
	}

	// O teto é comparado com o valor já convertido para BRL; outra moeda
	// aqui seria tratada como BRL sem aviso.
	if c := p.PLD.ReportingThreshold.Currency; c != model.CurrencyBRL {
		errs = append(errs, fmt.Errorf("pld.reporting_threshold.currency: esperado %s, recebido %q", model.CurrencyBRL, c))
	}

	return errors.Join(errs...)
}
