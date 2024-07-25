package costs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/iterator"

	"github.com/doitintl/bigquery/iface"
	mpaFs "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	bq "github.com/doitintl/hello/scheduled-tasks/spot0/dal/bigquery"
	fs "github.com/doitintl/hello/scheduled-tasks/spot0/dal/firestore"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

type SpotZeroCostsService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	bqService      bq.ISpot0CostsBigQuery
	fsService      fs.ISpot0CostsFireStore
	mpaFsService   mpaFs.MasterPayerAccounts
}

func NewSpotScalingCostsService(loggerProvider logger.Provider, conn *connection.Connection) *SpotZeroCostsService {
	ctx := context.Background()

	bqService, err := bq.NewBigQueryService(ctx)
	if err != nil {
		panic(err)
	}

	fsService := fs.NewSpot0CostsFirestoreWithClient(conn.Firestore(ctx))

	mpaFsService, err := mpaFs.NewMasterPayerAccountDAL(ctx, common.ProjectID)
	if err != nil {
		panic(err)
	}

	return &SpotZeroCostsService{
		loggerProvider,
		conn,
		bqService,
		fsService,
		mpaFsService,
	}
}

func (s *SpotZeroCostsService) SpotScalingDailyCosts(ctx context.Context, startDate, endDate, year, month, accountID string) error {
	err := s.bqService.AggregateDailySavings(ctx, startDate, endDate, accountID)
	if err != nil {
		return fmt.Errorf("error aggregating daily savings: %w", err)
	}

	monthlyUsageIter, err := s.bqService.GetMonthlyUsage(ctx, year, month, accountID)
	if err != nil {
		return fmt.Errorf("error getting monthly usage: %w", err)
	}

	err = s.saveResults(ctx, monthlyUsageIter)
	if err != nil {
		return fmt.Errorf("error saving monthly usage results: %w", err)
	}

	return nil
}

func (s *SpotZeroCostsService) SpotScalingMonthlyCosts(ctx context.Context, year, month, accountID string) error {
	if year == "" || month == "" {
		year, month = times.PrevMonth(time.Now())
	}

	monthlyUsageIter, err := s.bqService.GetMonthlyUsage(ctx, year, month, accountID)
	if err != nil {
		return fmt.Errorf("error getting monthly usage: %w", err)
	}

	if err := s.saveResults(ctx, monthlyUsageIter); err != nil {
		return fmt.Errorf("error saving monthly usage results: %w", err)
	}

	return nil
}

// saveResults iterates through the results and writes them to Firestore
func (s *SpotZeroCostsService) saveResults(ctx context.Context, it iface.RowIterator) error {
	log := s.loggerProvider(ctx)

	for {
		var row bq.AsgMonthlyUsage
		err := it.Next(&row)

		if err == iterator.Done {
			break
		}

		if err != nil {
			return fmt.Errorf("error iterating through monthly usage results: %w", err)
		}

		usage := convertToUsageDoc(row)

		err = s.fsService.UpdateASGsUsage(ctx, usage)

		if err != nil {
			log.Infof("error updating usage for %s: %v", usage.DocID, err)
			continue
		}
	}

	return nil
}

// convertToUsageDoc converts AsgMonthlyUsage to UsageDoc
func convertToUsageDoc(row bq.AsgMonthlyUsage) fs.UsageDoc {
	var usage = fs.UsageDoc{
		DocID:        row.DocID,
		YearMonthKey: fmt.Sprintf("%s_%s", row.BillingYear, strings.TrimLeft(row.BillingMonth, "0")),
		Usage: fs.Usage{
			SpotInstances: fs.InstancesSummary{
				TotalCost:  row.CurMonthSpotSpending,
				TotalHours: row.CurMonthSpotHours,
				Instances:  nil,
			},
			OnDemandInstances: fs.InstancesSummary{
				TotalCost:  row.CurMonthOnDemandSpending,
				TotalHours: row.CurMonthOnDemandHours,
				Instances:  nil,
			},
			TotalSavings:          row.CurMonthTotalSavings,
			OnDemandInstancePrice: row.OnDemandInstancePrice,
		},
	}

	for _, instance := range row.InstanceDetails {
		// Check if the instance is spot/ on-demand/ both using the costs
		if instance.CurMonthSpotSpending != 0 {
			instanceSummary := &fs.InstanceSummary{
				Cost:         instance.CurMonthSpotSpending,
				AmountHours:  instance.CurMonthSpotHours,
				InstanceType: instance.InstanceType,
				Platform:     instance.Platform,
				OnDemandCost: instance.OnDemandCost,
			}
			usage.Usage.SpotInstances.Instances = append(usage.Usage.SpotInstances.Instances, instanceSummary)
		}

		if instance.CurMonthOnDemandSpending != 0 {
			instanceSummary := &fs.InstanceSummary{
				Cost:         instance.CurMonthOnDemandSpending,
				AmountHours:  instance.CurMonthOnDemandHours,
				InstanceType: instance.InstanceType,
				Platform:     instance.Platform,
				OnDemandCost: instance.OnDemandCost,
			}
			usage.Usage.OnDemandInstances.Instances = append(usage.Usage.OnDemandInstances.Instances, instanceSummary)
		}
	}

	return usage
}
