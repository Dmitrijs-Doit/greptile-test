package service

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	eventpb "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub/proto"
)

func TestDataHubMetric(t *testing.T) {
	tests := []struct {
		name   string
		events []*eventpb.Event
		want   map[string]bool
	}{
		{
			name: "a bunch of events with fixed values and labels",
			events: []*eventpb.Event{
				{
					BillingAccountId: makeStringPtr("test-ba-1"),
					Metrics: []*eventpb.Event_Metric{
						{
							Type:  "cost",
							Value: 100,
						},
						{
							Type:  "rides",
							Value: 5,
						},
						{
							Type:  "deployments",
							Value: 2,
						},
						{
							Type:  "savings",
							Value: 13,
						},
					},
				},
				{
					BillingAccountId: makeStringPtr("test-ba-1"),
					Metrics: []*eventpb.Event_Metric{
						{
							Type:  "usage",
							Value: 2,
						},
						{
							Type:  "sales",
							Value: 3,
						},
						{
							Type:  "deployments",
							Value: 100,
						},
					},
				},
			},
			want: map[string]bool{
				"rides":       true,
				"deployments": true,
				"sales":       true,
			},
		},
	}

	for _, tt := range tests {
		s := &DataHubMetadata{}

		t.Run(tt.name, func(t *testing.T) {
			got := s.extractExtendedMetricTypesFromEvents(tt.events)

			gotArr := make([]string, len(got))

			idx := 0
			for key, _ := range got {
				gotArr[idx] = key
				idx++
			}

			sort.Strings(gotArr)

			wantArr := make([]string, len(tt.want))
			idx = 0
			for key, _ := range tt.want {
				wantArr[idx] = key
				idx++
			}

			sort.Strings(wantArr)

			assert.Equal(t, wantArr, gotArr)
		})
	}
}
