package utils

import (
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/dataStructures"
)

func GetDummyTableName(version int64, iteration int) string {
	return fmt.Sprintf(consts.DummyBQTableNameTemplate, GetProjectNameUnderscore(), version, iteration)
}

func GetDummyBillingAccount(version int64, iteration int) string {
	return fmt.Sprintf(consts.DummyBillingAccountTemplate, version, iteration)
}

func GetDummyCustomerID(billingAccount string) string {
	return fmt.Sprintf(consts.DummyCustomerIDTemplate, billingAccount)
}

func GetCopyToDummyTablejobPrefix(atm *dataStructures.AutomationTaskMetadata) string {
	return fmt.Sprintf(consts.CopyToDummyJobPrefixTemplate, atm.BillingAccountID, atm.Version, atm.Iteration)
}

func GetProjectNameUnderscore() string {
	return strings.ReplaceAll(common.ProjectID, "-", "_")
}
