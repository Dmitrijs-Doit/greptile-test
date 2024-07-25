package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	doitFirestore "github.com/doitintl/firestore"
	attributionGroupsMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionsMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	reportValidatorServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reportvalidator/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	doitEmployeesMock "github.com/doitintl/hello/scheduled-tasks/doitemployees/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func TestReportTemplateService_validateManagedreport(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider         logger.Provider
		employeeService        *doitEmployeesMock.ServiceInterface
		reportValidatorService *reportValidatorServiceMocks.IReportValidatorService
		reportTemplateDAL      *mocks.ReportTemplateFirestore
		attributionDAL         *attributionsMock.Attributions
		attributionGroupDAL    *attributionGroupsMock.AttributionGroups
	}

	type args struct {
		ctx            context.Context
		requesterEmail string
		report         *report.Report
	}

	email := "test@doit.com"

	attributionID := "111"
	attributionRef1 := firestore.DocumentRef{}

	presetAttrName := "some preset attribution name"
	presetAttributions := []*attribution.Attribution{
		{
			Type: string(attribution.ObjectTypePreset),
			Name: presetAttrName,
		},
	}

	customAttrName := "some custom attribution name"
	customAttributions := []*attribution.Attribution{
		{
			Type: string(attribution.ObjectTypeCustom),
			Name: customAttrName,
		},
	}

	agID1 := "1"
	agID2 := "2"
	agRef1 := firestore.DocumentRef{}

	presetAGName := "some preset attribution group name"
	presetAGs := []*attributiongroups.AttributionGroup{
		{
			Type: attribution.ObjectTypePreset,
			Name: presetAGName,
		},
	}

	customAGName := "some custom attribution group name"
	customAGs := []*attributiongroups.AttributionGroup{
		{
			Type: attribution.ObjectTypeCustom,
			Name: customAGName,
		},
	}

	validReport := report.NewDefaultReport()
	validReport.Config.Filters = []*report.ConfigFilter{{
		BaseConfigFilter: report.BaseConfigFilter{
			ID:     "attribution:attribution",
			Values: &[]string{attributionID},
		},
	}}
	validReport.Config.Rows = []string{"attribution_group:" + agID1}
	validReport.Config.Optional = []report.OptionalField{
		{
			Key:  "system-label",
			Type: metadataDomain.MetadataFieldTypeSystemLabel,
		},
	}

	reportWithoutConfig := report.NewDefaultReport()
	reportWithoutConfig.Config = nil

	reportWithInvalidConfig := report.NewDefaultReport()

	reportWithAttributionInFilters := report.NewDefaultReport()
	reportWithAttributionInFilters.Config.Filters = []*report.ConfigFilter{{
		BaseConfigFilter: report.BaseConfigFilter{
			ID:     "attribution:attribution",
			Values: &[]string{attributionID},
		},
	}}

	reportWithAg := report.NewDefaultReport()
	reportWithAg.Config.Rows = []string{"attribution_group:" + agID1}
	reportWithAg.Config.Cols = []string{"attribution_group:" + agID2}

	reportWithAgInFilters := report.NewDefaultReport()
	reportWithAgInFilters.Config.Filters = []*report.ConfigFilter{{
		BaseConfigFilter: report.BaseConfigFilter{
			ID:     "attribution_group:" + agID1,
			Values: &[]string{attributionID},
		},
	}}

	reportWithMetric := report.NewDefaultReport()
	reportWithMetric.Config.CalculatedMetric = &firestore.DocumentRef{
		ID:   "metric",
		Path: "metric/path",
	}

	reportWithLabel := report.NewDefaultReport()
	reportWithLabel.Config.Optional = []report.OptionalField{
		{
			Key:  "label",
			Type: metadataDomain.MetadataFieldTypeLabel,
		},
	}

	reportWithProjectLabel := report.NewDefaultReport()
	reportWithProjectLabel.Config.Optional = []report.OptionalField{
		{
			Key:  "label",
			Type: metadataDomain.MetadataFieldTypeProjectLabel,
		},
	}

	reportWithTag := report.NewDefaultReport()
	reportWithTag.Config.Optional = []report.OptionalField{
		{
			Key:  "tag",
			Type: metadataDomain.MetadataFieldTypeTag,
		},
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "valid report",
			args: args{
				ctx:            ctx,
				requesterEmail: email,
				report:         validReport,
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportValidatorService.On("Validate", ctx, validReport).
					Return(nil, nil).
					Once()
				f.attributionDAL.On("GetRef", ctx, attributionID).
					Return(&attributionRef1, nil).
					Once()
				f.attributionGroupDAL.On("GetRef", ctx, agID1).
					Return(&agRef1, nil).
					Once()
				f.attributionDAL.On("GetAttributions", ctx, []*firestore.DocumentRef{&attributionRef1}).
					Return(presetAttributions, nil).
					Once()
				f.attributionGroupDAL.On("GetAll", ctx, []*firestore.DocumentRef{&agRef1}).
					Return(presetAGs, nil).
					Once()
			},
		},
		{
			name: "error report without config",
			args: args{
				ctx:            ctx,
				requesterEmail: email,
				report:         reportWithoutConfig,
			},
			wantErr:     true,
			expectedErr: domain.ErrNoReportTemplateConfig,
		},
		{
			name: "error report with invalid config",
			args: args{
				ctx:            ctx,
				requesterEmail: email,
				report:         reportWithInvalidConfig,
			},
			wantErr:     true,
			expectedErr: domain.ErrInvalidReportTemplateConfig,
			on: func(f *fields) {
				f.reportValidatorService.On("Validate", ctx, reportWithInvalidConfig).
					Return([]errormsg.ErrorMsg{{Field: "field", Message: "msg"}}, errors.New("error")).
					Once()
			},
		},
		{
			name: "error report with custom attribution in filters",
			args: args{
				ctx:            ctx,
				requesterEmail: email,
				report:         reportWithAttributionInFilters,
			},
			wantErr:     true,
			expectedErr: domain.ValidationErr{Name: customAttrName, Type: domain.CustomAttributionErrType},
			on: func(f *fields) {
				f.reportValidatorService.On(
					"Validate", ctx, reportWithAttributionInFilters).
					Return(nil, nil).
					Once()
				f.attributionDAL.On("GetRef", ctx, attributionID).
					Return(&attributionRef1, nil).
					Once()
				f.attributionDAL.On("GetAttributions", ctx, []*firestore.DocumentRef{&attributionRef1}).
					Return(customAttributions, nil).
					Once()
			},
		},
		{
			name: "error report with custom attribution group in rows and cols",
			args: args{
				ctx:            ctx,
				requesterEmail: email,
				report:         reportWithAg,
			},
			wantErr:     true,
			expectedErr: domain.ValidationErr{Name: customAGName, Type: domain.CustomAGErrType},
			on: func(f *fields) {
				f.reportValidatorService.On(
					"Validate", ctx, reportWithAg).
					Return(nil, nil).
					Once()
				f.attributionGroupDAL.On("GetRef", ctx, agID1).
					Return(&agRef1, nil).
					Once()
				f.attributionGroupDAL.On("GetRef", ctx, agID2).
					Return(&agRef1, nil).
					Once()
				f.attributionGroupDAL.On("GetAll", ctx, []*firestore.DocumentRef{&agRef1, &agRef1}).
					Return(customAGs, nil).
					Once()
			},
		},
		{
			name: "error report with custom attribution group in filters",
			args: args{
				ctx:            ctx,
				requesterEmail: email,
				report:         reportWithAgInFilters,
			},
			wantErr:     true,
			expectedErr: domain.ValidationErr{Name: customAGName, Type: domain.CustomAGErrType},
			on: func(f *fields) {
				f.reportValidatorService.On("Validate", ctx, reportWithAgInFilters).
					Return(nil, nil).
					Once()
				f.attributionGroupDAL.On("GetRef", ctx, agID1).
					Return(&agRef1, nil).
					Once()
				f.attributionGroupDAL.On("GetAll", ctx, []*firestore.DocumentRef{&agRef1}).
					Return(customAGs, nil).
					Once()
			},
		},
		{
			name: "error report with custom metric",
			args: args{
				ctx:            ctx,
				requesterEmail: email,
				report:         reportWithMetric,
			},
			wantErr:     true,
			expectedErr: domain.ErrCustomMetric,
			on: func(f *fields) {
				f.reportValidatorService.On("Validate", ctx, reportWithMetric).
					Return(nil, nil).
					Once()
			},
		},
		{
			name: "error report with custom label",
			args: args{
				ctx:            ctx,
				requesterEmail: email,
				report:         reportWithLabel,
			},
			wantErr:     true,
			expectedErr: domain.ErrCustomLabel,
			on: func(f *fields) {
				f.reportValidatorService.On("Validate", ctx, reportWithLabel).
					Return(nil, nil).
					Once()
			},
		},
		{
			name: "error report with project label",
			args: args{
				ctx:            ctx,
				requesterEmail: email,
				report:         reportWithProjectLabel,
			},
			wantErr:     true,
			expectedErr: domain.ErrCustomLabel,
			on: func(f *fields) {
				f.reportValidatorService.On("Validate", ctx, reportWithProjectLabel).
					Return(nil, nil).
					Once()
			},
		},
		{
			name: "error report with tag",
			args: args{
				ctx:            ctx,
				requesterEmail: email,
				report:         reportWithTag,
			},
			wantErr:     true,
			expectedErr: domain.ErrCustomLabel,
			on: func(f *fields) {
				f.reportValidatorService.On("Validate", ctx, reportWithTag).
					Return(nil, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:         logger.FromContext,
				employeeService:        &doitEmployeesMock.ServiceInterface{},
				reportValidatorService: &reportValidatorServiceMocks.IReportValidatorService{},
				reportTemplateDAL:      &mocks.ReportTemplateFirestore{},
				attributionDAL:         &attributionsMock.Attributions{},
				attributionGroupDAL:    &attributionGroupsMock.AttributionGroups{},
			}

			s := &ReportTemplateService{
				loggerProvider:         tt.fields.loggerProvider,
				employeeService:        tt.fields.employeeService,
				reportValidatorService: tt.fields.reportValidatorService,
				reportTemplateDAL:      tt.fields.reportTemplateDAL,
				attributionDAL:         tt.fields.attributionDAL,
				attributionGroupDAL:    tt.fields.attributionGroupDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			_, err := s.validateManagedReport(ctx, tt.args.report)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportTemplateService.validateManagedReport() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTemplateService.validateManagedReport() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestReportTemplateService_isAutoApprove(t *testing.T) {
	type args struct {
		visibility             domain.Visibility
		isTemplateLibraryAdmin bool
	}

	tests := []struct {
		name        string
		args        args
		expectedRes bool
	}{
		{
			name: "auto approve when private visibility and not admin",
			args: args{
				visibility:             domain.VisibilityPrivate,
				isTemplateLibraryAdmin: false,
			},
			expectedRes: true,
		},
		{
			name: "auto approve when internal visibility and not admin",
			args: args{
				visibility:             domain.VisibilityInternal,
				isTemplateLibraryAdmin: false,
			},
			expectedRes: false,
		},
		{
			name: "auto approve when internal visibility and admin",
			args: args{
				visibility:             domain.VisibilityInternal,
				isTemplateLibraryAdmin: true,
			},
			expectedRes: true,
		},
		{
			name: "auto approve when global visibility and not admin",
			args: args{
				visibility:             domain.VisibilityGlobal,
				isTemplateLibraryAdmin: false,
			},
			expectedRes: false,
		},
		{
			name: "auto approve when internal visibility and admin",
			args: args{
				visibility:             domain.VisibilityGlobal,
				isTemplateLibraryAdmin: true,
			},
			expectedRes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := isAutoApprove(tt.args.visibility, tt.args.isTemplateLibraryAdmin)

			assert.Equal(t, tt.expectedRes, res)
		})
	}
}

func TestReportTemplateService_validateVisibility(t *testing.T) {
	type args struct {
		visibility string
	}

	tests := []struct {
		name        string
		args        args
		expectedRes []errormsg.ErrorMsg
	}{
		{
			name: "visibility private",
			args: args{
				visibility: "private",
			},
			expectedRes: nil,
		},
		{
			name: "visibility internal",
			args: args{
				visibility: "internal",
			},
			expectedRes: nil,
		},
		{
			name: "visibility global",
			args: args{
				visibility: "global",
			},
			expectedRes: nil,
		},
		{
			name: "visibility invalid",
			args: args{
				visibility: "some random value",
			},
			expectedRes: []errormsg.ErrorMsg{
				errormsg.ErrorMsg{
					Field:   "visibility",
					Message: "invalid report template visibility: some random value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := validateVisibility(domain.Visibility(tt.args.visibility))

			assert.Equal(t, tt.expectedRes, res)
		})
	}
}

func TestReportTemplateService_getReportTemplateWithLastVersion(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider         logger.Provider
		employeeService        *doitEmployeesMock.ServiceInterface
		reportValidatorService *reportValidatorServiceMocks.IReportValidatorService
		reportTemplateDAL      *mocks.ReportTemplateFirestore
		attributionDAL         *attributionsMock.Attributions
		attributionGroupDAL    *attributionGroupsMock.AttributionGroups
	}

	type args struct {
		ctx            context.Context
		reportTemplate *domain.ReportTemplate
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successfully get report template with last version",
			args: args{
				ctx:            ctx,
				reportTemplate: &domain.ReportTemplate{LastVersion: &firestore.DocumentRef{}},
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("GetVersionByRef", ctx, &firestore.DocumentRef{}).
					Return(&domain.ReportTemplateVersion{}, nil).
					Once()
			},
		},
		{
			name: "error empty report template",
			args: args{
				ctx:            ctx,
				reportTemplate: nil,
			},
			wantErr: true,
		},
		{
			name: "error last version not found",
			args: args{
				ctx:            ctx,
				reportTemplate: &domain.ReportTemplate{LastVersion: &firestore.DocumentRef{}},
			},
			wantErr: true,
			on: func(f *fields) {
				f.reportTemplateDAL.On("GetVersionByRef", ctx, &firestore.DocumentRef{}).
					Return(nil, doitFirestore.ErrNotFound).
					Once()
			},
		},
	}

	for _, tt := range tests {
		tt.fields = fields{
			loggerProvider:         logger.FromContext,
			employeeService:        &doitEmployeesMock.ServiceInterface{},
			reportValidatorService: &reportValidatorServiceMocks.IReportValidatorService{},
			reportTemplateDAL:      &mocks.ReportTemplateFirestore{},
			attributionDAL:         &attributionsMock.Attributions{},
			attributionGroupDAL:    &attributionGroupsMock.AttributionGroups{},
		}

		s := &ReportTemplateService{
			loggerProvider:         tt.fields.loggerProvider,
			employeeService:        tt.fields.employeeService,
			reportValidatorService: tt.fields.reportValidatorService,
			reportTemplateDAL:      tt.fields.reportTemplateDAL,
			attributionDAL:         tt.fields.attributionDAL,
			attributionGroupDAL:    tt.fields.attributionGroupDAL,
		}

		if tt.on != nil {
			tt.on(&tt.fields)
		}

		_, err := s.getReportTemplateWithLastVersion(ctx, tt.args.reportTemplate)

		if (err != nil) != tt.wantErr {
			t.Errorf("ReportTemplateService.getReportTemplateWithLastVersion() error = %v, wantErr %v", err, tt.wantErr)
		}

		if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
			t.Errorf("ReportTemplateService.getReportTemplateWithLastVersion() error = %v, expectedErr %v", err, tt.expectedErr)
		}
	}
}
