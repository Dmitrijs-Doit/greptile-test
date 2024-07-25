package model

type AsgState struct {
	AsgName               string `firestore:"asgName,omitempty"`
	AccountName           string `firestore:"accountName,omitempty"`
	AccountId             string `firestore:"accountId,omitempty"`
	Region                string `firestore:"region,omitempty"`
	Error                 string `firestore:"error"`
	SpotisizeError        bool   `firestore:"spotisizeError"`
	SpotisizeErrorDesc    string `firestore:"spotisizeErrorDesc"`
	SpotisizeNotSupported bool   `firestore:"spotisizeNotSupported"`
	ManagedStatus         string `firestore:"managedStatus"`
	Mode                  string `firestore:"mode"`
}
