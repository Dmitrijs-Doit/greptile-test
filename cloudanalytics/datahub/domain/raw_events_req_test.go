package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
)

func TestRawEventsReq(t *testing.T) {
	type args struct {
		rawEventsReq RawEventsReq
	}

	tests := []struct {
		name        string
		args        args
		expectedErr []errormsg.ErrorMsg
	}{
		{
			name: "successful validation",
			args: args{
				rawEventsReq: RawEventsReq{
					Dataset: "datadog",
					Source:  "csv",
					Schema:  []string{"field1, field2"},
					RawEvents: [][]string{
						{
							"111", "222",
						},
						{
							"333", "444",
						},
					},
					Filename: "somefilename",
					Execute:  true,
				},
			},
			expectedErr: nil,
		},
		{
			name: "errors on validation",
			args: args{
				rawEventsReq: RawEventsReq{},
			},
			expectedErr: []errormsg.ErrorMsg{
				{
					Field:   "source",
					Message: "`source` field can not be empty",
				},
				{
					Field:   "dataset",
					Message: "`dataset` field can not be empty",
				},
				{
					Field:   "filename",
					Message: "`filename` field can not be empty",
				},
				{
					Field:   "schema",
					Message: "`schema` field can not be empty",
				},
				{
					Field:   "rawEvents",
					Message: "`rawEvents` field can not be empty",
				},
			},
		},
		{
			name: "not 'csv' source is not supported",
			args: args{
				rawEventsReq: RawEventsReq{
					Dataset: "datadog",
					Source:  "blabla",
					Schema:  []string{"field1, field2"},
					RawEvents: [][]string{
						{
							"111", "222",
						},
						{
							"333", "444",
						},
					},
					Filename: "somefilename",
					Execute:  true,
				},
			},
			expectedErr: []errormsg.ErrorMsg{
				{
					Field:   SourceField,
					Message: InvalidSourceTypeMsg,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			errs := tt.args.rawEventsReq.Validate()

			assert.Equal(t, errs, tt.expectedErr)
		})
	}
}
