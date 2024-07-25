package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"

	attributionGroupsMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/mocks"
	attributionsMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	doitEmployeesMock "github.com/doitintl/hello/scheduled-tasks/doitemployees/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func TestReportTemplateService_DeleteReportTemplate(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider      logger.Provider
		employeeService     *doitEmployeesMock.ServiceInterface
		reportTemplateDAL   *mocks.ReportTemplateFirestore
		attributionDAL      *attributionsMock.Attributions
		attributionGroupDAL *attributionGroupsMock.AttributionGroups
	}

	type args struct {
		ctx              context.Context
		requesterEmail   string
		reportTemplateID string
	}

	activeVersionRef := firestore.DocumentRef{}
	lastVersionRef := firestore.DocumentRef{}

	errorRetrievingReportTemplate := errors.New("error retrieving report template")
	errorDeletingReportTemplate := errors.New("error deleting report template")

	reportTemplateID := "123"
	email := "test@doit.com"

	var txNil *firestore.Transaction

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successful deletion with template library admin",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, txNil, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: &lastVersionRef,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(true, nil).
					Once()
				f.reportTemplateDAL.On("HideReportTemplate", testutils.ContextBackgroundMock, reportTemplateID).
					Return(nil).
					Once()
			},
		},
		{
			name: "successful deletion with owner",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, txNil, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: &lastVersionRef,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(false, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, &lastVersionRef).
					Return(&domain.ReportTemplateVersion{
						Visibility: domain.VisibilityGlobal,
						CreatedBy:  email,
					}, nil).
					Once()
				f.reportTemplateDAL.On("HideReportTemplate", testutils.ContextBackgroundMock, reportTemplateID).
					Return(nil).
					Once()
			},
		},
		{
			name: "successful deletion of private template with owner",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, txNil, reportTemplateID).
					Return(&domain.ReportTemplate{
						ActiveVersion: &activeVersionRef,
						LastVersion:   &lastVersionRef,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(false, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, &lastVersionRef).
					Return(&domain.ReportTemplateVersion{
						Visibility: domain.VisibilityPrivate,
						CreatedBy:  email,
					}, nil).
					Once()
				f.reportTemplateDAL.On("HideReportTemplate", testutils.ContextBackgroundMock, reportTemplateID).
					Return(nil).
					Once()
			},
		},
		{
			name: "error deleting private template not with owner",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr:     true,
			expectedErr: domain.ErrUnauthorizedDelete,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, txNil, reportTemplateID).
					Return(&domain.ReportTemplate{
						ActiveVersion: &activeVersionRef,
						LastVersion:   &lastVersionRef,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(false, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, &lastVersionRef).
					Return(&domain.ReportTemplateVersion{
						Visibility: domain.VisibilityPrivate,
					}, nil).
					Once()
			},
		},
		{
			name: "error deleting approved template with owner",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr:     true,
			expectedErr: domain.ErrUnauthorizedDelete,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, txNil, reportTemplateID).
					Return(&domain.ReportTemplate{
						ActiveVersion: &activeVersionRef,
						LastVersion:   &lastVersionRef,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(false, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, &lastVersionRef).
					Return(&domain.ReportTemplateVersion{
						Visibility: domain.VisibilityGlobal,
						CreatedBy:  email,
					}, nil).
					Once()
			},
		},
		{
			name: "error requester not owner and not admin",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr:     true,
			expectedErr: domain.ErrUnauthorizedDelete,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, txNil, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: &lastVersionRef,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(false, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, &lastVersionRef).
					Return(&domain.ReportTemplateVersion{
						Visibility: domain.VisibilityGlobal,
					}, nil).
					Once()
			},
		},
		{
			name: "error retrieving report template",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr:     true,
			expectedErr: errorRetrievingReportTemplate,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, txNil, reportTemplateID).
					Return(nil, errorRetrievingReportTemplate).
					Once()
			},
		},
		{
			name: "error deleting report template",
			args: args{
				ctx:              ctx,
				requesterEmail:   email,
				reportTemplateID: reportTemplateID,
			},
			wantErr:     true,
			expectedErr: errorDeletingReportTemplate,
			on: func(f *fields) {
				f.reportTemplateDAL.On("Get", testutils.ContextBackgroundMock, txNil, reportTemplateID).
					Return(&domain.ReportTemplate{
						LastVersion: &lastVersionRef,
					}, nil).
					Once()
				f.reportTemplateDAL.On("GetVersionByRef", testutils.ContextBackgroundMock, &lastVersionRef).
					Return(&domain.ReportTemplateVersion{
						Visibility: domain.VisibilityPrivate,
						CreatedBy:  email,
					}, nil).
					Once()
				f.employeeService.On("CheckDoiTEmployeeRole", testutils.ContextBackgroundMock, templateLibraryAdminRole, email).
					Return(false, nil).
					Once()
				f.reportTemplateDAL.On("HideReportTemplate", testutils.ContextBackgroundMock, reportTemplateID).
					Return(errorDeletingReportTemplate).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:      logger.FromContext,
				employeeService:     &doitEmployeesMock.ServiceInterface{},
				reportTemplateDAL:   &mocks.ReportTemplateFirestore{},
				attributionDAL:      &attributionsMock.Attributions{},
				attributionGroupDAL: &attributionGroupsMock.AttributionGroups{},
			}

			s := &ReportTemplateService{
				loggerProvider:      tt.fields.loggerProvider,
				employeeService:     tt.fields.employeeService,
				reportTemplateDAL:   tt.fields.reportTemplateDAL,
				attributionDAL:      tt.fields.attributionDAL,
				attributionGroupDAL: tt.fields.attributionGroupDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.DeleteReportTemplate(ctx, tt.args.requesterEmail, tt.args.reportTemplateID)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportTemplateService.DeleteReportTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportTemplateService.DeleteReportTemplate() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}
