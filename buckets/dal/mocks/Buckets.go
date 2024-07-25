// Code generated by mockery v2.26.1. DO NOT EDIT.

package mocks

import (
	context "context"

	common "github.com/doitintl/hello/scheduled-tasks/common"

	firestore "cloud.google.com/go/firestore"

	mock "github.com/stretchr/testify/mock"
)

// Buckets is an autogenerated mock type for the Buckets type
type Buckets struct {
	mock.Mock
}

// GetBucket provides a mock function with given fields: ctx, entityID, bucketID
func (_m *Buckets) GetBucket(ctx context.Context, entityID string, bucketID string) (*common.Bucket, error) {
	ret := _m.Called(ctx, entityID, bucketID)

	var r0 *common.Bucket

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*common.Bucket, error)); ok {
		return rf(ctx, entityID, bucketID)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string, string) *common.Bucket); ok {
		r0 = rf(ctx, entityID, bucketID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*common.Bucket)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, entityID, bucketID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBuckets provides a mock function with given fields: ctx, entityID
func (_m *Buckets) GetBuckets(ctx context.Context, entityID string) ([]common.Bucket, error) {
	ret := _m.Called(ctx, entityID)

	var r0 []common.Bucket

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string) ([]common.Bucket, error)); ok {
		return rf(ctx, entityID)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string) []common.Bucket); ok {
		r0 = rf(ctx, entityID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]common.Bucket)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, entityID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateBucket provides a mock function with given fields: ctx, entityID, bucketID, updates
func (_m *Buckets) UpdateBucket(ctx context.Context, entityID string, bucketID string, updates []firestore.Update) error {
	ret := _m.Called(ctx, entityID, bucketID, updates)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, []firestore.Update) error); ok {
		r0 = rf(ctx, entityID, bucketID, updates)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewBuckets interface {
	mock.TestingT
	Cleanup(func())
}

// NewBuckets creates a new instance of Buckets. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewBuckets(t mockConstructorTestingTNewBuckets) *Buckets {
	mock := &Buckets{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}