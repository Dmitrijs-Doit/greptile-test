package dal

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/iterator"
	"gotest.tools/assert"

	doitBQMocks "github.com/doitintl/bigquery/mocks"
	discoveryDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/discovery/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestRunDiscoveryQueryAndProcessRows(t *testing.T) {
	type fields struct {
		loggerMocks       *loggerMocks.ILogger
		queryHandlerMocks *doitBQMocks.QueryHandler
		iteratorMock      *doitBQMocks.RowIterator
	}

	ctx := context.Background()
	readError := errors.New("read error")
	bq := &bigquery.Client{}
	emptyRowProcessor := func(row *[]bigquery.Value) {
		// NOOP for testing
	}
	testQuery := "-- TEST QUERY"

	testIterator := doitBQMocks.NewRowIterator(t)
	testIteratorErr := errors.New("iterator error")

	tests := []struct {
		name      string
		on        func(*fields)
		want      []*bigquery.ValuesSaver
		wantedErr error
	}{
		{
			name:      "Read fails",
			wantedErr: readError,
			on: func(f *fields) {
				f.queryHandlerMocks.On("Read", ctx, mock.AnythingOfType("*bigquery.Query")).
					Return(nil, readError).Once()
			},
		},
		{
			name:      "Iterator fails",
			wantedErr: testIteratorErr,
			on: func(f *fields) {
				f.queryHandlerMocks.On("Read", ctx, mock.AnythingOfType("*bigquery.Query")).
					Return(f.iteratorMock, nil).Once()
				f.iteratorMock.On("Next", mock.Anything).
					Return(testIteratorErr).Once()
			},
		},
		{
			name: "Happy path",
			on: func(f *fields) {
				f.queryHandlerMocks.On("Read", ctx, mock.AnythingOfType("*bigquery.Query")).
					Return(f.iteratorMock, nil).Once()
				f.iteratorMock.On("Next", mock.Anything).
					Return(nil).Once()
				f.iteratorMock.On("Next", mock.Anything).
					Return(iterator.Done).Once()
			},
			want: []*bigquery.ValuesSaver{
				{
					Schema: discoveryDomain.TablesSchema,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				loggerMocks:       loggerMocks.NewILogger(t),
				queryHandlerMocks: doitBQMocks.NewQueryHandler(t),
				iteratorMock:      testIterator,
			}

			d := &BigqueryDAL{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return fields.loggerMocks
				},
				queryHandler: fields.queryHandlerMocks,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			got, gotErr := d.runDiscoveryQueryAndProcessRows(ctx, bq, emptyRowProcessor, testQuery)
			if gotErr != nil && tt.wantedErr == nil {
				t.Errorf("BQLens.TestRunDiscoveryQueryAndProcessRows() error = %v, wantErr %v", gotErr, tt.wantedErr)
			}

			if tt.wantedErr != nil {
				assert.Equal(t, tt.wantedErr.Error(), gotErr.Error())
			}

			if tt.want != nil {
				assert.DeepEqual(t, tt.want, got)
			}
		})
	}
}

func Test_compareMetadata(t *testing.T) {
	type args struct {
		metaData1 *bigquery.TableMetadata
		metaData2 *bigquery.TableMetadata
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Equal metadata",
			args: args{
				metaData1: &bigquery.TableMetadata{
					Schema: bigquery.Schema{
						{Name: "ts", Type: bigquery.TimestampFieldType},
						{Name: "labels", Type: bigquery.RecordFieldType, Repeated: true,
							Schema: bigquery.Schema{
								{Name: "key", Type: bigquery.StringFieldType},
								{Name: "value", Type: bigquery.StringFieldType},
							},
						},
					},
				},
				metaData2: &bigquery.TableMetadata{
					Schema: bigquery.Schema{
						{Name: "ts", Type: bigquery.TimestampFieldType},
						{Name: "labels", Type: bigquery.RecordFieldType, Repeated: true,
							Schema: bigquery.Schema{
								{Name: "key", Type: bigquery.StringFieldType},
								{Name: "value", Type: bigquery.StringFieldType},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Different metadata",
			args: args{
				metaData1: &bigquery.TableMetadata{
					Schema: bigquery.Schema{
						{Name: "ts", Type: bigquery.TimestampFieldType},
						{Name: "labels", Type: bigquery.RecordFieldType, Repeated: true,
							Schema: bigquery.Schema{
								{Name: "key", Type: bigquery.StringFieldType},
								{Name: "value", Type: bigquery.StringFieldType},
							},
						},
					},
				},
				metaData2: &bigquery.TableMetadata{
					Schema: bigquery.Schema{
						{Name: "ts", Type: bigquery.TimestampFieldType},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareMetadata(tt.args.metaData1, tt.args.metaData2); got != tt.want {
				t.Errorf("compareMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}
