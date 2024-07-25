package dal

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	doitFirestore "github.com/doitintl/firestore"
	datahubMetricDalMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/datahubmetric/mocks"
	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub"
	metadataMetadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	service "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/datahub"
	"github.com/doitintl/hello/scheduled-tasks/common"
	testPackage "github.com/doitintl/tests"
)

func preTestDataHubMetadata(ctx context.Context) (*DataHubMetadataFirestore, error) {
	d, err := NewDataHubMetadataFirestore(ctx, common.TestProjectID)
	if err != nil {
		return nil, err
	}

	if err := testPackage.LoadTestData("DataHubMetadata"); err != nil {
		return nil, err
	}

	return d, nil
}

func TestUpdateMetadataDocs(t *testing.T) {
	ctx := context.Background()

	testsCustomerID := "JhV7WydpTlW8DfVRVVMg"
	testCustomerRef := &firestore.DocumentRef{
		Path: "projects/doitintl-cmp-dev/databases/(default)/documents/customers/" + testsCustomerID,
		ID:   testsCustomerID,
	}

	d, err := preTestDataHubMetadata(ctx)
	if err != nil {
		t.Error(err)
	}

	datahubMetricDal := datahubMetricDalMocks.DataHubMetricFirestore{}

	s := service.NewDataHubMetadataService(nil, d, &datahubMetricDal, nil, nil)

	type args struct {
		customerRef           *firestore.DocumentRef
		updatefunc            domain.UpdateMetadataDocsPostTxFunc
		mergeMetadataDoc      domain.MergeMetadataDocFunc
		metadataByCustomer    domain.MetadataByCustomer
		metricTypesByCustomer domain.MetricTypesByCustomer
		mergeMetricTypesDoc   domain.MergeDataHubMetricTypesDocFunc
	}

	tests := []struct {
		name      string
		args      args
		wantDocID []string
		want      []*metadataDomain.OrgMetadataModel
		wantErr   bool
	}{
		{
			name: "update metadata. mergeMetadataDoc fails",
			args: args{
				customerRef: testCustomerRef,
				metadataByCustomer: domain.MetadataByCustomer{
					testCustomerRef: {
						string(metadataMetadataDomain.MetadataFieldKeyProjectID): {
							string(metadataMetadataDomain.MetadataFieldKeyProjectID): {
								"test-project-35",
							},
						},
					},
				},
				mergeMetadataDoc: func(
					ctx context.Context,
					tx *firestore.Transaction,
					mdField metadataMetadataDomain.MetadataField,
					customerRef *firestore.DocumentRef,
					key string,
					values []string,
				) (string, map[string]interface{}, error) {
					return "", nil, fmt.Errorf("error 1138")
				},
			},
			wantErr: true,
		},
		{
			name: "update metadata. updatefunc fails",
			args: args{
				customerRef: testCustomerRef,
				metadataByCustomer: domain.MetadataByCustomer{
					testCustomerRef: {
						string(metadataMetadataDomain.MetadataFieldKeyProjectID): {
							string(metadataMetadataDomain.MetadataFieldKeyProjectID): {
								"test-project-35",
							},
						},
					},
				},
				mergeMetadataDoc: s.MergeMetadataDoc,
				updatefunc:       func() error { return fmt.Errorf("error 1138") },
			},
			wantErr: true,
		},
		{
			name: "update metadata. One file exists, two don't",
			args: args{
				customerRef: testCustomerRef,
				metadataByCustomer: domain.MetadataByCustomer{
					testCustomerRef: {
						string(metadataMetadataDomain.MetadataFieldKeyProjectID): {
							string(metadataMetadataDomain.MetadataFieldKeyProjectID): {
								"test-project-9",
								"test-project-0",
								"test-project-8",
								"test-project-2",
							},
						},
						string(metadataMetadataDomain.MetadataFieldKeyProjectName): {
							string(metadataMetadataDomain.MetadataFieldKeyProjectName): {
								"test-project-name-3",
								"test-project-name-0",
								"test-project-name-1",
							},
						},
						string(metadataMetadataDomain.MetadataFieldTypeLabel): {
							"testCustomLabel": {
								"test-custom-lavel-value-3",
								"test-custom-lavel-value-0",
								"test-custom-lavel-value-1",
							},
						},
					},
				},
				mergeMetadataDoc: s.MergeMetadataDoc,
				updatefunc:       func() error { return nil },
			},
			wantDocID: []string{
				string(metadataMetadataDomain.MetadataFieldTypeFixed) + ":" + string(metadataMetadataDomain.MetadataFieldKeyProjectID),
				string(metadataMetadataDomain.MetadataFieldTypeFixed) + ":" + string(metadataMetadataDomain.MetadataFieldKeyProjectName),
				string(metadataMetadataDomain.MetadataFieldTypeLabel) + ":" + "dGVzdEN1c3RvbUxhYmVs",
			},
			want: []*metadataDomain.OrgMetadataModel{
				{
					Order:  12,
					Field:  "T.project_id",
					Label:  "Project/Account ID",
					Plural: "Project/Account ids",
					Type:   metadataMetadataDomain.MetadataFieldTypeFixed,
					Key:    "project_id",
					Values: []string{"test-project-0", "test-project-1", "test-project-2", "test-project-8", "test-project-9"},
				},
				{
					Order:        14,
					Field:        "T.project_name",
					Label:        "Project/Account name",
					Plural:       "Project/Account names",
					NullFallback: common.String("[Project/Account name N/A]"),
					Type:         metadataMetadataDomain.MetadataFieldTypeFixed,
					Key:          "project_name",
					Values:       []string{"test-project-name-0", "test-project-name-1", "test-project-name-3"},
				},
				{
					Order:        1000,
					Field:        "T.labels",
					Label:        "testCustomLabel",
					Plural:       "label values",
					NullFallback: common.String("[Label N/A]"),
					Type:         metadataMetadataDomain.MetadataFieldTypeLabel,
					Key:          "testCustomLabel",
					Values:       []string{"test-custom-lavel-value-0", "test-custom-lavel-value-1", "test-custom-lavel-value-3"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := d.UpdateMetadataDocs(
				ctx,
				tt.args.mergeMetadataDoc,
				tt.args.updatefunc,
				tt.args.metadataByCustomer,
				tt.args.metricTypesByCustomer,
				tt.args.mergeMetricTypesDoc,
			)
			if err != nil && tt.wantErr == false {
				t.Errorf("UpdateMetadataDocs error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for i := range tt.wantDocID {
				got, err := d.Get(ctx, tt.args.customerRef.ID, tt.wantDocID[i])
				if err != nil {
					t.Error(err)
				}

				validateFields(t, got, tt.want[i])
				assert.Equal(t, got.Values, tt.want[i].Values)
			}
		})
	}
}

func TestDataHubMetadataFirestore_DeleteCustomerMetadata(t *testing.T) {
	ctx := context.Background()

	datahubMetadataFirestore, err := NewDataHubMetadataFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	if err := testPackage.LoadTestData("DataHubMetadata"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name             string
		args             args
		wantErr          bool
		expectedErr      error
		documentIdToTest string
	}{
		{
			name: "delete datahub api metadata for existing customer",
			args: args{
				ctx:        ctx,
				customerID: "to_delete",
			},
			wantErr:          false,
			documentIdToTest: "label:dGVzdEN1c3RvbUxhYmVs",
		},
		{
			name: "no fail on trying to delete non-existing customer api metadata",
			args: args{
				ctx:        ctx,
				customerID: "non-existing-id",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := datahubMetadataFirestore.DeleteCustomerMetadata(
				tt.args.ctx,
				tt.args.customerID,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("datahubMetadataFirestore.DeleteCustomerMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && tt.documentIdToTest != "" {
				_, err := datahubMetadataFirestore.Get(ctx, tt.args.customerID, tt.documentIdToTest)
				assert.Equal(t, err, doitFirestore.ErrNotFound)
			}
		})
	}
}

func validateFields(t *testing.T, got *metadataDomain.OrgMetadataModel, want *metadataDomain.OrgMetadataModel) {
	errorMessageTpl := "invalid field %s. wanted %v, got %v"

	if got.Order != want.Order {
		t.Errorf(errorMessageTpl, "order", want.Order, got.Order)
	}

	if got.Field != want.Field {
		t.Errorf(errorMessageTpl, "field", want.Field, got.Field)
	}

	if got.Label != want.Label {
		t.Errorf(errorMessageTpl, "label", want.Label, got.Label)
	}

	if got.Plural != want.Plural {
		t.Errorf(errorMessageTpl, "plural", want.Plural, got.Plural)
	}

	if got.Type != want.Type {
		t.Errorf(errorMessageTpl, "type", want.Type, got.Type)
	}

	if got.DisableRegexpFilter != want.DisableRegexpFilter {
		t.Errorf(errorMessageTpl, "disableRegexpFilter", want.Customer, got.Customer)
	}

	if got.Key != want.Key {
		t.Errorf(errorMessageTpl, "key", want.Key, got.Key)
	}
}
