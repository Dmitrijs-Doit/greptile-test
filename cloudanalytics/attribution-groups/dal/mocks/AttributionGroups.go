// Code generated by mockery v2.35.2. DO NOT EDIT.

package mocks

import (
	attributiongroups "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attribution "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"

	collab "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"

	context "context"

	firestore "cloud.google.com/go/firestore"

	mock "github.com/stretchr/testify/mock"
)

// AttributionGroups is an autogenerated mock type for the AttributionGroups type
type AttributionGroups struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, attributionGroup
func (_m *AttributionGroups) Create(ctx context.Context, attributionGroup *attributiongroups.AttributionGroup) (string, error) {
	ret := _m.Called(ctx, attributionGroup)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *attributiongroups.AttributionGroup) (string, error)); ok {
		return rf(ctx, attributionGroup)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *attributiongroups.AttributionGroup) string); ok {
		r0 = rf(ctx, attributionGroup)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *attributiongroups.AttributionGroup) error); ok {
		r1 = rf(ctx, attributionGroup)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete provides a mock function with given fields: ctx, id
func (_m *AttributionGroups) Delete(ctx context.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Get provides a mock function with given fields: ctx, id
func (_m *AttributionGroups) Get(ctx context.Context, id string) (*attributiongroups.AttributionGroup, error) {
	ret := _m.Called(ctx, id)

	var r0 *attributiongroups.AttributionGroup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*attributiongroups.AttributionGroup, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *attributiongroups.AttributionGroup); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*attributiongroups.AttributionGroup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAll provides a mock function with given fields: ctx, attributionGroupsRefs
func (_m *AttributionGroups) GetAll(ctx context.Context, attributionGroupsRefs []*firestore.DocumentRef) ([]*attributiongroups.AttributionGroup, error) {
	ret := _m.Called(ctx, attributionGroupsRefs)

	var r0 []*attributiongroups.AttributionGroup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []*firestore.DocumentRef) ([]*attributiongroups.AttributionGroup, error)); ok {
		return rf(ctx, attributionGroupsRefs)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []*firestore.DocumentRef) []*attributiongroups.AttributionGroup); ok {
		r0 = rf(ctx, attributionGroupsRefs)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*attributiongroups.AttributionGroup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []*firestore.DocumentRef) error); ok {
		r1 = rf(ctx, attributionGroupsRefs)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByCustomer provides a mock function with given fields: ctx, customerRef, attrRef
func (_m *AttributionGroups) GetByCustomer(ctx context.Context, customerRef *firestore.DocumentRef, attrRef *firestore.DocumentRef) ([]*attributiongroups.AttributionGroup, error) {
	ret := _m.Called(ctx, customerRef, attrRef)

	var r0 []*attributiongroups.AttributionGroup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, *firestore.DocumentRef) ([]*attributiongroups.AttributionGroup, error)); ok {
		return rf(ctx, customerRef, attrRef)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, *firestore.DocumentRef) []*attributiongroups.AttributionGroup); ok {
		r0 = rf(ctx, customerRef, attrRef)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*attributiongroups.AttributionGroup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *firestore.DocumentRef, *firestore.DocumentRef) error); ok {
		r1 = rf(ctx, customerRef, attrRef)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByName provides a mock function with given fields: ctx, customerRef, name
func (_m *AttributionGroups) GetByName(ctx context.Context, customerRef *firestore.DocumentRef, name string) (*attributiongroups.AttributionGroup, error) {
	ret := _m.Called(ctx, customerRef, name)

	var r0 *attributiongroups.AttributionGroup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, string) (*attributiongroups.AttributionGroup, error)); ok {
		return rf(ctx, customerRef, name)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, string) *attributiongroups.AttributionGroup); ok {
		r0 = rf(ctx, customerRef, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*attributiongroups.AttributionGroup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *firestore.DocumentRef, string) error); ok {
		r1 = rf(ctx, customerRef, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByType provides a mock function with given fields: ctx, customerRef, attrGroupType
func (_m *AttributionGroups) GetByType(ctx context.Context, customerRef *firestore.DocumentRef, attrGroupType attribution.ObjectType) ([]*attributiongroups.AttributionGroup, error) {
	ret := _m.Called(ctx, customerRef, attrGroupType)

	var r0 []*attributiongroups.AttributionGroup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, attribution.ObjectType) ([]*attributiongroups.AttributionGroup, error)); ok {
		return rf(ctx, customerRef, attrGroupType)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, attribution.ObjectType) []*attributiongroups.AttributionGroup); ok {
		r0 = rf(ctx, customerRef, attrGroupType)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*attributiongroups.AttributionGroup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *firestore.DocumentRef, attribution.ObjectType) error); ok {
		r1 = rf(ctx, customerRef, attrGroupType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRef provides a mock function with given fields: ctx, attributionGroupID
func (_m *AttributionGroups) GetRef(ctx context.Context, attributionGroupID string) *firestore.DocumentRef {
	ret := _m.Called(ctx, attributionGroupID)

	var r0 *firestore.DocumentRef
	if rf, ok := ret.Get(0).(func(context.Context, string) *firestore.DocumentRef); ok {
		r0 = rf(ctx, attributionGroupID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*firestore.DocumentRef)
		}
	}

	return r0
}

// List provides a mock function with given fields: ctx, customerRef, email
func (_m *AttributionGroups) List(ctx context.Context, customerRef *firestore.DocumentRef, email string) ([]attributiongroups.AttributionGroup, error) {
	ret := _m.Called(ctx, customerRef, email)

	var r0 []attributiongroups.AttributionGroup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, string) ([]attributiongroups.AttributionGroup, error)); ok {
		return rf(ctx, customerRef, email)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, string) []attributiongroups.AttributionGroup); ok {
		r0 = rf(ctx, customerRef, email)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]attributiongroups.AttributionGroup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *firestore.DocumentRef, string) error); ok {
		r1 = rf(ctx, customerRef, email)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Share provides a mock function with given fields: ctx, attributionGroupID, collaborators, public
func (_m *AttributionGroups) Share(ctx context.Context, attributionGroupID string, collaborators []collab.Collaborator, public *collab.PublicAccess) error {
	ret := _m.Called(ctx, attributionGroupID, collaborators, public)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []collab.Collaborator, *collab.PublicAccess) error); ok {
		r0 = rf(ctx, attributionGroupID, collaborators, public)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Update provides a mock function with given fields: ctx, id, attributionGroup
func (_m *AttributionGroups) Update(ctx context.Context, id string, attributionGroup *attributiongroups.AttributionGroup) error {
	ret := _m.Called(ctx, id, attributionGroup)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *attributiongroups.AttributionGroup) error); ok {
		r0 = rf(ctx, id, attributionGroup)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewAttributionGroups creates a new instance of AttributionGroups. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAttributionGroups(t interface {
	mock.TestingT
	Cleanup(func())
}) *AttributionGroups {
	mock := &AttributionGroups{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
