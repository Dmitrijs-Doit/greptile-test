package costs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	awsMock "github.com/doitintl/aws/mocks"
	awsPMock "github.com/doitintl/aws/providers/mocks"
	mpaFsMocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	bq "github.com/doitintl/hello/scheduled-tasks/spot0/dal/bigquery"
	bigQueryCostMock "github.com/doitintl/hello/scheduled-tasks/spot0/dal/bigquery/mocks"
	slackMocks "github.com/doitintl/hello/scheduled-tasks/spot0/dal/slack/mocks"
	doitSmMock "github.com/doitintl/secretmanager/mocks"
)

func TestUpdateCostAllocationSingleAccountTags(t *testing.T) {
	bqMock := bigQueryCostMock.NewISpot0CostsBigQuery(t)
	mpaFsMock := mpaFsMocks.NewMasterPayerAccounts(t)
	smMock := doitSmMock.NewISecretClient(t)
	loggerProvider := logger.FromContext
	awsProviderMock := awsPMock.NewICostExplorerProvider(t)
	awsCeMock := awsMock.NewICostExplorerService(t)
	slackMock := slackMocks.NewISlack(t)

	s := SpotZeroCostsExplorerService{
		loggerProvider,
		bqMock,
		mpaFsMock,
		smMock,
		awsProviderMock,
		slackMock,
	}

	nonBillingAsg1 := bq.NonBillingAsg{
		PrimaryDomain: "domain1",
		Account:       "1",
	}
	nonBillingAsg2 := bq.NonBillingAsg{
		PrimaryDomain: "domain2",
		Account:       "2",
	}

	cusID := "111"
	mpa1 := domain.MasterPayerAccount{
		Domain:        nonBillingAsg1.PrimaryDomain,
		AccountNumber: "11",
		Features: &domain.Features{
			NRA: true,
		},
		RoleARN:    "roll1",
		CustomerID: &cusID,
	}

	mpa2 := domain.MasterPayerAccount{
		Domain:        nonBillingAsg2.PrimaryDomain,
		AccountNumber: "21",
		Features: &domain.Features{
			NRA: false,
		},
		RoleARN:    "roll2",
		CustomerID: &cusID,
	}

	mpa3 := domain.MasterPayerAccount{
		Domain:        nonBillingAsg2.PrimaryDomain,
		AccountNumber: "22",
		Features: &domain.Features{
			NRA: false,
		},
		RoleARN:    "roll3",
		CustomerID: &cusID,
	}

	bqMock.On("GetNonBillingTagsDomains", mock.Anything).Return([]*bq.NonBillingAsg{
		&nonBillingAsg1, &nonBillingAsg2,
	}, nil)

	smMock.On("GetSecretContentAsStruct", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mpaFsMock.On("GetMasterPayerAccountsForDomain", mock.Anything, mock.Anything).Return([]*domain.MasterPayerAccount{&mpa1}, nil).Once()
	mpaFsMock.On("GetMasterPayerAccountsForDomain", mock.Anything, mock.Anything).Return([]*domain.MasterPayerAccount{&mpa2, &mpa3}, nil).Once()

	awsCeMock.On("CreateCostExplorerService", mock.Anything, mock.Anything).Return()
	awsCeMock.On("UpdateCostAllocationTagsStatus", mock.Anything).Return(nil, nil)
	awsCeMock.On("GetSessionByCreds", mock.Anything, mock.Anything).Return(nil, nil)
	awsCeMock.On("GetAssumeRoleSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	awsProviderMock.On("GetEmptyService").Return(awsCeMock)

	slackMock.On("PublishToSlack", mock.Anything, mock.Anything).Return(nil, nil)

	ctx := context.Background()

	err := s.UpdateCostAllocationTags(ctx)
	assert.NoError(t, err)
	awsCeMock.AssertExpectations(t)
}
