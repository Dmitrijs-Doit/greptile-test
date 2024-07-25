package service

import (
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
)

type ChargeSummary struct {
	ID                  string                 `firestore:"id"`
	AccountID           domain.StripeAccountID `firestore:"account_id"`
	Amount              int64                  `firestore:"amount"`
	Captured            bool                   `firestore:"captured"`
	Created             int64                  `firestore:"created"`
	Currency            stripe.Currency        `firestore:"currency"`
	Customer            string                 `firestore:"customer"`
	Description         string                 `firestore:"description"`
	Livemode            bool                   `firestore:"livemode"`
	Metadata            map[string]string      `firestore:"metadata"`
	Paid                bool                   `firestore:"paid"`
	Status              string                 `firestore:"status"`
	PriorityID          string                 `firestore:"priorityId"`
	PriorityCompany     string                 `firestore:"priorityCompany"`
	InvoiceID           string                 `firestore:"invoiceId"`
	StatementDescriptor string                 `firestore:"statement_descriptor"`
	Timestamp           time.Time              `firestore:"timestamp,serverTimestamp"`
}

type PaymentIntent struct {
	Refs                      PaymentRefs                `firestore:"refs"`
	ID                        string                     `firestore:"id"`
	AccountID                 domain.StripeAccountID     `firestore:"account_id"`
	Amount                    int64                      `firestore:"amount"`
	AmountCapturable          int64                      `firestore:"amount_capturable"`
	AmountReceived            int64                      `firestore:"amount_received"`
	Created                   int64                      `firestore:"created"`
	Currency                  stripe.Currency            `firestore:"currency"`
	Customer                  string                     `firestore:"customer"`
	Description               string                     `firestore:"description"`
	Livemode                  bool                       `firestore:"livemode"`
	Metadata                  map[string]string          `firestore:"metadata"`
	Status                    stripe.PaymentIntentStatus `firestore:"status"`
	StatementDescriptor       string                     `firestore:"statement_descriptor"`
	StatementDescriptorSuffix string                     `firestore:"statement_descriptor_suffix"`
	CanceledAt                *int64                     `firestore:"canceled_at"`
	PaymentMethodTypes        []string                   `firestore:"payment_method_types"`
	Timestamp                 time.Time                  `firestore:"timestamp,serverTimestamp"`
}

type PaymentRefs struct {
	Customer *firestore.DocumentRef `firestore:"customer"`
	Entity   *firestore.DocumentRef `firestore:"entity"`
	Invoice  *firestore.DocumentRef `firestore:"invoice"`
}
