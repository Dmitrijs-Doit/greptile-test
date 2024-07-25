package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/csmengagement/mocks"
	customerDalMock "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/customer/domain"
	loggerMock "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	userDalMock "github.com/doitintl/hello/scheduled-tasks/user/dal/mocks"
	ncMock "github.com/doitintl/notificationcenter/mocks"
	nc "github.com/doitintl/notificationcenter/pkg"
)

func TestService_SendFirstInvoiceEmail(t *testing.T) {
	ctx := context.Background()
	lMock := loggerMock.ILogger{}

	notificationMock := ncMock.NewNotificationSender(t)
	csmService := mocks.NewICsmEngagement(t)
	cDalMock := customerDalMock.NewCustomers(t)
	uDalMock := userDalMock.NewIUserFirestoreDAL(t)

	lMock.On("Info", mock.Anything)
	lMock.On("Infof", mock.Anything, mock.Anything)
	lMock.On("Infof", mock.Anything, mock.Anything, mock.Anything)
	s := service{
		l:                  &lMock,
		notificationSender: notificationMock,
		csmService:         csmService,
		customerDAL:        cDalMock,
		userDAL:            uDalMock,
	}

	t.Run("Run on real env for debug - skip on ci/cd tests", func(t *testing.T) {

	})

	t.Run("should skip zero mrr as standalone customer", func(t *testing.T) {
		csmService = mocks.NewICsmEngagement(t)
		csmService.On("GetCustomerMRR", mock.Anything, mock.Anything, mock.Anything).Return(float64(0), nil)
		s.csmService = csmService

		err := s.SendFirstInvoiceEmail(ctx, "2LAIuL6mqexozk2pM1qk")
		assert.NoError(t, err)
		assert.Contains(t, lMock.Calls[1].Arguments[0], "skipping standalone customer")
	})

	t.Run("should pick the <20K template for <20K mrr", func(t *testing.T) {
		csmService = mocks.NewICsmEngagement(t)
		csmService.On("GetCustomerMRR", mock.Anything, mock.Anything, mock.Anything).Return(float64(19999), nil)
		s.csmService = csmService

		cDalMock = customerDalMock.NewCustomers(t)
		cDalMock.On("GetCustomer", mock.Anything, mock.Anything).Return(&common.Customer{
			Name: "Any customer",
		}, nil)
		cDalMock.On("GetCustomerAccountTeam", mock.Anything, mock.Anything).Return(
			[]domain.AccountManagerListItem{
				{Name: "John Doe", Email: "john_d@doit.com", Role: common.AccountManagerRoleFSR},
				{Name: "Jane Doe", Email: "jahe_d@doit.com", Role: common.AccountManagerRoleCSM, CalendlyLink: "https://calendy-link.com"},
				{Name: "Elvis Presley", Email: "elvise_p@doit.com", Role: common.AccountManagerRoleFSR},
			}, nil)

		s.customerDAL = cDalMock

		uDalMock = userDalMock.NewIUserFirestoreDAL(t)
		uDalMock.On("GetCustomerUsersWithInvoiceNotification", mock.Anything, mock.Anything, mock.Anything).Return([]*common.User{
			{Email: "user1.@customer.com"}, {Email: "user2.@customer.com"},
		}, nil)

		s.userDAL = uDalMock

		notificationMock = ncMock.NewNotificationSender(t)
		notificationMock.On("Send", mock.Anything, mock.Anything).Return("responseId", nil)
		s.notificationSender = notificationMock

		err := s.SendFirstInvoiceEmail(ctx, "2LAIuL6mqexozk2pM1qk")
		ncContent := notificationMock.Mock.Calls[0].Arguments[1].(nc.Notification)

		assert.NoError(t, err)
		assert.Equal(t, firstInvoiceTemplateUnder20k, ncContent.Template)
	})

	t.Run("should pick the >20K template for >20K mrr", func(t *testing.T) {
		csmService = mocks.NewICsmEngagement(t)
		csmService.On("GetCustomerMRR", mock.Anything, mock.Anything, mock.Anything).Return(float64(20001), nil)
		s.csmService = csmService

		cDalMock = customerDalMock.NewCustomers(t)
		cDalMock.On("GetCustomer", mock.Anything, mock.Anything).Return(&common.Customer{
			Name: "Any customer",
		}, nil)
		cDalMock.On("GetCustomerAccountTeam", mock.Anything, mock.Anything).Return(
			[]domain.AccountManagerListItem{
				{Name: "John Doe", Email: "john_d@doit.com", Role: common.AccountManagerRoleFSR},
				{Name: "Jane Doe", Email: "jahe_d@doit.com", Role: common.AccountManagerRoleCSM, CalendlyLink: "https://calendy-link.com"},
				{Name: "Elvis Presley", Email: "elvise_p@doit.com", Role: common.AccountManagerRoleFSR},
			}, nil)

		s.customerDAL = cDalMock

		uDalMock = userDalMock.NewIUserFirestoreDAL(t)
		uDalMock.On("GetCustomerUsersWithInvoiceNotification", mock.Anything, mock.Anything, mock.Anything).Return([]*common.User{
			{Email: "user1.@customer.com"}, {Email: "user2.@customer.com"},
		}, nil)

		s.userDAL = uDalMock

		notificationMock = ncMock.NewNotificationSender(t)
		notificationMock.On("Send", mock.Anything, mock.Anything).Return("responseId", nil)
		s.notificationSender = notificationMock

		err := s.SendFirstInvoiceEmail(ctx, "2LAIuL6mqexozk2pM1qk")
		ncContent := notificationMock.Mock.Calls[0].Arguments[1].(nc.Notification)

		assert.NoError(t, err)
		assert.Equal(t, firstInvoiceTemplateAbove20k, ncContent.Template)
		assert.Equal(t, "https://calendy-link.com", ncContent.Data["calendlyLink"])
	})
}
