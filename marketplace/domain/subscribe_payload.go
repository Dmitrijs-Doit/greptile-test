package domain

type SubscribePayload struct {
	ProcurementAccountID string  `json:"procurementAccountId"`
	CustomerID           string  `json:"customerId"`
	Email                string  `json:"email"`
	UID                  string  `json:"uid"`
	Product              Product `json:"product"`
	UserID               string  `json:"firestoreUserId"`
}
