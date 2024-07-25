package domain

import (
	"time"

	dashboardDomain "github.com/doitintl/hello/scheduled-tasks/dashboard"
	nc "github.com/doitintl/notificationcenter/pkg"
)

type HandleReportSubscriptionRequest struct {
	CustomerID string
	ConfigID   string
}

type NotificationWidgetItem struct {
	ImageURL    string `json:"imageURL"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ReportID    string `json:"-"`
}

type SubscriptionData struct {
	Notification           *nc.Notification
	Dashboard              *dashboardDomain.Dashboard
	ScheduleTime           time.Time
	OrganizationID         string
	CustomerID             string
	ShouldRefreshDashboard bool
	TimeLastAccessed       *time.Time
}
