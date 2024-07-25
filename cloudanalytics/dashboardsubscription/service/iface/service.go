package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/dashboardsubscription/domain"
)

type IService interface {
	HandleDashboardSubscription(ctx context.Context, request domain.HandleReportSubscriptionRequest) error
}
