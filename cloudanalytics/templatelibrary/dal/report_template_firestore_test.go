package dal

import (
	"context"
	"errors"
	"math/rand"
	"strconv"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	testPackage "github.com/doitintl/tests"
)

var (
	reportTemplateID = "GTBCyzg709tvqlV2Gj54"
	TemplateLibrary  = "TemplateLibrary"
	TemplateVersions = "TemplateVersions"
)

func TestReportTemplateDAL_Get(t *testing.T) {
	ctx := context.Background()

	reportTemplateFirestoreDAL, err := NewReportTemplateFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	type args struct {
		ctx              context.Context
		reportTemplateID string
	}

	if err := testPackage.LoadTestData(TemplateLibrary); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "err on empty reportTemplateID",
			args: args{
				ctx:              ctx,
				reportTemplateID: "",
			},
			wantErr:     true,
			expectedErr: domain.ErrInvalidReportTemplateID,
		},
		{
			name: "fail on getting non-existing reportTemplateID",
			args: args{
				ctx:              ctx,
				reportTemplateID: "non-existing-id",
			},
			wantErr:     true,
			expectedErr: doitFirestore.ErrNotFound,
		},
		{
			name: "success getting reportTemplate",
			args: args{
				ctx:              ctx,
				reportTemplateID: reportTemplateID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reportTemplate, err := reportTemplateFirestoreDAL.Get(tt.args.ctx, nil, tt.args.reportTemplateID)

			if (err != nil) != tt.wantErr {
				t.Errorf("reportTemplateFirestoreDAL.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("reportTemplateFirestoreDAL.Get() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if !tt.wantErr {
				assert.Equal(t, reportTemplate.ID, reportTemplateID)
			}
		})
	}
}

func TestReportTemplateDAL_Create(t *testing.T) {
	ctx := context.Background()
	email := "test@doit.com"

	reportTemplateFirestoreDAL, err := NewReportTemplateFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	type args struct {
		ctx            context.Context
		tx             *firestore.Transaction
		email          string
		reportTemplate *domain.ReportTemplate
	}

	reportTemplate := domain.NewDefaultReportTemplate()
	reportTemplate.Hidden = true

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "create report template",
			args: args{
				ctx:            ctx,
				email:          email,
				reportTemplate: reportTemplate,
			},
			wantErr: false,
		},
		{
			name: "error no report template",
			args: args{
				ctx:   ctx,
				email: email,
			},
			wantErr:     true,
			expectedErr: domain.ErrInvalidReportTemplate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createdReportTemplateRef, err := reportTemplateFirestoreDAL.CreateReportTemplate(
				tt.args.ctx,
				tt.args.tx,
				tt.args.reportTemplate,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("reportTemplateFirestoreDAL.CreateReportTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("reportTemplateFirestoreDAL.CreateReportTemplate() err = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if !tt.wantErr {
				savedReportTemplate, err := reportTemplateFirestoreDAL.Get(tt.args.ctx, nil, createdReportTemplateRef.ID)
				if err != nil {
					t.Errorf("error fetching report template during check, err = %v", err)
					return
				}

				assert.Equal(t, savedReportTemplate.ID, createdReportTemplateRef.ID)
			}
		})
	}
}

func TestReportTemplateDAL_Update(t *testing.T) {
	ctx := context.Background()

	reportTemplateFirestoreDAL, err := NewReportTemplateFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	type args struct {
		ctx              context.Context
		tx               *firestore.Transaction
		reportTemplateID string
		reportTemplate   *domain.ReportTemplate
	}

	if err := testPackage.LoadTestData(TemplateLibrary); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "update report template",
			args: args{
				ctx:              ctx,
				reportTemplateID: reportTemplateID,
				reportTemplate:   domain.NewDefaultReportTemplate(),
			},
			wantErr: false,
		},
		{
			name: "fail on trying to update non-existing report template",
			args: args{
				ctx:              ctx,
				reportTemplateID: "non-existing-id",
				reportTemplate:   domain.NewDefaultReportTemplate(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reportTemplateFirestoreDAL.UpdateReportTemplate(
				tt.args.ctx,
				tt.args.tx,
				tt.args.reportTemplateID,
				tt.args.reportTemplate,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("reportTemplateFirestoreDAL.UpdateReportTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("reportTemplateFirestoreDAL.UpdateReportTemplate() err = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
		})
	}
}

func TestReportTemplateDAL_Delete(t *testing.T) {
	ctx := context.Background()

	reportTemplateFirestoreDAL, err := NewReportTemplateFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	type args struct {
		ctx              context.Context
		reportTemplateID string
	}

	if err := testPackage.LoadTestData(TemplateLibrary); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "err on empty reportTemplateID",
			args: args{
				ctx:              ctx,
				reportTemplateID: "",
			},
			wantErr:     true,
			expectedErr: domain.ErrInvalidReportTemplateID,
		},
		{
			name: "success deleting reportTemplate",
			args: args{
				ctx:              ctx,
				reportTemplateID: reportTemplateID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reportTemplateFirestoreDAL.DeleteReportTemplate(tt.args.ctx, tt.args.reportTemplateID)
			if err != nil {
				if !tt.wantErr || err.Error() != tt.expectedErr.Error() {
					t.Errorf("ReportTemplateDAL.Delete() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				return
			}

			_, err = reportTemplateFirestoreDAL.Get(ctx, nil, tt.args.reportTemplateID)
			assert.ErrorIs(t, err, doitFirestore.ErrNotFound)
		})
	}
}

func TestReportTemplateDAL_Hide(t *testing.T) {
	ctx := context.Background()

	reportTemplateFirestoreDAL, err := NewReportTemplateFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	type args struct {
		ctx              context.Context
		reportTemplateID string
	}

	if err := testPackage.LoadTestData(TemplateLibrary); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "err on empty reportTemplateID",
			args: args{
				ctx:              ctx,
				reportTemplateID: "",
			},
			wantErr:     true,
			expectedErr: domain.ErrInvalidReportTemplateID,
		},
		{
			name: "successfully hide reportTemplate",
			args: args{
				ctx:              ctx,
				reportTemplateID: reportTemplateID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reportTemplateFirestoreDAL.HideReportTemplate(tt.args.ctx, tt.args.reportTemplateID)
			if err != nil {
				if !tt.wantErr || err.Error() != tt.expectedErr.Error() {
					t.Errorf("ReportTemplateDAL.Hide() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				return
			}

			reportTemplate, _ := reportTemplateFirestoreDAL.Get(ctx, nil, tt.args.reportTemplateID)
			assert.Equal(t, true, reportTemplate.Hidden)
		})
	}
}

func TestReportTemplateDAL_CreateVersion(t *testing.T) {
	ctx := context.Background()

	reportTemplateFirestoreDAL, err := NewReportTemplateFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	type args struct {
		ctx                     context.Context
		reportTemplateVersionID string
		reportTemplateID        string
		reportTemplateVersion   *domain.ReportTemplateVersion
	}

	reportTemplateVersionID := strconv.Itoa(rand.Intn(100))

	reportTemplateVersion := domain.ReportTemplateVersion{}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "create report template version",
			args: args{
				ctx:                     ctx,
				reportTemplateVersionID: "0",
				reportTemplateID:        "some_report_template_id",
				reportTemplateVersion:   &reportTemplateVersion,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createdReportVersionRef, err := reportTemplateFirestoreDAL.CreateVersion(
				tt.args.ctx,
				nil,
				reportTemplateVersionID,
				tt.args.reportTemplateID,
				tt.args.reportTemplateVersion,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("reportTemplateFirestoreDAL.CreateVersion() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("reportTemplateFirestoreDAL.CreateVersion() err = %v, expectedErr %v", err, tt.expectedErr)
			}

			if !tt.wantErr {
				savedVersionByRef, err := reportTemplateFirestoreDAL.GetVersionByRef(tt.args.ctx, createdReportVersionRef)
				if err != nil {
					t.Errorf("error fetching report template during check, err = %v", err)
				}

				assert.Equal(t, savedVersionByRef.ID, createdReportVersionRef.ID)
			}
		})
	}
}

func TestReportTemplateDAL_GetTemplates(t *testing.T) {
	ctx := context.Background()

	reportTemplateFirestoreDAL, err := NewReportTemplateFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Fatal(err)
	}

	if err := testPackage.LoadTestData(TemplateLibrary); err != nil {
		t.Fatal(err)
	}

	reportTemplates, err := reportTemplateFirestoreDAL.GetTemplates(ctx)
	if err != nil {
		t.Errorf("error fetching report templates, err = %v", err)
		return
	}

	assert.Equal(t, 2, len(reportTemplates))
}

func TestReportTemplateDAL_GetVersions(t *testing.T) {
	ctx := context.Background()

	reportTemplateFirestoreDAL, err := NewReportTemplateFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Fatal(err)
	}

	if err := testPackage.LoadTestData(TemplateLibrary); err != nil {
		t.Fatal(err)
	}

	if err := testPackage.LoadTestData(TemplateVersions); err != nil {
		t.Fatal(err)
	}

	reportTemplates, err := reportTemplateFirestoreDAL.GetTemplates(ctx)
	if err != nil {
		t.Errorf("error fetching report templates, err = %v", err)
	}

	var versionRefs []*firestore.DocumentRef

	for _, template := range reportTemplates {
		if template.ActiveVersion != nil {
			versionRefs = append(versionRefs, template.ActiveVersion)
		}
	}

	templateVersions, err := reportTemplateFirestoreDAL.GetVersions(ctx, versionRefs)
	if err != nil {
		t.Errorf("error fetching report template versions, err = %v", err)
		return
	}

	assert.Equal(t, 2, len(templateVersions))
}
