package dal

import (
	"context"
	"net/http"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"

	bqIface "github.com/doitintl/bigquery/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func Test_historicalJobs_GetSinkFirstRecordTime(t *testing.T) {
	ctx := context.Background()

	testBQClient, err := bigquery.NewClient(ctx, common.TestProjectID)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx             context.Context
		client          *bigquery.Client
		projectLocation string
		projectID       string
	}

	tests := []struct {
		name    string
		h       *HistoricalJobs
		args    args
		wantErr bool
	}{
		{
			name: "Test GetSinkFirstRecordTime",
			h:    &HistoricalJobs{},
			args: args{
				ctx:             ctx,
				client:          testBQClient,
				projectLocation: "US",
				projectID:       common.TestProjectID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HistoricalJobs{}

			got, err := h.GetSinkFirstRecordTime(tt.args.ctx, tt.args.client, tt.args.projectLocation, tt.args.projectID)

			if (err != nil) != tt.wantErr {
				t.Errorf("historicalJobs.GetSinkFirstRecordTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			t.Log(got)
		})
	}
}

func TestHistoricalJobs_GetJobsList(t *testing.T) {
	t.Skip("long running test - test localy")

	ctx := context.Background()

	testBQClient, err := bigquery.NewClient(ctx, common.TestProjectID)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		loggerMocks *loggerMocks.ILogger
	}

	type args struct {
		ctx             context.Context
		client          *bigquery.Client
		projectID       string
		minCreationTime time.Time
		maxCreationTime time.Time
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test GetJobsList",
			args: args{
				ctx:             ctx,
				client:          testBQClient,
				projectID:       common.TestProjectID,
				minCreationTime: time.Now().Add(-time.Hour * 24),
				maxCreationTime: time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				loggerMocks: loggerMocks.NewILogger(t),
			}

			h := &HistoricalJobs{
				loggerProvider: func(_ context.Context) logger.ILogger {
					return fields.loggerMocks
				},
			}

			got, err := h.GetJobsList(tt.args.ctx, tt.args.client, tt.args.projectID, tt.args.minCreationTime, tt.args.maxCreationTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("HistoricalJobs.GetJobsList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for job := range got {
				t.Log(job)
			}
		})
	}
}

type mockInserter struct {
	Throttle int
}

func (m *mockInserter) Put(_ context.Context, src interface{}) error {
	bools := src.([]*bool)

	if len(bools) > m.Throttle {
		return &googleapi.Error{
			Code: http.StatusRequestEntityTooLarge,
		}
	}

	for _, b := range bools {
		*b = true
	}

	return nil
}

func Test_insertRecords(t *testing.T) {
	generateRows := func(n int) []*bool {
		var rows []*bool

		for i := 0; i < n; i++ {
			rows = append(rows, new(bool))
		}

		return rows
	}

	type args struct {
		ctx      context.Context
		inserter bqIface.IfcInserter
		rows     []*bool
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test insertRecords Throttling",
			args: args{
				ctx:      context.Background(),
				inserter: &mockInserter{Throttle: 40},
				rows:     generateRows(111),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := insertRecords(tt.args.ctx, tt.args.inserter, tt.args.rows); (err != nil) != tt.wantErr {
				t.Errorf("insertRecords2() error = %v, wantErr %v", err, tt.wantErr)
			}

			for i, row := range tt.args.rows {
				if !*row {
					t.Errorf("row not inserted: %d", i)
				}
			}
		})
	}
}
