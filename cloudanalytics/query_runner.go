package cloudanalytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/bigquery/iface"
	bqLensProxyClientIface "github.com/doitintl/bq-lens-proxy/client/iface"
	bqLensProxyDomain "github.com/doitintl/bq-lens-proxy/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

func runQueryThroughProxy(
	ctx context.Context,
	qr *QueryRequest,
	r *runQueryParams,
	proxyClient bqLensProxyClientIface.BQLensProxyClient,
	serverDurationSteps map[string]int64,
) (*runQueryRes, error) {
	startTimeProcessing := time.Now()

	l := logger.FromContext(ctx)

	var (
		res runQueryRes
	)

	if r.email != "" {
		// append a comment with the user's email to the query string
		r.queryString = "-- " + r.email + "\n" + r.queryString
	}

	l.Info(r.queryString)
	l.Infof("running query through proxy for customer %s", r.customerID)

	house, feature, module := domainOrigin.MapOriginToHouseFeatureModule(qr.Origin)

	priority := bigquery.InteractivePriority

	req := &bqLensProxyDomain.RunQueryRequest{
		CustomerID: r.customerID,
		Parameters: r.queryParams,
		Labels: map[string]string{
			labelCloudReportsCustomer:        labelReg.ReplaceAllString(strings.ToLower(r.customerID), "_"),
			labelCloudReportsUser:            labelReg.ReplaceAllString(r.email, "_"),
			labelCloudAnalyticsOrigin:        qr.Origin,
			common.LabelKeyEnv.String():      common.GetEnvironmentLabel(),
			common.LabelKeyHouse.String():    house.String(),
			common.LabelKeyFeature.String():  feature.String(),
			common.LabelKeyModule.String():   module.String(),
			common.LabelKeyCustomer.String(): labelReg.ReplaceAllString(strings.ToLower(r.customerID), "_"),
		},
		QueryString:       r.queryString,
		DisableQueryCache: false,
		DryRun:            false,
		Priority:          &priority,
	}

	iter, err := proxyClient.RunQuery(ctx, req)
	if err != nil {
		return nil, err
	}

	queryMs := time.Since(startTimeProcessing).Milliseconds()

	res.result.Details = map[string]interface{}{}

	return runQueryTail(l, &res, r, qr, iter, queryMs, startTimeProcessing, serverDurationSteps)
}

func runQuery(
	ctx context.Context,
	conn *connection.Connection,
	bq *bigquery.Client,
	qr *QueryRequest,
	r *runQueryParams,
	serverDurationSteps map[string]int64,
) (*runQueryRes, error) {
	startTimeProcessing := time.Now()

	l := logger.FromContext(ctx)

	var res runQueryRes

	if bq == nil {
		bqOrigin, ok := domainOrigin.BigqueryForOrigin(ctx, qr.Origin, conn)
		if !ok {
			l.Infof("could not get bq client for origin %s, using default client", qr.Origin)
		}

		bq = bqOrigin
	}

	if r.email != "" {
		// append a comment with the user's email to the query string
		r.queryString = "-- " + r.email + "\n" + r.queryString
	}

	jobID := fmt.Sprintf("%s_%s", cloudAnalyticsReportPrefix, qr.Origin)

	l.Info(r.queryString)

	if common.IsLocalhost {
		prettyJSON, err := json.MarshalIndent(r.queryParams, "", "  ")
		if err == nil {
			l.Info(string(prettyJSON))
		}

	}
	l.Infof("running query on project %s", bq.Project())

	queryJob := bq.Query(r.queryString)
	queryJob.DryRun = false
	queryJob.UseLegacySQL = false
	queryJob.DisableQueryCache = false
	queryJob.JobIDConfig = bigquery.JobIDConfig{JobID: jobID, AddJobIDSuffix: true}
	queryJob.Priority = bigquery.InteractivePriority
	queryJob.Parameters = r.queryParams
	queryJob.MaxBillingTier = 1
	house, feature, module := domainOrigin.MapOriginToHouseFeatureModule(qr.Origin)
	queryJob.Labels = map[string]string{
		labelCloudReportsCustomer:        labelReg.ReplaceAllString(strings.ToLower(r.customerID), "_"),
		labelCloudReportsUser:            labelReg.ReplaceAllString(r.email, "_"),
		labelCloudAnalyticsOrigin:        qr.Origin,
		common.LabelKeyEnv.String():      common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():    house.String(),
		common.LabelKeyFeature.String():  feature.String(),
		common.LabelKeyModule.String():   module.String(),
		common.LabelKeyCustomer.String(): labelReg.ReplaceAllString(strings.ToLower(r.customerID), "_"),
	}

	// If the BigQuery client uses the project that is not under a reservation,
	// limit the bytes billed to 5TB to put a cap on the cost of one single query.
	if bq.Project() == domainOrigin.OnDemandProdProject {
		queryJob.MaxBytesBilled = 5 * common.TebiByte
	}

	job, err := queryJob.Run(ctx)
	if err != nil {
		return nil, err
	}

	l.Infof("%s:%s.%s", job.ProjectID(), job.Location(), job.ID())

	ctxWait, cancelWait := jobWaitContext(ctx, qr.Origin)
	defer cancelWait()

	status, err := job.Wait(ctxWait)
	if err != nil {
		if err == context.DeadlineExceeded {
			l.Debug("query running for too long; abort.")

			if err := job.Cancel(ctx); err != nil {
				l.Debugf("failed to cancel query job with error: %s", err)
			}

			res.result.Error = &QueryResultError{
				Code: ErrorCodeQueryTimeout,
			}

			return &res, nil
		}

		if gapiErr, ok := err.(*googleapi.Error); ok {
			if gapiErr.Code == http.StatusBadRequest {
				switch ErrorCode(gapiErr.Message) {
				case ErrorCodeResultTooLarge, ErrorCodeSeriesCountTooLarge:
					res.result.Error = &QueryResultError{
						Code:   ErrorCode(gapiErr.Message),
						Status: http.StatusRequestEntityTooLarge,
					}

					return &res, nil
				}
			}
		}

		return nil, err
	}

	queryMs := time.Since(startTimeProcessing).Milliseconds()

	if err := status.Err(); err != nil {
		return nil, err
	}

	iter, err := job.Read(ctx)
	if err != nil {
		return nil, err
	}

	queryStats := status.Statistics.Details.(*bigquery.QueryStatistics)
	duration := status.Statistics.EndTime.Sub(status.Statistics.StartTime)

	var biEngineMode string
	if queryStats.BIEngineStatistics != nil {
		biEngineMode = queryStats.BIEngineStatistics.BIEngineMode
	}

	onlineQuery := qr.Origin == domainOrigin.QueryOriginClient || qr.Origin == domainOrigin.QueryOriginClientReservation

	res.result.Details = map[string]interface{}{
		"jobId":               job.ID(),
		"totalRows":           iter.TotalRows,
		"billingTier":         queryStats.BillingTier,
		"biEngineMode":        biEngineMode,
		"cacheHit":            queryStats.CacheHit,
		"totalBytesBilled":    queryStats.TotalBytesBilled,
		"totalBytesProcessed": queryStats.TotalBytesProcessed,
		"slotMillis":          queryStats.SlotMillis,
		"aggregation":         getAggregationType(queryStats),
		"startTime":           status.Statistics.StartTime,
		"endTime":             status.Statistics.EndTime,
		"duration":            duration.Seconds(),
		"durationMs":          int64(duration) / 1e6,
		"onlineQuery":         onlineQuery,
	}

	if iter.TotalRows == 0 {
		res.result.Error = &QueryResultError{
			Status: http.StatusNotFound,
			Code:   ErrorCodeResultEmpty,
		}

		return &res, nil
	}

	return runQueryTail(l, &res, r, qr, iter, queryMs, startTimeProcessing, serverDurationSteps)
}

func runQueryTail(
	l logger.ILogger,
	res *runQueryRes,
	r *runQueryParams,
	qr *QueryRequest,
	iter iface.RowIterator,
	queryMs int64,
	startTimeProcessing time.Time,
	serverDurationSteps map[string]int64,
) (*runQueryRes, error) {
	for {
		row, err := nextRow(iter, r.isComparative)
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}

		// Discard the error checks column
		row = row[:len(row)-1]
		if r.forecastMode {
			isForecastData, ok := row[0].(bool)
			if !ok {
				err := fmt.Errorf("invalid result row for forecast %v", row)
				return nil, err
			}

			row = row[1:]
			res.allRows = append(res.allRows, row)
			// only return to client rows that are not for forecasting and remove bool indicator for forecast
			if !isForecastData {
				res.rows = append(res.rows, row)
			}
		} else {
			res.rows = append(res.rows, row)
		}
	}

	defer func() {
		if serverDurationSteps != nil {
			serverDurationSteps["queryMs"] = queryMs
			serverDurationSteps["readMs"] = time.Since(startTimeProcessing).Milliseconds() - queryMs
		}
	}()

	if len(res.rows) == 0 {
		res.result.Error = &QueryResultError{
			Status: http.StatusNotFound,
			Code:   ErrorCodeResultEmpty,
		}

		return res, nil
	}

	if r.isComparative {
		if resultRows, err := qr.addMissingComparativeRows(&res.rows); err != nil {
			l.Debugf("failed to add comparative missing comparative rows with error: %s", err)
		} else if resultRows != nil {
			res.rows = *resultRows
		}
	}

	return res, nil
}
