package dal

import (
	"context"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/generatedaccounts/domain"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/iface"
)

type AwsAccounts interface {
	CreateAwsAccount(ctx context.Context, accountName string, email string) (*string, error)
	CreateAwsAccountCommand(ctx context.Context, accountID string, commandType domain.AwsAccountCommandType) (*string, error)
	GetAwsAccountIDByEmail(ctx context.Context, email string) (*string, error)
	GetAwsAccountCommandByAccountID(ctx context.Context, accountID string) (*domain.AwsAccountCommand, *firestore.DocumentRef, error)
}

type AwsAccountsDAL struct {
	firestoreClient  *firestore.Client
	documentsHandler iface.DocumentsHandler
}

const (
	appCollection                = "app"
	autoGeneratedAwsAccountsDoc  = "auto-generated-aws-accounts"
	awsAccountsCollection        = "awsAccounts"
	awsAccountCommandsCollection = "awsAccountCommands"
)

func NewAwsAccountsDAL(fs *firestore.Client) *AwsAccountsDAL {
	return &AwsAccountsDAL{
		firestoreClient:  fs,
		documentsHandler: doitFirestore.DocumentHandler{},
	}
}

func (d *AwsAccountsDAL) CreateAwsAccount(ctx context.Context, accountName string, email string) (*string, error) {
	completeSteps := make([]domain.AwsAccountCompletionStep, 0)
	awsAccountCreateData := domain.AwsAccount{
		AccountName:         accountName,
		Email:               email,
		CompleteSteps:       completeSteps,
		GenuineAwsAccountID: nil,
	}

	awsAccountRef, _, err := d.awsAccountsCollection().Add(ctx, awsAccountCreateData)
	if err != nil {
		return nil, err
	}

	return &awsAccountRef.ID, nil
}

func (d *AwsAccountsDAL) CreateAwsAccountCommand(ctx context.Context, accountID string, commandType domain.AwsAccountCommandType) (*string, error) {
	awsAccountCreateData := domain.AwsAccountCommand{
		AccountID:           accountID,
		ErrorMessage:        nil,
		ProcessingStartedAt: nil,
		RetryCount:          0,
		Status:              domain.AwsAccountCommandStatusScheduled,
		Type:                commandType,
	}

	awsAccountCommandRef, _, err := d.awsAccountCommandsCollection().Add(ctx, awsAccountCreateData)
	if err != nil {
		return nil, err
	}

	return &awsAccountCommandRef.ID, nil
}

func (d *AwsAccountsDAL) GetAwsAccountIDByEmail(ctx context.Context, email string) (*string, error) {
	docsSnaps, err := d.awsAccountsCollection().Where("email", "==", email).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	if len(docsSnaps) == 0 {
		return nil, nil
	}

	firstSnap := docsSnaps[0]

	return &firstSnap.Ref.ID, nil
}

func (d *AwsAccountsDAL) GetAwsAccountCommandByAccountID(ctx context.Context, accountID string) (*domain.AwsAccountCommand, *firestore.DocumentRef, error) {
	docsSnaps, err := d.awsAccountCommandsCollection().Where("accountId", "==", accountID).Documents(ctx).GetAll()
	if err != nil {
		return nil, nil, err
	}

	if len(docsSnaps) == 0 {
		return nil, nil, nil
	}

	firstSnap := docsSnaps[0]

	var awsAccountCommand domain.AwsAccountCommand
	if err := firstSnap.DataTo(&awsAccountCommand); err != nil {
		return nil, nil, err
	}

	return &awsAccountCommand, firstSnap.Ref, nil
}

func (d *AwsAccountsDAL) appCollection() *firestore.CollectionRef {
	return d.firestoreClient.Collection(appCollection)
}

func (d *AwsAccountsDAL) autoGeneratedAwsAccountsDocument() *firestore.DocumentRef {
	return d.appCollection().Doc(autoGeneratedAwsAccountsDoc)
}

func (d *AwsAccountsDAL) awsAccountsCollection() *firestore.CollectionRef {
	return d.autoGeneratedAwsAccountsDocument().Collection(awsAccountsCollection)
}

func (d *AwsAccountsDAL) awsAccountCommandsCollection() *firestore.CollectionRef {
	return d.autoGeneratedAwsAccountsDocument().Collection(awsAccountCommandsCollection)
}
