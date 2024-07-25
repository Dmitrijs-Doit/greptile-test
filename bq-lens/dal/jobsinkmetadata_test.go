package dal

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/tests"
	"github.com/zeebo/assert"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func preloadTestData(t *testing.T) {
	if err := tests.LoadTestData("BQLens"); err != nil {
		t.Fatal("failed to load test data")
	}
}
func TestJobsSinksMetadataDal_GetJobSinkMetadata(t *testing.T) {
	mockSinkID := "test-sink-id"
	mockCustomerID := "test-customer-id"
	mockAccountID := "test-service-account-id"

	type args struct {
		ctx   context.Context
		jobID string
	}

	tests := []struct {
		name    string
		args    args
		want    pkg.SinkMetadata
		wantErr bool
	}{
		{
			name: "Test GetJobSinkMetadata",
			args: args{
				ctx:   context.Background(),
				jobID: mockSinkID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preloadTestData(t)

			fs, err := firestore.NewClient(context.Background(),
				common.TestProjectID,
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
			)
			assert.NoError(t, err)

			d := NewJobsSinksMetadataDal(fs)

			got, err := d.GetSinkMetadata(tt.args.ctx, tt.args.jobID)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardDAL.GetJobSinkMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, got.Customer.ID, mockCustomerID)
			assert.Equal(t, got.ServiceAccount.ID, mockAccountID)
		})
	}
}

func TestJobsSinksMetadataDal_DeleteSinkMetadata(t *testing.T) {
	mockSinkID := "test-sink-id"

	type args struct {
		ctx   context.Context
		jobID string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test DeleteSinkMetadata",
			args: args{
				ctx:   context.Background(),
				jobID: mockSinkID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preloadTestData(t)

			fs, err := firestore.NewClient(context.Background(),
				common.TestProjectID,
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
			)
			assert.NoError(t, err)

			d := NewJobsSinksMetadataDal(fs)

			err = d.DeleteSinkMetadata(tt.args.ctx, tt.args.jobID)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardDAL.DeleteSinkMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestJobsSinksMetadataDal_UpdateSinkProjectProgress(t *testing.T) {
	mockSinkID := "test-sink-id"
	mockProjectID := "test-project-id"

	type args struct {
		ctx      context.Context
		sinkID   string
		project  string
		progress int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test UpdateSinkProjectProgress",
			args: args{
				ctx:      context.Background(),
				sinkID:   mockSinkID,
				project:  mockProjectID,
				progress: 100,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preloadTestData(t)

			fs, err := firestore.NewClient(context.Background(),
				common.TestProjectID,
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
			)
			assert.NoError(t, err)

			d := NewJobsSinksMetadataDal(fs)

			if err := d.UpdateSinkProjectProgress(tt.args.ctx, tt.args.sinkID, tt.args.project, tt.args.progress); (err != nil) != tt.wantErr {
				t.Errorf("JobsSinksMetadataDal.UpdateSinkProjectProgress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobsSinksMetadataDal_UpdateBackfillProgress(t *testing.T) {
	mockSinkID := "test-sink-id"
	projects := []string{"test-project-id1", "test-project-id2"}

	type args struct {
		ctx                    context.Context
		sinkID                 string
		projectsToBeBackfilled []string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test SaveProjectsToBackfill",
			args: args{
				ctx:                    context.Background(),
				sinkID:                 mockSinkID,
				projectsToBeBackfilled: projects,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preloadTestData(t)

			fs, err := firestore.NewClient(context.Background(),
				common.TestProjectID,
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
			)
			assert.NoError(t, err)

			d := NewJobsSinksMetadataDal(fs)

			if err := d.UpdateBackfillProgress(tt.args.ctx, tt.args.sinkID, tt.args.projectsToBeBackfilled); (err != nil) != tt.wantErr {
				t.Errorf("JobsSinksMetadataDal.SaveProjectsToBackfill() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobsSinksMetadataDal_UpdateBackfillForProjectAndDate(t *testing.T) {
	mockSinkID := "test-sink-id"
	mockProjectID := "test-project-id"
	mockDate := time.Now()
	minTime := time.Time{}
	mockDateBackInfo := &domain.DateBackfillInfo{
		BackfillMinCreationTime: minTime,
		BackfillMaxCreationTime: time.Time{}.Add(time.Hour),
		BackfillDone:            false,
	}

	type args struct {
		ctx          context.Context
		sinkID       string
		project      string
		date         time.Time
		dateBackInfo *domain.DateBackfillInfo
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test UpdateBackfillForProjectAndDate",
			args: args{
				ctx:          context.Background(),
				sinkID:       mockSinkID,
				project:      mockProjectID,
				date:         mockDate,
				dateBackInfo: mockDateBackInfo,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preloadTestData(t)

			fs, err := firestore.NewClient(context.Background(),
				common.TestProjectID,
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
			)
			assert.NoError(t, err)

			d := NewJobsSinksMetadataDal(fs)

			if err := d.UpdateBackfillForProjectAndDate(tt.args.ctx, tt.args.sinkID, tt.args.project, tt.args.date, tt.args.dateBackInfo); (err != nil) != tt.wantErr {
				t.Errorf("JobsSinksMetadataDal.UpdateBackfillForProjectAndDate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
