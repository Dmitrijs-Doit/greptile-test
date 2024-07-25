package utils

import (
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/consts"
)

func GetCopyToDummyQuery(billingAccount string, rows int64) string {
	return fmt.Sprintf("SELECT \"%s\" as billing_account_id, service, sku, "+
		"usage_start_time, usage_end_time, project, labels, system_labels, "+
		"location, CURRENT_TIMESTAMP() as export_time, cost, currency, currency_conversion_rate, "+
		"usage, credits, invoice, cost_type, adjustment_info "+
		"FROM `%s` LIMIT %d", billingAccount, CreateDummyTableFullName(), rows)
}

//func GetCopyToDummyQuery(billingAccount string, rows int64) string {
//	return fmt.Sprintf("SELECT \"%s\" as billing_account_id, service, sku, "+
//		"usage_start_time, usage_end_time, project, labels, system_labels, "+
//		"location, \"%s\" as export_time, cost, currency, currency_conversion_rate, "+
//		"usage, credits, invoice, cost_type, adjustment_info "+
//		"FROM `%s` LIMIT %d", billingAccount, time.Now().Truncate(time.Second).Format(billingConsts.ExportTimeLayout), CreateDummyTableFullName(), rows)
//
//}

func CreateDummyTableFullName() string {
	return fmt.Sprintf("%s.%s.%s", consts.DummyBQProjectName, consts.DummyBQDatasetName, consts.DummyBQTableNameOriginal)
}
