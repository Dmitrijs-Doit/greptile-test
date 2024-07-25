package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub"
	eventpb "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub/proto"
)

func TestAggregateEventsMetadata(t *testing.T) {
	tests := []struct {
		name   string
		events []*eventpb.Event
		want   domain.CustomerMetadata
	}{
		{
			name: "a bunch of events with fixed values and labels",
			events: []*eventpb.Event{
				{
					BillingAccountId: makeStringPtr("test-ba-1"),
					ProjectId:        makeStringPtr("test-project-id-1"),
					Cloud:            "Datadog",
					Location: &eventpb.Event_Location{
						Location: makeStringPtr("location-1"),
						Country:  makeStringPtr("country-1"),
					},
					Labels: map[string]string{
						"label-1": "label-1-value-1",
					},
					ProjectLabels: map[string]string{
						"project-label-1": "project-label-1-value-1",
					},
				},
				{
					BillingAccountId:   makeStringPtr("test-ba-2"),
					ProjectId:          makeStringPtr("test-project-id-2"),
					ServiceDescription: makeStringPtr("service-description-1"),
					ServiceId:          makeStringPtr("service-id-1"),
					SkuId:              makeStringPtr("sku-id-1"),
					SkuDescription:     makeStringPtr("sku-description-1"),
					Operation:          makeStringPtr("operation-1"),
					ResourceId:         makeStringPtr("resource-id-1"),
					ResourceGlobalId:   makeStringPtr("resource-global-id-1"),
				},
				{
					BillingAccountId: makeStringPtr("test-ba-3"),
					Cloud:            "Databricks",
					ProjectId:        makeStringPtr("test-project-id-3"),
					ProjectName:      makeStringPtr("project-name-1"),
					ProjectNumber:    makeStringPtr("project-number-1"),
					Location: &eventpb.Event_Location{
						Location: makeStringPtr("location-1"),
						Country:  makeStringPtr("country-1"),
					},
					Labels: map[string]string{
						"label-1": "label-1-value-2",
						"label-2": "label-2-value-1",
					},
				},
				{
					BillingAccountId: makeStringPtr("test-ba-1"),
					ProjectId:        makeStringPtr("test-project-id-1"),
					Cloud:            "Datadog",
					Location: &eventpb.Event_Location{
						Location: makeStringPtr("location-2"),
						Country:  makeStringPtr("country-2"),
						Region:   makeStringPtr("region-1"),
						Zone:     makeStringPtr("zone-1"),
					},
					Labels: map[string]string{
						"label-1": "label-1-value-1",
						"label-2": "label-2-value-2",
					},
				},
			},
			want: domain.CustomerMetadata{
				"cloud_provider":               map[string][]string{"cloud_provider": {"Datadog", "Databricks", "Datadog"}},
				"billing_account_id":           map[string][]string{"billing_account_id": {"test-ba-1", "test-ba-2", "test-ba-3", "test-ba-1"}},
				"project_id":                   map[string][]string{"project_id": {"test-project-id-1", "test-project-id-2", "test-project-id-3", "test-project-id-1"}},
				"project_name":                 map[string][]string{"project_name": {"project-name-1"}},
				"project_number":               map[string][]string{"project_number": {"project-number-1"}},
				"service_description":          map[string][]string{"service_description": {"service-description-1"}},
				"service_id":                   map[string][]string{"service_id": {"service-id-1"}},
				"sku_description":              map[string][]string{"sku_description": {"sku-description-1"}},
				"sku_id":                       map[string][]string{"sku_id": {"sku-id-1"}},
				"operation":                    map[string][]string{"operation": {"operation-1"}},
				"location":                     map[string][]string{"location": {"location-1", "location-1", "location-2"}},
				"country":                      map[string][]string{"country": {"country-1", "country-1", "country-2"}},
				"region":                       map[string][]string{"region": {"region-1"}},
				"zone":                         map[string][]string{"zone": {"zone-1"}},
				"resource_id":                  map[string][]string{"resource_id": {"resource-id-1"}},
				"global_resource_id":           map[string][]string{"global_resource_id": {"resource-global-id-1"}},
				"optional:labels_keys":         map[string][]string{"labels_keys": {"label-1", "label-2"}},
				"optional:project_labels_keys": map[string][]string{"project_labels_keys": {"project-label-1"}},
				"label": map[string][]string{
					"label-1": {"label-1-value-1", "label-1-value-2", "label-1-value-1"},
					"label-2": {"label-2-value-1", "label-2-value-2"},
				},
				"project_label": map[string][]string{
					"project-label-1": {"project-label-1-value-1"},
				},
			},
		},
	}

	for _, tt := range tests {
		s := &DataHubMetadata{}

		t.Run(tt.name, func(t *testing.T) {
			got := s.aggregateEventsMetadata(tt.events)
			assert.Equal(t, tt.want, got)
		})
	}
}

func makeStringPtr(s string) *string {
	return &s
}
