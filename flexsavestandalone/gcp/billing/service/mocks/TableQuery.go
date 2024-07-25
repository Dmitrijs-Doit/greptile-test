// Code generated by mockery v2.16.0. DO NOT EDIT.

package mocks

import (
	context "context"

	bigquery "cloud.google.com/go/bigquery"

	dataStructures "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// TableQuery is an autogenerated mock type for the TableQuery type
type TableQuery struct {
	mock.Mock
}

// CopyFromAlternativeLocalToTmpTable provides a mock function with given fields: ctx, itm
func (_m *TableQuery) CopyFromAlternativeLocalToTmpTable(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (*bigquery.Job, error) {
	ret := _m.Called(ctx, itm)

	var r0 *bigquery.Job
	if rf, ok := ret.Get(0).(func(context.Context, *dataStructures.InternalTaskMetadata) *bigquery.Job); ok {
		r0 = rf(ctx, itm)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bigquery.Job)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *dataStructures.InternalTaskMetadata) error); ok {
		r1 = rf(ctx, itm)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CopyFromLocalToTmpTable provides a mock function with given fields: ctx, itm
func (_m *TableQuery) CopyFromLocalToTmpTable(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (*bigquery.Job, error) {
	ret := _m.Called(ctx, itm)

	var r0 *bigquery.Job
	if rf, ok := ret.Get(0).(func(context.Context, *dataStructures.InternalTaskMetadata) *bigquery.Job); ok {
		r0 = rf(ctx, itm)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bigquery.Job)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *dataStructures.InternalTaskMetadata) error); ok {
		r1 = rf(ctx, itm)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CopyFromTmpTableAllRows provides a mock function with given fields: ctx, imm, itms
func (_m *TableQuery) CopyFromTmpTableAllRows(ctx context.Context, imm *dataStructures.InternalManagerMetadata, itms []*dataStructures.InternalTaskMetadata) (*bigquery.Job, error) {
	ret := _m.Called(ctx, imm, itms)

	var r0 *bigquery.Job
	if rf, ok := ret.Get(0).(func(context.Context, *dataStructures.InternalManagerMetadata, []*dataStructures.InternalTaskMetadata) *bigquery.Job); ok {
		r0 = rf(ctx, imm, itms)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bigquery.Job)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *dataStructures.InternalManagerMetadata, []*dataStructures.InternalTaskMetadata) error); ok {
		r1 = rf(ctx, imm, itms)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteRowsFromUnifiedByBA provides a mock function with given fields: ctx, billingAccount
func (_m *TableQuery) DeleteRowsFromUnifiedByBA(ctx context.Context, billingAccount string) (*bigquery.Job, error) {
	ret := _m.Called(ctx, billingAccount)

	var r0 *bigquery.Job
	if rf, ok := ret.Get(0).(func(context.Context, string) *bigquery.Job); ok {
		r0 = rf(ctx, billingAccount)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bigquery.Job)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, billingAccount)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomerRowsCount provides a mock function with given fields: ctx, billingAccountID, segment
func (_m *TableQuery) GetCustomerRowsCount(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[dataStructures.HashableSegment]int, error) {
	ret := _m.Called(ctx, billingAccountID, segment)

	var r0 map[dataStructures.HashableSegment]int
	if rf, ok := ret.Get(0).(func(context.Context, string, *dataStructures.Segment) map[dataStructures.HashableSegment]int); ok {
		r0 = rf(ctx, billingAccountID, segment)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[dataStructures.HashableSegment]int)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *dataStructures.Segment) error); ok {
		r1 = rf(ctx, billingAccountID, segment)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomerRowsCountByExportTime provides a mock function with given fields: ctx, billingAccountID, segment
func (_m *TableQuery) GetCustomerRowsCountByExportTime(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[string]int64, error) {
	ret := _m.Called(ctx, billingAccountID, segment)

	var r0 map[string]int64
	if rf, ok := ret.Get(0).(func(context.Context, string, *dataStructures.Segment) map[string]int64); ok {
		r0 = rf(ctx, billingAccountID, segment)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]int64)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *dataStructures.Segment) error); ok {
		r1 = rf(ctx, billingAccountID, segment)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomerRowsCountPerTimeRange provides a mock function with given fields: ctx, billingAccountID, startTime, endTime
func (_m *TableQuery) GetCustomerRowsCountPerTimeRange(ctx context.Context, billingAccountID string, startTime *time.Time, endTime *time.Time) (int64, error) {
	ret := _m.Called(ctx, billingAccountID, startTime, endTime)

	var r0 int64
	if rf, ok := ret.Get(0).(func(context.Context, string, *time.Time, *time.Time) int64); ok {
		r0 = rf(ctx, billingAccountID, startTime, endTime)
	} else {
		r0 = ret.Get(0).(int64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *time.Time, *time.Time) error); ok {
		r1 = rf(ctx, billingAccountID, startTime, endTime)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomersTableNewestRecordTime provides a mock function with given fields: ctx, customerBQ, t
func (_m *TableQuery) GetCustomersTableNewestRecordTime(ctx context.Context, customerBQ *bigquery.Client, t *dataStructures.BillingTableInfo) (time.Time, error) {
	ret := _m.Called(ctx, customerBQ, t)

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client, *dataStructures.BillingTableInfo) time.Time); ok {
		r0 = rf(ctx, customerBQ, t)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *bigquery.Client, *dataStructures.BillingTableInfo) error); ok {
		r1 = rf(ctx, customerBQ, t)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomersTableOldestRecordTime provides a mock function with given fields: ctx, customerBQ, t
func (_m *TableQuery) GetCustomersTableOldestRecordTime(ctx context.Context, customerBQ *bigquery.Client, t *dataStructures.BillingTableInfo) (time.Time, error) {
	ret := _m.Called(ctx, customerBQ, t)

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client, *dataStructures.BillingTableInfo) time.Time); ok {
		r0 = rf(ctx, customerBQ, t)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *bigquery.Client, *dataStructures.BillingTableInfo) error); ok {
		r1 = rf(ctx, customerBQ, t)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomersTableOldestRecordTimeNewerThan provides a mock function with given fields: ctx, customerBQ, t, minExportTime
func (_m *TableQuery) GetCustomersTableOldestRecordTimeNewerThan(ctx context.Context, customerBQ *bigquery.Client, t *dataStructures.BillingTableInfo, minExportTime time.Time) (time.Time, error) {
	ret := _m.Called(ctx, customerBQ, t, minExportTime)

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client, *dataStructures.BillingTableInfo, time.Time) time.Time); ok {
		r0 = rf(ctx, customerBQ, t, minExportTime)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *bigquery.Client, *dataStructures.BillingTableInfo, time.Time) error); ok {
		r1 = rf(ctx, customerBQ, t, minExportTime)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetFromUnifiedTableRowsCountPerTimeRange provides a mock function with given fields: ctx, billingAccount, startTime, endTime
func (_m *TableQuery) GetFromUnifiedTableRowsCountPerTimeRange(ctx context.Context, billingAccount string, startTime *time.Time, endTime *time.Time) (int64, error) {
	ret := _m.Called(ctx, billingAccount, startTime, endTime)

	var r0 int64
	if rf, ok := ret.Get(0).(func(context.Context, string, *time.Time, *time.Time) int64); ok {
		r0 = rf(ctx, billingAccount, startTime, endTime)
	} else {
		r0 = ret.Get(0).(int64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *time.Time, *time.Time) error); ok {
		r1 = rf(ctx, billingAccount, startTime, endTime)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLUnifiedRowsCount provides a mock function with given fields: ctx, billingAccountID, segment
func (_m *TableQuery) GetLUnifiedRowsCount(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[dataStructures.HashableSegment]int, error) {
	ret := _m.Called(ctx, billingAccountID, segment)

	var r0 map[dataStructures.HashableSegment]int
	if rf, ok := ret.Get(0).(func(context.Context, string, *dataStructures.Segment) map[dataStructures.HashableSegment]int); ok {
		r0 = rf(ctx, billingAccountID, segment)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[dataStructures.HashableSegment]int)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *dataStructures.Segment) error); ok {
		r1 = rf(ctx, billingAccountID, segment)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLUnifiedRowsCountByExportTime provides a mock function with given fields: ctx, billingAccountID, segment
func (_m *TableQuery) GetLUnifiedRowsCountByExportTime(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[string]int64, error) {
	ret := _m.Called(ctx, billingAccountID, segment)

	var r0 map[string]int64
	if rf, ok := ret.Get(0).(func(context.Context, string, *dataStructures.Segment) map[string]int64); ok {
		r0 = rf(ctx, billingAccountID, segment)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]int64)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *dataStructures.Segment) error); ok {
		r1 = rf(ctx, billingAccountID, segment)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLocalRowsCount provides a mock function with given fields: ctx, billingAccountID, segment
func (_m *TableQuery) GetLocalRowsCount(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[dataStructures.HashableSegment]int, error) {
	ret := _m.Called(ctx, billingAccountID, segment)

	var r0 map[dataStructures.HashableSegment]int
	if rf, ok := ret.Get(0).(func(context.Context, string, *dataStructures.Segment) map[dataStructures.HashableSegment]int); ok {
		r0 = rf(ctx, billingAccountID, segment)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[dataStructures.HashableSegment]int)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *dataStructures.Segment) error); ok {
		r1 = rf(ctx, billingAccountID, segment)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLocalRowsCountByExportTime provides a mock function with given fields: ctx, billingAccountID, segment
func (_m *TableQuery) GetLocalRowsCountByExportTime(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[string]int64, error) {
	ret := _m.Called(ctx, billingAccountID, segment)

	var r0 map[string]int64
	if rf, ok := ret.Get(0).(func(context.Context, string, *dataStructures.Segment) map[string]int64); ok {
		r0 = rf(ctx, billingAccountID, segment)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]int64)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *dataStructures.Segment) error); ok {
		r1 = rf(ctx, billingAccountID, segment)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLocalRowsCountPerTimeRange provides a mock function with given fields: ctx, billingAccount, startTime, endTime
func (_m *TableQuery) GetLocalRowsCountPerTimeRange(ctx context.Context, billingAccount string, startTime *time.Time, endTime *time.Time) (int64, error) {
	ret := _m.Called(ctx, billingAccount, startTime, endTime)

	var r0 int64
	if rf, ok := ret.Get(0).(func(context.Context, string, *time.Time, *time.Time) int64); ok {
		r0 = rf(ctx, billingAccount, startTime, endTime)
	} else {
		r0 = ret.Get(0).(int64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *time.Time, *time.Time) error); ok {
		r1 = rf(ctx, billingAccount, startTime, endTime)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLocalTableLatestExportTime provides a mock function with given fields: ctx, bq, billingAccountID
func (_m *TableQuery) GetLocalTableLatestExportTime(ctx context.Context, bq *bigquery.Client, billingAccountID string) (*time.Time, error) {
	ret := _m.Called(ctx, bq, billingAccountID)

	var r0 *time.Time
	if rf, ok := ret.Get(0).(func(context.Context, *bigquery.Client, string) *time.Time); ok {
		r0 = rf(ctx, bq, billingAccountID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*time.Time)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *bigquery.Client, string) error); ok {
		r1 = rf(ctx, bq, billingAccountID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLocalTableNewestRecordTime provides a mock function with given fields: ctx, itm
func (_m *TableQuery) GetLocalTableNewestRecordTime(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (time.Time, error) {
	ret := _m.Called(ctx, itm)

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(context.Context, *dataStructures.InternalTaskMetadata) time.Time); ok {
		r0 = rf(ctx, itm)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *dataStructures.InternalTaskMetadata) error); ok {
		r1 = rf(ctx, itm)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLocalTableOldestRecordTime provides a mock function with given fields: ctx, itm
func (_m *TableQuery) GetLocalTableOldestRecordTime(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (time.Time, error) {
	ret := _m.Called(ctx, itm)

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(context.Context, *dataStructures.InternalTaskMetadata) time.Time); ok {
		r0 = rf(ctx, itm)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *dataStructures.InternalTaskMetadata) error); ok {
		r1 = rf(ctx, itm)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRawBillingNewestRecordTime provides a mock function with given fields: ctx
func (_m *TableQuery) GetRawBillingNewestRecordTime(ctx context.Context) (time.Time, error) {
	ret := _m.Called(ctx)

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(context.Context) time.Time); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRawBillingOldestRecordTime provides a mock function with given fields: ctx
func (_m *TableQuery) GetRawBillingOldestRecordTime(ctx context.Context) (time.Time, error) {
	ret := _m.Called(ctx)

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(context.Context) time.Time); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUnifiedTableNewestRecordByBA provides a mock function with given fields: ctx, itm
func (_m *TableQuery) GetUnifiedTableNewestRecordByBA(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (time.Time, error) {
	ret := _m.Called(ctx, itm)

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(context.Context, *dataStructures.InternalTaskMetadata) time.Time); ok {
		r0 = rf(ctx, itm)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *dataStructures.InternalTaskMetadata) error); ok {
		r1 = rf(ctx, itm)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUnifiedTableOldestRecordByBA provides a mock function with given fields: ctx, itm
func (_m *TableQuery) GetUnifiedTableOldestRecordByBA(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (time.Time, error) {
	ret := _m.Called(ctx, itm)

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(context.Context, *dataStructures.InternalTaskMetadata) time.Time); ok {
		r0 = rf(ctx, itm)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *dataStructures.InternalTaskMetadata) error); ok {
		r1 = rf(ctx, itm)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUnifiedTableOldestRecordTimeNewerThan provides a mock function with given fields: ctx, minExportTime
func (_m *TableQuery) GetUnifiedTableOldestRecordTimeNewerThan(ctx context.Context, minExportTime time.Time) (time.Time, error) {
	ret := _m.Called(ctx, minExportTime)

	var r0 time.Time
	if rf, ok := ret.Get(0).(func(context.Context, time.Time) time.Time); ok {
		r0 = rf(ctx, minExportTime)
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, time.Time) error); ok {
		r1 = rf(ctx, minExportTime)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MarkTmpTableBillingRowsAsVerified provides a mock function with given fields: ctx, itm
func (_m *TableQuery) MarkTmpTableBillingRowsAsVerified(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (*bigquery.Job, error) {
	ret := _m.Called(ctx, itm)

	var r0 *bigquery.Job
	if rf, ok := ret.Get(0).(func(context.Context, *dataStructures.InternalTaskMetadata) *bigquery.Job); ok {
		r0 = rf(ctx, itm)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bigquery.Job)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *dataStructures.InternalTaskMetadata) error); ok {
		r1 = rf(ctx, itm)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RunDataFreshnessReport provides a mock function with given fields: ctx
func (_m *TableQuery) RunDataFreshnessReport(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RunDetailedTableRewritesMapping provides a mock function with given fields: ctx
func (_m *TableQuery) RunDetailedTableRewritesMapping(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// TruncateUnifiedTableContent provides a mock function with given fields: ctx
func (_m *TableQuery) TruncateUnifiedTableContent(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewTableQuery interface {
	mock.TestingT
	Cleanup(func())
}

// NewTableQuery creates a new instance of TableQuery. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewTableQuery(t mockConstructorTestingTNewTableQuery) *TableQuery {
	mock := &TableQuery{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
