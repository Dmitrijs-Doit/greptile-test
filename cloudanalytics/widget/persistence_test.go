package widget

import (
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
)

type DaysAgo struct {
	Days float32
}

func (h *DaysAgo) Time() time.Time {
	if h.Days == 0 {
		return time.Time{}
	}

	return time.Now().Add(time.Duration(-h.Days*24) * time.Hour)
}

func TestWidgetService_shouldUpdateWidgetForAccessAndRefreshTime(t *testing.T) {
	type args struct {
		orgLastAccessed DaysAgo
		timeRefreshed   DaysAgo
	}

	tests := []struct {
		args args
		want bool
	}{
		{
			args: args{
				orgLastAccessed: DaysAgo{0},
				timeRefreshed:   DaysAgo{1},
			},
			want: false,
		},
		{
			args: args{
				orgLastAccessed: DaysAgo{1},
				timeRefreshed:   DaysAgo{0.1},
			},
			want: false,
		},
		{
			args: args{
				orgLastAccessed: DaysAgo{1},
				timeRefreshed:   DaysAgo{0.6},
			},
			want: true,
		},
		{
			args: args{
				orgLastAccessed: DaysAgo{4},
				timeRefreshed:   DaysAgo{3.1},
			},
			want: true,
		},
		{
			args: args{
				orgLastAccessed: DaysAgo{4},
				timeRefreshed:   DaysAgo{0.01},
			},
			want: false,
		},
		{
			args: args{
				orgLastAccessed: DaysAgo{10},
				timeRefreshed:   DaysAgo{10},
			},
			want: true,
		},
		{
			args: args{
				orgLastAccessed: DaysAgo{10},
				timeRefreshed:   DaysAgo{5},
			},
			want: false,
		},
		{
			args: args{
				orgLastAccessed: DaysAgo{20},
				timeRefreshed:   DaysAgo{20},
			},
			want: true,
		},
		{
			args: args{
				orgLastAccessed: DaysAgo{20},
				timeRefreshed:   DaysAgo{10},
			},
			want: false,
		},
		{
			args: args{
				orgLastAccessed: DaysAgo{35},
				timeRefreshed:   DaysAgo{20},
			},
			want: true,
		},
		{
			args: args{
				orgLastAccessed: DaysAgo{35},
				timeRefreshed:   DaysAgo{15},
			},
			want: false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			s := &WidgetService{}
			if got := s.shouldUpdateWidgetForAccessAndRefreshTime(tt.args.orgLastAccessed.Time(), tt.args.timeRefreshed.Time()); got != tt.want {
				t.Errorf("shouldUpdateWidgetForAccessAndRefreshTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWidgetService_shouldUpdateWidget(t *testing.T) {
	type args struct {
		widgetReportTimeRefreshed time.Time
		organization              *common.Organization
		report                    report.Report
		minUpdateDelayMinutes     int
	}

	now := time.Now()

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "No widget refresh time",
			args: args{
				widgetReportTimeRefreshed: time.Time{},
				organization:              nil,
				report:                    report.Report{},
			},
			want: true,
		},
		{
			name: "Custom minUpdateDelayMinutes provided and refresh earlier",
			args: args{
				widgetReportTimeRefreshed: now.Add(time.Duration(-5) * time.Minute),
				organization:              nil,
				report: report.Report{
					TimeModified: now.Add(time.Duration(-6) * time.Minute),
				},
				minUpdateDelayMinutes: 10,
			},
			want: false,
		},
		{
			name: "Custom minUpdateDelayMinutes provided and refresh later",
			args: args{
				widgetReportTimeRefreshed: now.Add(time.Duration(-20) * time.Minute),
				organization:              nil,
				report: report.Report{
					TimeModified: now.Add(time.Duration(-21) * time.Minute),
				},
				minUpdateDelayMinutes: 10,
			},
			want: true,
		},
		{
			name: "Root org with report modified before refresh and widget refreshed less than 1 hour ago",
			args: args{
				widgetReportTimeRefreshed: now.Add(time.Duration(-5) * time.Minute),
				organization:              orgWithIDAndLastAccessed(organizations.RootOrgID, time.Time{}),
				report: report.Report{
					TimeModified: now.Add(time.Duration(-20) * time.Minute),
				},
			},
			want: false,
		},
		{
			name: "Root org with report modified before refresh and widget refreshed more than 12 hour ago",
			args: args{
				widgetReportTimeRefreshed: now.Add(time.Duration(-13) * time.Hour),
				organization:              orgWithIDAndLastAccessed(organizations.RootOrgID, time.Time{}),
				report: report.Report{
					TimeModified: now.Add(time.Duration(-1) * time.Hour),
				},
			},
			want: true,
		},
		{
			name: "Root org with report modified after last refresh and widget refreshed less than 1 hour ago",
			args: args{
				widgetReportTimeRefreshed: now.Add(time.Duration(-15) * time.Minute),
				organization:              orgWithIDAndLastAccessed(organizations.RootOrgID, time.Time{}),
				report: report.Report{
					TimeModified: now.Add(time.Duration(-10) * time.Minute),
				},
			},
			want: true,
		},
		{
			name: "Root org with report modified after last refresh and widget refreshed more than 1 hour ago",
			args: args{
				widgetReportTimeRefreshed: now.Add(time.Duration(-90) * time.Minute),
				organization:              orgWithIDAndLastAccessed(organizations.RootOrgID, time.Time{}),
				report: report.Report{
					TimeModified: now.Add(time.Duration(-20) * time.Minute),
				},
			},
			want: true,
		},
		{
			name: "Preset org and widget refreshed less than 24 hours ago",
			args: args{
				widgetReportTimeRefreshed: now.Add(time.Duration(-2) * time.Hour),
				organization:              orgWithIDAndLastAccessed(organizations.PresetGCPOrgID, time.Time{}),
			},
			want: false,
		},
		{
			name: "Preset org and widget refreshed more than 24 hours ago",
			args: args{
				widgetReportTimeRefreshed: now.Add(time.Duration(-25) * time.Hour),
				organization:              orgWithIDAndLastAccessed(organizations.PresetAWSOrgID, time.Time{}),
			},
			want: true,
		},
		{
			name: "Preset org with report modified after last refresh and widget refreshed less than 1 hour ago",
			args: args{
				widgetReportTimeRefreshed: now.Add(time.Duration(-15) * time.Minute),
				organization:              orgWithIDAndLastAccessed(organizations.PresetGCPOrgID, time.Time{}),
				report: report.Report{
					TimeModified: now.Add(time.Duration(-10) * time.Minute),
				},
			},
			want: true,
		},
		{
			name: "Random org and with last accessed empty and present widgetReportTimeRefreshed",
			args: args{
				widgetReportTimeRefreshed: now,
				organization:              orgWithIDAndLastAccessed("123", time.Time{}),
			},
			want: false,
		},
		{
			name: "Random org with access a day ago and refresh a day ago",
			args: args{
				widgetReportTimeRefreshed: now.Add(time.Duration(-24) * time.Hour),
				organization:              orgWithIDAndLastAccessed("123", now.Add(time.Duration(-24)*time.Hour)),
			},
			want: true,
		},
		{
			name: "Random org with access 5 days ago refresh a day ago",
			args: args{
				widgetReportTimeRefreshed: now.Add(time.Duration(-24) * time.Hour),
				organization:              orgWithIDAndLastAccessed("123", now.Add(time.Duration(-24*7)*time.Hour)),
			},
			want: false,
		},
		{
			name: "Random org with report modified after last refresh and widget refreshed less than 1 hour ago",
			args: args{
				widgetReportTimeRefreshed: now.Add(time.Duration(-15) * time.Minute),
				organization:              orgWithIDAndLastAccessed("123", time.Time{}),
				report: report.Report{
					TimeModified: now.Add(time.Duration(-10) * time.Minute),
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &WidgetService{}
			if got := s.shouldUpdateWidget(tt.args.widgetReportTimeRefreshed, tt.args.organization, &tt.args.report, tt.args.minUpdateDelayMinutes); got != tt.want {
				t.Errorf("shouldUpdateWidget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func orgWithIDAndLastAccessed(id string, lastAccessed time.Time) *common.Organization {
	return &common.Organization{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: &firestore.DocumentRef{ID: id},
		},
		LastAccessed: lastAccessed,
	}
}
