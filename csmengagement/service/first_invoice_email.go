package service

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/customer/domain"
	nc "github.com/doitintl/notificationcenter/pkg"
)

const (
	firstInvoiceTemplateAbove20k = "FQ9ZAR2CQNMC19J6M3JG4H2RJ63S"
	firstInvoiceTemplateUnder20k = "T3BQFNK4XA4942PGHJS91EKR1JXG"
)

func (s *service) SendFirstInvoiceEmail(ctx context.Context, customerID string) error {
	s.l.Infof("SendFirstInvoiceEmail, customerID: %service", customerID)

	mrr, err := s.csmService.GetCustomerMRR(ctx, customerID, true)
	if err != nil {
		s.l.Errorf("SendFirstInvoiceEmail => GetCustomerMRR error: %s", err)
		return err
	}

	if mrr == 0 {
		s.l.Info("SendFirstInvoiceEmail => skipping standalone customer ")
		return nil
	}

	customer, err := s.customerDAL.GetCustomer(ctx, customerID)
	if err != nil {
		s.l.Errorf("SendFirstInvoiceEmail => GetCustomer error: %s", err)
		return err
	}

	accountTeam, err := s.customerDAL.GetCustomerAccountTeam(ctx, customerID)
	if err != nil {
		s.l.Errorf("SendFirstInvoiceEmail => GetCustomerAccountTeam error: %s", err)
		return err
	}

	to, err := s.getTo(ctx, customerID)
	if err != nil {
		s.l.Errorf("SendFirstInvoiceEmail => getTo error: %s", err)
		return err
	}

	if len(to) == 0 {
		s.l.Info("SendFirstInvoiceEmail => no recipients for first invoice")
		return nil
	}

	bcc := getBcc(accountTeam)

	accountTeamTemplate, calendlyLink := GetAccountTeamHTMLTemplate(accountTeam)

	template := firstInvoiceTemplateUnder20k
	if mrr > 20000 {
		template = firstInvoiceTemplateAbove20k
	}

	n := nc.Notification{
		Template: template,
		Email:    to,
		BCC:      bcc,
		EmailFrom: nc.EmailFrom{
			Name:  "DoiT International",
			Email: "csm@doit-intl.com",
		},
		Data: map[string]interface{}{
			"customerName":        customer.Name,
			"invoiceLink":         getInvoiceLink(customerID),
			"accountTeamTemplate": accountTeamTemplate,
			"calendlyLink":        calendlyLink,
		},
		Mock: !common.Production,
	}

	res, err := s.notificationSender.Send(ctx, n)
	if err != nil {
		s.l.Errorf("SendFirstInvoiceEmail => error sending message: %s", err)
		return err
	}

	s.l.Infof("SendFirstInvoiceEmail, res: %s customerID: %s", res, customerID)

	return nil
}

func getInvoiceLink(customerID string) string {
	sub := "dev-app"
	if common.Production {
		sub = "console"
	}

	return fmt.Sprintf("https://%s.doit.com/customers/%s/invoices", sub, customerID)
}

func getLiItem(name, role, email string) string {
	return fmt.Sprintf(`<li>%s, %s, (<a href='mailto:%s'>%s</a>)</li>`, name, role, email, email)
}

func getBcc(accountTeam []domain.AccountManagerListItem) []string {
	var bcc []string

	for _, accountManager := range accountTeam {
		bcc = append(bcc, accountManager.Email)
	}

	return bcc
}

func (s *service) getTo(ctx context.Context, customerID string) ([]string, error) {
	var to []string

	users, err := s.userDAL.GetCustomerUsersWithInvoiceNotification(ctx, customerID, "")
	if err != nil {
		return to, err
	}

	for _, user := range users {
		to = append(to, user.Email)
	}

	return to, nil
}
