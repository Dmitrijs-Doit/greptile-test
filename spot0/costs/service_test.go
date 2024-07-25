package costs

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/iterator"

	"github.com/doitintl/bigquery/iface"
	bigQueryMock "github.com/doitintl/bigquery/mocks"
	mpaFsMocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	testTools "github.com/doitintl/hello/scheduled-tasks/common/test_tools"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	bq "github.com/doitintl/hello/scheduled-tasks/spot0/dal/bigquery"
	bigQueryCostMock "github.com/doitintl/hello/scheduled-tasks/spot0/dal/bigquery/mocks"
	fs "github.com/doitintl/hello/scheduled-tasks/spot0/dal/firestore"
	fireStoreCostMock "github.com/doitintl/hello/scheduled-tasks/spot0/dal/firestore/mocks"
)

func NewSpotScalingCostsServiceTest(t *testing.T, excludeMocks map[string]string) (*SpotZeroCostsService, *bigQueryCostMock.ISpot0CostsBigQuery, *fireStoreCostMock.ISpot0CostsFireStore, *mpaFsMocks.MasterPayerAccounts) {
	bqMock := new(bigQueryCostMock.ISpot0CostsBigQuery)
	fsMock := new(fireStoreCostMock.ISpot0CostsFireStore)
	mpaFsMock := new(mpaFsMocks.MasterPayerAccounts)
	loggerProvider := logger.FromContext
	s := SpotZeroCostsService{
		loggerProvider,
		nil,
		bqMock,
		fsMock,
		mpaFsMock,
	}

	if _, ok := excludeMocks["GetMonthlyUsage"]; !ok {
		var asgMonthlyUsage []*bq.AsgMonthlyUsage

		err := testTools.ConvertJSONFileIntoStruct("testdata", "daily_usage_results.json", &asgMonthlyUsage)
		if err != nil {
			t.Fatalf("could not convert json test file into struct. error %s", err)
		}

		bqMock.On("GetMonthlyUsage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(func() iface.RowIterator {
				q := &bigQueryMock.RowIterator{}
				q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

					arg := args.Get(0).(*bq.AsgMonthlyUsage)
					*arg = *asgMonthlyUsage[0]

				}).Once()
				q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

					arg := args.Get(0).(*bq.AsgMonthlyUsage)
					*arg = *asgMonthlyUsage[1]

				}).Once()
				q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

					arg := args.Get(0).(*bq.AsgMonthlyUsage)
					*arg = *asgMonthlyUsage[2]

				}).Once()
				q.On("Next", mock.Anything).Return(iterator.Done).Once()
				return q
			}(), nil).Once()
	}

	if _, ok := excludeMocks["AggregateDailySavings"]; !ok {
		bqMock.On("AggregateDailySavings", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	}

	if _, ok := excludeMocks["GetNonBillingTagsAsg"]; !ok {
		bqMock.On("GetNonBillingTagsAsg", mock.Anything, mock.Anything).Return(
			[]*bq.NonBillingAsg{
				{
					PrimaryDomain: "abc.com",
					Account:       "123456",
					Region:        "us-east-1",
					AsgName:       "abc_asg1",
					NoRootAccess:  false,
				},
				{
					PrimaryDomain: "abc.com",
					Account:       "123456",
					Region:        "us-east-1",
					AsgName:       "abc_asg2",
					NoRootAccess:  false,
				},
				{
					PrimaryDomain: "def.com",
					Account:       "23456",
					Region:        "us-east-1",
					AsgName:       "def_asg1",
					NoRootAccess:  false,
				},
			}, nil)
	}

	if _, ok := excludeMocks["UpdateASGsUsage"]; !ok {
		fsMock.On("UpdateASGsUsage", mock.Anything, mock.Anything).Return(nil)
	}

	if _, ok := excludeMocks["GetMPAWithoutRootAccess"]; !ok {
		mpaFsMock.On("GetMPAWithoutRootAccess", mock.Anything, mock.Anything).Return(
			map[string]bool{
				"123456": true,
			}, nil)
	}

	return &s, bqMock, fsMock, mpaFsMock
}

func TestSpotScalingDailyCosts(t *testing.T) {
	ctx := context.Background()

	t.Run("SpotScalingDailyCosts", func(t *testing.T) {
		s, bqMocks, fsMocks, _ := NewSpotScalingCostsServiceTest(t, nil)
		err := s.SpotScalingDailyCosts(ctx, "", "", "", "", "")
		assert.NoError(t, err)
		assert.Equal(t, 2, len(bqMocks.Mock.Calls))

		// length of data is 3
		fsMocks.Mock.AssertNumberOfCalls(t, "UpdateASGsUsage", 3)

		// assert that spot and on demand instances are formatted correctly
		// one on demand instance
		updateUsageCallUsageDoc1 := fsMocks.Mock.Calls[0].Arguments.Get(1).(fs.UsageDoc)
		assert.Equal(t, 0, len(updateUsageCallUsageDoc1.Usage.SpotInstances.Instances))
		assert.Equal(t, 1, len(updateUsageCallUsageDoc1.Usage.OnDemandInstances.Instances))
		// four spot instances
		updateUsageCallUsageDoc2 := fsMocks.Mock.Calls[1].Arguments.Get(1).(fs.UsageDoc)
		assert.Equal(t, 4, len(updateUsageCallUsageDoc2.Usage.SpotInstances.Instances))
		assert.Equal(t, 0, len(updateUsageCallUsageDoc2.Usage.OnDemandInstances.Instances))
		// one on demand, four on spot instances
		updateUsageCallUsageDoc3 := fsMocks.Mock.Calls[2].Arguments.Get(1).(fs.UsageDoc)
		assert.Equal(t, 4, len(updateUsageCallUsageDoc3.Usage.SpotInstances.Instances))
		assert.Equal(t, 1, len(updateUsageCallUsageDoc3.Usage.OnDemandInstances.Instances))
	})
}

func TestSpotScalingMonthlyCosts(t *testing.T) {
	ctx := context.Background()

	t.Run("SpotScalingDailyCosts", func(t *testing.T) {
		s, bqMocks, fsMocks, _ := NewSpotScalingCostsServiceTest(t, nil)
		err := s.SpotScalingMonthlyCosts(ctx, "", "", "")
		assert.NoError(t, err)
		assert.Equal(t, 1, len(bqMocks.Mock.Calls))

		// length of data is 3
		fsMocks.Mock.AssertNumberOfCalls(t, "UpdateASGsUsage", 3)

		// assert that spot and on demand instances are formatted correctly
		// one on demand instance
		updateUsageCallUsageDoc1 := fsMocks.Mock.Calls[0].Arguments.Get(1).(fs.UsageDoc)
		assert.Equal(t, 0, len(updateUsageCallUsageDoc1.Usage.SpotInstances.Instances))
		assert.Equal(t, 1, len(updateUsageCallUsageDoc1.Usage.OnDemandInstances.Instances))
		// four spot instances
		updateUsageCallUsageDoc2 := fsMocks.Mock.Calls[1].Arguments.Get(1).(fs.UsageDoc)
		assert.Equal(t, 4, len(updateUsageCallUsageDoc2.Usage.SpotInstances.Instances))
		assert.Equal(t, 0, len(updateUsageCallUsageDoc2.Usage.OnDemandInstances.Instances))
		// one on demand, four on spot instances
		updateUsageCallUsageDoc3 := fsMocks.Mock.Calls[2].Arguments.Get(1).(fs.UsageDoc)
		assert.Equal(t, 4, len(updateUsageCallUsageDoc3.Usage.SpotInstances.Instances))
		assert.Equal(t, 1, len(updateUsageCallUsageDoc3.Usage.OnDemandInstances.Instances))
	})
}

func Test_convertToUsageDoc(t *testing.T) {
	type args struct {
		row bq.AsgMonthlyUsage
	}

	tests := []struct {
		name string
		args args
		want fs.UsageDoc
	}{
		{
			name: "one spot and one on-demand instance",
			args: args{row: bq.AsgMonthlyUsage{
				BillingYear:              "2020",
				BillingMonth:             "09",
				DocID:                    "docId",
				CurMonthSpotSpending:     3,
				CurMonthSpotHours:        10,
				CurMonthOnDemandSpending: 30,
				CurMonthOnDemandHours:    5,
				CurMonthTotalSavings:     17,
				InstanceDetails: []bq.InstanceDetail{
					{
						InstanceType:             "t2.micro",
						CurMonthSpotSpending:     3,
						CurMonthSpotHours:        10,
						CurMonthOnDemandSpending: 0,
						CurMonthOnDemandHours:    0,
						OnDemandCost:             20,
						Platform:                 "Linux/UNIX",
					},
					{
						InstanceType:             "a1.medium",
						CurMonthSpotSpending:     0,
						CurMonthSpotHours:        0,
						CurMonthOnDemandSpending: 30,
						CurMonthOnDemandHours:    5,
						OnDemandCost:             30,
						Platform:                 "Linux/UNIX",
					},
				},
			}},
			want: fs.UsageDoc{
				DocID:        "docId",
				YearMonthKey: "2020_9",
				Usage: fs.Usage{
					TotalSavings: 17,
					SpotInstances: fs.InstancesSummary{
						TotalCost:  3,
						TotalHours: 10,
						Instances: []*fs.InstanceSummary{
							{
								Cost:         3,
								AmountHours:  10,
								InstanceType: "t2.micro",
								Platform:     "Linux/UNIX",
								OnDemandCost: 20,
							},
						},
					},
					OnDemandInstances: fs.InstancesSummary{
						TotalCost:  30,
						TotalHours: 5,
						Instances: []*fs.InstanceSummary{
							{
								Cost:         30,
								AmountHours:  5,
								InstanceType: "a1.medium",
								Platform:     "Linux/UNIX",
								OnDemandCost: 30,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertToUsageDoc(tt.args.row); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertToUsageDoc() = %v, want %v", got, tt.want)
			}
		})
	}
}
