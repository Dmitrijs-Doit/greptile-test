package scripts

import (
	"errors"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
)

const (
	reportsCollection = "dashboards/google-cloud-reports/savedReports"

	metricsCollection = "cloudAnalytics/metrics/cloudAnalyticsMetrics"
)

type CheckForMetricInReportsInput struct {
	ProjectID string   `json:"project_id"`
	MetricIDs []string `json:"metric_ids"`
}

func CheckForMetricInReports(ctx *gin.Context) []error {
	var params CheckForMetricInReportsInput
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, params.ProjectID)
	if err != nil {
		return []error{err}
	}

	for _, metricID := range params.MetricIDs {
		metricRef := fs.Collection(metricsCollection).Doc(metricID)
		if metricRef == nil {
			return []error{errors.New("metric not found")}
		}

		docSnaps, err := fs.Collection(reportsCollection).
			Where("config.calculatedMetric", "==", metricRef).
			Documents(ctx).GetAll()
		if err != nil {
			return []error{err}
		}

		var reportIDs []string
		for _, docSnap := range docSnaps {
			reportIDs = append(reportIDs, docSnap.Ref.ID)
		}

		if len(reportIDs) > 0 {
			println("metric found in reports:")

			for _, reportID := range reportIDs {
				println(reportID)
			}
		} else {
			println("metric not found in any reports")
		}
	}

	return nil
}
