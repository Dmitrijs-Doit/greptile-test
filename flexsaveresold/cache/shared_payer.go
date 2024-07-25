package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/googleapi"

	fspkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/accounts"
	cloudHealth "github.com/doitintl/hello/scheduled-tasks/cloudhealth/dal"
	cloudHealthIface "github.com/doitintl/hello/scheduled-tasks/cloudhealth/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	pkg "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/internal/aws_recommendations"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/internal/aws_usage"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type awsUsageServiceInterface interface {
	GetMonthlyOnDemand(ctx context.Context, customerID string, applicableMonths []string) (map[string]float64, error)
}

type awsRecommendationsServiceInterface interface {
	GetRecommendations(ctx context.Context, customerID string) ([]types.Recommendation, error)
}

//go:generate mockery --name SharedPayerService --output ./mocks
type SharedPayerService struct {
	loggerProvider logger.Provider
	*connection.Connection

	awsRecommendationsService awsRecommendationsServiceInterface
	awsUsageService           awsUsageServiceInterface
	awsAccountsService        accounts.Service
	cloudHealthDAL            cloudHealthIface.CloudHealthDAL
	bigQueryService           bq.BigQueryServiceInterface
	customerDAL               customerDal.Customers
}

func NewSharedPayerService(log logger.Provider, conn *connection.Connection) *SharedPayerService {
	bigQueryService, err := bq.NewBigQueryService()
	if err != nil {
		panic(err)
	}

	awsRecommendationsService := aws_recommendations.NewAWSRecommendationsService(log, conn)

	accountService, err := accounts.NewService()
	if err != nil {
		panic(err)
	}

	cloudHealthDal := cloudHealth.NewCloudHealthDAL(conn.Firestore(context.Background()))

	customerDAL := customerDal.NewCustomersFirestoreWithClient(conn.Firestore)

	awsUsageService := aws_usage.NewAWSUsageService(log, conn, bigQueryService, cloudHealthDal, customerDAL)

	return &SharedPayerService{
		log,
		conn,
		awsRecommendationsService,
		awsUsageService,
		accountService,
		cloudHealthDal,
		bigQueryService,
		customerDAL,
	}
}

func (s *SharedPayerService) GetCache(ctx context.Context, configInfo pkg.CustomerInputAttributes, timeParams pkg.TimeParams) (*fspkg.FlexsaveSavings, error) {
	if len(configInfo.AssetIDs) == 0 {
		return nil, nil
	}

	fs := s.Firestore(ctx)
	log := s.loggerProvider(ctx)

	customerRef := fs.Collection("customers").Doc(configInfo.CustomerID)

	savingsHistory := s.getMonthlyFlexSaveSavings(ctx, timeParams.ApplicableMonths, customerRef)

	var nextMonthSavings float64

	var calculatedSavingsHistory map[string]*fspkg.FlexsaveMonthSummary

	reasonCantEnable := noError

	monthlyOnDemand, err := s.awsUsageService.GetMonthlyOnDemand(ctx, customerRef.ID, timeParams.ApplicableMonths)
	if err != nil {
		if datasetNotFound(err) {
			return nil, nil
		}

		if !errors.Is(&cloudHealth.CHTError{}, err) {
			return nil, err
		}
	}

	if !configInfo.IsEnabled {
		recommendations, err := s.awsRecommendationsService.GetRecommendations(ctx, customerRef.ID)
		if err != nil {
			log.Warningf("error: %v getting recommendations for customer: %v", err.Error(), configInfo.CustomerID)
		}

		for _, recommendation := range recommendations {
			nextMonthSavings += *recommendation.Savings
		}

		hasCloudHealthTable, err := s.checkHasCloudHealthTable(ctx, customerRef)
		if err != nil && !errors.Is(&cloudHealth.CHTError{}, err) {
			log.Errorf("when checking if should mark as cloudhealth config error for customer %s, an error occurred: %v ", configInfo.CustomerID, err.Error())
			return nil, err
		}

		hasBeenLongSinceOnboarded, err := s.getHasBeenLongSinceOnboarded(ctx, configInfo.AssetIDs)
		if err != nil {
			log.Errorf("error: %v checking if had no spend in 30 days from activation for customer: %s", err.Error(), configInfo.CustomerID)
		}

		reasonCantEnable = getSharedPayerReasonCantEnable(nextMonthSavings, configInfo.AssetIDs, hasCloudHealthTable, hasBeenLongSinceOnboarded)
	}

	if len(monthlyOnDemand) > 0 {
		calculatedSavingsHistory = makeMonthlyOnDemand(monthlyOnDemand, timeParams.ApplicableMonths, savingsHistory)
	}

	savingsSummary := makeSavingsSummary(
		timeParams.CurrentMonth,
		nextMonthSavings,
	)

	return &fspkg.FlexsaveSavings{
		ReasonCantEnable: reasonCantEnable,
		SavingsHistory:   calculatedSavingsHistory,
		SavingsSummary:   savingsSummary,
	}, nil
}

func (s *SharedPayerService) checkHasCloudHealthTable(ctx context.Context, customerRef *firestore.DocumentRef) (bool, error) {
	chID, err := s.cloudHealthDAL.GetCustomerCloudHealthID(ctx, customerRef)
	if err != nil {
		return false, err
	}

	err = s.bigQueryService.CheckActiveBillingTableExists(ctx, chID)
	if err == bq.ErrNoActiveTable {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *SharedPayerService) getHasBeenLongSinceOnboarded(ctx context.Context, assetIDs []string) (bool, error) {
	extendedAssetIDs := make([]string, 0)

	for _, assetID := range assetIDs {
		extendedAssetIDs = append(extendedAssetIDs, fmt.Sprintf("amazon-web-services-%s", assetID))
	}

	oldestAssetAge, err := s.awsAccountsService.GetOldestJoinTimestampAge(ctx, extendedAssetIDs, time.Now().UTC())
	if err != nil {
		return false, err
	}

	const daysAfterWhichShouldCheckIfCloudHealthIsConfigured = 30

	return oldestAssetAge > daysAfterWhichShouldCheckIfCloudHealthIsConfigured, nil
}

func makeMonthlyOnDemand(monthlyOnDemand map[string]float64, applicableMonths []string, metrics pkg.SpendDataMonthly) pkg.SpendDataMonthly {
	result := pkg.SpendDataMonthly{}

	for i := range applicableMonths {
		calculatedMonth := applicableMonths[i]
		_, onDemandPresent := monthlyOnDemand[calculatedMonth]
		_, metricsPresent := metrics[calculatedMonth]

		if onDemandPresent && metricsPresent {
			result[calculatedMonth] = &fspkg.FlexsaveMonthSummary{
				OnDemandSpend: common.Round(monthlyOnDemand[calculatedMonth] - metrics[calculatedMonth].Savings),
				Savings:       metrics[calculatedMonth].Savings,
			}
		} else {
			result[calculatedMonth] = &fspkg.FlexsaveMonthSummary{
				OnDemandSpend: 0,
				Savings:       0,
			}
		}
	}

	return result
}

func (s *SharedPayerService) getMonthlyFlexSaveSavings(ctx context.Context, applicableMonths []string, customerRef *firestore.DocumentRef) pkg.SpendDataMonthly {
	log := s.loggerProvider(ctx)

	t := time.Now().Truncate(time.Hour * 24).UTC()
	firstOfNextMonth := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC)

	metrics := make(pkg.SpendDataMonthly)

	for i := 0; i < flexsaveHistoryMonthAmount; i++ {
		// use first day of next month minus one day when subtracting months to guarantee previous calendar month
		month := firstOfNextMonth.AddDate(0, -i, -1)
		monthTotal := 0.0

		var metric fspkg.FlexsaveMonthSummary

		invoiceAdjustment, err := getCustomerFlexSaveInvoiceAdjustments(ctx, customerRef, month)
		if err != nil {
			log.Error(err)
			continue
		}

		for _, adjustment := range invoiceAdjustment {
			monthTotal += -1 * adjustment.Amount
		}

		metric.Savings = monthTotal
		metrics[applicableMonths[i]] = &metric
	}

	return metrics
}

func getCustomerFlexSaveInvoiceAdjustments(ctx context.Context, customerRef *firestore.DocumentRef, invoiceMonth time.Time) ([]*common.InvoiceAdjustment, error) {
	invoiceAdjustments := make([]*common.InvoiceAdjustment, 0)

	var docSnaps []*firestore.DocumentSnapshot

	var err error

	docSnaps, err = customerRef.Collection("customerInvoiceAdjustments").
		Where("type", "==", common.Assets.AmazonWebServices).
		// invoiceMonth should always be first of month, but regenerate the timestamp to be safe
		Where("invoiceMonths", "array-contains", time.Date(invoiceMonth.Year(), invoiceMonth.Month(), 1, 0, 0, 0, 0, time.UTC)).
		Documents(ctx).GetAll()

	if err != nil {
		return nil, err
	}

	substrings := []string{flexsaveDetailPrefix, flexRIDetailPrefix}

	for _, docSnap := range docSnaps {
		var invoiceAdjustment common.InvoiceAdjustment
		if err := docSnap.DataTo(&invoiceAdjustment); err != nil {
			return nil, err
		}

		invoiceAdjustment.Snapshot = docSnap
		for _, sub := range substrings {
			if strings.Contains(invoiceAdjustment.Details, sub) {
				invoiceAdjustments = append(invoiceAdjustments, &invoiceAdjustment)
			}
		}
	}

	return invoiceAdjustments, nil
}

func getSharedPayerReasonCantEnable(nextMonthSavings float64, allAssets []string, hasCloudHealthTable bool, hasBeenLongSinceOnboarded bool) string {
	if len(allAssets) == 0 {
		return errNoAssets
	}

	if nextMonthSavings > 0 {
		return noError
	}

	if !hasBeenLongSinceOnboarded {
		return errNoSpend
	}

	if !hasCloudHealthTable {
		return errCHNotConfigured
	}

	return errNoSpendInThirtyDays
}

func makeSavingsSummary(currentMonth string, predictedSavings float64) *fspkg.FlexsaveSavingsSummary {
	return &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
			Month: currentMonth,
		},
		NextMonth: &fspkg.FlexsaveMonthSummary{
			Savings: predictedSavings,
		},
	}
}

func datasetNotFound(err error) bool {
	var gErr *googleapi.Error

	if errors.As(err, &gErr) && gErr.Code == 404 {
		return true
	}

	return false
}
