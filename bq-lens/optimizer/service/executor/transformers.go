package executor

import (
	"fmt"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

func (s *Executor) assignTransformer(timeRange bqmodels.TimeRange, queryResult interface{}, tctx domain.TransformerContext) (dal.RecommendationSummary, error) {
	now := s.timeNow()

	switch res := queryResult.(type) {
	case []bqmodels.CostFromTableTypesResult:
		return TransformCostFromTableTypes(timeRange, res, now), nil
	case []bqmodels.ScheduledQueriesMovementResult:
		return TransformScheduledQueriesMovement(timeRange, tctx.Discount, tctx.TotalScanPricePerPeriod, res, now), nil
	case []bqmodels.StandardScheduledQueriesMovementResult:
		return TransformStandardScheduledQueriesMovement(timeRange, tctx.Discount, tctx.TotalScanPricePerPeriod, res, now), nil
	case []bqmodels.EnterpriseScheduledQueriesMovementResult:
		return TransformEnterpriseScheduledQueriesMovement(timeRange, tctx.Discount, tctx.TotalScanPricePerPeriod, res, now), nil
	case []bqmodels.EnterprisePlusScheduledQueriesMovementResult:
		return TransformEnterprisePlusScheduledQueriesMovement(timeRange, tctx.Discount, tctx.TotalScanPricePerPeriod, res, now), nil
	case []bqmodels.TableStoragePriceResult:
		return TransformTableStoragePrice(timeRange, res, now)
	case []bqmodels.DatasetStoragePriceResult:
		return TransformDatasetStoragePrice(timeRange, res, now)
	case []bqmodels.ProjectStoragePriceResult:
		return TransformProjectStoragePrice(timeRange, res, now)
	case []bqmodels.TableStorageTBResult:
		return TransformTableStorageTB(res, now)
	case []bqmodels.DatasetStorageTBResult:
		return TransformDatasetStorageTB(res, now)
	case []bqmodels.ProjectStorageTBResult:
		return TransformProjectStorageTB(res, now)
	case *bqmodels.RunBillingProjectResult:
		return TransformBillingProject(timeRange, res, now)
	case *bqmodels.RunStandardBillingProjectResult:
		return TransformStandardBillingProject(timeRange, res, now)
	case *bqmodels.RunEnterpriseBillingProjectResult:
		return TransformEnterpriseBillingProject(timeRange, res, now)
	case *bqmodels.RunEnterprisePlusBillingProjectResult:
		return TransformEnterprisePlusBillingProject(timeRange, res, now)
	case *bqmodels.RunUserSlotsResult:
		return TransformUserSlots(timeRange, res, now)
	case *bqmodels.RunStandardUserSlotsResult:
		return TransformStandardUserSlots(timeRange, res, now)
	case *bqmodels.RunEnterpriseUserSlotsResult:
		return TransformEnterpriseUserSlots(timeRange, res, now)
	case *bqmodels.RunEnterprisePlusUserSlotsResult:
		return TransformEnterprisePlusUserSlots(timeRange, res, now)
	case []bqmodels.FlatRateSlotsExplorerResult:
		return TransformFlatRateSlotsExplorer(timeRange, res, now)
	case []bqmodels.OnDemandSlotsExplorerResult:
		return TransformOnDemandSlotsExplorer(timeRange, res, now)
	case []bqmodels.StandardSlotsExplorerResult:
		return TransformStandardSlotsExplorer(timeRange, res, now)
	case []bqmodels.EnterpriseSlotsExplorerResult:
		return TransformEnterpriseSlotsExplorer(timeRange, res, now)
	case []bqmodels.EnterprisePlusSlotsExplorerResult:
		return TransformEnterprisePlusSlotsExplorer(timeRange, res, now)
	case *bqmodels.RunODBillingProjectResult:
		return TransformODBillingProject(timeRange, tctx.Discount, res, now)
	case *bqmodels.RunODUserResult:
		return TransformODUser(timeRange, tctx.Discount, res, now)
	case *bqmodels.RunODProjectResult:
		return TransformODProject(timeRange, tctx.Discount, res, now)
	case *bqmodels.RunODDatasetResult:
		return TransformODataset(timeRange, tctx.Discount, res, now)
	default:
		return nil, fmt.Errorf("unhandled query result type %+v", queryResult)
	}
}
