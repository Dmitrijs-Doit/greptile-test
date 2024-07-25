package amazonwebservices

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/organizations"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

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
	PostMortem      *PostMortem          `firestore:"postMortem"`
}

type PostMortem struct {
	Status        string    `firestore:"status"`
	LeftTimestamp time.Time `firestore:"leftTimestamp"`
}

func (s *AWSService) UpdateAccounts(ctx context.Context) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	flexsaveAccountIDs, err := s.flexsaveAPI.ListFlexsaveAccountsWithCache(ctx, time.Minute*10)
	if err != nil {
		return err
	}

	masterPayerAccounts, err := dal.GetMasterPayerAccounts(ctx, fs)
	if err != nil {
		return err
	}

	updateCompletedSuccessfully := true
	startTime := time.Now().UTC()

	for _, mpa := range masterPayerAccounts.Accounts {
		if mpa.Status != domain.MasterPayerAccountStatusActive {
			continue
		}

		creds, err := mpa.NewCredentials("")
		if err != nil {
			l.Errorf("aws mpa %s new credentials error: %s", mpa.Name, err)
			continue
		}

		sess, err := session.NewSession(&aws.Config{
			Region:      aws.String(endpoints.UsEast1RegionID),
			Credentials: creds,
		})
		if err != nil {
			l.Errorf("aws mpa %s session error: %s", mpa.Name, err)
			continue
		}

		svc := organizations.New(sess)

		if err := svc.ListAccountsPages(&organizations.ListAccountsInput{},
			func(page *organizations.ListAccountsOutput, lastPage bool) bool {
				batch := fs.Batch()
				for _, account := range page.Accounts {
					docID := fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, *account.Id)
					accountRef := fs.Collection("integrations").Doc(common.Assets.AmazonWebServices).Collection("accounts").Doc(docID)
					payerAccount := &domain.PayerAccount{
						AccountID:   mpa.AccountNumber,
						DisplayName: mpa.FriendlyName,
					}
					batch.Set(accountRef, &Account{
						ID:              *account.Id,
						Name:            *account.Name,
						Status:          *account.Status,
						Arn:             *account.Arn,
						Email:           *account.Email,
						JoinedMethod:    *account.JoinedMethod,
						JoinedTimestamp: *account.JoinedTimestamp,
						PayerAccount:    payerAccount,
					})

				}
				if _, err := batch.Commit(ctx); err != nil {
					l.Errorf("list accounts batch commit error: %s", err)
					updateCompletedSuccessfully = false
				}
				return !lastPage
			}); err != nil {
			l.Errorf("aws mpa %s list accounts error: %s", mpa.Name, err)
			continue
		}
	}

	if !updateCompletedSuccessfully {
		return errors.New("update accounts did not complete successfully")
	}

	docs, err := fs.Collection("integrations").Doc("amazon-web-services").Collection("accounts").Where("timestamp", "<", startTime).Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, docSnap := range docs {
		var account Account
		if err := docSnap.DataTo(&account); err != nil {
			l.Error(err)
			continue
		}

		masterPayerAccount, ok := masterPayerAccounts.Accounts[account.PayerAccount.AccountID]
		if !ok {
			l.Error(fmt.Errorf("missing master payer account for accountID: %s", account.PayerAccount.AccountID))
			continue
		}

		creds, err := masterPayerAccount.NewCredentials("")
		if err != nil {
			l.Error(err)
			continue
		}

		sess, err := session.NewSession(&aws.Config{
			Region:      aws.String(endpoints.UsEast1RegionID),
			Credentials: creds,
		})
		if err != nil {
			l.Error(err)
			continue
		}

		svc := organizations.New(sess)

		if _, err := svc.DescribeAccount(&organizations.DescribeAccountInput{AccountId: aws.String(account.ID)}); err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case organizations.ErrCodeAccountNotFoundException:
					if account.PostMortem == nil {
						// notify
						isFlexsaveAccount := slice.Contains(flexsaveAccountIDs, account.ID)

						if !isFlexsaveAccount {
							if err := onAccountLeftOrganization(fs, docSnap.Ref, &account); err != nil {
								l.Warningf("onAccountLeftOrganization failed with error: %s", err)
							}
						}

						// add post-mortem report
						_, err := docSnap.Ref.Update(ctx, []firestore.Update{
							{Path: "postMortem", Value: &PostMortem{Status: "LEFT", LeftTimestamp: time.Now().UTC()}},
						})
						if err != nil {
							l.Errorf("failed to add post-mortem to aws account ref with error: %s", err)
						}
					}
				default:
				}
			}
		}
	}

	return nil
}

func onAccountLeftOrganization(fs *firestore.Client, accountRef *firestore.DocumentRef, account *Account) error {
	ctx := context.Background()

	docSnap, err := fs.Collection("assetSettings").Doc(accountRef.ID).Get(ctx)
	if err != nil {
		return err
	}

	var as common.AssetSettings
	if err := docSnap.DataTo(&as); err != nil {
		return err
	}

	// Don't post alerts for the msp.doit.com customer
	if as.Customer.ID == "6QJMHMUaIYdEShihSweH" {
		return nil
	}

	customerDocSnap, err := as.Customer.Get(ctx)
	if err != nil {
		return err
	}

	var customer common.Customer
	if err := customerDocSnap.DataTo(&customer); err != nil {
		return err
	}

	message := map[string]interface{}{
		"text": "Everyone, get in <!here>!",
		"attachments": []map[string]interface{}{
			{
				"ts":         time.Now().Unix(),
				"color":      "#F44336",
				"title":      "Account left our organization!",
				"title_link": fmt.Sprintf("https://console.doit.com/customers/%s/assets/amazon-web-services", as.Customer.ID),
				"thumb_url":  "https://storage.googleapis.com/hello-static-assets/logos/amazon-web-services.png",
				"fields": []map[string]interface{}{
					{
						"title": "Customer",
						"value": fmt.Sprintf("<https://console.doit.com/customers/%s|%s>", as.Customer.ID, customer.Name),
						"short": true,
					},
					{
						"title": "Payer Account",
						"value": account.PayerAccount.DisplayName,
						"short": true,
					},
					{
						"title": "Email",
						"value": account.Email,
						"short": true,
					},
					{
						"title": "Account",
						"value": account.ID,
						"short": true,
					},
				},
			},
		},
	}

	if _, err := common.PublishToSlack(ctx, message, slackChannel); err != nil {
		return err
	}

	return nil
}

func GetAccountOrganization(ctx context.Context, fs *firestore.Client, docID string) *Account {
	accountRef := fs.Collection("integrations").Doc("amazon-web-services").Collection("accounts").Doc(docID)

	docSnap, err := accountRef.Get(ctx)
	if err != nil {
		return nil
	}

	var account Account
	if err := docSnap.DataTo(&account); err != nil {
		return nil
	}

	return &account
}
