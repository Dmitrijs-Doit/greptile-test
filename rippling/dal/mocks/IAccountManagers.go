// Code generated by mockery v2.30.1. DO NOT EDIT.

package mocks

import (
	context "context"

	firestore "cloud.google.com/go/firestore"

	mock "github.com/stretchr/testify/mock"

	pkg "github.com/doitintl/rippling/pkg"

	ripplingpkg "github.com/doitintl/hello/scheduled-tasks/rippling/pkg"
)

// IAccountManagers is an autogenerated mock type for the IAccountManagers type
type IAccountManagers struct {
	mock.Mock
}

// AddNew provides a mock function with given fields: ctx, amFromRippling
func (_m *IAccountManagers) AddNew(ctx context.Context, amFromRippling *pkg.Employee) (*firestore.DocumentRef, error) {
	ret := _m.Called(ctx, amFromRippling)

	var r0 *firestore.DocumentRef

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, *pkg.Employee) (*firestore.DocumentRef, error)); ok {
		return rf(ctx, amFromRippling)
	}

	if rf, ok := ret.Get(0).(func(context.Context, *pkg.Employee) *firestore.DocumentRef); ok {
		r0 = rf(ctx, amFromRippling)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*firestore.DocumentRef)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *pkg.Employee) error); ok {
		r1 = rf(ctx, amFromRippling)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// BackfillUnfamiliarDepartments provides a mock function with given fields: ctx, employeesRippling, amRipplingMap
func (_m *IAccountManagers) BackfillUnfamiliarDepartments(ctx context.Context, employeesRippling []*pkg.Employee, amRipplingMap ripplingpkg.AccountManagersMap) (ripplingpkg.AccountManagersMap, error) {
	ret := _m.Called(ctx, employeesRippling, amRipplingMap)

	var r0 ripplingpkg.AccountManagersMap

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, []*pkg.Employee, ripplingpkg.AccountManagersMap) (ripplingpkg.AccountManagersMap, error)); ok {
		return rf(ctx, employeesRippling, amRipplingMap)
	}

	if rf, ok := ret.Get(0).(func(context.Context, []*pkg.Employee, ripplingpkg.AccountManagersMap) ripplingpkg.AccountManagersMap); ok {
		r0 = rf(ctx, employeesRippling, amRipplingMap)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ripplingpkg.AccountManagersMap)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []*pkg.Employee, ripplingpkg.AccountManagersMap) error); ok {
		r1 = rf(ctx, employeesRippling, amRipplingMap)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetOrAdd provides a mock function with given fields: ctx, amFromRippling
func (_m *IAccountManagers) GetOrAdd(ctx context.Context, amFromRippling *pkg.Employee) (*firestore.DocumentRef, error) {
	ret := _m.Called(ctx, amFromRippling)

	var r0 *firestore.DocumentRef

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, *pkg.Employee) (*firestore.DocumentRef, error)); ok {
		return rf(ctx, amFromRippling)
	}

	if rf, ok := ret.Get(0).(func(context.Context, *pkg.Employee) *firestore.DocumentRef); ok {
		r0 = rf(ctx, amFromRippling)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*firestore.DocumentRef)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *pkg.Employee) error); ok {
		r1 = rf(ctx, amFromRippling)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRipplingDepartmentToCMPRoleMap provides a mock function with given fields: ctx
func (_m *IAccountManagers) GetRipplingDepartmentToCMPRoleMap(ctx context.Context) (ripplingpkg.RipplingDepartmentToCMPRoleMap, error) {
	ret := _m.Called(ctx)

	var r0 ripplingpkg.RipplingDepartmentToCMPRoleMap

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context) (ripplingpkg.RipplingDepartmentToCMPRoleMap, error)); ok {
		return rf(ctx)
	}

	if rf, ok := ret.Get(0).(func(context.Context) ripplingpkg.RipplingDepartmentToCMPRoleMap); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ripplingpkg.RipplingDepartmentToCMPRoleMap)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateAM provides a mock function with given fields: ctx, amRippling, amRipplingMap
func (_m *IAccountManagers) UpdateAM(ctx context.Context, amRippling *pkg.Employee, amRipplingMap ripplingpkg.AccountManagersMap) error {
	ret := _m.Called(ctx, amRippling, amRipplingMap)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *pkg.Employee, ripplingpkg.AccountManagersMap) error); ok {
		r0 = rf(ctx, amRippling, amRipplingMap)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateStatus provides a mock function with given fields: ctx, amRef, status
func (_m *IAccountManagers) UpdateStatus(ctx context.Context, amRef *firestore.DocumentRef, status pkg.RoleState) error {
	ret := _m.Called(ctx, amRef, status)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, pkg.RoleState) error); ok {
		r0 = rf(ctx, amRef, status)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIAccountManagers creates a new instance of IAccountManagers. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewIAccountManagers(t interface {
	mock.TestingT
	Cleanup(func())
}) *IAccountManagers {
	mock := &IAccountManagers{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
