package dal

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/bqutils"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/courier/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type CourierBQ struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
}

func NewCourierBQ(
	loggerProvider logger.Provider,
	conn *connection.Connection,
) (*CourierBQ, error) {
	return &CourierBQ{
		loggerProvider,
		conn,
	}, nil
}

func (d CourierBQ) SaveMessages(
	ctx context.Context,
	notificationID domain.Notification,
	messagesPerDayMap map[time.Time][]*domain.MessageBQ,
) error {
	l := d.loggerProvider(ctx)

	projectID := common.ProjectID

	bq, ok := domainOrigin.Bigquery(ctx, d.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	table := fmt.Sprintf("%s_%s", domain.CourierNotificationsTable, notificationID)
	dataset := bq.DatasetInProject(projectID, domain.CourierNotificationsDataset)

	if exists, _, err := common.BigQueryDatasetExists(ctx, bq, projectID, domain.CourierNotificationsDataset); err != nil {
		return err
	} else if !exists {
		l.Infof("destination dataset %s in project %s does not exist, creating one", domain.CourierNotificationsDataset, projectID)

		datasetMetadata := bigquery.DatasetMetadata{
			Name:        "courier_notifications_export",
			Description: "dataset for courier notifications",
		}

		if err := dataset.Create(ctx, &datasetMetadata); err != nil {
			return err
		}
	}

	schema := getSchema()

	sortedDays := sortKeys(messagesPerDayMap)

	for _, day := range sortedDays {
		messagesBQDay, ok := messagesPerDayMap[day]
		if !ok {
			return errors.New("data in a day not found")
		}

		partitionedTable := table + "$" + day.Format("20060102")

		requestData := bqutils.BigQueryTableLoaderRequest{
			DestinationProjectID:   projectID,
			DestinationDatasetID:   domain.CourierNotificationsDataset,
			DestinationTableName:   partitionedTable,
			ObjectDir:              table,
			ConfigJobID:            table,
			WriteDisposition:       bigquery.WriteTruncate,
			RequirePartitionFilter: false,
			PartitionField:         "enqueued",
		}

		bqRequest := bqutils.BigQueryTableLoaderParams{
			Client: bq,
			Schema: &schema,
			Data:   &requestData,
			Rows:   make([]interface{}, 0),
		}

		for _, notification := range messagesBQDay {
			bqRequest.Rows = append(bqRequest.Rows, notification)
		}

		if err := bqutils.BigQueryTableLoader(ctx, bqRequest); err != nil {
			return fmt.Errorf("error loading data to bq table %s, partition: %s, caused by %s", bqRequest.Data.DestinationTableName, day, err.Error())
		}
	}

	return nil
}

func getSchema() bigquery.Schema {
	return bigquery.Schema{
		{Name: "id", Type: bigquery.StringFieldType, Required: true},
		{Name: "enqueued", Type: bigquery.TimestampFieldType, Required: true},
		{Name: "sent", Type: bigquery.TimestampFieldType},
		{Name: "delivered", Type: bigquery.TimestampFieldType},
		{Name: "opened", Type: bigquery.TimestampFieldType},
		{Name: "clicked", Type: bigquery.TimestampFieldType},
		{Name: "status", Type: bigquery.StringFieldType, Required: true},
		{Name: "recipient", Type: bigquery.StringFieldType},
		{Name: "event", Type: bigquery.StringFieldType},
		{Name: "notification", Type: bigquery.StringFieldType},
		{Name: "error", Type: bigquery.StringFieldType},
		{Name: "reason", Type: bigquery.StringFieldType},
	}
}

func sortKeys(messages map[time.Time][]*domain.MessageBQ) []time.Time {
	keys := make([]time.Time, 0, len(messages))

	for key := range messages {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Before(keys[j])
	})

	return keys
}
