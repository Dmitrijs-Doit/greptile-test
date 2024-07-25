// Code generated by mockery v2.32.0. DO NOT EDIT.

package mocks

import (
	context "context"

	firestore "cloud.google.com/go/firestore"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"

	mock "github.com/stretchr/testify/mock"
)

// ICSPFirestore is an autogenerated mock type for the ICSPFirestore type
type ICSPFirestore struct {
	mock.Mock
}

// AddRemoveToCopiedTables provides a mock function with given fields: ctx, add, idx, done, data
func (_m *ICSPFirestore) AddRemoveToCopiedTables(ctx context.Context, add bool, idx int, done bool, data *domain.CSPBillingAccountUpdateData) (int, error) {
	ret := _m.Called(ctx, add, idx, done, data)

	var r0 int

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, bool, int, bool, *domain.CSPBillingAccountUpdateData) (int, error)); ok {
		return rf(ctx, add, idx, done, data)
	}

	if rf, ok := ret.Get(0).(func(context.Context, bool, int, bool, *domain.CSPBillingAccountUpdateData) int); ok {
		r0 = rf(ctx, add, idx, done, data)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, bool, int, bool, *domain.CSPBillingAccountUpdateData) error); ok {
		r1 = rf(ctx, add, idx, done, data)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DecStillRunning provides a mock function with given fields: ctx, data
func (_m *ICSPFirestore) DecStillRunning(ctx context.Context, data *domain.CSPBillingAccountUpdateData) (int, error) {
	ret := _m.Called(ctx, data)

	var r0 int

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, *domain.CSPBillingAccountUpdateData) (int, error)); ok {
		return rf(ctx, data)
	}

	if rf, ok := ret.Get(0).(func(context.Context, *domain.CSPBillingAccountUpdateData) int); ok {
		r0 = rf(ctx, data)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *domain.CSPBillingAccountUpdateData) error); ok {
		r1 = rf(ctx, data)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetActiveStandaloneAccounts provides a mock function with given fields: ctx
func (_m *ICSPFirestore) GetActiveStandaloneAccounts(ctx context.Context) (map[string]map[string]interface{}, error) {
	ret := _m.Called(ctx)

	var r0 map[string]map[string]interface{}

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context) (map[string]map[string]interface{}, error)); ok {
		return rf(ctx)
	}

	if rf, ok := ret.Get(0).(func(context.Context) map[string]map[string]interface{}); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]map[string]interface{})
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAssetsForTask provides a mock function with given fields: ctx, params
func (_m *ICSPFirestore) GetAssetsForTask(ctx context.Context, params *domain.UpdateCspTaskParams) ([]*firestore.DocumentSnapshot, error) {
	ret := _m.Called(ctx, params)

	var r0 []*firestore.DocumentSnapshot

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, *domain.UpdateCspTaskParams) ([]*firestore.DocumentSnapshot, error)); ok {
		return rf(ctx, params)
	}

	if rf, ok := ret.Get(0).(func(context.Context, *domain.UpdateCspTaskParams) []*firestore.DocumentSnapshot); ok {
		r0 = rf(ctx, params)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*firestore.DocumentSnapshot)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *domain.UpdateCspTaskParams) error); ok {
		r1 = rf(ctx, params)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetFirestoreCountersDocRef provides a mock function with given fields: ctx, mode, billingAccountID
func (_m *ICSPFirestore) GetFirestoreCountersDocRef(ctx context.Context, mode domain.CSPMode, billingAccountID string) *firestore.DocumentRef {
	ret := _m.Called(ctx, mode, billingAccountID)

	var r0 *firestore.DocumentRef
	if rf, ok := ret.Get(0).(func(context.Context, domain.CSPMode, string) *firestore.DocumentRef); ok {
		r0 = rf(ctx, mode, billingAccountID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*firestore.DocumentRef)
		}
	}

	return r0
}

// GetFirestoreData provides a mock function with given fields: ctx, data, fsData
func (_m *ICSPFirestore) GetFirestoreData(ctx context.Context, data *domain.CSPBillingAccountUpdateData, fsData *domain.CSPFirestoreData) error {
	ret := _m.Called(ctx, data, fsData)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *domain.CSPBillingAccountUpdateData, *domain.CSPFirestoreData) error); ok {
		r0 = rf(ctx, data, fsData)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetOrIncTableIndex provides a mock function with given fields: ctx, curIdx, data
func (_m *ICSPFirestore) GetOrIncTableIndex(ctx context.Context, curIdx int, data *domain.CSPBillingAccountUpdateData) (string, int, error) {
	ret := _m.Called(ctx, curIdx, data)

	var r0 string

	var r1 int

	var r2 error

	if rf, ok := ret.Get(0).(func(context.Context, int, *domain.CSPBillingAccountUpdateData) (string, int, error)); ok {
		return rf(ctx, curIdx, data)
	}

	if rf, ok := ret.Get(0).(func(context.Context, int, *domain.CSPBillingAccountUpdateData) string); ok {
		r0 = rf(ctx, curIdx, data)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, int, *domain.CSPBillingAccountUpdateData) int); ok {
		r1 = rf(ctx, curIdx, data)
	} else {
		r1 = ret.Get(1).(int)
	}

	if rf, ok := ret.Get(2).(func(context.Context, int, *domain.CSPBillingAccountUpdateData) error); ok {
		r2 = rf(ctx, curIdx, data)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// SetDataCopied provides a mock function with given fields: ctx, data
func (_m *ICSPFirestore) SetDataCopied(ctx context.Context, data *domain.CSPBillingAccountUpdateData) (bool, error) {
	ret := _m.Called(ctx, data)

	var r0 bool

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, *domain.CSPBillingAccountUpdateData) (bool, error)); ok {
		return rf(ctx, data)
	}

	if rf, ok := ret.Get(0).(func(context.Context, *domain.CSPBillingAccountUpdateData) bool); ok {
		r0 = rf(ctx, data)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *domain.CSPBillingAccountUpdateData) error); ok {
		r1 = rf(ctx, data)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetTaskState provides a mock function with given fields: ctx, state, data
func (_m *ICSPFirestore) SetTaskState(ctx context.Context, state domain.TaskState, data *domain.CSPBillingAccountUpdateData) (int, error) {
	ret := _m.Called(ctx, state, data)

	var r0 int

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, domain.TaskState, *domain.CSPBillingAccountUpdateData) (int, error)); ok {
		return rf(ctx, state, data)
	}

	if rf, ok := ret.Get(0).(func(context.Context, domain.TaskState, *domain.CSPBillingAccountUpdateData) int); ok {
		r0 = rf(ctx, state, data)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, domain.TaskState, *domain.CSPBillingAccountUpdateData) error); ok {
		r1 = rf(ctx, state, data)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewICSPFirestore creates a new instance of ICSPFirestore. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewICSPFirestore(t interface {
	mock.TestingT
	Cleanup(func())
}) *ICSPFirestore {
	mock := &ICSPFirestore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}