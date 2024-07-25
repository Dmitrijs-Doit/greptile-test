package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	reportMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	reportValidatorServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reportvalidator/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	doitEmployeesMock "github.com/doitintl/hello/scheduled-tasks/doitemployees/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
	ncMocks "github.com/doitintl/notificationcenter/mocks"
)

func TestReportTemplateService_UpdateReportTemplate(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider         logger.Provider
		employeeService        *doitEmployeesMock.ServiceInterface
		reportTemplateDAL      *mocks.ReportTemplateFirestore
		reportValidatorService *reportValidatorServiceMocks.IReportValidatorService
	}

	type args struct {
		ctx               context.Context
		requesterEmail    string
		isDoitEmployee    bool
		reportTemplateID  string
		reportTemplateReq *domain.ReportTemplateReq
	}

	reportTemplateID := "123"
	email := "test@doit.com"

	config := &report.Config{}

	reportTemplateReq := &domain.ReportTemplateReq{
		Config:     config,
		Visibility: domain.VisibilityInternal,
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
			name: "successfully update the report template version",
			args: args{
				ctx:               ctx,
				requesterEmail:    email,
				isDoitEmployee:    true,
				reportTemplateID:  reportTemplateID,
				reportTemplateReq: reportTemplateReq,
			},
			wantErr: false,
			on: func(f *fields) {
				f.employeeService.On(
					"CheckDoiTEmployeeRole",
					testutils.ContextBackgroundMock,
					templateLibraryAdminRole,
					email,
				).
					Return(true, nil).
					Once()
				f.reportValidatorService.On(
					"Validate",
					ctx,
					&report.Report{
						Config: reportTemplateReq.Config,
						Type:   report.ReportTypeManaged,
					},
				).
					Return(nil, nil).
					Once()
				f.reportTemplateDAL.On("RunTransaction", testutils.ContextBackgroundMock, mock.AnythingOfType("dal.TransactionFunc")).
					Return(&domain.ReportTemplate{LastVersion: &firestore.DocumentRef{}}, nil, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, &firestore.DocumentRef{}).
					Return(&domain.ReportTemplateVersion{}, nil, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:         logger.FromContext,
				employeeService:        &doitEmployeesMock.ServiceInterface{},
				reportTemplateDAL:      &mocks.ReportTemplateFirestore{},
				reportValidatorService: &reportValidatorServiceMocks.IReportValidatorService{},
			}

			s := &ReportTemplateService{
				loggerProvider:         tt.fields.loggerProvider,
				employeeService:        tt.fields.employeeService,
				reportTemplateDAL:      tt.fields.reportTemplateDAL,
				reportValidatorService: tt.fields.reportValidatorService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			_, _, err := s.UpdateReportTemplate(
				ctx,
				tt.args.requesterEmail,
				tt.args.isDoitEmployee,
				tt.args.reportTemplateID,
				tt.args.reportTemplateReq,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportTemplateService.UpdateReportTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTemplateService.UpdateReportTemplate() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestReportTemplateService_updateReportTemplateTxFunc(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider     logger.Provider
		employeeService    *doitEmployeesMock.ServiceInterface
		reportTemplateDAL  *mocks.ReportTemplateFirestore
		reportDAL          *reportMocks.Reports
		notificationClient *ncMocks.NotificationSender
	}

	type args struct {
		ctx              context.Context
		requesterEmail   string
		isDoitEmployee   bool
		reportTemplateID string
		report           *report.Report
		categories       []string
		cloud            []string
		visibility       domain.Visibility
	}

	email := "test@doit.com"

	tx := &firestore.Transaction{}

	lastVersionRef := &firestore.DocumentRef{
		ID: "0",
	}

	expectedNewVersionID := "1"

	reportTemplateVersionRef := &firestore.DocumentRef{
		ID: "some version id",
	}

	reportTemplateID := "123"
	reportTemplateRef := &firestore.DocumentRef{
		ID: reportTemplateID,
	}

	createdVersionRef := &firestore.DocumentRef{}

	createdReportRef := &firestore.DocumentRef{
		ID: "some report id",
	}

	reportRef := &firestore.DocumentRef{
		ID: "existing report id",
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
			name: "successfully update the existing report template version when it is pending and not approver",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				isDoitEmployee:   true,
				reportTemplateID: reportTemplateID,
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
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: reportTemplateVersionRef,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(false, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, reportTemplateVersionRef).
					Return(
						&domain.ReportTemplateVersion{
							Approval: domain.Approval{
								Status: domain.StatusPending,
							},
							Cloud:      []string{"old cloud"},
							Categories: []string{"old category"},
							Visibility: domain.VisibilityPrivate,
							Report:     lastVersionRef,
							CreatedBy:  email,
						}, nil).
					Once()
				f.reportDAL.On("Get", testutils.ContextBackgroundMock, lastVersionRef.ID).
					Return(
						&report.Report{}, nil).
					Once()
				f.reportTemplateDAL.On(
					"UpdateReportTemplateVersion",
					testutils.ContextBackgroundMock,
					tx,
					&domain.ReportTemplateVersion{
						Active: false,
						Approval: domain.Approval{
							Status: domain.StatusPending,
						},
						Cloud:      []string{"new cloud"},
						Categories: []string{"new category"},
						Visibility: domain.VisibilityGlobal,
						Report:     lastVersionRef,
						CreatedBy:  email,
					},
				).
					Return(nil).
					Once()
				f.reportDAL.On("Update", testutils.ContextBackgroundMock, lastVersionRef.ID, &report.Report{
					Name:        "new report name",
					Description: "new description name",
					Config: &report.Config{
						Aggregator: report.AggregatorTotal,
					},
				}).
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
			name: "successfully update and auto-approve the existing report template version when it is pending and user is an approver",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				isDoitEmployee:   true,
				reportTemplateID: reportTemplateID,
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
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: lastVersionRef,
						ID:          reportTemplateID,
						Ref:         reportTemplateRef,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(true, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, lastVersionRef).
					Return(
						&domain.ReportTemplateVersion{
							Approval: domain.Approval{
								Status: domain.StatusPending,
							},
							Cloud:      []string{"old cloud"},
							Categories: []string{"old category"},
							Visibility: domain.VisibilityPrivate,
							Report:     reportRef,
							CreatedBy:  email,
							Ref:        lastVersionRef,
						}, nil).
					Once()
				f.reportDAL.On("Get", testutils.ContextBackgroundMock, reportRef.ID).
					Return(
						&report.Report{
							Ref: reportRef,
						}, nil).
					Once()
				f.reportTemplateDAL.On(
					"UpdateReportTemplateVersion",
					testutils.ContextBackgroundMock,
					tx,
					&domain.ReportTemplateVersion{
						Active: true,
						Approval: domain.Approval{
							Status:       domain.StatusApproved,
							ApprovedBy:   &email,
							TimeApproved: &now,
						},
						Cloud:      []string{"new cloud"},
						Categories: []string{"new category"},
						Visibility: domain.VisibilityGlobal,
						Report:     reportRef,
						CreatedBy:  email,
						Ref:        lastVersionRef,
					},
				).
					Return(nil).
					Once()
				f.reportDAL.On("Update", testutils.ContextBackgroundMock, reportRef.ID, &report.Report{
					Name:        "new report name",
					Description: "new description name",
					Config: &report.Config{
						Aggregator: report.AggregatorTotal,
					},
					Ref: reportRef,
				}).
					Return(nil).
					Once()
				f.reportTemplateDAL.On(
					"UpdateReportTemplate",
					testutils.ContextBackgroundMock,
					tx,
					reportTemplateID,
					&domain.ReportTemplate{
						LastVersion:   lastVersionRef,
						ActiveVersion: lastVersionRef,
						ActiveReport:  reportRef,
						Ref:           reportTemplateRef,
						ID:            reportTemplateID,
					},
				).
					Return(nil).
					Once()
			},
		},
		{
			name: "successfully update the existing report template version when it is rejected and user is not approver",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				isDoitEmployee:   true,
				reportTemplateID: reportTemplateID,
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
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: lastVersionRef,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(false, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, lastVersionRef).
					Return(
						&domain.ReportTemplateVersion{
							Approval: domain.Approval{
								Status: domain.StatusRejected,
							},
							Cloud:      []string{"old cloud"},
							Categories: []string{"old category"},
							Visibility: domain.VisibilityPrivate,
							Report:     lastVersionRef,
							Template:   reportTemplateVersionRef,
							CreatedBy:  email,
						}, nil).
					Once()
				f.reportDAL.On("Get", testutils.ContextBackgroundMock, lastVersionRef.ID).
					Return(
						&report.Report{}, nil).
					Once()
				f.reportTemplateDAL.On(
					"UpdateReportTemplateVersion",
					testutils.ContextBackgroundMock,
					tx,
					&domain.ReportTemplateVersion{
						Active: false,
						Approval: domain.Approval{
							Status: domain.StatusPending,
						},
						Cloud:      []string{"new cloud"},
						Categories: []string{"new category"},
						Visibility: domain.VisibilityGlobal,
						Report:     lastVersionRef,
						Template:   reportTemplateVersionRef,
						CreatedBy:  email,
					},
				).
					Return(nil).
					Once()
				f.reportDAL.On("Update", testutils.ContextBackgroundMock, lastVersionRef.ID, &report.Report{
					Name:        "new report name",
					Description: "new description name",
					Config: &report.Config{
						Aggregator: report.AggregatorTotal,
					},
				}).
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
			name: "error updating when existing latest version is approved and user is not doit employee",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				isDoitEmployee:   false,
				reportTemplateID: reportTemplateID,
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
			wantErr: true,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion:   lastVersionRef,
						ActiveVersion: lastVersionRef,
						Ref:           reportTemplateRef,
						ID:            reportTemplateRef.ID,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(false, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, lastVersionRef).
					Return(
						&domain.ReportTemplateVersion{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
							Approval: domain.Approval{
								Status: domain.StatusApproved,
							},
							Cloud:      []string{"old cloud"},
							Categories: []string{"old category"},
							Visibility: domain.VisibilityGlobal,
							Report:     createdReportRef,
							Template:   reportTemplateVersionRef,
							Ref:        lastVersionRef,
							CreatedBy:  email,
						}, nil).
					Once()
			},
		},
		{
			name: "error updating when template is hidden",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				isDoitEmployee:   false,
				reportTemplateID: reportTemplateID,
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
			wantErr: true,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						Hidden:        true,
						LastVersion:   lastVersionRef,
						ActiveVersion: lastVersionRef,
						Ref:           reportTemplateRef,
						ID:            reportTemplateRef.ID,
					}, nil).
					Once()
			},
		},
		{
			name: "successfully create a new report template version when existing latest version is approved and user is not approver (visibility global)",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				isDoitEmployee:   true,
				reportTemplateID: reportTemplateID,
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
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion:   lastVersionRef,
						ActiveVersion: lastVersionRef,
						Ref:           reportTemplateRef,
						ID:            reportTemplateRef.ID,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(false, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, lastVersionRef).
					Return(
						&domain.ReportTemplateVersion{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
							Approval: domain.Approval{
								Status: domain.StatusApproved,
							},
							Cloud:      []string{"old cloud"},
							Categories: []string{"old category"},
							Visibility: domain.VisibilityGlobal,
							Report:     createdReportRef,
							Template:   reportTemplateVersionRef,
							Ref:        lastVersionRef,
							CreatedBy:  email,
						}, nil).
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
					"CreateVersion",
					testutils.ContextBackgroundMock,
					tx,
					expectedNewVersionID,
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
						PreviousVersion: lastVersionRef,
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
						ActiveVersion: lastVersionRef,
						Ref:           reportTemplateRef,
						ID:            reportTemplateID,
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
			name: "successfully create a new report template version when existing latest version is approved and user is not approver (visibility changed from private to global)",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				isDoitEmployee:   true,
				reportTemplateID: reportTemplateID,
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
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion:   lastVersionRef,
						ActiveVersion: lastVersionRef,
						Ref:           reportTemplateRef,
						ID:            reportTemplateRef.ID,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(false, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, lastVersionRef).
					Return(
						&domain.ReportTemplateVersion{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
							Approval: domain.Approval{
								Status: domain.StatusApproved,
							},
							Cloud:      []string{"old cloud"},
							Categories: []string{"old category"},
							Visibility: domain.VisibilityPrivate,
							Report:     createdReportRef,
							Template:   reportTemplateVersionRef,
							Ref:        lastVersionRef,
							CreatedBy:  email,
						}, nil).
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
					"CreateVersion",
					testutils.ContextBackgroundMock,
					tx,
					expectedNewVersionID,
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
						PreviousVersion: lastVersionRef,
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
						ActiveVersion: lastVersionRef,
						Ref:           reportTemplateRef,
						ID:            reportTemplateID,
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
			name: "successfully create a new report template version and auto-approve when existing latest version is approved and user is an approver",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				isDoitEmployee:   true,
				reportTemplateID: reportTemplateID,
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
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion:   lastVersionRef,
						ActiveVersion: lastVersionRef,
						Ref:           reportTemplateRef,
						ID:            reportTemplateRef.ID,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(true, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, lastVersionRef).
					Return(
						&domain.ReportTemplateVersion{
							Collaborators: []collab.Collaborator{
								{
									Email: email,
									Role:  collab.CollaboratorRoleOwner,
								},
							},
							Approval: domain.Approval{
								Status: domain.StatusApproved,
							},
							Cloud:      []string{"old cloud"},
							Categories: []string{"old category"},
							Visibility: domain.VisibilityPrivate,
							Report:     createdReportRef,
							Template:   reportTemplateVersionRef,
							Ref:        lastVersionRef,
							CreatedBy:  email,
						}, nil).
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
					"CreateVersion",
					testutils.ContextBackgroundMock,
					tx,
					expectedNewVersionID,
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
						PreviousVersion: lastVersionRef,
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
						Ref:           reportTemplateRef,
						ID:            reportTemplateID,
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

			updateReportTemplateTxFunc := s.getUpdateReportTemplateTxFunc(
				ctx,
				tt.args.requesterEmail,
				tt.args.isDoitEmployee,
				tt.args.reportTemplateID,
				tt.args.report,
				tt.args.categories,
				tt.args.cloud,
				tt.args.visibility,
				now,
			)

			_, err := updateReportTemplateTxFunc(ctx, tx)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportTemplateService.updateReportTemplateTxFunc() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTemplateService.updateReportTemplateTxFunc() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestReportTemplateService_canChangeVisibility(t *testing.T) {
	type args struct {
		lastVisibility domain.Visibility
		newVisibility  domain.Visibility
	}

	tests := []struct {
		name        string
		args        args
		expectedErr error
	}{
		{
			name: "can change to the same private",
			args: args{
				lastVisibility: domain.VisibilityPrivate,
				newVisibility:  domain.VisibilityPrivate,
			},
			expectedErr: nil,
		},
		{
			name: "can change to the same internal",
			args: args{
				lastVisibility: domain.VisibilityInternal,
				newVisibility:  domain.VisibilityInternal,
			},
			expectedErr: nil,
		},
		{
			name: "can change to the same global",
			args: args{
				lastVisibility: domain.VisibilityGlobal,
				newVisibility:  domain.VisibilityGlobal,
			},
			expectedErr: nil,
		},
		{
			name: "can change from private to internal",
			args: args{
				lastVisibility: domain.VisibilityPrivate,
				newVisibility:  domain.VisibilityInternal,
			},
			expectedErr: nil,
		},
		{
			name: "can change from private to global",
			args: args{
				lastVisibility: domain.VisibilityPrivate,
				newVisibility:  domain.VisibilityGlobal,
			},
			expectedErr: nil,
		},
		{
			name: "can change from internal to global",
			args: args{
				lastVisibility: domain.VisibilityInternal,
				newVisibility:  domain.VisibilityGlobal,
			},
			expectedErr: nil,
		},
		{
			name: "err change from internal to private",
			args: args{
				lastVisibility: domain.VisibilityInternal,
				newVisibility:  domain.VisibilityPrivate,
			},
			expectedErr: ErrVisibilityCanNotBeDemoted,
		},
		{
			name: "err change from global to internal",
			args: args{
				lastVisibility: domain.VisibilityGlobal,
				newVisibility:  domain.VisibilityInternal,
			},
			expectedErr: ErrVisibilityCanNotBeDemoted,
		},
		{
			name: "err change from global to private",
			args: args{
				lastVisibility: domain.VisibilityGlobal,
				newVisibility:  domain.VisibilityPrivate,
			},
			expectedErr: ErrVisibilityCanNotBeDemoted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := canChangeVisibility(
				tt.args.lastVisibility,
				tt.args.newVisibility,
			)

			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
