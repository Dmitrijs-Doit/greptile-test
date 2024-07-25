package stats

import (
	"context"
	"errors"
	"testing"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	reportsDalMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func TestReportStatsService_UpdateReportStats(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider logger.Provider
		reportDAL      *reportsDalMock.Reports
	}

	type args struct {
		ctx           context.Context
		reportID      string
		origin        domain.QueryOrigin
		resultDetails map[string]interface{}
	}

	reportID := "123"

	serverDurationMsVal := int64(4)
	totalBytesProcessedKey := int64(8)

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successfully update stats when origin is API",
			args: args{
				ctx:      ctx,
				reportID: reportID,
				origin:   domain.QueryOriginReportsAPI,
				resultDetails: map[string]interface{}{
					report.ServerDurationMsKey:    serverDurationMsVal,
					report.TotalBytesProcessedKey: totalBytesProcessedKey,
				},
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportDAL.On(
					"UpdateStats",
					testutils.ContextBackgroundMock,
					reportID,
					domain.QueryOriginReportsAPI,
					&serverDurationMsVal,
					&totalBytesProcessedKey,
				).
					Return(nil).
					Once()
			},
		},
		{
			name: "do not update stats when origin is client",
			args: args{
				ctx:      ctx,
				reportID: reportID,
				origin:   domain.QueryOriginClient,
				resultDetails: map[string]interface{}{
					report.ServerDurationMsKey:    serverDurationMsVal,
					report.TotalBytesProcessedKey: totalBytesProcessedKey,
				},
			},
			wantErr: false,
		},
		{
			name: "do not update stats when origin is API and total bytes processed is 0, meaning it's a cached response",
			args: args{
				ctx:      ctx,
				reportID: reportID,
				origin:   domain.QueryOriginClient,
				resultDetails: map[string]interface{}{
					report.ServerDurationMsKey:    serverDurationMsVal,
					report.TotalBytesProcessedKey: 0,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				reportDAL:      reportsDalMock.NewReports(t),
			}

			s := &ReportStatsService{
				loggerProvider: tt.fields.loggerProvider,
				reportDAL:      tt.fields.reportDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.UpdateReportStats(ctx, tt.args.reportID, tt.args.origin, tt.args.resultDetails)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportStatsService.UpdateReportStats() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("ReportStatsService.UpdateReportStats() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}
