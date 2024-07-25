package handlers

import (
	"errors"
	"net/http/httptest"
	"testing"

	manageMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/manage/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestResoldAWSCache_UpdateCustomerStatuses(t *testing.T) {
	type fields struct {
		manageFlexsave manageMocks.Service
	}

	tests := []struct {
		name    string
		fields  fields
		on      func(*fields)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return error if manageFlexsave.PayerStatusUpdateForEnabledCustomers returns error",
			on: func(f *fields) {
				f.manageFlexsave.On("PayerStatusUpdateForEnabledCustomers", mock.Anything).Return(errors.New("error"))
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},

		{
			name: "should return error if manageFlexsave.EnableEligiblePayers returns error",
			on: func(f *fields) {
				f.manageFlexsave.On("PayerStatusUpdateForEnabledCustomers", mock.Anything).Return(nil)
				f.manageFlexsave.On("EnableEligiblePayers", mock.Anything).Return(errors.New("error"))
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},

		{
			name: "should run all updates successfully",
			on: func(f *fields) {
				f.manageFlexsave.On("PayerStatusUpdateForEnabledCustomers", mock.Anything).Return((nil))
				f.manageFlexsave.On("EnableEligiblePayers", mock.Anything).Return(nil)
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Nil(t, err)
				return true
			},
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			f := fields{}

			tt.on(&f)

			h := &ResoldAWSCache{
				manageFlexsave: &f.manageFlexsave,
			}

			w := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(w)

			tt.wantErr(t, h.UpdateCustomerStatuses(context))
		})
	}
}
