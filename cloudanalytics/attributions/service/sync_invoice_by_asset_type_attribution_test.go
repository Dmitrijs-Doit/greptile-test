package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	assetsDalMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func TestAttributionsService_CreateAttributionsForInvoiceAssetTypes(t *testing.T) {
	type fields struct {
		attributionsDal *mocks.Attributions
		assetsDal       *assetsDalMocks.Assets
	}

	type args struct {
		ctx context.Context
		req SyncInvoiceByAssetTypeAttributionRequest
	}

	ctx := context.Background()
	publicAccessView := collab.PublicAccessView
	req := SyncInvoiceByAssetTypeAttributionRequest{
		AttributionGroup: &attributiongroups.AttributionGroup{
			Attributions: []*firestore.DocumentRef{
				{},
			},
		},
		Entity: &common.Entity{
			Name:       "Entity",
			PriorityID: "123",
			Snapshot:   &firestore.DocumentSnapshot{Ref: &firestore.DocumentRef{ID: "entity-ID"}},
		},
		Customer: &common.Customer{Snapshot: &firestore.DocumentSnapshot{Ref: &firestore.DocumentRef{ID: "customer-id"}}},
	}

	expectedAttributions := []*attribution.Attribution{
		{Name: "attribution 1"},
		{Name: "attribution 2"},
		{Name: "[123] Entity - Google Cloud", Ref: &firestore.DocumentRef{ID: "gcp-attribution"}},
		{Name: "[123] Entity - Amazon Web Services", Ref: &firestore.DocumentRef{ID: "aws-attribution"}},
		{Name: "attribution 3"},
		{Name: "Entity1 - Google Cloud"},
		{Name: "attribution 4"},
		{Name: "Entity1 - Amazon Web Services"},
	}

	getAssetsResult := []*pkg.BaseAsset{
		{AssetType: common.Assets.GoogleCloud, ID: "asset-1"},
		{AssetType: common.Assets.GoogleCloudDirect, ID: "asset-2"},
		{AssetType: common.Assets.GoogleCloudProject, ID: "asset-3"},
		{AssetType: common.Assets.GoogleCloudReseller, ID: "asset-4"},
		{AssetType: common.Assets.GoogleCloudStandalone, ID: "asset-5"},
		{AssetType: common.Assets.AmazonWebServices, ID: "asset-6"},
		{AssetType: common.Assets.AmazonWebServicesReseller, ID: "asset-7"},
		{AssetType: common.Assets.AmazonWebServicesStandalone, ID: "asset-8"},
		{AssetType: common.Assets.BetterCloud, ID: "asset-9"},
		{AssetType: common.Assets.GSuite, ID: "asset-10"},
		{AssetType: common.Assets.MicrosoftAzure, ID: "asset-11"},
		{AssetType: common.Assets.Office365, ID: "asset-12"},
		{AssetType: common.Assets.Zendesk, ID: "asset-13"},
	}

	gcpFilters := []report.BaseConfigFilter{
		{

			ID:     "fixed:project_id",
			Key:    metadata.MetadataFieldKeyProjectID,
			Type:   metadata.MetadataFieldTypeFixed,
			Values: &[]string{"asset-3"},
			Field:  "T.project_id",
		},
		{
			ID:     "fixed:billing_account_id",
			Key:    metadata.MetadataFieldKeyBillingAccountID,
			Type:   metadata.MetadataFieldTypeFixed,
			Values: &[]string{"asset-1"},
			Field:  "T.billing_account_id",
		},
		{

			ID:        "fixed:project_id",
			Key:       metadata.MetadataFieldKeyProjectID,
			Type:      metadata.MetadataFieldTypeFixed,
			AllowNull: true,
			Field:     "T.project_id",
		},
		{
			Key:     metadata.MetadataFieldKeyServiceDescription,
			Type:    metadata.MetadataFieldTypeFixed,
			Values:  &[]string{"Looker"},
			ID:      "fixed:service_description",
			Field:   "T.service_description",
			Inverse: true,
		},
	}

	awsFilters := []report.BaseConfigFilter{
		{
			ID:     "fixed:project_id",
			Key:    metadata.MetadataFieldKeyProjectID,
			Type:   metadata.MetadataFieldTypeFixed,
			Values: &[]string{"asset-6"},
			Field:  "T.project_id",
		},
	}

	tests := []struct {
		name           string
		args           args
		wantErr        bool
		expectedErr    error
		fields         fields
		on             func(*fields)
		expectedResult []*firestore.DocumentRef
	}{
		{
			name: "Successfully create attribution for invoice by asset type with existing attributions",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.attributionsDal.On("GetAttributions", ctx, req.AttributionGroup.Attributions).Return(expectedAttributions, nil)
				f.assetsDal.On("GetAssetsInEntity", ctx, req.Entity.Snapshot.Ref).Return(getAssetsResult, nil)
				f.attributionsDal.On("UpdateAttribution", ctx, "gcp-attribution", []firestore.Update{
					{Path: "filters", Value: gcpFilters},
					{Path: "formula", Value: "A OR (B AND C AND D)"},
					{Path: "type", Value: "managed"},
					{Path: "classification", Value: "invoice"},
					{Path: "hidden", Value: true},
					{Path: "collaborators", Value: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}}},
					{Path: "public", Value: &publicAccessView},
					{Path: "cloud", Value: []string{"google-cloud"}},
				}).Return(nil)
				f.attributionsDal.On("UpdateAttribution", ctx, "aws-attribution", []firestore.Update{
					{Path: "filters", Value: awsFilters},
					{Path: "formula", Value: "A"},
					{Path: "type", Value: "managed"},
					{Path: "classification", Value: "invoice"},
					{Path: "hidden", Value: true},
					{Path: "collaborators", Value: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}}},
					{Path: "public", Value: &publicAccessView},
					{Path: "cloud", Value: []string{"amazon-web-services"}},
				}).Return(nil)
				f.assetsDal.On("GetAWSAsset", ctx, "asset-6").Return(&pkg.AWSAsset{}, nil)

			},
			expectedResult: []*firestore.DocumentRef{
				{ID: "gcp-attribution"},
				{ID: "aws-attribution"},
			},
		},
		{
			name: "Successfully create attribution for invoice by asset type no existing attributions",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.attributionsDal.On("GetAttributions", ctx, mock.AnythingOfType("[]*firestore.DocumentRef")).Return([]*attribution.Attribution{}, nil)
				f.attributionsDal.On("CreateAttribution", ctx, &attribution.Attribution{
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
					Access: collab.Access{
						Collaborators: []collab.Collaborator{
							{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
						},
						Public: &publicAccessView,
					},
					Customer: req.Customer.Snapshot.Ref,
					Name:     "[123] Entity - Google Cloud",
					Cloud:    []string{"google-cloud"},
				}).Return(&attribution.Attribution{Ref: &firestore.DocumentRef{ID: "gcp-attribution"}}, nil)
				f.attributionsDal.On("CreateAttribution", ctx, &attribution.Attribution{
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
					Access: collab.Access{
						Collaborators: []collab.Collaborator{
							{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
						},
						Public: &publicAccessView,
					},
					Customer: req.Customer.Snapshot.Ref,
					Name:     "[123] Entity - Amazon Web Services",
					Cloud:    []string{"amazon-web-services"},
				}).Return(&attribution.Attribution{Ref: &firestore.DocumentRef{ID: "aws-attribution"}}, nil)
				f.assetsDal.On("GetAssetsInEntity", ctx, req.Entity.Snapshot.Ref).Return(getAssetsResult, nil)
				f.attributionsDal.On("UpdateAttribution", ctx, "gcp-attribution", []firestore.Update{
					{Path: "filters", Value: gcpFilters},
					{Path: "formula", Value: "A OR (B AND C AND D)"},
					{Path: "type", Value: "managed"},
					{Path: "classification", Value: "invoice"},
					{Path: "hidden", Value: true},
					{Path: "collaborators", Value: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}}},
					{Path: "public", Value: &publicAccessView},
					{Path: "cloud", Value: []string{"google-cloud"}},
				}).Return(nil)
				f.attributionsDal.On("UpdateAttribution", ctx, "aws-attribution", []firestore.Update{
					{Path: "filters", Value: awsFilters},
					{Path: "formula", Value: "A"},
					{Path: "type", Value: "managed"},
					{Path: "classification", Value: "invoice"},
					{Path: "hidden", Value: true},
					{Path: "collaborators", Value: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}}},
					{Path: "public", Value: &publicAccessView},
					{Path: "cloud", Value: []string{"amazon-web-services"}},
				}).Return(nil)
				f.assetsDal.On("GetAWSAsset", ctx, "asset-6").Return(&pkg.AWSAsset{}, nil)

			},
			expectedResult: []*firestore.DocumentRef{
				{ID: "gcp-attribution"},
				{ID: "aws-attribution"},
			},
		},
		{
			name: "Error getting attributions",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.attributionsDal.On("GetAttributions", ctx, mock.AnythingOfType("[]*firestore.DocumentRef")).Return(nil, errors.New("error getting attributions"))
			},
			wantErr:     true,
			expectedErr: errors.New("error getting attributions"),
		},

		{
			name: "Error creating gcp attribution",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.attributionsDal.On("GetAttributions", ctx, mock.AnythingOfType("[]*firestore.DocumentRef")).Return([]*attribution.Attribution{}, nil)
				f.attributionsDal.On("CreateAttribution", ctx, &attribution.Attribution{
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
					Access: collab.Access{
						Collaborators: []collab.Collaborator{
							{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
						},
						Public: &publicAccessView,
					},
					Customer: req.Customer.Snapshot.Ref,
					Name:     "[123] Entity - Google Cloud",
					Cloud:    []string{"google-cloud"},
				}).Return(nil, errors.New("error creating gcp attribution"))
			},
			wantErr:     true,
			expectedErr: errors.New("error creating gcp attribution"),
		},
		{
			name: "Error creating aws attribution",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.attributionsDal.On("GetAttributions", ctx, mock.AnythingOfType("[]*firestore.DocumentRef")).Return([]*attribution.Attribution{}, nil)
				f.attributionsDal.On("CreateAttribution", ctx, &attribution.Attribution{
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
					Access: collab.Access{
						Collaborators: []collab.Collaborator{
							{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
						},
						Public: &publicAccessView,
					},
					Customer: req.Customer.Snapshot.Ref,
					Name:     "[123] Entity - Google Cloud",
					Cloud:    []string{"google-cloud"},
				}).Return(&attribution.Attribution{Ref: &firestore.DocumentRef{ID: "gcp-attribution"}}, nil)
				f.attributionsDal.On("CreateAttribution", ctx, &attribution.Attribution{
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
					Access: collab.Access{
						Collaborators: []collab.Collaborator{
							{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
						},
						Public: &publicAccessView,
					},
					Customer: req.Customer.Snapshot.Ref,
					Name:     "[123] Entity - Amazon Web Services",
					Cloud:    []string{"amazon-web-services"},
				}).Return(nil, errors.New("error creating aws attribution"))
			},
			wantErr:     true,
			expectedErr: errors.New("error creating aws attribution"),
		},
		{
			name: "Error getting assets in entity",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.attributionsDal.On("GetAttributions", ctx, mock.AnythingOfType("[]*firestore.DocumentRef")).Return([]*attribution.Attribution{}, nil)
				f.attributionsDal.On("CreateAttribution", ctx, &attribution.Attribution{
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
					Access: collab.Access{
						Collaborators: []collab.Collaborator{
							{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
						},
						Public: &publicAccessView,
					},
					Customer: req.Customer.Snapshot.Ref,
					Name:     "[123] Entity - Google Cloud",
					Cloud:    []string{"google-cloud"},
				}).Return(&attribution.Attribution{Ref: &firestore.DocumentRef{ID: "gcp-attribution"}}, nil)
				f.attributionsDal.On("CreateAttribution", ctx, &attribution.Attribution{
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
					Access: collab.Access{
						Collaborators: []collab.Collaborator{
							{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
						},
						Public: &publicAccessView,
					},
					Customer: req.Customer.Snapshot.Ref,
					Name:     "[123] Entity - Amazon Web Services",
					Cloud:    []string{"amazon-web-services"},
				}).Return(&attribution.Attribution{Ref: &firestore.DocumentRef{ID: "aws-attribution"}}, nil)
				f.assetsDal.On("GetAssetsInEntity", ctx, req.Entity.Snapshot.Ref).Return(nil, errors.New("error getting assets in entity"))
			},
			wantErr:     true,
			expectedErr: errors.New("error getting assets in entity"),
		},
		{
			name: "Error updating gcp assets and formula",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.attributionsDal.On("GetAttributions", ctx, mock.AnythingOfType("[]*firestore.DocumentRef")).Return([]*attribution.Attribution{}, nil)
				f.attributionsDal.On("CreateAttribution", ctx, &attribution.Attribution{
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
					Access: collab.Access{
						Collaborators: []collab.Collaborator{
							{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
						},
						Public: &publicAccessView,
					},
					Customer: req.Customer.Snapshot.Ref,
					Name:     "[123] Entity - Google Cloud",
					Cloud:    []string{"google-cloud"},
				}).Return(&attribution.Attribution{Ref: &firestore.DocumentRef{ID: "gcp-attribution"}}, nil)
				f.attributionsDal.On("CreateAttribution", ctx, &attribution.Attribution{
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
					Access: collab.Access{
						Collaborators: []collab.Collaborator{
							{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
						},
						Public: &publicAccessView,
					},
					Customer: req.Customer.Snapshot.Ref,
					Name:     "[123] Entity - Amazon Web Services",
					Cloud:    []string{"amazon-web-services"},
				}).Return(&attribution.Attribution{Ref: &firestore.DocumentRef{ID: "aws-attribution"}}, nil)
				f.assetsDal.On("GetAssetsInEntity", ctx, req.Entity.Snapshot.Ref).Return(getAssetsResult, nil)
				f.attributionsDal.On("UpdateAttribution", ctx, "gcp-attribution", []firestore.Update{
					{Path: "filters", Value: gcpFilters},
					{Path: "formula", Value: "A OR (B AND C AND D)"},
					{Path: "type", Value: "managed"},
					{Path: "classification", Value: "invoice"},
					{Path: "hidden", Value: true},
					{Path: "collaborators", Value: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}}},
					{Path: "public", Value: &publicAccessView},
					{Path: "cloud", Value: []string{"google-cloud"}},
				}).Return(errors.New("error updating gcp attribution"))
			},
			wantErr:     true,
			expectedErr: errors.New("error updating gcp attribution"),
		},
		{
			name: "Error updating aws attribution",
			args: args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.attributionsDal.On("GetAttributions", ctx, mock.AnythingOfType("[]*firestore.DocumentRef")).Return([]*attribution.Attribution{}, nil)
				f.attributionsDal.On("CreateAttribution", ctx, &attribution.Attribution{
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
					Access: collab.Access{
						Collaborators: []collab.Collaborator{
							{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
						},
						Public: &publicAccessView,
					},
					Customer: req.Customer.Snapshot.Ref,
					Name:     "[123] Entity - Google Cloud",
					Cloud:    []string{"google-cloud"},
				}).Return(&attribution.Attribution{Ref: &firestore.DocumentRef{ID: "gcp-attribution"}}, nil)
				f.attributionsDal.On("CreateAttribution", ctx, &attribution.Attribution{
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
					Access: collab.Access{
						Collaborators: []collab.Collaborator{
							{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
						},
						Public: &publicAccessView,
					},
					Customer: req.Customer.Snapshot.Ref,
					Name:     "[123] Entity - Amazon Web Services",
					Cloud:    []string{"amazon-web-services"},
				}).Return(&attribution.Attribution{Ref: &firestore.DocumentRef{ID: "aws-attribution"}}, nil)
				f.assetsDal.On("GetAssetsInEntity", ctx, req.Entity.Snapshot.Ref).Return(getAssetsResult, nil)
				f.attributionsDal.On("UpdateAttribution", ctx, "gcp-attribution", []firestore.Update{
					{Path: "filters", Value: gcpFilters},
					{Path: "formula", Value: "A OR (B AND C AND D)"},
					{Path: "type", Value: "managed"},
					{Path: "classification", Value: "invoice"},
					{Path: "hidden", Value: true},
					{Path: "collaborators", Value: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}}},
					{Path: "public", Value: &publicAccessView},
					{Path: "cloud", Value: []string{"google-cloud"}},
				}).Return(nil)
				f.attributionsDal.On("UpdateAttribution", ctx, "aws-attribution", []firestore.Update{
					{Path: "filters", Value: awsFilters},
					{Path: "formula", Value: "A"},
					{Path: "type", Value: "managed"},
					{Path: "classification", Value: "invoice"},
					{Path: "hidden", Value: true},
					{Path: "collaborators", Value: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}}},
					{Path: "public", Value: &publicAccessView},
					{Path: "cloud", Value: []string{"amazon-web-services"}},
				}).Return(errors.New("error updating aws attribution"))
				f.assetsDal.On("GetAWSAsset", ctx, "asset-6").Return(&pkg.AWSAsset{}, nil)

			},
			wantErr:     true,
			expectedErr: errors.New("error updating aws attribution"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				attributionsDal: &mocks.Attributions{},
				assetsDal:       &assetsDalMocks.Assets{},
			}

			s := &AttributionsService{
				dal:       tt.fields.attributionsDal,
				assetsDal: tt.fields.assetsDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			attributions, err := s.CreateAttributionsForInvoiceAssetTypes(ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionsService.CreateAttributionsForInvoiceAssetTypes() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else if !tt.wantErr {
				assert.Equal(t, tt.expectedResult, attributions)
			}
		})
	}
}

func TestAttributionsService_getExistingAttributionsForEntity(t *testing.T) {
	attributions := []*attribution.Attribution{
		{Name: "attribution 1"},
		{Name: "attribution 2"},
		{Name: "[123] Entity - Google Cloud"},
		{Name: "[123] Entity - Amazon Web Services"},
		{Name: "attribution 3"},
		{Name: "[145] Entity1 - Google Cloud"},
		{Name: "attribution 4"},
		{Name: "[145] Entity1 - Amazon Web Services"},
	}

	entity := &common.Entity{
		Name:       "Entity",
		PriorityID: "123",
	}

	gcpAttribution, awsAttribution := getExistingAttributionsForEntity(attributions, entity)

	assert.Equal(t, &attribution.Attribution{Name: "[123] Entity - Google Cloud"}, gcpAttribution)
	assert.Equal(t, &attribution.Attribution{Name: "[123] Entity - Amazon Web Services"}, awsAttribution)

	attributions = []*attribution.Attribution{
		{Name: "attribution 1"},
	}

	gcpAttribution, awsAttribution = getExistingAttributionsForEntity(attributions, entity)

	assert.Nil(t, gcpAttribution)
	assert.Nil(t, awsAttribution)
}

func TestAttributionsService_updateAttributionFiltersAndFormula(t *testing.T) {
	type fields struct {
		attributionsDal *mocks.Attributions
	}

	type args struct {
		ctx         context.Context
		assets      []*pkg.BaseAsset
		attribution *attribution.Attribution
	}

	ctx := context.Background()
	assets := []*pkg.BaseAsset{
		{
			ID:        common.Assets.GoogleCloud + "-first-ID",
			AssetType: common.Assets.GoogleCloud,
		},
	}
	attribution := &attribution.Attribution{
		Ref: &firestore.DocumentRef{
			ID: "attribution-id",
		},
	}

	expectedFilters := []report.BaseConfigFilter{
		{
			Key:       metadata.MetadataFieldKeyBillingAccountID,
			Type:      metadata.MetadataFieldTypeFixed,
			Values:    &[]string{"first-ID"},
			ID:        "fixed:billing_account_id",
			Field:     "T.billing_account_id",
			AllowNull: false,
			Regexp:    nil,
			Inverse:   false,
		},
		{
			Key:       metadata.MetadataFieldKeyProjectID,
			Type:      metadata.MetadataFieldTypeFixed,
			ID:        "fixed:project_id",
			Field:     "T.project_id",
			AllowNull: true,
		},
		{
			Key:     metadata.MetadataFieldKeyServiceDescription,
			Type:    metadata.MetadataFieldTypeFixed,
			Values:  &[]string{"Looker"},
			ID:      "fixed:service_description",
			Field:   "T.service_description",
			Inverse: true,
		},
	}
	publicAccessView := collab.PublicAccessView

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
		fields      fields
		on          func(*fields)
	}{
		{
			name: "Successfully update attributions filters and formula",
			args: args{
				ctx,
				assets,
				attribution,
			},
			on: func(f *fields) {
				f.attributionsDal.On("UpdateAttribution", ctx, attribution.Ref.ID, []firestore.Update{
					{Path: "filters", Value: expectedFilters},
					{Path: "formula", Value: "(A AND B AND C)"},
					{Path: "type", Value: "managed"},
					{Path: "classification", Value: "invoice"},
					{Path: "hidden", Value: true},
					{Path: "collaborators", Value: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}}},
					{Path: "public", Value: &publicAccessView},
					{Path: "cloud", Value: []string{"google-cloud"}},
				}).Return(nil)
			},
		},
		{
			name: "Error updating attributions filters and formula",
			args: args{
				ctx,
				assets,
				attribution,
			},
			on: func(f *fields) {
				f.attributionsDal.On("UpdateAttribution", ctx, attribution.Ref.ID, []firestore.Update{
					{Path: "filters", Value: expectedFilters},
					{Path: "formula", Value: "(A AND B AND C)"},
					{Path: "type", Value: "managed"},
					{Path: "classification", Value: "invoice"},
					{Path: "hidden", Value: true},
					{Path: "collaborators", Value: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}}},
					{Path: "public", Value: &publicAccessView},
					{Path: "cloud", Value: []string{"google-cloud"}},
				}).Return(errors.New("error updating attribution"))
			},
			wantErr:     true,
			expectedErr: errors.New("error updating attribution"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				attributionsDal: &mocks.Attributions{},
			}

			s := &AttributionsService{
				dal: tt.fields.attributionsDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.updateAttribution(ctx, tt.args.assets, tt.args.attribution, common.Assets.GoogleCloud)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionsService.updateAttributionFiltersAndFormula() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else if !tt.wantErr {
				assert.Nil(t, err)
			}
		})
	}
}

func TestAttributionsService_getGCPandAWSassets(t *testing.T) {
	assets := []*pkg.BaseAsset{
		{AssetType: common.Assets.GoogleCloud, ID: "asset-1"},
		{AssetType: common.Assets.GoogleCloudDirect, ID: "asset-2"},
		{AssetType: common.Assets.GoogleCloudProject, ID: "asset-3"},
		{AssetType: common.Assets.GoogleCloudReseller, ID: "asset-4"},
		{AssetType: common.Assets.GoogleCloudStandalone, ID: "asset-5"},
		{AssetType: common.Assets.AmazonWebServices, ID: "asset-6"},
		{AssetType: common.Assets.AmazonWebServicesReseller, ID: "asset-7"},
		{AssetType: common.Assets.AmazonWebServicesStandalone, ID: "asset-8"},
		{AssetType: common.Assets.BetterCloud, ID: "asset-9"},
		{AssetType: common.Assets.GSuite, ID: "asset-10"},
		{AssetType: common.Assets.MicrosoftAzure, ID: "asset-11"},
		{AssetType: common.Assets.Office365, ID: "asset-12"},
		{AssetType: common.Assets.Zendesk, ID: "asset-13"},
		{AssetType: common.Assets.AmazonWebServices, ID: "asset-14"},
		{AssetType: common.Assets.GoogleCloudProject, ID: "asset-15"},
		{AssetType: common.Assets.GoogleCloud, ID: "asset-16"},
	}

	gcpAssets, awsAssets := getGCPandAWSassets(assets)

	assert.Equal(t, []*pkg.BaseAsset{
		{AssetType: common.Assets.GoogleCloud, ID: "asset-1"},
		{AssetType: common.Assets.GoogleCloudProject, ID: "asset-3"},
		{AssetType: common.Assets.GoogleCloudProject, ID: "asset-15"},
		{AssetType: common.Assets.GoogleCloud, ID: "asset-16"},
	}, gcpAssets)
	assert.Equal(t, []*pkg.BaseAsset{
		{AssetType: common.Assets.AmazonWebServices, ID: "asset-6"},
		{AssetType: common.Assets.AmazonWebServices, ID: "asset-14"},
	}, awsAssets)
}

func TestAttributionsService_getAttributionsInGroup(t *testing.T) {
	type fields struct {
		attributionsDal *mocks.Attributions
	}

	type args struct {
		ctx              context.Context
		attributionGroup *attributiongroups.AttributionGroup
	}

	ctx := context.Background()
	attributionGroup := &attributiongroups.AttributionGroup{
		Attributions: []*firestore.DocumentRef{
			{},
			{},
		},
	}

	expectedAttributions := []*attribution.Attribution{
		{},
		{},
	}

	tests := []struct {
		name           string
		args           args
		wantErr        bool
		expectedErr    error
		fields         fields
		on             func(*fields)
		expectedResult []*attribution.Attribution
	}{
		{
			name: "Successfully get attribution group attributions",
			args: args{
				ctx,
				attributionGroup,
			},
			on: func(f *fields) {
				f.attributionsDal.On("GetAttributions", ctx, attributionGroup.Attributions).Return(expectedAttributions, nil)
			},
			expectedResult: expectedAttributions,
		},
		{
			name: "Error getting attribution group attributions",
			args: args{
				ctx,
				attributionGroup,
			},
			on: func(f *fields) {
				f.attributionsDal.On("GetAttributions", ctx, attributionGroup.Attributions).Return(nil, errors.New("error getting attributions"))
			},
			wantErr:     true,
			expectedErr: errors.New("error getting attributions"),
		},
		{
			name: "Empty attribution group attributions",
			args: args{
				ctx,
				&attributiongroups.AttributionGroup{},
			},
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				attributionsDal: &mocks.Attributions{},
			}

			s := &AttributionsService{
				dal: tt.fields.attributionsDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			attributions, err := s.getAttributionsInGroup(ctx, tt.args.attributionGroup)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionsService.getAttributionsInGroup() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else if !tt.wantErr {
				assert.Equal(t, tt.expectedResult, attributions)
			}
		})
	}
}

func TestAttributionsService_createAttributionForType(t *testing.T) {
	type fields struct {
		attributionsDal *mocks.Attributions
	}

	type args struct {
		ctx        context.Context
		req        SyncInvoiceByAssetTypeAttributionRequest
		assetsType string
	}

	ctx := context.Background()
	req := SyncInvoiceByAssetTypeAttributionRequest{
		Customer: &common.Customer{Snapshot: &firestore.DocumentSnapshot{Ref: &firestore.DocumentRef{}}},
		Entity:   &common.Entity{Name: "entity-name", PriorityID: "123"},
	}
	assetsType := common.Assets.GoogleCloud
	publicAccessView := collab.PublicAccessView
	expectedAttribution := &attribution.Attribution{
		Name: "[123] entity-name - Google Cloud",
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
		fields      fields
		on          func(*fields)
	}{
		{
			name: "Successfully create attribution for type",
			args: args{
				ctx,
				req,
				assetsType,
			},
			on: func(f *fields) {
				f.attributionsDal.On("CreateAttribution", ctx, &attribution.Attribution{
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
					Access: collab.Access{
						Collaborators: []collab.Collaborator{
							{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
						},
						Public: &publicAccessView,
					},
					Customer: req.Customer.Snapshot.Ref,
					Name:     "[123] entity-name - Google Cloud",
					Cloud:    []string{"google-cloud"},
				}).Return(expectedAttribution, nil)
			},
		},
		{
			name: "Error creating attribution",
			args: args{
				ctx,
				req,
				assetsType,
			},
			on: func(f *fields) {
				f.attributionsDal.On("CreateAttribution", ctx, &attribution.Attribution{
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
					Access: collab.Access{
						Collaborators: []collab.Collaborator{
							{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
						},
						Public: &publicAccessView,
					},
					Customer: req.Customer.Snapshot.Ref,
					Name:     "[123] entity-name - Google Cloud",
					Cloud:    []string{"google-cloud"},
				}).Return(nil, errors.New("error creating attribution"))
			},
			wantErr:     true,
			expectedErr: errors.New("error creating attribution"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				attributionsDal: &mocks.Attributions{},
			}

			s := &AttributionsService{
				dal: tt.fields.attributionsDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			attributions, err := s.createAttributionForType(ctx, tt.args.req, tt.args.assetsType)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionsService.createAttributionForType() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else if !tt.wantErr {
				assert.Equal(t, expectedAttribution, attributions)
			}
		})
	}
}
