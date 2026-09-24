package model

import (
	"fmt"
	"regexp"
	"strings"
)

// Formatos aceitos para códigos de país (ISO 3166-1 alpha-2) e de moeda
// (ISO 4217). Valida-se só o formato; a lista oficial não é consultada.
var (
	countryCodePattern  = regexp.MustCompile(`^[A-Z]{2}$`)
	currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)
)

// FieldError descreve um campo inválido da requisição.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError agrega todos os campos inválidos de uma requisição, para
// que o cliente corrija tudo de uma vez em vez de descobrir erro por erro.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		parts[i] = f.Field + ": " + f.Message
	}
	return fmt.Sprintf("requisicao invalida: %s", strings.Join(parts, "; "))
}

// Validate confere se a requisição tem tudo o que o motor precisa para
// avaliá-la. Sem isso, zero values (valor 0, método vazio, nome vazio) fariam
// as regras "não se aplicarem" e a operação sairia conforme por omissão.
func (r EvaluationRequest) Validate() error {
	var fields []FieldError
	add := func(field, message string) {
		fields = append(fields, FieldError{Field: field, Message: message})
	}

	if !r.Product.Type.IsValid() {
		add("product.type", "tipo de produto desconhecido")
	}
	if !countryCodePattern.MatchString(r.Product.Origin) {
		add("product.origin", "deve ser codigo de pais ISO 3166-1 alpha-2 (ex.: US)")
	}
	if !currencyCodePattern.MatchString(r.Product.OriginCurrency) {
		add("product.origin_currency", "deve ser codigo de moeda ISO 4217 (ex.: USD)")
	}

	amount := r.Operation.Amount
	switch {
	case !amount.IsPositive():
		add("operation.amount", "deve ser maior que zero")
	case !amount.Equal(amount.Round(2)):
		add("operation.amount", "deve ter no maximo 2 casas decimais")
	}
	if !currencyCodePattern.MatchString(r.Operation.Currency) {
		add("operation.currency", "deve ser codigo de moeda ISO 4217 (ex.: USD)")
	}
	if !r.Operation.Method.IsValid() {
		add("operation.method", "metodo de operacao desconhecido")
	}
	if strings.TrimSpace(r.Operation.Counterparty.Name) == "" {
		add("operation.counterparty.name", "obrigatorio")
	}

	if len(fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: fields}
}
