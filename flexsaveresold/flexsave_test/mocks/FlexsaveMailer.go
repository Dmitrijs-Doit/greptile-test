// Code generated by mockery v2.10.0. DO NOT EDIT.

package mocks

import (
	mailer "github.com/doitintl/hello/scheduled-tasks/mailer"
	mock "github.com/stretchr/testify/mock"
)

// FlexsaveMailer is an autogenerated mock type for the FlexsaveMailer type
type FlexsaveMailer struct {
	mock.Mock
}

// SendNotification provides a mock function with given fields: sn, to, params
func (_m *FlexsaveMailer) SendNotification(sn *mailer.SimpleNotification, to string, params map[string]interface{}) error {
	ret := _m.Called(sn, to, params)

	var r0 error
	if rf, ok := ret.Get(0).(func(*mailer.SimpleNotification, string, map[string]interface{}) error); ok {
		r0 = rf(sn, to, params)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
