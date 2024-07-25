package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/mock"

	assetsDalMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	costAllocationDalMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/dal/mocks"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/domain/cost_allocation"
	customersDalMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func TestUpdateCustomerMissingClusters(t *testing.T) {
	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		assetsDal          *assetsDalMocks.Assets
		customersDal       *customersDalMocks.Customers
		dal                *costAllocationDalMocks.CostAllocations
	}

	testCustomerID := "test-customer"
	testCustomerDocumentRef := &firestore.DocumentRef{ID: testCustomerID}
	testBillingAccountID := "test-billing-account"
	assetsDalErr := errors.New("assets dal error")
	emptyDocument := &domain.CostAllocation{
		Customer: testCustomerDocumentRef,
	}

	noClustersDocument := &domain.CostAllocation{
		Customer:          testCustomerDocumentRef,
		BillingAccountIds: []string{testBillingAccountID},
	}

	withClustersDocument := &domain.CostAllocation{
		Customer:          testCustomerDocumentRef,
		BillingAccountIds: []string{testBillingAccountID},
		Labels: map[string][]string{
			domain.ClustersLabel: []string{
				"gke-cost-allocation-cluster-1",
				"gke-cost-allocation-cluster-2",
			},
		},
	}

	withClustersDocumentPlusUnenabled := &domain.CostAllocation{
		Customer:          testCustomerDocumentRef,
		BillingAccountIds: []string{testBillingAccountID},
		Labels: map[string][]string{
			domain.ClustersLabel: []string{
				"gke-cost-allocation-cluster-1",
				"gke-cost-allocation-cluster-2",
			},
		},
		UnenabledClusters: []string{
			"non-gke-cost-allocation-cluster-1",
		},
	}

	gcpAssets := []*pkg.GCPAsset{{
		Properties: &pkg.GCPProperties{},
	}}

	costAllocationDalErr := errors.New("cost allocation dal error")

	tests := []struct {
		name       string
		ca         *domain.CostAllocation
		baClusters domain.BillingAccountsClusters
		on         func(*fields)
		wantErr    bool
	}{
		{
			name: "GetCustomerGCPAssets fails",
			ca:   emptyDocument,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.assetsDal.On("GetCustomerGCPAssets", testutils.ContextBackgroundMock, testCustomerDocumentRef.ID).
					Return(nil, assetsDalErr).Once()
				f.loggerProviderMock.On("Error", assetsDalErr).Once()
			},
			wantErr: true,
		},
		{
			name: "No assets",
			ca:   emptyDocument,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.assetsDal.On("GetCustomerGCPAssets", testutils.ContextBackgroundMock, testCustomerDocumentRef.ID).
					Return([]*pkg.GCPAsset{}, nil).Once()
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string"), testCustomerID).Once()
			},
		},
		{
			name: "No clusters",
			ca:   noClustersDocument,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.assetsDal.On("GetCustomerGCPAssets", testutils.ContextBackgroundMock, testCustomerDocumentRef.ID).
					Return(gcpAssets, nil).Once()
				f.dal.On("UpdateCostAllocation", testutils.ContextBackgroundMock, testCustomerID, noClustersDocument).
					Return(nil).Once()
			},
		},
		{
			name: "No clusters and update fails",
			ca:   noClustersDocument,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.assetsDal.On("GetCustomerGCPAssets", testutils.ContextBackgroundMock, testCustomerDocumentRef.ID).
					Return(gcpAssets, nil).Once()
				f.dal.On("UpdateCostAllocation", testutils.ContextBackgroundMock, testCustomerID, noClustersDocument).
					Return(costAllocationDalErr).Once()
				f.loggerProviderMock.On("Error", costAllocationDalErr).Once()
			},
			wantErr: true,
		},
		{
			name: "non-gke cost allocation clusters are updated",
			ca:   withClustersDocument,
			baClusters: domain.BillingAccountsClusters{testBillingAccountID: domain.BillingAccountClusters{
				"gke-cost-allocation-cluster-1":     struct{}{},
				"gke-cost-allocation-cluster-2":     struct{}{},
				"non-gke-cost-allocation-cluster-1": struct{}{},
			}},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.AnythingOfType("map[string]string")).Once()
				f.assetsDal.On("GetCustomerGCPAssets", testutils.ContextBackgroundMock, testCustomerDocumentRef.ID).
					Return(gcpAssets, nil).Once()
				f.dal.On("UpdateCostAllocation", testutils.ContextBackgroundMock, testCustomerID, withClustersDocumentPlusUnenabled).
					Return(nil).Once()
			},
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				loggerProviderMock: loggerMocks.ILogger{},
				assetsDal:          &assetsDalMocks.Assets{},
				customersDal:       &customersDalMocks.Customers{},
				dal:                &costAllocationDalMocks.CostAllocations{},
			}

			c := &CostAllocationService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
				assetsDal:    fields.assetsDal,
				customersDal: fields.customersDal,
				dal:          fields.dal,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			err := c.updateCustomerMissingClusters(ctx, tt.ca, tt.baClusters)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateCustomerMissingClusters error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
