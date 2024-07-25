// Code generated by mockery v2.35.2. DO NOT EDIT.

package mocks

import (
	context "context"

	cloudanalytics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"

	externalreport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"

	mock "github.com/stretchr/testify/mock"

	report "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

// ReportTierService is an autogenerated mock type for the ReportTierService type
type ReportTierService struct {
	mock.Mock
}

// CheckAccessToCustomReport provides a mock function with given fields: ctx, customerID
func (_m *ReportTierService) CheckAccessToCustomReport(ctx context.Context, customerID string) (*domain.AccessDeniedError, error) {
	ret := _m.Called(ctx, customerID)

	var r0 *domain.AccessDeniedError
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*domain.AccessDeniedError, error)); ok {
		return rf(ctx, customerID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *domain.AccessDeniedError); ok {
		r0 = rf(ctx, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.AccessDeniedError)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CheckAccessToExtendedMetric provides a mock function with given fields: ctx, customerID, extendedMetric
func (_m *ReportTierService) CheckAccessToExtendedMetric(ctx context.Context, customerID string, extendedMetric string) (*domain.AccessDeniedError, error) {
	ret := _m.Called(ctx, customerID, extendedMetric)

	var r0 *domain.AccessDeniedError
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*domain.AccessDeniedError, error)); ok {
		return rf(ctx, customerID, extendedMetric)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *domain.AccessDeniedError); ok {
		r0 = rf(ctx, customerID, extendedMetric)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.AccessDeniedError)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, customerID, extendedMetric)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CheckAccessToExternalReport provides a mock function with given fields: ctx, customerID, externalReport, checkFeaturesAccess
func (_m *ReportTierService) CheckAccessToExternalReport(ctx context.Context, customerID string, externalReport *externalreport.ExternalReport, checkFeaturesAccess bool) (*domain.AccessDeniedError, error) {
	ret := _m.Called(ctx, customerID, externalReport, checkFeaturesAccess)

	var r0 *domain.AccessDeniedError
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *externalreport.ExternalReport, bool) (*domain.AccessDeniedError, error)); ok {
		return rf(ctx, customerID, externalReport, checkFeaturesAccess)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *externalreport.ExternalReport, bool) *domain.AccessDeniedError); ok {
		r0 = rf(ctx, customerID, externalReport, checkFeaturesAccess)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.AccessDeniedError)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *externalreport.ExternalReport, bool) error); ok {
		r1 = rf(ctx, customerID, externalReport, checkFeaturesAccess)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CheckAccessToPresetReport provides a mock function with given fields: ctx, customerID
func (_m *ReportTierService) CheckAccessToPresetReport(ctx context.Context, customerID string) (*domain.AccessDeniedError, error) {
	ret := _m.Called(ctx, customerID)

	var r0 *domain.AccessDeniedError
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*domain.AccessDeniedError, error)); ok {
		return rf(ctx, customerID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *domain.AccessDeniedError); ok {
		r0 = rf(ctx, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.AccessDeniedError)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CheckAccessToQueryRequest provides a mock function with given fields: ctx, customerID, qr
func (_m *ReportTierService) CheckAccessToQueryRequest(ctx context.Context, customerID string, qr *cloudanalytics.QueryRequest) (*domain.AccessDeniedError, error) {
	ret := _m.Called(ctx, customerID, qr)

	var r0 *domain.AccessDeniedError
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *cloudanalytics.QueryRequest) (*domain.AccessDeniedError, error)); ok {
		return rf(ctx, customerID, qr)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *cloudanalytics.QueryRequest) *domain.AccessDeniedError); ok {
		r0 = rf(ctx, customerID, qr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.AccessDeniedError)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *cloudanalytics.QueryRequest) error); ok {
		r1 = rf(ctx, customerID, qr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CheckAccessToReport provides a mock function with given fields: ctx, customerID, _a2
func (_m *ReportTierService) CheckAccessToReport(ctx context.Context, customerID string, _a2 *report.Report) (*domain.AccessDeniedError, error) {
	ret := _m.Called(ctx, customerID, _a2)

	var r0 *domain.AccessDeniedError
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *report.Report) (*domain.AccessDeniedError, error)); ok {
		return rf(ctx, customerID, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *report.Report) *domain.AccessDeniedError); ok {
		r0 = rf(ctx, customerID, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.AccessDeniedError)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *report.Report) error); ok {
		r1 = rf(ctx, customerID, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CheckAccessToReportID provides a mock function with given fields: ctx, customerID, reportID
func (_m *ReportTierService) CheckAccessToReportID(ctx context.Context, customerID string, reportID string) (*domain.AccessDeniedError, error) {
	ret := _m.Called(ctx, customerID, reportID)

	var r0 *domain.AccessDeniedError
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*domain.AccessDeniedError, error)); ok {
		return rf(ctx, customerID, reportID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *domain.AccessDeniedError); ok {
		r0 = rf(ctx, customerID, reportID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.AccessDeniedError)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, customerID, reportID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CheckAccessToReportType provides a mock function with given fields: ctx, customerID, reportType
func (_m *ReportTierService) CheckAccessToReportType(ctx context.Context, customerID string, reportType string) (*domain.AccessDeniedError, error) {
	ret := _m.Called(ctx, customerID, reportType)

	var r0 *domain.AccessDeniedError
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*domain.AccessDeniedError, error)); ok {
		return rf(ctx, customerID, reportType)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *domain.AccessDeniedError); ok {
		r0 = rf(ctx, customerID, reportType)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.AccessDeniedError)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, customerID, reportType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomerEntitlementIDs provides a mock function with given fields: ctx, customerID
func (_m *ReportTierService) GetCustomerEntitlementIDs(ctx context.Context, customerID string) ([]string, error) {
	ret := _m.Called(ctx, customerID)

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]string, error)); ok {
		return rf(ctx, customerID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []string); ok {
		r0 = rf(ctx, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewReportTierService creates a new instance of ReportTierService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewReportTierService(t interface {
	mock.TestingT
	Cleanup(func())
}) *ReportTierService {
	mock := &ReportTierService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
