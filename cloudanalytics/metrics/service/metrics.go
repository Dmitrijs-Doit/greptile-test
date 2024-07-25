package service

import (
	"context"

	extendedMetricDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/config/dal"
	datahubMetricMetricDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/datahubmetric"
	datahubMetricDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/datahubmetric/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal"
	metricsDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal/iface"
	reportsDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	reportsDALIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type MetricsService struct {
	loggerProvider    logger.Provider
	metricsDAL        metricsDAL.Metrics
	reportsDAL        reportsDALIface.Reports
	datahubMetricDAL  datahubMetricDalIface.DataHubMetricFirestore
	extendedMetricDAL extendedMetricDal.Configs
}

func NewMetricsService(
	loggerProvider logger.Provider,
	conn *connection.Connection) *MetricsService {
	return &MetricsService{
		loggerProvider,
		dal.NewMetricsFirestoreWithClient(conn.Firestore),
		reportsDAL.NewReportsFirestoreWithClient(conn.Firestore),
		datahubMetricMetricDal.NewDataHubMetricFirestoreWithClient(conn.Firestore),
		extendedMetricDal.NewConfigsFirestoreWithClient(conn.Firestore),
	}
}

func (s *MetricsService) DeleteMany(ctx context.Context, req DeleteMetricsRequest) error {
	if err := s.checkMetricsNotPreset(ctx, req.IDs); err != nil {
		return err
	}

	if err := s.checkMetricsNotInUse(ctx, req.IDs); err != nil {
		return err
	}

	return s.metricsDAL.DeleteMany(ctx, req.IDs)
}
