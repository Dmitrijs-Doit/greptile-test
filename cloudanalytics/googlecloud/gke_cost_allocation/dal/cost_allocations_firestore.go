package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/domain/cost_allocation"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	cloudAnalyticsCollection     = "cloudAnalytics"
	costAllocationDoc            = "gke-cost-allocations"
	cloudAnalyticsCostAllocation = "cloudAnalyticsGkeCostAllocations"
)

type CostAllocationsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

func NewCostAllocationsFirestoreWithClient(fun connection.FirestoreFromContextFun) *CostAllocationsFirestore {
	return &CostAllocationsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *CostAllocationsFirestore) getCloudAnalyticsCollectionRef(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(cloudAnalyticsCollection)
}

func (d *CostAllocationsFirestore) getCostAllocationConfigRef(ctx context.Context) *firestore.DocumentRef {
	return d.getCloudAnalyticsCollectionRef(ctx).Doc(costAllocationDoc)
}

func (d *CostAllocationsFirestore) GetCostAllocationConfig(ctx context.Context) (*domain.CostAllocationConfig, error) {
	ref := d.getCostAllocationConfigRef(ctx)
	result, err := ref.Get(ctx)

	if err != nil {
		return nil, err
	}

	var config domain.CostAllocationConfig
	err = result.DataTo(&config)

	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (d *CostAllocationsFirestore) UpdateCostAllocationConfig(ctx context.Context, newValue *domain.CostAllocationConfig) error {
	_, err := d.getCostAllocationConfigRef(ctx).Set(ctx, newValue)
	return err
}

func (d *CostAllocationsFirestore) getCustomerCostAllocationRef(ctx context.Context, customerID string) *firestore.DocumentRef {
	return d.getCostAllocationConfigRef(ctx).Collection(cloudAnalyticsCostAllocation).Doc(customerID)
}

// TODO: Refactor method to return a slice of *domain.CostAllocation
func (d *CostAllocationsFirestore) GetAllCostAllocationDocs(ctx context.Context) ([]*firestore.DocumentSnapshot, error) {
	ref := d.getCostAllocationConfigRef(ctx).Collection(cloudAnalyticsCostAllocation)
	return ref.Documents(ctx).GetAll()
}

func (d *CostAllocationsFirestore) GetAllEnabledCostAllocation(ctx context.Context) ([]*firestore.DocumentSnapshot, error) {
	ref := d.getCostAllocationConfigRef(ctx).Collection(cloudAnalyticsCostAllocation).Where("enabled", "==", true)
	return ref.Documents(ctx).GetAll()
}

func (d *CostAllocationsFirestore) GetCostAllocation(ctx context.Context, customerID string) (*domain.CostAllocation, error) {
	docRef := d.getCustomerCostAllocationRef(ctx, customerID)

	var ca domain.CostAllocation

	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	err = doc.DataTo(&ca)
	if err != nil {
		return nil, err
	}

	return &ca, err
}

func (d *CostAllocationsFirestore) UpdateCostAllocation(ctx context.Context, customerID string, newValue *domain.CostAllocation) error {
	_, err := d.getCustomerCostAllocationRef(ctx, customerID).Set(ctx, newValue)
	return err
}

func (d *CostAllocationsFirestore) CommitCostAllocations(ctx context.Context, newValues *map[string]domain.CostAllocation) []error {
	var errors []error

	var jobs []*firestore.BulkWriterJob

	bulkWriter := d.firestoreClientFun(ctx).BulkWriter(ctx)

	for customerID, v := range *newValues {
		job, err := bulkWriter.Set(d.getCustomerCostAllocationRef(ctx, customerID), v)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		jobs = append(jobs, job)
	}

	bulkWriter.End()

	for _, job := range jobs {
		_, err := job.Results()
		if err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}
