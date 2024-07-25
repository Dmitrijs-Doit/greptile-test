package externalreport

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionGroupsServiceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	attributionServiceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	domainSplit "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	splittingServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/service/mocks"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
)

func TestExternalReport_NewExternalSplitFromInternal(t *testing.T) {
	type fields struct {
		splittingService        *splittingServiceMocks.ISplittingService
		attributionService      *attributionServiceMock.AttributionsIface
		attributionGroupService *attributionGroupsServiceMock.AttributionGroupsIface
	}

	ctx := context.Background()

	targetVal := 1.0

	split := domainSplit.Split{
		ID:     "attribution_group:attrgroup111",
		Key:    "",
		Type:   "attribution_group",
		Origin: "attribution:attr1",
		Mode:   "custom",
		Targets: []domainSplit.SplitTarget{
			{
				ID:    "attribution:attr2",
				Value: targetVal,
			},
		},
		IncludeOrigin: true,
	}

	splits := []domainSplit.Split{split}

	externalSplit := domainExternalReport.ExternalSplit{
		ID:   "attrgroup111",
		Type: metadata.MetadataFieldTypeAttributionGroup,
		Mode: domainSplit.ModeCustom,
		Origin: domainExternalReport.ExternalOrigin{
			ID:   "attr1",
			Type: metadata.MetadataFieldTypeAttribution,
		},
		IncludeOrigin: true,
		Targets: []domainExternalReport.ExternalSplitTarget{
			{
				ID:    "attr2",
				Type:  metadata.MetadataFieldTypeAttribution,
				Value: &targetVal,
			},
		},
	}

	tests := []struct {
		name                 string
		fields               fields
		on                   func(*fields)
		externalSplits       []*domainExternalReport.ExternalSplit
		want                 []domainSplit.Split
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name:           "happy path",
			externalSplits: []*domainExternalReport.ExternalSplit{&externalSplit},
			on: func(f *fields) {
				f.splittingService.On("ValidateSplitsReq",
					&splits,
				).Return(nil)
				f.attributionService.On("GetAttributions",
					ctx,
					mock.AnythingOfType("[]string"),
				).
					Return([]*attribution.Attribution{
						{
							ID: "attr1",
						},
						{
							ID: "attr2",
						},
					}, nil)
				f.attributionGroupService.On("GetAttributionGroups",
					ctx,
					[]string{"attrgroup111"},
				).
					Return([]*attributiongroups.AttributionGroup{
						{
							ID: "attrgroup111",
						},
					}, nil)
			},
			want:                 splits,
			wantValidationErrors: nil,
		},
		{
			name:           "fail when validation of splitReq fails",
			externalSplits: []*domainExternalReport.ExternalSplit{&externalSplit},
			on: func(f *fields) {
				f.splittingService.On("ValidateSplitsReq",
					&splits,
				).Return([]error{errors.New("some validateSplitReqError")})
				f.attributionService.On("GetAttributions",
					ctx,
					mock.AnythingOfType("[]string"),
				).
					Return([]*attribution.Attribution{
						{
							ID: "attr1",
						},
						{
							ID: "attr2",
						},
					}, nil)
				f.attributionGroupService.On("GetAttributionGroups",
					ctx,
					[]string{"attrgroup111"},
				).
					Return([]*attributiongroups.AttributionGroup{
						{
							ID: "attrgroup111",
						},
					}, nil)
			},
			want: splits,
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   "split",
					Message: "some validateSplitReqError",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				splittingService:        &splittingServiceMocks.ISplittingService{},
				attributionService:      &attributionServiceMock.AttributionsIface{},
				attributionGroupService: &attributionGroupsServiceMock.AttributionGroupsIface{},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &Service{
				splittingService:        tt.fields.splittingService,
				attributionService:      tt.fields.attributionService,
				attributionGroupService: tt.fields.attributionGroupService,
			}

			got, validationErrors, err := s.NewExternalSplitToInternal(ctx, tt.externalSplits)
			if (err != nil) != tt.wantErr {
				t.Errorf("external_report.ExternalSplitFromInternal() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)

			assert.Equal(t, tt.want, got)
		})
	}
}
