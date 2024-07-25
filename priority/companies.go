package priority

import "github.com/doitintl/hello/scheduled-tasks/fixer"

// CompanyCode represents a priority company ID
type CompanyCode string

// Company codes that represents the priority companies IDs
const (
	CompanyCodeISR CompanyCode = "doit"
	CompanyCodeUSA CompanyCode = "doitint"
	CompanyCodeUK  CompanyCode = "doituk"
	CompanyCodeAUS CompanyCode = "doitaus"
	CompanyCodeDE  CompanyCode = "doitde"
	CompanyCodeFR  CompanyCode = "doitfr"
	CompanyCodeNL  CompanyCode = "doitnl"
	CompanyCodeCH  CompanyCode = "doitch"
	CompanyCodeCA  CompanyCode = "doitca"
	CompanyCodeSE  CompanyCode = "doitse"
	CompanyCodeES  CompanyCode = "doites"
	CompanyCodeIE  CompanyCode = "doitie"
	CompanyCodeEE  CompanyCode = "doitee"
	CompanyCodeSG  CompanyCode = "doitsg"
	CompanyCodeJP  CompanyCode = "doitjp"
	CompanyCodeID  CompanyCode = "doitid"
)

var DefaultCurrency = map[CompanyCode]fixer.Currency{
	CompanyCodeISR: fixer.ILS,
	CompanyCodeUSA: fixer.USD,
	CompanyCodeUK:  fixer.GBP,
	CompanyCodeAUS: fixer.AUD,
	CompanyCodeDE:  fixer.EUR,
	CompanyCodeFR:  fixer.EUR,
	CompanyCodeNL:  fixer.EUR,
	CompanyCodeCH:  fixer.CHF,
	CompanyCodeCA:  fixer.CAD,
	CompanyCodeSE:  fixer.SEK,
	CompanyCodeES:  fixer.EUR,
	CompanyCodeIE:  fixer.EUR,
	CompanyCodeEE:  fixer.EUR,
	CompanyCodeSG:  fixer.SGD,
	CompanyCodeJP:  fixer.JPY,
	CompanyCodeID:  fixer.IDR,
}

var Companies = []CompanyCode{
	CompanyCodeISR,
	CompanyCodeUSA,
	CompanyCodeUK,
	CompanyCodeAUS,
	CompanyCodeDE,
	CompanyCodeFR,
	CompanyCodeNL,
	CompanyCodeCH,
	CompanyCodeCA,
	CompanyCodeSE,
	CompanyCodeES,
	CompanyCodeIE,
	CompanyCodeEE,
	CompanyCodeSG,
	CompanyCodeJP,
	CompanyCodeID,
}

type CompanyInfo struct {
	CompanyID    string         `firestore:"companyId"`
	CompanyName  string         `firestore:"companyName"`
	Countries    []string       `firestore:"countries"`
	WireTransfer []KeyValuePair `firestore:"wireTransfer"`
}

type KeyValuePair struct {
	Key   string `firestore:"key"`
	Value string `firestore:"value"`
}

// CompaniesDetails can be used to update the firestore document app/priority-v2
// Accounts info in go/banks
var CompaniesDetails = []CompanyInfo{
	{
		CompanyID:   "doitint",
		CompanyName: "DoiT International USA, Inc",
		Countries:   []string{"United States"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "SWIFT",
				Value: "CITIUS33",
			},
			{
				Key:   "Account",
				Value: "#31268173",
			},
			{
				Key:   "ABA",
				Value: "021000089",
			},
			{
				Key:   "Bank",
				Value: "Citibank, N.A.",
			},
		},
	},
	{
		CompanyID:   "doit",
		CompanyName: "DoiT International Ltd",
		Countries:   []string{"Israel"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "SWIFT",
				Value: "LUMIILIT",
			},
			{
				Key:   "Account",
				Value: "#340900/43",
			},
			{
				Key:   "IBAN",
				Value: "IL63 0108 6400 0003 4090 043",
			},
			{
				Key:   "Bank",
				Value: "Bank Leumi",
			},
			{
				Key:   "Branch",
				Value: "864",
			},
		},
	},
	{
		CompanyID:   "doituk",
		CompanyName: "DoiT International UK&I Ltd",
		Countries:   []string{"United Kingdom"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "Sort Code",
				Value: "185008",
			},
			{
				Key:   "Account",
				Value: "#12388871",
			},
			{
				Key:   "IBAN",
				Value: "GB91CITI18500812388871",
			},
			{
				Key:   "Bank",
				Value: "Citi Bank, United Kingdom, London",
			},
		},
	},
	{
		CompanyID:   "doitde",
		CompanyName: "DoiT International, DACH GmbH",
		Countries:   []string{"Germany"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "Account",
				Value: "#218788068",
			},
			{
				Key:   "IBAN",
				Value: "DE89502109000218788068",
			},
			{
				Key:   "Bank",
				Value: "Citi Bank Europe plc",
			},
		},
	},
	{
		CompanyID:   "doitaus",
		CompanyName: "DoiT International AUS PTY Ltd",
		Countries:   []string{"Australia"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "Account",
				Value: "#235879018",
			},
			{
				Key:   "ABN",
				Value: "34 072 814 058",
			},
			{
				Key:   "Bank",
				Value: "Citi Bank, N.A. (Sydney)",
			},
		},
	},
	{
		CompanyID:   "doitfr",
		CompanyName: "DoiT International France SAS",
		Countries:   []string{"France"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "Account",
				Value: "#0659395622",
			},
			{
				Key:   "IBAN",
				Value: "FR7611689007000065939562274",
			},
			{
				Key:   "Bank",
				Value: "Citi Bank, Paris",
			},
		},
	},
	{
		CompanyID:   "doitnl",
		CompanyName: "DoiT International NL BV",
		Countries:   []string{"Netherlands"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "Account",
				Value: "#2032315521",
			},
			{
				Key:   "IBAN",
				Value: "NL60CITI2032315521",
			},
			{
				Key:   "Bank",
				Value: "Citi Bank, N.A. (Amsterdam)",
			},
		},
	},
	{
		CompanyID:   "doitch",
		CompanyName: "DoiT International CH SARL",
		Countries:   []string{"Switzerland"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "Account",
				Value: "#14083024",
			},
			{
				Key:   "SWIFT",
				Value: "CITIGB2L",
			},
			{
				Key:   "IBAN",
				Value: "CH4189095000014083024",
			},
			{
				Key:   "Bank",
				Value: "Citi Bank",
			},
		},
	},
	{
		CompanyID:   "doitca",
		CompanyName: "DoiT Holdings International CA Ltd",
		Countries:   []string{"Canada"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "Account",
				Value: "#2012890007",
			},
			{
				Key:   "SWIFT",
				Value: "CITICATTBHC",
			},
			{
				Key:   "ABA",
				Value: "032820012",
			},
			{
				Key:   "Bank",
				Value: "Citibank, N.A. Canadian Branch",
			},
		},
	},
	{
		CompanyID:   "doitse",
		CompanyName: "DoiT Multi-Cloud Sverige International AB",
		Countries:   []string{"Sweden"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "Account",
				Value: "#90401033378",
			},
			{
				Key:   "SWIFT",
				Value: "CITISESX",
			},
			{
				Key:   "IBAN",
				Value: "SE8290400000090401033378",
			},
			{
				Key:   "Bank",
				Value: "Citi Bank",
			},
		},
	},
	{
		CompanyID:   "doites",
		CompanyName: "DoiT International Multi-Cloud Espana S.L",
		Countries:   []string{"Spain"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "Account",
				Value: "#15968613",
			},
			{
				Key:   "SWIFT",
				Value: "CITIESMX",
			},
			{
				Key:   "IBAN",
				Value: "ES6014740000120015968613",
			},
			{
				Key:   "Bank",
				Value: "Citi Bank",
			},
		},
	},
	{
		CompanyID:   "doitie",
		CompanyName: "DoiT International Multi-Cloud Ireland Ltd",
		Countries:   []string{"Ireland"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "Account",
				Value: "#39119749",
			},
			{
				Key:   "SWIFT",
				Value: "CITIIE2X",
			},
			{
				Key:   "IBAN",
				Value: "IE13CITI99005139119749",
			},
			{
				Key:   "Bank",
				Value: "Citi Bank",
			},
		},
	},
	{
		CompanyID:   "doitee",
		CompanyName: "DoiT Multi-Cloud International Estonia OÜ",
		Countries:   []string{"Estonia"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "Account",
				Value: "#39119749",
			},
			{
				Key:   "SWIFT",
				Value: "HABAEE2X",
			},
			{
				Key:   "IBAN",
				Value: "EE522200221079129678",
			},
		},
	},
	{
		CompanyID:   "doitsg",
		CompanyName: "DoiT International USA, Inc",
		Countries:   []string{"Singapore"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "SWIFT",
				Value: "SVBKUS6S",
			},
			{
				Key:   "Account",
				Value: "#3302149895",
			},
			{
				Key:   "ABA",
				Value: "121140399",
			},
			{
				Key:   "Bank",
				Value: "SILICON VALLEY BANK",
			},
		},
	},
	{
		CompanyID:   "doitjp",
		CompanyName: "DoiT International Japan Co., Ltd",
		Countries:   []string{"Japan"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "SWIFT",
				Value: "CITIUS33",
			},
			{
				Key:   "Account",
				Value: "#31268173",
			},
			{
				Key:   "ABA",
				Value: "021000089",
			},
			{
				Key:   "Bank",
				Value: "Citibank, N.A.",
			},
		},
	},
	{
		CompanyID:   "doitid",
		CompanyName: "PT DoiT International Indonesia",
		Countries:   []string{"Indonesia"},
		WireTransfer: []KeyValuePair{
			{
				Key:   "SWIFT",
				Value: "CITIUS33",
			},
			{
				Key:   "Account",
				Value: "#31268173",
			},
			{
				Key:   "ABA",
				Value: "021000089",
			},
			{
				Key:   "Bank",
				Value: "Citibank, N.A.",
			},
		},
	},
}

func IsExportCountry(companyCode CompanyCode, country string) bool {
	switch companyCode {
	case CompanyCodeISR:
		return country != "Israel"
	case CompanyCodeUSA:
		return country != "United States"
	case CompanyCodeUK:
		return country != "United Kingdom"
	case CompanyCodeAUS:
		return country != "Australia"
	case CompanyCodeDE:
		return country != "Germany"
	case CompanyCodeFR:
		return country != "France"
	case CompanyCodeNL:
		return country != "Netherlands"
	case CompanyCodeCH:
		return country != "Switzerland"
	case CompanyCodeCA:
		return country != "Canada"
	case CompanyCodeSE:
		return country != "Sweden"
	case CompanyCodeES:
		return country != "Spain"
	case CompanyCodeIE:
		return country != "Ireland"
	case CompanyCodeEE:
		return country != "Estonia"
	case CompanyCodeSG:
		return country != "Singapore"
	case CompanyCodeJP:
		return country != "Japan"
	case CompanyCodeID:
		return country != "Indonesia"
	default:
		return true
	}
}
