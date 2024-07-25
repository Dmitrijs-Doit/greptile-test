package slack

import (
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/slack/domain"
)

// GetChartFields - collaborators, public, name
func GetChartFields(chart interface{}) ([]collab.Collaborator, bool, string) {
	var collaborators []collab.Collaborator

	isPublic := false
	name := ""

	switch chartParsed := chart.(type) {
	case *budget.Budget:
		collaborators = chartParsed.Collaborators
		isPublic = chartParsed.Public != nil
		name = chartParsed.Name
	case *report.Report:
		collaborators = chartParsed.Collaborators
		isPublic = chartParsed.Public != nil
		name = chartParsed.Name
	}

	return collaborators, isPublic, name
}

func getCollaboratorRole(chart interface{}, email string) *collab.CollaboratorRole {
	collaborators, _, _ := GetChartFields(chart)
	for _, collaborator := range collaborators {
		if email == collaborator.Email {
			return &collaborator.Role
		}
	}

	return nil
}

func ParseChartCollaborationReq(req *domain.ChartCollaborationReq) *domain.ChartCollaborationPayload {
	eventValue := req.Value
	role := collab.CollaboratorRoleViewer
	public := false
	groupToShareWith := domain.AudienceChannel

	if eventValue == domain.ChannelEditor || eventValue == domain.WorkspaceEditor {
		role = collab.CollaboratorRoleEditor
	}

	if eventValue == domain.WorkspaceViewer || eventValue == domain.WorkspaceEditor {
		public = true
		groupToShareWith = domain.AudienceWorkspace
	}

	cancelAction := eventValue == domain.CancelAction

	urlSplit := strings.Split(req.UnfurlURL, "customers/")
	if len(urlSplit) < 2 {
		return nil
	}

	pathSegments := strings.Split(urlSplit[1], "/")
	customerID := pathSegments[0]
	chartType := pathSegments[2]
	chartID := pathSegments[3]

	return &domain.ChartCollaborationPayload{
		Role:             role,
		Public:           public,
		GroupToShareWith: groupToShareWith,
		CancelAction:     cancelAction,
		CustomerID:       customerID,
		ChartType:        domain.ChartType(chartType),
		ChartID:          chartID,
		WorkspaceID:      req.WorkspaceID,
		ChannelID:        req.ChannelID,
		ResponseURL:      req.ResponseURL,
		SlackUserID:      req.UserID,
	}
}
