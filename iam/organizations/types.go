package organizations

type RemoveIAMOrgsRequest struct {
	CustomerID string
	OrgIDs     []string `json:"orgIds"`
	UserID     string
}
