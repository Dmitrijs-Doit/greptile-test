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

func TestReportTemplateService_RejectReportTemplate(t *testing.T) {
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
		comment          string
	}

	reportTemplateID := "123"
	email := "test@doit.com"
	comment := "some reject comment"

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successfully reject the report template",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
				comment:          comment,
			},
			wantErr: false,
			on: func(f *fields) {
				f.employeeService.On(
					"CheckDoiTEmployeeRole",
					ctx,
					templateLibraryAdminRole,
					email,
				).
					Return(true, nil).
					Once()
				f.reportTemplateDAL.On("RunTransaction", ctx, mock.AnythingOfType("dal.TransactionFunc")).
					Return(&domain.ReportTemplate{LastVersion: &firestore.DocumentRef{}}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, &firestore.DocumentRef{}).
					Return(&domain.ReportTemplateVersion{}, nil, nil).
					Once()
			},
		},
		{
			name: "no access to reject the report",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr:     true,
			expectedErr: domain.ErrUnauthorizedReject,
			on: func(f *fields) {
				f.employeeService.On(
					"CheckDoiTEmployeeRole",
					ctx,
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

			_, err := s.RejectReportTemplate(
				ctx,
				tt.args.requesterEmail,
				tt.args.reportTemplateID,
				tt.args.comment,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportTemplateService.RejectReportTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTemplateService.RejectReportTemplate() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestReportTemplateService_rejectReportTemplateTxFunc(t *testing.T) {
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
		comment          string
	}

	reportTemplateID := "123"
	email := "test@doit.com"
	comment := "some reject comment"

	tx := &firestore.Transaction{}

	lastVersionRef := &firestore.DocumentRef{}

	now := time.Now()

	reportID := "some report id"
	reportRef := &firestore.DocumentRef{
		ID: reportID,
	}

	report := reportDomain.Report{
		Name: "some report name",
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
			name: "successfully reject the report template",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
				comment:          comment,
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", ctx, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: lastVersionRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", ctx, lastVersionRef).
					Return(&domain.ReportTemplateVersion{
						Approval: domain.Approval{
							Status: domain.StatusPending,
						},
						Report: reportRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On(
					"UpdateReportTemplateVersion",
					ctx,
					tx,
					&domain.ReportTemplateVersion{
						Active: false,
						Approval: domain.Approval{
							Status: domain.StatusRejected,
							Changes: []domain.Message{
								{
									Email:     email,
									Text:      comment,
									Timestamp: now,
								},
							},
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
			name: "successfully reject the report template and send email to collaborator",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
				comment:          comment,
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", ctx, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: lastVersionRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", ctx, lastVersionRef).
					Return(&domain.ReportTemplateVersion{
						Approval: domain.Approval{
							Status: domain.StatusPending,
						},
						Report: reportRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On(
					"UpdateReportTemplateVersion",
					ctx,
					tx,
					&domain.ReportTemplateVersion{
						Active: false,
						Approval: domain.Approval{
							Status: domain.StatusRejected,
							Changes: []domain.Message{
								{
									Email:     email,
									Text:      comment,
									Timestamp: now,
								},
							},
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
				f.notificationClient.On("Send", ctx, mock.AnythingOfType("notificationcenter.Notification")).
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
				f.reportTemplateDAL.On("Get", ctx, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: lastVersionRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", ctx, lastVersionRef).
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
			name: "error if lastReportTemplateVersion is already rejected",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr:     true,
			expectedErr: ErrVersionIsRejected,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", ctx, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: lastVersionRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", ctx, lastVersionRef).
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
			name: "error if lastReportTemplateVersion is approved",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr:     true,
			expectedErr: ErrVersionIsApproved,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", ctx, tx, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: lastVersionRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", ctx, lastVersionRef).
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
				f.reportTemplateDAL.On("Get", ctx, tx, reportTemplateID).
					Return(&domain.ReportTemplate{Hidden: true}, nil).
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

			rejectReportTemplateTxFunc := s.getRejectReportTemplateTxFunc(
				ctx,
				tt.args.requesterEmail,
				tt.args.reportTemplateID,
				comment,
				now,
			)

			_, err := rejectReportTemplateTxFunc(ctx, tx)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportTemplateService.rejectReportTemplateTxFunc() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTemplateService.rejectReportTemplateTxFunc() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}
