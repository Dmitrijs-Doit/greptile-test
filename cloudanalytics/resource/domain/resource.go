package domain

import (
	alertDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	attributiongroupsDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	budgetDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	metricDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type ResourceType string

const (
	Organizations     ResourceType = "organizations"
	Metrics           ResourceType = "metrics"
	Alerts            ResourceType = "alerts"
	Budgets           ResourceType = "budgets"
	AttributionGroups ResourceType = "attributionGroups"
	Reports           ResourceType = "reports"
	DailyDigests      ResourceType = "dailyDigests"
	WeeklyDigests     ResourceType = "weeklyDigests"
)

type Resource struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Owner string `json:"owner,omitempty"`
}

func NewResourcesFromAttributionGroups(
	requesterEmail string,
	attrGroups []*attributiongroupsDomain.AttributionGroup,
) []Resource {
	var resources []Resource

	for _, attrGroup := range attrGroups {
		resources = append(resources, Resource{
			ID:    attrGroup.ID,
			Name:  attrGroup.Name,
			Owner: getOwnerIfNoAccess(requesterEmail, attrGroup.Access),
		})
	}

	return resources
}

func NewResourcesFromBudgets(
	requesterEmail string,
	budgets []*budgetDomain.Budget,
) []Resource {
	var resources []Resource

	for _, budget := range budgets {
		resources = append(resources, Resource{
			ID:    budget.ID,
			Name:  budget.Name,
			Owner: getOwnerIfNoAccess(requesterEmail, budget.Access),
		})
	}

	return resources
}

func NewResourcesFromAlerts(requesterEmail string, alerts []*alertDomain.Alert) []Resource {
	var resources []Resource

	for _, alert := range alerts {
		resources = append(resources, Resource{
			ID:    alert.ID,
			Name:  alert.Name,
			Owner: getOwnerIfNoAccess(requesterEmail, alert.Access),
		})
	}

	return resources
}

func NewResourcesFromMetrics(metrics []*metricDomain.CalculatedMetric) []Resource {
	var resources []Resource

	for _, metric := range metrics {
		resources = append(resources, Resource{
			ID:   metric.ID,
			Name: metric.Name,
		})
	}

	return resources
}

func NewResourcesFromOrgs(orgs []*common.Organization) []Resource {
	var resources []Resource

	for _, org := range orgs {
		resources = append(resources, Resource{
			ID:   org.ID,
			Name: org.Name,
		})
	}

	return resources
}

func getOwnerIfNoAccess(requesterEmail string, access collab.Access) string {
	var owner string

	if !access.CanView(requesterEmail) {
		owner = access.GetOwner()
	}

	return owner
}
