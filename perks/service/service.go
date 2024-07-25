package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	"github.com/doitintl/hello/scheduled-tasks/perks/domain"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type PerkService struct {
	*logger.Logging
	*connection.Connection
	dal.Customers
}

func NewPerkService(log *logger.Logging, conn *connection.Connection) (*PerkService, error) {
	return &PerkService{
		log,
		conn,
		dal.NewCustomersFirestoreWithClient(conn.Firestore),
	}, nil
}

func (s *PerkService) SendRegisterInterestEmail(ctx context.Context, customerID string, r domain.RegisterInterest) error {
	l := s.Logger(ctx)

	if !common.Production {
		l.Infof("register interest mail to: %s was not sent in development", r.UserEmail)
		return nil
	}

	customer, err := s.GetCustomer(ctx, customerID)
	if err != nil {
		l.Errorf("failed to read customer\n Error: %v", err)
		return err
	}

	p := mail.NewPersonalization()

	tos, err := common.GetCustomerAccountManagersEmails(ctx, customer, common.AccountManagerCompanyDoit)
	if err != nil {
		l.Errorf("failed to read account manager emails of customer\n Error: %v", err)
		return err
	}

	p.AddTos(tos...)
	p.AddCCs(
		mail.NewEmail("", "mp-sales@doit.com"),
		mail.NewEmail("", "mp-sales@doit-intl.com"),
	)
	p.SetDynamicTemplateData("userEmail", r.UserEmail)
	p.SetDynamicTemplateData("company", customer.PrimaryDomain)
	p.SetDynamicTemplateData("perkName", r.PerkName)
	p.SetDynamicTemplateData("registerInterestMethod", r.ClickedOn)

	personalization := []*mail.Personalization{p}
	if err := mailer.SendEmailWithPersonalizations(personalization, mailer.Config.DynamicTemplates.PerkRegisterInterest, []string{}); err != nil {
		l.Errorf("failed to send perk register interest mail\n Error: %v", err)
		return err
	}

	return nil
}
