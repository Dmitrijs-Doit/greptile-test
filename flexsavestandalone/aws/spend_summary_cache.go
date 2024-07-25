package aws

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	doitfs "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
)

const daysToForecastFrom = 31

type configCreationAttributes struct {
	customerID   string
	spendSummary *FlexsaveStandaloneData
	timeEnabled  *time.Time
}

// FanoutFSCacheDataForCustomers fanout cache data for all standalone customers
func (s *AwsStandaloneService) FanoutFSCacheDataForCustomers(ctx context.Context, monthNumber string) error {
	var taskError error

	logger := s.loggerProvider(ctx)

	customerIDs, err := s.payers.GetAWSStandaloneCustomerIDs(ctx)
	if err != nil {
		return err
	}

	for _, customerID := range customerIDs {
		baseURI := fmt.Sprintf("/tasks/flex-ri/standalone/cache/customer/%s?numberOfMonths=%s", customerID, monthNumber)

		var task = common.CloudTaskConfig{
			Method:       cloudtaskspb.HttpMethod_POST,
			Path:         baseURI,
			Queue:        common.TaskQueueFlexSaveStandaloneSpendSummary,
			ScheduleTime: nil,
		}

		if _, err := common.CreateCloudTask(ctx, &task); err != nil {
			logger.Error(err)
			taskError = err
		}
	}

	return taskError
}

func (s *AwsStandaloneService) UpdateStandaloneCustomerSpendSummary(ctx context.Context, customerID string, numberOfMonths int) error {
	logger := s.loggerProvider(ctx)

	var standaloneConfigs []*types.PayerConfig

	payerConfigs, err := s.payers.GetPayerConfigsForCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	for _, config := range payerConfigs {
		if config.Type == string(pkg.StandalonePayerConfigTypeAWS) && config.Status != "disabled" {
			standaloneConfigs = append(standaloneConfigs, config)
		}
	}

	var newlyOnboarded bool

	twoDaysAgo := s.now().UTC().AddDate(0, 0, -2).Truncate(time.Hour * 24)

	var payerAccountIDS []string

	var timeEnabled *time.Time

	for _, config := range standaloneConfigs {
		payerAccountIDS = append(payerAccountIDS, config.AccountID)
		newlyOnboarded = config.TimeEnabled == nil || config.TimeEnabled.After(twoDaysAgo)

		if config.TimeEnabled != nil && (timeEnabled == nil || config.TimeEnabled.After(*timeEnabled)) {
			timeEnabled = config.TimeEnabled
		}
	}

	beginningOfStartMonth := time.Date(s.now().Year(), s.now().Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -numberOfMonths, 0)
	if timeEnabled != nil && timeEnabled.After(beginningOfStartMonth) && numberOfMonths > 2 {
		beginningOfEnablementMonth := time.Date(timeEnabled.Year(), timeEnabled.Month(), 1, 0, 0, 0, 0, time.UTC)
		beginningOfThisMonth := time.Date(s.now().Year(), s.now().Month(), 1, 0, 0, 0, 0, time.UTC)
		monthToCheck := beginningOfEnablementMonth

		for i := 1; i <= numberOfMonths; i++ {
			if monthToCheck == beginningOfThisMonth {
				numberOfMonths = i + 1
				break
			}

			monthToCheck = monthToCheck.AddDate(0, 1, 0)
		}
	}

	var timeParams timeParams

	timeParams.Now = s.now()
	timeParams.ApplicableMonths = utils.GetApplicableMonths(timeParams.Now, numberOfMonths)
	timeParams.CurrentMonth = timeParams.ApplicableMonths[0]
	timeParams.PreviousMonth = timeParams.ApplicableMonths[1]
	fromTime := timeParams.Now.AddDate(0, 0, -daysToForecastFrom)

	queryParams := BigQueryRequestParams{
		CustomerID:       customerID,
		AccountIDs:       payerAccountIDS,
		NumberOfMonth:    numberOfMonths,
		ForecastFromTime: fromTime,
	}

	spend, err := s.GetCustomerSpend(ctx, queryParams, timeParams)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoTable) && newlyOnboarded:
			break
		default:
			logger.Errorf("error: %v getting spend summary for customer: %v", err, customerID)
		}
	}

	adjustedSpend, forecastResult, err := s.validateSpendResults(&spend, payerAccountIDS, timeParams.CurrentMonth)
	if err != nil {
		return err
	}

	spendSummary, err := s.buildFlexsaveSpendSummary(ctx, adjustedSpend, forecastResult, timeParams)
	if err != nil {
		return err
	}

	configValues := configCreationAttributes{
		customerID:   customerID,
		spendSummary: spendSummary,
		timeEnabled:  timeEnabled,
	}

	fsc, err := s.integrationsDAL.GetFlexsaveConfigurationCustomer(ctx, customerID)
	if err != nil && err != doitfs.ErrNotFound {
		return err
	}

	if err != nil || fsc.AWS.SavingsHistory == nil {
		return s.createFlexsaveConfiguration(ctx, configValues, timeParams)
	}

	for month, data := range fsc.AWS.SavingsHistory {
		if spendSummary.Spend[month] == nil {
			spendSummary.Spend[month] = data
		}
	}

	// update AWS Flexsave configuration for the last two months
	fsc.AWS.SavingsHistory = spendSummary.Spend

	fsc.AWS.SavingsSummary = spendSummary.SavingsSummary

	// backfill for old customers that might not have `TimeEnabled` set yet
	if fsc.AWS.Enabled && fsc.AWS.TimeEnabled == nil {
		fsc.AWS.TimeEnabled = timeEnabled
	}

	return s.integrationsDAL.UpdateFlexsaveConfigurationCustomer(
		ctx,
		customerID,
		map[string]*pkg.FlexsaveSavings{
			"AWS": &fsc.AWS,
		})
}

func (s *AwsStandaloneService) createFlexsaveConfiguration(ctx context.Context, configAttributes configCreationAttributes, timeParams timeParams) error {
	aws := &pkg.FlexsaveSavings{
		Enabled:        true,
		SavingsHistory: configAttributes.spendSummary.Spend,
		Timestamp:      &timeParams.Now,
		SavingsSummary: configAttributes.spendSummary.SavingsSummary,
		TimeEnabled:    configAttributes.timeEnabled,
	}

	if err := s.integrationsDAL.UpdateFlexsaveConfigurationCustomer(
		ctx,
		configAttributes.customerID,
		map[string]*pkg.FlexsaveSavings{
			"AWS": aws,
		}); err != nil {
		return err
	}

	return nil
}
