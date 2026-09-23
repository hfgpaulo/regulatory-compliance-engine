package model

import "time"

// Evaluation é a avaliação persistida: um documento auto-contido que embute a
// requisição original e o veredito produzido, com identidade e data.
//
// Optou-se por um único documento (embedded) em vez de coleções separadas de
// submissões e vereditos: é o modelo idiomático de MongoDB — uma escrita, e
// uma leitura traz o quadro completo, sem "join".
type Evaluation struct {
	ID        string            `json:"id"         bson:"_id"`
	CreatedAt time.Time         `json:"created_at" bson:"created_at"`
	Request   EvaluationRequest `json:"request"    bson:"request"`
	Report    GapReport         `json:"report"     bson:"report"`
}
