package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metadataIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/iface"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
)

const scopeMaxItems = 26

func (s *AnalyticsAlertsService) validateCreateAlertRequest(ctx context.Context, args ExternalAPICreateUpdateArgsReq) (*domain.Alert, []error) {
	alertRequest := args.AlertRequest

	validatedAlert := domain.Alert{
		Name:            alertRequest.Name,
		Customer:        s.customersDAL.GetRef(ctx, args.CustomerID),
		TimeCreated:     time.Now(),
		TimeModified:    time.Now(),
		TimeLastAlerted: nil,
		Recipients:      alertRequest.Recipients,
		IsValid:         true,
	}

	validatedAlert.Config = &domain.Config{
		Currency:     alertRequest.Config.Currency,
		TimeInterval: alertRequest.Config.TimeInterval,
		Aggregator:   report.AggregatorTotal,
		Values:       []float64{alertRequest.Config.Value},
		Rows:         []string{alertRequest.Config.EvaluateForEach},
	}

	errs := []error{}

	if len(alertRequest.Recipients) == 0 {
		validatedAlert.Recipients = []string{args.Email}
	} else {
		err := s.validateRecipients(ctx, args)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if alertRequest.Config.Condition == "" {
		alertRequest.Config.Condition = ConditionPercentage
	}

	validatedAlert.Config.Condition = fromAPICondition(alertRequest.Config.Condition)
	if validatedAlert.Config.Condition == "" {
		errs = append(errs, errormsg.ErrorMsg{Field: "config.condition", Message: ErrInvalidValue})
	}

	if validatedAlert.Config.Currency == "" {
		customer, _ := s.customersDAL.GetCustomer(ctx, args.CustomerID)
		validatedAlert.Config.Currency = fixer.Currency(common.GetCustomerCurrency(customer))
	} else if !fixer.SupportedCurrency(string(validatedAlert.Config.Currency)) {
		errs = append(errs, errormsg.ErrorMsg{Field: "config.currency", Message: ErrNotSupportedCurrency})
	}

	metricConfig, err := s.validateMetric(ctx, args.CustomerID, alertRequest)
	if err != nil {
		errs = append(errs, err)
	}

	if metricConfig != nil {
		validatedAlert.Config.Metric = metricConfig.Metric
		validatedAlert.Config.CalculatedMetric = metricConfig.CalculatedMetric
		validatedAlert.Config.ExtendedMetric = metricConfig.ExtendedMetric
	}

	if !validateOperator(alertRequest.Config.Operator) {
		errs = append(errs, errormsg.ErrorMsg{Field: "config.operator", Message: ErrInvalidValue})
	} else {
		validatedAlert.Config.Operator = alertRequest.Config.Operator.ToMetricFilter()
	}

	if !ValidateTimeInterval(validatedAlert.Config.TimeInterval) {
		errs = append(errs, errormsg.ErrorMsg{Field: "config.timeInterval", Message: ErrInvalidValue})
	}

	if alertRequest.Config.DataSource == "" {
		validatedAlert.Config.DataSource = report.DataSourceBilling
	} else if alertRequest.Config.DataSource == domainExternalReport.ExternalDataSourceBQLens {
		errs = append(errs, errormsg.ErrorMsg{Field: "config.dataSource", Message: ErrInvalidValue})
	} else {
		dataSource, dataSourceValidationErrors := alertRequest.Config.DataSource.ToInternal()
		if len(dataSourceValidationErrors) > 0 {
			errs = append(errs, errormsg.ErrorMsg{Field: "config.dataSource", Message: dataSourceValidationErrors[0].Message})
		} else {
			validatedAlert.Config.DataSource = *dataSource
		}
	}

	// TODO Deprecate creation with attributions
	attributions, err := s.validateAttributions(ctx, args.CustomerID, alertRequest)
	if err != nil {
		errs = append(errs, err)
	}

	if attributions != nil {
		validatedAlert.Config.Scope = attributions
	}

	// Scopes is required
	scopes, err := s.handleCreateScopes(ctx, args)
	if err != nil {
		errs = append(errs, err)
	}

	validatedAlert.Config.Filters = scopes

	if err := s.validateMetadata(ctx, args); err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		validatedAlert.Organization = s.getOrganizationRef(ctx, args.IsDoitEmployee, args.UserID, args.CustomerID)
	}

	return &validatedAlert, errs
}

func (s *AnalyticsAlertsService) validateUpdateAlertRequest(ctx context.Context, args ExternalAPICreateUpdateArgsReq) ([]firestore.Update, []error) {
	var updates []firestore.Update

	var errs []error

	alertRequest := args.AlertRequest

	if alertRequest.Name != "" {
		addUpdate("name", alertRequest.Name, &updates, errs)
	}

	if len(alertRequest.Recipients) > 0 {
		if err := s.validateRecipients(ctx, args); err != nil {
			errs = append(errs, err)
		}

		addUpdate("recipients", alertRequest.Recipients, &updates, errs)
	}

	if alertRequest.Config == nil || reflect.DeepEqual(*alertRequest.Config, AlertConfigAPI{}) {
		return updates, errs
	}

	if alertRequest.Config.Condition != "" {
		path := "config.condition"
		condition := fromAPICondition(alertRequest.Config.Condition)

		if condition == "" {
			errs = append(errs, errormsg.ErrorMsg{Field: path, Message: ErrInvalidValue})
		}

		if condition == domain.ConditionForecast && alertRequest.Config.EvaluateForEach == "" {
			addUpdate("config.rows", []string{}, &updates, errs)
		}

		addUpdate(path, condition, &updates, errs)
	}

	if alertRequest.Config.Currency != "" {
		path := "config.currency"

		if !fixer.SupportedCurrency(string(alertRequest.Config.Currency)) {
			errs = append(errs, errormsg.ErrorMsg{Field: path, Message: ErrNotSupportedCurrency})
		}

		addUpdate(path, alertRequest.Config.Currency, &updates, errs)
	}

	if alertRequest.Config.Metric.Type != "" || alertRequest.Config.Metric.Value != "" {
		metricConfig, err := s.validateMetric(ctx, args.CustomerID, alertRequest)
		if err != nil {
			errs = append(errs, err)
		}

		if metricConfig != nil {
			addUpdate("config.metric", metricConfig.Metric, &updates, errs)
			addUpdate("config.calculatedMetric", metricConfig.CalculatedMetric, &updates, errs)
			addUpdate("config.extendedMetric", metricConfig.ExtendedMetric, &updates, errs)
		}
	}

	operator := alertRequest.Config.Operator.ToMetricFilter()

	if alertRequest.Config.Operator != "" {
		path := "config.operator"

		if !validateOperator(alertRequest.Config.Operator) {
			errs = append(errs, errormsg.ErrorMsg{Field: path, Message: ErrInvalidValue})
		}

		addUpdate(path, operator, &updates, errs)
	}

	if alertRequest.Config.TimeInterval != "" {
		path := "config.timeInterval"

		if !ValidateTimeInterval(alertRequest.Config.TimeInterval) {
			errs = append(errs, errormsg.ErrorMsg{Field: path, Message: ErrInvalidValue})
		}

		addUpdate(path, alertRequest.Config.TimeInterval, &updates, errs)
	}

	if alertRequest.Config.Value != 0 {
		addUpdate("config.values", []float64{alertRequest.Config.Value}, &updates, errs)
	}

	if alertRequest.Config.Attributions != nil && len(alertRequest.Config.Attributions) > 0 {
		attributions, err := s.validateAttributions(ctx, args.CustomerID, alertRequest)
		if err != nil {
			errs = append(errs, err)
		}

		addUpdate("config.scope", attributions, &updates, errs)
	}

	if len(alertRequest.Config.Scopes) > 0 {
		scopes, err := s.handleCreateScopes(ctx, args)
		if err != nil {
			errs = append(errs, err)
		}

		addUpdate("config.filters", scopes, &updates, errs)
	}

	if alertRequest.Config.EvaluateForEach != "" {
		if err := s.validateMetadata(ctx, args); err != nil {
			errs = append(errs, err)
		}

		addUpdate("config.rows", []string{alertRequest.Config.EvaluateForEach}, &updates, errs)
	}

	if alertRequest.Config.DataSource != "" {
		if alertRequest.Config.DataSource == domainExternalReport.ExternalDataSourceBQLens || !alertRequest.Config.DataSource.ValidateDataSource() {
			errs = append(errs, errormsg.ErrorMsg{Field: "config.dataSource", Message: ErrInvalidValue})
		}

		addUpdate("config.dataSource", alertRequest.Config.DataSource, &updates, errs)
	}

	return updates, errs
}

func addUpdate(path string, value interface{}, updates *[]firestore.Update, errs []error) {
	if len(errs) == 0 {
		*updates = append(*updates, firestore.Update{
			Path:  path,
			Value: value,
		})
	}
}

func (s *AnalyticsAlertsService) validateAttributions(ctx context.Context, customerID string, alertRequest *AlertRequest) ([]*firestore.DocumentRef, error) {
	valueFieldName := "config.attributions"

	var attributionErrors []string

	var attributions []*firestore.DocumentRef

	for _, attributionID := range alertRequest.Config.Attributions {
		attribution, err := s.attributionsDAL.GetAttribution(ctx, attributionID)
		if err != nil {
			attributionErrors = append(attributionErrors, attributionID+" not found")
			continue
		}

		if attribution.Type == string(domainAttributions.ObjectTypeManaged) {
			attributionErrors = append(attributionErrors, attributionID+" invalid: managed attributions cannot be used")
			continue
		}

		if attribution.Type == "custom" && attribution.Customer.ID != customerID {
			attributionErrors = append(attributionErrors, attributionID+" not permitted for the user")
			continue
		}

		attributions = append(attributions, s.attributionsDAL.GetRef(ctx, attributionID))
	}

	if len(attributionErrors) > 0 {
		return nil, errormsg.ErrorMsg{Field: valueFieldName, Message: strings.Join(attributionErrors[:], ", ")}
	}

	return attributions, nil
}

func (s *AnalyticsAlertsService) validateMetric(ctx context.Context, customerID string, alertRequest *AlertRequest) (*domain.Config, error) {
	valueFieldName := "config.metric.value"
	typeFieldName := "config.metric.type"

	var config = domain.Config{}

	switch alertRequest.Config.Metric.Type {
	case BasicMetric:
		config.Metric = report.MetricTextToEnum(string(alertRequest.Config.Metric.Value))
		if config.Metric == 4 {
			return nil, errormsg.ErrorMsg{Field: valueFieldName, Message: ErrInvalidValue}
		}

	case CustomMetric:
		config.Metric = report.MetricCustom

		customMetric, err := s.metrics.GetCustomMetric(ctx, alertRequest.Config.Metric.Value)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return nil, errormsg.ErrorMsg{Field: valueFieldName, Message: ErrNotFound}
			}

			return nil, errormsg.ErrorMsg{Field: valueFieldName, Message: ErrUnknown}
		}

		if customMetric.Customer != nil && customMetric.Customer.ID != customerID {
			return nil, errormsg.ErrorMsg{Field: valueFieldName, Message: ErrForbiddenID}
		}

		config.CalculatedMetric = s.metrics.GetRef(ctx, alertRequest.Config.Metric.Value)

	case ExtendedMetric:
		config.Metric = report.MetricExtended
		config.ExtendedMetric = &alertRequest.Config.Metric.Value

	default:
		return nil, errormsg.ErrorMsg{Field: typeFieldName, Message: ErrInvalidValue}
	}

	return &config, nil
}

func (s *AnalyticsAlertsService) validateRecipients(ctx context.Context, args ExternalAPICreateUpdateArgsReq) error {
	customer, _ := s.customersDAL.GetCustomer(ctx, args.CustomerID)

	errMessage := validateRecipientsAgainstDomains(args.AlertRequest.Recipients, customer.Domains, args.IsDoitEmployee)
	if errMessage != "" {
		return errormsg.ErrorMsg{Field: "recipients", Message: errMessage}
	}

	return nil
}

func (s *AnalyticsAlertsService) validateMetadata(ctx context.Context, args ExternalAPICreateUpdateArgsReq) error {
	fieldName := "config.evaluateForEach"
	config := args.AlertRequest.Config
	metadata := config.EvaluateForEach

	if metadata == "" {
		return nil
	}

	if config.Condition == ConditionForecast {
		return errormsg.ErrorMsg{Field: fieldName, Message: ErrForecastMetadataIncompatible}
	}

	filters := strings.Split(metadata, ":")
	if len(filters) != 2 {
		return errormsg.ErrorMsg{Field: fieldName, Message: ErrInvalidValue}
	}

	if _, err := s.metadataService.ExternalAPIGet(
		metadataIface.ExternalAPIGetArgs{
			Ctx:            ctx,
			IsDoitEmployee: args.IsDoitEmployee,
			UserID:         args.UserID,
			CustomerID:     args.CustomerID,
			KeyFilter:      filters[1],
			TypeFilter:     filters[0],
		},
	); err != nil {
		return errormsg.ErrorMsg{Field: fieldName, Message: ErrNotFound}
	}

	return nil
}

func (s *AnalyticsAlertsService) getOrganizationRef(ctx context.Context, isDoitEmployee bool, userID string, customerID string) *firestore.DocumentRef {
	if !isDoitEmployee && userID != "" {
		user, err := s.userDal.Get(ctx, userID)
		if err != nil {
			return nil
		}

		if len(user.Organizations) > 0 {
			return user.Organizations[0]
		}
	}

	return s.alertsDal.GetCustomerOrgRef(ctx, customerID, rootOrgID)
}

func (s *AnalyticsAlertsService) validateScopes(ctx context.Context, args ExternalAPICreateUpdateArgsReq) error {
	scopes := args.AlertRequest.Config.Scopes
	externalPath := "config.scopes"

	err := s.validateScopeFields(scopes)
	if err == nil { // further validate if initial check passes
		err = s.validateScopeFilters(ctx, scopes, args.UserID, args.CustomerID, args.Email, args.IsDoitEmployee)
		if err != nil {
			return errormsg.ErrorMsg{Field: externalPath, Message: ErrInvalidValue}
		}
	} else {
		return errormsg.ErrorMsg{Field: externalPath, Message: ErrInvalidValue}
	}

	return nil
}

func (s *AnalyticsAlertsService) validateScopeFields(scopes []Scope) error {
	for i, scope := range scopes {
		if scope.Type == "" {
			return fmt.Errorf("type field is missing in scope %d", i+1)
		}

		if scope.Key == "" {
			return fmt.Errorf("key field is missing in scope %d", i+1)
		}

		hasValues := scope.Values != nil && len(*scope.Values) > 0
		hasRegex := scope.Regexp != nil

		// same as !XOR
		if hasValues == hasRegex {
			return fmt.Errorf("scope %d must have either regex or values but not both", i+1)
		}

		if scope.Regexp != nil {
			_, err := regexp.Compile(*scope.Regexp)
			if err != nil {
				return fmt.Errorf("scope %d has invalid regexp", i+1)
			}
		}
	}

	return nil
}

func (s *AnalyticsAlertsService) validateScopeFilters(ctx context.Context, scopes []Scope, userID, customerID, userEmail string, isDoitEmployee bool) error {
	if len(scopes) > scopeMaxItems {
		return errors.New("too many scopes provided")
	}

	dimensions, err := s.metadataService.ExternalAPIList(
		metadataIface.ExternalAPIListArgs{
			Ctx:            ctx,
			IsDoitEmployee: isDoitEmployee,
			UserID:         userID,
			CustomerID:     customerID,
			UserEmail:      userEmail,
		},
	)
	if err != nil {
		return err
	}

	for i, scope := range scopes {
		filterExists := false

		for _, dimension := range dimensions {
			if scope.Key == dimension.ID && scope.Type == dimension.Type {
				filterExists = true
				break
			}
		}

		if !filterExists {
			return fmt.Errorf("scope %d is not valid", i+1)
		}
	}

	return nil
}

func (s *AnalyticsAlertsService) handleCreateScopes(ctx context.Context, args ExternalAPICreateUpdateArgsReq) ([]*report.ConfigFilter, error) {
	scopes := args.AlertRequest.Config.Scopes

	if len(scopes) == 0 {
		return nil, errormsg.ErrorMsg{Field: "config.scopes", Message: ErrNotFound}
	}

	if err := s.validateScopes(ctx, args); err != nil {
		return nil, err
	}

	filters, err := toConfigFilters(scopes)
	if err != nil {
		return nil, errormsg.ErrorMsg{Field: "config.scopes", Message: ErrInvalidScopeMetadataType}
	}

	// TODO remove this when more than one filter is supported
	if len(filters) > 0 {
		return []*report.ConfigFilter{{BaseConfigFilter: filters[0]}}, nil
	}

	return nil, nil
}

func toConfigFilters(scopes []Scope) ([]report.BaseConfigFilter, error) {
	filters := make([]report.BaseConfigFilter, 0, len(scopes))

	for _, s := range scopes {
		id := metadata.ToInternalID(s.Type, s.Key)

		md, key, err := cloudanalytics.ParseID(id)
		if err != nil {
			return nil, err
		}

		f := report.BaseConfigFilter{
			Key:       key,
			Type:      md.Type,
			Values:    s.Values,
			ID:        id,
			Field:     md.Field,
			Inverse:   s.Inverse,
			Regexp:    s.Regexp,
			AllowNull: s.AllowNull,
		}

		filters = append(filters, f)
	}

	return filters, nil
}
