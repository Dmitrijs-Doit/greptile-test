package service

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/stripe/stripe-go/v74"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
)

func (s *StripeService) sendPaymentFailedNotification(ctx context.Context, entity *common.Entity, invoice *invoices.FullInvoice, amount int64, stripeErr *stripe.Error) error {
	if stripeErr.Type == stripe.ErrorTypeCard {
		switch stripeErr.Code {
		case stripe.ErrorCodeCardDeclined:
		case stripe.ErrorCodeExpiredCard:
		case stripe.ErrorCodeIncorrectCVC:
		case stripe.ErrorCodeIncorrectZip:
		case stripe.ErrorCodeIncorrectNumber:
		case stripe.ErrorCodeInvalidCVC:
		case stripe.ErrorCodeInvalidExpiryMonth:
		case stripe.ErrorCodeInvalidExpiryYear:
		case stripe.ErrorCodeInvalidNumber:
		default:
			return nil
		}
	} else {
		return nil
	}

	l := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)

	msgPrinter := message.NewPrinter(language.English)

	customer, err := common.GetCustomer(ctx, entity.Customer)
	if err != nil {
		return err
	}

	entityRef := invoice.Entity
	ref := entityRef.Collection("entityMetadata").Doc("stripe-card-alert")

	dayDuration := 24 * time.Hour
	shouldSendNotification := false
	now := time.Now().UTC().Truncate(dayDuration)

	if err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docSnap, err := tx.Get(ref)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}

		if docSnap.Exists() {
			v, err := docSnap.DataAt("timestamp")
			if err != nil {
				return err
			}
			lastNotification := v.(time.Time)

			// If the last notification was sent less then 7 days ago, skip sending
			if !lastNotification.IsZero() && lastNotification.Add(dayDuration*7).After(now) {
				return nil
			}
		}

		if err := tx.Set(ref, map[string]interface{}{
			"timestamp": now,
		}); err != nil {
			return err
		}

		shouldSendNotification = true
		return nil
	}, firestore.MaxAttempts(5)); err != nil {
		return err
	}

	if !shouldSendNotification {
		return nil
	}

	tos := make([]*mail.Email, 0)
	ccs := make([]*mail.Email, 0)
	bccs := make([]*mail.Email, 0)

	if entity.Contact != nil && entity.Contact.Email != nil {
		var name string
		if entity.Contact.Name != nil {
			name = *entity.Contact.Name
		}

		tos = append(tos, mail.NewEmail(name, *entity.Contact.Email))
	}

	userDocSnaps, err := fs.Collection("users").
		Where("customer.ref", "==", entity.Customer).
		Where("entities", "array-contains", entityRef).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, docSnap := range userDocSnaps {
		var user common.User
		if err := docSnap.DataTo(&user); err != nil {
			l.Warning(err.Error())
			continue
		}

		if user.Email != "" && user.HasEntitiesPermission(ctx) && user.NotificationsPaymentReminders() {
			if mailer.Index(tos, user.Email) == -1 {
				tos = append(tos, mail.NewEmail(user.DisplayName, user.Email))
			}
		}
	}

	if len(tos) <= 0 {
		return nil
	}

	if accountManager, err := common.GetAccountManager(ctx, customer.AccountManager); err != nil {
		l.Warning(err.Error())
	} else if accountManager != nil {
		if mailer.Index(tos, accountManager.Email) == -1 {
			ccs = append(ccs, mail.NewEmail(accountManager.Name, accountManager.Email))
		}
	}

	for _, email := range []string{"vadim@doit.com", "dror@doit.com"} {
		if mailer.Index(tos, email) == -1 && mailer.Index(ccs, email) == -1 {
			bccs = append(bccs, mail.NewEmail("", email))
		}
	}

	var products []string

	for _, p := range invoice.Products {
		productLabel := common.FormatAssetType(p)
		if productLabel != "" && productLabel != "other" {
			products = append(products, productLabel)
		}
	}

	m := mail.NewV3Mail()
	m.SetTemplateID(mailer.Config.DynamicTemplates.CreditCardPaymentFailed)
	m.SetFrom(mail.NewEmail(mailer.Config.BillingName, mailer.Config.BillingEmail))

	enable := false
	m.SetTrackingSettings(&mail.TrackingSettings{SubscriptionTracking: &mail.SubscriptionTrackingSetting{Enable: &enable}})

	p := mail.NewPersonalization()
	p.AddTos(tos...)
	p.AddCCs(ccs...)
	p.AddBCCs(bccs...)
	p.SetDynamicTemplateData("products", strings.Join(products, ", "))
	p.SetDynamicTemplateData("customer_id", invoice.Customer.ID)
	p.SetDynamicTemplateData("entity_id", invoice.Entity.ID)
	p.SetDynamicTemplateData("invoice_id", invoice.ID)
	p.SetDynamicTemplateData("domain", customer.PrimaryDomain)
	p.SetDynamicTemplateData("date", now.Format("02 Jan 2006"))
	p.SetDynamicTemplateData("amount", fixer.FormatCurrencyAmountInt64(msgPrinter, amount, invoice.Symbol))
	m.AddPersonalizations(p)

	request := sendgrid.GetRequest(mailer.Config.APIKey, mailer.Config.MailSendPath, mailer.Config.BaseURL)
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)

	if _, err := sendgrid.MakeRequestRetry(request); err != nil {
		return err
	}

	l.Println("payment failed notification sent successfully")

	return nil
}
