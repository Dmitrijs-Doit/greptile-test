package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/mock"

	reportMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/mocks"
	reportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	doitEmployeesMock "github.com/doitintl/hello/scheduled-tasks/doitemployees/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
	ncMocks "github.com/doitintl/notificationcenter/mocks"
)

func TestReportTemplateService_ApproveReportTemplate(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider    logger.Provider
		employeeService   *doitEmployeesMock.ServiceInterface
		reportTemplateDAL *mocks.ReportTemplateFirestore
	}

	type args struct {
		ctx              context.Context
		requesterEmail   string
		reportTemplateID string
	}

	reportTemplateID := "123"
	email := "test@doit.com"

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successfully approve the report template",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
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
				f.reportTemplateDAL.On("RunTransaction", testutils.ContextBackgroundMock, mock.AnythingOfType("dal.TransactionFunc")).
					Return(&domain.ReportTemplate{LastVersion: &firestore.DocumentRef{}}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, &firestore.DocumentRef{}).
					Return(&domain.ReportTemplateVersion{}, nil).
					Once()
			},
		},
		{
			name: "no access to approve the report",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr:     true,
			expectedErr: domain.ErrUnauthorizedApprove,
			on: func(f *fields) {
				f.employeeService.On(
					"CheckDoiTEmployeeRole",
					testutils.ContextBackgroundMock,
					templateLibraryAdminRole,
					email,
				).
					Return(false, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:    logger.FromContext,
				employeeService:   &doitEmployeesMock.ServiceInterface{},
				reportTemplateDAL: &mocks.ReportTemplateFirestore{},
			}

			s := &ReportTemplateService{
				loggerProvider:    tt.fields.loggerProvider,
				employeeService:   tt.fields.employeeService,
				reportTemplateDAL: tt.fields.reportTemplateDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			_, err := s.ApproveReportTemplate(ctx, tt.args.requesterEmail, tt.args.reportTemplateID)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportTemplateService.ApproveReportTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTemplateService.ApproveReportTemplate() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestReportTemplateService_approveReportTemplateTxFunc(t *testing.T) {
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
		reportTemplateID string
	}

	reportTemplateID := "123"
	email := "test@doit.com"

	tx := &firestore.Transaction{}

	lastVersionRef := &firestore.DocumentRef{}

	previousVersionRef := &firestore.DocumentRef{}

	reportID := "some report id"
	reportRef := &firestore.DocumentRef{
		ID: reportID,
	}

	report := reportDomain.Report{
		Name: "some report name",
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
			name: "successfully approve the report template",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: lastVersionRef,
						ID:          reportTemplateID,
					}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, lastVersionRef).
					Return(&domain.ReportTemplateVersion{
						Approval: domain.Approval{
							Status: domain.StatusPending,
						},
						Report: reportRef,
					}, nil).
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
						ID:            reportTemplateID,
					},
				).
					Return(nil).
					Once()
				f.reportTemplateDAL.On(
					"UpdateReportTemplateVersion",
					testutils.ContextBackgroundMock,
					tx,
					&domain.ReportTemplateVersion{
						Active: true,
						Approval: domain.Approval{
							ApprovedBy:   &email,
							Status:       domain.StatusApproved,
							TimeApproved: &now,
						},
						Report: reportRef,
					},
				).
					Return(nil).
					Once()
				f.reportDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					reportID,
				).
					Return(&report, nil).
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
			name: "error if lastReportTemplateVersion is canceled",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr:     true,
			expectedErr: ErrVersionIsCanceled,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: lastVersionRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, lastVersionRef).
					Return(&domain.ReportTemplateVersion{
						Active: false,
						Approval: domain.Approval{
							Status: domain.StatusCanceled,
						},
					}, nil).
					Once()
			},
		},
		{
			name: "error if lastReportTemplateVersion is rejected",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr:     true,
			expectedErr: ErrVersionIsRejected,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: lastVersionRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, lastVersionRef).
					Return(&domain.ReportTemplateVersion{
						Active: false,
						Approval: domain.Approval{
							Status: domain.StatusRejected,
						},
					}, nil).
					Once()
			},
		},
		{
			name: "error if lastReportTemplateVersion is already approved",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr:     true,
			expectedErr: ErrVersionIsApproved,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: lastVersionRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, lastVersionRef).
					Return(&domain.ReportTemplateVersion{
						Active: false,
						Approval: domain.Approval{
							Status: domain.StatusApproved,
						},
					}, nil).
					Once()
			},
		},
		{
			name: "error if template is hidden",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr:     true,
			expectedErr: ErrTemplateIsHidden,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{Hidden: true}, nil).
					Once()
			},
		},
		{
			name: "successfully update the previous reportTemplateVersion when it exists and was active",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: lastVersionRef,
						ID:          reportTemplateID,
					}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, lastVersionRef).
					Return(&domain.ReportTemplateVersion{
						Approval: domain.Approval{
							Status: domain.StatusPending,
						},
						Report:          reportRef,
						PreviousVersion: previousVersionRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, previousVersionRef).
					Return(&domain.ReportTemplateVersion{
						Approval: domain.Approval{
							Status:     domain.StatusApproved,
							ApprovedBy: &email,
						},
						Active: true,
					}, nil).
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
						ID:            reportTemplateID,
					},
				).
					Return(nil).
					Once()
				f.reportTemplateDAL.On(
					"UpdateReportTemplateVersion",
					testutils.ContextBackgroundMock,
					tx,
					&domain.ReportTemplateVersion{
						Active: true,
						Approval: domain.Approval{
							ApprovedBy:   &email,
							Status:       domain.StatusApproved,
							TimeApproved: &now,
						},
						Report:          reportRef,
						PreviousVersion: previousVersionRef,
					},
				).
					Return(nil).
					Once()
				f.reportTemplateDAL.On(
					"UpdateReportTemplateVersion",
					testutils.ContextBackgroundMock,
					tx,
					&domain.ReportTemplateVersion{
						Active: false,
						Approval: domain.Approval{
							ApprovedBy: &email,
							Status:     domain.StatusApproved,
						},
						PreviousVersion: nil,
					},
				).
					Return(nil).
					Once()
				f.reportDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					reportID,
				).
					Return(&report, nil).
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

			approveReportTemplateTxFunc := s.getApproveReportTemplateTxFunc(
				ctx,
				tt.args.requesterEmail,
				tt.args.reportTemplateID,
				now,
			)

			_, err := approveReportTemplateTxFunc(ctx, tx)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportTemplateService.approveReportTemplateTxFunc() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTemplateService.approveReportTemplateTxFunc() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}
