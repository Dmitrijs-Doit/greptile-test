package domain

type Customers struct {
	Value []*Customer `json:"value"`
}

type Customer struct {
	ID           string               `json:"CUSTNAME"`
	Name         string               `json:"CUSTDES"`
	CountryName  *string              `json:"COUNTRYNAME"`
	InactiveFlag *string              `json:"INACTIVEFLAG"`
	PayTermDesc  *string              `json:"PAYDES"`
	Address      *string              `json:"ADDRESS"`
	Address2     *string              `json:"ADDRESS2"`
	Address3     *string              `json:"ADDRESS3"`
	State        *string              `json:"STATE"`
	Statea       *string              `json:"STATEA"`
	StateCode    *string              `json:"STATECODE"`
	StateName    *string              `json:"STATENAME"`
	Zip          *string              `json:"ZIP"`
	Personnel    []*CustomerPersonnel `json:"CUSTPERSONNEL_SUBFORM"`
}

// CustomerPersonnel represents a customer's contact person details
type CustomerPersonnel struct {
	Name      *string `json:"NAME"`
	Firm      *string `json:"FIRM"`
	FirstName *string `json:"FIRSTNAME"`
	LastName  *string `json:"LASTNAME"`
	CivFlag   *string `json:"CIVFLAG"`
	Email     *string `json:"EMAIL"`
	Phone     *string `json:"PHONENUM"`
}
