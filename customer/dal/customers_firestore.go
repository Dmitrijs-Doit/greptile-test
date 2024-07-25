package dal

import (
	"context"
	"errors"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/pkg"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/customer/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
)

const (
	customersCollection            = "customers"
	accountConfigurationCollection = "accountConfiguration"
	integrationsCollection         = "integrations"
	accountManagersCollection      = "accountManagers"
)

// CustomersFirestore is used to interact with customers stored on Firestore.
type CustomersFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewCustomersFirestore returns a new CustomersFirestore instance with given project id.
func NewCustomersFirestore(ctx context.Context, projectID string) (*CustomersFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewCustomersFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewCustomersFirestoreWithClient returns a new CustomersFirestore using given client.
func NewCustomersFirestoreWithClient(fun connection.FirestoreFromContextFun) *CustomersFirestore {
	return &CustomersFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *CustomersFirestore) GetRef(ctx context.Context, ID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(customersCollection).Doc(ID)
}

// GetCustomer returns customer's data.
func (d *CustomersFirestore) GetCustomer(ctx context.Context, customerID string) (*common.Customer, error) {
	if customerID == "" {
		return nil, errors.New("invalid customer id")
	}

	doc := d.GetRef(ctx, customerID)

	snap, err := d.documentsHandler.Get(ctx, doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var customer common.Customer

	if err := snap.DataTo(&customer); err != nil {
		return nil, err
	}

	customer.Snapshot = snap.Snapshot()

	customer.ID = snap.ID()

	return &customer, nil
}

func (d *CustomersFirestore) GetCustomersByIDs(ctx context.Context, ids []string) ([]*common.Customer, error) {
	collection := d.firestoreClientFun(ctx).Collection(customersCollection)
	docRefs := make([]*firestore.DocumentRef, 0, len(ids))

	for _, id := range ids {
		docRefs = append(docRefs, collection.Doc(id))
	}

	fs := d.firestoreClientFun(ctx)

	docSnaps, err := fs.GetAll(ctx, docRefs)
	if err != nil {
		return nil, err
	}

	var customers []*common.Customer

	for _, docSnap := range docSnaps {
		var customer common.Customer

		if !docSnap.Exists() {
			continue
		}

		if err := docSnap.DataTo(&customer); err != nil {
			return nil, err
		}

		customer.Snapshot = docSnap

		customer.ID = docSnap.Ref.ID

		customers = append(customers, &customer)
	}

	return customers, nil
}

func (d *CustomersFirestore) GetCustomers(
	ctx context.Context,
) ([]*firestore.DocumentSnapshot, error) {
	fs := d.firestoreClientFun(ctx)

	docSnaps, err := fs.Collection(customersCollection).
		Select().Documents(ctx).GetAll()

	if err != nil {
		return nil, err
	}

	return docSnaps, nil
}

func (d *CustomersFirestore) GetPresentationCustomers(
	ctx context.Context,
) ([]*common.Customer, error) {
	fs := d.firestoreClientFun(ctx)

	docSnaps, err := fs.Collection(customersCollection).Where(
		"presentationMode.isPredefined", "==", true,
	).Documents(ctx).GetAll()

	if err != nil {
		return nil, err
	}

	var customers []*common.Customer

	for _, docSnap := range docSnaps {
		var customer common.Customer

		if err := docSnap.DataTo(&customer); err != nil {
			return nil, err
		}

		customer.Snapshot = docSnap
		customer.ID = docSnap.Ref.ID

		customers = append(customers, &customer)
	}

	return customers, nil
}

func (d *CustomersFirestore) GetPresentationCustomersWithAssetType(
	ctx context.Context,
	assetType string,
) ([]*firestore.DocumentSnapshot, error) {
	fs := d.firestoreClientFun(ctx)

	docSnaps, err := fs.Collection(customersCollection).
		Where("presentationMode.isPredefined", "==", true).
		Where("assets", common.ArrayContains, assetType).
		Documents(ctx).GetAll()

	if err != nil {
		return nil, err
	}

	// TODO: return *common.Customer instead of *firestore.DocumentSnapshot
	return docSnaps, nil
}

func (d *CustomersFirestore) GetAWSCustomers(
	ctx context.Context,
) ([]*firestore.DocumentSnapshot, error) {
	fs := d.firestoreClientFun(ctx)

	docSnaps, err := fs.Collection(customersCollection).
		Where("assets", common.ArrayContains, common.Assets.AmazonWebServices).
		Select().Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	return docSnaps, nil
}

func (d *CustomersFirestore) GetMSAzureCustomers(
	ctx context.Context,
) ([]*firestore.DocumentSnapshot, error) {
	fs := d.firestoreClientFun(ctx)

	docSnaps, err := fs.Collection(customersCollection).
		Where("assets", common.ArrayContains, common.Assets.MicrosoftAzure).
		Select().Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	return docSnaps, nil
}

func (d *CustomersFirestore) GetCloudhealthCustomers(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
) ([]*firestore.DocumentSnapshot, error) {
	fs := d.firestoreClientFun(ctx)

	chtCustomersCollection := fs.Collection(integrationsCollection).
		Doc("cloudhealth").
		Collection("cloudhealthCustomers")

	chtDocSnaps, err := chtCustomersCollection.
		Where("customer", "==", customerRef).
		Where("disabled", "==", false).
		Select().Documents(ctx).GetAll()

	return chtDocSnaps, err
}

func (d *CustomersFirestore) GetAllCustomerRefs(ctx context.Context) ([]*firestore.DocumentRef, error) {
	refs, err := d.firestoreClientFun(ctx).Collection(customersCollection).Select().Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var ret []*firestore.DocumentRef

	for _, ref := range refs {
		ret = append(ret, ref.Ref)
	}

	return ret, nil
}

func (d *CustomersFirestore) GetAllCustomerIDs(ctx context.Context) ([]string, error) {
	refs, err := d.GetAllCustomerRefs(ctx)
	if err != nil {
		return nil, err
	}

	var IDs []string

	for _, ref := range refs {
		IDs = append(IDs, ref.ID)
	}

	return IDs, nil
}

func (d *CustomersFirestore) GetCustomerOrgs(ctx context.Context, customerID string, orgID string) ([]*common.Organization, error) {
	if customerID == "" {
		return nil, errors.New("invalid empty customer id")
	}

	customerRef := d.GetRef(ctx, customerID)
	customerOrgsCollection := customerRef.Collection("customerOrgs")

	// Retrieve a specific organization scenario
	if orgID != "" {
		var orgDocRef *firestore.DocumentRef

		if organizations.IsPresetOrg(orgID) {
			orgDocRef = d.firestoreClientFun(ctx).Collection("organizations").Doc(orgID)
		} else {
			orgDocRef = customerOrgsCollection.Doc(orgID)
		}

		docSnap, err := orgDocRef.Get(ctx)
		if err != nil {
			return nil, err
		}

		var org common.Organization
		if err := docSnap.DataTo(&org); err != nil {
			return nil, err
		}

		org.Snapshot = docSnap
		org.ID = docSnap.Ref.ID

		return []*common.Organization{&org}, nil
	}

	// Retrieve all customer organizations scenario
	orgSnaps, err := customerOrgsCollection.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	presetOrgSnaps, err := d.firestoreClientFun(ctx).Collection("organizations").Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	isCSP := customerID == domainQuery.CSPCustomerID

	for _, orgSnap := range presetOrgSnaps {
		// CSP customer should not use AWS/GCP preset orgs (partner orgs)
		if isCSP &&
			(orgSnap.Ref.ID == organizations.PresetAWSOrgID || orgSnap.Ref.ID == organizations.PresetGCPOrgID) {
			continue
		}

		orgSnaps = append(orgSnaps, orgSnap)
	}

	var orgs []*common.Organization

	for _, orgSnap := range orgSnaps {
		var org common.Organization
		if err := orgSnap.DataTo(&org); err != nil {
			return nil, err
		}

		org.Snapshot = orgSnap
		org.ID = orgSnap.Ref.ID

		orgs = append(orgs, &org)
	}

	return orgs, nil
}

func (d *CustomersFirestore) GetCustomerAWSAccountConfiguration(ctx context.Context, customerRef *firestore.DocumentRef) (*common.AWSSettings, error) {
	if customerRef == nil {
		return nil, errors.New("customer reference is nil")
	}

	collection := customerRef.Collection(accountConfigurationCollection)

	docSnap, err := collection.Doc(common.Assets.AmazonWebServices).Get(ctx)
	if err != nil {
		// If doc not found it means that the config was never created. Don't throw an error and return the default values
		if status.Code(err) == codes.NotFound {
			return &common.AWSSettings{}, nil
		}

		return nil, err
	}

	var awsSettings *common.AWSSettings
	if err := docSnap.DataTo(&awsSettings); err != nil {
		return nil, err
	}

	return awsSettings, nil
}

func (d *CustomersFirestore) GetPrimaryDomain(ctx context.Context, customerID string) (string, error) {
	customerData, err := d.GetRef(ctx, customerID).Get(ctx)
	if err != nil {
		return "", err
	}

	domain, err := customerData.DataAtPath([]string{"primaryDomain"})
	if err != nil {
		return "", err
	}

	primaryDomain := domain.(string)

	return primaryDomain, nil
}

func (d *CustomersFirestore) UpdateCustomerFieldValue(ctx context.Context, customerID string, fieldPath string, value interface{}) error {
	customerRef := d.GetRef(ctx, customerID)
	_, err := customerRef.Update(ctx, []firestore.Update{
		{FieldPath: []string{fieldPath}, Value: value},
	})

	return err
}

func (d *CustomersFirestore) UpdateCustomerFieldValueDeep(ctx context.Context, customerID string, fieldPath []string, value interface{}) error {
	customerRef := d.GetRef(ctx, customerID)
	_, err := customerRef.Update(ctx, []firestore.Update{
		{
			FieldPath: fieldPath,
			Value:     value,
		},
	})

	return err
}

func (d *CustomersFirestore) DeleteCustomer(ctx context.Context, customerID string) error {
	if customerID == "" {
		return ErrInvalidCustomerID
	}

	docRef := d.GetRef(ctx, customerID)

	bulkWriter := d.firestoreClientFun(ctx).BulkWriter(ctx)

	if err := d.documentsHandler.DeleteDocAndSubCollections(ctx, docRef, bulkWriter); err != nil {
		return err
	}

	bulkWriter.End()

	return nil
}

func (d *CustomersFirestore) GetCustomerAccountTeam(ctx context.Context, customerID string) ([]domain.AccountManagerListItem, error) {
	fs := d.firestoreClientFun(ctx)

	customer, err := d.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	accountTeams := customer.AccountTeam

	var accountTeamList []domain.AccountManagerListItem

	for _, account := range accountTeams {
		accountRef := account.Ref
		accountID := accountRef.ID

		accountManagerSnap, err := fs.Collection(accountManagersCollection).Doc(accountID).Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				log.Printf("accountID %s does not exist", accountID)
				continue
			}

			return nil, err
		}

		company, err := accountManagerSnap.DataAt("company")
		if err != nil {
			return nil, err
		}

		if company == "doit" {
			var accountTeamItem domain.AccountManagerListItem

			if err := accountManagerSnap.DataTo(&accountTeamItem); err != nil {
				return nil, err
			}

			accountTeamItem.ID = accountID

			accountTeamList = append(accountTeamList, accountTeamItem)
		}
	}

	return accountTeamList, nil
}

// GetCustomerOrPresentationModeCustomer returns customers or presentation customers data.
func (d *CustomersFirestore) GetCustomerOrPresentationModeCustomer(ctx context.Context, customerID string) (*common.Customer, error) {
	customer, err := d.GetCustomer(ctx, customerID)

	if err != nil {
		return nil, err
	}

	if customer.PresentationMode != nil && customer.PresentationMode.Enabled && customer.PresentationMode.CustomerID != "" {
		presentationCustomer, err := d.GetCustomer(ctx, customer.PresentationMode.CustomerID)

		if err != nil {
			log.Printf("presentation customer %s does not exist", customer.PresentationMode.CustomerID)
			return customer, nil
		}

		return presentationCustomer, nil
	}

	return customer, nil
}

func (d *CustomersFirestore) GetCustomersByTier(ctx context.Context, tierRef *firestore.DocumentRef, packageType pkg.PackageTierType) ([]*firestore.DocumentSnapshot, error) {
	tierFieldPath := fmt.Sprintf("tiers.%s.tier", string(packageType))

	docSnaps, err := d.firestoreClientFun(ctx).
		Collection(customersCollection).
		Where(tierFieldPath, "==", tierRef).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	return docSnaps, nil
}
