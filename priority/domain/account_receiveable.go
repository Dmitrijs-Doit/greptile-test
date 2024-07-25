package domain

type AccountsReceivable struct {
	Value []*AccountReceivable `json:"value"`
}

type AccountReceivable struct {
	ID      string  `json:"ACCNAME"`
	Code    *string `json:"CODE"`
	Balance float64 `json:"BALANCE3"`
}
