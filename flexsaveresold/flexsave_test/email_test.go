package flexsave

import (
	"net/http/httptest"
	"testing"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexsave_test/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestEmail(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	loggerMock := logger.FromContext

	emailMock := mocks.FlexsaveMailer{}
	expectedNotification1 := mailer.SimpleNotification{
		TemplateID: mailer.Config.DynamicTemplates.FlexsaveActivation,
		BCCs: []string{
			"am1@gmail.com",
			"am2@gmail.com",
		},
		Categories: []string{mailer.CatagoryFlexsave},
	}

	expectedNotification2 := mailer.SimpleNotification{
		TemplateID: mailer.Config.DynamicTemplates.FlexsaveActivation,
		Categories: []string{mailer.CatagoryFlexsave}}

	emailMock.On(
		"SendNotification",
		&expectedNotification1,
		"user1@gmail.com",
		map[string]interface{}{
			"first_name": "",
			"cloud":      "GCP", "marketplace": true,
			"dashboard_link": "https://dev-app.doit.com/customers/DUMMY_CUSTOMER/flexsave-gcp",
			"support_link":   "https://dev-app.doit.com/customers/DUMMY_CUSTOMER/support/new",
		},
	).
		Return(nil).
		On(
			"SendNotification",
			&expectedNotification2,
			"user2@gmail.com",
			map[string]interface{}{
				"first_name": "",
				"cloud":      "GCP", "marketplace": true,
				"dashboard_link": "https://dev-app.doit.com/customers/DUMMY_CUSTOMER/flexsave-gcp",
				"support_link":   "https://dev-app.doit.com/customers/DUMMY_CUSTOMER/support/new",
			},
		).
		Return(nil)

	service := flexsaveresold.Email{
		loggerMock,
		&emailMock,
	}

	user1 := common.User{Email: "user1@gmail.com"}
	user2 := common.User{Email: "user2@gmail.com"}
	mockCustomersWithPermissions := []*common.User{&user1, &user2}

	am1 := common.AccountManager{Email: "am1@gmail.com"}
	am2 := common.AccountManager{Email: "am2@gmail.com"}
	mockAccountManagers := []*common.AccountManager{&am1, &am2}

	emailParams := types.WelcomeEmailParams{
		Cloud:       common.GCP,
		CustomerID:  "DUMMY_CUSTOMER",
		Marketplace: true,
	}

	result := service.SendWelcomeEmail(ctx, &emailParams, mockCustomersWithPermissions, mockAccountManagers)
	assert.Nil(t, result)

	err := service.SendWelcomeEmail(ctx, nil, mockCustomersWithPermissions, mockAccountManagers)
	assert.Error(t, err, "empty email params")
}
