package dal

import (
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
)

type Procedure string

const (
	AvalaraInvoices Procedure = "AVALARAINVOICES"
	CloseAnInvoice  Procedure = "CLOSEANINVOICE"
	DeleteAnInvoice Procedure = "DELCIV"

	PrintIsraelFInvoices  Procedure = "ROTL_WWWSHOWFIV2"
	PrintIsraelAInvoices  Procedure = "ROTL_WWWSHOWAIV_ENG"
	PrintForeignFInvoices Procedure = "ROTL_WWWSHOWFIVUSA"
	PrintForeignAInvoices Procedure = "ROTL_WWWSHOWAIV_USA"
)

func (p Procedure) String() string {
	return string(p)
}

func getPrintAnInvoiceProcedureName(customerCountryName string, invoiceType priorityDomain.InvoiceType) Procedure {
	if customerCountryName == "Israel" {
		if invoiceType == priorityDomain.ForeignInvoiceType {
			return PrintIsraelFInvoices
		}

		return PrintIsraelAInvoices
	}

	if invoiceType == priorityDomain.ForeignInvoiceType {
		return PrintForeignFInvoices
	}

	return PrintForeignAInvoices
}

func getPrintAnInvoiceProcedureFormat(customerCountryName string, invoiceType priorityDomain.InvoiceType) int {
	switch customerCountryName {
	case "Israel":
		if invoiceType == priorityDomain.ForeignInvoiceType {
			return -103
		}

		return -102
	case "United States":
		if invoiceType == priorityDomain.ForeignInvoiceType {
			return -101
		}

		return -102
	case "United Kingdom":
		if invoiceType == priorityDomain.ForeignInvoiceType {
			return -102
		}

		return -104
	case "Australia":
		if invoiceType == priorityDomain.ForeignInvoiceType {
			return -104
		}

		return -103
	case "Germany":
		return -105
	case "France":
		if invoiceType == priorityDomain.ForeignInvoiceType {
			return -101
		}

		return -106
	case "Netherlands":
		if invoiceType == priorityDomain.ForeignInvoiceType {
			return -108
		}

		return -107
	case "Switzerland":
		if invoiceType == priorityDomain.ForeignInvoiceType {
			return -101
		}

		return -108
	case "Canada":
		if invoiceType == priorityDomain.ForeignInvoiceType {
			return -101
		}

		return -110
	case "Sweden":
		if invoiceType == priorityDomain.ForeignInvoiceType {
			return -103
		}

		return -109
	case "Spain":
		if invoiceType == priorityDomain.ForeignInvoiceType {
			return -106
		}

		return -111
	case "Ireland":
		if invoiceType == priorityDomain.ForeignInvoiceType {
			return -101
		}

		return -112
	case "Estonia":
		if invoiceType == priorityDomain.ForeignInvoiceType {
			return -107
		}

		return -113
	case "Singapore":
		return -101
	}

	return 0
}
