package utils

import (
	"fmt"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func GenerateBaseTestAlert() *domain.Alert {
	return &domain.Alert{
		Config: &domain.Config{
			DataSource: report.DataSourceBilling,
			Values:     []float64{100},
			Operator:   report.MetricFilterGreaterThan,
			Condition:  domain.ConditionValue,
			Rows: []string{
				"fixed:country",
			},
			TimeInterval: report.TimeIntervalDay,
			IgnoreValuesRange: &domain.IgnoreValuesRange{
				LowerBound: -1,
				UpperBound: 1,
			},
		},
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{
					Email: "",
				},
			},
		},
		Customer: &firestore.DocumentRef{
			ID: "test_customer",
		},
		Etag: "123",
	}
}

func GenerateTestAlert() *domain.Alert {
	alert := GenerateBaseTestAlert()
	alert.Config.Scope = []*firestore.DocumentRef{{ID: "test"}}

	return alert
}

func GenerateTestAlertWithFilters() *domain.Alert {
	filter := report.ConfigFilter{
		BaseConfigFilter: report.BaseConfigFilter{
			ID:     fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, metadata.MetadataFieldTypeAttribution),
			Key:    string(metadata.MetadataFieldTypeAttribution),
			Type:   metadata.MetadataFieldTypeAttribution,
			Values: &[]string{"test"},
		},
	}

	alert := GenerateBaseTestAlert()
	alert.Config.Filters = []*report.ConfigFilter{&filter}

	return alert
}
