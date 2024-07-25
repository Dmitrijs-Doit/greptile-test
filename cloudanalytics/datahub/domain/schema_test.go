package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
)

func TestSchema_NewSchema(t *testing.T) {
	type args struct {
		rawSchema []string
	}

	tests := []struct {
		name           string
		args           args
		expectedSchema *Schema
		expectedErr    []errormsg.ErrorMsg
	}{
		{
			name: "valid raw schema",
			args: args{
				rawSchema: []string{"usage_date", "event_id", "project_id", "project_label.app", "label.house", "metric.cost"},
			},
			expectedSchema: &Schema{
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
			expectedErr: nil,
		},
		{
			name: "valid raw schema multiple labels, metrics etc..",
			args: args{
				rawSchema: []string{
					"usage_date",
					"event_id",
					"project_id",
					"project_label.app",
					"project_label.location",
					"label.house",
					"label.team",
					"metric.cost",
					"metric.rides",
				},
			},
			expectedSchema: &Schema{
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
					FieldType: "project_label",
					FieldKey:  "location",
				},
				{
					FieldType: "label",
					FieldKey:  "house",
				},
				{
					FieldType: "label",
					FieldKey:  "team",
				},
				{
					FieldType: "metric",
					FieldKey:  "cost",
				},
				{
					FieldType: "metric",
					FieldKey:  "rides",
				},
			},
			expectedErr: nil,
		},
		{
			name: "no usage_date, no dimension, no metric",
			args: args{
				rawSchema: []string{"event_id"},
			},
			expectedErr: []errormsg.ErrorMsg{
				{
					Field:   "",
					Message: "at least one dimension or label must be provided in schema",
				},
				{
					Field:   "usage_date",
					Message: "field must be provided in schema",
				},
				{
					Field:   "",
					Message: "at least one 'metric' field must be provided in schema",
				},
			},
		},
		{
			name: "invalid empty schema",
			args: args{
				rawSchema: []string{},
			},
			expectedErr: []errormsg.ErrorMsg{
				{
					Field:   "schema",
					Message: "`schema` field can not be empty",
				},
			},
		},
		{
			name: "invalid label field, missing value after dot",
			args: args{
				rawSchema: []string{"usage_date", "event_id", "project_id", "project_label.app", "label.", "metric.cost"},
			},
			expectedErr: []errormsg.ErrorMsg{
				{
					Field:   "label.",
					Message: "label key can not be empty in schema",
				},
			},
		},
		{
			name: "invalid project_label field, missing value after dot",
			args: args{
				rawSchema: []string{"usage_date", "event_id", "project_id", "project_label.", "label.aaa", "metric.cost"},
			},
			expectedErr: []errormsg.ErrorMsg{
				{
					Field:   "project_label.",
					Message: "project_label key can not be empty in schema",
				},
			},
		},
		{
			name: "invalid metric field, missing value after dot",
			args: args{
				rawSchema: []string{"usage_date", "event_id", "project_id", "project_label.app", "label.aaa", "metric."},
			},
			expectedErr: []errormsg.ErrorMsg{
				{
					Field:   "metric.",
					Message: "metric key can not be empty in schema",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			schema, errs := NewSchema(tt.args.rawSchema)

			if errs == nil {
				assert.Equal(t, tt.expectedSchema, schema)
			}
			assert.Equal(t, tt.expectedErr, errs)
		})
	}
}
