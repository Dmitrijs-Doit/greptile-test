// Code generated by mockery v2.41.0. DO NOT EDIT.


package mocks

import (
	cloudanalytics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	budget "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"

	context "context"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/domain"

	mock "github.com/stretchr/testify/mock"

	report "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

// IHighcharts is an autogenerated mock type for the IHighcharts type
type IHighcharts struct {
	mock.Mock
}

// GetBudgetImages provides a mock function with given fields: ctx, budgetID, customerID, highchartsFontSettings
func (_m *IHighcharts) GetBudgetImages(ctx context.Context, budgetID string, customerID string, highchartsFontSettings *domain.HighchartsFontSettings) (string, string, error) {
	ret := _m.Called(ctx, budgetID, customerID, highchartsFontSettings)

	if len(ret) == 0 {
		panic("no return value specified for GetBudgetImages")
	}

	var r0 string
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, *domain.HighchartsFontSettings) (string, string, error)); ok {
		return rf(ctx, budgetID, customerID, highchartsFontSettings)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, *domain.HighchartsFontSettings) string); ok {
		r0 = rf(ctx, budgetID, customerID, highchartsFontSettings)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, *domain.HighchartsFontSettings) string); ok {
		r1 = rf(ctx, budgetID, customerID, highchartsFontSettings)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(context.Context, string, string, *domain.HighchartsFontSettings) error); ok {
		r2 = rf(ctx, budgetID, customerID, highchartsFontSettings)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetChartImage provides a mock function with given fields: ctx, hcr
func (_m *IHighcharts) GetChartImage(ctx context.Context, hcr *domain.HighchartsRequest) ([]byte, error) {
	ret := _m.Called(ctx, hcr)

	if len(ret) == 0 {
		panic("no return value specified for GetChartImage")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *domain.HighchartsRequest) ([]byte, error)); ok {
		return rf(ctx, hcr)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *domain.HighchartsRequest) []byte); ok {
		r0 = rf(ctx, hcr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *domain.HighchartsRequest) error); ok {
		r1 = rf(ctx, hcr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetHighchartsRequestBudget provides a mock function with given fields: utilization, _a1, highchartsFontSettings
func (_m *IHighcharts) GetHighchartsRequestBudget(utilization string, _a1 *budget.Budget, highchartsFontSettings *domain.HighchartsFontSettings) *domain.HighchartsRequest {
	ret := _m.Called(utilization, _a1, highchartsFontSettings)

	if len(ret) == 0 {
		panic("no return value specified for GetHighchartsRequestBudget")
	}

	var r0 *domain.HighchartsRequest
	if rf, ok := ret.Get(0).(func(string, *budget.Budget, *domain.HighchartsFontSettings) *domain.HighchartsRequest); ok {
		r0 = rf(utilization, _a1, highchartsFontSettings)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.HighchartsRequest)
		}
	}

	return r0
}

// GetHighchartsRequestReport provides a mock function with given fields: ctx, reportQueryRequest, reportQueryResult, r, isTreemapExact, highchartsFontSettings
func (_m *IHighcharts) GetHighchartsRequestReport(ctx context.Context, reportQueryRequest *cloudanalytics.QueryRequest, reportQueryResult *cloudanalytics.QueryResult, r *report.Report, isTreemapExact bool, highchartsFontSettings *domain.HighchartsFontSettings) (*domain.HighchartsRequest, error) {
	ret := _m.Called(ctx, reportQueryRequest, reportQueryResult, r, isTreemapExact, highchartsFontSettings)

	if len(ret) == 0 {
		panic("no return value specified for GetHighchartsRequestReport")
	}

	var r0 *domain.HighchartsRequest
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *cloudanalytics.QueryRequest, *cloudanalytics.QueryResult, *report.Report, bool, *domain.HighchartsFontSettings) (*domain.HighchartsRequest, error)); ok {
		return rf(ctx, reportQueryRequest, reportQueryResult, r, isTreemapExact, highchartsFontSettings)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *cloudanalytics.QueryRequest, *cloudanalytics.QueryResult, *report.Report, bool, *domain.HighchartsFontSettings) *domain.HighchartsRequest); ok {
		r0 = rf(ctx, reportQueryRequest, reportQueryResult, r, isTreemapExact, highchartsFontSettings)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.HighchartsRequest)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *cloudanalytics.QueryRequest, *cloudanalytics.QueryResult, *report.Report, bool, *domain.HighchartsFontSettings) error); ok {
		r1 = rf(ctx, reportQueryRequest, reportQueryResult, r, isTreemapExact, highchartsFontSettings)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLabels provides a mock function with given fields: highchartsFontSettings
func (_m *IHighcharts) GetLabels(highchartsFontSettings *domain.HighchartsFontSettings) domain.HighchartsDataAxisLabels {
	ret := _m.Called(highchartsFontSettings)

	if len(ret) == 0 {
		panic("no return value specified for GetLabels")
	}

	var r0 domain.HighchartsDataAxisLabels
	if rf, ok := ret.Get(0).(func(*domain.HighchartsFontSettings) domain.HighchartsDataAxisLabels); ok {
		r0 = rf(highchartsFontSettings)
	} else {
		r0 = ret.Get(0).(domain.HighchartsDataAxisLabels)
	}

	return r0
}

// GetReportImage provides a mock function with given fields: ctx, reportID, customerID, highchartsFontSettings
func (_m *IHighcharts) GetReportImage(ctx context.Context, reportID string, customerID string, highchartsFontSettings *domain.HighchartsFontSettings) (string, error) {
	ret := _m.Called(ctx, reportID, customerID, highchartsFontSettings)

	if len(ret) == 0 {
		panic("no return value specified for GetReportImage")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, *domain.HighchartsFontSettings) (string, error)); ok {
		return rf(ctx, reportID, customerID, highchartsFontSettings)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, *domain.HighchartsFontSettings) string); ok {
		r0 = rf(ctx, reportID, customerID, highchartsFontSettings)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, *domain.HighchartsFontSettings) error); ok {
		r1 = rf(ctx, reportID, customerID, highchartsFontSettings)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetReportImageData provides a mock function with given fields: ctx, reportID, customerID, highchartsFontSettings
func (_m *IHighcharts) GetReportImageData(ctx context.Context, reportID string, customerID string, highchartsFontSettings *domain.HighchartsFontSettings) ([]byte, error) {
	ret := _m.Called(ctx, reportID, customerID, highchartsFontSettings)

	if len(ret) == 0 {
		panic("no return value specified for GetReportImageData")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, *domain.HighchartsFontSettings) ([]byte, error)); ok {
		return rf(ctx, reportID, customerID, highchartsFontSettings)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, *domain.HighchartsFontSettings) []byte); ok {
		r0 = rf(ctx, reportID, customerID, highchartsFontSettings)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, *domain.HighchartsFontSettings) error); ok {
		r1 = rf(ctx, reportID, customerID, highchartsFontSettings)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStyle provides a mock function with given fields: highchartsFontSettings
func (_m *IHighcharts) GetStyle(highchartsFontSettings *domain.HighchartsFontSettings) domain.HighchartsDataStyle {
	ret := _m.Called(highchartsFontSettings)

	if len(ret) == 0 {
		panic("no return value specified for GetStyle")
	}

	var r0 domain.HighchartsDataStyle
	if rf, ok := ret.Get(0).(func(*domain.HighchartsFontSettings) domain.HighchartsDataStyle); ok {
		r0 = rf(highchartsFontSettings)
	} else {
		r0 = ret.Get(0).(domain.HighchartsDataStyle)
	}

	return r0
}

// SaveImageToGCS provides a mock function with given fields: ctx, imageData, chartID, customerID, chartType
func (_m *IHighcharts) SaveImageToGCS(ctx context.Context, imageData []byte, chartID string, customerID string, chartType string) (string, error) {
	ret := _m.Called(ctx, imageData, chartID, customerID, chartType)

	if len(ret) == 0 {
		panic("no return value specified for SaveImageToGCS")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte, string, string, string) (string, error)); ok {
		return rf(ctx, imageData, chartID, customerID, chartType)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte, string, string, string) string); ok {
		r0 = rf(ctx, imageData, chartID, customerID, chartType)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte, string, string, string) error); ok {
		r1 = rf(ctx, imageData, chartID, customerID, chartType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewIHighcharts creates a new instance of IHighcharts. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewIHighcharts(t interface {
	mock.TestingT
	Cleanup(func())
}) *IHighcharts {
	mock := &IHighcharts{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
