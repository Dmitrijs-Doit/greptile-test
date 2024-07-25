package firestore

import (
	"context"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/zeebo/assert"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/tests"
)

func preloadTestData(t *testing.T) {
	if err := tests.LoadTestData("BQLens"); err != nil {
		t.Fatal("failed to load test data")
	}
}

func TestOnboardDAL_DeleteCostSimulationData(t *testing.T) {
	mockCustomerID := "test-customer-id"

	type args struct {
		ctx        context.Context
		customerID string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test DeleteCostSimulationData",
			args: args{
				ctx:        context.Background(),
				customerID: mockCustomerID,
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

			d := NewDAL(fs)

			err = d.DeleteCostSimulationData(tt.args.ctx, tt.args.customerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardDAL.DeleteCostSimulationData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestOnboardDAL_DeleteOptimizerData(t *testing.T) {
	mockCustomerID := "test-customer-id"

	type args struct {
		ctx        context.Context
		customerID string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test DeleteOptimizerData",
			args: args{
				ctx:        context.Background(),
				customerID: mockCustomerID,
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

			d := NewDAL(fs)

			err = d.DeleteOptimizerData(tt.args.ctx, tt.args.customerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardDAL.DeleteOptimizerData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
