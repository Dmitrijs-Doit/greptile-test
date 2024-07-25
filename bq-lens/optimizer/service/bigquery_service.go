package service

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"

	bqLensDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/domain"
	bigueryDALIface "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/bigquery/iface"
	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service/executor"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

type BigQueryService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	bqDAL          bigueryDALIface.Bigquery
}

func NewBigQueryService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	dal bigueryDALIface.Bigquery,
) *BigQueryService {
	return &BigQueryService{
		loggerProvider: loggerProvider,
		conn:           conn,
		bqDAL:          dal,
	}
}

func (s *BigQueryService) GetCustomerDiscounts(ctx context.Context, bq *bigquery.Client) (map[string]float64, error) {
	query := bqmodels.DiscountsAllCustomers

	discounts, err := s.bqDAL.RunDiscountsAllCustomersQuery(ctx, query, bq)
	if err != nil {
		return nil, err
	}

	allCustomerDiscounts := make(map[string]float64)
	for _, discount := range discounts {
		allCustomerDiscounts[discount.CustomerID] = discount.Discount
	}

	return allCustomerDiscounts, nil
}

func (s *BigQueryService) GetDatasetLocationAndProjectID(ctx context.Context, bq *bigquery.Client, datasetID string) (string, string, error) {
	return s.bqDAL.GetDatasetLocationAndProjectID(ctx, bq, datasetID)
}

func (s *BigQueryService) GetAggregatedJobStatistics(ctx context.Context, bq *bigquery.Client, projectID, location string) ([]bqmodels.AggregatedJobStatistic, error) {
	return s.bqDAL.RunAggregatedJobStatisticsQuery(ctx, bq, projectID, location)
}

func (s *BigQueryService) GetTableDiscoveryMetadata(ctx context.Context, bq *bigquery.Client) (*bigquery.TableMetadata, error) {
	return s.bqDAL.GetTableDiscoveryMetadata(ctx, bq)
}

func (s *BigQueryService) GetMinAndMaxDates(ctx context.Context, bq *bigquery.Client, projectID string, location string) (*bqmodels.CheckCompleteDaysResult, error) {
	checkCompleteDays := bqmodels.CheckCompleteDays

	query := strings.NewReplacer(
		"{projectIdPlaceHolder}",
		projectID,
		"{datasetIdPlaceHolder}",
		bqLensDomain.DoitCmpDatasetID,
	).Replace(checkCompleteDays)

	res, err := s.bqDAL.RunCheckCompleteDaysQuery(ctx, query, bq)
	if err != nil {
		return nil, err
	}

	return &res[0], nil
}

func (s *BigQueryService) GenerateStorageRecommendation(
	ctx context.Context,
	customerID string,
	bq *bigquery.Client,
	discount float64,
	replacements domain.Replacements,
	now time.Time,
	hasTableDiscovery bool,
) (domain.PeriodTotalPrice, dal.RecommendationSummary, error) {
	l := s.loggerProvider(ctx)
	l.SetLabels(DefaultLogFields)

	replacements.StartDate = now.AddDate(0, 0, -30).Format(times.YearMonthDayLayout)

	scanPrices, err := s.bqDAL.RunTotalScanPricePerPeriod(ctx, bq, replacements, now)
	if err != nil {
		return nil, nil, err
	}

	if len(scanPrices) == 0 {
		return nil, nil, errScanPriceNotFound
	}

	periodScanPrices := domain.PeriodTotalPrice{
		bqmodels.TimeRangeDay: {
			TotalScanPrice: scanPrices[0].TotalUpTo1DayAgo.Float64 * discount * executor.PricePerTBScan,
		},
		bqmodels.TimeRangeWeek: {
			TotalScanPrice: scanPrices[0].TotalUpTo7DaysAgo.Float64 * discount * executor.PricePerTBScan,
		},
		bqmodels.TimeRangeMonth: {
			TotalScanPrice: scanPrices[0].TotalUpTo30DaysAgo.Float64 * discount * executor.PricePerTBScan,
		},
	}

	if !hasTableDiscovery {
		l.Infof("skipping storage recommendations for customer '%s' due to missing table discovery", customerID)

		return periodScanPrices, nil, nil
	}

	storageData, err := s.bqDAL.RunStorageRecommendationsQuery(ctx, bq, replacements, now)
	if err != nil {
		l.Error(wrapOperationError("RunStorageRecommendationsQuery", customerID, err).Error())

		return periodScanPrices, nil, nil
	}

	if len(storageData) == 0 {
		l.Infof("no data received from storage recommendations query for customer '%s'", customerID)

		return periodScanPrices, nil, nil
	}

	totalStorageCost := storageData[0].TotalStorageCost.Float64 * discount
	recommendationsSummary := make(dal.RecommendationSummary)

	for _, period := range bqmodels.DataPeriods {
		daysInPeriod, err := domain.GetDayBasedOnTimeRange(period)
		if err != nil {
			return nil, nil, err
		}

		priceDetails := periodScanPrices[period]
		priceDetails.TotalStoragePrice = (totalStorageCost / executor.DaysInMonth) * float64(daysInPeriod)
		priceDetails.TotalPrice = priceDetails.TotalStoragePrice + priceDetails.TotalScanPrice
		periodScanPrices[period] = priceDetails

		// Process storage recommendations for each time range
		periodRecommendations := executor.TransformStorageRecommendations(period, discount, storageData, priceDetails.TotalPrice, now)

		for queryID, recommendations := range periodRecommendations {
			if _, exists := recommendationsSummary[queryID]; !exists {
				recommendationsSummary[queryID] = make(dal.TimeRangeRecommendation)
			}

			for timePeriod, recommendation := range recommendations {
				recommendationsSummary[queryID][timePeriod] = recommendation
			}
		}
	}

	return periodScanPrices, recommendationsSummary, nil
}

func (s *BigQueryService) GetBillingProjectsWithEditions(ctx context.Context, bq *bigquery.Client) (map[string][]domain.BillingProjectWithReservation, error) {
	query := bqmodels.BillingProjectsWithEditionsQuery

	billingProjectsWithReservations, err := s.bqDAL.RunBillingProjectsWithEditionsQuery(ctx, query, bq)
	if err != nil {
		return nil, err
	}

	allCustomerBillingProjectsWithReservations := make(map[string][]domain.BillingProjectWithReservation)
	for _, projectsWithReservations := range billingProjectsWithReservations {
		allCustomerBillingProjectsWithReservations[projectsWithReservations.CustomerID] =
			append(allCustomerBillingProjectsWithReservations[projectsWithReservations.CustomerID],
				domain.BillingProjectWithReservation{
					Project:  projectsWithReservations.Project,
					Location: projectsWithReservations.Location,
				})
	}

	return allCustomerBillingProjectsWithReservations, nil
}

func (s *BigQueryService) GetBillingProjectsWithEditionsSingleCustomer(
	ctx context.Context,
	bq *bigquery.Client,
	customerID string,
) ([]domain.BillingProjectWithReservation, error) {
	var response []domain.BillingProjectWithReservation

	query := strings.NewReplacer(
		"{customer_id}",
		customerID,
	).Replace(bqmodels.BillingProjectsWithEditionsSingleCustomerQuery)

	billingProjectsWithReservations, err := s.bqDAL.RunBillingProjectsWithEditionsQuery(ctx, query, bq)
	if err != nil {
		return nil, err
	}

	for _, project := range billingProjectsWithReservations {
		response = append(response, domain.BillingProjectWithReservation{
			Project:  project.Project,
			Location: project.Location,
		})
	}

	return response, nil
}
