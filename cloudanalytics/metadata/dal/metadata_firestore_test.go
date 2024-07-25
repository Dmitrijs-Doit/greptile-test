package dal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	iface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/common"
	testPackage "github.com/doitintl/tests"
)

var ctx = context.Background()
var testsCustomOrgID = "customOrgId"
var testsCustomerID = "JhV7WydpOlW8DeVRVVNf"

func TestNewFirestoreMetadataDAL(t *testing.T) {
	_, err := NewMetadataFirestore(ctx, common.TestProjectID)
	assert.NoError(t, err)

	d := NewMetadataFirestoreWithClient(nil)
	assert.NotNil(t, d)
}

func preTestMetadata(t *testing.T) (*MetadataFirestore, error) {
	d, err := NewMetadataFirestore(ctx, common.TestProjectID)
	if err != nil {
		return nil, err
	}

	if err := testPackage.LoadTestData("analytics-metadata"); err != nil {
		return nil, err
	}

	return d, nil
}

func TestMetadataFirestore_List(t *testing.T) {
	d, err := preTestMetadata(t)
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name      string
		args      iface.ListArgs
		want      []iface.ListItem
		wantErr   bool
		condition func(got []iface.ListItem, want []iface.ListItem) bool
	}{
		{
			name: "list metadata",
			args: iface.ListArgs{
				Ctx:         ctx,
				CustomerRef: d.GetCustomerRef(ctx, testsCustomerID),
				OrgRef:      d.GetCustomerOrgRef(ctx, testsCustomerID, testsCustomOrgID),
				TypesFilter: []metadata.MetadataFieldType{metadata.MetadataFieldTypeFixed, metadata.MetadataFieldTypeAttribution, metadata.MetadataFieldTypeOptional, metadata.MetadataFieldTypeGKE},
			},
			want: []iface.ListItem{
				{Key: "attribution", Label: "Attribution", Type: metadata.MetadataFieldTypeAttribution},
			},
			wantErr: false,
			condition: func(got []iface.ListItem, want []iface.ListItem) bool {
				return len(got) == 1 &&
					got[0].Key == want[0].Key &&
					got[0].Label == want[0].Label &&
					got[0].Type == want[0].Type
			},
		},
		{
			name: "list metadata with empty types",
			args: iface.ListArgs{
				Ctx:         ctx,
				CustomerRef: d.GetCustomerRef(ctx, testsCustomerID),
				OrgRef:      d.GetCustomerOrgRef(ctx, testsCustomerID, testsCustomOrgID),
				TypesFilter: []metadata.MetadataFieldType{},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.List(tt.args)
			if err != nil && tt.wantErr == false {
				t.Errorf("MetadataFirestore.List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.condition != nil {
				assert.True(t, tt.condition(got, tt.want))
			}
		})
	}
}

func TestMetadataFirestore_Get(t *testing.T) {
	d, err := preTestMetadata(t)
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name      string
		args      iface.GetArgs
		want      []iface.GetItem
		wantErr   bool
		condition func(got []iface.GetItem, want []iface.GetItem) bool
	}{
		{
			name: "get dimensions",
			args: iface.GetArgs{
				Ctx:         ctx,
				CustomerRef: d.GetCustomerRef(ctx, testsCustomerID),
				OrgRef:      d.GetPresetOrgRef(ctx, metadata.DoitOrgID),
				KeyFilter:   "attribution",
				TypeFilter:  string(metadata.MetadataFieldTypeAttribution),
			},
			want: []iface.GetItem{
				{Key: "attribution", Label: "Attribution", Type: metadata.MetadataFieldTypeAttribution, Values: []string{}, Cloud: "google-cloud"},
			},
			wantErr: false,
			condition: func(got []iface.GetItem, want []iface.GetItem) bool {
				return len(got) == 1 &&
					got[0].Key == want[0].Key &&
					got[0].Label == want[0].Label &&
					got[0].Type == want[0].Type &&
					got[0].Cloud == want[0].Cloud
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.Get(tt.args)
			if err != nil && tt.wantErr == false {
				t.Errorf("MetadataFirestore.List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.condition != nil {
				assert.True(t, tt.condition(got, tt.want))
			}
		})
	}
}

func TestMetadataFirestore_ListMap(t *testing.T) {
	d, err := preTestMetadata(t)
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name      string
		args      iface.ListArgs
		want      map[metadata.MetadataFieldType][]iface.ListItem
		wantErr   bool
		condition func(got map[metadata.MetadataFieldType][]iface.ListItem, want map[metadata.MetadataFieldType][]iface.ListItem) bool
	}{
		{
			name: "listmap metadata",
			args: iface.ListArgs{
				Ctx:         ctx,
				CustomerRef: d.GetCustomerRef(ctx, testsCustomerID),
				OrgRef:      d.GetCustomerOrgRef(ctx, testsCustomerID, testsCustomOrgID),
				TypesFilter: []metadata.MetadataFieldType{metadata.MetadataFieldTypeFixed, metadata.MetadataFieldTypeAttribution, metadata.MetadataFieldTypeOptional, metadata.MetadataFieldTypeGKE},
			},
			want: map[metadata.MetadataFieldType][]iface.ListItem{
				metadata.MetadataFieldTypeAttribution: {
					{Key: "attribution", Label: "Attribution", Type: metadata.MetadataFieldTypeAttribution},
				},
			},
			wantErr: false,
			condition: func(got map[metadata.MetadataFieldType][]iface.ListItem, want map[metadata.MetadataFieldType][]iface.ListItem) bool {
				if len(got[metadata.MetadataFieldTypeAttribution]) != 1 {
					return false
				}
				gotElem := got[metadata.MetadataFieldTypeAttribution][0]
				wantElem := want[metadata.MetadataFieldTypeAttribution][0]
				return gotElem.Key == wantElem.Key &&
					gotElem.Label == wantElem.Label &&
					gotElem.Type == wantElem.Type
			},
		},
		{
			name: "listmap metadata with empty types",
			args: iface.ListArgs{
				Ctx:         ctx,
				CustomerRef: d.GetCustomerRef(ctx, testsCustomerID),
				OrgRef:      d.GetCustomerOrgRef(ctx, testsCustomerID, testsCustomOrgID),
				TypesFilter: []metadata.MetadataFieldType{},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.ListMap(tt.args)
			if err != nil && tt.wantErr == false {
				t.Errorf("MetadataFirestore.List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.condition != nil {
				assert.True(t, tt.condition(got, tt.want))
			}
		})
	}
}
