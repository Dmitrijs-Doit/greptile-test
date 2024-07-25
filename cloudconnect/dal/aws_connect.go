package dal

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	notificationcenter "github.com/doitintl/notificationcenter/pkg"
)

type AwsConnect struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

type Spot0CustomerFlags struct {
	Customer         *firestore.DocumentRef `firestore:"customer"`
	WelcomeEmailSent *time.Time             `firestore:"welcomeEmailSent"`
}

type MailRecipient struct {
	Email     string
	FirstName string
}

const welcomeEmailTemplate = "VW6NVJHS6HMPJCJ61RWZNFA09JFR"

func NewAwsConnect(ctx context.Context) (IAwsConnect, error) {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("could not initialize firestore client. error %s", err)
	}

	return NewAwsConnectWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

func NewAwsConnectWithClient(fun connection.FirestoreFromContextFun) IAwsConnect {
	return &AwsConnect{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (a *AwsConnect) Close(ctx context.Context) {
	a.firestoreClientFun(ctx).Close()
}

func (a *AwsConnect) GetSpot0CustomerFlags(ctx context.Context, customerID string) (*Spot0CustomerFlags, error) {
	docRef := a.firestoreClientFun(ctx).Collection("spot0").Doc("spotApp").Collection("spot0Customers").Doc(customerID)

	docSnap, err := a.documentsHandler.Get(ctx, docRef)
	if err != nil {
		return nil, err
	}

	var spot0CustomerFlags Spot0CustomerFlags
	if err = docSnap.DataTo(&spot0CustomerFlags); err != nil {
		return nil, err
	}

	return &spot0CustomerFlags, nil
}

func (a *AwsConnect) SetSpot0CustomerFlags(ctx context.Context, customerID string) error {
	var spot0CustomerFlags Spot0CustomerFlags

	customerRef := a.firestoreClientFun(ctx).Collection("customers").Doc(customerID)
	timeSent := time.Now()
	spot0CustomerFlags.Customer = customerRef
	spot0CustomerFlags.WelcomeEmailSent = &timeSent

	docRef := a.firestoreClientFun(ctx).Collection("spot0").Doc("spotApp").Collection("spot0Customers").Doc(customerID)
	_, err := a.documentsHandler.Set(ctx, docRef, spot0CustomerFlags)

	return err
}

func (a *AwsConnect) GetCustomer(ctx context.Context, customerID string) (*common.Customer, error) {
	customerRef := a.firestoreClientFun(ctx).Collection("customers").Doc(customerID)
	return common.GetCustomer(ctx, customerRef)
}

func (a *AwsConnect) GetCustomerAccountManagers(ctx context.Context, customer *common.Customer, company common.AccountManagerCompany) ([]*common.AccountManager, error) {
	return common.GetCustomerAccountManagers(ctx, customer, company)
}

func (a *AwsConnect) GetCustomerAdmins(ctx context.Context, customerID string) ([]common.User, error) {
	var admins []common.User

	customerRef := a.firestoreClientFun(ctx).Collection("customers").Doc(customerID)
	adminRoleRef := a.firestoreClientFun(ctx).Collection("roles").Doc(string(common.PresetRoleAdmin))

	docIter := a.firestoreClientFun(ctx).Collection("users").
		Where("customer.ref", "==", customerRef).
		Where("roles", "array-contains", adminRoleRef).
		Documents(ctx)

	docSnaps, err := a.documentsHandler.GetAll(docIter)
	if err != nil {
		return admins, err
	}

	for _, docSnap := range docSnaps {
		var user common.User
		if err := docSnap.DataTo(&user); err != nil {
			return admins, err
		}

		admins = append(admins, user)
	}

	return admins, nil
}

func (a *AwsConnect) SendMail(ctx context.Context, mailRecipients []MailRecipient, bccs []string, companyName string) error {
	if common.Production {
		client, err := notificationcenter.NewClient(ctx, common.ProjectID)
		if err != nil {
			return err
		}

		for _, mailRecipient := range mailRecipients {
			reqID, err := client.Send(
				ctx,
				notificationcenter.Notification{
					Email:    []string{mailRecipient.Email},
					BCC:      bccs,
					Template: welcomeEmailTemplate,
					Data: map[string]interface{}{
						"firstName": mailRecipient.FirstName,
						"company":   companyName,
					},
				},
			)
			log.Printf("sent welcome email, request ID: %s\n", reqID)

			if err != nil {
				return err
			}
		}
	}

	return nil
}
