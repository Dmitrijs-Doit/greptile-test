package scripts

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

type DeleteFirestoreDocsInput struct {
	ProjectID       string   `json:"projectId"`
	CustomerID      string   `json:"customerId"`
	Owner           string   `json:"owner"`
	Collections     []string `json:"collections"`
	OnlyUntitled    *bool    `json:"onlyUntitled"`
	MinTimeModified string   `json:"minTimeModified"`
}

var analyticsCollectionsMap = map[string]string{
	"reports":      "dashboards/google-cloud-reports/savedReports",
	"attributions": "dashboards/google-cloud-reports/attributions",
	"budgets":      "cloudAnalytics/budgets/cloudAnalyticsBudgets",
	"metrics":      "cloudAnalytics/metrics/cloudAnalyticsMetrics",
	"alerts":       "cloudAnalytics/alerts/cloudAnalyticsAlerts",
}

/*
DeleteAnalyticsFirestoreDocs deletes the default analytics documents in firestore

Example payload:
{
  "projectId": "doitintl-cmp-dev",
  "customerId": "xxxxxxxx",
  "collections": ["reports", "attributions", "budgets", "metrics", "alerts"] (can be one or more),
  "minTimeModified": "YYYY-MM-DD" (so that the resource with timeModified less from that will be deleted)
}

This will delete for the customer "xxxxxxxx" after the given date the documents that have
the default name for the "reports", "attributions", "budgets", "metrics" and "alerts" collections.
*/

func DeleteAnalyticsFirestoreDocs(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	var params DeleteFirestoreDocsInput

	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	if params.ProjectID == "" || params.CustomerID == "" || len(params.Collections) == 0 || params.MinTimeModified == "" {
		err := errors.New("invalid input parameters")
		return []error{err}
	}

	if params.OnlyUntitled == nil {
		l.Infof("OnlyUntitled parameter not provided, defaulting to true")

		params.OnlyUntitled = common.Bool(true)
	}

	minTime, err := time.Parse(times.YearMonthDayLayout, params.MinTimeModified)
	if err != nil {
		err := errors.New("invalid date parameter")
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, params.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	bw := fs.BulkWriter(ctx)
	customerRef := fs.Collection("customers").Doc(params.CustomerID)

	for _, collection := range params.Collections {
		collectionPath, ok := analyticsCollectionsMap[collection]
		if !ok {
			err := errors.New("invalid collection")
			return []error{err}
		}

		q := fs.Collection(collectionPath).Where("customer", "==", customerRef)

		if *params.OnlyUntitled {
			q = q.Where("name", "==", getDefaultName(collection))
		}

		if params.Owner != "" {
			q = q.Where("collaborators", "array-contains", collab.Collaborator{
				Email: params.Owner,
				Role:  collab.CollaboratorRoleOwner,
			})
		}

		docSnaps, err := q.Select("customer", "name", "timeModified").Documents(ctx).GetAll()
		if err != nil {
			return []error{err}
		}

		for _, docSnap := range docSnaps {
			if docSnap.Data()["timeModified"].(time.Time).Before(minTime) {
				l.Infof("Deleting document '%s'", docSnap.Ref.Path)

				if _, err := bw.Delete(docSnap.Ref); err != nil {
					return []error{err}
				}
			}
		}
	}

	bw.End()

	return nil
}

func getDefaultName(collection string) string {
	return fmt.Sprintf("%s %s", "Untitled", strings.TrimSuffix(collection, "s"))
}
