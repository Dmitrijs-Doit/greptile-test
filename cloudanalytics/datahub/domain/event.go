package domain

import (
	"fmt"
	"strconv"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
)

type Metric struct {
	Value float64 `json:"value"`
	Type  string  `json:"type"`
}

type Dimension struct {
	Key   string      `json:"key"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type Event struct {
	Cloud      *string      `json:"cloud"`
	ID         *string      `json:"id"`
	Dimensions []*Dimension `json:"dimensions"`
	Time       time.Time    `json:"time"`
	Metrics    []*Metric    `json:"metrics,omitempty"`
}

const maxErrsLimit = 100

func NewEventsFromRawEvents(dataset string, schema *Schema, rawEvents [][]string) ([]*Event, []errormsg.ErrorMsg) {
	var events []*Event

	var errs []errormsg.ErrorMsg

	for i, rawEvent := range rawEvents {
		rowN := i + 1
		event, validationErr := newEventFromRawEvent(
			dataset,
			*schema,
			rowN,
			rawEvent,
		)
		if validationErr != nil {
			errs = append(errs, validationErr...)

			// no reason to try to process even more rows, if we already have quite some invalid ones
			if len(errs) > maxErrsLimit {
				return nil, errs
			}
		} else {
			events = append(events, event)
		}
	}

	return events, errs
}

func newEventFromRawEvent(
	dataset string,
	schema Schema,
	rowPos int,
	rawEvent []string,
) (*Event, []errormsg.ErrorMsg) {
	var errs []errormsg.ErrorMsg

	event := Event{
		Cloud: &dataset,
	}

	if len(schema) != len(rawEvent) {
		errs = append(errs, errormsg.ErrorMsg{
			Field:   "row: " + strconv.Itoa(rowPos),
			Message: InvalidColumnsLengthMsg,
		})

		return nil, errs
	}

	for colPos, rawEventVal := range rawEvent {
		schemaField := schema[colPos]

		fieldWithPos := "row: " + strconv.Itoa(rowPos)

		switch schemaField.FieldType {
		case SchemaFieldTypeUsageDate:
			parsedTime, err := parseUsageDateField(rawEventVal)
			if err != nil {
				errs = append(errs, errormsg.ErrorMsg{
					Field:   fieldWithPos,
					Message: fmt.Sprintf(InvalidValueForColumnTpl, rawEventVal, SchemaFieldTypeUsageDate),
				})

				continue
			}

			event.Time = parsedTime
		case SchemaFieldTypeEventID:
			eventID := rawEventVal
			event.ID = &eventID
		case SchemaFieldTypeFixed,
			SchemaFieldTypeProjectLabel,
			SchemaFieldTypeLabel:
			event.Dimensions = append(event.Dimensions, &Dimension{
				Key:   schemaField.FieldKey,
				Type:  schemaField.FieldType,
				Value: rawEventVal,
			})
		case SchemaFieldTypeMetric:
			parsedMetricVal, err := strconv.ParseFloat(rawEventVal, 64)
			if err != nil {
				errs = append(errs, errormsg.ErrorMsg{
					Field:   fieldWithPos,
					Message: fmt.Sprintf(InvalidValueForColumnTpl, rawEventVal, SchemaFieldTypeUsageDate),
				})

				continue
			}

			event.Metrics = append(event.Metrics, &Metric{
				Type:  schemaField.FieldKey,
				Value: parsedMetricVal,
			})
		default:
			errs = append(errs, errormsg.ErrorMsg{
				Field:   strconv.Itoa(rowPos),
				Message: InvalidFieldTypeMsg,
			})
		}
	}

	return &event, errs
}
