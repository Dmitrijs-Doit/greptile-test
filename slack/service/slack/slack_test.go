package slack

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	slackgo "github.com/slack-go/slack"
	"github.com/stretchr/testify/mock"

	doitFirestore "github.com/doitintl/firestore"
	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	reportPkg "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/firebase/tenant"
	tenantMocks "github.com/doitintl/hello/scheduled-tasks/firebase/tenant/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/slack/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/slack/domain"
	budgetsMocks "github.com/doitintl/hello/scheduled-tasks/slack/service/budgets/mocks"
	reportsMocks "github.com/doitintl/hello/scheduled-tasks/slack/service/reports/mocks"
	"github.com/doitintl/slackapi"
	"github.com/doitintl/slackapi/dynamicclient"
	slackapiMocks "github.com/doitintl/slackapi/mocks"
)

type fields struct {
	loggerProvider *loggerMocks.ILogger
	Connection     *connection.Connection
	firestoreDAL   *mocks.IFirestoreDAL
	slackDAL       *mocks.ISlackDAL
	budgetsService *budgetsMocks.IBudgetsService
	reportsService *reportsMocks.IReportsService
	tenantService  *tenant.TenantService
	doitsyBotToken string
	API            *slackapi.SlackAPI
}

func initFields(t *testing.T) (*fields, error) {
	tenantService, err := tenant.NewTenantsServiceWithDalClient(&tenantMocks.Tenants{})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	logging, err := logger.NewLogging(ctx)
	if err != nil {
		return nil, fmt.Errorf("main: could not initialize logging. error %s", err)
	}

	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		return nil, fmt.Errorf("main: could not initialize db connections. error %s", err)
	}

	return &fields{
		loggerProvider: &loggerMocks.ILogger{},
		Connection:     conn,
		firestoreDAL:   &mocks.IFirestoreDAL{},
		slackDAL:       &mocks.ISlackDAL{},
		budgetsService: &budgetsMocks.IBudgetsService{},
		reportsService: &reportsMocks.IReportsService{},
		tenantService:  tenantService,
		doitsyBotToken: "",
		API: &slackapi.SlackAPI{
			Client:       &dynamicclient.SlackDynamicClient{},
			DoitsyClient: slackapiMocks.NewISlackClient(t),
			YallaClient:  slackapiMocks.NewISlackClient(t),
		},
	}, nil
}

func initService(fields *fields) *SlackService {
	return &SlackService{
		loggerProvider: func(ctx context.Context) logger.ILogger {
			return fields.loggerProvider
		},
		Connection:     fields.Connection,
		firestoreDAL:   fields.firestoreDAL,
		slackDAL:       fields.slackDAL,
		budgetsService: fields.budgetsService,
		reportsService: fields.reportsService,
		tenantService:  fields.tenantService,
		doitsyBotToken: fields.doitsyBotToken,
		API:            fields.API,
	}
}

func TestSlackService_HandleLinkSharedEvent(t *testing.T) {
	type args struct {
		ctx context.Context
		req *domain.SlackRequest
	}

	ctx := context.Background()

	getRequest := func(url string) *domain.SlackRequest {
		return &domain.SlackRequest{
			Challenge: "",
			TeamID:    "workspaceDoit",
			Event: domain.SlackEvent{
				Type:      "link_shared",
				Channel:   "channel name",
				User:      "U010TS64Z0R",
				MessageTs: "123452389.9875",
				Links: []domain.SlackLink{{
					URL: url,
				}},
			},
		}
	}

	reportURL := "https://console.app.doit.com/customers/ABCDE123456789/analytics/reports/mrV8lKa7USq4NFVsPIhd"
	budgetURL := "https://console.app.doit.com/customers/ABCDE123456789/analytics/budgets/mrV8lKa7USq4NFVsPIhd"
	invalidURL := "https://console.app.doit.com/customers/ABCDE123456789"

	doer := "ofir.cohen@doit.com"
	user := "tupac.shakur@customer.com"
	user2 := "biggie.smalls@customer.com"
	customerRef := &firestore.DocumentRef{
		ID: "ABCDE123456789",
	}
	wrongCustomerRef := &firestore.DocumentRef{
		ID: "this-is-wrong",
	}

	budget := &budget.Budget{
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{
					Email: user,
					Role:  collab.CollaboratorRoleEditor,
				},
				{
					Email: doer,
					Role:  collab.CollaboratorRoleOwner,
				}},
			Public: nil,
		},
		Name: "budget name",
	}

	report := &reportPkg.Report{
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{
					Email: user,
					Role:  collab.CollaboratorRoleEditor,
				},
				{
					Email: doer,
					Role:  collab.CollaboratorRoleOwner,
				}},
		},
		Name: "report name",
	}

	tests := []struct {
		name    string
		args    *args
		on      func(*fields)
		want    *domain.MixpanelProperties
		wantErr bool
		assert  func(*testing.T, *fields)
	}{
		{
			name: "valid: type - report √  sender - doer √ role - owner √ slack channel - private √",
			args: &args{
				ctx, getRequest(reportURL),
			},
			on: func(f *fields) {
				f.loggerProvider.
					On("Infof", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				f.slackDAL.
					On("GetUserEmail", mock.Anything, mock.Anything, mock.Anything).
					Return(doer, nil).
					Once().
					On("SendUnfurl", mock.Anything, mock.Anything).
					Return(nil).
					On("SendUnfurlWithEphemeral", mock.Anything, mock.Anything).
					Return(nil).
					On("GetAllChannelMemberEmails", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("channel not found")).
					On("GetChannelInfo", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("channel not found"))
				f.budgetsService.
					On("GetUnfurlPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, nil, nil).
					Once().
					On("UpdateSharing", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				f.reportsService.
					On("GetUnfurlPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(report, nil, nil).
					Once().
					On("UpdateSharing", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				f.firestoreDAL.
					On("GetDoitEmployee", mock.Anything, mock.Anything).
					Return(&firestorePkg.User{
						Customer: firestorePkg.UserCustomer{
							Ref: customerRef,
						},
					}, nil).
					On("GetSharedChannel", mock.Anything, mock.Anything).
					Return(nil, doitFirestore.ErrNotFound).
					On("GetWorkspaceDecrypted", mock.Anything, mock.Anything).
					Return(&firestorePkg.SlackWorkspace{
						Name: "workspace-name",
					}, "", "", "", nil)
			},
			want:    nil,
			wantErr: false,
			assert: func(t *testing.T, f *fields) {
				f.slackDAL.AssertNumberOfCalls(t, "GetUserEmail", 1)
				f.slackDAL.AssertNumberOfCalls(t, "SendUnfurl", 2)
				f.slackDAL.AssertNumberOfCalls(t, "SendUnfurlWithEphemeral", 1)
				f.slackDAL.AssertNumberOfCalls(t, "GetAllChannelMemberEmails", 1)
				f.slackDAL.AssertNumberOfCalls(t, "GetChannelInfo", 1)
				f.budgetsService.AssertNumberOfCalls(t, "GetUnfurlPayload", 0)
				f.budgetsService.AssertNumberOfCalls(t, "UpdateSharing", 0)
				f.reportsService.AssertNumberOfCalls(t, "GetUnfurlPayload", 1)
				f.reportsService.AssertNumberOfCalls(t, "UpdateSharing", 0)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetDoitEmployee", 1)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetSharedChannel", 1)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetWorkspaceDecrypted", 1)
			},
		},
		{
			name: "valid: type - budget √ sender - user √ role - editor √ slack channel - public √",
			args: &args{
				ctx, getRequest(budgetURL),
			},
			on: func(f *fields) {
				f.loggerProvider.
					On("Infof", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				f.slackDAL.
					On("GetUserEmail", mock.Anything, mock.Anything, mock.Anything).
					Return(user, nil).
					Once().
					On("SendUnfurl", mock.Anything, mock.Anything).
					Return(nil).
					On("SendUnfurlWithEphemeral", mock.Anything, mock.Anything).
					Return(nil).
					On("GetAllChannelMemberEmails", mock.Anything, mock.Anything, mock.Anything).
					Return([]string{user2, user, doer}, nil).
					On("GetChannelInfo", mock.Anything, mock.Anything, mock.Anything).
					Return(&slackgo.Channel{GroupConversation: slackgo.GroupConversation{Name: "public channel"}}, nil)
				f.budgetsService.
					On("GetUnfurlPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(budget, nil, nil).
					Once().
					On("UpdateSharing", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				f.reportsService.
					On("GetUnfurlPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(report, nil, nil).
					Once().
					On("UpdateSharing", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				f.firestoreDAL.
					On("GetDoitEmployee", mock.Anything, mock.Anything).
					Return(nil, doitFirestore.ErrNotFound).
					On("GetUser", mock.Anything, mock.Anything).
					Return(&firestorePkg.User{
						Customer: firestorePkg.UserCustomer{
							Ref: customerRef,
						},
					}, nil).
					On("UserHasCloudAnalyticsPermission", mock.Anything, mock.Anything).
					Return(true, nil).
					On("GetSharedChannel", mock.Anything, mock.Anything).
					Return(nil, doitFirestore.ErrNotFound).
					On("GetWorkspaceDecrypted", mock.Anything, mock.Anything).
					Return(&firestorePkg.SlackWorkspace{
						Name: "workspace-name",
					}, "", "", "", nil)
			},
			want:    nil,
			wantErr: false,
			assert: func(t *testing.T, f *fields) {
				f.slackDAL.AssertNumberOfCalls(t, "GetUserEmail", 1)
				f.slackDAL.AssertNumberOfCalls(t, "SendUnfurl", 2)
				f.slackDAL.AssertNumberOfCalls(t, "SendUnfurlWithEphemeral", 1)
				f.slackDAL.AssertNumberOfCalls(t, "GetAllChannelMemberEmails", 1)
				f.slackDAL.AssertNumberOfCalls(t, "GetChannelInfo", 1)
				f.budgetsService.AssertNumberOfCalls(t, "GetUnfurlPayload", 1)
				f.budgetsService.AssertNumberOfCalls(t, "UpdateSharing", 0)
				f.reportsService.AssertNumberOfCalls(t, "GetUnfurlPayload", 0)
				f.reportsService.AssertNumberOfCalls(t, "UpdateSharing", 0)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetDoitEmployee", 0)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetSharedChannel", 1)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetWorkspaceDecrypted", 1)
			},
		},
		{
			name: "valid: type - report √ sender - user √ role - editor √ slack channel - shared √",
			args: &args{
				ctx, getRequest(reportURL),
			},
			on: func(f *fields) {
				f.loggerProvider.
					On("Infof", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				f.slackDAL.
					On("GetUserEmail", mock.Anything, mock.Anything, mock.Anything).
					Return(user, nil).
					Once().
					On("SendUnfurl", mock.Anything, mock.Anything).
					Return(nil).
					On("SendUnfurlWithEphemeral", mock.Anything, mock.Anything).
					Return(nil).
					On("GetAllChannelMemberEmails", mock.Anything, mock.Anything, mock.Anything).
					Return([]string{user2, user, doer}, nil).
					On("GetChannelInfo", mock.Anything, mock.Anything, mock.Anything).
					Return(&slackgo.Channel{GroupConversation: slackgo.GroupConversation{Name: "public channel"}}, nil)
				f.budgetsService.
					On("GetUnfurlPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(budget, nil, nil).
					Once().
					On("UpdateSharing", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				f.reportsService.
					On("GetUnfurlPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(report, nil, nil).
					Once().
					On("UpdateSharing", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				f.firestoreDAL.
					On("GetDoitEmployee", mock.Anything, mock.Anything).
					Return(nil, doitFirestore.ErrNotFound).
					On("GetUser", mock.Anything, mock.Anything).
					Return(&firestorePkg.User{
						Customer: firestorePkg.UserCustomer{
							Ref: customerRef,
						},
					}, nil).
					On("UserHasCloudAnalyticsPermission", mock.Anything, mock.Anything).
					Return(true, nil).
					On("GetSharedChannel", mock.Anything, mock.Anything).
					Return(&firestorePkg.SharedChannel{
						Customer: customerRef,
					}, nil).
					On("GetWorkspaceDecrypted", mock.Anything, mock.Anything).
					Return(&firestorePkg.SlackWorkspace{
						Name: "workspace-name",
					}, "", "", "", nil)
			},
			want:    nil,
			wantErr: false,
			assert: func(t *testing.T, f *fields) {
				f.slackDAL.AssertNumberOfCalls(t, "GetUserEmail", 1)
				f.slackDAL.AssertNumberOfCalls(t, "SendUnfurl", 2)
				f.slackDAL.AssertNumberOfCalls(t, "SendUnfurlWithEphemeral", 1)
				f.slackDAL.AssertNumberOfCalls(t, "GetAllChannelMemberEmails", 1)
				f.slackDAL.AssertNumberOfCalls(t, "GetChannelInfo", 1)
				f.budgetsService.AssertNumberOfCalls(t, "GetUnfurlPayload", 0)
				f.budgetsService.AssertNumberOfCalls(t, "UpdateSharing", 0)
				f.reportsService.AssertNumberOfCalls(t, "GetUnfurlPayload", 1)
				f.reportsService.AssertNumberOfCalls(t, "UpdateSharing", 0)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetDoitEmployee", 0)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetSharedChannel", 1)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetWorkspaceDecrypted", 1)
			},
		},
		{
			name: "invalid: type - invalid link X sender - user √ role - editor √ slack channel - shared √",
			args: &args{
				ctx, getRequest(invalidURL),
			},
			on: func(f *fields) {
				f.loggerProvider.
					On("Infof", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					On("Warningf", mock.Anything, mock.Anything, mock.Anything)
				f.slackDAL.
					On("GetUserEmail", mock.Anything, mock.Anything, mock.Anything).
					Return(user, nil).
					Once().
					On("SendUnfurl", mock.Anything, mock.Anything).
					Return(nil).
					On("SendUnfurlWithEphemeral", mock.Anything, mock.Anything).
					Return(nil).
					On("GetAllChannelMemberEmails", mock.Anything, mock.Anything, mock.Anything).
					Return([]string{user2, user, doer}, nil).
					On("GetChannelInfo", mock.Anything, mock.Anything, mock.Anything).
					Return(&slackgo.Channel{GroupConversation: slackgo.GroupConversation{Name: "public channel"}}, nil)
				f.budgetsService.
					On("GetUnfurlPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(budget, nil, nil).
					Once().
					On("UpdateSharing", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				f.reportsService.
					On("GetUnfurlPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(report, nil, nil).
					Once().
					On("UpdateSharing", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				f.firestoreDAL.
					On("GetDoitEmployee", mock.Anything, mock.Anything).
					Return(nil, doitFirestore.ErrNotFound).
					On("GetUser", mock.Anything, mock.Anything).
					Return(&firestorePkg.User{
						Customer: firestorePkg.UserCustomer{
							Ref: customerRef,
						},
					}, nil).
					On("UserHasCloudAnalyticsPermission", mock.Anything, mock.Anything).
					Return(true, nil).
					On("GetSharedChannel", mock.Anything, mock.Anything).
					Return(&firestorePkg.SharedChannel{
						Customer: customerRef,
					}, nil).
					On("GetWorkspaceDecrypted", mock.Anything, mock.Anything).
					Return(&firestorePkg.SlackWorkspace{
						Name: "workspace-name",
					}, "", "", "", nil)
			},
			want:    nil,
			wantErr: false,
			assert: func(t *testing.T, f *fields) {
				f.slackDAL.AssertNumberOfCalls(t, "GetUserEmail", 0)
				f.slackDAL.AssertNumberOfCalls(t, "SendUnfurl", 0)
				f.slackDAL.AssertNumberOfCalls(t, "SendUnfurlWithEphemeral", 0)
				f.slackDAL.AssertNumberOfCalls(t, "GetAllChannelMemberEmails", 0)
				f.slackDAL.AssertNumberOfCalls(t, "GetChannelInfo", 0)
				f.budgetsService.AssertNumberOfCalls(t, "GetUnfurlPayload", 0)
				f.budgetsService.AssertNumberOfCalls(t, "UpdateSharing", 0)
				f.reportsService.AssertNumberOfCalls(t, "GetUnfurlPayload", 0)
				f.reportsService.AssertNumberOfCalls(t, "UpdateSharing", 0)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetDoitEmployee", 0)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetSharedChannel", 0)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetWorkspaceDecrypted", 0)
			},
		},
		{
			name: "invalid: type - budget √ sender - bad user X role - editor √ slack channel - shared √",
			args: &args{
				ctx, getRequest(budgetURL),
			},
			on: func(f *fields) {
				f.loggerProvider.
					On("Infof", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					On("Error", mock.Anything)
				f.slackDAL.
					On("GetUserEmail", mock.Anything, mock.Anything, mock.Anything).
					Return(user, nil).
					Once().
					On("SendUnfurl", mock.Anything, mock.Anything).
					Return(nil).
					On("SendUnfurlWithEphemeral", mock.Anything, mock.Anything).
					Return(nil).
					On("GetAllChannelMemberEmails", mock.Anything, mock.Anything, mock.Anything).
					Return([]string{user2, user, doer}, nil).
					On("GetChannelInfo", mock.Anything, mock.Anything, mock.Anything).
					Return(&slackgo.Channel{GroupConversation: slackgo.GroupConversation{Name: "public channel"}}, nil)
				f.budgetsService.
					On("GetUnfurlPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(budget, nil, nil).
					Once().
					On("UpdateSharing", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				f.reportsService.
					On("GetUnfurlPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(report, nil, nil).
					Once().
					On("UpdateSharing", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				f.firestoreDAL.
					On("GetDoitEmployee", mock.Anything, mock.Anything).
					Return(nil, doitFirestore.ErrNotFound).
					On("GetUser", mock.Anything, mock.Anything).
					Return(&firestorePkg.User{
						Customer: firestorePkg.UserCustomer{
							Ref: wrongCustomerRef,
						},
					}, nil).
					On("UserHasCloudAnalyticsPermission", mock.Anything, mock.Anything).
					Return(true, nil).
					On("GetSharedChannel", mock.Anything, mock.Anything).
					Return(&firestorePkg.SharedChannel{
						Customer: customerRef,
					}, nil).
					On("GetWorkspaceDecrypted", mock.Anything, mock.Anything).
					Return(&firestorePkg.SlackWorkspace{
						Name: "workspace-name",
					}, "", "", "", nil)
			},
			want:    nil,
			wantErr: true,
			assert: func(t *testing.T, f *fields) {
				f.slackDAL.AssertNumberOfCalls(t, "GetUserEmail", 1)
				f.slackDAL.AssertNumberOfCalls(t, "SendUnfurl", 2)
				f.slackDAL.AssertNumberOfCalls(t, "SendUnfurlWithEphemeral", 0)
				f.slackDAL.AssertNumberOfCalls(t, "GetAllChannelMemberEmails", 0)
				f.slackDAL.AssertNumberOfCalls(t, "GetChannelInfo", 0)
				f.budgetsService.AssertNumberOfCalls(t, "GetUnfurlPayload", 1)
				f.budgetsService.AssertNumberOfCalls(t, "UpdateSharing", 0)
				f.reportsService.AssertNumberOfCalls(t, "GetUnfurlPayload", 0)
				f.reportsService.AssertNumberOfCalls(t, "UpdateSharing", 0)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetDoitEmployee", 0)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetSharedChannel", 0)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetWorkspaceDecrypted", 0)
			},
		},
		{
			name: "invalid: type - budget √ sender - user √ role - editor √ slack channel - bad shared x",
			args: &args{
				ctx, getRequest(budgetURL),
			},
			on: func(f *fields) {
				f.loggerProvider.
					On("Infof", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					On("Error", mock.Anything)
				f.slackDAL.
					On("GetUserEmail", mock.Anything, mock.Anything, mock.Anything).
					Return(user, nil).
					Once().
					On("SendUnfurl", mock.Anything, mock.Anything).
					Return(nil).
					On("SendUnfurlWithEphemeral", mock.Anything, mock.Anything).
					Return(nil).
					On("GetAllChannelMemberEmails", mock.Anything, mock.Anything, mock.Anything).
					Return([]string{user2, user, doer}, nil).
					On("GetChannelInfo", mock.Anything, mock.Anything, mock.Anything).
					Return(&slackgo.Channel{GroupConversation: slackgo.GroupConversation{Name: "public channel"}}, nil)
				f.budgetsService.
					On("GetUnfurlPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(budget, nil, nil).
					Once().
					On("UpdateSharing", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				f.reportsService.
					On("GetUnfurlPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(report, nil, nil).
					Once().
					On("UpdateSharing", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				f.firestoreDAL.
					On("GetDoitEmployee", mock.Anything, mock.Anything).
					Return(nil, doitFirestore.ErrNotFound).
					On("GetUser", mock.Anything, mock.Anything).
					Return(&firestorePkg.User{
						Customer: firestorePkg.UserCustomer{
							Ref: customerRef,
						},
					}, nil).
					On("UserHasCloudAnalyticsPermission", mock.Anything, mock.Anything).
					Return(true, nil).
					On("GetSharedChannel", mock.Anything, mock.Anything).
					Return(&firestorePkg.SharedChannel{
						Customer: wrongCustomerRef,
					}, nil).
					On("GetWorkspaceDecrypted", mock.Anything, mock.Anything).
					Return(&firestorePkg.SlackWorkspace{
						Name: "workspace-name",
					}, "", "", "", nil)
			},
			want:    nil,
			wantErr: true,
			assert: func(t *testing.T, f *fields) {
				f.slackDAL.AssertNumberOfCalls(t, "GetUserEmail", 1)
				f.slackDAL.AssertNumberOfCalls(t, "SendUnfurl", 2)
				f.slackDAL.AssertNumberOfCalls(t, "SendUnfurlWithEphemeral", 0)
				f.slackDAL.AssertNumberOfCalls(t, "GetAllChannelMemberEmails", 0)
				f.slackDAL.AssertNumberOfCalls(t, "GetChannelInfo", 0)
				f.budgetsService.AssertNumberOfCalls(t, "GetUnfurlPayload", 1)
				f.budgetsService.AssertNumberOfCalls(t, "UpdateSharing", 0)
				f.reportsService.AssertNumberOfCalls(t, "GetUnfurlPayload", 0)
				f.reportsService.AssertNumberOfCalls(t, "UpdateSharing", 0)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetDoitEmployee", 0)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetSharedChannel", 1)
				f.firestoreDAL.AssertNumberOfCalls(t, "GetWorkspaceDecrypted", 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := initFields(t)
			if err != nil {
				t.Errorf("Failed to initiate fields SlackService error = %v", err)
				return
			}

			if tt.on != nil {
				tt.on(f)
			}

			service := initService(f)

			_, err = service.HandleLinkSharedEvent(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackService.HandleLinkSharedEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}
