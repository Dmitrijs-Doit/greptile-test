package exportservice

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/bqutils"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/mixpanel"
)

const (
	mixpanelEventsTable   = "mixpanel_events"
	mixpanelEventsDataset = "mixpanel_events_export"
	errorPrefix           = "mixpanel event loader - "
	layout                = "2006-01-02"
	hourlyRateLimit       = 60
	dayInHours            = 24
)

type EventExporterService struct {
	*logger.Logging
	*connection.Connection
	*mixpanel.Service
	bigQueryFromContextFun connection.BigQueryFromContextFun
}

func NewEventExporterService(log *logger.Logging, conn *connection.Connection) (*EventExporterService, error) {
	return &EventExporterService{
		log,
		conn,
		mixpanel.NewService(),
		conn.Bigquery,
	}, nil
}

func (e *EventExporterService) GetEvents(ctx *gin.Context, interval mixpanel.EventInterval) (map[time.Time][]mixpanel.Event, error) {
	l := e.Logger(ctx)

	startDate, endDate, err := getInterval(interval)
	if err != nil {
		return nil, fmt.Errorf("%s %s", errorPrefix, err.Error())
	}

	eventsByPartition := make(map[time.Time][]mixpanel.Event)

	remainingPartitionsToBackfill := getPartitionsToBackfill(startDate, endDate)

	chunkSize := int(math.Ceil(float64(remainingPartitionsToBackfill) / float64(hourlyRateLimit)))
	currentStart := startDate

	for remainingPartitionsToBackfill > 0 {
		if remainingPartitionsToBackfill < chunkSize {
			chunkSize = remainingPartitionsToBackfill
		}

		currentEnd := currentStart.AddDate(0, 0, chunkSize-1)

		eventsStr, err := e.GetMixpanelEventsFromMixpanelClient(ctx, currentStart, currentEnd)
		if err != nil {
			return nil, fmt.Errorf("%s %s", errorPrefix, err.Error())
		}

		parsedEvents := parseStringIntoEvent(l, eventsStr)
		eventsByPartition = mergeMaps(eventsByPartition, parsedEvents)
		remainingPartitionsToBackfill -= chunkSize
		currentStart = currentEnd.AddDate(0, 0, 1)
	}

	return eventsByPartition, nil
}

func getSchema() bigquery.Schema {
	return bigquery.Schema{
		{Name: "event", Type: bigquery.StringFieldType},
		{Name: "time", Type: bigquery.TimestampFieldType},
		{Name: "distinctID", Type: bigquery.StringFieldType},
		{Name: "customerID", Type: bigquery.StringFieldType},
		{Name: "customerDomain", Type: bigquery.StringFieldType},
		{Name: "email", Type: bigquery.StringFieldType},
		{Name: "mpApiEndpoint", Type: bigquery.StringFieldType},
		{Name: "mpApiTimestampMs", Type: bigquery.IntegerFieldType},
		{Name: "feature", Type: bigquery.StringFieldType},
		{Name: "method", Type: bigquery.StringFieldType},
		{Name: "status", Type: bigquery.IntegerFieldType},
		{Name: "userAgent", Type: bigquery.StringFieldType},
	}
}

func (e *EventExporterService) getTableLoaderRequest(ctx *gin.Context, events []mixpanel.Event) (bqutils.BigQueryTableLoaderParams, error) {
	projectID := common.ProjectID
	client := e.bigQueryFromContextFun(ctx)
	requestData := bqutils.BigQueryTableLoaderRequest{
		DestinationProjectID:   projectID,
		DestinationDatasetID:   mixpanelEventsDataset,
		DestinationTableName:   mixpanelEventsTable,
		ObjectDir:              mixpanelEventsTable,
		ConfigJobID:            mixpanelEventsTable,
		WriteDisposition:       bigquery.WriteTruncate,
		RequirePartitionFilter: false,
		PartitionField:         "time",
		Clustering:             &[]string{"customerID"},
	}
	schema := getSchema()

	schemaIsValid, err := validateSchema(mixpanel.BQEvent{}, schema)
	if err != nil {
		return bqutils.BigQueryTableLoaderParams{}, err
	}

	if !schemaIsValid {
		return bqutils.BigQueryTableLoaderParams{}, fmt.Errorf("%s schema does not match row", errorPrefix)
	}

	loaderAttributes :=
		bqutils.BigQueryTableLoaderParams{
			Client: client,
			Schema: &schema,
			Data:   &requestData,
		}
	rows := make([]interface{}, len(events))

	for i, v := range events {
		bqEvent := convertEventToBQStruct(v)
		rows[i] = bqEvent
	}

	loaderAttributes.Rows = rows

	return loaderAttributes, nil
}
func (e *EventExporterService) ExportToBQ(ctx *gin.Context, events map[time.Time][]mixpanel.Event) error {
	l := e.Logger(ctx)

	tableExists := false
	schemaIsSame := false
	sortedKeys := sortKeys(events)

	for _, key := range sortedKeys {
		request, err := e.getTableLoaderRequest(ctx, events[key])
		if err != nil {
			return err
		}

		if !tableExists {
			var err error

			tableExists, _, err = common.BigQueryTableExists(ctx,
				request.Client,
				request.Data.DestinationProjectID,
				request.Data.DestinationDatasetID,
				request.Data.DestinationTableName)
			if err != nil {
				return err
			}
		}

		if tableExists {
			if !schemaIsSame {
				md, err := request.Client.Dataset(request.Data.DestinationDatasetID).Table(request.Data.DestinationTableName).Metadata(ctx)
				if err != nil {
					l.Errorf("failed to get table: %s", err)
					return err
				}

				if len(md.Schema) != len(getSchema()) {
					if _, err := request.Client.Dataset(request.Data.DestinationDatasetID).Table(request.Data.DestinationTableName).Update(ctx, bigquery.TableMetadataToUpdate{
						Schema: getSchema(),
					}, ""); err != nil {
						l.Errorf("failed to update table schema: %s", err)
						return err
					}
				}

				schemaIsSame = true
			}

			tableName := mixpanelEventsTable
			tableName += "$" + key.Format("20060102")
			request.Data.DestinationTableName = tableName
		}

		if err := bqutils.BigQueryTableLoader(ctx, request); err != nil {
			return fmt.Errorf("error loading data to bq table %s caused by %s", request.Data.DestinationTableName, err.Error())
		}

		tableExists = true
	}

	return nil
}

func sortKeys(events map[time.Time][]mixpanel.Event) []time.Time {
	// Step 1: Extract the keys (partitions) into a separate slice
	keys := make([]time.Time, 0, len(events))
	for key := range events {
		keys = append(keys, key)
	}

	// Step 2: Sort the slice of keys in ascending order
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Before(keys[j])
	})

	return keys
}

func getInterval(interval mixpanel.EventInterval) (time.Time, time.Time, error) {
	// we check for the previous day's events, to get the entire 24 hours worth of events
	var (
		parsedStartDate time.Time
		parsedEndDate   time.Time
		err             error
	)

	// if input is not provided, no need to try to parse it
	if interval.StartDate != "" && interval.EndDate != "" {
		parsedStartDate, err = time.Parse(layout, interval.StartDate)
		if err != nil {
			return parsedStartDate, parsedEndDate, fmt.Errorf("failed to parse startDate: %s, error: %s", interval.StartDate, err)
		}

		parsedEndDate, err = time.Parse(layout, interval.EndDate)
		if err != nil {
			return parsedStartDate, parsedEndDate, fmt.Errorf("failed to parse endDate: %s, error: %s", interval.EndDate, err)
		}
	}

	yesterday := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day()-1, 0, 0, 0, 0, time.UTC)
	if parsedStartDate.IsZero() {
		parsedStartDate = yesterday
	}

	if parsedEndDate.IsZero() {
		parsedEndDate = yesterday
	}

	if parsedStartDate.After(parsedEndDate) {
		return parsedStartDate, parsedEndDate, fmt.Errorf("start date is after end date, cannot export data")
	}

	today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
	if parsedEndDate.After(today) {
		parsedEndDate = today
	}

	if parsedStartDate.After(yesterday) {
		parsedStartDate = yesterday
	}

	return parsedStartDate, parsedEndDate, nil
}
func convertEventToBQStruct(event mixpanel.Event) mixpanel.BQEvent {
	res := mixpanel.BQEvent{
		Event:            event.Event,
		Time:             event.Properties.Time,
		DistinctID:       event.Properties.DistinctID,
		CustomerID:       event.Properties.CustomerID,
		CustomerDomain:   event.Properties.CustomerDomain,
		Email:            event.Properties.Email,
		MpAPIEndpoint:    event.Properties.MpAPIEndpoint,
		MpAPITimestampMS: event.Properties.MpAPITimestampMS,
		Feature:          event.Properties.Feature,
		Method:           event.Properties.Method,
		Status:           event.Properties.Status,
		UserAgent:        event.Properties.UserAgent,
	}

	return res
}

func validateSchema(row mixpanel.BQEvent, expectedSchema bigquery.Schema) (bool, error) {
	inferredSchema, err := bigquery.InferSchema(row)
	if err != nil {
		return false, err
	}

	if len(inferredSchema) !=
		len(expectedSchema) {
		return false, nil
	}

	for i := range inferredSchema {
		if inferredSchema[i].Name != expectedSchema[i].Name || inferredSchema[i].Type != expectedSchema[i].Type {
			return false, nil
		}
	}

	return true, nil
}

func mergeMaps(map1 map[time.Time][]mixpanel.Event, map2 map[time.Time][]mixpanel.Event) map[time.Time][]mixpanel.Event {
	mergedMap := make(map[time.Time][]mixpanel.Event)

	for k, v := range map1 {
		mergedMap[k] = append(mergedMap[k], v...)
	}

	for k, v := range map2 {
		mergedMap[k] = append(mergedMap[k], v...)
	}

	return mergedMap
}

func parseStringIntoEvent(l logger.ILogger, eventsStr []string) map[time.Time][]mixpanel.Event {
	events := make(map[time.Time][]mixpanel.Event)

	for _, eventStr := range eventsStr {
		if eventStr == "" {
			continue
		}

		var event mixpanel.Event

		event, err := event.DecodeJSONIntoEventStruct([]byte(eventStr))
		if err != nil {
			l.Errorf("failed to decode JSON: %s", err)
			continue
		}

		if event.Event != mixpanel.ExternalAPIEvent {
			// a client event has a slightly different schema:
			event, err = generateEventFromClientEventString(l, eventStr)
			if err != nil {
				l.Errorf("failed to genereate event from client event string: %s", err)
				continue
			}
		}

		if event.Properties.CustomerID == "" {
			continue
		}

		if event.Properties.TimeMillisecond == 0 {
			continue
		}

		if !strings.Contains(event.Properties.UserAgent, "Zapier") {
			event.Properties.UserAgent = ""
		}

		if strings.Contains(event.Properties.Email, "doit") {
			continue
		}

		key := time.Date(event.Properties.Time.Year(), event.Properties.Time.Month(), event.Properties.Time.Day(), 0, 0, 0, 0, time.UTC)
		events[key] = append(events[key], event)
	}

	return events
}

func (e *EventExporterService) GetMixpanelEventsFromMixpanelClient(ctx *gin.Context, chunkStartDate, chunkEndDate time.Time) ([]string, error) {
	return e.ExportMixpanelEvents(ctx, chunkStartDate, chunkEndDate)
}

func generateEventFromClientEventString(l logger.ILogger, eventStr string) (mixpanel.Event, error) {
	var clientEvent mixpanel.ClientEvent

	clientEvent, err := clientEvent.DecodeJSONIntoClientEventStruct([]byte(eventStr))
	if err != nil {
		l.Errorf("failed to decode JSON: %s", err)
		return mixpanel.Event{}, err
	}

	return mixpanel.Event{
		Event: clientEvent.Event,
		Properties: mixpanel.Properties{
			Time:             clientEvent.Properties.Time,
			TimeMillisecond:  clientEvent.Properties.TimeMillisecond,
			DistinctID:       clientEvent.Properties.DistinctID,
			CustomerID:       clientEvent.Properties.CustomerID,
			CustomerDomain:   clientEvent.Properties.PrimaryDomain,
			Email:            clientEvent.Properties.Email,
			MpAPIEndpoint:    clientEvent.Properties.MpAPIEndpoint,
			MpAPITimestampMS: clientEvent.Properties.MpAPITimestampMS,
			URL:              "",
			Feature:          "",
			Method:           "",
			RequestURL:       "",
			Status:           0,
			UserAgent:        "",
		},
	}, nil
}

func getPartitionsToBackfill(startDate time.Time, endDate time.Time) int {
	diffInHours := endDate.Add(time.Hour * 24).Sub(startDate).Hours()
	return int(diffInHours / dayInHours)
}
