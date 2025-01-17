// Code generated by mockery v2.38.0. DO NOT EDIT.

package mocks

import (
	context "context"

	common "github.com/doitintl/hello/scheduled-tasks/common"

	domain "github.com/doitintl/hello/scheduled-tasks/user/domain"

	firestore "cloud.google.com/go/firestore"

	mock "github.com/stretchr/testify/mock"

	pkg "github.com/doitintl/firestore/pkg"

	time "time"
)

// IUserFirestoreDAL is an autogenerated mock type for the IUserFirestoreDAL type
type IUserFirestoreDAL struct {
	mock.Mock
}

// ClearUserNotifications provides a mock function with given fields: ctx, user
func (_m *IUserFirestoreDAL) ClearUserNotifications(ctx context.Context, user *pkg.User) error {
	ret := _m.Called(ctx, user)

	if len(ret) == 0 {
		panic("no return value specified for ClearUserNotifications")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *pkg.User) error); ok {
		r0 = rf(ctx, user)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Get provides a mock function with given fields: ctx, id
func (_m *IUserFirestoreDAL) Get(ctx context.Context, id string) (*common.User, error) {
	ret := _m.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 *common.User

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string) (*common.User, error)); ok {
		return rf(ctx, id)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string) *common.User); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*common.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomerUsersByRoles provides a mock function with given fields: ctx, customerID, roles
func (_m *IUserFirestoreDAL) GetCustomerUsersByRoles(ctx context.Context, customerID string, roles []common.PresetRole) ([]*common.User, error) {
	ret := _m.Called(ctx, customerID, roles)

	if len(ret) == 0 {
		panic("no return value specified for GetCustomerUsersByRoles")
	}

	var r0 []*common.User

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string, []common.PresetRole) ([]*common.User, error)); ok {
		return rf(ctx, customerID, roles)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string, []common.PresetRole) []*common.User); ok {
		r0 = rf(ctx, customerID, roles)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*common.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, []common.PresetRole) error); ok {
		r1 = rf(ctx, customerID, roles)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomerUsersWithInvoiceNotification provides a mock function with given fields: ctx, customerID, entityID
func (_m *IUserFirestoreDAL) GetCustomerUsersWithInvoiceNotification(ctx context.Context, customerID string, entityID string) ([]*common.User, error) {
	ret := _m.Called(ctx, customerID, entityID)

	if len(ret) == 0 {
		panic("no return value specified for GetCustomerUsersWithInvoiceNotification")
	}

	var r0 []*common.User

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string, string) ([]*common.User, error)); ok {
		return rf(ctx, customerID, entityID)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string, string) []*common.User); ok {
		r0 = rf(ctx, customerID, entityID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*common.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, customerID, entityID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomerUsersWithNotifications provides a mock function with given fields: ctx, customerID, isRestore
func (_m *IUserFirestoreDAL) GetCustomerUsersWithNotifications(ctx context.Context, customerID string, isRestore bool) ([]*pkg.User, error) {
	ret := _m.Called(ctx, customerID, isRestore)

	if len(ret) == 0 {
		panic("no return value specified for GetCustomerUsersWithNotifications")
	}

	var r0 []*pkg.User

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string, bool) ([]*pkg.User, error)); ok {
		return rf(ctx, customerID, isRestore)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string, bool) []*pkg.User); ok {
		r0 = rf(ctx, customerID, isRestore)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pkg.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, bool) error); ok {
		r1 = rf(ctx, customerID, isRestore)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLastUserEngagementTimeForCustomer provides a mock function with given fields: ctx, customerID
func (_m *IUserFirestoreDAL) GetLastUserEngagementTimeForCustomer(ctx context.Context, customerID string) (*time.Time, error) {
	ret := _m.Called(ctx, customerID)

	if len(ret) == 0 {
		panic("no return value specified for GetLastUserEngagementTimeForCustomer")
	}

	var r0 *time.Time

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string) (*time.Time, error)); ok {
		return rf(ctx, customerID)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string) *time.Time); ok {
		r0 = rf(ctx, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*time.Time)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRef provides a mock function with given fields: ctx, ID
func (_m *IUserFirestoreDAL) GetRef(ctx context.Context, ID string) *firestore.DocumentRef {
	ret := _m.Called(ctx, ID)

	if len(ret) == 0 {
		panic("no return value specified for GetRef")
	}

	var r0 *firestore.DocumentRef
	if rf, ok := ret.Get(0).(func(context.Context, string) *firestore.DocumentRef); ok {
		r0 = rf(ctx, ID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*firestore.DocumentRef)
		}
	}

	return r0
}

// GetUserByEmail provides a mock function with given fields: ctx, email, customerID
func (_m *IUserFirestoreDAL) GetUserByEmail(ctx context.Context, email string, customerID string) (*domain.User, error) {
	ret := _m.Called(ctx, email, customerID)

	if len(ret) == 0 {
		panic("no return value specified for GetUserByEmail")
	}

	var r0 *domain.User

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*domain.User, error)); ok {
		return rf(ctx, email, customerID)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string, string) *domain.User); ok {
		r0 = rf(ctx, email, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, email, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUsersWithRecentEngagement provides a mock function with given fields: ctx
func (_m *IUserFirestoreDAL) GetUsersWithRecentEngagement(ctx context.Context) ([]common.User, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GetUsersWithRecentEngagement")
	}

	var r0 []common.User

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context) ([]common.User, error)); ok {
		return rf(ctx)
	}

	if rf, ok := ret.Get(0).(func(context.Context) []common.User); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]common.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListUsers provides a mock function with given fields: ctx, customerRef, limit
func (_m *IUserFirestoreDAL) ListUsers(ctx context.Context, customerRef *firestore.DocumentRef, limit int) ([]*domain.User, error) {
	ret := _m.Called(ctx, customerRef, limit)

	if len(ret) == 0 {
		panic("no return value specified for ListUsers")
	}

	var r0 []*domain.User

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, int) ([]*domain.User, error)); ok {
		return rf(ctx, customerRef, limit)
	}

	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, int) []*domain.User); ok {
		r0 = rf(ctx, customerRef, limit)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*domain.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *firestore.DocumentRef, int) error); ok {
		r1 = rf(ctx, customerRef, limit)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RestoreUserNotifications provides a mock function with given fields: ctx, user
func (_m *IUserFirestoreDAL) RestoreUserNotifications(ctx context.Context, user *pkg.User) error {
	ret := _m.Called(ctx, user)

	if len(ret) == 0 {
		panic("no return value specified for RestoreUserNotifications")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *pkg.User) error); ok {
		r0 = rf(ctx, user)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIUserFirestoreDAL creates a new instance of IUserFirestoreDAL. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewIUserFirestoreDAL(t interface {
	mock.TestingT
	Cleanup(func())
}) *IUserFirestoreDAL {
	mock := &IUserFirestoreDAL{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
