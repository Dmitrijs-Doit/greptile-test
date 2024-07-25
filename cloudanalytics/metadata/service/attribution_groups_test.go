package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	attributionGroupsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/mocks"
	ag "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/common"
	testTools "github.com/doitintl/hello/scheduled-tasks/common/test_tools"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
)

func TestMetadataService_AttributionGroupsMetadata(t *testing.T) {
	type fields struct {
		customersDAL         *customerMocks.Customers
		attributionGroupsDAL *attributionGroupsMocks.AttributionGroups
	}

	ctx := context.Background()

	customerID := "ImoC9XkrutBysJvyqlBm"
	email := "yoni@doit.com"

	customerRef := &firestore.DocumentRef{
		ID: customerID,
	}

	customer := &common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: customerRef,
		},
	}

	var attributionGroups []ag.AttributionGroup
	if err := testTools.ConvertJSONFileIntoStruct("testData", "attr_groups.json", &attributionGroups); err != nil {
		t.Fatalf("could not convert json test file into struct. error %s", err)
	}

	var attrGroupsMD []*domain.OrgMetadataModel
	if err := testTools.ConvertJSONFileIntoStruct("testData", "attr_groups_md.json", &attrGroupsMD); err != nil {
		t.Fatalf("could not convert json test file into struct. error %s", err)
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "successfully returns attribution groups metadata",
			on: func(f *fields) {
				f.customersDAL.On("GetCustomer", ctx, customerID).
					Return(customer, nil)
				f.customersDAL.On("GetCustomerOrPresentationModeCustomer", ctx, customerID).
					Return(customer, nil)
				f.attributionGroupsDAL.On("List", ctx, customerRef, email).
					Return(attributionGroups, nil)
			},
		},
		{
			name: "error retrieving attribution groups",
			on: func(f *fields) {
				f.customersDAL.On("GetCustomer", ctx, customerID).
					Return(customer, nil)
				f.customersDAL.On("GetCustomerOrPresentationModeCustomer", ctx, customerID).
					Return(customer, nil)
				f.attributionGroupsDAL.On("List", ctx, customerRef, email).
					Return(nil, fmt.Errorf("error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				customersDAL:         &customerMocks.Customers{},
				attributionGroupsDAL: &attributionGroupsMocks.AttributionGroups{},
			}

			s := &MetadataService{
				attributionGroupsDAL: tt.fields.attributionGroupsDAL,
				customerDal:          tt.fields.customersDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			res, err := s.AttributionGroupsMetadata(ctx, customerID, email)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("MetadataService.AttributionGroupsMetadata() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			assert.Equal(t, attrGroupsMD, res, fmt.Sprintf("MetadataService.AttributionGroupsMetadata() = %v, want %v", res, attrGroupsMD))
		})
	}
}

func TestMetadataService_convertToOrgMetadataModel(t *testing.T) {
	now := time.Now()
	nullFallback := "stuff"
	attrGroup := ag.AttributionGroup{
		ID:           "some_attr_group_id",
		Name:         "Chaim's Attribution Group",
		Type:         "custom",
		TimeModified: now,
		Attributions: []*firestore.DocumentRef{
			{
				ID: "attr_id_1",
			},
			{
				ID: "attr_id_2",
			},
			{
				ID: "attr_id_3",
			},
		},
		NullFallback: &nullFallback,
	}

	expectedAGMD := domain.OrgMetadataModel{
		Key:                 attrGroup.ID,
		Label:               attrGroup.Name,
		Timestamp:           attrGroup.TimeModified,
		Type:                metadata.MetadataFieldTypeAttributionGroup,
		Values:              []string{"attr_id_1", "attr_id_2", "attr_id_3"},
		ID:                  "attribution_group:some_attr_group_id",
		Plural:              "Chaim's Attribution Group values",
		NullFallback:        &nullFallback,
		DisableRegexpFilter: true,
		ObjectType:          domainAttributions.ObjectTypeCustom,
	}

	agMD := convertToOrgMetadataModel(&attrGroup)

	assert.Equal(t, expectedAGMD, *agMD, fmt.Sprintf("ConvertToReportOrgMetadataModel() = %v, want %v", agMD, expectedAGMD))
}

func TestMetadataService_filterAttributionGroupsByCloud(t *testing.T) {
	groups := []ag.AttributionGroup{
		{
			Name:  "Group 0",
			Cloud: nil,
			Type:  domainAttributions.ObjectTypePreset,
		},
		{
			Name:  "Group 1",
			Cloud: []string{common.Assets.GoogleCloud},
			Type:  domainAttributions.ObjectTypePreset,
		},
		{
			Name:  "Group 2",
			Cloud: []string{common.Assets.AmazonWebServices},
			Type:  domainAttributions.ObjectTypePreset,
		},
		{
			Name:  "Group 3",
			Cloud: []string{common.Assets.AmazonWebServices, common.Assets.AmazonWebServicesStandalone},
			Type:  domainAttributions.ObjectTypePreset,
		},
		{
			Name:  "Group 4",
			Cloud: []string{common.Assets.GoogleCloud, common.Assets.GoogleCloudStandalone},
			Type:  domainAttributions.ObjectTypePreset,
		},
		{
			Name:  "Group 5",
			Cloud: []string{common.Assets.AmazonWebServicesStandalone},
			Type:  domainAttributions.ObjectTypePreset,
		},
		{
			Name:  "Group 6",
			Cloud: []string{common.Assets.GoogleCloudStandalone},
			Type:  domainAttributions.ObjectTypePreset,
		},
		{
			Name: "Custom",
			Type: domainAttributions.ObjectTypeCustom,
		},
	}

	// Test the function with both Google Cloud and AWS assets
	assets := []string{common.Assets.GoogleCloud, common.Assets.AmazonWebServices}
	filteredGroups := filterAttributionGroupsByCloud(groups, assets)
	assert.Len(t, filteredGroups, 6)
	assert.Equal(t, "Group 0", filteredGroups[0].Name)
	assert.Equal(t, "Group 1", filteredGroups[1].Name)
	assert.Equal(t, "Group 2", filteredGroups[2].Name)
	assert.Equal(t, "Group 3", filteredGroups[3].Name)
	assert.Equal(t, "Group 4", filteredGroups[4].Name)
	assert.Equal(t, "Custom", filteredGroups[5].Name)

	// Test the function with only Google Cloud assets
	assets = []string{common.Assets.GoogleCloud}
	filteredGroups = filterAttributionGroupsByCloud(groups, assets)
	assert.Len(t, filteredGroups, 4)
	assert.Equal(t, "Group 0", filteredGroups[0].Name)
	assert.Equal(t, "Group 1", filteredGroups[1].Name)
	assert.Equal(t, "Group 4", filteredGroups[2].Name)
	assert.Equal(t, "Custom", filteredGroups[3].Name)

	// Test the function with only AWS assets
	assets = []string{common.Assets.AmazonWebServices}
	filteredGroups = filterAttributionGroupsByCloud(groups, assets)
	assert.Len(t, filteredGroups, 4)
	assert.Equal(t, "Group 0", filteredGroups[0].Name)
	assert.Equal(t, "Group 2", filteredGroups[1].Name)
	assert.Equal(t, "Group 3", filteredGroups[2].Name)
	assert.Equal(t, "Custom", filteredGroups[3].Name)

	// Test the function with mixed asset types
	assets = []string{common.Assets.AmazonWebServices, common.Assets.GoogleCloudStandalone}
	filteredGroups = filterAttributionGroupsByCloud(groups, assets)
	assert.Len(t, filteredGroups, 6)
	assert.Equal(t, "Group 0", filteredGroups[0].Name)
	assert.Equal(t, "Group 2", filteredGroups[1].Name)
	assert.Equal(t, "Group 3", filteredGroups[2].Name)
	assert.Equal(t, "Group 4", filteredGroups[3].Name)
	assert.Equal(t, "Group 6", filteredGroups[4].Name)
	assert.Equal(t, "Custom", filteredGroups[5].Name)

	// Test the function with mixed asset types
	assets = []string{common.Assets.GoogleCloud, common.Assets.AmazonWebServicesStandalone}
	filteredGroups = filterAttributionGroupsByCloud(groups, assets)
	assert.Len(t, filteredGroups, 6)
	assert.Equal(t, "Group 0", filteredGroups[0].Name)
	assert.Equal(t, "Group 1", filteredGroups[1].Name)
	assert.Equal(t, "Group 3", filteredGroups[2].Name)
	assert.Equal(t, "Group 4", filteredGroups[3].Name)
	assert.Equal(t, "Group 5", filteredGroups[4].Name)
	assert.Equal(t, "Custom", filteredGroups[5].Name)

	// Test the function with GCP standalone
	assets = []string{common.Assets.GoogleCloudStandalone}
	filteredGroups = filterAttributionGroupsByCloud(groups, assets)
	assert.Len(t, filteredGroups, 4)
	assert.Equal(t, "Group 0", filteredGroups[0].Name)
	assert.Equal(t, "Group 4", filteredGroups[1].Name)
	assert.Equal(t, "Group 6", filteredGroups[2].Name)
	assert.Equal(t, "Custom", filteredGroups[3].Name)

	// Test the function with AWS standalone
	assets = []string{common.Assets.AmazonWebServicesStandalone}
	filteredGroups = filterAttributionGroupsByCloud(groups, assets)
	assert.Len(t, filteredGroups, 4)
	assert.Equal(t, "Group 0", filteredGroups[0].Name)
	assert.Equal(t, "Group 3", filteredGroups[1].Name)
	assert.Equal(t, "Group 5", filteredGroups[2].Name)
	assert.Equal(t, "Custom", filteredGroups[3].Name)

	// Test the function with other assets
	assets = []string{common.Assets.MicrosoftAzure}
	filteredGroups = filterAttributionGroupsByCloud(groups, assets)
	assert.Len(t, filteredGroups, 2)
	assert.Equal(t, "Group 0", filteredGroups[0].Name)
	assert.Equal(t, "Custom", filteredGroups[1].Name)

	// Test the function with empty assets
	assets = []string{}
	filteredGroups = filterAttributionGroupsByCloud(groups, assets)
	assert.Len(t, filteredGroups, len(groups))
}
