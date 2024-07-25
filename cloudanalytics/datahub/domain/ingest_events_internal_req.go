package domain

type IngestEventsInternalReq struct {
	CustomerID string   `json:"customerId"`
	Email      string   `json:"email"`
	Execute    bool     `json:"execute"`
	Source     string   `json:"source"`
	FileName   string   `json:"filename"`
	Events     []*Event `json:"events"`
}
