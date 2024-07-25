package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
)

func TestEvent_newEventFromRawEvent(t *testing.T) {
	type args struct {
		dataset  string
		schema   Schema
		rowPos   int
		rawEvent []string
	}

	cloud := "datadog"
	id := "73f8b9da-ebb2-4046-8226-1015ee94b499"

	tests := []struct {
		name          string
		args          args
		expectedEvent *Event
		expectedErr   []errormsg.ErrorMsg
	}{
		{
			name: "successful rawEvent matches the schema",
			args: args{
				dataset: cloud,
				schema: Schema{
					{
						FieldType: "usage_date",
						FieldKey:  "usage_date",
					},
					{
						FieldType: "event_id",
						FieldKey:  "event_id",
					},
					{
						FieldType: "fixed",
						FieldKey:  "project_id",
					},
					{
						FieldType: "project_label",
						FieldKey:  "app",
					},
					{
						FieldType: "label",
						FieldKey:  "house",
					},
					{
						FieldType: "metric",
						FieldKey:  "cost",
					},
				},
				rowPos:   2,
				rawEvent: []string{"2024-03-01T00:00:00Z", id, "pr1", "some_app", "adoption", "12"},
			},
			expectedEvent: &Event{
				Cloud: &cloud,
				ID:    &id,
				Time:  time.Date(2024, 03, 1, 0, 0, 0, 0, time.UTC),
				Dimensions: []*Dimension{
					{
						Type:  SchemaFieldTypeFixed,
						Key:   "project_id",
						Value: "pr1",
					},
					{
						Type:  SchemaFieldTypeProjectLabel,
						Key:   "app",
						Value: "some_app",
					},
					{
						Type:  SchemaFieldTypeLabel,
						Key:   "house",
						Value: "adoption",
					},
				},
				Metrics: []*Metric{
					{
						Type:  "cost",
						Value: 12,
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "error when rawEvent length do not match the schema",
			args: args{
				dataset: cloud,
				schema: Schema{
					{
						FieldType: "usage_date",
						FieldKey:  "usage_date",
					},
					{
						FieldType: "event_id",
						FieldKey:  "event_id",
					},
					{
						FieldType: "fixed",
						FieldKey:  "project_id",
					},
					{
						FieldType: "project_label",
						FieldKey:  "app",
					},
					{
						FieldType: "label",
						FieldKey:  "house",
					},
					{
						FieldType: "metric",
						FieldKey:  "cost",
					},
				},
				rowPos:   2,
				rawEvent: []string{"2024-03-01T00:00:00Z", id, "pr1", "some_app", "12"},
			},
			expectedEvent: nil,
			expectedErr: []errormsg.ErrorMsg{
				{
					Field:   "row: 2",
					Message: "number of columns does not match the number of header fields",
				},
			},
		},
		{
			name: "error on invalid datetime and metric value",
			args: args{
				dataset: cloud,
				schema: Schema{
					{
						FieldType: "usage_date",
						FieldKey:  "usage_date",
					},
					{
						FieldType: "event_id",
						FieldKey:  "event_id",
					},
					{
						FieldType: "fixed",
						FieldKey:  "project_id",
					},
					{
						FieldType: "metric",
						FieldKey:  "cost",
					},
				},
				rowPos:   2,
				rawEvent: []string{"2024-03-01T00:WRONG:00Z", id, "pr1", "invalid-12"},
			},
			expectedEvent: nil,
			expectedErr: []errormsg.ErrorMsg{
				{
					Field:   "row: 2",
					Message: "invalid value '2024-03-01T00:WRONG:00Z' provided for column 'usage_date'",
				},
				{
					Field:   "row: 2",
					Message: "invalid value 'invalid-12' provided for column 'usage_date'",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			event, errs := newEventFromRawEvent(
				tt.args.dataset,
				tt.args.schema,
				tt.args.rowPos,
				tt.args.rawEvent,
			)

			if errs == nil {
				assert.Equal(t, event, tt.expectedEvent)
			}

			assert.Equal(t, errs, tt.expectedErr)
		})
	}
}
