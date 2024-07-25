package costs

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	bq "github.com/doitintl/hello/scheduled-tasks/spot0/dal/bigquery"
	fs "github.com/doitintl/hello/scheduled-tasks/spot0/dal/firestore"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ASGCustomerEmail struct {
	logger     logger.ILogger
	bqService  bq.ISpot0CostsBigQuery
	fsService  fs.ISpot0CostsFireStore
	aswConnect dal.IAwsConnect
	mailer     IasgCustomerMailer
}

func NewASGCustomerEmail(loggerProvider logger.Provider, conn *connection.Connection) *ASGCustomerEmail {
	ctx := context.Background()

	bqService, err := bq.NewBigQueryService(ctx)
	if err != nil {
		panic(err)
	}

	fsService := fs.NewSpot0CostsFirestoreWithClient(conn.Firestore(ctx))
	aswConnect := dal.NewAwsConnectWithClient(func(ctx context.Context) *firestore.Client { return conn.Firestore(ctx) })
	logger := loggerProvider(ctx)

	return &ASGCustomerEmail{
		logger,
		bqService,
		fsService,
		aswConnect,
		asgCustomerMailer{},
	}
}

// SendMarketingEmails emails customers that are not using Spot-scaling, but have ASGs
// Keeps track of customers that have already been emailed in Firestore
// Limits the amount of customers to send emails to on each execution
// The Email contains a link to schedule a demo, so we don't want to send too many emails at once
func (a ASGCustomerEmail) SendMarketingEmail(ctx context.Context, maxCustomersPerTask int, minDaysOnboarded int) error {
	onboardedBefore := time.Now().AddDate(0, 0, -minDaysOnboarded)

	primaryDomains, err := a.bqService.GetDomainsWithASGs(ctx)
	if err != nil {
		return err
	}

	customerCount := 0
	for _, pd := range primaryDomains {
		// get customer to send email to
		if customerCount >= maxCustomersPerTask {
			break
		}

		customer, err := a.fsService.GetCustomerFromPrimaryDomain(ctx, pd)
		if err != nil {
			return err
		}

		usingSpotScaling, err := a.fsService.CustomerIsUsingSpotScaling(ctx, customer)
		if err != nil {
			return err
		}

		if usingSpotScaling {
			continue
		}
		// only new customers have the onboardingDate data
		// if onboardingDate is not found (old customer), it will be set to 0001-01-01T00:00:00Z
		onboardingDate, err := a.fsService.GetCustomerTimeCreated(ctx, customer)
		if err != nil {
			a.logger.Debug("error getting customer onboarding date: ", customer.ID, err)
		}

		if !onboardingDate.Before(onboardedBefore) {
			continue
		}

		// send emails to customer's admin users with AMs in BCC
		admins, err := a.aswConnect.GetCustomerAdmins(ctx, customer.ID)
		if err != nil {
			return err
		}

		AMs, err := a.fsService.GetCustomerAMs(ctx, customer)
		if err != nil {
			return err
		}

		sent, err := a.emailTransaction(ctx, customer, admins, AMs)
		if err != nil {
			return err
		}

		if sent {
			customerCount++
		}
	}

	return nil
}

// emailTransaction sends emails and updates Firestore
func (a ASGCustomerEmail) emailTransaction(ctx context.Context, customer *firestore.DocumentRef, admins []common.User, AMs []common.AccountManager) (bool, error) {
	added, err := a.fsService.AddASGCustomerToList(ctx, customer)
	if err != nil || !added {
		return false, err
	}

	a.logger.Info("Sending Email to ASG customer ", customer.ID)

	err = a.mailer.SendEmails(ctx, admins, AMs, common.Production, mailer.SendEmailWithPersonalizations)
	if err != nil {
		deleteErr := a.fsService.DeleteASGCustomerFromList(ctx, customer)
		if deleteErr != nil {
			a.logger.Error("email not sent, error deleting ASG customer from list: ", customer.ID, deleteErr)
		}

		return false, err
	}

	return true, nil
}

type IasgCustomerMailer interface {
	SendEmails(
		ctx context.Context,
		users []common.User,
		AMs []common.AccountManager,
		prod bool,
		sendEmailWithPersonalizations func([]*mail.Personalization, string, []string) error,
	) error
}

type asgCustomerMailer struct{}

// SendEmails sends emails to users, with AMs in BCC, using SendGrid's dynamic template
func (m asgCustomerMailer) SendEmails(
	ctx context.Context,
	users []common.User,
	AMs []common.AccountManager,
	prod bool,
	sendEmailWithPersonalizations func([]*mail.Personalization, string, []string) error,
) error {
	personalizations := m.GetEmailPersonalizations(ctx, users, AMs)
	templateID := mailer.Config.DynamicTemplates.SpotScalingMarketing
	categories := []string{"spot-scaling"}

	if !prod {
		return nil
	}

	return sendEmailWithPersonalizations(personalizations, templateID, categories)
}

// GetEmailPersonalizations returns a Sendgrid personalization object for each user
func (m asgCustomerMailer) GetEmailPersonalizations(ctx context.Context, users []common.User, AMs []common.AccountManager) []*mail.Personalization {
	AMsEmails := make([]*mail.Email, len(AMs))
	for i, AM := range AMs {
		AMsEmails[i] = mail.NewEmail(AM.Name, AM.Email)
	}

	var personalizations []*mail.Personalization

	for _, user := range users {
		p := mail.NewPersonalization()
		p.AddTos(mail.NewEmail(user.DisplayName, user.Email))
		p.AddBCCs(AMsEmails...)
		p.AddBCCs(mail.NewEmail("Francisco de la Cortina", "francisco@doit-intl.com"))
		p.SetDynamicTemplateData("firstName", cases.Title(language.English).String(user.FirstName))
		p.SetDynamicTemplateData("company", user.Customer.Name)
		personalizations = append(personalizations, p)
	}

	return personalizations
}
