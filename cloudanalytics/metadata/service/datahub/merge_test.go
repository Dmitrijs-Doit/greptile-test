package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	metadataFirestoreMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/mocks"
	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	metadataMetadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
)

func TestMergeMergeMetadataDoc(t *testing.T) {
	ctx := context.Background()
	testCustomerID := "JhV7WydpTlW8DfVRVVMg"
	testCustomerRef := &firestore.DocumentRef{
		Path: "projects/doitintl-cmp-dev/databases/(default)/documents/customers/" + testCustomerID,
		ID:   testCustomerID,
	}

	testCustomLabelDocID := "label:dGVzdEN1c3RvbUxhYmVs"

	testDocID := "fixed:project_id"
	testCustomerMetadataDocRef := &firestore.DocumentRef{
		Path: "projects/doitintl-cmp-dev/databases/(default)/some/path" + testCustomerID,
		ID:   testDocID,
	}

	testOrgID := "root"

	testCustomerOrgRef := &firestore.DocumentRef{
		Path: "projects/doitintl-cmp-dev/databases/(default)/some/path" + testOrgID,
		ID:   testOrgID,
	}

	var txNil *firestore.Transaction

	type fields struct {
		datahubMetadataFirestore *metadataFirestoreMocks.DataHubMetadataFirestore
	}

	type args struct {
		md          metadataMetadataDomain.MetadataField
		customerRef *firestore.DocumentRef
		key         string
		values      []string
	}

	tests := []struct {
		name          string
		fields        fields
		args          args
		on            func(*fields)
		wantDocID     string
		wantTargetMap map[string]interface{}
		wantErr       bool
	}{
		{
			name: "invalid metadata type",
			args: args{
				md: metadataMetadataDomain.MetadataField{},
			},
			wantErr: true,
		},
		{
			name: "GetMergeableDocument fails",
			args: args{
				md:          queryDomain.KeyMap["project_id"],
				customerRef: testCustomerRef,
				key:         "project_id",
				values:      []string{"test-project-id-4", "test-project-id-5"},
			},
			on: func(f *fields) {
				f.datahubMetadataFirestore.
					On("GetCustomerOrgRef",
						ctx,
						testCustomerID,
					).Return(testCustomerOrgRef).Once()
				f.datahubMetadataFirestore.
					On("GetCustomerMetadataDocRef",
						ctx,
						testCustomerID,
						testDocID,
					).Return(testCustomerMetadataDocRef).Once()
				f.datahubMetadataFirestore.
					On("GetMergeableDocument",
						txNil,
						testCustomerMetadataDocRef,
					).Return(nil, errors.New("error 1138")).Once()
			},
			wantErr: true,
		},
		{
			name: "merge values into empty document",
			args: args{
				md:          queryDomain.KeyMap["labels"],
				customerRef: testCustomerRef,
				key:         "testCustomLabel",
				values:      []string{"test-label-2", "test-label-1"},
			},
			on: func(f *fields) {
				f.datahubMetadataFirestore.
					On("GetCustomerOrgRef",
						ctx,
						testCustomerID,
					).Return(testCustomerOrgRef).Once()
				f.datahubMetadataFirestore.
					On("GetCustomerMetadataDocRef",
						ctx,
						testCustomerID,
						testCustomLabelDocID,
					).Return(testCustomerMetadataDocRef).Once()
				f.datahubMetadataFirestore.
					On("GetMergeableDocument",
						txNil,
						testCustomerMetadataDocRef,
					).Return(&metadataDomain.OrgMetadataModel{}, nil).Once()
			},
			wantDocID: testCustomLabelDocID,
			wantTargetMap: map[string]interface{}{
				"order":  1000,
				"key":    "testCustomLabel",
				"values": []string{"test-label-1", "test-label-2"},
			},
		},
		{
			name: "merge values into existing document",
			args: args{
				md:          queryDomain.KeyMap["project_id"],
				customerRef: testCustomerRef,
				key:         "project_id",
				values:      []string{"test-project-id-2", "test-project-id-1"},
			},
			on: func(f *fields) {
				f.datahubMetadataFirestore.
					On("GetCustomerOrgRef",
						ctx,
						testCustomerID,
					).Return(testCustomerOrgRef).Once()
				f.datahubMetadataFirestore.
					On("GetCustomerMetadataDocRef",
						ctx,
						testCustomerID,
						testDocID,
					).Return(testCustomerMetadataDocRef).Once()
				f.datahubMetadataFirestore.
					On("GetMergeableDocument",
						txNil,
						testCustomerMetadataDocRef,
					).Return(&metadataDomain.OrgMetadataModel{
					Values: []string{"test-project-id-2", "test-project-id-3"},
				}, nil).Once()
			},
			wantDocID: testDocID,
			wantTargetMap: map[string]interface{}{
				"order":  12,
				"key":    "project_id",
				"values": []string{"test-project-id-1", "test-project-id-2", "test-project-id-3"},
			},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()

		tt.fields = fields{
			datahubMetadataFirestore: metadataFirestoreMocks.NewDataHubMetadataFirestore(t),
		}

		s := &DataHubMetadata{
			datahubMetadataFirestore: tt.fields.datahubMetadataFirestore,
		}

		if tt.on != nil {
			tt.on(&tt.fields)
		}

		t.Run(tt.name, func(t *testing.T) {
			gotDocID, gotTargetMap, err := s.MergeMetadataDoc(ctx, txNil, tt.args.md, tt.args.customerRef, tt.args.key, tt.args.values)

			if (err != nil) != tt.wantErr {
				t.Errorf("TestMergeMergeMetadataDoc() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantDocID, gotDocID)

			validateMap(t, gotTargetMap, tt.wantTargetMap)
		})
	}
}

func validateMap(t *testing.T, got map[string]interface{}, want map[string]interface{}) {
	keys := []string{"order", "key", "values"}

	for _, key := range keys {
		assert.Equal(t, want[key], got[key])
	}
}
