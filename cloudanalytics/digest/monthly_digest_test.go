package digest

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	fsdal "github.com/doitintl/firestore"
	announcekitMocks "github.com/doitintl/hello/scheduled-tasks/announcekit/mocks"
	highchartsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/service/mocks"
	analyticsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard"
	dashboardMocks "github.com/doitintl/hello/scheduled-tasks/dashboard/mocks"
	"github.com/doitintl/hello/scheduled-tasks/fixer/converter/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func setupDigestService(t *testing.T) *DigestService {
	ctx := context.Background()

	logging, err := logger.NewLogging(ctx)
	if err != nil {
		t.Error(err)
	}

	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		t.Error(err)
	}

	widgetService, err := widget.NewWidgetService(logger.FromContext, conn)
	if err != nil {
		panic(err)
	}

	return &DigestService{
		logger.FromContext,
		conn,
		&analyticsMocks.CloudAnalytics{},
		&dashboardMocks.Dashboards{},
		fsdal.NewIntegrationsDALWithClient(conn.Firestore(context.Background())),
		&mocks.Converter{},
		widgetService,
		&announcekitMocks.AnnounceKit{},
		nil,
		nil,
		&highchartsMocks.IHighcharts{},
	}
}
func TestGetCustomerSupportTickets(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	s := setupDigestService(t)
	dal := s.dashboardDAL.(*dashboardMocks.Dashboards)

	testCases := []struct {
		name          string
		expected      []*dashboard.TicketSummary
		ticketSummary []*dashboard.TicketSummary
		errorInput    error
	}{
		{
			name:          "failure",
			expected:      nil,
			errorInput:    errors.New("failed to get customer support tickets"),
			ticketSummary: nil,
		},
		{
			name: "success",
			expected: []*dashboard.TicketSummary{
				{
					ID:      11,
					Subject: "Test",
					Score:   "Good",
				},
			},
			errorInput: nil,
			ticketSummary: []*dashboard.TicketSummary{
				{
					ID:      11,
					Subject: "Test",
					Score:   "good",
				},
			},
		},
		{
			name: "success without score",
			expected: []*dashboard.TicketSummary{
				{
					ID:      11,
					Subject: "Test",
					Score:   "Rate Now",
				},
			},
			errorInput: nil,
			ticketSummary: []*dashboard.TicketSummary{
				{
					ID:      11,
					Subject: "Test",
					Score:   "unoffered",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dal.On("GetCustomerTicketStatistics", mock.Anything, mock.AnythingOfType("string")).
				Return(tc.ticketSummary, tc.errorInput).
				Once()

			actual, err := s.getCustomerSupportTickets(ctx, "customerID")
			t.Logf("%+v", actual)
			assert.Equal(tc.expected, actual)

			if err != nil {
				assert.Equal(tc.errorInput, err)
			}
		})
	}
}

func TestTrendArrow(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name     string
		value    float64
		expected TrendProperites
	}{
		{
			name:  "positive",
			value: 12.3,
			expected: TrendProperites{
				Color: green,
				Value: "↑12.3",
			},
		},
		{
			name:  "negative",
			value: -4.3,
			expected: TrendProperites{
				Color: red,
				Value: "↓4.3",
			},
		},
		{
			name:  "zero",
			value: 0.0,
			expected: TrendProperites{
				Color: black,
				Value: "0",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(tc.expected, trendArrow(tc.value))
		})
	}
}

func TestGetMonthlySpendByRow(t *testing.T) {
	assert := assert.New(t)

	type args struct {
		year      int
		month     time.Month
		reportRow []interface{}
	}

	val := 973.4

	testCases := []struct {
		name     string
		args     *args
		expected *float64
	}{
		{
			name: "success",
			args: &args{
				year:  2022,
				month: 1,
				reportRow: []interface{}{
					"google-cloud",
					"2022",
					"01",
					973.4,
					534.3,
					0,
				},
			},
			expected: &val,
		},
		{
			name: "bad row",
			args: &args{
				year:  2022,
				month: 1,
				reportRow: []interface{}{
					"2022",
					"01",
					973.4,
				},
			},
			expected: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(tc.expected, getMonthlySpendByRow(tc.args.year, tc.args.month, tc.args.reportRow))
		})
	}
}

func TestCalculateTrend(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name      string
		month     float64
		prevMonth float64
		expected  float64
	}{
		{
			name:      "trend down",
			month:     0,
			prevMonth: 1,
			expected:  -100,
		},
		{
			name:      "trend no change",
			month:     1,
			prevMonth: 0,
			expected:  0,
		},
		{
			name:      "trend up",
			month:     2,
			prevMonth: 1,
			expected:  100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(tc.expected, calculateTrend(tc.month, tc.prevMonth))
		})
	}
}

func TestGetPlatformLogo(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(awsLogoURL, getPlatformLogo("AWS"))
}

func TestGetPlatform(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(common.Assets.AmazonWebServices, getPlatform("AWS"))
	assert.Equal(common.Assets.GoogleCloud, getPlatform("GCP"))
}

func TestGetDailyForecast(t *testing.T) {
	assert := assert.New(t)
	testCases := []struct {
		name     string
		month    float64
		row      [][]bigquery.Value
		expected float64
	}{
		{
			name:  "one day",
			month: 5,
			row: [][]bigquery.Value{
				{"forecast", "2022", "05", "05", 4.4},
			},
			expected: 4.4,
		},
		{
			name:  "multiple days",
			month: 5,
			row: [][]bigquery.Value{
				{"forecast", "2022", "05", "01", 4.4},
				{"forecast", "2022", "05", "02", 4.4},
				{"forecast", "2022", "05", "03", 4.4},
			},
			expected: 13.200000000000001,
		},
		{
			name:  "row is invalid",
			month: 5,
			row: [][]bigquery.Value{
				{"forecast", "2022", "05", "01"},
			},
			expected: 0,
		},
		{
			name:  "empty row",
			month: 5,
			row: [][]bigquery.Value{
				{},
			},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(tc.expected, sumDailyForecast(tc.row, time.Month(tc.month)))
		})
	}
}
