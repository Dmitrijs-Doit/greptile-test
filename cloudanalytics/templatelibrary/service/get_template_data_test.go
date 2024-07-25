package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
)

func TestReportTemplateService_GetTemplateData(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		reportTemplateDAL *mocks.ReportTemplateFirestore
	}

	type args struct {
		ctx            context.Context
		isDoitEmployee bool
	}

	activeVersionRef := firestore.DocumentRef{
		ID: "activeVersionRef",
	}

	lastVersionRef := firestore.DocumentRef{
		ID: "lastVersionRef",
	}

	template := domain.ReportTemplate{
		ActiveVersion: &activeVersionRef,
		LastVersion:   &lastVersionRef,
	}

	expectedDoerVersion := domain.ReportTemplateVersion{
		ID: "lastVersionRef",
	}

	expectedNonDoerVersion := domain.ReportTemplateVersion{
		ID: "activeVersionRef",
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		on      func(*fields)
		wantErr bool
	}{
		{
			name: "success for doit employee",
			args: args{
				ctx:            ctx,
				isDoitEmployee: true,
			},
			on: func(f *fields) {
				f.reportTemplateDAL.On("GetTemplates", ctx).Return([]domain.ReportTemplate{template}, nil)
				f.reportTemplateDAL.On("GetVersions", ctx, []*firestore.DocumentRef{&lastVersionRef}).Return([]domain.ReportTemplateVersion{expectedDoerVersion}, nil)
			},
			wantErr: false,
		},
		{
			name: "success for non-doit employee",
			args: args{
				ctx:            ctx,
				isDoitEmployee: false,
			},
			on: func(f *fields) {
				f.reportTemplateDAL.On("GetTemplates", ctx).Return([]domain.ReportTemplate{template}, nil)
				f.reportTemplateDAL.On("GetVersions", ctx, []*firestore.DocumentRef{&activeVersionRef}).Return([]domain.ReportTemplateVersion{expectedNonDoerVersion}, nil)
			},
			wantErr: false,
		},
		{
			name: "error getting templates",
			args: args{
				ctx:            ctx,
				isDoitEmployee: false,
			},
			on: func(f *fields) {
				f.reportTemplateDAL.On("GetTemplates", ctx).Return(nil, errors.New("some error fetching templates"))
			},
			wantErr: true,
		},
		{
			name: "error getting versions",
			args: args{
				ctx:            ctx,
				isDoitEmployee: false,
			},
			on: func(f *fields) {
				f.reportTemplateDAL.On("GetTemplates", ctx).Return([]domain.ReportTemplate{template}, nil)
				f.reportTemplateDAL.On("GetVersions", ctx, []*firestore.DocumentRef{&activeVersionRef}).Return(nil, errors.New("some error fetching versions"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				reportTemplateDAL: &mocks.ReportTemplateFirestore{},
			}

			s := &ReportTemplateService{
				reportTemplateDAL: tt.fields.reportTemplateDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			_, _, err := s.GetTemplateData(tt.args.ctx, tt.args.isDoitEmployee)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReportTemplateService.GetTemplateData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
