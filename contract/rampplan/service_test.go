package rampplan

import (
	"context"
	_ "embed"
	"encoding/json"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/firestore/pkg"
	analyticsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/mocks"
	agMocks "github.com/doitintl/hello/scheduled-tasks/contract/attributiongroup/dal/mocks"
	cMocks "github.com/doitintl/hello/scheduled-tasks/contract/dal/mocks"
	rpMocks "github.com/doitintl/hello/scheduled-tasks/contract/rampplan/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

//go:embed testData/contract.json
var contractJSON []byte

//go:embed testData/expectedRampPlan.json
var expecterRampPlanJSON []byte

func TestService_CreateRampPlan(t *testing.T) {
	t.Run("ramp plan should be created correctly", func(t *testing.T) {
		ctx := context.Background()

		var contract pkg.Contract
		if err := json.Unmarshal(contractJSON, &contract); err != nil {
			t.Fatal(err)
		}

		var log *logger.Logging

		contractsMock := cMocks.NewContractFirestore(t)
		contractsMock.On("GetCustomerContractByID", mock.Anything, mock.Anything, mock.Anything).Return(&contract, nil)

		attributionMocks := agMocks.NewAttributionGroup(t)
		attributionMocks.On("GetRampPlanEligibleSpendAttributionGroup", mock.Anything).Return([]*firestore.DocumentSnapshot{{Ref: &firestore.DocumentRef{}}}, nil)

		rampPlanMocks := rpMocks.NewRampPlans(t)
		rampPlanMocks.On("AddRampPlan", mock.Anything, mock.Anything).Return(&firestore.DocumentRef{ID: "12345"}, nil, nil)

		service := &Service{
			log,
			nil,
			&analyticsMocks.CloudAnalytics{},
			rampPlanMocks,
			contractsMock,
			attributionMocks,
		}

		err := service.CreateRampPlan(ctx, "ABCD", "EFGH", "")
		assert.NoError(t, err)

		rampPlan := (rampPlanMocks.Mock.Calls[0].Arguments[1]).(*pkg.RampPlan)

		var expectedRampPlan pkg.RampPlan
		if err := json.Unmarshal(expecterRampPlanJSON, &expectedRampPlan); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, expectedRampPlan.CommitmentPeriods, rampPlan.CommitmentPeriods)
		assert.Equal(t, expectedRampPlan.Name, rampPlan.Name)
		assert.Equal(t, expectedRampPlan.TargetAmount, rampPlan.TargetAmount)
	})
}
