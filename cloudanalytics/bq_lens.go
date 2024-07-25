package cloudanalytics

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	pricebookDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/bqlens"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"google.golang.org/api/iterator"
)

func getBQAuditLogsTableQuery(
	ctx context.Context,
	discount string,
	bqLensQueryArgs *bqlens.BQLensQueryArgs,
) (tables []string, err error) {
	orderedFields, nonNullFieldsMapping := bqlens.GetBQAuditLogsFields(discount)
	reportFields := getQueryFieldsString(orderedFields, nonNullFieldsMapping)

	subQuery, err := bqlens.GetBQAuditLogsTableSubQuery(ctx, bqLensQueryArgs)
	if err != nil {
		return tables, err
	}

	bqLensTable := fmt.Sprintf("SELECT %s FROM ( %s )", reportFields, subQuery)

	tables = append(tables, bqLensTable)

	return tables, nil
}

func getBQDiscount(ctx context.Context, conn *connection.Connection, fs *firestore.Client, customerID string, b *QueryRequest, r *runQueryParams) (bqDiscount string, err error) {
	bqDiscount = "1.0"

	bqDiscountTables, err := getBigQueryDiscountTables(ctx, fs, customerID)
	if err != nil {
		return bqDiscount, err
	}

	withClauseDiscountsTable := getRawDataQueryString(bqDiscountTables)
	r.queryString = bqlens.GetBQDiscountQuery(withClauseDiscountsTable)

	discountsQueryResponse, err := runQuery(ctx, conn, nil, b, r, nil)
	if err != nil {
		return bqDiscount, err
	}

	if discountsQueryResponse.result.Error != nil || len(discountsQueryResponse.rows) == 0 {
		return bqDiscount, errors.New("failed to get BQ discount")
	}

	if discountsQueryResponse.rows[0] != nil && discountsQueryResponse.rows[0][0] != nil {
		bqDiscount = fmt.Sprintf("%v", discountsQueryResponse.rows[0][0])
	}

	return bqDiscount, nil
}

func getFlatRateSKUUsage(
	ctx context.Context,
	bq *bigquery.Client,
	origin string,
	customerID string,
	startTime string,
	endTime string,
) ([]pricebookDomain.UsageType, error) {
	jobID := fmt.Sprintf("%s_%s", cloudAnalyticsReportPrefix, origin)

	query := bqlens.GetLegacyFlatRateSKUsQuery(customerID, startTime, endTime)
	queryJob := bq.Query(query)

	queryJob.DryRun = false
	queryJob.UseLegacySQL = false
	queryJob.DisableQueryCache = false
	queryJob.JobIDConfig = bigquery.JobIDConfig{JobID: jobID, AddJobIDSuffix: true}
	queryJob.Priority = bigquery.InteractivePriority
	queryJob.MaxBillingTier = 1
	house, feature, module := domainOrigin.MapOriginToHouseFeatureModule(origin)
	queryJob.Labels = map[string]string{
		labelCloudReportsCustomer:        labelReg.ReplaceAllString(strings.ToLower(customerID), "_"),
		labelCloudAnalyticsOrigin:        origin,
		common.LabelKeyEnv.String():      common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():    house.String(),
		common.LabelKeyFeature.String():  feature.String(),
		common.LabelKeyModule.String():   module.String(),
		common.LabelKeyCustomer.String(): labelReg.ReplaceAllString(strings.ToLower(customerID), "_"),
	}

	job, err := queryJob.Run(ctx)
	if err != nil {
		return nil, err
	}

	iter, err := job.Read(ctx)
	if err != nil {
		return nil, err
	}

	var results []pricebookDomain.UsageType

	var row struct {
		Description string `bigquery:"description"`
	}

	for {
		err := iter.Next(&row)
		if err != nil {
			if errors.Is(err, iterator.Done) {
				return results, nil
			}

			return nil, err
		}

		if usageType, ok := descriptionToUsageType(row.Description); ok {
			results = append(results, usageType)
		}
	}
}

// We're interested in these two as they are the ones we have observed that
// result in reservations usage price needing to be overriden.
var descriptionUsages = map[string]pricebookDomain.UsageType{
	"Annual":  pricebookDomain.Commit1Yr,
	"Monthly": pricebookDomain.Commit1Mo,
}

func descriptionToUsageType(description string) (pricebookDomain.UsageType, bool) {
	for substring, usageType := range descriptionUsages {
		if strings.Contains(description, substring) {
			return usageType, true
		}

	}

	return "", false
}
