package service

import (
	"errors"
	"regexp"
	"strings"

	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

func validateRecipientsAgainstDomains(recipients []string, domains []string, isDoitEmployee bool) string {
	doitDomains := []string{"doit.com", "doit-intl.com"}

	if isDoitEmployee {
		domains = append(domains, doitDomains...)
	}

	for _, email := range recipients {
		if !isEmailValid(email) {
			return ErrNotValidEmail
		}

		components := strings.Split(email, "@")
		emailDomain := components[1]

		if !slice.Contains(domains, emailDomain) && !isSlackOrMSTeamsEmail(email) {
			return ErrForbiddenEmail
		}
	}

	return ""
}

func isEmailValid(e string) bool {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return emailRegex.MatchString(e)
}

func isSlackOrMSTeamsEmail(e string) bool {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.slack.com|teams.ms$`)
	return emailRegex.MatchString(e)
}

func toAlertAPI(alert *domain.Alert) (*AlertAPI, error) {
	attributions := []string{}

	var lastAlerted *int64

	if alert.TimeLastAlerted != nil {
		time := alert.TimeLastAlerted.UnixMilli()
		lastAlerted = &time
	}

	alertAPI := &AlertAPI{
		ID:          alert.ID,
		Name:        alert.Name,
		CreateTime:  alert.TimeCreated.UnixMilli(),
		UpdateTime:  alert.TimeModified.UnixMilli(),
		LastAlerted: lastAlerted,
		Recipients:  alert.Recipients,
	}

	if alert.Config == nil {
		return alertAPI, nil
	}

	for _, ref := range alert.Config.Scope {
		attributions = append(attributions, ref.ID)
	}

	var metricConfig MetricConfig

	if alert.Config.CalculatedMetric != nil {
		metricConfig = MetricConfig{
			Type:  CustomMetric,
			Value: alert.Config.CalculatedMetric.ID,
		}
	} else if alert.Config.ExtendedMetric != nil {
		metricConfig = MetricConfig{
			Type:  ExtendedMetric,
			Value: *alert.Config.ExtendedMetric,
		}
	} else {
		value, err := domainQuery.GetMetricString(alert.Config.Metric)
		if err != nil {
			return nil, errors.New("no metric found")
		}

		metricConfig = MetricConfig{
			Type:  BasicMetric,
			Value: value,
		}
	}

	var evaluateForEach string
	if alert.Config.Rows != nil {
		evaluateForEach = alert.Config.Rows[0]
	}

	scopes := make([]Scope, 0, len(alert.Config.Filters))
	for _, filter := range alert.Config.Filters {
		scopes = append(scopes, Scope{
			Key:       filter.Key,
			Type:      filter.Type,
			Values:    filter.Values,
			Inverse:   filter.Inverse,
			Regexp:    filter.Regexp,
			AllowNull: filter.AllowNull,
		})
	}

	if alert.Config.DataSource == "" {
		alert.Config.DataSource = report.DataSourceBilling
	}

	dataSource, dataSourceValidationErrors := domainExternalReport.NewExternalDatasourceFromInternal(alert.Config.DataSource)
	if len(dataSourceValidationErrors) > 0 {
		return nil, dataSourceValidationErrors[0]
	}

	alertAPI.Config = &AlertConfigAPI{
		Attributions:    attributions,
		Metric:          metricConfig,
		Currency:        alert.Config.Currency,
		TimeInterval:    alert.Config.TimeInterval,
		Condition:       toAPICondition(alert.Config.Condition),
		Operator:        alert.Config.Operator.ToMetricFilterText(),
		Value:           alert.Config.Values[0],
		EvaluateForEach: evaluateForEach,
		Scopes:          scopes,
		DataSource:      *dataSource,
	}

	return alertAPI, nil
}

func toListAlertAPI(alerts []domain.Alert) ([]customerapi.SortableItem, error) {
	apiAlerts := make([]customerapi.SortableItem, len(alerts))

	for i, alert := range alerts {
		var owner string

		alertAPI, err := toAlertAPI(&alert)
		if err != nil {
			return nil, err
		}

		for _, collaborator := range alert.Collaborators {
			if collaborator.Role == collab.CollaboratorRoleOwner {
				owner = collaborator.Email
			}
		}

		apiAlerts[i] = ListAlertAPI{
			ID:          alertAPI.ID,
			Name:        alertAPI.Name,
			Owner:       owner,
			LastAlerted: alertAPI.LastAlerted,
			CreateTime:  alertAPI.CreateTime,
			UpdateTime:  alertAPI.UpdateTime,
			Config:      alertAPI.Config,
		}
	}

	return apiAlerts, nil
}
