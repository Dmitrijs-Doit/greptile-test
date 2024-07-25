package datahubmetric

import (
	"context"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal"
	datahubMetricDalMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/datahubmetric/mocks"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub"
	service "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/datahub"
	"github.com/doitintl/hello/scheduled-tasks/common"
	testPackage "github.com/doitintl/tests"
)

func TestUpdateMetadataDocs(t *testing.T) {
	ctx := context.Background()

	testsCustomerID := "JhV7WydpTlW8DfVRVVMg"
	testCustomerRef := &firestore.DocumentRef{
		Path: "projects/doitintl-cmp-dev/databases/(default)/documents/customers/" + testsCustomerID,
		ID:   testsCustomerID,
	}

	type args struct {
		customerRef           *firestore.DocumentRef
		updatefunc            domain.UpdateMetadataDocsPostTxFunc
		mergeMetadataDoc      domain.MergeMetadataDocFunc
		metadataByCustomer    domain.MetadataByCustomer
		metricTypesByCustomer domain.MetricTypesByCustomer
	}

	type fields struct {
		datahubMetricFirestore *datahubMetricDalMocks.DataHubMetricFirestore
	}

	datahubMetricDocRef := &firestore.DocumentRef{
		Path: "projects/doitintl-cmp-dev/databases/(default)/documents/cloudAnalytics/metrics/datahubMetrics/" + testsCustomerID,
	}

	datahubMetrics := &domain.DataHubMetrics{
		Metrics: []domain.DataHubMetric{
			{
				DataSource: "datahub-billing",
				Key:        "rides",
				Label:      "rides",
			},
			{
				DataSource: "datahub-billing",
				Key:        "some-existing-metric",
				Label:      "some-existing-metric",
			},
		},
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		fields      fields
		on          func(*fields)
		expectedLen int
	}{
		{
			name: "update metric types",
			args: args{
				customerRef: testCustomerRef,
				metricTypesByCustomer: map[*firestore.DocumentRef]map[string]bool{
					testCustomerRef: {
						"rides":       true,
						"deployments": true,
					},
				},
				updatefunc: func() error {
					return nil
				},
			},
			expectedLen: 3,
			on: func(f *fields) {
				f.datahubMetricFirestore.On(
					"GetRef",
					mock.AnythingOfType("*context.valueCtx"),
					testsCustomerID,
				).Return(datahubMetricDocRef).
					Once()
				f.datahubMetricFirestore.On(
					"GetMergeableDocument",
					mock.AnythingOfType("*firestore.Transaction"),
					mock.AnythingOfType("*firestore.DocumentRef"),
				).Return(datahubMetrics, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				datahubMetricFirestore: &datahubMetricDalMocks.DataHubMetricFirestore{},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			datahubMetadataFirestoreDAL, err := dal.NewDataHubMetadataFirestore(ctx, common.TestProjectID)
			if err != nil {
				t.Fatal(err)
			}

			s := service.NewDataHubMetadataService(nil, nil, tt.fields.datahubMetricFirestore, nil, nil)

			err = datahubMetadataFirestoreDAL.UpdateMetadataDocs(
				ctx,
				tt.args.mergeMetadataDoc,
				tt.args.updatefunc,
				tt.args.metadataByCustomer,
				tt.args.metricTypesByCustomer,
				s.MergeExtendedMetricTypesDoc,
			)
			if err != nil && tt.wantErr == false {
				t.Errorf("UpdateMetadataDocs error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			datahubMetricFirestoreDAL, err := NewDataHubMetricFirestore(ctx, common.TestProjectID)
			if err != nil {
				t.Fatal(err)
			}

			datahubMetric, err := datahubMetricFirestoreDAL.Get(ctx, tt.args.customerRef.ID)
			if err != nil {
				t.Errorf("Get error = %v", err)
				return
			}

			if !tt.wantErr && len(datahubMetric.Metrics) != tt.expectedLen {
				t.Errorf("UpdateMetadataDocs.len actualLen = %d, wantLen %d", len(datahubMetric.Metrics), tt.expectedLen)
				return
			}
		})
	}
}

func TestDataHubMetricFirestore_Get(t *testing.T) {
	ctx := context.Background()

	datahubMetricFirestore, err := NewDataHubMetricFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	if err := testPackage.LoadTestData("DataHubMetrics"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "get datahub api metrics for existing customer",
			args: args{
				ctx:        ctx,
				customerID: "11111",
			},
			wantErr: false,
		},
		{
			name: "fail on trying to get non-existing customer api metrics",
			args: args{
				ctx:        ctx,
				customerID: "non-existing-id",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := datahubMetricFirestore.Get(
				tt.args.ctx,
				tt.args.customerID,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("datahubMetricFirestore.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataHubMetricFirestore_Delete(t *testing.T) {
	ctx := context.Background()

	vMetricFirestore, err := NewDataHubMetricFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	if err := testPackage.LoadTestData("DataHubMetrics"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "delete datahub api metrics for existing customer",
			args: args{
				ctx:        ctx,
				customerID: "to_delete",
			},
			wantErr: false,
		},
		{
			name: "fail on trying to delete non-existing customer api metrics",
			args: args{
				ctx:        ctx,
				customerID: "non-existing-id",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := vMetricFirestore.Delete(
				tt.args.ctx,
				tt.args.customerID,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("datahubMetricFirestore.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				_, err := vMetricFirestore.Get(ctx, tt.args.customerID)
				assert.Equal(t, err, doitFirestore.ErrNotFound)
			}
		})
	}
}
