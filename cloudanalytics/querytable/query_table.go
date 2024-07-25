package querytable

import (
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func getFullTableName(tableName string) string {
	projectID := googleCloudConsts.CustomBillingDev
	if common.Production {
		projectID = googleCloudConsts.CustomBillingProd
	}

	return fmt.Sprintf(consts.FullTableTemplate, projectID, googleCloudConsts.CustomBillingDataset, tableName)
}

// GetAWSLineItemsTable returns the (full) table name for AWS line items (eg. where cost_type is FlexSave and the cloud is AWS)
func GetAWSLineItemsTable() string {
	var projectName string
	if common.Production {
		projectName = googleCloudConsts.CustomBillingProd
	} else {
		projectName = googleCloudConsts.CustomBillingDev
	}

	return fmt.Sprintf("%s.aws_custom_billing.aws_custom_billing_export_recent", projectName)
}

func GetCustomerFeaturesTable() string {
	projectID := CustomerFeaturesDev
	if common.Production {
		projectID = CustomerFeaturesProd
	}

	return fmt.Sprintf(consts.FullTableTemplate, projectID, CustomerFeaturesDataset, CustomerFeaturesTable)
}

func GetCustomerFeaturesIdentificationTable() string {
	projectID := CustomerFeaturesDev
	if common.Production {
		projectID = CustomerFeaturesProd
	}

	return fmt.Sprintf(consts.FullTableTemplate, projectID, CustomerFeaturesDataset, CustomerFeaturesIdTable)
}
