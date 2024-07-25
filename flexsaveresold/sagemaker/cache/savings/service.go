package savings

import (
	"context"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/common"
	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/iface"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

//go:generate mockery --name Service --output ./mocks
type Service interface {
	CreateSavingsHistory(ctx context.Context, customerID string, now time.Time, numberOfMonths int) (map[string]iface.MonthSummary, error)
}

type service struct {
	LoggerProvider  logger.Provider
	bigQueryService bq.BigQueryServiceInterface
}

func NewService(log logger.Provider) Service {
	bigQueryService, err := bq.NewBigQueryService()
	if err != nil {
		panic(err)
	}

	return &service{
		log,
		bigQueryService,
	}
}

func (s service) CreateSavingsHistory(ctx context.Context, customerID string, now time.Time, numberOfMonths int) (map[string]iface.MonthSummary, error) {
	err := s.bigQueryService.CheckActiveBillingTableExists(ctx, customerID)
	if err != nil {
		return nil, err
	}

	params := bq.BigQueryParams{
		Context:             ctx,
		CustomerID:          customerID,
		FirstOfCurrentMonth: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		NumberOfMonths:      numberOfMonths,
	}

	spendByMonths := make(map[string]float64)
	savingsByMonths := make(map[string]float64)

	monthlyData := make(map[string]iface.MonthSummary)

	errChan := make(chan error)
	savingsMonthlyChan := make(chan map[string]float64)
	onDemandChan := make(chan map[string]float64)

	go s.bigQueryService.GetCustomerOnDemand(params, sageMakerOnDemandQuery, onDemandChan, errChan)
	go s.bigQueryService.GetCustomerSavings(params, sageMakerSavingsQuery, savingsMonthlyChan, errChan)

	numberOfChannels := len([]interface{}{savingsMonthlyChan, onDemandChan})

	for i := 0; i < numberOfChannels; i++ {
		select {
		case spendByMonths = <-onDemandChan:
		case savingsByMonths = <-savingsMonthlyChan:
		case err := <-errChan:
			return monthlyData, err
		}
	}

	for month, spend := range spendByMonths {
		var value iface.MonthSummary
		value.OnDemandSpend = common.Round(spend) - common.Round(savingsByMonths[month])
		monthlyData[month] = value
	}

	for month, saving := range savingsByMonths {
		value := monthlyData[month]
		value.Savings = common.Round(saving)
		monthlyData[month] = value
	}

	return monthlyData, nil
}
