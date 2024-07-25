// Code generated by mockery v2.13.1. DO NOT EDIT.

package accounts

import (
	context "context"
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// MockService is an autogenerated mock type for the Service type
type MockService struct {
	mock.Mock
}

// GetOldestJoinTimestampAge provides a mock function with given fields: ctx, ids, now
func (_m *MockService) GetOldestJoinTimestampAge(ctx context.Context, ids []string, now time.Time) (int, error) {
	ret := _m.Called(ctx, ids, now)

	var r0 int
	if rf, ok := ret.Get(0).(func(context.Context, []string, time.Time) int); ok {
		r0 = rf(ctx, ids, now)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, []string, time.Time) error); ok {
		r1 = rf(ctx, ids, now)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMockService interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockService creates a new instance of MockService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockService(t mockConstructorTestingTNewMockService) *MockService {
	mock := &MockService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
