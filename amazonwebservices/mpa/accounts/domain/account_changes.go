package domain

type AccountChanges struct {
	Before Account `json:"before"`
	After  Account `json:"after"`
}

type Account struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	ARN    string `json:"arn"`
	Email  string `json:"email"`
	Payer  Payer  `json:"payerAccount"`
}

type Payer struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}
