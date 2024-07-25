package scripts

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
)

const (
	customerID          string = "BJFQxzDUkfSxy1kq8sSd"
	globalDashboardID   string = "nClxTjylyAbIQW9AQNRk"
	dashboardTypeGlobal string = "gke-lens"
	publicDashboards    string = "public-dashboards"
)

var reportsToPresetWidgets = []string{
	"66t7EMkmIOcZ4oqsC8fS",
}

func AddPublicDashboard(ctx *gin.Context) []error {
	createPublicDashboardFromID(ctx, customerID, globalDashboardID, dashboardTypeGlobal)
	return nil
}

func createPublicDashboardFromID(ctx *gin.Context, customerID string, ID string, dashboardType string) []error {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}

	dashboardToCreate, _ := fs.Collection("customers").Doc(customerID).Collection("publicDashboards").Doc(ID).Get(ctx)
	globalDashboard := dashboardToCreate.Data()
	globalDashboard["dashboardType"] = dashboardType
	globalDashboard["widgetHeight"] = 1.5

	_, err2 := fs.Collection("dashboards").Doc("customization").Collection(publicDashboards).Doc(ID).Set(ctx, globalDashboard, firestore.MergeAll)
	if err2 != nil {
		log.Printf("An error has occurred: %s", err)
	}

	return nil
}

func UpdateReportsToPresetWidgetsHandler(ctx *gin.Context) []error {
	if err := updateReportsToPresetWidgets(ctx, reportsToPresetWidgets, true); err != nil {
		fmt.Println("updateReportsToPresetWidgets err: ", err.Error())
		return []error{err}
	}

	return nil
}

// run this function manually to update reports to be a public global report for widgets
func updateReportsToPresetWidgets(ctx *gin.Context, reportsID []string, hidden bool) error {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return err
	}
	defer fs.Close()

	for _, reportID := range reportsID {
		if _, err := fs.Collection("dashboards").Doc("google-cloud-reports").Collection("savedReports").Doc(reportID).Update(ctx, []firestore.Update{
			{Path: "hidden", Value: hidden},
			{Path: "public", Value: collab.CollaboratorRoleViewer},
			{Path: "type", Value: "preset"},
			{Path: "customer", Value: nil},
			{Path: "config.currency", Value: firestore.Delete},
			{Path: "config.timezone", Value: firestore.Delete},
			{Path: "collaborators", Value: []map[string]interface{}{
				{"email": "doit-intl.com", "role": "owner"},
			}},
		}); err != nil {
			return err
		}
	}

	return nil
}

func insert(a []dashboard.DashboardWidget, index int, value dashboard.DashboardWidget) []dashboard.DashboardWidget {
	if len(a) == index {
		return append(a, value)
	}

	a = append(a[:index+1], a[index:]...)
	a[index] = value

	return a
}

// Use this fuction manually to add widget report to existing dashboard
func AddWidgetToDashboard(ctx *gin.Context) []error {
	index := 1 // the index of the widget in the dashboard

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}

	defer fs.Close()

	dashboardToCreate, _ := fs.Collection("dashboards").Doc("customization").Collection(publicDashboards).Doc("YnJfHkFLw7lsLINCkISG").Get(ctx)

	var d dashboard.Dashboard

	if err := dashboardToCreate.DataTo(&d); err != nil {
		return []error{err}
	}

	var w dashboard.DashboardWidget
	w.Name = "cloudReports::2Gi0e4pPA3wsfJNOOohW_ss2m7rGY0OjuPDyJub6g"
	w.CardWidth = int(4)

	d.Widgets = insert(d.Widgets, index, w)
	fs.Collection("dashboards").Doc("customization").Collection(publicDashboards).Doc("YnJfHkFLw7lsLINCkISG").Set(ctx, d)

	return nil
}

type DashboardWidget struct {
	WidgetName string `json:"name"`
}

// Remove widget from all dashboards
func RemoveWidget(ctx *gin.Context) []error {
	dryRun := ctx.Query("dryRun") == "true"
	l := logger.FromContext(ctx)

	var params DashboardWidget
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	var errors []error

	var widget = map[string]string{
		"name": params.WidgetName,
	}

	var w dashboard.DashboardWidget
	w.Name = params.WidgetName
	w.Visible = true
	w.Template = false

	iter1 := fs.CollectionGroup("dashboards").Where("widgets", "array-contains", w).Documents(ctx)
	iter2 := fs.CollectionGroup("dashboards").Where("widgets", "array-contains", widget).Documents(ctx)

	counter1, errors := updateIter(ctx, iter1, params, dryRun, l, errors)

	if err != nil {
		return []error{err}
	}

	counter2, errors := updateIter(ctx, iter2, params, dryRun, l, errors)

	if err != nil {
		return []error{err}
	}

	l.Infof("Total dashboards updated: %d", counter1+counter2)

	return errors
}

func updateIter(ctx *gin.Context, iter *firestore.DocumentIterator, params DashboardWidget, dryRun bool, l logger.ILogger, errors []error) (int, []error) {
	counter := 0

	var dashboardsSnaps []*firestore.DocumentSnapshot

	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		}

		dashboardsSnaps = append(dashboardsSnaps, docSnap)
	}

	common.RunConcurrentJobsOnCollection(ctx, dashboardsSnaps, 5, func(ctx context.Context, docSnap *firestore.DocumentSnapshot) {
		// remove the widget item from the widgets list
		dashboard := docSnap.Data()
		currWidgets := dashboard["widgets"].([]interface{})
		withoutActions := make([]interface{}, 0)

		for _, widget := range currWidgets {
			if widget.(map[string]interface{})["name"] != params.WidgetName {
				withoutActions = append(withoutActions, widget)
			}
		}

		l.Infof("dashboard %s\n", docSnap.Ref.Path)
		// logger.Infof("current widgets %v", currWidgets)
		// logger.Infof("widgets without %s %v", params.WidgetName, withoutActions)
		counter++

		if !dryRun {
			l.Infof("Updating")

			_, err := docSnap.Ref.Update(ctx, []firestore.Update{
				{Path: "widgets", Value: withoutActions},
			})
			if err != nil {
				errors = append(errors, err)
			}
		}
	})

	return counter, errors
}
