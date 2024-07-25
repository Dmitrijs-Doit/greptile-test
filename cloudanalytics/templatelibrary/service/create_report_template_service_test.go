package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/mock"

	attributionGroupsMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionsMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	reportMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	reportValidatorServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reportvalidator/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	doitEmployeesMock "github.com/doitintl/hello/scheduled-tasks/doitemployees/mocks"
	permissionsDomain "github.com/doitintl/hello/scheduled-tasks/framework/mid/permissions/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
	ncMocks "github.com/doitintl/notificationcenter/mocks"
)

var (
	templateLibraryAdminRole = string(permissionsDomain.DoitRoleCATemplateLibraryAdmin)
)

func TestReportTemplateService_CreateReportTemplate(t *testing.T) {
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
		ctx               context.Context
		requesterEmail    string
		reportTemplateReq *domain.ReportTemplateReq
	}

	email := "test@doit.com"
	attributionID := "111"
	attributionRef1 := firestore.DocumentRef{}
	presetAttributions := []*attribution.Attribution{{Type: string(attribution.ObjectTypePreset)}}

	agID1 := "1"

	agRef1 := firestore.DocumentRef{}
	presetAGs := []*attributiongroups.AttributionGroup{{Type: attribution.ObjectTypePreset}}

	reportName := "report name"
	reportDescription := "report desc"

	reportTemplateReq := domain.ReportTemplateReq{
		Name:        reportName,
		Description: reportDescription,
		Config:      report.NewConfig(),
		Visibility:  domain.VisibilityInternal,
	}

	reportTemplateReq.Config.Filters = []*report.ConfigFilter{
		{
			BaseConfigFilter: report.BaseConfigFilter{
				ID:     "attribution:attribution",
				Values: &[]string{attributionID},
			},
		},
	}

	reportTemplateReq.Config.Rows = []string{"attribution_group:" + agID1}
	reportTemplateReq.Config.Optional = []report.OptionalField{
		{
			Key:  "system-label",
			Type: metadataDomain.MetadataFieldTypeSystemLabel,
		},
	}

	managedReport := report.NewDefaultReport()
	managedReport.Name = "report name"
	managedReport.Description = "report desc"
	managedReport.Config.Filters = []*report.ConfigFilter{
		{
			BaseConfigFilter: report.BaseConfigFilter{
				ID:     "attribution:attribution",
				Values: &[]string{attributionID},
			},
		},
	}
	managedReport.Config.Rows = []string{"attribution_group:" + agID1}
	managedReport.Config.Optional = []report.OptionalField{
		{
			Key:  "system-label",
			Type: metadataDomain.MetadataFieldTypeSystemLabel,
		},
	}
	managedReport.Type = "managed"

	reportTemplateWithoutConfigReq := domain.ReportTemplateReq{
		Name:        "report name",
		Description: "report desc",
	}

	reportTemplateWithInvalidConfigReq := domain.ReportTemplateReq{
		Name:        "report name",
		Description: "report desc",
		Config:      report.NewConfig(),
	}
	reportTemplateWithInvalidConfigReq.Config.Aggregator = "some invalid aggr"

	reportWithInvalidConfig := report.NewDefaultReport()
	reportWithInvalidConfig.Name = reportTemplateWithInvalidConfigReq.Name
	reportWithInvalidConfig.Description = reportTemplateWithInvalidConfigReq.Description
	reportWithInvalidConfig.Config.Aggregator = "some invalid aggr"
	reportWithInvalidConfig.Type = report.ReportTypeManaged

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "report template successful creation",
			args: args{
				ctx:               ctx,
				requesterEmail:    email,
				reportTemplateReq: &reportTemplateReq,
			},
			on: func(f *fields) {
				f.reportValidatorService.On("Validate", ctx, managedReport).
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
				f.reportTemplateDAL.On("RunTransaction", testutils.ContextBackgroundMock, mock.AnythingOfType("dal.TransactionFunc")).
					Return(&domain.ReportTemplate{LastVersion: &firestore.DocumentRef{}}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, &firestore.DocumentRef{}).
					Return(&domain.ReportTemplateVersion{}, nil).
					Once()
			},
		},
		{
			name: "error report template without config",
			args: args{
				ctx:               ctx,
				requesterEmail:    email,
				reportTemplateReq: &reportTemplateWithoutConfigReq,
			},
			wantErr:     true,
			expectedErr: domain.ErrNoReportTemplateConfig,
		},
		{
			name: "error report template with invalid config",
			args: args{
				ctx:               ctx,
				requesterEmail:    email,
				reportTemplateReq: &reportTemplateWithInvalidConfigReq,
			},
			wantErr:     true,
			expectedErr: domain.ErrInvalidReportTemplateConfig,
			on: func(f *fields) {
				f.reportValidatorService.On("Validate", ctx, &report.Report{
					Name:        reportName,
					Description: reportDescription,
					Config:      reportTemplateWithInvalidConfigReq.Config,
					Type:        report.ReportTypeManaged,
				},
				).
					Return([]errormsg.ErrorMsg{{Field: "field", Message: "msg"}}, errors.New("error")).
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

			_, _, err := s.CreateReportTemplate(ctx, tt.args.requesterEmail, tt.args.reportTemplateReq)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportTemplateService.CreateReportTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTemplateService.CreateReportTemplate() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestReportTemplateService_createReportTemplateTxFunc(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider     logger.Provider
		employeeService    *doitEmployeesMock.ServiceInterface
		reportTemplateDAL  *mocks.ReportTemplateFirestore
		reportDAL          *reportMocks.Reports
		notificationClient *ncMocks.NotificationSender
	}

	type args struct {
		ctx            context.Context
		requesterEmail string
		reportTemplate *domain.ReportTemplate
		report         *report.Report
		categories     []string
		cloud          []string
		visibility     domain.Visibility
	}

	email := "test@doit.com"

	reportTemplateVersionID := "0"

	tx := &firestore.Transaction{}

	reportTemplateID := "123"
	reportTemplateRef := &firestore.DocumentRef{
		ID: reportTemplateID,
	}

	createdVersionRef := &firestore.DocumentRef{}

	createdReportRef := &firestore.DocumentRef{
		ID: "some report id",
	}

	now := time.Now()

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successfully create reportTemplate and version in pending state when user is not an approver",
			args: args{
				ctx:            ctx,
				requesterEmail: email,
				reportTemplate: &domain.ReportTemplate{},
				report: &report.Report{
					Name:        "new report name",
					Description: "new description name",
					Config: &report.Config{
						Aggregator: report.AggregatorTotal,
					},
				},
				categories: []string{"new category"},
				cloud:      []string{"new cloud"},
				visibility: domain.VisibilityGlobal,
			},
			on: func(f *fields) {
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(false, nil).
					Once()
				f.reportDAL.On(
					"Create",
					testutils.ContextBackgroundMock,
					tx,
					&report.Report{
						Name:        "new report name",
						Description: "new description name",
						Config: &report.Config{
							Aggregator: report.AggregatorTotal,
						},
					},
				).
					Return(&report.Report{
						Ref: createdReportRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On(
					"CreateReportTemplate",
					testutils.ContextBackgroundMock,
					tx,
					&domain.ReportTemplate{},
				).
					Return(reportTemplateRef, nil).
					Once()
				f.reportTemplateDAL.On(
					"CreateVersion",
					testutils.ContextBackgroundMock,
					tx,
					reportTemplateVersionID,
					reportTemplateID,
					&domain.ReportTemplateVersion{
						CreatedBy: email,
						Active:    false,
						Approval: domain.Approval{
							Status: domain.StatusPending,
						},
						Cloud:           []string{"new cloud"},
						Categories:      []string{"new category"},
						Visibility:      domain.VisibilityGlobal,
						Report:          createdReportRef,
						PreviousVersion: nil,
						Template:        reportTemplateRef,
					},
				).
					Return(createdVersionRef, nil).
					Once()
				f.reportTemplateDAL.On(
					"UpdateReportTemplate",
					testutils.ContextBackgroundMock,
					tx,
					reportTemplateID,
					&domain.ReportTemplate{
						LastVersion: createdVersionRef,
						ID:          reportTemplateID,
						Ref:         reportTemplateRef,
					},
				).
					Return(nil).
					Once()
				f.notificationClient.On(
					"Send",
					ctx,
					mock.AnythingOfType("notificationcenter.Notification"),
				).
					Return("", nil).
					Once()
			},
		},
		{
			name: "successfully create and auto-approve reportTemplate and version in pending state when visibility is private",
			args: args{
				ctx:            ctx,
				requesterEmail: email,
				reportTemplate: &domain.ReportTemplate{},
				report: &report.Report{
					Name:        "new report name",
					Description: "new description name",
					Config: &report.Config{
						Aggregator: report.AggregatorTotal,
					},
				},
				categories: []string{"new category"},
				cloud:      []string{"new cloud"},
				visibility: domain.VisibilityPrivate,
			},
			on: func(f *fields) {
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(false, nil).
					Once()
				f.reportDAL.On(
					"Create",
					testutils.ContextBackgroundMock,
					tx,
					&report.Report{
						Name:        "new report name",
						Description: "new description name",
						Config: &report.Config{
							Aggregator: report.AggregatorTotal,
						},
					},
				).
					Return(&report.Report{
						Ref: createdReportRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On(
					"CreateReportTemplate",
					testutils.ContextBackgroundMock,
					tx,
					&domain.ReportTemplate{},
				).
					Return(reportTemplateRef, nil).
					Once()
				f.reportTemplateDAL.On(
					"CreateVersion",
					testutils.ContextBackgroundMock,
					tx,
					reportTemplateVersionID,
					reportTemplateID,
					&domain.ReportTemplateVersion{
						CreatedBy: email,
						Active:    true,
						Approval: domain.Approval{
							Status:       domain.StatusApproved,
							ApprovedBy:   &email,
							TimeApproved: &now,
						},
						Collaborators: []collab.Collaborator{
							{
								Email: email,
								Role:  collab.CollaboratorRoleOwner,
							},
						},
						Cloud:           []string{"new cloud"},
						Categories:      []string{"new category"},
						Visibility:      domain.VisibilityPrivate,
						Report:          createdReportRef,
						PreviousVersion: nil,
						Template:        reportTemplateRef,
					},
				).
					Return(createdVersionRef, nil).
					Once()
				f.reportTemplateDAL.On(
					"UpdateReportTemplate",
					testutils.ContextBackgroundMock,
					tx,
					reportTemplateID,
					&domain.ReportTemplate{
						LastVersion:   createdVersionRef,
						ActiveVersion: createdVersionRef,
						ActiveReport:  createdReportRef,
						ID:            reportTemplateID,
						Ref:           reportTemplateRef,
					},
				).
					Return(nil).
					Once()
			},
		},
		{
			name: "successfully create and auto-approve reportTemplate and version in pending state when visibility is global and user is auto-approver",
			args: args{
				ctx:            ctx,
				requesterEmail: email,
				reportTemplate: &domain.ReportTemplate{},
				report: &report.Report{
					Name:        "new report name",
					Description: "new description name",
					Config: &report.Config{
						Aggregator: report.AggregatorTotal,
					},
				},
				categories: []string{"new category"},
				cloud:      []string{"new cloud"},
				visibility: domain.VisibilityGlobal,
			},
			on: func(f *fields) {
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(true, nil).
					Once()
				f.reportDAL.On(
					"Create",
					testutils.ContextBackgroundMock,
					tx,
					&report.Report{
						Name:        "new report name",
						Description: "new description name",
						Config: &report.Config{
							Aggregator: report.AggregatorTotal,
						},
					},
				).
					Return(&report.Report{
						Ref: createdReportRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On(
					"CreateReportTemplate",
					testutils.ContextBackgroundMock,
					tx,
					&domain.ReportTemplate{},
				).
					Return(reportTemplateRef, nil).
					Once()
				f.reportTemplateDAL.On(
					"CreateVersion",
					testutils.ContextBackgroundMock,
					tx,
					reportTemplateVersionID,
					reportTemplateID,
					&domain.ReportTemplateVersion{
						CreatedBy: email,
						Active:    true,
						Approval: domain.Approval{
							Status:       domain.StatusApproved,
							ApprovedBy:   &email,
							TimeApproved: &now,
						},
						Cloud:           []string{"new cloud"},
						Categories:      []string{"new category"},
						Visibility:      domain.VisibilityGlobal,
						Report:          createdReportRef,
						PreviousVersion: nil,
						Template:        reportTemplateRef,
					},
				).
					Return(createdVersionRef, nil).
					Once()
				f.reportTemplateDAL.On(
					"UpdateReportTemplate",
					testutils.ContextBackgroundMock,
					tx,
					reportTemplateID,
					&domain.ReportTemplate{
						LastVersion:   createdVersionRef,
						ActiveVersion: createdVersionRef,
						ActiveReport:  createdReportRef,
						ID:            reportTemplateID,
						Ref:           reportTemplateRef,
					},
				).
					Return(nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:     logger.FromContext,
				employeeService:    doitEmployeesMock.NewServiceInterface(t),
				reportTemplateDAL:  mocks.NewReportTemplateFirestore(t),
				reportDAL:          reportMocks.NewReports(t),
				notificationClient: ncMocks.NewNotificationSender(t),
			}

			s := &ReportTemplateService{
				loggerProvider:     tt.fields.loggerProvider,
				employeeService:    tt.fields.employeeService,
				reportTemplateDAL:  tt.fields.reportTemplateDAL,
				reportDAL:          tt.fields.reportDAL,
				notificationClient: tt.fields.notificationClient,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			createReportTemplateTxFunc := s.getCreateReportTemplateTxFunc(
				ctx,
				tt.args.requesterEmail,
				tt.args.reportTemplate,
				tt.args.report,
				tt.args.categories,
				tt.args.cloud,
				tt.args.visibility,
				now,
			)

			_, err := createReportTemplateTxFunc(ctx, tx)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportTemplateService.createReportTemplateTxFunc() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTemplateService.createReportTemplateTxFunc() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}
