package service

import (
	"context"
	"sort"
	"time"

	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/mailer"
)

type PaymentIntentRow struct {
	CustomerName     string                     `json:"customer_name"`
	PriorityID       string                     `json:"priority_id"`
	Status           stripe.PaymentIntentStatus `json:"status"`
	Timestamp        int64                      `json:"timestamp"`
	CreateTime       string                     `json:"create_time"`
	Currency         string                     `json:"currency"`
	Total            float64                    `json:"total"`
	TotalReceived    float64                    `json:"total_received"`
	Amount           int64                      `json:"amount"`
	AmountReceived   int64                      `json:"amount_received"`
	Metadata         map[string]string          `json:"metadata"`
	LastPaymentError *stripe.Error              `json:"last_payment_error"`
}

func (s *StripeService) PaymentsDigest(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC)
	startTime := today.Add(time.Hour * 24 * -7)
	endTime := today.Add(time.Hour * 24)

	succesfulPayments := make([]*PaymentIntentRow, 0)
	failedPayments := make([]*PaymentIntentRow, 0)
	achPayments := make([]*PaymentIntentRow, 0)
	sepaPayments := make([]*PaymentIntentRow, 0)
	bacsPayments := make([]*PaymentIntentRow, 0)
	acssPayments := make([]*PaymentIntentRow, 0)
	disputes := s.Disputes(ctx, startTime, endTime)

	params := &stripe.PaymentIntentListParams{
		ListParams: stripe.ListParams{
			Limit:  stripe.Int64(100),
			Expand: []*string{stripe.String("data.customer")},
		},
		CreatedRange: &stripe.RangeQueryParams{
			GreaterThanOrEqual: startTime.Unix(),
			LesserThan:         endTime.Unix(),
		},
	}

	iter := s.stripeClient.PaymentIntents.List(params)
	for iter.Next() {
		pi := iter.PaymentIntent()
		created := time.Unix(pi.Created, 0).UTC()
		row := PaymentIntentRow{
			CustomerName:     pi.Customer.Name,
			PriorityID:       pi.Customer.Metadata["priority_id"],
			Status:           pi.Status,
			Timestamp:        pi.Created,
			CreateTime:       created.Format("02/01 15:04:05"),
			Currency:         CurrencyToUpperString(pi.Currency),
			Amount:           pi.Amount,
			AmountReceived:   pi.AmountReceived,
			Total:            float64(pi.Amount) / 100,
			TotalReceived:    float64(pi.AmountReceived) / 100,
			Metadata:         pi.Metadata,
			LastPaymentError: pi.LastPaymentError,
		}

		if len(pi.PaymentMethodTypes) == 0 {
			continue
		}

		switch pi.PaymentMethodTypes[0] {
		case string(stripe.PaymentMethodTypeUSBankAccount):
			achPayments = append(achPayments, &row)
		case string(stripe.PaymentMethodTypeSEPADebit):
			sepaPayments = append(sepaPayments, &row)
		case string(stripe.PaymentMethodTypeBACSDebit):
			bacsPayments = append(bacsPayments, &row)
		case string(stripe.PaymentMethodTypeACSSDebit):
			acssPayments = append(acssPayments, &row)
		default:
			if (today.Before(created) && created.Before(endTime)) || today.Equal(created) {
				if pi.Status == stripe.PaymentIntentStatusSucceeded {
					succesfulPayments = append(succesfulPayments, &row)
				} else {
					failedPayments = append(failedPayments, &row)
				}
			}
		}
	}

	sort.Slice(succesfulPayments, func(i, j int) bool {
		if succesfulPayments[i].PriorityID == succesfulPayments[j].PriorityID {
			return succesfulPayments[i].Timestamp < succesfulPayments[j].Timestamp
		}

		return succesfulPayments[i].PriorityID < succesfulPayments[j].PriorityID
	})
	sort.Slice(failedPayments, func(i, j int) bool {
		if failedPayments[i].PriorityID == failedPayments[j].PriorityID {
			return failedPayments[i].Timestamp < failedPayments[j].Timestamp
		}

		return failedPayments[i].PriorityID < failedPayments[j].PriorityID
	})
	sort.Slice(achPayments, func(i, j int) bool {
		return achPayments[i].Timestamp > achPayments[j].Timestamp
	})
	sort.Slice(sepaPayments, func(i, j int) bool {
		return sepaPayments[i].Timestamp > sepaPayments[j].Timestamp
	})
	sort.Slice(bacsPayments, func(i, j int) bool {
		return bacsPayments[i].Timestamp > bacsPayments[j].Timestamp
	})
	sort.Slice(acssPayments, func(i, j int) bool {
		return acssPayments[i].Timestamp > acssPayments[j].Timestamp
	})

	enableTracking := false
	m := mail.NewV3Mail()
	m.SetTemplateID(mailer.Config.DynamicTemplates.StripeDailyDigest)
	m.SetFrom(mail.NewEmail(mailer.Config.NoReplyName, mailer.Config.NoReplyEmail))
	m.SetTrackingSettings(&mail.TrackingSettings{SubscriptionTracking: &mail.SubscriptionTrackingSetting{Enable: &enableTracking}})

	personalization := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail("Dror Levy", "dror@doit.com"),
		mail.NewEmail("Vadim Solovey", "vadim@doit.com"),
		mail.NewEmail("accounting", "accounting@doit-intl.com"),
	}

	personalization.AddTos(tos...)
	personalization.SetDynamicTemplateData("date", today.Format("Jan 2, 2006"))
	personalization.SetDynamicTemplateData("failed_payments", failedPayments)
	personalization.SetDynamicTemplateData("successful_payments", succesfulPayments)
	personalization.SetDynamicTemplateData("ach_payments", achPayments)
	personalization.SetDynamicTemplateData("sepa_payments", sepaPayments)
	personalization.SetDynamicTemplateData("bacs_payments", bacsPayments)
	personalization.SetDynamicTemplateData("acss_payments", acssPayments)
	personalization.SetDynamicTemplateData("disputes", disputes)
	m.AddPersonalizations(personalization)

	request := sendgrid.GetRequest(mailer.Config.APIKey, mailer.Config.MailSendPath, mailer.Config.BaseURL)
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)

	response, err := sendgrid.MakeRequestRetry(request)
	if err != nil {
		l.Error(err)
	} else {
		l.Println(response.StatusCode)
		l.Println(response.Body)
	}

	return nil
}

type DisputeRow struct {
	Created       string               `json:"created"`
	CustomerName  string               `json:"customer_name"`
	CustomerEmail string               `json:"customer_email"`
	PriorityID    string               `json:"priority_id"`
	InvoiceID     string               `json:"invoice_id"`
	ReceiptID     string               `json:"receipt_id"`
	Status        stripe.DisputeStatus `json:"status"`
	Reason        stripe.DisputeReason `json:"reason"`
	Currency      string               `json:"currency"`
	Amount        int64                `json:"amount"`
}

func (s *StripeService) Disputes(ctx context.Context, gte time.Time, lt time.Time) []*DisputeRow {
	params := &stripe.DisputeListParams{
		ListParams: stripe.ListParams{
			Limit:  stripe.Int64(100), // stripe automatically paginates if there are more than 100 disputes
			Expand: []*string{stripe.String("data.payment_intent.customer")},
		},
		CreatedRange: &stripe.RangeQueryParams{
			GreaterThanOrEqual: gte.Unix(),
			LesserThan:         lt.Unix(),
		},
	}
	iter := s.stripeClient.Disputes.List(params)
	disputes := make([]*DisputeRow, 0)

	for iter.Next() {
		d := iter.Dispute()
		CustomerName := ""
		PriorityID := ""
		InvoiceID := ""
		ReceiptID := ""
		CustomerEmail := ""

		if d.PaymentIntent != nil {
			PriorityID = d.PaymentIntent.Metadata["priority_id"]
			InvoiceID = d.PaymentIntent.Metadata["invoice_id"]
			ReceiptID = d.PaymentIntent.Metadata["receipt_id"]
			CustomerEmail = d.PaymentIntent.Metadata["email"]

			if d.PaymentIntent.Customer != nil {
				CustomerName = d.PaymentIntent.Customer.Name
			}
		}

		row := DisputeRow{
			Created:       time.Unix(d.Created, 0).Format(time.RFC1123),
			CustomerName:  CustomerName,
			CustomerEmail: CustomerEmail,
			PriorityID:    PriorityID,
			InvoiceID:     InvoiceID,
			ReceiptID:     ReceiptID,
			Status:        d.Status,
			Reason:        d.Reason,
			Currency:      string(d.Currency),
			Amount:        d.Amount,
		}
		disputes = append(disputes, &row)
	}

	return disputes
}
