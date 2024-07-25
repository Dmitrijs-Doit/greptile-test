package stats

import (
	"context"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type ReportStatsService struct {
	loggerProvider logger.Provider
	reportDAL      iface.Reports
}

func NewReportStatsService(
	loggerProvider logger.Provider,
	reportDAL iface.Reports,
) (*ReportStatsService, error) {
	return &ReportStatsService{
		loggerProvider,
		reportDAL,
	}, nil
}

func (s *ReportStatsService) UpdateReportStats(
	ctx context.Context,
	reportID string,
	origin domain.QueryOrigin,
	resultDetails map[string]interface{},
) error {
	if origin == domain.QueryOriginClient || origin == domain.QueryOriginClientReservation {
		return nil
	}

	var serverDurationMs *int64

	if val, ok := resultDetails[report.ServerDurationMsKey]; ok {
		if valInt, ok := val.(int64); ok {
			serverDurationMs = &valInt
		}
	}

	var totalBytesProcessed *int64

	if val, ok := resultDetails[report.TotalBytesProcessedKey]; ok {
		if valInt, ok := val.(int64); ok {
			totalBytesProcessed = &valInt
		}
	}

	if totalBytesProcessed != nil && *totalBytesProcessed == 0 {
		return nil
	}

	if err := s.reportDAL.UpdateStats(
		ctx,
		reportID,
		origin,
		serverDurationMs,
		totalBytesProcessed,
	); err != nil {
		return err
	}

	return nil
}
