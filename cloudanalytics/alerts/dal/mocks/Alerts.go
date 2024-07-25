// Code generated by mockery v2.35.2. DO NOT EDIT.

package mocks

import (
	context "context"

	collab "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"

	firestore "cloud.google.com/go/firestore"

	iface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal/iface"

	mock "github.com/stretchr/testify/mock"
)

// Alerts is an autogenerated mock type for the Alerts type
type Alerts struct {
	mock.Mock
}

// CreateAlert provides a mock function with given fields: ctx, alert
func (_m *Alerts) CreateAlert(ctx context.Context, alert *domain.Alert) (*domain.Alert, error) {
	ret := _m.Called(ctx, alert)

	var r0 *domain.Alert
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *domain.Alert) (*domain.Alert, error)); ok {
		return rf(ctx, alert)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *domain.Alert) *domain.Alert); ok {
		r0 = rf(ctx, alert)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.Alert)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *domain.Alert) error); ok {
		r1 = rf(ctx, alert)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteAlert provides a mock function with given fields: ctx, alertID
func (_m *Alerts) DeleteAlert(ctx context.Context, alertID string) error {
	ret := _m.Called(ctx, alertID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, alertID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetAlert provides a mock function with given fields: ctx, alertID
func (_m *Alerts) GetAlert(ctx context.Context, alertID string) (*domain.Alert, error) {
	ret := _m.Called(ctx, alertID)

	var r0 *domain.Alert
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*domain.Alert, error)); ok {
		return rf(ctx, alertID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *domain.Alert); ok {
		r0 = rf(ctx, alertID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.Alert)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, alertID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAlerts provides a mock function with given fields: ctx
func (_m *Alerts) GetAlerts(ctx context.Context) ([]domain.Alert, error) {
	ret := _m.Called(ctx)

	var r0 []domain.Alert
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]domain.Alert, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []domain.Alert); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]domain.Alert)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAlertsByCustomer provides a mock function with given fields: ctx, args
func (_m *Alerts) GetAlertsByCustomer(ctx context.Context, args *iface.AlertsByCustomerArgs) ([]domain.Alert, error) {
	ret := _m.Called(ctx, args)

	var r0 []domain.Alert
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *iface.AlertsByCustomerArgs) ([]domain.Alert, error)); ok {
		return rf(ctx, args)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *iface.AlertsByCustomerArgs) []domain.Alert); ok {
		r0 = rf(ctx, args)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]domain.Alert)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *iface.AlertsByCustomerArgs) error); ok {
		r1 = rf(ctx, args)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAllAlertsByCustomer provides a mock function with given fields: ctx, customerRef
func (_m *Alerts) GetAllAlertsByCustomer(ctx context.Context, customerRef *firestore.DocumentRef) ([]domain.Alert, error) {
	ret := _m.Called(ctx, customerRef)

	var r0 []domain.Alert
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef) ([]domain.Alert, error)); ok {
		return rf(ctx, customerRef)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef) []domain.Alert); ok {
		r0 = rf(ctx, customerRef)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]domain.Alert)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *firestore.DocumentRef) error); ok {
		r1 = rf(ctx, customerRef)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByCustomerAndAttribution provides a mock function with given fields: ctx, customerRef, attrRef
func (_m *Alerts) GetByCustomerAndAttribution(ctx context.Context, customerRef *firestore.DocumentRef, attrRef *firestore.DocumentRef) ([]*domain.Alert, error) {
	ret := _m.Called(ctx, customerRef, attrRef)

	var r0 []*domain.Alert
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, *firestore.DocumentRef) ([]*domain.Alert, error)); ok {
		return rf(ctx, customerRef, attrRef)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *firestore.DocumentRef, *firestore.DocumentRef) []*domain.Alert); ok {
		r0 = rf(ctx, customerRef, attrRef)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*domain.Alert)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *firestore.DocumentRef, *firestore.DocumentRef) error); ok {
		r1 = rf(ctx, customerRef, attrRef)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCustomerOrgRef provides a mock function with given fields: ctx, customerID, orgID
func (_m *Alerts) GetCustomerOrgRef(ctx context.Context, customerID string, orgID string) *firestore.DocumentRef {
	ret := _m.Called(ctx, customerID, orgID)

	var r0 *firestore.DocumentRef
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *firestore.DocumentRef); ok {
		r0 = rf(ctx, customerID, orgID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*firestore.DocumentRef)
		}
	}

	return r0
}

// GetRef provides a mock function with given fields: ctx, alertID
func (_m *Alerts) GetRef(ctx context.Context, alertID string) *firestore.DocumentRef {
	ret := _m.Called(ctx, alertID)

	var r0 *firestore.DocumentRef
	if rf, ok := ret.Get(0).(func(context.Context, string) *firestore.DocumentRef); ok {
		r0 = rf(ctx, alertID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*firestore.DocumentRef)
		}
	}

	return r0
}

// Share provides a mock function with given fields: ctx, alertID, collaborators, public
func (_m *Alerts) Share(ctx context.Context, alertID string, collaborators []collab.Collaborator, public *collab.PublicAccess) error {
	ret := _m.Called(ctx, alertID, collaborators, public)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []collab.Collaborator, *collab.PublicAccess) error); ok {
		r0 = rf(ctx, alertID, collaborators, public)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateAlert provides a mock function with given fields: ctx, alertID, updates
func (_m *Alerts) UpdateAlert(ctx context.Context, alertID string, updates []firestore.Update) error {
	ret := _m.Called(ctx, alertID, updates)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []firestore.Update) error); ok {
		r0 = rf(ctx, alertID, updates)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateAlertNotified provides a mock function with given fields: ctx, alertID
func (_m *Alerts) UpdateAlertNotified(ctx context.Context, alertID string) error {
	ret := _m.Called(ctx, alertID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, alertID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewAlerts creates a new instance of Alerts. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAlerts(t interface {
	mock.TestingT
	Cleanup(func())
}) *Alerts {
	mock := &Alerts{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}