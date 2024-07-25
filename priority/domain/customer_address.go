package domain

// CustomerAddress represents a customer's address in priority
type CustomerAddress struct {
	CountryName string `json:"COUNTRYNAME"`
	Address     string `json:"ADDRESS"`
	City        string `json:"ADDRESS2"`
	StateName   string `json:"STATENAME"`
	StateA      string `json:"STATEA"`
	State       string `json:"STATE"`
	Zip         string `json:"ZIP"`
}
