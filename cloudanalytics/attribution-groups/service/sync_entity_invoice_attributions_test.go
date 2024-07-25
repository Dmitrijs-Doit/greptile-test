package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"

	assetsDalMock "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	bucketsDalMock "github.com/doitintl/hello/scheduled-tasks/buckets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	attributionsDalMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/mocks"
	attributionsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service"
	attributionsServiceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/mocks"
	customerDalMock "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	entityDalMock "github.com/doitintl/hello/scheduled-tasks/entity/dal/mocks"

	"github.com/doitintl/hello/scheduled-tasks/common"

	"github.com/stretchr/testify/assert"
)

func TestAnalyticsAttributionGroupsService_removeUnnecessaryAttributions(t *testing.T) {
	type fields struct {
		attributionsDal      *attributionsDalMock.Attributions
		attributionGroupsDAL *mocks.AttributionGroups
	}

	type args struct {
		ctx                 context.Context
		attributionGroupRef *firestore.DocumentRef
		newAttributions     []*firestore.DocumentRef
		oldAttributions     []*firestore.DocumentRef
	}

	var ctx = context.Background()

	attributionGroupRef := firestore.DocumentRef{
		ID: "attribution-group-id",
	}

	newAttributions := []*firestore.DocumentRef{
		{ID: "attribution1"},
		{ID: "attribution2"},
		{ID: "attribution3"},
	}

	oldAttributions := []*firestore.DocumentRef{
		{ID: "attribution1"},
		{ID: "attribution4"},
		{ID: "attribution5"},
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error

		on func(*fields)
	}{
		{
			name: "Successfully remove unnecessary attributions",
			args: args{
				ctx,
				&attributionGroupRef,
				newAttributions,
				oldAttributions,
			},
			on: func(f *fields) {
				f.attributionGroupsDAL.On("Get", ctx, attributionGroupRef.ID).Return(&attributiongroups.AttributionGroup{
					Attributions: oldAttributions,
				}, nil).Once()
				f.attributionsDal.On("DeleteAttribution", ctx, "attribution4").Return(nil).Once()
				f.attributionsDal.On("DeleteAttribution", ctx, "attribution5").Return(nil).Once()
			},
		},
		{
			name: "Error getting attribution group",
			args: args{
				ctx,
				&attributionGroupRef,
				newAttributions,
				oldAttributions,
			},
			on: func(f *fields) {
				f.attributionGroupsDAL.On("Get", ctx, attributionGroupRef.ID).Return(nil, errors.New("error getting attribution group")).Once()
			},
			wantErr:     true,
			expectedErr: errors.New("error getting attribution group"),
		},
		{
			name: "Error deleting attribution",
			args: args{
				ctx,
				&attributionGroupRef,
				newAttributions,
				oldAttributions,
			},
			on: func(f *fields) {
				f.attributionGroupsDAL.On("Get", ctx, attributionGroupRef.ID).Return(&attributiongroups.AttributionGroup{
					Attributions: oldAttributions,
				}, nil).Once()
				f.attributionsDal.On("DeleteAttribution", ctx, "attribution4").Return(nil).Once()
				f.attributionsDal.On("DeleteAttribution", ctx, "attribution5").Return(errors.New("error deleting attribution")).Once()
			},
			wantErr:     true,
			expectedErr: errors.New("error deleting attribution"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				attributionsDal:      &attributionsDalMock.Attributions{},
				attributionGroupsDAL: &mocks.AttributionGroups{},
			}
			s := &AttributionGroupsService{
				attributionsDAL:      tt.fields.attributionsDal,
				attributionGroupsDAL: tt.fields.attributionGroupsDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.removeUnnecessaryAttributions(tt.args.ctx, tt.args.attributionGroupRef, tt.args.newAttributions)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsService.removeUnnecessaryAttributions() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

func TestAnalyticsAttributionGroupsService_getAttributionGroupRef(t *testing.T) {
	type fields struct {
		attributionGroupsDal *mocks.AttributionGroups
		customerDal          *customerDalMock.Customers
	}

	type args struct {
		ctx      context.Context
		entity   *common.Entity
		customer *common.Customer
	}

	var ctx = context.Background()

	attributionGroupRef := &firestore.DocumentRef{
		ID: "attribution-group-id",
	}

	entity := common.Entity{
		Invoicing: common.Invoicing{
			AttributionGroup: nil,
		},
		Snapshot: &firestore.DocumentSnapshot{
			Ref: &firestore.DocumentRef{
				ID: "entity-id",
			},
		},
	}

	customer := common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: &firestore.DocumentRef{
				ID: "customer-id",
			},
		},
	}

	publicAccesView := collab.PublicAccessView
	createAttributionGroupParam := &attributiongroups.AttributionGroup{
		Name:           "Invoices",
		Type:           domainAttributions.ObjectTypeManaged,
		Classification: domainAttributions.Invoice,
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{Email: "doit.com", Role: "owner"},
			},
			Public: &publicAccesView,
		},
		Customer: customer.Snapshot.Ref,
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error

		on func(*fields)
	}{
		{
			name: "Successfully get attribution group ref",
			args: args{
				ctx,
				&entity,
				&customer,
			},
			on: func(f *fields) {
				f.attributionGroupsDal.On("Create", ctx, createAttributionGroupParam).Return(attributionGroupRef.ID, nil)
				f.attributionGroupsDal.On("GetRef", ctx, attributionGroupRef.ID).Return(attributionGroupRef)
				f.customerDal.On("UpdateCustomerFieldValue", ctx, customer.Snapshot.Ref.ID, "invoiceAttributionGroup", attributionGroupRef).Return(nil).Once()
			},
		},
		{
			name: "Error creating attribution group",
			args: args{
				ctx,
				&entity,
				&customer,
			},
			on: func(f *fields) {
				f.attributionGroupsDal.On("Create", ctx, createAttributionGroupParam).Return("", errors.New("error creating attribution group"))
			},
			expectedErr: errors.New("error creating attribution group"),
			wantErr:     true,
		},
		{
			name: "Error updating customer",
			args: args{
				ctx,
				&entity,
				&customer,
			},
			on: func(f *fields) {
				f.attributionGroupsDal.On("Create", ctx, createAttributionGroupParam).Return(attributionGroupRef.ID, nil)
				f.attributionGroupsDal.On("GetRef", ctx, attributionGroupRef.ID).Return(attributionGroupRef)
				f.customerDal.On("UpdateCustomerFieldValue", ctx, customer.Snapshot.Ref.ID, "invoiceAttributionGroup", attributionGroupRef).Return(errors.New("error updating customer")).Once()
			},
			expectedErr: errors.New("error updating customer"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				attributionGroupsDal: &mocks.AttributionGroups{},
				customerDal:          &customerDalMock.Customers{},
			}
			s := &AttributionGroupsService{
				attributionGroupsDAL: tt.fields.attributionGroupsDal,
				customersDAL:         tt.fields.customerDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.getAttributionGroupRef(tt.args.ctx, tt.args.entity, tt.args.customer)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsService.getAttributionGroupRef() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}

			if !tt.wantErr {
				assert.EqualValues(t, attributionGroupRef, got)
			}
		})
	}
}

func TestAnalyticsAttributionGroupsService_getAttributionsForBuckets(t *testing.T) {
	type fields struct {
		entityDal           *entityDalMock.Entites
		bucketsDal          *bucketsDalMock.Buckets
		assetsDal           *assetsDalMock.Assets
		attributionsService *attributionsServiceMock.AttributionsIface
		attributionsDAL     *attributionsDalMock.Attributions
	}

	type args struct {
		ctx      context.Context
		entity   *common.Entity
		customer *common.Customer
	}

	var ctx = context.Background()

	entity := common.Entity{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: &firestore.DocumentRef{
				ID: "entity-id",
			},
		},
		Invoicing: common.Invoicing{
			Default: &firestore.DocumentRef{
				ID: "default-id",
			},
		},
	}

	customer := common.Customer{}

	getBucketsResult := []common.Bucket{
		{Ref: &firestore.DocumentRef{ID: "bucket1"}},
		{Ref: &firestore.DocumentRef{ID: "bucket2"}},
		{Ref: &firestore.DocumentRef{ID: "bucket3"}},
		{Ref: &firestore.DocumentRef{ID: "bucket4"}, Attribution: &firestore.DocumentRef{ID: "test"}},
	}

	getAssetsResult1 := []*pkg.BaseAsset{
		{AssetType: common.Assets.AmazonWebServices},
	}

	getAssetsResult2 := []*pkg.BaseAsset{
		{AssetType: common.Assets.GoogleCloudProject},
	}

	getAssetsResult3 := []*pkg.BaseAsset{
		{AssetType: common.Assets.GSuite},
	}

	var getAssetsResult4 []*pkg.BaseAsset

	createBucketResult1 := firestore.DocumentRef{ID: "attribution1"}
	createBucketResult2 := firestore.DocumentRef{ID: "attribution2"}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error

		on func(*fields)
	}{
		{
			name: "Successfully get attributions for buckets",
			args: args{
				ctx,
				&entity,
				&customer,
			},
			on: func(f *fields) {
				f.bucketsDal.On("GetBuckets", ctx, entity.Snapshot.Ref.ID).Return(getBucketsResult[:3], nil)
				f.assetsDal.On("GetAssetsInBucket", ctx, getBucketsResult[0].Ref).Return(getAssetsResult1, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, getBucketsResult[1].Ref).Return(getAssetsResult2, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, getBucketsResult[2].Ref).Return(getAssetsResult3, nil).Once()
				f.attributionsService.On("CreateBucketAttribution", ctx, &attributionsService.SyncBucketAttributionRequest{
					Customer: &customer,
					Bucket:   &getBucketsResult[0],
					Entity:   &entity,
					Assets:   getAssetsResult1,
				}).Return(&createBucketResult1, nil)
				f.attributionsService.On("CreateBucketAttribution", ctx, &attributionsService.SyncBucketAttributionRequest{
					Customer: &customer,
					Bucket:   &getBucketsResult[1],
					Entity:   &entity,
					Assets:   getAssetsResult2,
				}).Return(&createBucketResult2, nil)
			},
		},
		{
			name: "Successfully remove attribution if no assets in bucket",
			args: args{
				ctx,
				&entity,
				&customer,
			},
			on: func(f *fields) {
				f.bucketsDal.On("GetBuckets", ctx, entity.Snapshot.Ref.ID).Return(getBucketsResult, nil)
				f.assetsDal.On("GetAssetsInBucket", ctx, getBucketsResult[0].Ref).Return(getAssetsResult1, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, getBucketsResult[1].Ref).Return(getAssetsResult2, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, getBucketsResult[2].Ref).Return(getAssetsResult3, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, getBucketsResult[3].Ref).Return(getAssetsResult4, nil).Once()
				f.bucketsDal.On("UpdateBucket", ctx, entity.Snapshot.Ref.ID, getBucketsResult[3].Ref.ID, []firestore.Update{{
					Path:  "attribution",
					Value: nil,
				}}).Return(nil)
				f.attributionsDAL.On("DeleteAttribution", ctx, getBucketsResult[3].Attribution.ID).Return(nil)
				f.attributionsService.On("CreateBucketAttribution", ctx, &attributionsService.SyncBucketAttributionRequest{
					Customer: &customer,
					Bucket:   &getBucketsResult[0],
					Entity:   &entity,
					Assets:   getAssetsResult1,
				}).Return(&createBucketResult1, nil)
				f.attributionsService.On("CreateBucketAttribution", ctx, &attributionsService.SyncBucketAttributionRequest{
					Customer: &customer,
					Bucket:   &getBucketsResult[1],
					Entity:   &entity,
					Assets:   getAssetsResult2,
				}).Return(&createBucketResult2, nil)
			},
		},
		{
			name: "Skip bucket if attribution is already nil",
			args: args{
				ctx,
				&entity,
				&customer,
			},
			on: func(f *fields) {
				buckets := []common.Bucket{{Ref: &firestore.DocumentRef{ID: "nil-attr-bucket"}, Attribution: nil}}
				buckets = append(buckets, getBucketsResult[:2]...)

				f.bucketsDal.On("GetBuckets", ctx, entity.Snapshot.Ref.ID).Return(buckets, nil)
				f.assetsDal.On("GetAssetsInBucket", ctx, buckets[0].Ref).Return([]*pkg.BaseAsset{}, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, buckets[1].Ref).Return(getAssetsResult1, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, buckets[2].Ref).Return(getAssetsResult2, nil).Once()
				f.attributionsService.On("CreateBucketAttribution", ctx, &attributionsService.SyncBucketAttributionRequest{
					Customer: &customer,
					Bucket:   &getBucketsResult[0],
					Entity:   &entity,
					Assets:   getAssetsResult1,
				}).Return(&createBucketResult1, nil)
				f.attributionsService.On("CreateBucketAttribution", ctx, &attributionsService.SyncBucketAttributionRequest{
					Customer: &customer,
					Bucket:   &getBucketsResult[1],
					Entity:   &entity,
					Assets:   getAssetsResult2,
				}).Return(&createBucketResult2, nil)
			},
		},
		{
			name: "Error deleting attribution when assets are empty",
			args: args{
				ctx,
				&entity,
				&customer,
			},
			on: func(f *fields) {
				buckets := []common.Bucket{{Ref: &firestore.DocumentRef{ID: "nil-attr-bucket"}, Attribution: &firestore.DocumentRef{ID: "test"}}}
				buckets = append(buckets, getBucketsResult[:2]...)

				f.bucketsDal.On("GetBuckets", ctx, entity.Snapshot.Ref.ID).Return(buckets, nil)
				f.assetsDal.On("GetAssetsInBucket", ctx, buckets[0].Ref).Return([]*pkg.BaseAsset{}, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, buckets[1].Ref).Return(getAssetsResult1, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, buckets[2].Ref).Return(getAssetsResult2, nil).Once()
				f.attributionsDAL.On("DeleteAttribution", ctx, buckets[0].Attribution.ID).Return(errors.New("error deleting attribution"))

				f.attributionsService.On("CreateBucketAttribution", ctx, &attributionsService.SyncBucketAttributionRequest{
					Customer: &customer,
					Bucket:   &getBucketsResult[0],
					Entity:   &entity,
					Assets:   getAssetsResult1,
				}).Return(&createBucketResult1, nil)
				f.attributionsService.On("CreateBucketAttribution", ctx, &attributionsService.SyncBucketAttributionRequest{
					Customer: &customer,
					Bucket:   &getBucketsResult[1],
					Entity:   &entity,
					Assets:   getAssetsResult2,
				}).Return(&createBucketResult2, nil)
			},
			wantErr:     true,
			expectedErr: errors.New("error deleting attribution"),
		},
		{
			name: "Error updating bucket when assets are empty",
			args: args{
				ctx,
				&entity,
				&customer,
			},
			on: func(f *fields) {
				buckets := []common.Bucket{{Ref: &firestore.DocumentRef{ID: "nil-attr-bucket"}, Attribution: &firestore.DocumentRef{ID: "test"}}}
				buckets = append(buckets, getBucketsResult[:2]...)

				f.bucketsDal.On("GetBuckets", ctx, entity.Snapshot.Ref.ID).Return(buckets, nil)
				f.assetsDal.On("GetAssetsInBucket", ctx, buckets[0].Ref).Return([]*pkg.BaseAsset{}, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, buckets[1].Ref).Return(getAssetsResult1, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, buckets[2].Ref).Return(getAssetsResult2, nil).Once()
				f.attributionsDAL.On("DeleteAttribution", ctx, buckets[0].Attribution.ID).Return(nil)
				f.bucketsDal.On("UpdateBucket", ctx, entity.Snapshot.Ref.ID, buckets[0].Ref.ID, []firestore.Update{{
					Path:  "attribution",
					Value: nil,
				}}).Return(errors.New("error updating bucket"))

				f.attributionsService.On("CreateBucketAttribution", ctx, &attributionsService.SyncBucketAttributionRequest{
					Customer: &customer,
					Bucket:   &getBucketsResult[0],
					Entity:   &entity,
					Assets:   getAssetsResult1,
				}).Return(&createBucketResult1, nil)
				f.attributionsService.On("CreateBucketAttribution", ctx, &attributionsService.SyncBucketAttributionRequest{
					Customer: &customer,
					Bucket:   &getBucketsResult[1],
					Entity:   &entity,
					Assets:   getAssetsResult2,
				}).Return(&createBucketResult2, nil)
			},
			wantErr:     true,
			expectedErr: errors.New("error updating bucket"),
		},
		{
			name: "Error getting buckets",
			args: args{
				ctx,
				&entity,
				&customer,
			},
			on: func(f *fields) {
				f.bucketsDal.On("GetBuckets", ctx, entity.Snapshot.Ref.ID).Return(nil, errors.New("error getting buckets"))
			},
			wantErr:     true,
			expectedErr: errors.New("error getting buckets"),
		},
		{
			name: "Error getting assets in bucket",
			args: args{
				ctx,
				&entity,
				&customer,
			},
			on: func(f *fields) {
				f.bucketsDal.On("GetBuckets", ctx, entity.Snapshot.Ref.ID).Return(getBucketsResult[:3], nil)
				f.assetsDal.On("GetAssetsInBucket", ctx, getBucketsResult[0].Ref).Return(getAssetsResult1, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, getBucketsResult[1].Ref).Return(nil, errors.New("error getting assets in bucket")).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, getBucketsResult[2].Ref).Return(getAssetsResult3, nil)
				f.attributionsService.On("CreateBucketAttribution", ctx, &attributionsService.SyncBucketAttributionRequest{
					Customer: &customer,
					Bucket:   &getBucketsResult[0],
					Entity:   &entity,
					Assets:   getAssetsResult1,
				}).Return(&createBucketResult1, nil)
			},
			wantErr:     true,
			expectedErr: errors.New("error getting assets in bucket"),
		},
		{
			name: "Error creating attribution",
			args: args{
				ctx,
				&entity,
				&customer,
			},
			on: func(f *fields) {
				f.bucketsDal.On("GetBuckets", ctx, entity.Snapshot.Ref.ID).Return(getBucketsResult[:3], nil)
				f.assetsDal.On("GetAssetsInBucket", ctx, getBucketsResult[0].Ref).Return(getAssetsResult1, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, getBucketsResult[1].Ref).Return(getAssetsResult2, nil).Once()
				f.assetsDal.On("GetAssetsInBucket", ctx, getBucketsResult[2].Ref).Return(getAssetsResult3, nil).Once()
				f.attributionsService.On("CreateBucketAttribution", ctx, &attributionsService.SyncBucketAttributionRequest{
					Customer: &customer,
					Bucket:   &getBucketsResult[0],
					Entity:   &entity,
					Assets:   getAssetsResult1,
				}).Return(&createBucketResult1, nil)
				f.attributionsService.On("CreateBucketAttribution", ctx, &attributionsService.SyncBucketAttributionRequest{
					Customer: &customer,
					Bucket:   &getBucketsResult[1],
					Entity:   &entity,
					Assets:   getAssetsResult2,
				}).Return(nil, errors.New("error creating attribution"))
			},
			wantErr:     true,
			expectedErr: errors.New("error creating attribution"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				entityDal:           &entityDalMock.Entites{},
				bucketsDal:          &bucketsDalMock.Buckets{},
				assetsDal:           &assetsDalMock.Assets{},
				attributionsService: &attributionsServiceMock.AttributionsIface{},
				attributionsDAL:     &attributionsDalMock.Attributions{},
			}
			s := &AttributionGroupsService{
				entityDal:           tt.fields.entityDal,
				bucketsDal:          tt.fields.bucketsDal,
				assetsDal:           tt.fields.assetsDal,
				attributionsService: tt.fields.attributionsService,
				attributionsDAL:     tt.fields.attributionsDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.getAttributionsForBuckets(tt.args.ctx, tt.args.entity, tt.args.customer)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsService.getAttributionsForBuckets() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}

			if !tt.wantErr {
				assert.Contains(t, got, &createBucketResult1)
				assert.Contains(t, got, &createBucketResult2)
			}
		})
	}
}

func TestAnalyticsAttributionGroupsService_getCustomerInvoiceByTypeAttributions(t *testing.T) {
	type fields struct {
		entityDal *entityDalMock.Entites
	}

	type args struct {
		ctx          context.Context
		attributions []*attribution.Attribution
		customer     *common.Customer
	}

	ctx := context.Background()

	customer := common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: &firestore.DocumentRef{
				ID: "customer-id",
			},
		},
	}

	getCustomerEntitiesResult := []*common.Entity{
		{Invoicing: common.Invoicing{Mode: "GROUP"}, Name: "entity-1", PriorityID: "123", Active: true},
		{Invoicing: common.Invoicing{Mode: "CUSTOM"}, Name: "entity-2", PriorityID: "456"},
		{Invoicing: common.Invoicing{Mode: "GROUP"}, Name: "entity-3", PriorityID: "789", Active: true},
	}

	attributions := []*attribution.Attribution{
		{Name: "[123] entity-1 - Google Cloud", Ref: &firestore.DocumentRef{ID: "attribution-1"}},
		{Name: "[123] entity-1 - Amazon Web Services", Ref: &firestore.DocumentRef{ID: "attribution-2"}},
		{Name: "entity-2 - Bucket 1", Ref: &firestore.DocumentRef{ID: "attribution-3"}},
		{Name: "entity-2 - Bucket 2", Ref: &firestore.DocumentRef{ID: "attribution-4"}},
		{Name: "[789] entity-3 - Google Cloud", Ref: &firestore.DocumentRef{ID: "attribution-5"}},
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantErr        bool
		expectedErr    error
		expectedResult []*firestore.DocumentRef

		on func(*fields)
	}{
		{
			name: "Successfully get attributions for invoice by type",
			args: args{
				ctx,
				attributions,
				&customer,
			},
			on: func(f *fields) {
				f.entityDal.On("GetCustomerEntities", ctx, customer.Snapshot.Ref).Return(getCustomerEntitiesResult, nil)
			},
			expectedResult: []*firestore.DocumentRef{
				{ID: "attribution-1"},
				{ID: "attribution-2"},
				{ID: "attribution-5"},
			},
		},
		{
			name: "Error getting customer entities",
			args: args{
				ctx,
				attributions,
				&customer,
			},
			on: func(f *fields) {
				f.entityDal.On("GetCustomerEntities", ctx, customer.Snapshot.Ref).Return(nil, errors.New("error getting entities"))
			},
			wantErr:     true,
			expectedErr: errors.New("error getting entities"),
		},
		{
			name: "Empty attributions list",
			args: args{
				ctx,
				[]*attribution.Attribution{},
				&customer,
			},
			on: func(f *fields) {
				f.entityDal.On("GetCustomerEntities", ctx, customer.Snapshot.Ref).Return(getCustomerEntitiesResult, nil)
			},
			expectedResult: []*firestore.DocumentRef{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				entityDal: &entityDalMock.Entites{},
			}
			s := &AttributionGroupsService{
				entityDal: tt.fields.entityDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.getCustomerInvoiceByTypeAttributions(tt.args.ctx, tt.args.attributions, tt.args.customer)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsService.getCustomerInvoiceByTypeAttributions() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}

			if !tt.wantErr {
				assert.Equal(t, tt.expectedResult, got)
			}
		})
	}
}
