package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	domainGoogleCloud "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/domain/cost_allocation"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func (c *CostAllocationService) ScheduleInitStandaloneAccounts(ctx context.Context, billingAccountIDs []string) error {
	var err error

	for _, billingAccountID := range billingAccountIDs {
		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf("/tasks/analytics/google-cloud/costAllocation/init-standalone-account/%s", billingAccountID),
			Queue:  common.TaskQueueCloudAnalyticsOnDemandTasks,
		}

		if _, err = c.conn.CloudTaskClient.CreateTask(ctx, config.Config(nil)); err != nil {
			c.loggerProvider(ctx).Errorf("failed to create gke cost allocation init task for billing account: %s, error: %v", billingAccountID, err)
		}
	}

	return err
}

func (c *CostAllocationService) InitStandaloneAccount(ctx context.Context, billingAccountID string) error {
	now := time.Now().UTC()

	l := c.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		serviceField: serviceName,
	})

	query := c.getInitStandaloneAccountQuery(billingAccountID)

	results, err := c.getResultsFromBQ(ctx, query)
	if err != nil {
		return err
	}

	if bqDataShowsCAEnabled(&results) {
		asset, err := c.assetsDal.Get(ctx, fmt.Sprintf("%s-%s", common.Assets.GoogleCloudStandalone, billingAccountID))
		if err != nil {
			return err
		}

		newValue := c.getNewCostAllocation(ctx, asset.Customer.ID, &results, now)
		if err := c.dal.UpdateCostAllocation(ctx, asset.Customer.ID, newValue); err != nil {
			return err
		}
	}

	return nil
}

func (c *CostAllocationService) getInitStandaloneAccountQuery(billingAccountID string) string {
	template := domain.CustomersGKECostAllocationQueryTmpl

	interval := domain.Interval{
		To:   `CURRENT_DATE()`,
		From: domain.GKECostAllocationFeatureStartedAt,
	}

	table := fmt.Sprintf("(SELECT * FROM `%s.%s.gcp_billing_export_resource_v1_%s`)",
		domainGoogleCloud.GetStandaloneProject(),
		domainGoogleCloud.BillingStandaloneDataset,
		strings.Replace(billingAccountID, "-", "_", -1))

	return strings.NewReplacer("{table}", table, "{export_time_from}", interval.From, "{export_time_to}", interval.To).Replace(template)
}
