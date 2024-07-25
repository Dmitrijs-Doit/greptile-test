package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cloudtasksMock "github.com/doitintl/cloudtasks/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	alertsMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	alertsTierService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service/alerttier"
	alertTierMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service/alerttier/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/utils"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	analyticsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/mocks"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
	dispatcherMock "github.com/doitintl/hello/scheduled-tasks/zapier/dispatch/mocks"
)

const (
	testAlertID = "test_alert_id"
)

var now = time.Now()

var qr = &cloudanalytics.QueryRequest{
	Metric: report.MetricCost,
	Rows: []*domainQuery.QueryRequestX{
		{},
	},
	Cols: []*domainQuery.QueryRequestX{
		{
			Key: "year",
		},
		{
			Key: "month",
		},
		{
			Key: "day",
		},
	},
}

func TestAnalyticsAlertsService_checkCondition(t *testing.T) {
	config := &domain.Config{
		Values:   []float64{100},
		Operator: report.MetricFilterGreaterThan,
	}

	type args struct {
		config *domain.Config
		value  float64
	}

	tests := []struct {
		name     string
		args     args
		want     bool
		operator report.MetricFilter
		values   []float64
	}{
		{
			name: "condition isn't met",
			args: args{
				config: config,
				value:  100,
			},
			operator: report.MetricFilterGreaterThan,
			values:   []float64{100},
			want:     false,
		},
		{
			name: "greater than condition is met",
			args: args{
				config: config,
				value:  101,
			},
			operator: report.MetricFilterGreaterThan,
			values:   []float64{100},
			want:     true,
		},
		{
			name: "less than condition is met",
			args: args{
				config: config,
				value:  99,
			},
			operator: report.MetricFilterLessThan,
			values:   []float64{100},
			want:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}
			config.Operator = tt.operator
			config.Values = tt.values

			if got := s.checkCondition(tt.args.config, tt.args.value); got != tt.want {
				t.Errorf("AnalyticsAlertsService.checkCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyticsAlertsService_validateAlert(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		err     error
		testNum int
	}{
		{
			name:    "valid alert",
			wantErr: false,
			err:     nil,
			testNum: 0,
		},
		{
			name:    "alert config values is empty",
			wantErr: true,
			err:     errors.New("alert config values is empty"),
			testNum: 1,
		},
		{
			name:    "collaborators is empty",
			wantErr: true,
			err:     errors.New("collaborators is empty"),
			testNum: 2,
		},
		{
			name:    "customer is null",
			wantErr: true,
			err:     errors.New("customer is null"),
			testNum: 3,
		},
		{
			name:    "config is null",
			wantErr: true,
			err:     errors.New("config is null"),
			testNum: 4,
		},
		{
			name:    "dimension is gke",
			wantErr: true,
			err:     errors.New("alert config rows contains gke dimension, which isn't supported"),
			testNum: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}
			alert := domain.Alert{
				Config: &domain.Config{
					Values:    []float64{100},
					Operator:  report.MetricFilterGreaterThan,
					Scope:     []*firestore.DocumentRef{{ID: "test"}},
					Condition: domain.ConditionForecast,
				},
				Recipients: []string{"test"},

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
				Name: "test_alert",
				Etag: "11123",
			}

			switch tt.testNum {
			case 1:
				alert.Config.Values = []float64{}
			case 2:
				alert.Collaborators = []collab.Collaborator{}
			case 3:
				alert.Customer = nil
			case 4:
				alert.Config = nil
			case 5:
				alert.Config.Rows = []string{"gke"}
			}

			err := s.validateAlert(&alert)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.validateAlert() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("AnalyticsAlertsService.validateAlert() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}

func TestAnalyticsAlertsService_refreshAlert(t *testing.T) {
	ctx := context.Background()
	alertID := "test_alert"

	tests := []struct {
		name    string
		wantErr bool
		testNum int
	}{
		{
			name:    "happy path",
			wantErr: false,
			testNum: 0,
		},
		{
			name:    "error from invalid table cell type - condition 'is'",
			wantErr: true,
			testNum: 1,
		},
		{
			name:    "error from invalid table cell type - condition 'is percentage'",
			wantErr: true,
			testNum: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cloudanalytics.QueryResult{
				Rows: [][]bigquery.Value{
					{
						"breakdown",
						"2022",
						"01",
						"01",
						float64(101),
					},
				},
			}

			alert := domain.Alert{
				Config: &domain.Config{
					Values:    []float64{100},
					Operator:  report.MetricFilterGreaterThan,
					Condition: domain.ConditionValue,
					Rows: []string{
						"breakdown",
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
			}

			s := &AnalyticsAlertsService{
				eventDispatcher: &dispatcherMock.Dispatcher{},
				loggerProvider: func(_ context.Context) logger.ILogger {
					return &loggerMocks.ILogger{}
				},
			}

			switch tt.testNum {
			case 0:
				alert.Config.Condition = domain.ConditionForecast
				result.ForecastRows = [][]bigquery.Value{
					{
						"some_value",
						"2022",
						"01",
						"01",
						float64(101),
					},
				}
			case 1:
				result.Rows[0][4] = "failed value"
			case 2:
				alert.Config.Condition = domain.ConditionPercentage
				result.Rows = [][]bigquery.Value{
					{
						"some_value",
						"2022",
						"01",
						"01",
						"ignored data",
						"ignored data",
						"ignored data",
						float64(101),
						float64(101),
					},
				}
			}

			if err := s.refreshAlert(ctx, &alert, result, qr, alertID); (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.RefreshAlert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalyticsAlertsService_checkRowForAlertPercentage(t *testing.T) {
	type args struct {
		row         []bigquery.Value
		alert       *domain.Alert
		metricIndex int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				row: []bigquery.Value{
					cloudanalytics.ComparativeColumnValue{
						Pct: float64(99),
					},
				},
				alert: &domain.Alert{
					Config: &domain.Config{
						Values:    []float64{100},
						Operator:  report.MetricFilterGreaterThan,
						Condition: domain.ConditionValue,
						Rows: []string{
							"breakdown",
						},
					},
				},
				metricIndex: 0,
			},
			wantErr: false,
		},
		{
			name: "rowData.Pct is int",
			args: args{
				row: []bigquery.Value{
					cloudanalytics.ComparativeColumnValue{
						Pct: int(99),
					},
				},
				alert: &domain.Alert{
					Config: &domain.Config{
						Values:    []float64{100},
						Operator:  report.MetricFilterGreaterThan,
						Condition: domain.ConditionValue,
						Rows: []string{
							"breakdown",
						},
					},
				},
				metricIndex: 0,
			},
			wantErr: false,
		},
		{
			name: "rowData.Pct is string - error",
			args: args{
				row: []bigquery.Value{
					cloudanalytics.ComparativeColumnValue{
						Pct: "99",
					},
				},
				alert: &domain.Alert{
					Config: &domain.Config{
						Values:    []float64{100},
						Operator:  report.MetricFilterGreaterThan,
						Condition: domain.ConditionValue,
						Rows: []string{
							"breakdown",
						},
					},
				},
				metricIndex: 0,
			},
			wantErr: true,
		},
		{
			name: "pct is 0",
			args: args{
				row: []bigquery.Value{
					cloudanalytics.ComparativeColumnValue{
						Pct: float64(0),
					},
				},
				alert: &domain.Alert{
					Config: &domain.Config{
						Values:    []float64{100},
						Operator:  report.MetricFilterGreaterThan,
						Condition: domain.ConditionValue,
						Rows: []string{
							"breakdown",
						},
					},
				},
				metricIndex: 0,
			},
			wantErr: false,
		},
		{
			name: "invalid table cell",
			args: args{
				row: []bigquery.Value{
					"invalid type",
				},
				alert: &domain.Alert{
					Config: &domain.Config{
						Values:    []float64{100},
						Operator:  report.MetricFilterGreaterThan,
						Condition: domain.ConditionValue,
						Rows: []string{
							"breakdown",
						},
					},
				},
				metricIndex: 0,
			},
			wantErr: true,
		},
	}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{
				loggerProvider: func(_ context.Context) logger.ILogger {
					return &loggerMocks.ILogger{}
				},
			}

			_, err := s.checkRowForAlertPercentage(ctx, tt.args.row, tt.args.alert, tt.args.metricIndex, alertID, &now)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.checkRowForAlertPercentage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalyticsAlertsService_checkRowForAlert(t *testing.T) {
	type args struct {
		row         []bigquery.Value
		alert       *domain.Alert
		metricIndex int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				row: []bigquery.Value{
					float64(100),
				},
				alert: &domain.Alert{
					Config: &domain.Config{
						Values:    []float64{100},
						Operator:  report.MetricFilterGreaterThan,
						Condition: domain.ConditionValue,
						Rows: []string{
							"breakdown",
						},
					},
				},
				metricIndex: 0,
			},
			wantErr: false,
		},
		{
			name: "invalid table cell",
			args: args{
				row: []bigquery.Value{
					"invalid type",
				},
				alert: &domain.Alert{
					Config: &domain.Config{
						Values:    []float64{100},
						Operator:  report.MetricFilterGreaterThan,
						Condition: domain.ConditionValue,
						Rows: []string{
							"breakdown",
						},
					},
				},
				metricIndex: 0,
			},
			wantErr: true,
		},
	}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}

			_, err := s.checkRowForAlertValue(ctx, tt.args.row, tt.args.alert, tt.args.metricIndex, alertID, &now)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.checkRowForAlertValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalyticsAlertsService_newNotification(t *testing.T) {
	type fields struct {
		alertsDal        alertsMock.Alerts
		notificationsDal alertsMock.Notifications
	}

	type args struct {
		alert *domain.Alert
		row   *[]bigquery.Value
		value float64
	}

	rows1 := []bigquery.Value{
		"breakdown",
	}
	rows2 := []bigquery.Value{
		123,
	}
	rows3 := []bigquery.Value{
		"breakdown:123",
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		on      func(f *fields)
		wantErr bool
		err     error
	}{
		{
			name: "notification doesn't meet the condition",
			args: args{
				alert: &domain.Alert{
					Config: &domain.Config{
						Values:    []float64{100},
						Operator:  report.MetricFilterGreaterThan,
						Condition: domain.ConditionValue,
						Rows: []string{
							"breakdown",
						},
						TimeInterval: report.TimeIntervalDay,
					},
				},
				row:   nil,
				value: domain.BreakdownLimitValue,
			},
		},
		{
			name: "getExpireByTime returns error",
			args: args{
				alert: &domain.Alert{
					Config: &domain.Config{
						Values:    []float64{100},
						Operator:  report.MetricFilterGreaterThan,
						Condition: domain.ConditionValue,
						Rows: []string{
							"breakdown",
						},
						TimeInterval: report.TimeIntervalHour,
					},
				},
				row:   &rows1,
				value: 1000,
			},
			on: func(f *fields) {
				f.alertsDal.On("GetRef", mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: true,
			err:     errors.New("invalid alert time interval"),
		},
		{
			name: "bigquery value is not string",
			args: args{
				alert: &domain.Alert{
					Config: &domain.Config{
						Values:    []float64{100},
						Operator:  report.MetricFilterGreaterThan,
						Condition: domain.ConditionValue,
						Rows: []string{
							"breakdown",
						},
						TimeInterval: report.TimeIntervalMonth,
					},
				},
				row:   &rows2,
				value: 1000,
			},
			on: func(f *fields) {
				f.alertsDal.On("GetRef", mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: true,
			err:     errors.New("unsupported field type for query result, value: 123"),
		},
		{
			name: "breakdown label is not valid",
			args: args{
				alert: &domain.Alert{
					Config: &domain.Config{
						Values:    []float64{100},
						Operator:  report.MetricFilterGreaterThan,
						Condition: domain.ConditionValue,
						Rows: []string{
							"breakdown",
						},
						TimeInterval: report.TimeIntervalMonth,
					},
				},
				row:   &rows1,
				value: 1000,
			},
			on: func(f *fields) {
				f.alertsDal.On("GetRef", mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: true,
			err:     errors.New("label id is not valid"),
		},
		{
			name: "add notification without errors",
			args: args{
				alert: &domain.Alert{
					Config: &domain.Config{
						Values:    []float64{100},
						Operator:  report.MetricFilterGreaterThan,
						Condition: domain.ConditionValue,
						Rows: []string{
							"breakdown:123",
						},
						TimeInterval: report.TimeIntervalMonth,
					},
				},
				row:   &rows3,
				value: 1000,
			},
			on: func(f *fields) {
				f.alertsDal.On("GetRef", mock.Anything, mock.Anything).Return(nil)
			},
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				alertsDal:        alertsMock.Alerts{},
				notificationsDal: alertsMock.Notifications{},
			}
			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &AnalyticsAlertsService{
				alertsDal:        &tt.fields.alertsDal,
				notificationsDal: &tt.fields.notificationsDal,
			}

			notification, err := s.newNotification(ctx, tt.args.alert, tt.args.row, tt.args.value, alertID, &now)
			if tt.wantErr {
				assert.Equal(t, tt.err, err)
			} else {
				assert.NoError(t, err)
			}

			if notification != nil {
				assert.Equal(t, *notification.Breakdown, "breakdown:123")
				assert.Equal(t, notification.Value, float64(1000))
			}
		})
	}
}

func getQueryRequestExpectedStruct() cloudanalytics.QueryRequest {
	var nilArr []string

	billingDataSource := report.DataSourceBilling

	return cloudanalytics.QueryRequest{
		Accounts:     []string{"account1"},
		Attributions: []*domainQuery.QueryRequestX{},
		Cols: []*domainQuery.QueryRequestX{
			{
				Type:     "datetime",
				Position: "col",
				ID:       "datetime:year",
				Field:    "T.usage_date_time",
				Key:      "year",
				Label:    "Year",
			},
			{
				Type:     "datetime",
				Position: "col",
				ID:       "datetime:month",
				Field:    "T.usage_date_time",
				Key:      "month",
				Label:    "Month",
			},
			{
				Type:     "datetime",
				Position: "col",
				ID:       "datetime:day",
				Field:    "T.usage_date_time",
				Key:      "day",
				Label:    "Day",
			},
		},
		DataSource: &billingDataSource,
		Filters: []*domainQuery.QueryRequestX{
			{
				Type:            "attribution",
				Position:        "unused",
				ID:              "attribution:attribution",
				Key:             "attribution",
				IncludeInFilter: true,
				Values:          &[]string{"test"},
			},
			{
				LimitConfig: domainQuery.LimitConfig{
					Limit:       10,
					LimitOrder:  &[]string{"desc"}[0],
					LimitMetric: &[]int{0}[0],
				},
				Type:            "fixed",
				Position:        "row",
				ID:              "fixed:country",
				Field:           "T.location.country",
				Key:             "country",
				IncludeInFilter: true,
				Inverse:         true,
				Values:          &nilArr,
			},
		},
		LimitAggregation: report.LimitAggregationNone,
		Origin:           domainOrigin.QueryOriginAlerts,
		Rows: []*domainQuery.QueryRequestX{
			{
				Type:     "fixed",
				Position: "row",
				ID:       "fixed:country",
				Field:    "T.location.country",
				Key:      "country",
				Label:    "Country",
			},
		},
		TimeSettings: &cloudanalytics.QueryRequestTimeSettings{
			Interval: "day",
			From:     &[]time.Time{time.Date(2023, 6, 22, 0, 0, 0, 0, time.UTC)}[0],
			To:       &[]time.Time{time.Date(2023, 6, 26, 0, 0, 0, 0, time.UTC)}[0],
		},
		MetricFiltres: []*domainQuery.QueryRequestMetricFilter{},
	}
}

func TestAnalyticsAlertsService_getReportRequest(t *testing.T) {
	type fields struct {
		dal             *alertsMock.Alerts
		notificationDal *alertsMock.Notifications
		cloudAnalytics  *analyticsMocks.CloudAnalytics
	}

	period := time.Now().UTC()
	formattedDate := getFormattedDate(report.TimeIntervalDay, period)

	expectedRequestIs := getQueryRequestExpectedStruct()

	expectedRequestPercentage := getQueryRequestExpectedStruct()
	expectedRequestPercentage.Comparative = &[]string{report.ComparativePercentageChange}[0]
	expectedRequestPercentage.MetricFiltres = []*domainQuery.QueryRequestMetricFilter{
		{
			Metric:   0,
			Operator: report.MetricFilterNotBetween,
			Values:   []float64{-1, 1},
		},
	}
	expectedRequestPercentage.Filters = append(expectedRequestPercentage.Filters, &domainQuery.QueryRequestX{
		Type:            "fixed",
		Position:        "unused",
		ID:              "fixed:credit",
		Field:           "report_value.credit",
		Key:             "credit",
		IncludeInFilter: true,
		Inverse:         true,
		Values:          &[]string{"Sustained Usage Discount"},
	})

	tests := []struct {
		name    string
		want    *cloudanalytics.QueryRequest
		wantErr bool
		testNum int
		fields  fields
		on      func(f *fields)
	}{
		{
			name:    "condition is",
			wantErr: false,
			testNum: 0,
			on: func(f *fields) {
				f.notificationDal.On("GetDetectedBreakdowns", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string"), testAlertID, formattedDate).Return([]string{}, 0, nil)
				f.cloudAnalytics.On("GetAccounts", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]string{"account1"}, nil)
				f.cloudAnalytics.On("GetAttributions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*domainQuery.QueryRequestX{}, nil)
			},
		},
		{
			name:    "condition percentage",
			wantErr: false,
			testNum: 1,
			on: func(f *fields) {
				f.notificationDal.On("GetDetectedBreakdowns", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string"), testAlertID, formattedDate).Return([]string{}, 0, nil)
				f.cloudAnalytics.On("GetAccounts", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]string{"account1"}, nil)
				f.cloudAnalytics.On("GetAttributions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*domainQuery.QueryRequestX{}, nil)
			},
		},
		{
			name:    "condition forecast",
			wantErr: false,
			testNum: 2,
			on: func(f *fields) {
				f.notificationDal.On("GetDetectedBreakdowns", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string"), testAlertID, formattedDate).Return([]string{}, 0, nil)
				f.cloudAnalytics.On("GetAccounts", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]string{"account1"}, nil)
				f.cloudAnalytics.On("GetAttributions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*domainQuery.QueryRequestX{}, nil)
			},
		},
		{
			name:    "with filters",
			wantErr: false,
			testNum: 3,
			on: func(f *fields) {
				f.notificationDal.On("GetDetectedBreakdowns", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string"), testAlertID, formattedDate).Return([]string{}, 0, nil)
				f.cloudAnalytics.On("GetAccounts", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]string{"account1"}, nil)
				f.cloudAnalytics.On("GetAttributions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*domainQuery.QueryRequestX{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				dal:             &alertsMock.Alerts{},
				notificationDal: &alertsMock.Notifications{},
				cloudAnalytics:  &analyticsMocks.CloudAnalytics{},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &AnalyticsAlertsService{
				alertsDal:        tt.fields.dal,
				notificationsDal: tt.fields.notificationDal,
				cloudAnalytics:   tt.fields.cloudAnalytics,
			}

			alert := utils.GenerateTestAlert()

			ctx := context.Background()
			ctx = context.WithValue(ctx, domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginAlerts)

			switch tt.testNum {
			case 1:
				alert.Config.Condition = domain.ConditionPercentage
			case 2:
				alert.Config.Condition = domain.ConditionForecast
			case 3:
				alert = utils.GenerateTestAlertWithFilters()
			}

			got, err := s.getReportRequest(ctx, alert, testAlertID)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.getReportRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && tt.testNum == 0 {
				got.TimeSettings = expectedRequestIs.TimeSettings // ignore time settings, since they depend on the current time
				assert.EqualValues(t, expectedRequestIs, *got)
			}

			if err == nil && tt.testNum == 1 {
				got.TimeSettings = expectedRequestPercentage.TimeSettings // ignore time settings, since they depend on the current time
				assert.EqualValues(t, expectedRequestPercentage, *got)
			}

			if err == nil && tt.testNum == 3 {
				got.TimeSettings = expectedRequestIs.TimeSettings // ignore time settings, since they depend on the current time
				assert.EqualValues(t, expectedRequestIs, *got)
			}
		})
	}
}

func TestAnalyticsAlertsService_buildBreakdownFilter(t *testing.T) {
	ctx := context.Background()
	alert := utils.GenerateTestAlert()
	alert.Config.TimeInterval = report.TimeIntervalMonth
	limitOrder := "desc"
	limitMetric := int(alert.Config.Metric)
	row := alert.Config.Rows[0]

	expected := &report.ConfigFilter{
		BaseConfigFilter: report.BaseConfigFilter{
			ID:      row,
			Inverse: true,
			Values:  &[]string{"israel"},
		},
		Limit:       domain.BreakdownLimitValue - 1,
		LimitOrder:  &limitOrder,
		LimitMetric: &limitMetric,
	}

	period := time.Now().UTC()
	formattedDate := getFormattedDate(report.TimeIntervalMonth, period)

	t.Run("buildBreakdownFilter success with valid return", func(t *testing.T) {
		mockedNotificationsDal := new(alertsMock.Notifications)
		s := AnalyticsAlertsService{
			notificationsDal: mockedNotificationsDal,
		}

		mockedNotificationsDal.On("GetDetectedBreakdowns", testutils.ContextBackgroundMock, alert.Etag, testAlertID, formattedDate).Return([]string{"israel"}, 1, nil)
		got, _ := s.buildBreakdownFilter(ctx, alert, testAlertID, row)
		assert.EqualValues(t, expected, got, "buildBreakdownFilter() = %+v, want %+v", got, expected)
	})
	t.Run("buildBreakdownFilter fail with error return", func(t *testing.T) {
		mockedNotificationsDal := new(alertsMock.Notifications)
		s := AnalyticsAlertsService{
			notificationsDal: mockedNotificationsDal,
		}
		// error from notificationsDal.GetDetectedBreakdowns
		mockedNotificationsDal.On("GetDetectedBreakdowns", testutils.ContextBackgroundMock, alert.Etag, testAlertID, formattedDate).Return([]string{}, 0, errors.New("error"))
		_, err := s.buildBreakdownFilter(ctx, alert, testAlertID, row)
		assert.Error(t, err, "buildBreakdownFilter() should return error")
	})
	t.Run("buildBreakdownFilter returns nil because of 10 unsentDetectedNotifications", func(t *testing.T) {
		mockedNotificationsDal := new(alertsMock.Notifications)
		s := AnalyticsAlertsService{
			notificationsDal: mockedNotificationsDal,
		}
		// error from notificationsDal.GetDetectedBreakdowns
		mockedNotificationsDal.On("GetDetectedBreakdowns", testutils.ContextBackgroundMock, alert.Etag, testAlertID, formattedDate).Return([]string{}, 10, nil)
		got, _ := s.buildBreakdownFilter(ctx, alert, testAlertID, row)
		assert.Nil(t, got, "buildBreakdownFilter() should return nil")
	})
}

func TestAnalyticsAlertsService_RefreshAlerts(t *testing.T) {
	type fields struct {
		loggerProvider   loggerMocks.ILogger
		alertsDal        alertsMock.Alerts
		cloudTaskClient  cloudtasksMock.CloudTaskClient
		alertTierService alertTierMocks.AlertTierService
	}

	customerID := "some customer id"

	customerRef := firestore.DocumentRef{ID: customerID}
	alerts := []domain.Alert{
		{
			Customer: &customerRef,
			ID:       "alert1",
		},
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		on      func(f *fields)
	}{
		{
			name:    "happy path",
			wantErr: false,
			on: func(f *fields) {
				f.alertsDal.On("GetAlerts", mock.Anything).Return(alerts, nil)
				f.alertTierService.On(
					"CheckAccessToAlerts",
					testutils.ContextBackgroundMock,
					customerID,
				).
					Return(nil, nil)
				f.cloudTaskClient.On("CreateTask", mock.Anything, mock.Anything).Return(&cloudtasksMock.Task{}, nil)
			},
		},
		{
			name:    "do not create a task when customer does not have 'alerts' entitlement ",
			wantErr: false,
			on: func(f *fields) {
				f.alertsDal.On("GetAlerts", mock.Anything).Return(alerts, nil)
				f.alertTierService.On(
					"CheckAccessToAlerts",
					testutils.ContextBackgroundMock,
					customerID,
				).Return(&alertsTierService.AccessDeniedAlerts, nil)
				f.loggerProvider.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"))
			},
		},
		{
			name:    "error getting alerts",
			wantErr: true,
			on: func(f *fields) {
				f.alertsDal.On("GetAlerts", mock.Anything).Return(nil, errors.New("error"))
			},
		},
		{
			name:    "error creating task",
			wantErr: false,
			on: func(f *fields) {
				f.alertsDal.On("GetAlerts", mock.Anything).Return(alerts, nil)
				f.alertTierService.On(
					"CheckAccessToAlerts",
					testutils.ContextBackgroundMock,
					customerID,
				).Return(nil, nil)
				f.cloudTaskClient.On("CreateTask", mock.Anything, mock.Anything).Return(nil, errors.New("error"))
				f.loggerProvider.On("Errorf", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:   loggerMocks.ILogger{},
				alertsDal:        alertsMock.Alerts{},
				cloudTaskClient:  cloudtasksMock.CloudTaskClient{},
				alertTierService: alertTierMocks.AlertTierService{},
			}
			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &AnalyticsAlertsService{
				alertsDal:       &tt.fields.alertsDal,
				cloudTaskClient: &tt.fields.cloudTaskClient,
				loggerProvider: func(_ context.Context) logger.ILogger {
					return &tt.fields.loggerProvider
				},
				alertTierService: &tt.fields.alertTierService,
			}

			ctx := context.Background()

			if err := s.RefreshAlerts(ctx); (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.RefreshAlerts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalyticsAlertsService_checkRowsForAlertForecast(t *testing.T) {
	today := time.Now().UTC()

	type args struct {
		ctx     context.Context
		rows    [][]bigquery.Value
		alert   *domain.Alert
		alertID string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "no rows",
			args: args{
				ctx:     context.Background(),
				rows:    [][]bigquery.Value{},
				alert:   &domain.Alert{},
				alertID: "",
			},
			wantErr: true,
			err:     errors.New("no forecast rows"),
		},
		{
			name: "happy path",
			args: args{
				ctx: context.Background(),
				rows: [][]bigquery.Value{
					{
						"1",
						fmt.Sprint(today.Year()),
						fmt.Sprint(today.Format("01")),
						fmt.Sprint(today.Format("02")),
						float64(0),
					},
				},
				alert: &domain.Alert{
					Config: &domain.Config{
						TimeInterval: report.TimeIntervalWeek,
						Values:       []float64{100},
						Operator:     report.MetricFilterGreaterThan,
					},
				},
				alertID: "1",
			},
			wantErr: false,
			err:     nil,
		},
		{
			name: "error on invalid table cell type",
			args: args{
				ctx: context.Background(),
				rows: [][]bigquery.Value{
					{
						"1",
						fmt.Sprint(today.Year()),
						fmt.Sprint(today.Format("01")),
						fmt.Sprint(today.Format("02")),
						1,
					},
				},
				alert: &domain.Alert{
					Config: &domain.Config{
						TimeInterval: report.TimeIntervalWeek,
						Values:       []float64{100},
						Operator:     report.MetricFilterGreaterThan,
					},
				},
				alertID: "1",
			},
			wantErr: true,
			err:     errors.New("invalid table cell"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}

			_, err := s.checkRowsForAlertForecast(tt.args.ctx, tt.args.rows, tt.args.alert, tt.args.alertID, qr)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.checkRowsForAlertForecast() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.err != nil && err.Error() != tt.err.Error() {
				t.Errorf("AnalyticsAlertsService.getTimeFromForecastRow() error = %v, wantErr %v", err, tt.err)
			}
		})
	}
}

func TestAnalyticsAlertsService_getExpireByTime(t *testing.T) {
	type args struct {
		timeInterval report.TimeInterval
	}

	now := time.Now().UTC()
	timeTruncated := now.Truncate(time.Hour * 24)

	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    time.Time
	}{
		{
			name: "get expire by time for day",
			args: args{
				timeInterval: report.TimeIntervalDay,
			},
			wantErr: false,
			want:    timeTruncated.AddDate(0, 2, 0),
		},
		{
			name: "get expire by time for week",
			args: args{
				timeInterval: report.TimeIntervalWeek,
			},
			wantErr: false,
			want:    timeTruncated.AddDate(0, 3, 0),
		},
		{
			name: "get expire by time for month",
			args: args{
				timeInterval: report.TimeIntervalMonth,
			},
			wantErr: false,
			want:    timeTruncated.AddDate(0, 6, 0),
		},
		{
			name: "get expire by time for quarter",
			args: args{
				timeInterval: report.TimeIntervalQuarter,
			},
			wantErr: false,
			want:    timeTruncated.AddDate(1, 0, 0),
		},
		{
			name: "get expire by time for year",
			args: args{
				timeInterval: report.TimeIntervalYear,
			},
			wantErr: false,
			want:    timeTruncated.AddDate(3, 0, 0),
		},
		{
			name: "get expire by time for invalid time interval (hour)",
			args: args{
				timeInterval: report.TimeIntervalHour,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}

			_, err := s.getExpireByTime(tt.args.timeInterval, now)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.getExpireByTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestAnalyticsAlertsService_getFormattedDate(t *testing.T) {
	type args struct {
		timeInterval report.TimeInterval
		date         time.Time
	}

	date := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	quarterNum := (int(date.Month()) / 3) + 1
	quarter := fmt.Sprintf("%s-Q%d", date.Format("2006"), quarterNum)
	year, week := date.ISOWeek()
	weekFormatted := fmt.Sprintf("%d-W%02d", year, week)

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get formatted date for day",
			args: args{
				timeInterval: report.TimeIntervalDay,
				date:         date,
			},
			want: "2020-01-01",
		},
		{
			name: "get formatted date for week",
			args: args{
				timeInterval: report.TimeIntervalWeek,
				date:         date,
			},
			want: weekFormatted,
		},
		{
			name: "get formatted date for month",
			args: args{
				timeInterval: report.TimeIntervalMonth,
				date:         date,
			},
			want: "2020-01",
		},
		{
			name: "get formatted date for quarter",
			args: args{
				timeInterval: report.TimeIntervalQuarter,
				date:         date,
			},
			want: quarter,
		},
		{
			name: "get formatted date for year",
			args: args{
				timeInterval: report.TimeIntervalYear,
				date:         date,
			},
			want: "2020",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, getFormattedDate(tt.args.timeInterval, tt.args.date), "getFormattedDate(%v, %v)", tt.args.timeInterval, tt.args.date)
		})
	}
}

func TestAnalyticsAlertsService_getTimeSettings(t *testing.T) {
	type args struct {
		today  time.Time
		config *domain.Config
	}

	testToday := time.Date(2023, 05, 01, 0, 0, 0, 0, time.UTC)
	percent := report.ComparativePercentageChange

	tests := []struct {
		name                string
		args                args
		wantTimeSettings    *report.TimeSettings
		wantCustomTimeRange *report.ConfigCustomTimeRange
		wantComparative     *string
	}{
		{
			name: "Condition is value and interval is day",
			args: args{
				today: testToday,
				config: &domain.Config{
					Condition:    domain.ConditionValue,
					TimeInterval: report.TimeIntervalDay,
				},
			},
			wantTimeSettings: &report.TimeSettings{
				Mode: report.TimeSettingsModeCustom,
			},
			wantCustomTimeRange: &report.ConfigCustomTimeRange{
				From: time.Date(2023, 04, 27, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2023, 05, 01, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Condition is value and interval is week",
			args: args{
				today: testToday,
				config: &domain.Config{
					Condition:    domain.ConditionValue,
					TimeInterval: report.TimeIntervalWeek,
				},
			},
			wantTimeSettings: &report.TimeSettings{
				Mode:           report.TimeSettingsModeLast,
				Unit:           report.TimeSettingsUnitDay,
				Amount:         7,
				IncludeCurrent: true,
			},
		},
		{
			name: "Condition is value and interval is month",
			args: args{
				today: testToday,
				config: &domain.Config{
					Condition:    domain.ConditionValue,
					TimeInterval: report.TimeIntervalMonth,
				},
			},
			wantTimeSettings: &report.TimeSettings{
				Mode: report.TimeSettingsModeCurrent,
				Unit: report.TimeSettingsUnitMonth,
			},
		},
		{
			name: "Condition is value and interval is quarter",
			args: args{
				today: testToday,
				config: &domain.Config{
					Condition:    domain.ConditionValue,
					TimeInterval: report.TimeIntervalQuarter,
				},
			},
			wantTimeSettings: &report.TimeSettings{
				Mode: report.TimeSettingsModeCurrent,
				Unit: report.TimeSettingsUnitQuarter,
			},
		},
		{
			name: "Condition is value and interval is year",
			args: args{
				today: testToday,
				config: &domain.Config{
					Condition:    domain.ConditionValue,
					TimeInterval: report.TimeIntervalYear,
				},
			},
			wantTimeSettings: &report.TimeSettings{
				Mode: report.TimeSettingsModeCurrent,
				Unit: report.TimeSettingsUnitYear,
			},
		},
		{
			name: "Condition is percentage and interval is day",
			args: args{
				today: testToday,
				config: &domain.Config{
					Condition:    domain.ConditionPercentage,
					TimeInterval: report.TimeIntervalDay,
				},
			},
			wantTimeSettings: &report.TimeSettings{
				Mode: report.TimeSettingsModeCustom,
			},
			wantCustomTimeRange: &report.ConfigCustomTimeRange{
				From: time.Date(2023, 04, 26, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2023, 05, 01, 0, 0, 0, 0, time.UTC),
			},
			wantComparative: &percent,
		},
		{
			name: "Condition is percentage and interval is week",
			args: args{
				today: testToday,
				config: &domain.Config{
					Condition:    domain.ConditionPercentage,
					TimeInterval: report.TimeIntervalWeek,
				},
			},
			wantTimeSettings: &report.TimeSettings{
				Mode:           report.TimeSettingsModeLast,
				Unit:           report.TimeSettingsUnitWeek,
				Amount:         2,
				IncludeCurrent: true,
			},
			wantComparative: &percent,
		},
		{
			name: "Condition is percentage and interval is month",
			args: args{
				today: testToday,
				config: &domain.Config{
					Condition:    domain.ConditionPercentage,
					TimeInterval: report.TimeIntervalMonth,
				},
			},
			wantTimeSettings: &report.TimeSettings{
				Mode:           report.TimeSettingsModeLast,
				Unit:           report.TimeSettingsUnitMonth,
				Amount:         2,
				IncludeCurrent: true,
			},
			wantComparative: &percent,
		},
		{
			name: "Condition is percentage and interval is quarter",
			args: args{
				today: testToday,
				config: &domain.Config{
					Condition:    domain.ConditionPercentage,
					TimeInterval: report.TimeIntervalQuarter,
				},
			},
			wantTimeSettings: &report.TimeSettings{
				Mode:           report.TimeSettingsModeLast,
				Unit:           report.TimeSettingsUnitQuarter,
				Amount:         2,
				IncludeCurrent: true,
			},
			wantComparative: &percent,
		},
		{
			name: "Condition is percentage and interval is year",
			args: args{
				today: testToday,
				config: &domain.Config{
					Condition:    domain.ConditionPercentage,
					TimeInterval: report.TimeIntervalYear,
				},
			},
			wantTimeSettings: &report.TimeSettings{
				Mode: report.TimeSettingsModeCustom,
			},
			wantCustomTimeRange: &report.ConfigCustomTimeRange{
				From: time.Date(2022, 05, 01, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2023, 05, 01, 0, 0, 0, 0, time.UTC),
			},
			wantComparative: &percent,
		},
		{
			name: "Condition is forecast and interval is week",
			args: args{
				today: testToday,
				config: &domain.Config{
					Condition:    domain.ConditionForecast,
					TimeInterval: report.TimeIntervalWeek,
				},
			},
			wantTimeSettings: &report.TimeSettings{
				Mode:           report.TimeSettingsModeLast,
				Unit:           report.TimeSettingsUnitMonth,
				Amount:         3,
				IncludeCurrent: true,
			},
		},
		{
			name: "Condition is forecast and interval is year",
			args: args{
				today: testToday,
				config: &domain.Config{
					Condition:    domain.ConditionForecast,
					TimeInterval: report.TimeIntervalYear,
				},
			},
			wantTimeSettings: &report.TimeSettings{
				Mode:           report.TimeSettingsModeLast,
				Unit:           report.TimeSettingsUnitMonth,
				Amount:         12,
				IncludeCurrent: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}
			gotTimeSettings, gotCustomTimeRange, gotComparative := s.getTimeSettings(tt.args.today, tt.args.config)
			assert.Equal(t, tt.wantTimeSettings, gotTimeSettings)
			assert.Equal(t, tt.wantCustomTimeRange, gotCustomTimeRange)
			assert.Equal(t, tt.wantComparative, gotComparative)
		})
	}
}

func TestAnalyticsAlertsService_isFirstPercentageDay(t *testing.T) {
	type args struct {
		rowTimeDetected *time.Time
	}

	testToday := time.Now().UTC().Truncate(time.Hour * 24)

	rowTimeDetected := testToday.AddDate(0, 0, -PercentageDailyRange)

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Row time detected is nil",
			args: args{
				rowTimeDetected: nil,
			},
			want: false,
		},
		{
			name: "Row time detected is not nil",
			args: args{
				rowTimeDetected: &rowTimeDetected,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}
			if got := s.isFirstPercentageDay(tt.args.rowTimeDetected); got != tt.want {
				t.Errorf("AnalyticsAlertsService.isFirstPercentageDay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyticsAlertsService_convertRowStringToInt(t *testing.T) {
	type args struct {
		bqValue bigquery.Value
	}

	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "Valid int",
			args: args{
				bqValue: "100",
			},
			want: 100,
		},
		{
			name: "Invalid int",
			args: args{
				bqValue: "100.1",
			},
			wantErr: true,
		},
		{
			name: "Invalid row type",
			args: args{
				bqValue: 100,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AnalyticsAlertsService{}

			got, err := s.convertRowStringToInt(tt.args.bqValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.convertRowStringToInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("AnalyticsAlertsService.convertRowStringToInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyticsAlertsService_getTimestampFromRow(t *testing.T) {
	type fields struct {
		Rows []*domainQuery.QueryRequestX
		Cols []*domainQuery.QueryRequestX
	}

	type args struct {
		row []bigquery.Value
	}

	colsDaily := []*domainQuery.QueryRequestX{
		{
			Key: "year",
		},
		{
			Key: "month",
		},
		{
			Key: "day",
		},
	}

	colsMonthly := []*domainQuery.QueryRequestX{
		{
			Key: "year",
		},
		{
			Key: "month",
		},
	}

	colsYearly := []*domainQuery.QueryRequestX{
		{
			Key: "year",
		},
	}

	wantTimeDay := time.Date(2021, 05, 01, 0, 0, 0, 0, time.UTC)
	wantTimeMonth := time.Date(2021, 05, 01, 0, 0, 0, 0, time.UTC)
	wantTimeYear := time.Date(2021, 01, 01, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *time.Time
		wantErr bool
	}{
		{
			name: "Valid date",
			fields: fields{
				Cols: colsDaily,
				Rows: []*domainQuery.QueryRequestX{{}},
			},
			args: args{
				row: []bigquery.Value{
					"breakdown",
					"2021",
					"05",
					"01",
				},
			},
			want: &wantTimeDay,
		},
		{
			name: "invalid row length",
			fields: fields{
				Cols: colsDaily,
				Rows: []*domainQuery.QueryRequestX{{}},
			},
			args: args{
				row: []bigquery.Value{},
			},
			wantErr: true,
		},
		{
			name: "Invalid year",
			fields: fields{
				Cols: colsDaily,
				Rows: []*domainQuery.QueryRequestX{{}},
			},
			args: args{
				row: []bigquery.Value{
					"breakdown",
					2021,
					"05",
					"01",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid month",
			fields: fields{
				Cols: colsDaily,
				Rows: []*domainQuery.QueryRequestX{{}},
			},
			args: args{
				row: []bigquery.Value{
					"breakdown",
					"2021",
					05,
					"01",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid day",
			fields: fields{
				Cols: colsDaily,
				Rows: []*domainQuery.QueryRequestX{{}},
			},
			args: args{
				row: []bigquery.Value{
					"breakdown",
					"2021",
					"05",
					01,
				},
			},
			wantErr: true,
		},
		{
			name: "get timestamp from row monthly",
			fields: fields{
				Cols: colsMonthly,
				Rows: []*domainQuery.QueryRequestX{{}},
			},
			args: args{
				row: []bigquery.Value{
					"breakdown",
					"2021",
					"05",
				},
			},
			want: &wantTimeMonth,
		},
		{
			name: "get timestamp from row yearly",
			fields: fields{
				Cols: colsYearly,
				Rows: []*domainQuery.QueryRequestX{{}},
			},
			args: args{
				row: []bigquery.Value{
					"breakdown",
					"2021",
				},
			},
			want: &wantTimeYear,
		},
		{
			name: "Invalid: missing month",
			fields: fields{
				Cols: colsMonthly,
				Rows: []*domainQuery.QueryRequestX{{}},
			},
			args: args{
				row: []bigquery.Value{
					"breakdown",
					"2021",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid: missing year",
			fields: fields{
				Cols: colsYearly,
				Rows: []*domainQuery.QueryRequestX{{}},
			},
			args: args{
				row: []bigquery.Value{
					"breakdown",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid day- 32 days in a month",
			fields: fields{
				Cols: colsDaily,
				Rows: []*domainQuery.QueryRequestX{{}},
			},
			args: args{
				row: []bigquery.Value{
					"breakdown",
					"2021",
					"05",
					"32",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid month- 13 months in a year",
			fields: fields{
				Cols: colsDaily,
				Rows: []*domainQuery.QueryRequestX{{}},
			},
			args: args{
				row: []bigquery.Value{
					"breakdown",
					"2021",
					"13",
					"1",
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid year- year cannot be 0",
			fields: fields{
				Cols: colsDaily,
				Rows: []*domainQuery.QueryRequestX{{}},
			},
			args: args{
				row: []bigquery.Value{
					"breakdown",
					"0",
					"13",
					"1",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qr := &cloudanalytics.QueryRequest{
				Rows: tt.fields.Rows,
				Cols: tt.fields.Cols,
			}
			s := &AnalyticsAlertsService{}

			got, err := s.getTimestampFromRow(qr, tt.args.row)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryRequest.GetTimestampFromRow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("QueryRequest.GetTimestampFromRow() = %v, want %v", got, tt.want)
			}
		})
	}
}
