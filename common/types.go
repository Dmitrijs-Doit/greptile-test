package common

import (
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/stripe/stripe-go/v74"
)

type EntityPaymentType string

const (
	EntityPaymentTypeCard          EntityPaymentType = "credit_card"
	EntityPaymentTypeWireTransfer  EntityPaymentType = "wire_transfer"
	EntityPaymentTypeBankAccount   EntityPaymentType = "bank_account"
	EntityPaymentTypeUSBankAccount EntityPaymentType = "us_bank_account"
	EntityPaymentTypeBillCom       EntityPaymentType = "bill_com"
	EntityPaymentTypeSEPADebit     EntityPaymentType = "sepa_debit"
	EntityPaymentTypeBACSDebit     EntityPaymentType = "bacs_debit"
	EntityPaymentTypeACSSDebit     EntityPaymentType = "acss_debit"
)

type BillingAddress struct {
	CountryName *string `firestore:"countryname"`
	Address     *string `firestore:"address"`
	Address2    *string `firestore:"address2"`
	Address3    *string `firestore:"address3"`
	State       *string `firestore:"state"`
	StateA      *string `firestore:"statea"`
	StateCode   *string `firestore:"statecode"`
	StateName   *string `firestore:"statename"`
	Zip         *string `firestore:"zip"`
}

type Entity struct {
	PriorityID      string                      `firestore:"priorityId"`
	PriorityCompany string                      `firestore:"priorityCompany"`
	Name            string                      `firestore:"name"`
	LowerName       string                      `firestore:"_name"`
	Active          bool                        `firestore:"active"`
	Country         *string                     `firestore:"country"`
	BillingAddress  BillingAddress              `firestore:"billingAddress"`
	Currency        *string                     `firestore:"currency"`
	Customer        *firestore.DocumentRef      `firestore:"customer"`
	Invoicing       Invoicing                   `firestore:"invoicing"`
	Contact         *EntityContact              `firestore:"contact"`
	Payment         *EntityPayment              `firestore:"payment"`
	Snapshot        *firestore.DocumentSnapshot `firestore:"-"`
}

func (e Entity) PayeeCountry() string {
	switch e.PriorityCompany {
	case "doit":
		return "IL"
	case "doitint":
		return "US"
	case "doituk":
		return "GB"
	case "doitaus":
		return "AU"
	case "doitde":
		return "DE"
	case "doitfr":
		return "FR"
	case "doitnl":
		return "NL"
	case "doitch":
		return "CH"
	case "doitca":
		return "CA"
	case "doitse":
		return "SE"
	case "doites":
		return "ES"
	case "doitie":
		return "IE"
	case "doitee":
		return "EE"
	case "doitsg":
		return "SG"
	case "doitjp":
		return "JP"
	case "doitid":
		return "ID"
	default:
		return ""
	}
}

type EntityPayment struct {
	Type        EntityPaymentType           `firestore:"type"`
	AccountID   string                      `firestore:"accountId"`
	Card        *PaymentMethodCard          `firestore:"card"`
	BankAccount *PaymentMethodUSBankAccount `firestore:"bankAccount"`
	SEPADebit   *PaymentMethodSEPADebit     `firestore:"sepaDebit"`
	BACSDebit   *PaymentMethodBACSDebit     `firestore:"bacsDebit"`
	ACSSDebit   *PaymentMethodACSSDebit     `firestore:"acssDebit"`
}

func (p EntityPayment) ID() string {
	switch p.Type {
	case EntityPaymentTypeCard:
		if p.Card != nil {
			return p.Card.ID
		}

		return ""
	case EntityPaymentTypeBankAccount, EntityPaymentTypeUSBankAccount:
		return p.BankAccount.ID
	case EntityPaymentTypeSEPADebit:
		return p.SEPADebit.ID
	case EntityPaymentTypeBACSDebit:
		return p.BACSDebit.ID
	case EntityPaymentTypeACSSDebit:
		return p.ACSSDebit.ID
	default:
		return ""
	}
}

func (p EntityPayment) String() string {
	switch p.Type {
	case EntityPaymentTypeWireTransfer:
		return fmt.Sprintf("Wire Transfer")
	case EntityPaymentTypeBillCom:
		return fmt.Sprintf("Bill.com")
	case EntityPaymentTypeCard:
		if p.Card != nil {
			return fmt.Sprintf("Credit Card (%02d/%d)", p.Card.ExpMonth, p.Card.ExpYear)
		}

		return "Credit Card"
	case EntityPaymentTypeBankAccount, EntityPaymentTypeUSBankAccount:
		if p.BankAccount != nil {
			return fmt.Sprintf("Bank Account (%s)", p.BankAccount.BankName)
		}

		return "Bank Account"
	case EntityPaymentTypeSEPADebit:
		return fmt.Sprintf("SEPA Debit (%s)", p.SEPADebit.BankCode)
	case EntityPaymentTypeBACSDebit:
		return fmt.Sprintf("BACS Debit (**** %s)", p.BACSDebit.Last4)
	case EntityPaymentTypeACSSDebit:
		return fmt.Sprintf("ACSS Debit (**** %s)", p.ACSSDebit.Last4)
	default:
		return ""
	}
}

type PaymentMethodCard struct {
	ID       string                        `firestore:"id"`
	Brand    stripe.PaymentMethodCardBrand `firestore:"brand"`
	Last4    string                        `firestore:"last4"`
	ExpYear  int64                         `firestore:"expYear"`
	ExpMonth int64                         `firestore:"expMonth"`
}

type PaymentMethodUSBankAccount struct {
	ID       string `firestore:"id"`
	Last4    string `firestore:"last4"`
	BankName string `firestore:"bankName"`
}

type PaymentMethodSEPADebit struct {
	ID       string `firestore:"id"`
	Last4    string `firestore:"last4"`
	Name     string `firestore:"name"`
	Email    string `firestore:"email"`
	BankCode string `firestore:"bankCode"`
}

type PaymentMethodBACSDebit struct {
	ID    string `firestore:"id"`
	Last4 string `firestore:"last4"`
	Name  string `firestore:"name"`
	Email string `firestore:"email"`
}

type PaymentMethodACSSDebit struct {
	ID    string `firestore:"id"`
	Last4 string `firestore:"last4"`
	Name  string `firestore:"name"`
	Email string `firestore:"email"`
}

type EntityContact struct {
	Name      *string `firestore:"name"`
	FirstName *string `firestore:"firstName"`
	LastName  *string `firestore:"lastName"`
	Email     *string `firestore:"email"`
	Phone     *string `firestore:"phone"`
}

type Invoicing struct {
	Mode             string                 `firestore:"mode"`
	Default          *firestore.DocumentRef `firestore:"default"`
	AutoAssignGCP    *bool                  `firestore:"autoAssignGCP"`
	AttributionGroup *firestore.DocumentRef `firestore:"attributionGroup"`
	Marketplace      map[string]interface{} `firestore:"marketplace"`
}

type AccountManagerCompany string

const (
	AccountManagerCompanyDoit AccountManagerCompany = "doit"
	AccountManagerCompanyGcp  AccountManagerCompany = "google_cloud_platform"
	AccountManagerCompanyAws  AccountManagerCompany = "amazon_web_services"
)

type AccountManagerRole string

const (
	AccountManagerRoleFSR AccountManagerRole = "account_manager"
	AccountManagerRoleSAM AccountManagerRole = "strategic_accounts_manager"
	AccountManagerRoleTAM AccountManagerRole = "technical_account_manager"
	AccountManagerRoleCSM AccountManagerRole = "customer_success_manager"
	AccountManagerRoleCE  AccountManagerRole = "customer_engineer"
	AccountManagerRolePSE AccountManagerRole = "partner_sales_engineer"
)

// AccountManager : Firestore account manager document structure
type AccountManager struct {
	Name     string                 `firestore:"name"`
	Email    string                 `firestore:"email"`
	Phone    string                 `firestore:"phone"`
	PhotoURL string                 `firestore:"photoURL"`
	Role     AccountManagerRole     `firestore:"role"`
	Company  AccountManagerCompany  `firestore:"company"`
	Manager  *firestore.DocumentRef `firestore:"manager"`
}

// CountriesInfo is mapping of 2 letter country code to name and region
type CountriesInfo struct {
	Code map[string]struct {
		Name   string `firestore:"name"`
		Region string `firestore:"region"`
	} `firestore:"code"`
}

// TableInfo represents the full path for a BigQuery table.
type TableInfo struct {
	ProjectID string `firestore:"projectId"`
	DatasetID string `firestore:"datasetId"`
	TableID   string `firestore:"tableId"`
}

type Bucket struct {
	Name        string                 `firestore:"name"`
	Attribution *firestore.DocumentRef `firestore:"attribution"`
	Ref         *firestore.DocumentRef `firestore:"-"`
}
