package costs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	aswConnectMock "github.com/doitintl/hello/scheduled-tasks/cloudconnect/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	loggerMock "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	mailerMock "github.com/doitintl/hello/scheduled-tasks/spot0/costs/mocks"
	bigQueryCostMock "github.com/doitintl/hello/scheduled-tasks/spot0/dal/bigquery/mocks"
	fireStoreCostMock "github.com/doitintl/hello/scheduled-tasks/spot0/dal/firestore/mocks"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestASGCustomerEmail_SendMarketingEmail(t *testing.T) {
	type fields struct {
		logger     *loggerMock.ILogger
		bqService  *bigQueryCostMock.ISpot0CostsBigQuery
		fsService  *fireStoreCostMock.ISpot0CostsFireStore
		aswConnect *aswConnectMock.IAwsConnect
		mailer     *mailerMock.IasgCustomerMailer
	}

	type args struct {
		ctx                 context.Context
		maxCustomersPerTask int
		minDaysOnboarded    int
	}

	tests := []struct {
		name    string
		fields  fields
		on      func(*fields)
		args    args
		wantErr error
	}{
		{
			name: "happy path",
			fields: fields{
				logger:     &loggerMock.ILogger{},
				fsService:  fireStoreCostMock.NewISpot0CostsFireStore(t),
				bqService:  bigQueryCostMock.NewISpot0CostsBigQuery(t),
				aswConnect: aswConnectMock.NewIAwsConnect(t),
				mailer:     mailerMock.NewIasgCustomerMailer(t),
			},
			on: func(f *fields) {
				f.bqService.On("GetDomainsWithASGs", mock.Anything).
					Return([]string{"domain1.com", "domain2.com", "domain3.com"}, nil)

				// customer 1 is already using spot scaling
				customer1 := firestore.DocumentRef{Path: "customers/1", ID: "1"}
				f.fsService.On("GetCustomerFromPrimaryDomain", mock.Anything, "domain1.com").
					Return(&customer1, nil)
				f.fsService.On("CustomerIsUsingSpotScaling", mock.Anything, &customer1).
					Return(true, nil)

				// customer 2 has already received the email
				customer2 := firestore.DocumentRef{Path: "customers/2", ID: "2"}
				f.fsService.On("GetCustomerFromPrimaryDomain", mock.Anything, "domain2.com").
					Return(&customer2, nil)
				f.fsService.On("CustomerIsUsingSpotScaling", mock.Anything, &customer2).
					Return(false, nil)
				f.fsService.On("GetCustomerTimeCreated", mock.Anything, &customer2).
					Return(time.Now().AddDate(0, 0, -60), nil)
				admins := []common.User{{DisplayName: "customer3_admin", Email: "email@domain3.com"}}
				f.aswConnect.On("GetCustomerAdmins", mock.Anything, customer2.ID).
					Return(admins, nil)
				AMs := []common.AccountManager{{Name: "customer3_am", Email: "customer3_AM@doit.com"}}
				f.fsService.On("GetCustomerAMs", mock.Anything, &customer2).
					Return(AMs, nil)
				f.fsService.On("AddASGCustomerToList", mock.Anything, &customer2).
					Return(false, nil)

				// customer 3 is a new ASG customer
				customer3 := firestore.DocumentRef{Path: "customers/3", ID: "3"}
				f.fsService.On("GetCustomerFromPrimaryDomain", mock.Anything, "domain3.com").
					Return(&customer3, nil)
				f.fsService.On("CustomerIsUsingSpotScaling", mock.Anything, &customer3).
					Return(false, nil)
				f.fsService.On("GetCustomerTimeCreated", mock.Anything, &customer3).
					Return(time.Now().AddDate(0, 0, -60), nil)
				admins = []common.User{{DisplayName: "customer3_admin", Email: "email@domain3.com"}}
				f.aswConnect.On("GetCustomerAdmins", mock.Anything, customer3.ID).
					Return(admins, nil)
				AMs = []common.AccountManager{{Name: "customer3_am", Email: "customer3_AM@doit.com"}}
				f.fsService.On("GetCustomerAMs", mock.Anything, &customer3).
					Return(AMs, nil)
				f.fsService.On("AddASGCustomerToList", mock.Anything, &customer3).
					Return(true, nil)
				f.logger.On("Info", "Sending Email to ASG customer ", customer3.ID).
					Return(nil)
				f.mailer.On("SendEmails", mock.Anything, admins, AMs, false, mock.Anything).
					Return(nil)
			},
			args: args{
				ctx:                 context.Background(),
				maxCustomersPerTask: 5,
				minDaysOnboarded:    30,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.on != nil {
				tt.on(&tt.fields)
			}

			h := ASGCustomerEmail{
				logger:     tt.fields.logger,
				bqService:  tt.fields.bqService,
				fsService:  tt.fields.fsService,
				aswConnect: tt.fields.aswConnect,
				mailer:     tt.fields.mailer,
			}
			err := h.SendMarketingEmail(tt.args.ctx, tt.args.maxCustomersPerTask, tt.args.minDaysOnboarded)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestASGCustomerEmail_emailTransaction(t *testing.T) {
	type fields struct {
		logger     *loggerMock.ILogger
		bqService  *bigQueryCostMock.ISpot0CostsBigQuery
		fsService  *fireStoreCostMock.ISpot0CostsFireStore
		aswConnect *aswConnectMock.IAwsConnect
		mailer     *mailerMock.IasgCustomerMailer
	}

	type args struct {
		ctx      context.Context
		customer *firestore.DocumentRef
		admins   []common.User
		AMs      []common.AccountManager
	}

	tests := []struct {
		name    string
		fields  fields
		on      func(*fields)
		args    args
		wantErr error
		want    bool
	}{
		{
			name: "Customer is already on the list",
			fields: fields{
				fsService: fireStoreCostMock.NewISpot0CostsFireStore(t),
			},
			on: func(f *fields) {
				f.fsService.On("AddASGCustomerToList", mock.Anything, mock.Anything).
					Return(false, nil)
			},
			args:    args{ctx: context.Background(), customer: &firestore.DocumentRef{Path: "customers/1", ID: "1"}},
			wantErr: nil,
			want:    false,
		},
		{
			name: "Customer is not on the list",
			fields: fields{
				fsService: fireStoreCostMock.NewISpot0CostsFireStore(t),
				mailer:    mailerMock.NewIasgCustomerMailer(t),
				logger:    &loggerMock.ILogger{},
			},
			on: func(f *fields) {
				f.fsService.On("AddASGCustomerToList", mock.Anything, mock.Anything).
					Return(true, nil)
				f.mailer.On("SendEmails", mock.Anything, mock.Anything, mock.Anything, false, mock.Anything).
					Return(nil)
				f.logger.On("Info", "Sending Email to ASG customer ", mock.Anything).
					Return(nil)
			},
			args:    args{ctx: context.Background(), customer: &firestore.DocumentRef{Path: "customers/1", ID: "1"}},
			wantErr: nil,
			want:    true,
		},
		{
			name: "Customer is not on the list, error sending email",
			fields: fields{
				fsService: fireStoreCostMock.NewISpot0CostsFireStore(t),
				mailer:    mailerMock.NewIasgCustomerMailer(t),
				logger:    &loggerMock.ILogger{},
			},
			on: func(f *fields) {
				f.fsService.On("AddASGCustomerToList", mock.Anything, mock.Anything).
					Return(true, nil)
				f.logger.On("Info", "Sending Email to ASG customer ", mock.Anything).
					Return(nil)
				f.mailer.On("SendEmails", mock.Anything, mock.Anything, mock.Anything, false, mock.Anything).
					Return(fmt.Errorf("error sending email"))
				f.fsService.On("DeleteASGCustomerFromList", mock.Anything, mock.Anything).
					Return(nil)
			},
			args:    args{ctx: context.Background(), customer: &firestore.DocumentRef{Path: "customers/1", ID: "1"}},
			wantErr: fmt.Errorf("error sending email"),
			want:    false,
		},
		{
			name: "Customer is not on the list, error sending email, error deleting customer from list",
			fields: fields{
				fsService: fireStoreCostMock.NewISpot0CostsFireStore(t),
				mailer:    mailerMock.NewIasgCustomerMailer(t),
				logger:    &loggerMock.ILogger{},
			},
			on: func(f *fields) {
				f.fsService.On("AddASGCustomerToList", mock.Anything, mock.Anything).
					Return(true, nil)
				f.logger.On("Info", "Sending Email to ASG customer ", mock.Anything).
					Return(nil)
				f.mailer.On("SendEmails", mock.Anything, mock.Anything, mock.Anything, false, mock.Anything).
					Return(fmt.Errorf("error sending email"))
				f.fsService.On("DeleteASGCustomerFromList", mock.Anything, mock.Anything).
					Return(fmt.Errorf("error deleting customer from list"))
				f.logger.On("Error", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			args:    args{ctx: context.Background(), customer: &firestore.DocumentRef{Path: "customers/1", ID: "1"}},
			wantErr: fmt.Errorf("error sending email"),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.on != nil {
				tt.on(&tt.fields)
			}

			a := ASGCustomerEmail{
				logger:     tt.fields.logger,
				bqService:  tt.fields.bqService,
				fsService:  tt.fields.fsService,
				aswConnect: tt.fields.aswConnect,
				mailer:     tt.fields.mailer,
			}

			got, err := a.emailTransaction(tt.args.ctx, tt.args.customer, tt.args.admins, tt.args.AMs)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if got != tt.want {
				t.Errorf("ASGCustomerEmail.emailTransaction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_asgCustomerMailer_GetEmailPersonalizations(t *testing.T) {
	users := []common.User{
		{DisplayName: "admin1", Email: "admin1@domain.com", FirstName: "first", Customer: common.UserCustomer{Name: "company"}},
	}
	AMs := []common.AccountManager{
		{Name: "AM1", Email: "AM1@doit.com"},
		{Name: "AM2", Email: "AM2@doit.com"},
	}
	m := asgCustomerMailer{}

	pers := m.GetEmailPersonalizations(context.Background(), users, AMs)
	if len(pers) != 1 {
		t.Errorf("expected 1 personalizations, got %d", len(pers))
	}

	if len(pers[0].To) != 1 {
		t.Errorf("expected 1 to, got %d", len(pers[0].To))
	}

	to := pers[0].To[0]
	if to.Address != "admin1@domain.com" {
		t.Errorf("wrong address: %s", to.Address)
	}

	if len(pers[0].BCC) != 3 {
		t.Errorf("expected 3 bcc, got %d", len(pers[0].BCC))
	}

	if pers[0].BCC[0].Address != "AM1@doit.com" {
		t.Errorf("wrong address: %s", pers[0].BCC[0].Address)
	}

	if pers[0].BCC[1].Address != "AM2@doit.com" {
		t.Errorf("wrong address: %s", pers[0].BCC[1].Address)
	}

	if pers[0].DynamicTemplateData["firstName"] != "First" {
		t.Errorf("wrong first name: %s", pers[0].DynamicTemplateData["firstName"])
	}

	if pers[0].DynamicTemplateData["company"] != "company" {
		t.Errorf("wrong company: %s", pers[0].DynamicTemplateData["company"])
	}
}

func Test_asgCustomerMailer_SendEmails(t *testing.T) {
	type args struct {
		ctx                           context.Context
		users                         []common.User
		AMs                           []common.AccountManager
		prod                          bool
		sendEmailWithPersonalizations func([]*mail.Personalization, string, []string) error
	}

	tests := []struct {
		name    string
		m       asgCustomerMailer
		args    args
		wantErr error
	}{
		{
			name: "email should not be sent on dev env",
			m:    asgCustomerMailer{},
			args: args{
				prod: false,
				sendEmailWithPersonalizations: func([]*mail.Personalization, string, []string) error {
					t.Errorf("should not be called")
					return nil
				},
			},
			wantErr: nil,
		},
		{
			name: "email should be sent on prod env",
			m:    asgCustomerMailer{},
			args: args{
				prod: true,
				sendEmailWithPersonalizations: func([]*mail.Personalization, string, []string) error {
					return fmt.Errorf("called and returned error")
				},
			},
			wantErr: fmt.Errorf("called and returned error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := asgCustomerMailer{}
			err := m.SendEmails(tt.args.ctx, tt.args.users, tt.args.AMs, tt.args.prod, tt.args.sendEmailWithPersonalizations)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
