package fs

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	doitFirestoreIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/aws"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Spot0CostsFirestore struct {
	firestoreClient  *firestore.Client
	documentsHandler doitFirestoreIface.DocumentsHandler
}

type Usage struct {
	SpotInstances         InstancesSummary `firestore:"spotInstances,omitempty"`
	OnDemandInstances     InstancesSummary `firestore:"onDemandInstances,omitempty"`
	OnDemandInstancePrice float64          `firestore:"onDemandInstancePrice,omitempty"`
	TotalSavings          float64          `firestore:"totalSavings"`
	TimeModified          time.Time        `firestore:"timeModified,serverTimestamp"`
}

type InstancesSummary struct {
	TotalCost  float64            `firestore:"totalCost"`
	TotalHours float64            `firestore:"totalHours"`
	Instances  []*InstanceSummary `firestore:"instances"`
}

type InstanceSummary struct {
	Cost         float64 `firestore:"actualCost"`
	AmountHours  float64 `firestore:"amountHours"`
	InstanceType string  `firestore:"instanceType"`
	Platform     string  `firestore:"platform"`
	OnDemandCost float64 `firestore:"onDemandCost,omitempty"`
}

type ASGCustomer struct {
	Customer           *firestore.DocumentRef `firestore:"customer"`
	MarketingEmailSent time.Time              `firestore:"marketingEmailSent,serverTimestamp"`
}

type UsageDoc struct {
	DocID        string
	YearMonthKey string
	Usage        Usage
}

type ISpot0CostsFireStore interface {
	UpdateASGsUsage(ctx context.Context, usageDoc UsageDoc) error
	GetCustomerAMs(ctx context.Context, docRef *firestore.DocumentRef) ([]common.AccountManager, error)
	GetCustomerFromPrimaryDomain(ctx context.Context, primaryDomain string) (*firestore.DocumentRef, error)
	CustomerIsUsingSpotScaling(ctx context.Context, customer *firestore.DocumentRef) (bool, error)
	AddASGCustomerToList(ctx context.Context, customer *firestore.DocumentRef) (bool, error)
	DeleteASGCustomerFromList(ctx context.Context, customer *firestore.DocumentRef) error
	GetCustomerTimeCreated(ctx context.Context, customer *firestore.DocumentRef) (time.Time, error)
}

func NewSpot0CostsFirestoreWithClient(fireStoreClient *firestore.Client) *Spot0CostsFirestore {
	return &Spot0CostsFirestore{
		firestoreClient:  fireStoreClient,
		documentsHandler: doitFirestore.DocumentHandler{},
	}
}

// UpdateASGsUsage updates (overrides keys if exists) the usage data for the given ASGs
func (a *Spot0CostsFirestore) UpdateASGsUsage(ctx context.Context, usageDoc UsageDoc) error {
	docRef := a.firestoreClient.Collection("spot0").Doc("spotApp").Collection("asgs").Doc(usageDoc.DocID)

	_, err := a.documentsHandler.Update(ctx, docRef, []firestore.Update{
		{
			Path:  "usage." + usageDoc.YearMonthKey,
			Value: usageDoc.Usage,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// GetCustomerAMs returns all AM/SAM/FSR of the given customer if any exist
func (a *Spot0CostsFirestore) GetCustomerAMs(ctx context.Context, docRef *firestore.DocumentRef) ([]common.AccountManager, error) {
	var customer common.Customer

	docSnap, err := docRef.Get(ctx)
	if err != nil {
		return []common.AccountManager{}, err
	}

	if err := docSnap.DataTo(&customer); err != nil {
		return []common.AccountManager{}, err
	}

	doitManagers, err := common.GetCustomerAccountManagers(ctx, &customer, common.AccountManagerCompanyDoit)
	if err != nil {
		return []common.AccountManager{}, err
	}

	var AMs []common.AccountManager

	for _, am := range doitManagers {
		// only AM/SAM/FSR
		if am.Role != common.AccountManagerRoleFSR && am.Role != common.AccountManagerRoleSAM {
			continue
		}

		AMs = append(AMs, *am)
	}

	return AMs, nil
}

// GetCustomerFromPrimaryDomain returns the customer document reference by the given primary domain
func (a *Spot0CostsFirestore) GetCustomerFromPrimaryDomain(ctx context.Context, primaryDomain string) (*firestore.DocumentRef, error) {
	docsIter := a.firestoreClient.Collection("customers").Where("primaryDomain", "==", primaryDomain).Documents(ctx)

	docSnap, err := docsIter.Next()
	if err != nil {
		return nil, err
	}

	return docSnap.Ref, nil
}

// CustomerIsUsingSpotScaling returns true if the customer has already linked their account to Spot-scaling
func (a *Spot0CostsFirestore) CustomerIsUsingSpotScaling(ctx context.Context, customer *firestore.DocumentRef) (bool, error) {
	docIter := customer.Collection("cloudConnect").
		Where("cloudPlatform", "==", common.Assets.AmazonWebServices).
		Where("supportedFeatures", "array-contains", aws.SupportedFeature{Name: "spot-scaling", HasRequiredPermissions: true}).
		Documents(ctx)

	_, err := docIter.Next()
	if err == iterator.Done {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

// AddASGCustomerToList adds the given customer to the ASG customers list, if not already exists
func (a *Spot0CostsFirestore) AddASGCustomerToList(ctx context.Context, customer *firestore.DocumentRef) (bool, error) {
	docRef := a.firestoreClient.Collection("spot0").Doc("spotApp").Collection("asgCustomers").Doc(customer.ID)

	_, err := docRef.Create(ctx, ASGCustomer{Customer: customer})
	if status.Code(err) == codes.AlreadyExists {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (a *Spot0CostsFirestore) DeleteASGCustomerFromList(ctx context.Context, customer *firestore.DocumentRef) error {
	docRef := a.firestoreClient.Collection("spot0").Doc("spotApp").Collection("asgCustomers").Doc(customer.ID)

	_, err := docRef.Delete(ctx)
	if err != nil {
		return err
	}

	return nil
}

// GetCustomerTimeCreated returns the time the customer was created
// If the timeCreated field does not exists it returns the default time.Time{}: 0001-01-01T00:00:00Z
func (a *Spot0CostsFirestore) GetCustomerTimeCreated(ctx context.Context, customer *firestore.DocumentRef) (time.Time, error) {
	docSnap, err := customer.Get(ctx)
	if err != nil {
		return time.Time{}, err
	}

	timeCreated, err := docSnap.DataAt("timeCreated")
	if err != nil {
		return time.Time{}, err
	}

	return timeCreated.(time.Time), nil
}
