package common

// Partners represents partenrs doc under the path '/app/partner-access'
type Partners struct {
	Partners []*Partner `firestore:"partners"`
}

// Partner represents a single partner data
type Partner struct {
	Name    string   `firestore:"name"`
	Domains []string `firestore:"domains"`
	Company string   `firestore:"company"`
	Auth    Auth     `firestore:"auth"`
}
