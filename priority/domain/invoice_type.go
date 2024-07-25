package domain

type InvoiceType string

const (
	ForeignInvoiceType InvoiceType = "F"
	LocalInvoiceType   InvoiceType = "A"
	ReceiptType        InvoiceType = "T"
)

func (it InvoiceType) String() string {
	return string(it)
}

func ToInvoiceType(str string) InvoiceType {
	if str == ForeignInvoiceType.String() {
		return ForeignInvoiceType
	}

	return LocalInvoiceType
}
