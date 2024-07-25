package generatedaccounts

import (
	"context"
	"fmt"
	"strconv"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/generatedaccounts/domain"
)

const (
	emailSuffix string = "@aws-reg.doit-intl.com"
	maxRetries  int    = 29
)

func (s *Service) CreateAccountsBatch(ctx context.Context, req *CreateAccountsBatchRequest) error {
	for i := req.FromIndex; i <= req.ToIndex; i++ {
		zeroPaddedIndex := fmt.Sprintf("%0"+strconv.Itoa(req.ZeroPadding)+"d", i)
		accountName := req.AccountNamePrefix + zeroPaddedIndex
		email := req.EmailPrefix + zeroPaddedIndex + emailSuffix

		awsAccountID, err := s.awsAccountsDAL.GetAwsAccountIDByEmail(ctx, email)
		if err != nil {
			return err
		}

		if awsAccountID == nil {
			var err error
			awsAccountID, err = s.awsAccountsDAL.CreateAwsAccount(ctx, accountName, email)

			if err != nil {
				return err
			}
		}

		awsAccountCommand, awsAccountCommandRef, err := s.awsAccountsDAL.GetAwsAccountCommandByAccountID(ctx, *awsAccountID)
		if err != nil {
			return err
		}

		shouldRemoveOldCommandAndCreateNewOne := awsAccountCommand != nil && awsAccountCommand.RetryCount >= maxRetries

		if shouldRemoveOldCommandAndCreateNewOne {
			_, err := awsAccountCommandRef.Delete(ctx)
			if err != nil {
				return err
			}
		}

		if awsAccountCommand == nil || shouldRemoveOldCommandAndCreateNewOne {
			_, err := s.awsAccountsDAL.CreateAwsAccountCommand(ctx, *awsAccountID, domain.AwsAccountCommandTypeCreateAwsAccount)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
