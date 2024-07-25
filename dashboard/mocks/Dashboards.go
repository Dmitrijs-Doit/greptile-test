// Code generated by mockery v2.40.3. DO NOT EDIT.

package mocks

import (
	context "context"

	dashboard "github.com/doitintl/hello/scheduled-tasks/dashboard"

	firestore "cloud.google.com/go/firestore"

	mock "github.com/stretchr/testify/mock"
)

// Dashboards is an autogenerated mock type for the Dashboards type
type Dashboards struct {
	mock.Mock
}

// GetCustomerDashboardsWithCloudReports provides a mock function with given fields: ctx, customerID
func (_m *Dashboards) GetCustomerDashboardsWithCloudReports(ctx context.Context, customerID string) ([]*dashboard.Dashboard, error) {
	ret := _m.Called(ctx, customerID)

	if len(ret) == 0 {
		panic("no return value specified for GetCustomerDashboardsWithCloudReports")
	}

	var r0 []*dashboard.Dashboard
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]*dashboard.Dashboard, error)); ok {
		return rf(ctx, customerID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []*dashboard.Dashboard); ok {
		r0 = rf(ctx, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dashboard.Dashboard)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomerTicketStatistics provides a mock function with given fields: ctx, customerID
func (_m *Dashboards) GetCustomerTicketStatistics(ctx context.Context, customerID string) ([]*dashboard.TicketSummary, error) {
	ret := _m.Called(ctx, customerID)

	if len(ret) == 0 {
		panic("no return value specified for GetCustomerTicketStatistics")
	}

	var r0 []*dashboard.TicketSummary
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]*dashboard.TicketSummary, error)); ok {
		return rf(ctx, customerID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []*dashboard.TicketSummary); ok {
		r0 = rf(ctx, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dashboard.TicketSummary)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDashboardsWithCloudReportsCustomerIDs provides a mock function with given fields: ctx
func (_m *Dashboards) GetDashboardsWithCloudReportsCustomerIDs(ctx context.Context) ([]string, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GetDashboardsWithCloudReportsCustomerIDs")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]string, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []string); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDashboardsWithPaths provides a mock function with given fields: ctx, paths
func (_m *Dashboards) GetDashboardsWithPaths(ctx context.Context, paths []string) ([]*dashboard.Dashboard, error) {
	ret := _m.Called(ctx, paths)

	if len(ret) == 0 {
		panic("no return value specified for GetDashboardsWithPaths")
	}

	var r0 []*dashboard.Dashboard
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []string) ([]*dashboard.Dashboard, error)); ok {
		return rf(ctx, paths)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []string) []*dashboard.Dashboard); ok {
		r0 = rf(ctx, paths)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dashboard.Dashboard)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []string) error); ok {
		r1 = rf(ctx, paths)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RemoveDashboardWidget provides a mock function with given fields: ctx, dashboardRef, widget
func (_m *Dashboards) RemoveDashboardWidget(ctx context.Context, dashboardRef *firestore.DocumentRef, widget dashboard.DashboardWidget) error {
	ret := _m.Called(ctx, dashboardRef, widget)

	if len(ret) == 0 {
		panic("no return value specified for RemoveDashboardWidget")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, dashboard.DashboardWidget) error); ok {
		r0 = rf(ctx, dashboardRef, widget)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewDashboards creates a new instance of Dashboards. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDashboards(t interface {
	mock.TestingT
	Cleanup(func())
}) *Dashboards {
	mock := &Dashboards{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
