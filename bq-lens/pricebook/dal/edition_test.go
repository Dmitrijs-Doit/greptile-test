package dal

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/tests"
)

func preloadTestData(t *testing.T) {
	if err := tests.LoadTestData("BQLens"); err != nil {
		t.Fatal("failed to load test data")
	}
}

func TestPricebookDAL_Get(t *testing.T) {
	type args struct {
		edition domain.Edition
	}

	tests := []struct {
		name    string
		args    args
		want    *domain.PricebookDocument
		wantErr error
	}{
		{
			name: "success standard edition",
			args: args{
				edition: domain.Standard,
			},
			want: &domain.PricebookDocument{
				string(domain.OnDemand): {
					"region1": 0.01,
					"region2": 0.05,
				},
			},
		},
		{
			name: "success enterprise edition",
			args: args{
				edition: domain.Enterprise,
			},
			want: &domain.PricebookDocument{
				string(domain.Commit1Yr): {
					"region1": 0.01,
					"region2": 0.05,
				},
			},
		},
		{
			name: "success enterprise plus edition",
			args: args{
				edition: domain.EnterprisePlus,
			},
			want: &domain.PricebookDocument{
				string(domain.Commit3Yr): {
					"region1": 0.01,
					"region2": 0.05,
				},
			},
		},
		{
			name: "not found",
			args: args{
				edition: "unknown",
			},
			wantErr: errors.New("NotFound"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preloadTestData(t)

			fs, err := firestore.NewClient(context.Background(),
				common.TestProjectID,
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			assert.NoError(t, err)

			d := NewPricebookDALWithClient(func(_ context.Context) *firestore.Client {
				return fs
			})

			got, err := d.Get(context.Background(), tt.args.edition)
			if err != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPricebookDAL_Set(t *testing.T) {
	var (
		ctx = context.Background()
	)

	type args struct {
		edition domain.Edition
		data    domain.PricebookDocument
	}

	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success standard edition",
			args: args{
				edition: domain.Standard,
				data: domain.PricebookDocument{
					string(domain.OnDemand): {
						"region1": 5.0,
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "success enterprise edition",
			args: args{
				edition: domain.Enterprise,
				data: domain.PricebookDocument{
					string(domain.Commit1Yr): {
						"region1": 5.0,
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "success enterprise plus edition",
			args: args{
				edition: domain.Enterprise,
				data: domain.PricebookDocument{
					string(domain.Commit3Yr): {
						"region1": 5.0,
					},
				},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preloadTestData(t)

			fs, err := firestore.NewClient(context.Background(),
				common.TestProjectID,
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			assert.NoError(t, err)

			d := NewPricebookDALWithClient(func(_ context.Context) *firestore.Client {
				return fs
			})

			tt.wantErr(t, d.Set(ctx, tt.args.edition, tt.args.data), fmt.Sprintf("Set(%v, %v, %v)", ctx, tt.args.edition, tt.args.data))

			got, err := d.Get(ctx, tt.args.edition)
			assert.NoError(t, err)

			assert.Equal(t, tt.args.data, *got)
		})
	}
}
