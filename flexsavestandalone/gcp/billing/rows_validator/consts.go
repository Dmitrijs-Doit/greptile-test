package rows_validator

import (
	"time"
)

const logPrefix string = "GCP_FS_SA - BILLING_ROW_VALIDATOR: "

type tableType string

const (
	customerTableType tableType = "customer"
	localTableType    tableType = "local"
	unifiedTableType  tableType = "unified"

	dayPeriod   time.Duration = 24 * time.Hour
	monthPeriod time.Duration = 31 * dayPeriod

	defaultTo           = "danielle@doit-intl.com"
	defaultSlackChannel = "#urgent-flexsave-billing-issues"
)

var (
	defaultCC = []string{"lionel@doit-intl.com", "miguel@doit-intl.com"}
)
