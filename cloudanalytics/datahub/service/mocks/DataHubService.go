// Code generated by mockery v2.35.2. DO NOT EDIT.

package mocks

import (
	context "context"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
	errormsg "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"

	mock "github.com/stretchr/testify/mock"
)

// DataHubService is an autogenerated mock type for the DataHubService type
type DataHubService struct {
	mock.Mock
}

// AddRawEvents provides a mock function with given fields: ctx, customerID, email, rawEventsReq
func (_m *DataHubService) AddRawEvents(ctx context.Context, customerID string, email string, rawEventsReq domain.RawEventsReq) ([]*domain.Event, []errormsg.ErrorMsg, error) {
	ret := _m.Called(ctx, customerID, email, rawEventsReq)

	var r0 []*domain.Event
	var r1 []errormsg.ErrorMsg
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, domain.RawEventsReq) ([]*domain.Event, []errormsg.ErrorMsg, error)); ok {
		return rf(ctx, customerID, email, rawEventsReq)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, domain.RawEventsReq) []*domain.Event); ok {
		r0 = rf(ctx, customerID, email, rawEventsReq)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*domain.Event)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, domain.RawEventsReq) []errormsg.ErrorMsg); ok {
		r1 = rf(ctx, customerID, email, rawEventsReq)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]errormsg.ErrorMsg)
		}
	}

	if rf, ok := ret.Get(2).(func(context.Context, string, string, domain.RawEventsReq) error); ok {
		r2 = rf(ctx, customerID, email, rawEventsReq)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// CreateDataset provides a mock function with given fields: ctx, customerID, email, datasetReq
func (_m *DataHubService) CreateDataset(ctx context.Context, customerID string, email string, datasetReq domain.CreateDatasetRequest) error {
	ret := _m.Called(ctx, customerID, email, datasetReq)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, domain.CreateDatasetRequest) error); ok {
		r0 = rf(ctx, customerID, email, datasetReq)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteAllCustomersDataHard provides a mock function with given fields: ctx
func (_m *DataHubService) DeleteAllCustomersDataHard(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteCustomerData provides a mock function with given fields: ctx, customerID
func (_m *DataHubService) DeleteCustomerData(ctx context.Context, customerID string) error {
	ret := _m.Called(ctx, customerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteCustomerDataByClouds provides a mock function with given fields: ctx, customerID, deleteRequest, deletedBy
func (_m *DataHubService) DeleteCustomerDataByClouds(ctx context.Context, customerID string, deleteRequest domain.DeleteDatasetsReq, deletedBy string) error {
	ret := _m.Called(ctx, customerID, deleteRequest, deletedBy)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, domain.DeleteDatasetsReq, string) error); ok {
		r0 = rf(ctx, customerID, deleteRequest, deletedBy)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteCustomerDataByEventIDs provides a mock function with given fields: ctx, customerID, deleteEventsReq, deletedBy
func (_m *DataHubService) DeleteCustomerDataByEventIDs(ctx context.Context, customerID string, deleteEventsReq domain.DeleteEventsReq, deletedBy string) error {
	ret := _m.Called(ctx, customerID, deleteEventsReq, deletedBy)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, domain.DeleteEventsReq, string) error); ok {
		r0 = rf(ctx, customerID, deleteEventsReq, deletedBy)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteCustomerDataHard provides a mock function with given fields: ctx, customerID
func (_m *DataHubService) DeleteCustomerDataHard(ctx context.Context, customerID string) error {
	ret := _m.Called(ctx, customerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteDatasetBatches provides a mock function with given fields: ctx, customerID, datasetName, deleteBatchesReq, deletedBy
func (_m *DataHubService) DeleteDatasetBatches(ctx context.Context, customerID string, datasetName string, deleteBatchesReq domain.DeleteBatchesReq, deletedBy string) error {
	ret := _m.Called(ctx, customerID, datasetName, deleteBatchesReq, deletedBy)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, domain.DeleteBatchesReq, string) error); ok {
		r0 = rf(ctx, customerID, datasetName, deleteBatchesReq, deletedBy)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetCustomerDatasetBatches provides a mock function with given fields: ctx, customerID, datasetName, forceRefresh
func (_m *DataHubService) GetCustomerDatasetBatches(ctx context.Context, customerID string, datasetName string, forceRefresh bool) (*domain.DatasetBatchesRes, error) {
	ret := _m.Called(ctx, customerID, datasetName, forceRefresh)

	var r0 *domain.DatasetBatchesRes
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, bool) (*domain.DatasetBatchesRes, error)); ok {
		return rf(ctx, customerID, datasetName, forceRefresh)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, bool) *domain.DatasetBatchesRes); ok {
		r0 = rf(ctx, customerID, datasetName, forceRefresh)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.DatasetBatchesRes)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, bool) error); ok {
		r1 = rf(ctx, customerID, datasetName, forceRefresh)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomerDatasets provides a mock function with given fields: ctx, customerID, forceRefresh
func (_m *DataHubService) GetCustomerDatasets(ctx context.Context, customerID string, forceRefresh bool) (*domain.CachedDatasetsRes, error) {
	ret := _m.Called(ctx, customerID, forceRefresh)

	var r0 *domain.CachedDatasetsRes
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, bool) (*domain.CachedDatasetsRes, error)); ok {
		return rf(ctx, customerID, forceRefresh)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, bool) *domain.CachedDatasetsRes); ok {
		r0 = rf(ctx, customerID, forceRefresh)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.CachedDatasetsRes)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, bool) error); ok {
		r1 = rf(ctx, customerID, forceRefresh)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewDataHubService creates a new instance of DataHubService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDataHubService(t interface {
	mock.TestingT
	Cleanup(func())
}) *DataHubService {
	mock := &DataHubService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
