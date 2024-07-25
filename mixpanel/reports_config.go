package mixpanel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/mixpanel"
)

type Report struct {
	ComputedAt time.Time        `json:"computed_at" firestore:"computed_at"`
	Data       DataReportConfig `json:"data" firestore:"data"`
	LegendSize int64            `json:"legend_size" firestore:"legend_size"`
}

type DataReportConfig struct {
	Series []string          `json:"series" firestore:"series"`
	Values EventReportConfig `json:"values" firestore:"values"`
}

type EventReportConfig struct {
	EventName map[string]int64 `json:"customer.open" firestore:"customer.open"`
}

const (
	activeUsersReportsPath = "integrations/mixpanel/activeUsersReports"
)

type ActiveUsersReportService struct {
	*logger.Logging
	*connection.Connection
	*mixpanel.Service
}

func NewActiveUsersReportService(log *logger.Logging, conn *connection.Connection) (*ActiveUsersReportService, error) {
	return &ActiveUsersReportService{
		log,
		conn,
		mixpanel.NewService(),
	}, nil
}

func (s *ActiveUsersReportService) updateActiveUsersReport(ctx context.Context, customerID string, report []byte) error {
	fs := s.Firestore(ctx)

	var m Report
	if err := json.Unmarshal(report, &m); err != nil {
		return err
	}

	if _, err := fs.Collection(activeUsersReportsPath).Doc(customerID).Set(ctx, m); err != nil {
		return err
	}

	return nil
}

func (s *ActiveUsersReportService) getCustomerActiveUsersReport(ctx context.Context, customerID string) (*firestore.DocumentSnapshot, error) {
	fs := s.Firestore(ctx)
	return fs.Collection(activeUsersReportsPath).Doc(customerID).Get(ctx)
}

func (s *ActiveUsersReportService) GetActiveUsersReport(ctx context.Context, customerID string, params map[string][]string) (Report, error) {
	logger := s.Logger(ctx)

	var report Report

	reportDoc, err := s.getCustomerActiveUsersReport(ctx, customerID)
	if err != nil {
		logger.Info(err)
	} else {
		if reportDoc.Exists() {
			if err := reportDoc.DataTo(&report); err != nil {
				logger.Info(err)
			}
			// is it the same day? then return the cached report
			if report.ComputedAt.Add(time.Hour * 24).After(time.Now()) {
				return report, nil
			}
		}
	}

	// get the report from mixpanel
	res, err := s.QuerySegmentationReport(ctx, params)
	if err != nil {
		logger.Info(err)
		// return the cached report if we got error from mixpanel
		if !report.ComputedAt.IsZero() {
			return report, nil
		}
		// If there is no cached report, return an error
		return report, err
	}

	// save the report to firestore
	if err := s.updateActiveUsersReport(ctx, customerID, res); err != nil {
		return report, err
	}

	if err := json.Unmarshal(res, &report); err != nil {
		return report, err
	}

	return report, nil
}

func (s *ActiveUsersReportService) BuildActiveUsersReportConfig(ctx context.Context, customerID string) map[string][]string {
	now := time.Now()
	lastMonth := now.AddDate(0, -1, 0)

	return map[string][]string{
		"project_id": {"1834975"},
		"from_date":  {lastMonth.Format("2006-01-02")},
		"to_date":    {now.Format("2006-01-02")},
		"unit":       {"day"},
		"event":      {"customer.open"},
		"type":       {"unique"},
		"where":      {fmt.Sprintf(`properties["Customer ID"]="%s" and properties["DoiT Employee?"]=false`, customerID)},
	}
}
