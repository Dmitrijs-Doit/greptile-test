package service

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	attributionsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	contractDomain "github.com/doitintl/hello/scheduled-tasks/contract/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type getEntityResult struct {
	entity *common.Entity
	err    error
}
type getAttributionResult struct {
	attributionRef *firestore.DocumentRef
	err            error
}

type getCustomerResult struct {
	customer *common.Customer
	err      error
}
type getAttributionsForGroupResult struct {
	attributionsRef []*firestore.DocumentRef
	err             error
}

func (s *AttributionGroupsService) SyncEntityInvoiceAttributions(ctx context.Context, req SyncEntityInvoiceAttributionsRequest) error {
	entity, customer, err := s.getEntityAndCustomer(ctx, req)
	if err != nil {
		return err
	}

	if !entity.Active {
		return nil
	}

	if customer.Terminated() {
		return attributiongroups.ErrCustomerIsTerminated
	}

	if entity.Customer.ID != customer.Snapshot.Ref.ID {
		return attributiongroups.ErrEntityFromDifferentCustomer
	}

	attributionGroup, newEntityAttributions, err := s.getEntityAttributionsAndUpsertAttributionGroup(ctx, entity, customer)
	if err != nil {
		return err
	}

	attributionGroupRef := s.attributionGroupsDAL.GetRef(ctx, attributionGroup.ID)

	var customerInvoiceAttributions []*attributionDomain.Attribution
	if len(attributionGroup.Attributions) > 0 {
		customerInvoiceAttributions, err = s.attributionsDAL.GetAttributions(ctx, attributionGroup.Attributions)
		if err != nil {
			return err
		}
	}

	bucketsAttributions, err := s.getCustomerBucketsAttributions(ctx, customer)
	if err != nil {
		return err
	}

	invoiceByTypeAttributions, err := s.getCustomerInvoiceByTypeAttributions(ctx, customerInvoiceAttributions, customer)
	if err != nil {
		return err
	}

	allEntitiesAttributions := append(bucketsAttributions, invoiceByTypeAttributions...)

	customerHasLookerContract, err := s.customerHasLookerContract(ctx, customer.Snapshot.Ref)
	if err != nil {
		return err
	}

	if customerHasLookerContract {
		lookerAttribution, err := s.getOrCreateLookerAttributionRef(ctx, customerInvoiceAttributions, customer)
		if err != nil {
			return err
		}

		allEntitiesAttributions = append(allEntitiesAttributions, lookerAttribution)
	}

	err = s.removeUnnecessaryAttributionsAndUpdateAttributionGroup(ctx, attributionGroupRef, newEntityAttributions, allEntitiesAttributions)
	if err != nil {
		return err
	}

	return nil
}

func (s *AttributionGroupsService) removeUnnecessaryAttributionsAndUpdateAttributionGroup(ctx context.Context, attributionGroupRef *firestore.DocumentRef, newEntityAttributions []*firestore.DocumentRef, allEntitiesAttributions []*firestore.DocumentRef) error {
	errorsChannel := make(chan error, 2)
	newGroupAttributions := allEntitiesAttributions

	for _, attribution := range newEntityAttributions {
		if !sliceContains(allEntitiesAttributions, attribution) {
			newGroupAttributions = append(newGroupAttributions, attribution)
		}
	}

	go func() {
		errorsChannel <- s.removeUnnecessaryAttributions(ctx, attributionGroupRef, newGroupAttributions)
	}()

	go func() {
		errorsChannel <- s.attributionGroupsDAL.Update(ctx, attributionGroupRef.ID, &attributiongroups.AttributionGroup{
			Name:         "Invoices",
			Attributions: newGroupAttributions,
		})
	}()

	errorResults := []error{}

	for i := 0; i < 2; i++ {
		err := <-errorsChannel
		if err != nil {
			errorResults = append(errorResults, err)
		}
	}

	if len(errorResults) > 0 {
		return errorResults[0]
	}

	return nil
}

func (s *AttributionGroupsService) getEntityAttributionsAndUpsertAttributionGroup(ctx context.Context, entity *common.Entity, customer *common.Customer) (*attributiongroups.AttributionGroup, []*firestore.DocumentRef, error) {
	attributionGroupRef, err := s.getAttributionGroupRef(ctx, entity, customer)
	if err != nil {
		return nil, nil, err
	}

	attributionGroup, err := s.attributionGroupsDAL.Get(ctx, attributionGroupRef.ID)
	if err != nil {
		return nil, nil, err
	}

	var entityAttributions []*firestore.DocumentRef

	if entity.Invoicing.Mode == "CUSTOM" {
		entityAttributions, err = s.getAttributionsForBuckets(ctx, entity, customer)
		if err != nil {
			return nil, nil, err
		}
	} else if entity.Invoicing.Mode == "GROUP" {
		entityAttributions, err = s.attributionsService.CreateAttributionsForInvoiceAssetTypes(ctx, attributionsService.SyncInvoiceByAssetTypeAttributionRequest{
			Customer:         customer,
			AttributionGroup: attributionGroup,
			Entity:           entity,
		})
		if err != nil {
			return nil, nil, err
		}
	}

	return attributionGroup, entityAttributions, nil
}

func (s *AttributionGroupsService) getEntityAndCustomer(ctx context.Context, req SyncEntityInvoiceAttributionsRequest) (*common.Entity, *common.Customer, error) {
	getEntity := make(chan getEntityResult)

	go func() {
		entity, err := s.entityDal.GetEntity(ctx, req.EntityID)
		getEntity <- getEntityResult{entity: entity, err: err}
	}()

	getCustomer := make(chan getCustomerResult)

	go func() {
		customer, err := s.customersDAL.GetCustomer(ctx, req.CustomerID)
		getCustomer <- getCustomerResult{customer: customer, err: err}
	}()

	getEntityResult := <-getEntity
	getCustomerResult := <-getCustomer

	if getEntityResult.err != nil {
		return nil, nil, getEntityResult.err
	}

	if getCustomerResult.err != nil {
		return nil, nil, getCustomerResult.err
	}

	return getEntityResult.entity, getCustomerResult.customer, nil
}

func (s *AttributionGroupsService) getCustomerBucketsAttributions(ctx context.Context, customer *common.Customer) ([]*firestore.DocumentRef, error) {
	customerBuckets, err := s.bucketsService.GetCustomerBuckets(ctx, customer.Snapshot.Ref)
	if err != nil {
		return nil, err
	}

	numJobs := len(customerBuckets)
	jobs := make(chan common.Bucket, numJobs)
	results := make(chan *getAttributionResult, numJobs)

	defer close(jobs)

	for i := 0; i < 100; i++ {
		go s.getExistingBucketAttributionWorker(ctx, jobs, results)
	}

	for _, bucket := range customerBuckets {
		jobs <- bucket
	}

	attributions := []*firestore.DocumentRef{}

	for i := 0; i < numJobs; i++ {
		result := <-results

		if result == nil {
			continue
		}

		if result.err != nil {
			return nil, result.err
		}

		attributions = append(attributions, result.attributionRef)
	}

	return attributions, nil
}

func (s *AttributionGroupsService) getCustomerInvoiceByTypeAttributions(ctx context.Context, customerInvoiceAttributions []*attributionDomain.Attribution, customer *common.Customer) ([]*firestore.DocumentRef, error) {
	customerEntities, err := s.entityDal.GetCustomerEntities(ctx, customer.Snapshot.Ref)

	if err != nil {
		return nil, err
	}

	invoiceByTypeAttributions := []*firestore.DocumentRef{}

	for _, entity := range customerEntities {
		entityNameWithoutQuotes := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(entity.Name, "`", ""), "\"", ""), "'", "")

		if entity.Active && entity.Invoicing.Mode == "GROUP" {
			for _, attribution := range customerInvoiceAttributions {
				if attribution.Name == "["+entity.PriorityID+"] "+entityNameWithoutQuotes+" - Google Cloud" ||
					attribution.Name == "["+entity.PriorityID+"] "+entityNameWithoutQuotes+" - Amazon Web Services" {
					invoiceByTypeAttributions = append(invoiceByTypeAttributions, attribution.Ref)
				}
			}
		}
	}

	return invoiceByTypeAttributions, nil
}

func (s *AttributionGroupsService) getExistingBucketAttributionWorker(ctx context.Context, buckets <-chan common.Bucket, results chan<- *getAttributionResult) {
	for bucket := range buckets {
		if bucket.Attribution != nil {
			_, err := s.attributionsDAL.GetAttribution(ctx, bucket.Attribution.ID)
			if err == attributionDomain.ErrNotFound {
				results <- nil
			} else {
				results <- &getAttributionResult{attributionRef: bucket.Attribution, err: err}
			}
		} else {
			results <- nil
		}
	}
}

func (s *AttributionGroupsService) getAttributionsForBuckets(ctx context.Context, entity *common.Entity, customer *common.Customer) ([]*firestore.DocumentRef, error) {
	entityBuckets, err := s.bucketsDal.GetBuckets(ctx, entity.Snapshot.Ref.ID)
	if err != nil {
		return nil, err
	}

	results := make(chan *getAttributionResult, len(entityBuckets))

	jobs := make(chan common.Bucket, len(entityBuckets))
	defer close(jobs)

	for i := 0; i < 100; i++ {
		go s.getAttributionForBucketWorker(ctx, entity, customer, jobs, results)
	}

	for _, bucket := range entityBuckets {
		jobs <- bucket
	}

	newAttributions := []*firestore.DocumentRef{}
	errorResults := []error{}

	for i := 0; i < len(entityBuckets); i++ {
		result := <-results
		if result == nil {
			continue
		}

		if result.err != nil {
			errorResults = append(errorResults, result.err)
		}

		newAttributions = append(newAttributions, result.attributionRef)
	}

	if len(errorResults) > 0 {
		return nil, errorResults[0]
	}

	return newAttributions, nil
}

func (s *AttributionGroupsService) getAttributionForBucketWorker(ctx context.Context, entity *common.Entity, customer *common.Customer, buckets <-chan common.Bucket, results chan<- *getAttributionResult) {
	for bucket := range buckets {
		assets, err := s.assetsDal.GetAssetsInBucket(ctx, bucket.Ref)
		if err != nil {
			results <- &getAttributionResult{attributionRef: nil, err: err}
			continue
		}

		if len(assets) == 0 && (entity.Invoicing.Default == nil || entity.Invoicing.Default.ID != bucket.Ref.ID) {
			results <- s.handleEmptyAssets(ctx, entity, bucket)
			continue
		}

		var bucketType string
		if len(assets) > 0 {
			bucketType = assets[0].AssetType
		}

		if bucketType == common.Assets.AmazonWebServices || bucketType == common.Assets.GoogleCloud || bucketType == common.Assets.GoogleCloudProject || (entity.Invoicing.Default != nil && bucket.Ref.ID == entity.Invoicing.Default.ID) {
			attributionRef, err := s.attributionsService.CreateBucketAttribution(ctx, &attributionsService.SyncBucketAttributionRequest{
				Customer: customer,
				Bucket:   &bucket,
				Entity:   entity,
				Assets:   assets,
			})
			results <- &getAttributionResult{attributionRef: attributionRef, err: err}

			continue
		}

		results <- nil
	}
}

func (s *AttributionGroupsService) getAttributionGroupRef(ctx context.Context, entity *common.Entity, customer *common.Customer) (*firestore.DocumentRef, error) {
	attributionGroupRef := customer.InvoiceAttributionGroup

	var attributionGroupSnap *firestore.DocumentSnapshot

	var err error

	if attributionGroupRef != nil {
		attributionGroupSnap, err = attributionGroupRef.Get(ctx)
		if err != nil && status.Code(err) != codes.NotFound {
			return nil, err
		}
	}

	if attributionGroupRef == nil || !attributionGroupSnap.Exists() {
		publicAccesView := collab.PublicAccessView
		attributionGroupID, err := s.attributionGroupsDAL.Create(ctx, &attributiongroups.AttributionGroup{
			Name:           "Invoices",
			Type:           attributionDomain.ObjectTypeManaged,
			Classification: attributionDomain.Invoice,
			Access: collab.Access{
				Collaborators: []collab.Collaborator{
					{Email: "doit.com", Role: "owner"},
				},
				Public: &publicAccesView,
			},
			Customer: customer.Snapshot.Ref,
		})

		if err != nil {
			return nil, err
		}

		attributionGroupRef = s.attributionGroupsDAL.GetRef(ctx, attributionGroupID)
		if err = s.customersDAL.UpdateCustomerFieldValue(ctx, customer.Snapshot.Ref.ID, "invoiceAttributionGroup", attributionGroupRef); err != nil {
			return nil, err
		}
	}

	return attributionGroupRef, nil
}

func (s *AttributionGroupsService) removeUnnecessaryAttributions(ctx context.Context, attributionGroupRef *firestore.DocumentRef, newGroupAttributions []*firestore.DocumentRef) error {
	attributionGroup, err := s.attributionGroupsDAL.Get(ctx, attributionGroupRef.ID)
	if err != nil {
		return err
	}

	errors := make(chan error, len(attributionGroup.Attributions))

	jobs := make(chan *firestore.DocumentRef, len(attributionGroup.Attributions))
	defer close(jobs)

	for i := 0; i < 100; i++ {
		go s.removeUnnecessaryAttributionsWorker(ctx, newGroupAttributions, jobs, errors)
	}

	for _, attribution := range attributionGroup.Attributions {
		jobs <- attribution
	}

	errorResults := []error{}

	for i := 0; i < len(attributionGroup.Attributions); i++ {
		err := <-errors
		if err != nil {
			errorResults = append(errorResults, err)
		}
	}

	if len(errorResults) > 0 {
		return errorResults[0]
	}

	return nil
}

func (s *AttributionGroupsService) removeUnnecessaryAttributionsWorker(ctx context.Context, newGroupAttributions []*firestore.DocumentRef, attributions <-chan *firestore.DocumentRef, errors chan<- error) {
	for attribution := range attributions {
		if !sliceContains(newGroupAttributions, attribution) {
			errors <- s.attributionsDAL.DeleteAttribution(ctx, attribution.ID)

			continue
		}

		errors <- nil
	}
}

func sliceContains(s []*firestore.DocumentRef, e *firestore.DocumentRef) bool {
	for _, a := range s {
		if a.ID == e.ID {
			return true
		}
	}

	return false
}

func (s *AttributionGroupsService) handleEmptyAssets(ctx context.Context, entity *common.Entity, bucket common.Bucket) *getAttributionResult {
	if bucket.Attribution == nil {
		return nil
	}

	// Remove attribution as it is empty
	if err := s.attributionsDAL.DeleteAttribution(ctx, bucket.Attribution.ID); err != nil {
		return &getAttributionResult{attributionRef: nil, err: err}
	}

	if err := s.bucketsDal.UpdateBucket(ctx, entity.Snapshot.Ref.ID, bucket.Ref.ID, []firestore.Update{
		{Path: "attribution", Value: nil},
	}); err != nil {
		return &getAttributionResult{attributionRef: nil, err: err}
	}

	return nil
}

func (s *AttributionGroupsService) getOrCreateLookerAttributionRef(ctx context.Context, customerInvoiceAttributions []*attributionDomain.Attribution, customer *common.Customer) (*firestore.DocumentRef, error) {
	publicAccessView := collab.PublicAccessView

	for _, attribution := range customerInvoiceAttributions {
		if attribution.Name == "Looker" {
			if err := s.attributionsDAL.UpdateAttribution(ctx, attribution.ID, []firestore.Update{
				{Path: "type", Value: "managed"},
				{Path: "classification", Value: "invoice"},
				{Path: "collaborators", Value: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}}},
				{Path: "public", Value: &publicAccessView},
				{Path: "hidden", Value: true},
			}); err != nil {
				return nil, err
			}

			return attribution.Ref, nil
		}
	}

	filterID := fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeFixed, metadata.MetadataFieldKeyServiceDescription)
	md, key, err := cloudanalytics.ParseID(filterID)

	if err != nil {
		return nil, err
	}

	attribution, err := s.attributionsDAL.CreateAttribution(ctx, &attributionDomain.Attribution{
		Type:           "managed",
		Classification: "invoice",
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
			},
			Public: &publicAccessView,
		},
		Hidden:   true,
		Customer: customer.Snapshot.Ref,
		Name:     "Looker",
		Filters: []report.BaseConfigFilter{
			{
				Field:  md.Field,
				ID:     filterID,
				Key:    key,
				Type:   md.Type,
				Values: &[]string{"Looker"},
			},
		},
		Formula: "A",
	})

	if err != nil {
		return nil, err
	}

	return attribution.Ref, nil
}

func (s *AttributionGroupsService) customerHasLookerContract(ctx context.Context, customerRef *firestore.DocumentRef) (bool, error) {
	lookerContracts, err := s.contractsDAL.GetContractsByType(ctx, customerRef, contractDomain.ContractTypeLooker)

	if err != nil {
		return false, err
	}

	return len(lookerContracts) > 0, nil
}
