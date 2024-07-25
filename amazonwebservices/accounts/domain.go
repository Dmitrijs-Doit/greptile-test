package accounts

import (
	"time"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
)

type PayerAccount struct {
	AccountID   string `firestore:"id"`
	DisplayName string `firestore:"displayName"`
}

type Account struct {
	ID              string               `firestore:"id"`
	Name            string               `firestore:"name"`
	Status          string               `firestore:"status"`
	Arn             string               `firestore:"arn"`
	Email           string               `firestore:"email"`
	JoinedMethod    string               `firestore:"joinedMethod"`
	JoinedTimestamp time.Time            `firestore:"joinedTimestamp"`
	Timestamp       time.Time            `firestore:"timestamp,serverTimestamp"`
	PayerAccount    *domain.PayerAccount `firestore:"payerAccount"`
}
