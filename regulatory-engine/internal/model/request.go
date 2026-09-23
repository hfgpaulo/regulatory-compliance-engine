package model

import "github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/money"

// EvaluationRequest é a entrada do endpoint POST /api/v1/evaluate: descreve
// um produto financeiro e a operação concreta que se pretende avaliar.
//
// As structs do domínio não conhecem Fiber nem MongoDB — carregam apenas as
// tags `json` (contrato da API) e `bson` (formato de persistência). Manter o
// domínio livre de dependências de transporte/banco facilita testes e evolução.
type EvaluationRequest struct {
	Product   Product   `json:"product"   bson:"product"`
	Operation Operation `json:"operation" bson:"operation"`
}

// Product descreve o produto financeiro que se pretende operar no Brasil.
type Product struct {
	Type           ProductType `json:"type"            bson:"type"`
	Origin         string      `json:"origin"          bson:"origin"`           // país de origem (ex.: "US")
	OriginCurrency string      `json:"origin_currency" bson:"origin_currency"` // moeda de origem (ex.: "USD")
}

// Operation descreve a operação financeira a ser avaliada.
//
// Amount usa decimal.Decimal (nunca float) para evitar erros de precisão em
// valores monetários — essencial num contexto de tributos e limites.
type Operation struct {
	Amount       money.Money     `json:"amount"       bson:"amount"`
	Currency     string          `json:"currency"     bson:"currency"`
	Method       OperationMethod `json:"method"       bson:"method"`
	Counterparty Counterparty    `json:"counterparty" bson:"counterparty"`
}

// Counterparty é a contraparte da operação.
type Counterparty struct {
	Name string `json:"name" bson:"name"`
	PEP  bool   `json:"pep"  bson:"pep"` // pessoa exposta politicamente
}
