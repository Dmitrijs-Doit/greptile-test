package flexsaveresold

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func Test_makeChunks(t *testing.T) {
	type args struct {
		recommendations []types.Recommendation
		size            int
	}

	tests := []struct {
		name                     string
		args                     args
		wantRecommendationChunks [][]types.Recommendation
	}{
		{
			name: "makes chunks empty list",
			args: args{
				recommendations: []types.Recommendation{},
				size:            3,
			},
			wantRecommendationChunks: [][]types.Recommendation{},
		},
		{
			name: "makes chunk if less than threshold",
			args: args{
				recommendations: []types.Recommendation{{InstanceFamily: "aa"}, {InstanceFamily: "bb"}},
				size:            3,
			},
			wantRecommendationChunks: [][]types.Recommendation{{{InstanceFamily: "aa"}, {InstanceFamily: "bb"}}},
		},
		{
			name: "makes chunks exact threshold multiple",
			args: args{
				recommendations: []types.Recommendation{{InstanceFamily: "aa"}, {InstanceFamily: "bb"}, {InstanceFamily: "cc"}},
				size:            3,
			},
			wantRecommendationChunks: [][]types.Recommendation{{{InstanceFamily: "aa"}, {InstanceFamily: "bb"}, {InstanceFamily: "cc"}}},
		},
		{
			name: "makes chunks",
			args: args{
				recommendations: []types.Recommendation{{InstanceFamily: "aa"}, {InstanceFamily: "bb"}, {InstanceFamily: "cc"}, {InstanceFamily: "dd"}, {InstanceFamily: "ee"}},
				size:            3,
			},
			wantRecommendationChunks: [][]types.Recommendation{{{InstanceFamily: "aa"}, {InstanceFamily: "bb"}, {InstanceFamily: "cc"}}, {{InstanceFamily: "dd"}, {InstanceFamily: "ee"}}},
		},
	}
	for _, tt := range tests {
		fmt.Printf("************************** \n")
		t.Run(tt.name, func(t *testing.T) {
			chunks := makeChunks(tt.args.recommendations, tt.args.size)
			assert.Equalf(t, tt.wantRecommendationChunks, chunks, "makeChunks(%v, %v)", tt.args.recommendations, tt.args.size)

			for _, chunk := range chunks {
				fmt.Printf("============ \n")

				for _, eachElement := range chunk {
					fmt.Printf("recom: %v \n", eachElement)
				}
			}
		})
	}
}

type TestableService struct {
	Service
}

func (s *TestableService) getInstanceSavingsByCustomer(ctx context.Context, pricingParams types.Recommendation, customerID string, resultChannel chan<- types.RecommendationsResultChannel, pos int) {
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	resultChannel <- types.RecommendationsResultChannel{}
}

func TestService_generateRecommendations(t *testing.T) {
	type args struct {
		ctx          context.Context
		customerID   string
		savingsInput []types.Recommendation
	}

	tests := []struct {
		name    string
		args    args
		want    []types.Recommendation
		wantErr bool
	}{
		{
			name: "generate recommendations executes",
			args: args{
				ctx:          context.Background(),
				customerID:   "100",
				savingsInput: []types.Recommendation{{}, {}, {}, {}, {}},
			},
			want:    []types.Recommendation{{}, {}, {}, {}, {}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testInstance := &TestableService{
				Service: Service{
					Logger: func(ctx context.Context) logger.ILogger {
						return logger.FromContext(ctx)
					},
				},
			}

			got, err := testInstance.generateRecommendations(testInstance, tt.args.ctx, tt.args.customerID, tt.args.savingsInput)

			assert.Equalf(t, tt.wantErr, err != nil, "generateRecommendations(%v, %v, %v)", tt.args.ctx, tt.args.customerID, tt.args.savingsInput)
			assert.Equalf(t, tt.want, got, "generateRecommendations(%v, %v, %v)", tt.args.ctx, tt.args.customerID, tt.args.savingsInput)
		})
	}
}
