package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	metadataFirestoreMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/mocks"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub"
	eventpb "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub/proto"
	customerDALMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
)

func TestEventsMetadataWorker(t *testing.T) {
	ctx := context.Background()
	testCustomerID := "JhV7WydpTlW8DfVRVVMg"
	testCustomerRef := &firestore.DocumentRef{
		Path: "projects/doitintl-cmp-dev/databases/(default)/documents/customers/" + testCustomerID,
		ID:   testCustomerID,
	}

	updateMetadataDocsErr := errors.New("error 1138")

	type fields struct {
		datahubMetadataFirestore *metadataFirestoreMocks.DataHubMetadataFirestore
		datahubMetadataGCS       *metadataFirestoreMocks.DataHubMetadataGCS
		customerDAL              *customerDALMocks.Customers
	}

	tests := []struct {
		name       string
		fields     fields
		workParams *eventsMetadataWorkUnit
		on         func(*fields)
		wantedErr  error
	}{
		{
			name: "UpdateMetadataDocs fails",
			workParams: &eventsMetadataWorkUnit{
				events: []*eventpb.Event{
					{
						ProjectId:  makeStringPtr("test-project-1"),
						CustomerId: testCustomerID,
						Cloud:      "datahub",
						Metrics: []*eventpb.Event_Metric{
							{
								Type:  "rides",
								Value: 102,
							},
						},
					},
				},
			},
			on: func(f *fields) {
				f.customerDAL.
					On("GetRef",
						ctx,
						testCustomerID,
					).Return(testCustomerRef).Once()
				f.datahubMetadataFirestore.
					On("UpdateMetadataDocs",
						ctx,
						mock.AnythingOfType("domain.MergeMetadataDocFunc"),
						mock.AnythingOfType("domain.UpdateMetadataDocsPostTxFunc"),
						domain.MetadataByCustomer{
							testCustomerRef: {
								"project_id":     map[string][]string{"project_id": {"test-project-1"}},
								"cloud_provider": map[string][]string{"cloud_provider": {"datahub"}}},
						},
						domain.MetricTypesByCustomer{
							testCustomerRef: {
								"rides": true,
							},
						},
						mock.AnythingOfType("domain.MergeDataHubMetricTypesDocFunc"),
					).Return(updateMetadataDocsErr).Once()
			},
			wantedErr: updateMetadataDocsErr,
		},
		{
			name: "happy path",
			workParams: &eventsMetadataWorkUnit{
				events: []*eventpb.Event{
					{
						BillingAccountId: makeStringPtr("test-ba-1"),
						ProjectId:        makeStringPtr("test-project-2"),
						Cloud:            "Datadog",
						CustomerId:       testCustomerID,
						Metrics: []*eventpb.Event_Metric{
							{
								Type:  "rides",
								Value: 102,
							},
						},
					},
				},
			},
			on: func(f *fields) {
				f.customerDAL.
					On("GetRef",
						ctx,
						testCustomerID,
					).Return(testCustomerRef).Once()
				f.datahubMetadataFirestore.
					On("UpdateMetadataDocs",
						ctx,
						mock.AnythingOfType("domain.MergeMetadataDocFunc"),
						mock.AnythingOfType("domain.UpdateMetadataDocsPostTxFunc"),
						domain.MetadataByCustomer{
							testCustomerRef: {
								"billing_account_id": map[string][]string{"billing_account_id": {"test-ba-1"}},
								"project_id":         map[string][]string{"project_id": {"test-project-2"}},
								"cloud_provider":     map[string][]string{"cloud_provider": {"Datadog"}}},
						},
						domain.MetricTypesByCustomer{
							testCustomerRef: {
								"rides": true,
							},
						},
						mock.AnythingOfType("domain.MergeDataHubMetricTypesDocFunc"),
					).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()

		tt.fields = fields{
			datahubMetadataFirestore: metadataFirestoreMocks.NewDataHubMetadataFirestore(t),
			datahubMetadataGCS:       metadataFirestoreMocks.NewDataHubMetadataGCS(t),
			customerDAL:              customerDALMocks.NewCustomers(t),
		}

		s := &DataHubMetadata{
			datahubMetadataFirestore: tt.fields.datahubMetadataFirestore,
			datahubMetadataGCS:       tt.fields.datahubMetadataGCS,
			customerDAL:              tt.fields.customerDAL,
		}

		if tt.on != nil {
			tt.on(&tt.fields)
		}

		t.Run(tt.name, func(t *testing.T) {
			err := s.eventsMetadataWorker(ctx, tt.workParams)
			assert.Equal(t, tt.wantedErr, err)
		})
	}
}
