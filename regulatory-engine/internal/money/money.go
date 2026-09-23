// Package money encapsula a representação de valores monetários.
//
// Money embute um decimal.Decimal (precisão exata, sem float) e padroniza a
// serialização em dois formatos:
//   - JSON: string com 2 casas ("1425.00"), para não perder precisão no cliente;
//   - BSON (MongoDB): Decimal128, o decimal nativo do Mongo — preciso e
//     consultável (dá para filtrar/agregar por valor).
package money

import (
	"fmt"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

// MarshalBSONValue grava o valor no MongoDB como Decimal128.
func (m Money) MarshalBSONValue() (bsontype.Type, []byte, error) {
	d128, err := primitive.ParseDecimal128(m.Decimal.String())
	if err != nil {
		return 0, nil, fmt.Errorf("money: erro ao converter para Decimal128: %w", err)
	}
	return bson.MarshalValue(d128)
}

// UnmarshalBSONValue lê um Decimal128 do MongoDB de volta para Money.
func (m *Money) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	var d128 primitive.Decimal128
	if err := bson.UnmarshalValue(t, data, &d128); err != nil {
		return err
	}
	d, err := decimal.NewFromString(d128.String())
	if err != nil {
		return fmt.Errorf("money: erro ao ler Decimal128: %w", err)
	}
	m.Decimal = d
	return nil
}
