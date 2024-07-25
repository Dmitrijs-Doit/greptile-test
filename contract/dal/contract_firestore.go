package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/doitintl/errors"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/contract/domain"
	"github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	contractsCollection = "contracts"
	activeFlagField     = "active"
	customerField       = "customer"
	typeField           = "type"
	isCommitmentField   = "isCommitment"
	endDateField        = "endDate"
	lastUpdatedField    = "lastUpdateDate"
	paymentTermField    = "paymentTerm"
	finalField          = "final"
)

// ContractFirestore is used to interact with contracts stored on Firestore.
type ContractFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewContractFirestore returns a new ContractFirestore instance with given project id.
func NewContractFirestore(ctx context.Context, projectID string) (*ContractFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewContractFirestoreWithClient(
		func(_ context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewContractFirestoreWithClient returns a new ContractFirestore using given client.
func NewContractFirestoreWithClient(fun connection.FirestoreFromContextFun) *ContractFirestore {
	return &ContractFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *ContractFirestore) ConvertSnapshotToContract(ctx context.Context, doc *firestore.DocumentSnapshot) (pkg.Contract, error) {
	// Convert document to Contract struct
	var contract pkg.Contract
	if err := doc.DataTo(&contract); err != nil {
		return pkg.Contract{}, err
	}

	return contract, nil
}

// GetContractsByType returns customer contracts of the given types.
func (d *ContractFirestore) GetContractsByType(ctx context.Context, customerRef *firestore.DocumentRef, contractTypes ...domain.ContractType) ([]common.Contract, error) {
	if customerRef == nil {
		return nil, ErrUndefinedCustomerRef
	}

	if len(contractTypes) == 0 {
		return nil, ErrUndefinedContractType
	}

	for _, t := range contractTypes {
		if t == "" {
			return nil, ErrUndefinedContractType
		}
	}

	contractsDocSnaps, err := d.firestoreClientFun(ctx).Collection(contractsCollection).
		Where(customerField, "==", customerRef).
		Where(typeField, "in", contractTypes).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, err
	}

	var contracts []common.Contract

	for _, contractSnap := range contractsDocSnaps {
		var contract common.Contract
		if err := contractSnap.DataTo(&contract); err != nil {
			return nil, err
		}

		contracts = append(contracts, contract)
	}

	return contracts, nil
}

// ListCustomerContractsByType returns customer contracts of a given type
func (d *ContractFirestore) ListCustomerNext10Contracts(ctx context.Context, customerRef *firestore.DocumentRef) ([]pkg.Contract, error) {
	if customerRef == nil {
		return nil, ErrUndefinedCustomerRef
	}

	contractsDocSnaps, err := d.firestoreClientFun(ctx).Collection(contractsCollection).
		Where(customerField, "==", customerRef).
		Where(typeField, "in", []string{string(pkg.NavigatorPackageTierType), string(pkg.SolvePackageTierType)}).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, err
	}

	customerContracts := []pkg.Contract{}

	for _, contractSnap := range contractsDocSnaps {
		var contract pkg.Contract
		if err := contractSnap.DataTo(&contract); err != nil {
			return nil, err
		}

		contract.ID = contractSnap.Ref.ID

		customerContracts = append(customerContracts, contract)
	}

	return customerContracts, nil
}

// ListCustomerContractsByType returns customer contracts of a given type
func (d *ContractFirestore) ListNext10Contracts(ctx context.Context) ([]pkg.Contract, error) {
	contractsDocSnaps, err := d.firestoreClientFun(ctx).Collection(contractsCollection).
		Where(typeField, "in", []string{string(pkg.NavigatorPackageTierType), string(pkg.SolvePackageTierType)}).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, err
	}

	contracts := []pkg.Contract{}

	for _, contractSnap := range contractsDocSnaps {
		var contract pkg.Contract
		if err := contractSnap.DataTo(&contract); err != nil {
			return nil, err
		}

		contract.ID = contractSnap.Ref.ID

		contracts = append(contracts, contract)
	}

	return contracts, nil
}

func (d *ContractFirestore) ListContracts(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	limit int,
) ([]common.Contract, error) {
	if customerRef == nil {
		return nil, ErrUndefinedCustomerRef
	}

	query := d.
		firestoreClientFun(ctx).
		Collection(contractsCollection).
		Where(customerField, "==", customerRef)
	if limit > 0 {
		query = query.Limit(limit)
	}

	contractsDocSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var contracts []common.Contract

	for _, contractSnap := range contractsDocSnaps {
		var contract common.Contract
		if err := contractSnap.DataTo(&contract); err != nil {
			return nil, err
		}

		contracts = append(contracts, contract)
	}

	return contracts, nil
}

func (d *ContractFirestore) GetCustomerContractByID(ctx context.Context, customerID, contractID string) (*pkg.Contract, error) {
	contractSnap, err := d.firestoreClientFun(ctx).
		Collection(contractsCollection).Doc(contractID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var contract pkg.Contract
	if err := contractSnap.DataTo(&contract); err != nil {
		return nil, err
	}

	if contract.Customer.ID != customerID {
		return nil, errors.Errorf("contract %s for customer %s not found", contractID, customerID)
	}

	contract.ID = contractID

	return &contract, nil
}

func (d *ContractFirestore) GetContractByID(ctx context.Context, contractID string) (*pkg.Contract, error) {
	contractSnap, err := d.firestoreClientFun(ctx).
		Collection(contractsCollection).Doc(contractID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var contract pkg.Contract
	if err := contractSnap.DataTo(&contract); err != nil {
		return nil, err
	}

	contract.ID = contractSnap.Ref.ID

	return &contract, nil
}

func (d *ContractFirestore) GetActiveContracts(ctx context.Context) ([]*firestore.DocumentSnapshot, error) {
	now := time.Now()

	contractSnaps, err := d.firestoreClientFun(ctx).Collection(contractsCollection).
		Where(isCommitmentField, "==", true).
		Where(activeFlagField, "==", true).
		Where(endDateField, ">=", now).
		Where(typeField, "in", []string{"amazon-web-services", "google-cloud"}).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	return contractSnaps, nil
}

func (d *ContractFirestore) GetActiveContractsForCustomer(ctx context.Context, customerID string) ([]*firestore.DocumentSnapshot, error) {
	customerRef := d.firestoreClientFun(ctx).Doc("customers/" + customerID)

	contractSnaps, err := d.firestoreClientFun(ctx).Collection(contractsCollection).
		Where(activeFlagField, "==", true).
		Where(customerField, "==", customerRef).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	return contractSnaps, nil
}

func (d *ContractFirestore) GetRef(ctx context.Context, contractID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(contractsCollection).Doc(contractID)
}

func (d *ContractFirestore) SetActiveFlag(ctx context.Context, contractID string, value bool) error {
	contractRef := d.GetRef(ctx, contractID)

	_, err := contractRef.Update(ctx, []firestore.Update{{FieldPath: []string{activeFlagField}, Value: value}})

	return err
}

func (d *ContractFirestore) CreateContract(ctx context.Context, req pkg.Contract) error {
	contractRef := d.firestoreClientFun(ctx).Collection(contractsCollection)

	_, _, err := contractRef.Add(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

func (d *ContractFirestore) CancelContract(ctx context.Context, contractID string) error {
	contractRef := d.GetRef(ctx, contractID)
	now := time.Now()

	_, err := d.documentsHandler.Update(ctx, contractRef, []firestore.Update{
		{
			Path: endDateField, Value: now,
		},
		{
			Path: "timestamp", Value: now,
		},
	})

	return err
}

// Get contract by payment term
func (d *ContractFirestore) GetNavigatorAndSolveContracts(ctx context.Context) ([]pkg.Contract, error) {
	contractsDocSnaps, err := d.firestoreClientFun(ctx).Collection(contractsCollection).
		Where(paymentTermField, "in", []string{string(domain.PaymentTermMonthly), string(domain.PaymentTermAnnual)}).
		Where(typeField, "in", []string{string(domain.ContractTypeNavigator), string(domain.ContractTypeSolve), string(domain.ContractTypeSolveAccelerator)}).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, err
	}

	contracts := []pkg.Contract{}

	for _, contractSnap := range contractsDocSnaps {
		var contract pkg.Contract
		if err := contractSnap.DataTo(&contract); err != nil {
			return nil, err
		}

		contract.ID = contractSnap.Ref.ID

		contracts = append(contracts, contract)
	}

	return contracts, nil
}

func (d *ContractFirestore) WriteBillingDataInContracts(ctx context.Context, contractBillingAggData domain.ContractBillingAggregatedData, billingMonth string, contractID string, lastUpdated string, final bool) error {
	docRef := d.firestoreClientFun(ctx).Collection(contractsCollection).Doc(contractID).Collection("billingData").Doc(billingMonth)

	_, err := docRef.Get(ctx)
	if err != nil {
		newData := map[string]interface{}{
			time.Now().Format("2006-01-02"): contractBillingAggData,
			lastUpdatedField:                lastUpdated,
			finalField:                      final,
		}

		_, err := docRef.Set(ctx, newData)
		if err != nil {
			return err
		}
	} else {
		_, err := docRef.Set(ctx, map[string]interface{}{
			lastUpdatedField: lastUpdated,
			finalField:       final,
			time.Now().Format("2006-01-02"): map[string]interface{}{
				"baseFee":     contractBillingAggData.BaseFee,
				"consumption": contractBillingAggData.Consumption,
			},
		}, firestore.MergeAll)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *ContractFirestore) UpdateContract(ctx context.Context, contractID string, contractUpdates []firestore.Update) error {
	docRef := d.firestoreClientFun(ctx).Collection(contractsCollection).Doc(contractID)

	_, err := docRef.Update(ctx, contractUpdates)
	if err != nil {
		return err
	}

	return nil
}

func (d *ContractFirestore) GetBillingDataOfContract(ctx context.Context, doc *firestore.DocumentSnapshot) (billingData map[string]map[string]interface{}, err error) {
	billingDataColRef := doc.Ref.Collection(domain.BillingDataField)

	billingData = make(map[string]map[string]interface{})

	iter := billingDataColRef.Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		var billingDataSingleMonth map[string]interface{}

		if err := doc.DataTo(&billingDataSingleMonth); err != nil {
			return nil, err
		}

		if billingDataSingleMonth != nil {
			billingData[doc.Ref.ID] = billingDataSingleMonth
		}
	}

	return billingData, nil
}

func (d *ContractFirestore) ListCustomersWithNext10Contracts(ctx context.Context) ([]*firestore.DocumentRef, error) {
	contractsDocSnaps, err := d.firestoreClientFun(ctx).Collection(contractsCollection).
		Where(typeField, "in", []string{string(pkg.NavigatorPackageTierType), string(pkg.SolvePackageTierType)}).
		Select(customerField).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, err
	}

	customerRefs := make([]*firestore.DocumentRef, 0, len(contractsDocSnaps))

	for _, contractSnap := range contractsDocSnaps {
		refContainer := struct {
			Ref *firestore.DocumentRef `firestore:"customer"`
		}{}

		if err := contractSnap.DataTo(&refContainer); err != nil {
			return nil, err
		}

		customerRefs = append(customerRefs, refContainer.Ref)
	}

	seen := make(map[string]struct{}, len(customerRefs))
	res := make([]*firestore.DocumentRef, 0, len(customerRefs))

	for _, ref := range customerRefs {
		if _, found := seen[ref.ID]; found {
			continue
		}

		seen[ref.ID] = struct{}{}

		res = append(res, ref)
	}

	return res, nil
}
func (d *ContractFirestore) GetActiveGoogleCloudContracts(ctx context.Context) ([]*firestore.DocumentSnapshot, error) {
	now := time.Now()

	commitmentContractsQuery := d.firestoreClientFun(ctx).Collection(contractsCollection).
		Where(isCommitmentField, "==", true).
		Where(activeFlagField, "==", true).
		Where(endDateField, ">=", now).
		Where(typeField, "==", domain.ContractTypeGoogleCloud)

	onDemandContractsQuery := d.firestoreClientFun(ctx).Collection(contractsCollection).
		Where(isCommitmentField, "==", false).
		Where(activeFlagField, "==", true).
		Where(typeField, "==", domain.ContractTypeGoogleCloud)

	contractSnaps, err := firebase.ExecuteQueries(ctx, []firestore.Query{commitmentContractsQuery, onDemandContractsQuery})

	return contractSnaps, err
}

func (d *ContractFirestore) UpdateContractSupport(ctx context.Context, inputs []domain.UpdateSupportInput) error {
	bulkWriter := d.firestoreClientFun(ctx).BulkWriter(ctx)

	for _, input := range inputs {
		if _, err := bulkWriter.Update(input.Ref, []firestore.Update{{
			Path:  "properties.gcpSupport",
			Value: input.Support,
		}}); err != nil {
			return err
		}
	}

	bulkWriter.End()

	return nil
}

func (d *ContractFirestore) DeleteContract(ctx context.Context, contractID string) error {
	contractRef := d.GetRef(ctx, contractID)

	return doitFirestore.DeleteDocumentAndSubcollections(ctx, d.firestoreClientFun(ctx), contractRef)
}
