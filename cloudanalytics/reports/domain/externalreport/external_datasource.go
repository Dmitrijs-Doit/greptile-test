package externalreport

import (
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type ExternalDataSource string

const (
	ExternalDataSourceBilling        ExternalDataSource = "billing"
	ExternalDataSourceBQLens         ExternalDataSource = "bqlens"
	ExternalDataSourceBillingDataHub ExternalDataSource = "billing-datahub"
)

func NewExternalDatasourceFromInternal(datasource report.DataSource) (*ExternalDataSource, []errormsg.ErrorMsg) {
	var externalDataSource ExternalDataSource

	switch datasource {
	case report.DataSourceBilling:
		externalDataSource = ExternalDataSourceBilling
	case report.DataSourceBillingDataHub:
		externalDataSource = ExternalDataSourceBillingDataHub
	case report.DataSourceBQLens:
		externalDataSource = ExternalDataSourceBQLens
	default:
		return nil, []errormsg.ErrorMsg{
			{
				Field:   DataSourceField,
				Message: fmt.Sprintf("%s: %s", ErrInvalidDatasourceValue, datasource),
			},
		}
	}

	return &externalDataSource, nil
}

func (externalDataSource ExternalDataSource) ToInternal() (*report.DataSource, []errormsg.ErrorMsg) {
	var dataSource report.DataSource

	switch externalDataSource {
	case ExternalDataSourceBilling:
		dataSource = report.DataSourceBilling
	case ExternalDataSourceBQLens:
		dataSource = report.DataSourceBQLens
	case ExternalDataSourceBillingDataHub:
		dataSource = report.DataSourceBillingDataHub
	default:
		return nil, []errormsg.ErrorMsg{
			{
				Field:   DataSourceField,
				Message: fmt.Sprintf("%s: %s", ErrInvalidDatasourceValue, externalDataSource),
			},
		}
	}

	return &dataSource, nil
}

func (externalDataSource ExternalDataSource) ValidateDataSource() bool {
	switch externalDataSource {
	case ExternalDataSourceBilling,
		ExternalDataSourceBQLens,
		ExternalDataSourceBillingDataHub:
		return true
	default:
		return false
	}
}
