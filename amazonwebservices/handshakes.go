package amazonwebservices

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudhealth"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
)

type Handshake struct {
	ID                  string                 `firestore:"id"`
	Target              string                 `firestore:"target"`
	PayerAccount        *domain.PayerAccount   `firestore:"payerAccount"`
	Requester           string                 `firestore:"requester"`
	Arn                 string                 `firestore:"arn"`
	State               string                 `firestore:"state"`
	Visible             bool                   `firestore:"visible"`
	Action              string                 `firestore:"action"`
	ExpirationTimestamp time.Time              `firestore:"expirationTimestamp"`
	RequestedTimestamp  time.Time              `firestore:"requestedTimestamp"`
	Parties             []*HandshakeParty      `firestore:"parties"`
	Customer            *firestore.DocumentRef `firestore:"customer"`
	Entity              *firestore.DocumentRef `firestore:"entity"`
	Timestamp           time.Time              `firestore:"timestamp,serverTimestamp"`
}

type HandshakeParty struct {
	ID   string `firestore:"id"`
	Type string `firestore:"type"`
}

func (s *AWSService) UpdateHandshakes(ctx context.Context) error {
	l := logger.FromContext(ctx)
	fs := s.conn.Firestore(ctx)

	masterPayerAccounts, err := dal.GetMasterPayerAccounts(ctx, fs)
	if err != nil {
		return err
	}

	for _, mpa := range masterPayerAccounts.Accounts {
		if mpa.Status != domain.MasterPayerAccountStatusActive {
			continue
		}

		creds, err := mpa.NewCredentials("doit-navigator-handshakes")
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

		if err := svc.ListHandshakesForOrganizationPages(&organizations.ListHandshakesForOrganizationInput{},
			func(page *organizations.ListHandshakesForOrganizationOutput, lastPage bool) bool {
				for _, handshake := range page.Handshakes {
					var prevHandshake *Handshake

					if err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
						handshakeRef := fs.Collection("integrations").Doc("amazon-web-services").Collection("handshakes").Doc(*handshake.Id)

						docSnap, err := tx.Get(handshakeRef)
						if err != nil && status.Code(err) != codes.NotFound {
							return err
						}

						if !docSnap.Exists() {
							return nil
						}

						var data Handshake

						if err := docSnap.DataTo(&data); err != nil {
							return err
						}

						if data.State == organizations.HandshakeStateOpen && data.State != *handshake.State {
							prevHandshake = &data

							if err := tx.Update(handshakeRef, []firestore.Update{
								{FieldPath: []string{"state"}, Value: *handshake.State},
								{FieldPath: []string{"visible"}, Value: *handshake.State != organizations.HandshakeStateAccepted},
								{FieldPath: []string{"timestamp"}, Value: firestore.ServerTimestamp},
							}); err != nil {
								return err
							}
						}

						return nil
					}, firestore.MaxAttempts(10)); err != nil {
						l.Errorf("list handshakes transaction error: %s", err)
						continue
					}

					if prevHandshake != nil {
						onHandshakeStateChanged(ctx, fs, prevHandshake, handshake, svc, mpa)
					}
				}

				return !lastPage
			}); err != nil {
			l.Errorf("aws mpa %s list handshakes error: %s", mpa.Name, err)
			continue
		}
	}

	return nil
}

func onHandshakeStateChanged(ctx context.Context, fs *firestore.Client, prevHandshake *Handshake, handshake *organizations.Handshake, svc *organizations.Organizations, payerAccount *domain.MasterPayerAccount) {
	l := logger.FromContext(ctx)

	refs := []*firestore.DocumentRef{prevHandshake.Customer, prevHandshake.Entity}

	docs, err := fs.GetAll(ctx, refs)
	if err != nil {
		l.Error(err)
		return
	}

	var (
		customer common.Customer
		entity   common.Entity
	)

	if err1, err2 := docs[0].DataTo(&customer), docs[1].DataTo(&entity); err1 != nil || err2 != nil {
		l.Error(err1)
		l.Error(err2)

		return
	}

	l.Info(customer)
	l.Info(entity)

	var account *organizations.Account

	var color string

	switch *handshake.State {
	case organizations.HandshakeStateAccepted:
		color = "#4CAF50"

		output, err := svc.DescribeAccount(&organizations.DescribeAccountInput{
			AccountId: aws.String(prevHandshake.Target),
		})
		if err != nil {
			l.Error(err.Error())
		} else {
			account = output.Account
			accountID := *account.Id

			if !payerAccount.IsDedicatedPayer() {
				if _, err := createCloudhealthResources(ctx, fs, prevHandshake.Customer, prevHandshake.Entity, customer, entity); err != nil {
					l.Warningf("failed to create cloudhealth resources for %s:\n%s", accountID, err.Error())
				}

				if err := sendCloudhealthRoleInstructions(account, prevHandshake); err != nil {
					l.Warningf("failed to send cht instructions for  %s:\n%s", accountID, err.Error())
				}
			}

			if err := createAwsAccountAsset(ctx, fs, account, prevHandshake.PayerAccount, prevHandshake.Customer, prevHandshake.Entity); err != nil {
				l.Warningf("failed to create aws account asset for  %s:\n%s", accountID, err.Error())
			}
		}
	case organizations.HandshakeStateCanceled:
		color = "#607D8B"
	case organizations.HandshakeStateExpired:
		color = "#9C27B0"
	case organizations.HandshakeStateDeclined:
		color = "#F44336"
	default:
		color = "#FF9800"
	}

	fields := []map[string]interface{}{
		{
			"title": "Customer",
			"value": fmt.Sprintf("<https://console.doit.com/customers/%s|%s>", prevHandshake.Customer.ID, customer.Name),
			"short": true,
		},
		{
			"title": "Priority ID",
			"value": entity.PriorityID,
			"short": true,
		},
		{
			"title": "Payer Account",
			"value": prevHandshake.PayerAccount.DisplayName,
			"short": true,
		},
		{
			"title": "State",
			"value": strings.Title(strings.ToLower(*handshake.State)),
			"short": true,
		},
	}

	if account != nil {
		fields = append(fields,
			map[string]interface{}{
				"title": "Account",
				"value": *account.Id,
				"short": true,
			}, map[string]interface{}{
				"title": "Email",
				"value": *account.Email,
				"short": true,
			})
	} else {
		fields = append(fields, map[string]interface{}{
			"title": "Account",
			"value": prevHandshake.Target,
			"short": true,
		})
	}

	message := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":          time.Now().Unix(),
				"color":       color,
				"author_name": fmt.Sprintf("<mailto:%s|%s>", prevHandshake.Requester, prevHandshake.Requester),
				"title":       "Invitation Status Update",
				"title_link":  fmt.Sprintf("https://console.doit.com/customers/%s/assets/amazon-web-services", prevHandshake.Customer.ID),
				"thumb_url":   "https://storage.googleapis.com/hello-static-assets/logos/amazon-web-services.png",
				"fields":      fields,
			},
		},
	}
	if _, err := common.PublishToSlack(ctx, message, slackChannel); err != nil {
		l.Error(err)
	}
}

// createCloudhealthResources creates a customer object in CHT
func createCloudhealthResources(ctx context.Context, fs *firestore.Client, customerRef, entityRef *firestore.DocumentRef, customer common.Customer, entity common.Entity) (*cloudhealth.Customer, error) {
	chtCutomersCollection := fs.Collection(IntegrationsCloudHealthCustomersCollection)

	// Check if we already have a cloudhealth customer
	docSnaps, err := chtCutomersCollection.
		Where("customer", "==", customerRef).
		Where("disabled", "==", false).
		Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	if len(docSnaps) == 0 {
		chtCustomer, err := createCloudhealthCustomer(&cloudhealth.Customer{
			Name:                        customer.PrimaryDomain,
			Classification:              cloudhealth.ClassificationManagedWithAccess,
			PartnerBillingConfiguration: cloudhealth.PartnerBillingConfiguration{Enabled: true},
			Address: cloudhealth.Address{
				Street1: "Rav Aluf David Elazar 12",
				City:    "Tel Aviv",
				State:   "Tel Aviv",
				ZipCode: entity.PriorityID,
				Country: "IL",
			},
		})
		if err != nil {
			return nil, err
		}

		chtCustomerID := strconv.FormatInt(chtCustomer.ID, 10)
		if _, err := chtCutomersCollection.Doc(chtCustomerID).Set(ctx, IntegrationCloudHealthCustomer{
			ID:       chtCustomer.ID,
			Name:     chtCustomer.Name,
			Customer: customerRef,
			Disabled: false,
		}); err != nil {
			return nil, err
		}

		return chtCustomer, nil
	}

	chtCustomerID := docSnaps[0].Ref.ID
	path := fmt.Sprintf("/v1/customers/%s", chtCustomerID)

	body, err := cloudhealth.Client.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var chtCustomer cloudhealth.Customer
	if err := json.Unmarshal(body, &chtCustomer); err != nil {
		return nil, err
	}

	return &chtCustomer, nil
}

func createCloudhealthCustomer(input *cloudhealth.Customer) (*cloudhealth.Customer, error) {
	path := "/v1/customers"

	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	body, err := cloudhealth.Client.Post(path, nil, data)
	if err != nil {
		return nil, err
	}

	var customer cloudhealth.Customer
	if err := json.Unmarshal(body, &customer); err != nil {
		return nil, err
	}

	return &customer, nil
}

func createCloudhealthAccount(customer *cloudhealth.Customer, input *cloudhealth.AwsAccount) (*cloudhealth.AwsAccount, error) {
	path := "/v1/aws_accounts"

	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	var params map[string][]string
	if customer != nil {
		params = make(map[string][]string)
		params["client_api_id"] = []string{strconv.FormatInt(customer.ID, 10)}
	}

	body, err := cloudhealth.Client.Post(path, params, data)
	if err != nil {
		return nil, err
	}

	var account cloudhealth.AwsAccount
	if err := json.Unmarshal(body, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

func sendCloudhealthRoleInstructions(account *organizations.Account, handshake *Handshake) error {
	data := struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}{
		ID:    *account.Id,
		Name:  *account.Name,
		Email: *account.Email,
	}

	m := mail.NewV3Mail()
	m.SetTemplateID(mailer.Config.DynamicTemplates.AmazonWebServicesInviteAccepted)
	m.SetFrom(mail.NewEmail(mailer.Config.NoReplyName, mailer.Config.NoReplyEmail))

	enable := false
	m.SetTrackingSettings(&mail.TrackingSettings{SubscriptionTracking: &mail.SubscriptionTrackingSetting{Enable: &enable}})

	emails := []*mail.Email{mail.NewEmail("", *account.Email)}
	if *account.Email != handshake.Requester {
		emails = append(emails, mail.NewEmail("", handshake.Requester))
	}

	personalization := mail.NewPersonalization()
	personalization.AddTos(emails...)
	personalization.SetDynamicTemplateData("data", data)
	m.AddPersonalizations(personalization)

	request := sendgrid.GetRequest(mailer.Config.APIKey, mailer.Config.MailSendPath, mailer.Config.BaseURL)
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)

	if _, err := sendgrid.MakeRequestRetry(request); err != nil {
		return err
	}

	return nil
}
