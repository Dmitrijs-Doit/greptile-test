package cspreport

import (
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	FieldBillingReportCSPTableCompatability string = "ARRAY(SELECT AS STRUCT cost, usage, savings, CAST(NULL AS FLOAT64) AS margin, credit, CAST(NULL AS STRUCT<key STRING, value FLOAT64, type STRING>) AS ext_metric FROM UNNEST(report)) AS report"
	FieldBillingReportCSPFullTable          string = "ARRAY(SELECT AS STRUCT cost, usage, savings, credit, CAST(NULL AS FLOAT64) AS margin, CAST(NULL AS STRUCT<key STRING, value FLOAT64, type STRING>) AS ext_metric FROM UNNEST(report)) AS report"
)

func GetCspReportFields(
	nullifyFields bool,
	fullTableMode bool,
	isLooker bool,
) string {
	cspReportFields := []string{
		domainQuery.FieldCustomerType,
		domainQuery.FieldPrimaryDomain,
		domainQuery.FieldClassification,
		domainQuery.FieldTerritory,
		domainQuery.FieldPayeeCountry,
		domainQuery.FieldPayerCountry,
		domainQuery.FieldFSR,
		domainQuery.FieldSAM,
		domainQuery.FieldTAM,
		domainQuery.FieldCSM,
	}

	if !fullTableMode {
		// add fields that are not available in the "FULL" table
		cspReportFields = append(cspReportFields, domainQuery.FieldCommitment)
	}

	if nullifyFields {
		for i, v := range cspReportFields {
			if v == domainQuery.FieldCustomerType && isLooker {
				cspReportFields[i] = fmt.Sprintf("\"%s\" AS %s", common.AssetTypeResold, v)
			} else {
				cspReportFields[i] = fmt.Sprintf("NULL AS %s", v)
			}
		}
	}

	return strings.Join(cspReportFields, consts.Comma)
}

func GetCSPMetadataProject() string {
	return common.ProjectID
}

func GetCSPMetadataDataset() string {
	return "cloud_analytics"
}
