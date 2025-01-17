// Code generated by mockery v2.35.2. DO NOT EDIT.

package mocks

import (
	context "context"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"

	mock "github.com/stretchr/testify/mock"
)

// DataHubBigQuery is an autogenerated mock type for the DataHubBigQuery type
type DataHubBigQuery struct {
	mock.Mock
}

// DeleteBigQueryData provides a mock function with given fields: ctx, customerID
func (_m *DataHubBigQuery) DeleteBigQueryData(ctx context.Context, customerID string) error {
	ret := _m.Called(ctx, customerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteBigQueryDataByBatches provides a mock function with given fields: ctx, customerID, datasetName, deleteReq, deletedBy
func (_m *DataHubBigQuery) DeleteBigQueryDataByBatches(ctx context.Context, customerID string, datasetName string, deleteReq domain.DeleteBatchesReq, deletedBy string) error {
	ret := _m.Called(ctx, customerID, datasetName, deleteReq, deletedBy)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, domain.DeleteBatchesReq, string) error); ok {
		r0 = rf(ctx, customerID, datasetName, deleteReq, deletedBy)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteBigQueryDataByClouds provides a mock function with given fields: ctx, customerID, deleteReq, deletedBy
func (_m *DataHubBigQuery) DeleteBigQueryDataByClouds(ctx context.Context, customerID string, deleteReq domain.DeleteDatasetsReq, deletedBy string) error {
	ret := _m.Called(ctx, customerID, deleteReq, deletedBy)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, domain.DeleteDatasetsReq, string) error); ok {
		r0 = rf(ctx, customerID, deleteReq, deletedBy)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteBigQueryDataByEventIDs provides a mock function with given fields: ctx, customerID, deleteEventsReq, deletedBy
func (_m *DataHubBigQuery) DeleteBigQueryDataByEventIDs(ctx context.Context, customerID string, deleteEventsReq domain.DeleteEventsReq, deletedBy string) error {
	ret := _m.Called(ctx, customerID, deleteEventsReq, deletedBy)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, domain.DeleteEventsReq, string) error); ok {
		r0 = rf(ctx, customerID, deleteEventsReq, deletedBy)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteBigQueryDataHard provides a mock function with given fields: ctx, customerID, softDeleteIntervalDays
func (_m *DataHubBigQuery) DeleteBigQueryDataHard(ctx context.Context, customerID string, softDeleteIntervalDays int) error {
	ret := _m.Called(ctx, customerID, softDeleteIntervalDays)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, int) error); ok {
		r0 = rf(ctx, customerID, softDeleteIntervalDays)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetCustomerDatasetBatches provides a mock function with given fields: ctx, customerID, datasetName
func (_m *DataHubBigQuery) GetCustomerDatasetBatches(ctx context.Context, customerID string, datasetName string) ([]domain.DatasetBatch, error) {
	ret := _m.Called(ctx, customerID, datasetName)

	var r0 []domain.DatasetBatch
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) ([]domain.DatasetBatch, error)); ok {
		return rf(ctx, customerID, datasetName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) []domain.DatasetBatch); ok {
		r0 = rf(ctx, customerID, datasetName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]domain.DatasetBatch)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, customerID, datasetName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomerDatasets provides a mock function with given fields: ctx, customerID
func (_m *DataHubBigQuery) GetCustomerDatasets(ctx context.Context, customerID string) ([]domain.CachedDataset, error) {
	ret := _m.Called(ctx, customerID)

	var r0 []domain.CachedDataset
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]domain.CachedDataset, error)); ok {
		return rf(ctx, customerID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []domain.CachedDataset); ok {
		r0 = rf(ctx, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]domain.CachedDataset)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomersWithSoftDeleteData provides a mock function with given fields: ctx, softDeleteIntervalDays
func (_m *DataHubBigQuery) GetCustomersWithSoftDeleteData(ctx context.Context, softDeleteIntervalDays int) ([]string, error) {
	ret := _m.Called(ctx, softDeleteIntervalDays)

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int) ([]string, error)); ok {
		return rf(ctx, softDeleteIntervalDays)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int) []string); ok {
		r0 = rf(ctx, softDeleteIntervalDays)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, int) error); ok {
		r1 = rf(ctx, softDeleteIntervalDays)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewDataHubBigQuery creates a new instance of DataHubBigQuery. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDataHubBigQuery(t interface {
	mock.TestingT
	Cleanup(func())
}) *DataHubBigQuery {
	mock := &DataHubBigQuery{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
