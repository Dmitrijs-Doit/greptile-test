package service

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

func GetTableClustering(isCSP bool) *bigquery.Clustering {
	if isCSP {
		return &bigquery.Clustering{Fields: []string{
			"territory",
			"primary_domain",
		}}
	}

	return &bigquery.Clustering{Fields: []string{
		"project_id",
		"service_description",
		"sku_description",
	}}
}

func RunBillingTableUpdateQuery(ctx context.Context, bq *bigquery.Client, query string, data *domain.BigQueryTableUpdateRequest) error {
	l := logger.FromContext(ctx)

	l.Info(query)

	partitions := getBillingTableUpdateQueryPartitions(data)

	if !data.CSP && len(partitions) > 31 {
		return fmt.Errorf("too many partitions %d, consider using allPartitions flag", len(partitions))
	}

	for _, partition := range partitions {
		queryJob := bq.Query(query)
		queryJob.Priority = bigquery.InteractivePriority
		queryJob.Parameters = data.QueryParameters

		var tableNameSuffix string
		if partition != nil {
			tableNameSuffix = fmt.Sprintf("$%s", partition.Format("20060102"))
			queryJob.SchemaUpdateOptions = []string{"ALLOW_FIELD_ADDITION"}
			queryJob.Parameters = append(queryJob.Parameters, bigquery.QueryParameter{Name: "partition", Value: *partition})

			l.Infof("Updating partition %v\n", partition)
		}

		dstDataset := bq.DatasetInProject(data.DestinationProjectID, data.DestinationDatasetID)
		if len(data.DestinationTableName) > 0 {
			queryJob.Dst = dstDataset.Table(data.DestinationTableName + tableNameSuffix)
		}

		queryJob.DefaultProjectID = data.DefaultProjectID
		queryJob.DefaultDatasetID = data.DefaultDatasetID
		queryJob.DryRun = false
		queryJob.UseLegacySQL = false
		queryJob.AllowLargeResults = true
		queryJob.DisableFlattenedResults = true
		queryJob.JobIDConfig = bigquery.JobIDConfig{JobID: data.ConfigJobID, AddJobIDSuffix: true}

		queryJob.Labels = map[string]string{
			common.LabelKeyEnv.String():     common.GetEnvironmentLabel(),
			common.LabelKeyHouse.String():   data.House.String(),
			common.LabelKeyModule.String():  data.Module.String(),
			common.LabelKeyFeature.String(): data.Feature.String(),
		}

		if !data.DML {
			queryJob.CreateDisposition = bigquery.CreateIfNeeded
			queryJob.WriteDisposition = data.WriteDisposition
		}

		if len(data.DestinationTableName) > 0 {
			queryJob.TimePartitioning = &bigquery.TimePartitioning{Type: "DAY", Field: "export_time"}
			queryJob.Clustering = data.Clustering
		}

		job, err := queryJob.Run(ctx)
		if err != nil {
			return err
		}

		l.Info(job.ID())

		if data.WaitTillDone {
			status, err := job.Wait(ctx)
			if err != nil {
				return err
			}

			if err := status.Err(); err != nil {
				return err
			}
		} else {
			// Wait to check if there was any immediate error
			ctxWait, cancelWait := context.WithTimeout(ctx, time.Second*5)
			defer cancelWait()

			status, err := job.Wait(ctxWait)
			if err != nil {
				// If Wait was canceled because deadline exceeded check that there are no query errors and complete the run
				if err == context.DeadlineExceeded {
					status, err := job.Status(ctx)
					if err != nil {
						return err
					}

					if err := status.Err(); err != nil {
						return err
					}
				} else {
					return err
				}
			} else if err := status.Err(); err != nil {
				return err
			}
		}
	}

	return nil
}

func getBillingTableUpdateQueryPartitions(data *domain.BigQueryTableUpdateRequest) []*time.Time {
	// If updating all partitions or appending from specific date
	// then take all the results of the query (not limiting to a specific partition)
	if data.AllPartitions || (data.FromDate != "" && data.WriteDisposition == bigquery.WriteAppend) {
		return []*time.Time{nil}
	}

	now := time.Now().UTC()
	today := now.Truncate(time.Hour * 24)
	partitions := make([]*time.Time, 0)

	// If FromDate was supplied and was parsed correctly, and using WriteTruncate disposition,
	// set to update all partitions from this date one by one
	if data.FromDate != "" {
		startDate, err := time.Parse(times.YearMonthDayLayout, data.FromDate)
		if err == nil {
			var endDate time.Time

			if data.FromDateNumPartitions > 0 {
				endDate = startDate.AddDate(0, 0, data.FromDateNumPartitions-1)
			} else {
				endDate = today
			}

			for day := startDate; !day.After(endDate); day = day.AddDate(0, 0, 1) {
				p := day
				partitions = append(partitions, &p)
			}
		}
	}

	// If no partitions were selected, set the default partition setting
	if len(partitions) == 0 {
		// Update yesterday's partition close to the day cut-off, and for CSP tables always
		if data.CSP || now.Hour() < 16 {
			p := today.Add(time.Hour * -24)
			partitions = append(partitions, &p)
		}

		// Update current day partition
		p1 := today
		partitions = append(partitions, &p1)
	}

	return partitions
}

// CalculateStartDate calculates the start date for the query.
// If fromDate is not a valid date, then it will return the first day of the previous month.
// If fromDate is a valid date, then it will return the fromDate itself.
func CalculateStartDate(startDate string) string {
	if _, err := time.Parse(times.YearMonthDayLayout, startDate); err != nil {
		today := times.CurrentDayUTC()
		return today.AddDate(0, -1, -1*today.Day()+1).Format(times.YearMonthDayLayout)
	}

	return startDate
}
