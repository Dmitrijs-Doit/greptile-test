// Code generated by mockery v2.16.0. DO NOT EDIT.

package mocks

import (
	context "context"

	dataStructures "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// Metadata is an autogenerated mock type for the Metadata type
type Metadata struct {
	mock.Mock
}

// CatchExternalManagerMetadata provides a mock function with given fields: ctx
func (_m *Metadata) CatchExternalManagerMetadata(ctx context.Context) (*dataStructures.ExternalManagerMetadata, error) {
	ret := _m.Called(ctx)

	var r0 *dataStructures.ExternalManagerMetadata
	if rf, ok := ret.Get(0).(func(context.Context) *dataStructures.ExternalManagerMetadata); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dataStructures.ExternalManagerMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CatchInternalManagerMetadata provides a mock function with given fields: ctx
func (_m *Metadata) CatchInternalManagerMetadata(ctx context.Context) (*dataStructures.InternalManagerMetadata, error) {
	ret := _m.Called(ctx)

	var r0 *dataStructures.InternalManagerMetadata
	if rf, ok := ret.Get(0).(func(context.Context) *dataStructures.InternalManagerMetadata); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dataStructures.InternalManagerMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateAllMetadata provides a mock function with given fields: ctx, rawBillingTargetTime, rawBillingOldestTime
func (_m *Metadata) CreateAllMetadata(ctx context.Context, rawBillingTargetTime *time.Time, rawBillingOldestTime *time.Time) error {
	ret := _m.Called(ctx, rawBillingTargetTime, rawBillingOldestTime)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *time.Time, *time.Time) error); ok {
		r0 = rf(ctx, rawBillingTargetTime, rawBillingOldestTime)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateMetadataForNewBillingID provides a mock function with given fields: ctx, requestParams, location, historyCopyTargetTime, oldestRecordTime
func (_m *Metadata) CreateMetadataForNewBillingID(ctx context.Context, requestParams *dataStructures.OnboardingRequestBody, location string, historyCopyTargetTime *time.Time, oldestRecordTime *time.Time) error {
	ret := _m.Called(ctx, requestParams, location, historyCopyTargetTime, oldestRecordTime)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *dataStructures.OnboardingRequestBody, string, *time.Time, *time.Time) error); ok {
		r0 = rf(ctx, requestParams, location, historyCopyTargetTime, oldestRecordTime)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteExternalManagerMetadata provides a mock function with given fields: ctx
func (_m *Metadata) DeleteExternalManagerMetadata(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteExternalTaskMetadata provides a mock function with given fields: ctx, billingID
func (_m *Metadata) DeleteExternalTaskMetadata(ctx context.Context, billingID string) error {
	ret := _m.Called(ctx, billingID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, billingID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteExternalTasksMetadata provides a mock function with given fields: ctx
func (_m *Metadata) DeleteExternalTasksMetadata(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteInternalManagerMetadata provides a mock function with given fields: ctx
func (_m *Metadata) DeleteInternalManagerMetadata(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteInternalTaskMetadata provides a mock function with given fields: ctx, billingID
func (_m *Metadata) DeleteInternalTaskMetadata(ctx context.Context, billingID string) error {
	ret := _m.Called(ctx, billingID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, billingID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteInternalTasksMetadata provides a mock function with given fields: ctx
func (_m *Metadata) DeleteInternalTasksMetadata(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteValidatorMetadata provides a mock function with given fields: ctx
func (_m *Metadata) DeleteValidatorMetadata(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetActiveExternalTasksMetadata provides a mock function with given fields: ctx
func (_m *Metadata) GetActiveExternalTasksMetadata(ctx context.Context) ([]*dataStructures.ExternalTaskMetadata, error) {
	ret := _m.Called(ctx)

	var r0 []*dataStructures.ExternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context) []*dataStructures.ExternalTaskMetadata); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dataStructures.ExternalTaskMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetActiveInternalTasksMetadata provides a mock function with given fields: ctx
func (_m *Metadata) GetActiveInternalTasksMetadata(ctx context.Context) ([]*dataStructures.InternalTaskMetadata, error) {
	ret := _m.Called(ctx)

	var r0 []*dataStructures.InternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context) []*dataStructures.InternalTaskMetadata); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dataStructures.InternalTaskMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAllInternalTasksMetadataByParams provides a mock function with given fields: ctx, iteration, possibleStates
func (_m *Metadata) GetAllInternalTasksMetadataByParams(ctx context.Context, iteration int64, possibleStates []dataStructures.InternalTaskState) ([]*dataStructures.InternalTaskMetadata, error) {
	ret := _m.Called(ctx, iteration, possibleStates)

	var r0 []*dataStructures.InternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context, int64, []dataStructures.InternalTaskState) []*dataStructures.InternalTaskMetadata); ok {
		r0 = rf(ctx, iteration, possibleStates)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dataStructures.InternalTaskMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int64, []dataStructures.InternalTaskState) error); ok {
		r1 = rf(ctx, iteration, possibleStates)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCreatedExternalTasksMetadata provides a mock function with given fields: ctx
func (_m *Metadata) GetCreatedExternalTasksMetadata(ctx context.Context) ([]*dataStructures.ExternalTaskMetadata, error) {
	ret := _m.Called(ctx)

	var r0 []*dataStructures.ExternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context) []*dataStructures.ExternalTaskMetadata); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dataStructures.ExternalTaskMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCreatedInternalTasksMetadata provides a mock function with given fields: ctx
func (_m *Metadata) GetCreatedInternalTasksMetadata(ctx context.Context) ([]*dataStructures.InternalTaskMetadata, error) {
	ret := _m.Called(ctx)

	var r0 []*dataStructures.InternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context) []*dataStructures.InternalTaskMetadata); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dataStructures.InternalTaskMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDeprecatedExternalTasksMetadata provides a mock function with given fields: ctx
func (_m *Metadata) GetDeprecatedExternalTasksMetadata(ctx context.Context) ([]*dataStructures.ExternalTaskMetadata, error) {
	ret := _m.Called(ctx)

	var r0 []*dataStructures.ExternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context) []*dataStructures.ExternalTaskMetadata); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dataStructures.ExternalTaskMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDeprecatedInternalTasksMetadata provides a mock function with given fields: ctx
func (_m *Metadata) GetDeprecatedInternalTasksMetadata(ctx context.Context) ([]*dataStructures.InternalTaskMetadata, error) {
	ret := _m.Called(ctx)

	var r0 []*dataStructures.InternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context) []*dataStructures.InternalTaskMetadata); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dataStructures.InternalTaskMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetExternalTaskMetadata provides a mock function with given fields: ctx, billingID
func (_m *Metadata) GetExternalTaskMetadata(ctx context.Context, billingID string) (*dataStructures.ExternalTaskMetadata, error) {
	ret := _m.Called(ctx, billingID)

	var r0 *dataStructures.ExternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context, string) *dataStructures.ExternalTaskMetadata); ok {
		r0 = rf(ctx, billingID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dataStructures.ExternalTaskMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, billingID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetExternalTasksMetadata provides a mock function with given fields: ctx
func (_m *Metadata) GetExternalTasksMetadata(ctx context.Context) ([]*dataStructures.ExternalTaskMetadata, error) {
	ret := _m.Called(ctx)

	var r0 []*dataStructures.ExternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context) []*dataStructures.ExternalTaskMetadata); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dataStructures.ExternalTaskMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetInternalManagerMetadata provides a mock function with given fields: ctx
func (_m *Metadata) GetInternalManagerMetadata(ctx context.Context) (*dataStructures.InternalManagerMetadata, error) {
	ret := _m.Called(ctx)

	var r0 *dataStructures.InternalManagerMetadata
	if rf, ok := ret.Get(0).(func(context.Context) *dataStructures.InternalManagerMetadata); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dataStructures.InternalManagerMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetInternalTaskMetadata provides a mock function with given fields: ctx, billingID
func (_m *Metadata) GetInternalTaskMetadata(ctx context.Context, billingID string) (*dataStructures.InternalTaskMetadata, error) {
	ret := _m.Called(ctx, billingID)

	var r0 *dataStructures.InternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context, string) *dataStructures.InternalTaskMetadata); ok {
		r0 = rf(ctx, billingID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dataStructures.InternalTaskMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, billingID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetInternalTasksMetadata provides a mock function with given fields: ctx
func (_m *Metadata) GetInternalTasksMetadata(ctx context.Context) ([]*dataStructures.InternalTaskMetadata, error) {
	ret := _m.Called(ctx)

	var r0 []*dataStructures.InternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context) []*dataStructures.InternalTaskMetadata); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dataStructures.InternalTaskMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRowsValidatorMetadata provides a mock function with given fields: ctx, billingAccoutnID
func (_m *Metadata) GetRowsValidatorMetadata(ctx context.Context, billingAccoutnID string) (*dataStructures.RowsValidatorMetadata, error) {
	ret := _m.Called(ctx, billingAccoutnID)

	var r0 *dataStructures.RowsValidatorMetadata
	if rf, ok := ret.Get(0).(func(context.Context, string) *dataStructures.RowsValidatorMetadata); ok {
		r0 = rf(ctx, billingAccoutnID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dataStructures.RowsValidatorMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, billingAccoutnID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MarkAllCurrentInternalTasksAsFailed provides a mock function with given fields: ctx, iteration
func (_m *Metadata) MarkAllCurrentInternalTasksAsFailed(ctx context.Context, iteration int64) error {
	ret := _m.Called(ctx, iteration)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) error); ok {
		r0 = rf(ctx, iteration)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MarkAllInternalVerifiedTasksAsDone provides a mock function with given fields: ctx
func (_m *Metadata) MarkAllInternalVerifiedTasksAsDone(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetExternalManagerMetadata provides a mock function with given fields: ctx, updateF
func (_m *Metadata) SetExternalManagerMetadata(ctx context.Context, updateF func(context.Context, *dataStructures.ExternalManagerMetadata) error) (*dataStructures.ExternalManagerMetadata, error) {
	ret := _m.Called(ctx, updateF)

	var r0 *dataStructures.ExternalManagerMetadata
	if rf, ok := ret.Get(0).(func(context.Context, func(context.Context, *dataStructures.ExternalManagerMetadata) error) *dataStructures.ExternalManagerMetadata); ok {
		r0 = rf(ctx, updateF)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dataStructures.ExternalManagerMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, func(context.Context, *dataStructures.ExternalManagerMetadata) error) error); ok {
		r1 = rf(ctx, updateF)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetExternalTaskMetadata provides a mock function with given fields: ctx, billingAccount, updateF
func (_m *Metadata) SetExternalTaskMetadata(ctx context.Context, billingAccount string, updateF func(context.Context, *dataStructures.ExternalTaskMetadata) error) (*dataStructures.ExternalTaskMetadata, error) {
	ret := _m.Called(ctx, billingAccount, updateF)

	var r0 *dataStructures.ExternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context, string, func(context.Context, *dataStructures.ExternalTaskMetadata) error) *dataStructures.ExternalTaskMetadata); ok {
		r0 = rf(ctx, billingAccount, updateF)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dataStructures.ExternalTaskMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, func(context.Context, *dataStructures.ExternalTaskMetadata) error) error); ok {
		r1 = rf(ctx, billingAccount, updateF)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetInternalAndExternalTasksMetadata provides a mock function with given fields: ctx, billingAccount, updateFunc
func (_m *Metadata) SetInternalAndExternalTasksMetadata(ctx context.Context, billingAccount string, updateFunc func(context.Context, *dataStructures.ExternalTaskMetadata, *dataStructures.InternalTaskMetadata) error) (*dataStructures.InternalTaskMetadata, *dataStructures.ExternalTaskMetadata, error) {
	ret := _m.Called(ctx, billingAccount, updateFunc)

	var r0 *dataStructures.InternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context, string, func(context.Context, *dataStructures.ExternalTaskMetadata, *dataStructures.InternalTaskMetadata) error) *dataStructures.InternalTaskMetadata); ok {
		r0 = rf(ctx, billingAccount, updateFunc)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dataStructures.InternalTaskMetadata)
		}
	}

	var r1 *dataStructures.ExternalTaskMetadata
	if rf, ok := ret.Get(1).(func(context.Context, string, func(context.Context, *dataStructures.ExternalTaskMetadata, *dataStructures.InternalTaskMetadata) error) *dataStructures.ExternalTaskMetadata); ok {
		r1 = rf(ctx, billingAccount, updateFunc)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*dataStructures.ExternalTaskMetadata)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, string, func(context.Context, *dataStructures.ExternalTaskMetadata, *dataStructures.InternalTaskMetadata) error) error); ok {
		r2 = rf(ctx, billingAccount, updateFunc)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// SetInternalManagerMetadata provides a mock function with given fields: ctx, updateF
func (_m *Metadata) SetInternalManagerMetadata(ctx context.Context, updateF func(context.Context, *dataStructures.InternalManagerMetadata) error) (*dataStructures.InternalManagerMetadata, error) {
	ret := _m.Called(ctx, updateF)

	var r0 *dataStructures.InternalManagerMetadata
	if rf, ok := ret.Get(0).(func(context.Context, func(context.Context, *dataStructures.InternalManagerMetadata) error) *dataStructures.InternalManagerMetadata); ok {
		r0 = rf(ctx, updateF)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dataStructures.InternalManagerMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, func(context.Context, *dataStructures.InternalManagerMetadata) error) error); ok {
		r1 = rf(ctx, updateF)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetInternalTaskMetadata provides a mock function with given fields: ctx, billingAccount, updateFunc
func (_m *Metadata) SetInternalTaskMetadata(ctx context.Context, billingAccount string, updateFunc func(context.Context, *dataStructures.InternalTaskMetadata) error) (*dataStructures.InternalTaskMetadata, error) {
	ret := _m.Called(ctx, billingAccount, updateFunc)

	var r0 *dataStructures.InternalTaskMetadata
	if rf, ok := ret.Get(0).(func(context.Context, string, func(context.Context, *dataStructures.InternalTaskMetadata) error) *dataStructures.InternalTaskMetadata); ok {
		r0 = rf(ctx, billingAccount, updateFunc)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dataStructures.InternalTaskMetadata)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, func(context.Context, *dataStructures.InternalTaskMetadata) error) error); ok {
		r1 = rf(ctx, billingAccount, updateFunc)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetInternalTasksMetadata provides a mock function with given fields: ctx, updateFunc
func (_m *Metadata) SetInternalTasksMetadata(ctx context.Context, updateFunc func(context.Context, *dataStructures.InternalTaskMetadata) error) error {
	ret := _m.Called(ctx, updateFunc)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, func(context.Context, *dataStructures.InternalTaskMetadata) error) error); ok {
		r0 = rf(ctx, updateFunc)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetRowsValidatorMetadata provides a mock function with given fields: ctx, billingAccoutnID, md
func (_m *Metadata) SetRowsValidatorMetadata(ctx context.Context, billingAccoutnID string, md *dataStructures.RowsValidatorMetadata) error {
	ret := _m.Called(ctx, billingAccoutnID, md)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *dataStructures.RowsValidatorMetadata) error); ok {
		r0 = rf(ctx, billingAccoutnID, md)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewMetadata interface {
	mock.TestingT
	Cleanup(func())
}

// NewMetadata creates a new instance of Metadata. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMetadata(t mockConstructorTestingTNewMetadata) *Metadata {
	mock := &Metadata{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
