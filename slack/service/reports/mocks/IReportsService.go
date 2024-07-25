// Code generated by mockery v2.32.0. DO NOT EDIT.

package mocks

import (
	context "context"

	collab "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"

	mock "github.com/stretchr/testify/mock"

	pkg "github.com/doitintl/firestore/pkg"

	report "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"

	slack "github.com/slack-go/slack"
)

// IReportsService is an autogenerated mock type for the IReportsService type
type IReportsService struct {
	mock.Mock
}

// Get provides a mock function with given fields: ctx, customerID, reportID
func (_m *IReportsService) Get(ctx context.Context, customerID string, reportID string) (*report.Report, error) {
	ret := _m.Called(ctx, customerID, reportID)

	var r0 *report.Report
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*report.Report, error)); ok {
		return rf(ctx, customerID, reportID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *report.Report); ok {
		r0 = rf(ctx, customerID, reportID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*report.Report)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, customerID, reportID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUnfurlPayload provides a mock function with given fields: ctx, reportID, customerID, URL
func (_m *IReportsService) GetUnfurlPayload(ctx context.Context, reportID string, customerID string, URL string) (*report.Report, map[string]slack.Attachment, error) {
	ret := _m.Called(ctx, reportID, customerID, URL)

	var r0 *report.Report
	var r1 map[string]slack.Attachment
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string) (*report.Report, map[string]slack.Attachment, error)); ok {
		return rf(ctx, reportID, customerID, URL)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string) *report.Report); ok {
		r0 = rf(ctx, reportID, customerID, URL)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*report.Report)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string) map[string]slack.Attachment); ok {
		r1 = rf(ctx, reportID, customerID, URL)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(map[string]slack.Attachment)
		}
	}

	if rf, ok := ret.Get(2).(func(context.Context, string, string, string) error); ok {
		r2 = rf(ctx, reportID, customerID, URL)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// UpdateSharing provides a mock function with given fields: ctx, reportID, customerID, requester, usersToAdd, role, public
func (_m *IReportsService) UpdateSharing(ctx context.Context, reportID string, customerID string, requester *pkg.User, usersToAdd []string, role collab.CollaboratorRole, public bool) error {
	ret := _m.Called(ctx, reportID, customerID, requester, usersToAdd, role, public)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, *pkg.User, []string, collab.CollaboratorRole, bool) error); ok {
		r0 = rf(ctx, reportID, customerID, requester, usersToAdd, role, public)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIReportsService creates a new instance of IReportsService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewIReportsService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IReportsService {
	mock := &IReportsService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
