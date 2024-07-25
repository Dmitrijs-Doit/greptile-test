// Code generated by mockery v2.35.2. DO NOT EDIT.

package mocks

import (
	connection "github.com/doitintl/hello/scheduled-tasks/framework/connection"
	gin "github.com/gin-gonic/gin"

	mock "github.com/stretchr/testify/mock"
)

// ReportAPIService is an autogenerated mock type for the ReportAPIService type
type ReportAPIService struct {
	mock.Mock
}

// ListReports provides a mock function with given fields: ctx, conn
func (_m *ReportAPIService) ListReports(ctx *gin.Context, conn *connection.Connection) {
	_m.Called(ctx, conn)
}

// RunReport provides a mock function with given fields: ctx, conn
func (_m *ReportAPIService) RunReport(ctx *gin.Context, conn *connection.Connection) {
	_m.Called(ctx, conn)
}

// NewReportAPIService creates a new instance of ReportAPIService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewReportAPIService(t interface {
	mock.TestingT
	Cleanup(func())
}) *ReportAPIService {
	mock := &ReportAPIService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
