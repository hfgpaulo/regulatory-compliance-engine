package model

// Este arquivo concentra os "vocabulários" do domínio como tipos próprios
// (em vez de strings soltas). Isso dá segurança de tipo, habilita o
// autocompletar e documenta os valores válidos em um único lugar.

// Domain identifica um domínio regulatório avaliado pelo motor.
type Domain string

const (
	DomainIOF     Domain = "IOF"
	DomainPLDCOAF Domain = "PLD_COAF"
)

// Status representa o resultado da avaliação em um domínio regulatório.
type Status string

const (
	StatusCompliant          Status = "compliant"           // conforme, nada a fazer
	StatusAdaptationRequired Status = "adaptation_required" // precisa de adaptação (ex.: reter IOF)
	StatusReportable         Status = "reportable"          // gera obrigação de comunicação (ex.: COAF)
)

// ProductType classifica o produto financeiro sob avaliação.
type ProductType string

const (
	ProductPersonalLoan ProductType = "personal_loan"
)

// IsValid informa se o tipo de produto é um dos valores conhecidos.
func (t ProductType) IsValid() bool {
	switch t {
	case ProductPersonalLoan:
		return true
	}
	return false
}

// OperationMethod descreve como a operação é realizada.
type OperationMethod string

const (
	MethodInternationalTransfer OperationMethod = "international_transfer"
)

// IsValid informa se o método de operação é um dos valores conhecidos.
func (m OperationMethod) IsValid() bool {
	switch m {
	case MethodInternationalTransfer:
		return true
	}
	return false
}
