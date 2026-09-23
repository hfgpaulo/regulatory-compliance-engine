package model

import "github.com/hfgpaulo/regulatory-compliance-engine/regulatory-engine/internal/money"

// GapReport é a saída do endpoint /evaluate: o veredito de conformidade da
// operação mais a lista de adaptações necessárias para operar no Brasil.
type GapReport struct {
	Compliant           bool     `json:"compliant"            bson:"compliant"`
	Results             []Result `json:"results"              bson:"results"`
	RequiredAdaptations []string `json:"required_adaptations" bson:"required_adaptations"`
}

// Result é o resultado da avaliação em um único domínio regulatório.
//
// CalculatedAmount é opcional (ponteiro + omitempty): só faz sentido quando
// o domínio calcula um valor — por exemplo, o montante de IOF. Domínios que
// não calculam nada (ex.: uma marcação de PLD) omitem o campo.
type Result struct {
	Domain           Domain           `json:"domain"                      bson:"domain"`
	Status           Status           `json:"status"                      bson:"status"`
	Detail           string           `json:"detail"                      bson:"detail"`
	CalculatedAmount *money.Money `json:"calculated_amount,omitempty" bson:"calculated_amount,omitempty"`
	Reference        string           `json:"reference"                   bson:"reference"`
}
