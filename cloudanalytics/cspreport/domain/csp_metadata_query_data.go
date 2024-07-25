package domain

type CSPMetadataQueryData struct {
	Cloud                          string
	BillingDataTableFullName       string
	MetadataTableFullName          string
	BindIDField                    string // billing table field with account id
	MetadataBindIDField            string
	EnchancedBillingDataQuery      string
	EnchancedBillingDataSelectFrom string
}
