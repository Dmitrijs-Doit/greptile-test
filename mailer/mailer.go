package mailer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

type SendGridConfig struct {
	APIKey       string `json:"api_key"`
	BaseURL      string `json:"base_url"`
	MailSendPath string `json:"mail_send_path"`

	// <billing@doit-intl.com>
	BillingEmail string `json:"billing_email"`
	BillingName  string `json:"billing_name"`
	// <noreply@doit-intl.com>
	NoReplyEmail string `json:"no_reply_email"`
	NoReplyName  string `json:"no_reply_name"`
	// <order-desk@doit-intl.com>
	OrderDeskEmail string `json:"order_desk_email"`
	OrderDeskName  string `json:"order_desk_name"`

	// Dynamic templates IDs
	DynamicTemplates DynamicTemplates `json:"dynamic_templates"`
}

type DynamicTemplates struct {
	AssetsDailyDigest                       string `json:"assets_daily_digest"`
	AmazonWebServicesAccountCreated         string `json:"aws_account_created"`
	AmazonWebServicesInviteAccepted         string `json:"aws_invite_accepted"`
	AmazonWebServicesTcoAnalysis            string `json:"aws_tco_analysis"`
	BillingProfileSignupNotification        string `json:"billing_profile_signup_notification"`
	BillingProfileUpdateNotification        string `json:"billing_profile_update_notification"`
	CloudReportShare                        string `json:"cloud_report_share"`
	CloudAnalyticsAlertsDigest              string `json:"cloud_analytics_alerts_digest"`
	CloudAnalyticsBudgetAlert               string `json:"cloud_analytics_budget_alert"`
	CloudAnalyticsBudgetForecastedDateAlert string `json:"cloud_analytics_budget_forecasted_date_alert"`
	CloudAnalyticsDailyDigest               string `json:"cloud_analytics_daily_digest"`
	CostAnomalyAlert                        string `json:"cost_anomaly_alert"`
	CreditCardPaymentFailed                 string `json:"credit_card_payment_failed"`
	FlexSaveAvailable                       string `json:"flexsave_available"`
	FlexsaveAvailableForAdmins              string `json:"flexsave_available_for_admins"`
	FlexSaveExpiration                      string `json:"flexsave_expiration"`
	FlexsaveWelcome                         string `json:"flexsave_welcome"`
	GoogleCloudNewBillingAccount            string `json:"gcp_new_billing_account"`
	KnownIssuesNotification                 string `json:"known_issues_notification"`
	MonthlyDigest                           string `json:"monthly_digest"`
	NewInvoiceNotification                  string `json:"new_invoice_notification"`
	NoticeToRemedy                          string `json:"notice_to_remedy"`
	OrderConfirmation                       string `json:"order_confirmation"`
	OrderDigest                             string `json:"order_digest"`
	SandboxAccountBudgetAlert               string `json:"sandbox_account_budget_alert"`
	ScheduledCloudReport                    string `json:"scheduled_cloud_report"`
	ServiceLimitNotification                string `json:"service_limit_notification"`
	SimpleNotification                      string `json:"simple_notification"`
	StripeDailyDigest                       string `json:"stripe_daily_digest"`
	UserInviteWithSignup                    string `json:"user_invite_w_signup"`
	UserInvite                              string `json:"user_invite"`
	SpotScalingMarketing                    string `json:"spot_scaling_marketing"`
	PerkRegisterInterest                    string `json:"perk_register_interest"`
	FlexsaveActivation                      string `json:"flexsave_activation"`
}

const (
	CatagoryAnomalies        string = "anomalies"
	CatagoryContractsBreach  string = "contracts-breach"
	CatagoryDigest           string = "digest"
	CatagoryDigestDaily      string = "digest-daily"
	CatagoryDigestMonthly    string = "digest-monthly"
	CatagoryFlexsave         string = "Flexsave"
	CatagoryInvoices         string = "invoices"
	CatagoryInvoicesReminder string = "invoices-reminder"
	CatagoryInvitation       string = "invitation"
	CatagoryReports          string = "reports"
	CatagoryScheduledReports string = "scheduled-reports"
	/* When adding/removing categories, please make sure to sync it with the categories on cloud functions which stored under the mailer.ts file*/
)

// SimpleNotification : Simple notification template data
type SimpleNotification struct {
	Subject    string
	Preheader  string
	Body       string
	CCs        []string
	BCCs       []string
	Attachment string
	TemplateID string
	Categories []string
}

// EntityUpdateNotification : Entity update template data
type EntityUpdateNotification struct {
	Email      string
	Name       string
	EntityName string
	Update     string
}

type InvoiceAttachments struct {
	PdfFile string
	Key     string
}

// Config : Sendgrid configuration
var Config SendGridConfig

func init() {
	ctx := context.Background()

	secretData, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretSendgrid)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.Unmarshal(secretData, &Config)
	if err != nil {
		log.Fatalln(err)
	}
}

func SendSimpleNotification(sn *SimpleNotification, email string) {
	m := mail.NewV3Mail()
	m.SetTemplateID(Config.DynamicTemplates.SimpleNotification)
	m.SetFrom(mail.NewEmail(Config.NoReplyName, Config.NoReplyEmail))

	enable := false
	m.SetTrackingSettings(&mail.TrackingSettings{SubscriptionTracking: &mail.SubscriptionTrackingSetting{Enable: &enable}})

	personalization := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail("", email),
	}
	personalization.AddTos(tos...)

	if len(sn.CCs) > 0 {
		ccs := make([]*mail.Email, 0)

		for _, cc := range sn.CCs {
			if cc != email {
				ccs = append(ccs, mail.NewEmail("", cc))
			}
		}

		if len(ccs) > 0 {
			personalization.AddCCs(ccs...)
		}
	}

	personalization.SetDynamicTemplateData("subject", sn.Subject)
	personalization.SetDynamicTemplateData("preheader", sn.Preheader)
	personalization.SetDynamicTemplateData("body", sn.Body)

	m.AddPersonalizations(personalization)

	request := sendgrid.GetRequest(Config.APIKey, Config.MailSendPath, Config.BaseURL)
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)

	response, err := sendgrid.MakeRequestRetry(request)
	if err != nil {
		log.Println(err)
	} else {
		log.Println(response.StatusCode)
		log.Println(response.Body)
	}
}

func SendEntityUpdateNotification(data *EntityUpdateNotification) error {
	m := mail.NewV3Mail()
	m.SetTemplateID(Config.DynamicTemplates.BillingProfileUpdateNotification)
	m.SetFrom(mail.NewEmail(Config.NoReplyName, Config.NoReplyEmail))

	enable := false
	m.SetTrackingSettings(&mail.TrackingSettings{SubscriptionTracking: &mail.SubscriptionTrackingSetting{Enable: &enable}})

	personalization := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail(data.Name, data.Email),
	}
	personalization.AddTos(tos...)

	ccs := []*mail.Email{
		mail.NewEmail("accounting", "accounting@doit-intl.com"),
	}
	personalization.AddCCs(ccs...)
	personalization.AddBCCs(mail.NewEmail("", "dror+bcc@doit.com"))

	personalization.SetDynamicTemplateData("first_name", data.Name)
	personalization.SetDynamicTemplateData("entity_name", data.EntityName)
	personalization.SetDynamicTemplateData("update", data.Update)

	m.AddPersonalizations(personalization)

	request := sendgrid.GetRequest(Config.APIKey, Config.MailSendPath, Config.BaseURL)
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)

	response, err := sendgrid.MakeRequestRetry(request)
	if err != nil {
		log.Println(err)
		return err
	}

	log.Println(response.StatusCode)
	log.Println(response.Body)

	return nil
}

func SendSimpleEmailWithTemplate(sn *SimpleNotification, emails []string, senderName string, senderEmail string, moreAttachments []InvoiceAttachments, template string) {
	m := mail.NewV3Mail()
	m.SetTemplateID(template)
	m.SetFrom(mail.NewEmail(senderName, senderEmail))

	enable := false
	m.SetTrackingSettings(&mail.TrackingSettings{SubscriptionTracking: &mail.SubscriptionTrackingSetting{Enable: &enable}})

	personalization := mail.NewPersonalization()

	tos := []*mail.Email{}
	for _, email := range emails {
		tos = append(tos, mail.NewEmail("", email))
	}

	personalization.AddTos(tos...)

	if len(sn.CCs) > 0 {
		ccs := make([]*mail.Email, 0)

		for _, cc := range sn.CCs {
			if !slice.Contains(emails, cc) {
				ccs = append(ccs, mail.NewEmail("", cc))
			}
		}

		if len(ccs) > 0 {
			personalization.AddCCs(ccs...)
		}
	}

	personalization.AddBCCs(mail.NewEmail("", "talc@doit-intl.com"))
	personalization.SetDynamicTemplateData("subject", sn.Subject)
	personalization.SetDynamicTemplateData("body", sn.Body)

	m.AddPersonalizations(personalization)

	a := mail.NewAttachment()
	a.SetContent(sn.Attachment)
	a.SetType("application/pdf")
	a.SetFilename("notice_to_remedy_breaches.pdf")
	a.SetDisposition("attachment")
	a.SetContentID("Notice To Remedy Breaches")
	m.AddAttachment(a)

	for _, attachment := range moreAttachments {
		invoicePdf := mail.NewAttachment()
		invoicePdf.SetContent(attachment.PdfFile)
		invoicePdf.SetType("application/pdf")
		invoicePdf.SetFilename(attachment.Key + ".pdf")
		invoicePdf.SetDisposition("attachment")
		invoicePdf.SetContentID(attachment.Key)
		m.AddAttachment(invoicePdf)
	}

	request := sendgrid.GetRequest(Config.APIKey, Config.MailSendPath, Config.BaseURL)
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)

	response, err := sendgrid.MakeRequestRetry(request)
	if err != nil {
		log.Println(err)
	} else {
		log.Println(response.StatusCode)
		log.Println(response.Body)
	}
}

func SendEmailWithTemplate(sn *SimpleNotification, params map[string]interface{}, email string) error {
	m := mail.NewV3Mail()
	m.SetTemplateID(sn.TemplateID)
	m.SetFrom(mail.NewEmail(Config.NoReplyName, Config.NoReplyEmail))

	enable := false
	m.SetTrackingSettings(&mail.TrackingSettings{SubscriptionTracking: &mail.SubscriptionTrackingSetting{Enable: &enable}})
	m.AddCategories(sn.Categories...)

	personalization := mail.NewPersonalization()
	tos := []*mail.Email{
		mail.NewEmail("", email),
	}
	personalization.AddTos(tos...)

	if len(sn.CCs) > 0 {
		ccs := make([]*mail.Email, 0)

		for _, cc := range sn.CCs {
			if cc != email {
				ccs = append(ccs, mail.NewEmail("", cc))
			}
		}

		if len(ccs) > 0 {
			personalization.AddCCs(ccs...)
		}
	}

	if len(sn.BCCs) > 0 {
		bccs := make([]*mail.Email, 0)

		for _, bcc := range sn.BCCs {
			if bcc != email {
				bccs = append(bccs, mail.NewEmail("", bcc))
			}
		}

		if len(bccs) > 0 {
			personalization.AddBCCs(bccs...)
		}
	}

	for key, param := range params {
		personalization.SetDynamicTemplateData(key, param)
	}

	m.AddPersonalizations(personalization)

	request := sendgrid.GetRequest(Config.APIKey, Config.MailSendPath, Config.BaseURL)
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)

	response, err := sendgrid.MakeRequestRetry(request)
	if err != nil {
		return err
	} else {
		log.Println(response.StatusCode)
		log.Println(response.Body)
	}

	return nil
}

func Index(vs []*mail.Email, address string) int {
	for i, v := range vs {
		if v.Address == address {
			return i
		}
	}

	return -1
}

func SendEmailWithPersonalizations(personalizations []*mail.Personalization, templateID string, categories []string) error {
	m := mail.NewV3Mail()
	m.SetFrom(mail.NewEmail(Config.NoReplyName, Config.NoReplyEmail))

	enable := false
	m.SetTrackingSettings(&mail.TrackingSettings{SubscriptionTracking: &mail.SubscriptionTrackingSetting{Enable: &enable}})
	m.SetTemplateID(templateID)
	m.AddPersonalizations(personalizations...)
	m.AddCategories(categories...)

	request := sendgrid.GetRequest(Config.APIKey, Config.MailSendPath, Config.BaseURL)
	request.Method = http.MethodPost
	request.Body = mail.GetRequestBody(m)

	if _, err := sendgrid.MakeRequestRetry(request); err != nil {
		return err
	}

	return nil
}

type Mailer struct {
}

func NewMailer() Mailer {
	return Mailer{}
}

func (Mailer) SendNotification(sn *SimpleNotification, to string, params map[string]interface{}) error {
	err := SendEmailWithTemplate(sn, params, to)
	if err != nil {
		return err
	}

	return nil
}

type CowardMailer struct{}

func (CowardMailer) SendNotification(sn *SimpleNotification, to string, params map[string]interface{}) error {
	marshaledNotification, err := json.Marshal(sn)
	if err != nil {
		return err
	}

	marshaledParams, err := json.Marshal(params)
	if err != nil {
		return err
	}

	fmt.Printf("Coward mailer not sending to %s, %s, with params: %s\n", to, string(marshaledNotification), string(marshaledParams))

	return nil
}
