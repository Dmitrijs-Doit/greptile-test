package service

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
)

// Deprecated. Use MPA DAL inside scheduled-tasks/amazonwebservices/accounts/service. Use MPA Service outsite.
// GetPayerAccount retrieves a payer account object from the list of payer accounts
func GetPayerAccount(ctx context.Context, fs *firestore.Client, accountID string) (*domain.PayerAccount, error) {
	account, err := dal.NewMasterPayerAccountDALWithClient(fs).GetMasterPayerAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	return &domain.PayerAccount{AccountID: account.AccountNumber, DisplayName: account.FriendlyName}, nil
}
