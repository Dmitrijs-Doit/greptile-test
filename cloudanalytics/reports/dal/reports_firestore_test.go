package dal

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/mocks"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	labelsDALIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
	labelsDALMocks "github.com/doitintl/hello/scheduled-tasks/labels/dal/mocks"
	testPackage "github.com/doitintl/tests"
)

func setupReports() (*ReportsFirestore, *mocks.DocumentsHandler) {
	fs, err := firestore.NewClient(context.Background(),
		common.TestProjectID,
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		panic(err)
	}

	dh := &mocks.DocumentsHandler{}

	return &ReportsFirestore{
		firestoreClientFun: func(ctx context.Context) *firestore.Client {
			return fs
		},
		documentsHandler: dh,
	}, dh
}

func TestNewFirestoreReportsDAL(t *testing.T) {
	_, err := NewReportsFirestore(context.Background(), common.TestProjectID)
	assert.NoError(t, err)

	d := NewReportsFirestoreWithClient(nil)
	assert.NotNil(t, d)
}

func TestReportsDAL_Get(t *testing.T) {
	ctx := context.Background()
	d, dh := setupReports()

	dh.
		On("Get", mock.Anything, mock.AnythingOfType("*firestore.DocumentRef")).
		Return(func() iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(nil)
			snap.On("ID").Return(mock.Anything)
			return snap
		}(), nil).
		Once()

	r, err := d.Get(ctx, "testReportId")
	assert.NoError(t, err)
	assert.NotNil(t, r)

	dh.
		On("Get", mock.Anything, mock.AnythingOfType("*firestore.DocumentRef")).
		Return(func() iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(fmt.Errorf("fail"))
			return snap
		}(), nil).
		Once()

	r, err = d.Get(ctx, "testReportId")
	assert.Nil(t, r)
	assert.Error(t, err)

	dh.
		On("Get", mock.Anything, mock.AnythingOfType("*firestore.DocumentRef")).
		Return(nil, fmt.Errorf("fail")).
		Once()

	r, err = d.Get(ctx, "testReportId")
	assert.Nil(t, r)
	assert.Error(t, err)

	dh.
		On("Get", mock.Anything, mock.AnythingOfType("*firestore.DocumentRef")).
		Return(nil, status.Error(codes.NotFound, "report not found, should fail")).
		Once()

	r, err = d.Get(ctx, "testReportId")
	assert.Nil(t, r)
	assert.ErrorIs(t, err, doitFirestore.ErrNotFound)

	r, err = d.Get(ctx, "")
	assert.Nil(t, r)
	assert.Error(t, err, "invalid report id")
}

func TestReportsDAL_Create(t *testing.T) {
	ctx := context.Background()

	reportsFirestoreDAL, err := NewReportsFirestore(ctx, "doitintl-cmp-dev")
	if err != nil {
		t.Error(err)
	}

	type args struct {
		ctx    context.Context
		report *report.Report
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "create report",
			args: args{
				ctx:    ctx,
				report: report.NewDefaultReport(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createdReport, err := reportsFirestoreDAL.Create(
				tt.args.ctx,
				nil,
				tt.args.report,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("reportsFirestoreDAL.NewReportsFirestore() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("reportsFirestoreDAL.NewReportsFirestore() err = %v, expectedErr %v", err, tt.expectedErr)
			}

			if !tt.wantErr {
				savedReport, err := reportsFirestoreDAL.Get(tt.args.ctx, createdReport.ID)
				if err != nil {
					t.Errorf("error fetching report during check, err = %v", err)
				}

				assert.Equal(t, savedReport.ID, createdReport.ID)
				assert.Equal(t, savedReport.Name, createdReport.Name)
			}
		})
	}
}

func TestReportsDAL_Update(t *testing.T) {
	ctx := context.Background()

	reportsFirestoreDAL, err := NewReportsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	type args struct {
		ctx      context.Context
		reportID string
		report   *report.Report
	}

	if err := testPackage.LoadTestData("Reports"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "update report",
			args: args{
				ctx:      ctx,
				reportID: "8mhLwxdZylr30vyVHiQE",
				report:   report.NewDefaultReport(),
			},
			wantErr: false,
		},
		{
			name: "fail on trying to update non-existing report",
			args: args{
				ctx:      ctx,
				reportID: "non-existing-report-id",
				report:   report.NewDefaultReport(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reportsFirestoreDAL.Update(
				tt.args.ctx,
				tt.args.reportID,
				tt.args.report,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("reportsFirestoreDAL.Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("reportsFirestoreDAL.Update() err = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestReportsFirestore_GetCustomerReports(t *testing.T) {
	type args struct {
		ctx        context.Context
		customerID string
	}

	if err := testPackage.LoadTestData("Reports"); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	d, err := NewReportsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "err on empty customerID",
			args: args{
				ctx:        ctx,
				customerID: "",
			},
			wantErr:     true,
			expectedErr: errors.New("invalid customer id"),
		},
		{
			name: "success getting reports for customer",
			args: args{
				ctx:        ctx,
				customerID: "customerID",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.GetCustomerReports(tt.args.ctx, tt.args.customerID)
			if err != nil {
				if !tt.wantErr || err.Error() != tt.expectedErr.Error() {
					t.Errorf("ReportsFirestore.GetCustomerReports() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				return
			}

			assert.Equal(t, 2, len(got))

			reportIDs := []string{"8mhLwxdZylr30vyVHiQE", "bnKYrmSLMPUBeblwUKrk"}
			for i, report := range got {
				assert.Equal(t, got[i].ID, reportIDs[i])
				assert.Equal(t, report.Customer.ID, "customerID")
				assert.Equal(t, report.Draft, false)
				assert.Equal(t, report.Name, "chaim test")
			}
		})
	}
}

var ctx = context.Background()

func NewFirestoreWithMockLabels(labelsMock labelsDALIface.Labels) (*ReportsFirestore, error) {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	fun := func(ctx context.Context) *firestore.Client {
		return fs
	}

	return &ReportsFirestore{
		firestoreClientFun: fun,
		labelsDAL:          labelsMock,
	}, nil
}

func TestReportsFirestore_DeleteReport(t *testing.T) {
	type fields struct {
		labelsDal *labelsDALMocks.Labels
	}

	type args struct {
		ctx      context.Context
		reportID string
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
		fields      fields
		on          func(f *fields)
	}{
		{
			name: "err on empty reportID",
			args: args{
				ctx:      ctx,
				reportID: "",
			},
			wantErr:     true,
			expectedErr: ErrInvalidReportID,
		},
		{
			name: "success deleting report",
			args: args{
				ctx:      ctx,
				reportID: "bnKYrmSLMPUBeblwUKrk",
			},
			wantErr: false,
			on: func(f *fields) {
				f.labelsDal.On("DeleteObjectWithLabels", ctx, mock.AnythingOfType("*firestore.DocumentRef")).Return(nil)
			},
		},
		{
			name: "error - delete object with labels error",
			args: args{
				ctx:      ctx,
				reportID: "bnKYrmSLMPUBeblwUKrk",
			},
			wantErr:     true,
			expectedErr: errors.New("error"),
			on: func(f *fields) {
				f.labelsDal.On("DeleteObjectWithLabels", ctx, mock.AnythingOfType("*firestore.DocumentRef")).Return(errors.New("error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				labelsDal: &labelsDALMocks.Labels{},
			}

			d, err := NewFirestoreWithMockLabels(tt.fields.labelsDal)
			if err != nil {
				t.Error(err)
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err = d.Delete(tt.args.ctx, tt.args.reportID)
			if err != nil {
				if !tt.wantErr || err.Error() != tt.expectedErr.Error() {
					t.Errorf("ReportsFirestore.DeleteReport() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				return
			}
		})
	}
}

var (
	testLastTimeRunTime = time.Date(2023, time.May, 22, 3, 14, 16, 0, time.UTC)
)

func TestReportsDAL_UpdateTimeLastRun(t *testing.T) {
	ctx := context.Background()

	reportsFirestoreDAL, err := NewReportsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	reportsFirestoreDAL.timeFunc = testTimeFunc

	type args struct {
		ctx      context.Context
		reportID string
		key      domainOrigin.QueryOrigin
	}

	if err := testPackage.LoadTestData("Reports"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		args    args
		want    map[string]*time.Time
		wantErr bool
	}{
		{
			name: "update last time run",
			args: args{
				ctx:      ctx,
				reportID: "8mhLwxdZylr30vyVHiQE",
				key:      domainOrigin.QueryOriginClient,
			},
			want: map[string]*time.Time{
				domainOrigin.QueryOriginClient: &testLastTimeRunTime,
			},
		},
		{
			name: "fail on trying to update last time run on non-existing report",
			args: args{
				ctx:      ctx,
				reportID: "non-existing-report-id",
				key:      domainOrigin.QueryOriginReportsAPI,
			},
			wantErr: true,
		},
		{
			name: "key is not allowed",
			args: args{
				ctx:      ctx,
				reportID: "8mhLwxdZylr30vyVHiQE",
				key:      domainOrigin.QueryOriginAlerts,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reportsFirestoreDAL.UpdateTimeLastRun(
				tt.args.ctx,
				tt.args.reportID,
				tt.args.key,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("reportsFirestoreDAL.UpdateTimeLastRun() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				report, err := reportsFirestoreDAL.Get(ctx, tt.args.reportID)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, tt.want, report.TimeLastRun)
			}
		})
	}
}

func TestReportsDAL_UpdateStats(t *testing.T) {
	ctx := context.Background()

	reportsFirestoreDAL, err := NewReportsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	type args struct {
		ctx                 context.Context
		reportID            string
		origin              domainOrigin.QueryOrigin
		serverDurationMs    *int64
		totalBytesProcessed *int64
	}

	if err := testPackage.LoadTestData("Reports"); err != nil {
		t.Fatal(err)
	}

	serverDurationMs := int64(12)
	totalBytesProcessed := int64(25)

	expectedReportStat := report.Stat{
		ServerDurationMs:    &serverDurationMs,
		TotalBytesProcessed: &totalBytesProcessed,
	}

	tests := []struct {
		name    string
		args    args
		want    map[domainOrigin.QueryOrigin]*report.Stat
		wantErr bool
	}{
		{
			name: "update stats",
			args: args{
				ctx:                 ctx,
				reportID:            "8mhLwxdZylr30vyVHiQE",
				origin:              domainOrigin.QueryOriginReportsAPI,
				serverDurationMs:    &serverDurationMs,
				totalBytesProcessed: &totalBytesProcessed,
			},
			want: map[domainOrigin.QueryOrigin]*report.Stat{
				domainOrigin.QueryOriginReportsAPI: &expectedReportStat,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reportsFirestoreDAL.UpdateStats(
				tt.args.ctx,
				tt.args.reportID,
				tt.args.origin,
				tt.args.serverDurationMs,
				tt.args.totalBytesProcessed,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("reportsFirestoreDAL.UpdateStats() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				report, err := reportsFirestoreDAL.Get(ctx, tt.args.reportID)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, tt.want, report.Stats)
			}
		})
	}
}

func testTimeFunc() time.Time {
	return testLastTimeRunTime
}
