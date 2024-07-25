package flexsaveresold

import (
	"context"
	"errors"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	logger "github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
)

type FlexsaveMailer interface {
	SendNotification(sn *mailer.SimpleNotification, to string, params map[string]interface{}) error
}

func newCowardMailer() FlexsaveMailer {
	return mailer.CowardMailer{}
}

type EmailInterface interface {
	SendWelcomeEmail(
		ctx context.Context,
		params *types.WelcomeEmailParams,
		usersWithPermissions []*common.User,
		accountManagers []*common.AccountManager,
	) error
}

type Email struct {
	Logger logger.Provider
	Mailer FlexsaveMailer
}

func NewMail(logger logger.Provider, conn *connection.Connection) *Email {
	flexsaveMailer := newCowardMailer()

	if common.Production {
		flexsaveMailer = mailer.NewMailer()
	}

	return &Email{
		logger,
		flexsaveMailer,
	}
}

func getCustomerDashboardUrl(customerID string, cloud string) string {
	if cloud == common.GCP {
		return fmt.Sprintf("https://%s/customers/%s/flexsave-gcp", common.Domain, customerID)
	}

	return fmt.Sprintf("https://%s/customers/%s/flexsave-aws", common.Domain, customerID)
}

func getCustomerSupportUrl(customerID string) string {
	return fmt.Sprintf("https://%s/customers/%s/support/new", common.Domain, customerID)
}

func (s *Email) SendWelcomeEmail(
	ctx context.Context,
	params *types.WelcomeEmailParams,
	usersWithPermissions []*common.User,
	accountManagers []*common.AccountManager,
) error {
	log := s.Logger(ctx)

	if params == nil {
		return errors.New("empty email params")
	}

	var bccs []string
	for _, am := range accountManagers {
		bccs = append(bccs, am.Email)
	}

	emailParams := make(map[string]interface{})
	emailParams["cloud"] = params.Cloud
	emailParams["marketplace"] = params.Marketplace
	emailParams["dashboard_link"] = getCustomerDashboardUrl(params.CustomerID, params.Cloud)
	emailParams["support_link"] = getCustomerSupportUrl(params.CustomerID)

	for index, user := range usersWithPermissions {
		to := user.Email

		emailParams["first_name"] = user.FirstName

		sn := mailer.SimpleNotification{}
		sn.TemplateID = mailer.Config.DynamicTemplates.FlexsaveActivation
		sn.Categories = []string{mailer.CatagoryFlexsave}

		if index == 0 {
			sn.BCCs = bccs
		}

		if err := s.Mailer.SendNotification(&sn, to, emailParams); err != nil {
			log.Error(err)
		}
	}

	return nil
}
