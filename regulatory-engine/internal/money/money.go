// Package money encapsula a representação de valores monetários.
//
// Money embute um decimal.Decimal (precisão exata, sem float) e padroniza a
// serialização: no JSON, dinheiro sempre sai como string com 2 casas
// ("1425.00"). Isso evita perda de precisão no cliente e dá consistência à
// API — todo valor monetário tem o mesmo formato.
package money

import "github.com/shopspring/decimal"

// Money representa um valor monetário. Embute decimal.Decimal, então herda
// todas as operações aritméticas (Mul, Add, etc.).
type Money struct {
	decimal.Decimal
}

// New cria um Money a partir de um decimal.
func New(d decimal.Decimal) Money {
	return Money{Decimal: d}
}

// MarshalJSON serializa o valor como string com 2 casas decimais.
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(`"` + m.Decimal.StringFixed(2) + `"`), nil
}

// UnmarshalJSON aceita tanto número (75000.00) quanto string ("75000.00"),
// delegando ao parser do decimal.
func (m *Money) UnmarshalJSON(data []byte) error {
	return m.Decimal.UnmarshalJSON(data)
}
