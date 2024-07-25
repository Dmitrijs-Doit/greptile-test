// Code generated by mockery v2.43.2. DO NOT EDIT.

package mocks

import (
	context "context"

	bigquery "cloud.google.com/go/bigquery"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/dal"

	domain "github.com/doitintl/hello/scheduled-tasks/bq-lens/discovery/domain"

	mock "github.com/stretchr/testify/mock"
)

// Bigquery is an autogenerated mock type for the Bigquery type
type Bigquery struct {
	mock.Mock
}

// EnsureTableIsCorrect provides a mock function with given fields: ctx, bq
func (_m *Bigquery) EnsureTableIsCorrect(ctx context.Context, bq *bigquery.Client) (*bigquery.Table, error) {
	ret := _m.Called(ctx, bq)

	if len(ret) == 0 {
		panic("no return value specified for EnsureTableIsCorrect")
	}

	var r0 *bigquery.Table
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client) (*bigquery.Table, error)); ok {
		return rf(ctx, bq)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client) *bigquery.Table); ok {
		r0 = rf(ctx, bq)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bigquery.Table)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *bigquery.Client) error); ok {
		r1 = rf(ctx, bq)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRegionsAndStorageBillingModelForProject provides a mock function with given fields: ctx, projectID, bq
func (_m *Bigquery) GetRegionsAndStorageBillingModelForProject(ctx context.Context, projectID string, bq *bigquery.Client) (domain.DatasetStorageBillingModel, []string, error) {
	ret := _m.Called(ctx, projectID, bq)

	if len(ret) == 0 {
		panic("no return value specified for GetRegionsAndStorageBillingModelForProject")
	}

	var r0 domain.DatasetStorageBillingModel
	var r1 []string
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *bigquery.Client) (domain.DatasetStorageBillingModel, []string, error)); ok {
		return rf(ctx, projectID, bq)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *bigquery.Client) domain.DatasetStorageBillingModel); ok {
		r0 = rf(ctx, projectID, bq)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(domain.DatasetStorageBillingModel)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *bigquery.Client) []string); ok {
		r1 = rf(ctx, projectID, bq)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]string)
		}
	}

	if rf, ok := ret.Get(2).(func(context.Context, string, *bigquery.Client) error); ok {
		r2 = rf(ctx, projectID, bq)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// RunDiscoveryQuery provides a mock function with given fields: ctx, bq, query, destinationTable, rowProcessor
func (_m *Bigquery) RunDiscoveryQuery(ctx context.Context, bq *bigquery.Client, query string, destinationTable *bigquery.Table, rowProcessor dal.RowProcessor) error {
	ret := _m.Called(ctx, bq, query, destinationTable, rowProcessor)

	if len(ret) == 0 {
		panic("no return value specified for RunDiscoveryQuery")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client, string, *bigquery.Table, dal.RowProcessor) error); ok {
		r0 = rf(ctx, bq, query, destinationTable, rowProcessor)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewBigquery creates a new instance of Bigquery. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBigquery(t interface {
	mock.TestingT
	Cleanup(func())
}) *Bigquery {
	mock := &Bigquery{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}