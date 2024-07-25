package googlecloud

import (
	"context"
	"fmt"
	"strings"
	"time"

	billingbudgets "google.golang.org/api/billingbudgets/v1beta1"
	cloudbilling "google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/option"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

func CreateBudget(ctx context.Context, projectBillingInfo *cloudbilling.ProjectBillingInfo, policy *SandboxPolicy) (*billingbudgets.GoogleCloudBillingBudgetsV1beta1Budget, error) {
	secretData, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretAppEngine)
	if err != nil {
		return nil, err
	}

	creds := option.WithCredentialsJSON(secretData)
	quotaProject := option.WithQuotaProject(common.ProjectID)

	bb, err := billingbudgets.NewService(ctx, creds, quotaProject)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	projectResourceName := strings.TrimSuffix(projectBillingInfo.Name, "/billingInfo")
	createBudgetReq := &billingbudgets.GoogleCloudBillingBudgetsV1beta1CreateBudgetRequest{
		Budget: &billingbudgets.GoogleCloudBillingBudgetsV1beta1Budget{
			DisplayName: fmt.Sprintf("%s_%s", now.Format(time.RFC3339), projectBillingInfo.ProjectId),
			AllUpdatesRule: &billingbudgets.GoogleCloudBillingBudgetsV1beta1AllUpdatesRule{
				PubsubTopic:   "projects/me-doit-intl-com/topics/sandbox-budgets",
				SchemaVersion: "1.0",
			},
			Amount: &billingbudgets.GoogleCloudBillingBudgetsV1beta1BudgetAmount{
				SpecifiedAmount: &billingbudgets.GoogleTypeMoney{
					Units: policy.Amount,
					Nanos: 0,
				},
			},
			BudgetFilter: &billingbudgets.GoogleCloudBillingBudgetsV1beta1Filter{
				CreditTypesTreatment: "EXCLUDE_ALL_CREDITS",
				Projects:             []string{projectResourceName},
			},
			ThresholdRules: []*billingbudgets.GoogleCloudBillingBudgetsV1beta1ThresholdRule{
				// {SpendBasis: "CURRENT_SPEND", ThresholdPercent: 1.0},
			},
		},
	}

	budget, err := bb.BillingAccounts.Budgets.Create(projectBillingInfo.BillingAccountName, createBudgetReq).Do()
	if err != nil {
		return nil, err
	}

	return budget, nil
}
