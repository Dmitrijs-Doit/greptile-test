package dal

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	SharedTenancy    = "shared"
	DedicatedTenancy = "dedicated"

	appCollection                 = "app"
	masterPayerDoc                = "master-payer-accounts"
	masterPayerAccountsCollection = "mpaAccounts"

	assetsCollection = "assets"
)

var (
	ErrorNotFound = fmt.Errorf("document not found")
)

//go:generate mockery --name MasterPayerAccounts
type MasterPayerAccounts interface {
	GetCustomerActiveDedicatedPayers(ctx context.Context, customerID string) ([]domain.MasterPayerAccount, error)
	GetMasterPayerAccount(ctx context.Context, accountNumber string) (*domain.MasterPayerAccount, error)
	GetMasterPayerAccounts(ctx context.Context, fs *firestore.Client) (*domain.MasterPayerAccounts, error)
	GetMasterPayerAccountByAccountNumber(ctx context.Context, accountNumber string) (*domain.MasterPayerAccount, error)
	GetMasterPayerAccountsForDomain(ctx context.Context, customerDomain string) ([]*domain.MasterPayerAccount, error)
	GetMPAWithoutRootAccess(ctx context.Context) (map[string]bool, error)
	UpdateMPAField(ctx context.Context, accountNumber string, updates []firestore.Update) error
	GetActiveAndRetiredPlesMpa(ctx context.Context) (map[string]*domain.MasterPayerAccount, error)
	RetireMPAandDeleteAssets(ctx context.Context, logger logger.ILogger, payerID string) error
}

type MasterPayerAccountDAL struct {
	firestoreClient  *firestore.Client
	documentsHandler iface.DocumentsHandler
}

func NewMasterPayerAccountDAL(ctx context.Context, projectID string) (*MasterPayerAccountDAL, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewMasterPayerAccountDALWithClient(fs), nil
}

func NewMasterPayerAccountDALWithClient(fs *firestore.Client) *MasterPayerAccountDAL {
	return &MasterPayerAccountDAL{
		firestoreClient:  fs,
		documentsHandler: doitFirestore.DocumentHandler{},
	}
}

func (d *MasterPayerAccountDAL) GetCustomerActiveDedicatedPayers(ctx context.Context, customerID string) ([]domain.MasterPayerAccount, error) {
	var mpas []domain.MasterPayerAccount

	docsSnaps, err := d.masterPayerAccountsCollection().Where("customerId", "==", customerID).Where("status", "==", "active").Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, snap := range docsSnaps {
		var mpa domain.MasterPayerAccount
		if err := snap.DataTo(&mpa); err != nil {
			return nil, err
		}

		if mpa.TenancyType == DedicatedTenancy && mpa.FlexSaveRecalculationStartDate != nil && !mpa.FlexSaveRecalculationStartDate.IsZero() {
			mpas = append(mpas, mpa)
		}
	}

	return mpas, nil
}

// GetMasterPayerAccounts fetches the master payer accounts from firestore
func GetMasterPayerAccounts(ctx context.Context, fs *firestore.Client) (*domain.MasterPayerAccounts, error) {
	var payerAccounts domain.MasterPayerAccounts

	accounts := make(map[string]*domain.MasterPayerAccount)

	query := fs.Collection(appCollection).Doc(masterPayerDoc).Collection(masterPayerAccountsCollection)

	docsSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, docsSnap := range docsSnaps {
		var mpa domain.MasterPayerAccount
		if err := docsSnap.DataTo(&mpa); err != nil {
			return nil, err
		}

		accounts[mpa.AccountNumber] = &mpa
	}

	payerAccounts.Accounts = accounts

	return &payerAccounts, nil
}

func GetMasterPayerAccountsByPayerIDs(ctx context.Context, fs *firestore.Client, payerAccountIDs ...string) (*domain.MasterPayerAccounts, error) {
	var payerAccounts domain.MasterPayerAccounts

	accounts := make(map[string]*domain.MasterPayerAccount)

	query := fs.Collection(appCollection).Doc(masterPayerDoc).Collection(masterPayerAccountsCollection).Where("accountNumber", "in", payerAccountIDs)

	docsSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, docsSnap := range docsSnaps {
		var mpa domain.MasterPayerAccount
		if err := docsSnap.DataTo(&mpa); err != nil {
			return nil, err
		}

		accounts[mpa.AccountNumber] = &mpa
	}

	payerAccounts.Accounts = accounts

	return &payerAccounts, nil
}

// GetMasterPayerAccounts fetches the master payer accounts from firestore
func (d *MasterPayerAccountDAL) GetMasterPayerAccounts(ctx context.Context, fs *firestore.Client) (*domain.MasterPayerAccounts, error) {
	return GetMasterPayerAccounts(ctx, fs)
}

// GetMasterPayerAccountsForDomain fetches master payer accounts for a given domain
func (d *MasterPayerAccountDAL) GetMasterPayerAccountsForDomain(ctx context.Context, customerDomain string) ([]*domain.MasterPayerAccount, error) {
	docRefs := d.masterPayerAccountsCollection().Where("domain", "==", customerDomain).Documents(ctx)

	docSnaps, err := d.documentsHandler.GetAll(docRefs)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrorNotFound
		}

		return nil, err
	}

	var masterPayerAccountsForDomain []*domain.MasterPayerAccount

	for _, docSnap := range docSnaps {
		var mpa domain.MasterPayerAccount
		if err := docSnap.DataTo(&mpa); err != nil {
			return nil, err
		}

		masterPayerAccountsForDomain = append(masterPayerAccountsForDomain, &mpa)
	}

	return masterPayerAccountsForDomain, nil
}

func (d *MasterPayerAccountDAL) GetMasterPayerAccountByAccountNumber(ctx context.Context, accountNumber string) (*domain.MasterPayerAccount, error) {
	docRefs := d.masterPayerAccountsCollection().
		Where("accountNumber", "==", accountNumber).
		Where("status", "in", []string{string(domain.MasterPayerAccountStatusPending), string(domain.MasterPayerAccountStatusActive)}).
		Documents(ctx)

	docSnaps, err := d.documentsHandler.GetAll(docRefs)
	if err != nil {
		return nil, err
	}

	if len(docSnaps) == 0 {
		return nil, ErrorNotFound
	}

	if len(docSnaps) > 1 {
		return nil, fmt.Errorf("%w: multiple master payer accounts exist for account number %s", ErrorNotFound, accountNumber)
	}

	var mpa domain.MasterPayerAccount
	if err := docSnaps[0].DataTo(&mpa); err != nil {
		return nil, err
	}

	return &mpa, nil
}

// GetMasterPayerAccount fetches an active master payer account from firestore by accountNumber
func (d *MasterPayerAccountDAL) GetMasterPayerAccount(ctx context.Context, accountNumber string) (*domain.MasterPayerAccount, error) {
	docRefs := d.masterPayerAccountsCollection().Where("accountNumber", "==", accountNumber).Where("status", "==", "active").Limit(1).Documents(ctx)

	docSnaps, err := d.documentsHandler.GetAll(docRefs)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrorNotFound
		}

		return nil, err
	}

	if len(docSnaps) == 0 {
		return nil, ErrorNotFound
	}

	var mpa domain.MasterPayerAccount
	if err := docSnaps[0].DataTo(&mpa); err != nil {
		return nil, err
	}

	return &mpa, nil
}

// ToMasterPayerAccount returns the account as the new MasterPayerAccount type
func ToMasterPayerAccount(ctx context.Context, p *domain.PayerAccount, fs *firestore.Client) (*domain.MasterPayerAccount, error) {
	masterPayerAccounts, err := GetMasterPayerAccounts(ctx, fs)
	if err != nil {
		return nil, err
	}

	masterPayerAccount, ok := masterPayerAccounts.Accounts[p.AccountID]
	if !ok {
		return nil, errors.New("master payer account not found")
	}

	return masterPayerAccount, nil
}

func HasFlexsaveEligibleDedicatedPayerAccount(ctx context.Context, fs *firestore.Client, log logger.ILogger, customerID string) (bool, error) {
	docsSnaps, err := fs.Collection(appCollection).Doc(masterPayerDoc).Collection(masterPayerAccountsCollection).Where("customerId", "==", customerID).Documents(ctx).GetAll()
	if err != nil {
		return false, err
	}

	for _, snap := range docsSnaps {
		var mpa *domain.MasterPayerAccount
		if err := snap.DataTo(&mpa); err != nil {
			log.Error(err)
			continue
		}

		if mpa.IsDedicatedPayer() {
			if mpa.FlexSaveRecalculationStartDate == nil || mpa.FlexSaveRecalculationStartDate.IsZero() || mpa.FlexSaveAllowed {
				log.Errorf("mpa %s is dedicated payer, but is in incorrect state. Please investigate", mpa.AccountNumber)
			}

			return true, nil
		}
	}

	return false, nil
}

// GetMPAWithoutRootAccess returns a map of customers-account -> no-root-access
func (d *MasterPayerAccountDAL) GetMPAWithoutRootAccess(ctx context.Context) (map[string]bool, error) {
	docsRefs := d.firestoreClient.Collection(appCollection).Doc(masterPayerDoc).Collection(masterPayerAccountsCollection).
		Where("status", "==", "active").
		Where("features.no-root-access", "==", true).
		Documents(ctx)

	docsSnaps, err := d.documentsHandler.GetAll(docsRefs)
	if err != nil {
		return nil, err
	}

	var mpaNoRootAccess = make(map[string]bool)

	for _, snap := range docsSnaps {
		var mpa domain.MasterPayerAccount
		if err := snap.DataTo(&mpa); err != nil {
			return nil, err
		}

		mpaNoRootAccess[mpa.AccountNumber] = true
	}

	return mpaNoRootAccess, nil
}

func (d *MasterPayerAccountDAL) UpdateMPAField(ctx context.Context, accountNumber string, updates []firestore.Update) error {
	docRefs := d.masterPayerAccountsCollection().Where("accountNumber", "==", accountNumber).Limit(1).Documents(ctx)

	for {
		docSnap, err := docRefs.Next()
		if err != nil {
			break
		}

		if _, err := d.documentsHandler.Update(ctx, docSnap.Ref, updates); err != nil {
			return err
		}
	}

	return nil
}

func (d *MasterPayerAccountDAL) appCollection() *firestore.CollectionRef {
	return d.firestoreClient.Collection(appCollection)
}

func (d *MasterPayerAccountDAL) masterPayerDocument() *firestore.DocumentRef {
	return d.appCollection().Doc(masterPayerDoc)
}

func (d *MasterPayerAccountDAL) masterPayerAccountsCollection() *firestore.CollectionRef {
	return d.masterPayerDocument().Collection(masterPayerAccountsCollection)
}

func (d *MasterPayerAccountDAL) GetActiveAndRetiredPlesMpa(ctx context.Context) (map[string]*domain.MasterPayerAccount, error) {
	accounts := make(map[string]*domain.MasterPayerAccount)

	query := d.masterPayerAccountsCollection().
		Where("support.model", "==", "partner-led").
		Where("support.tier", "==", "enterprise").
		Where("status", "in", []string{"active", "retired"})

	docsSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, docsSnap := range docsSnaps {
		var mpa domain.MasterPayerAccount
		if err := docsSnap.DataTo(&mpa); err != nil {
			return nil, err
		}

		accounts[mpa.AccountNumber] = &mpa
	}

	return accounts, nil
}

// RetireMPA retires the master payer account and deletes all assets associated with it
func (d *MasterPayerAccountDAL) RetireMPAandDeleteAssets(ctx context.Context, l logger.ILogger, payerID string) error {
	bulkWriterJobs := make([]*firestore.BulkWriterJob, 0)
	writer := d.firestoreClient.BulkWriter(ctx)

	job, err := d.retireMPA(ctx, l, writer, payerID)
	if err != nil {
		return err
	}
	jobs, err := d.deleteAssetsFromMPA(ctx, l, writer, payerID)
	if err != nil {
		return err
	}

	// Commit the bulk writer
	writer.Flush()

	// append all jobs to the bulkWriterJobs
	bulkWriterJobs = append(bulkWriterJobs, job)
	bulkWriterJobs = append(bulkWriterJobs, jobs...)

	// Check if any of the jobs failed
	for _, job := range bulkWriterJobs {
		if _, err := job.Results(); err != nil {
			return err
		}
	}

	return nil
}

func (d *MasterPayerAccountDAL) retireMPA(ctx context.Context, l logger.ILogger, writer *firestore.BulkWriter, payerID string) (*firestore.BulkWriterJob, error) {
	mpaIter := d.masterPayerAccountsCollection().Where("accountNumber", "==", payerID).Documents(ctx)
	docSnap, err := mpaIter.Next()
	if err != nil {
		return nil, err
	}

	jobs, err := writer.Update(docSnap.Ref, []firestore.Update{
		{Path: "status", Value: "retired"},
		{Path: "retireDate", Value: firestore.ServerTimestamp},
	})
	if err != nil {
		return nil, err
	}
	l.Infof("MPA retired: %v", payerID)

	return jobs, nil
}

func (d *MasterPayerAccountDAL) deleteAssetsFromMPA(ctx context.Context, l logger.ILogger, writer *firestore.BulkWriter, payerID string) ([]*firestore.BulkWriterJob, error) {
	var assetsDeleted []string
	var bulkWriterJobs []*firestore.BulkWriterJob

	assetsIter := d.firestoreClient.Collection(assetsCollection).Where("properties.organization.payerAccount.id", "==", payerID).Documents(ctx)
	assetSnaps, err := assetsIter.GetAll()
	if err != nil {
		return nil, err
	}

	for _, assetSnap := range assetSnaps {
		job, err := writer.Delete(assetSnap.Ref)
		if err != nil {
			return nil, err
		}
		assetsDeleted = append(assetsDeleted, assetSnap.Ref.ID)
		bulkWriterJobs = append(bulkWriterJobs, job)
	}
	l.Infof("Retire MPA == Deleted assets: %v\n", assetsDeleted)

	return bulkWriterJobs, nil
}
