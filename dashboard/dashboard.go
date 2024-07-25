package dashboard

import (
	"errors"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	"github.com/gin-gonic/gin"
)

type Dashboard struct {
	AllowToEdit         bool              `firestore:"allowToEdit"`
	CustomerID          string            `firestore:"customerId"`
	Email               string            `firestore:"email"`
	IsPublic            bool              `firestore:"isPublic"`
	Name                string            `firestore:"name"`
	PublicDashboardID   string            `firestore:"publicDashboardId"`
	DashboardType       string            `firestore:"dashboardType,omitempty"`
	DashboardIcon       interface{}       `firestore:"icon,omitempty"`
	SortNumber          int               `firestore:"sortNumber"`
	WidgetHeight        *float64          `firestore:"widgetHeight"`
	Widgets             []DashboardWidget `firestore:"widgets"`
	OwnerID             string            `firestore:"ownerId"`
	RequiredPermissions []string          `firestore:"requiredPermissions"`
	HasCloudReports     bool              `firestore:"hasCloudReports"`

	ID      string                 `firestore:"-"`
	Ref     *firestore.DocumentRef `firestore:"-"`
	DocPath string                 `firestore:"-"`
}

type WidgetRefreshState string

const (
	WidgetRefreshStateProcessing WidgetRefreshState = "processing"
	WidgetRefreshStateSuccess    WidgetRefreshState = "success"
	WidgetRefreshStateFailed     WidgetRefreshState = "failed"
	widgetPrefix                                    = "cloudReports::"
	widgetPrefixLen                                 = len(widgetPrefix)
)

var (
	ErrMissingCustomerID = errors.New("missing customer id")
	ErrMissingReportID   = errors.New("missing report id")
)

type DashboardWidget struct {
	Visible   bool               `firestore:"visible"`
	Template  bool               `firestore:"template"`
	Name      string             `firestore:"name"`
	CardWidth interface{}        `firestore:"cardWidth"`
	State     WidgetRefreshState `firestore:"state"`
}

func (dw *DashboardWidget) ExtractInfoFromName() (string, string, string, error) {
	if !strings.HasPrefix(dw.Name, widgetPrefix) {
		return "", "", "", ErrMissingCustomerID
	}

	widgetID := dw.Name[widgetPrefixLen:]

	parts := strings.Split(widgetID, "_")
	if len(parts) < 2 {
		return "", "", "", ErrMissingReportID
	}

	return widgetID, parts[0], parts[1], nil
}

type TicketStatistics struct {
	ResolvedTicketsLastMonth []*TicketSummary `firestore:"resolvedTicketsLastMonth"`
}

type TicketSummary struct {
	ID      int32  `firestore:"id" json:"id"`
	Subject string `firestore:"subject" json:"subject"`
	Score   string `firestore:"score" json:"score"`
}

type DashboardDetails struct {
	DashboardID   string
	DashboardType string
}

func GetCustomerDashboards(ctx *gin.Context) {
	fs := common.GetFirestoreClient(ctx)

	docs, err := fs.CollectionGroup("dashboards").Where("customerId", "==", ctx.Param("customerID")).Documents(ctx).GetAll()
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return
	}

	dashboards := make([]map[string]interface{}, 0)

	for _, dashboardDoc := range docs {
		dashboard := dashboardDoc.Data()
		dashboard["id"] = dashboardDoc.Ref.ID
		dashboards = append(dashboards, dashboard)
	}

	ctx.JSON(http.StatusOK, dashboards)
}
