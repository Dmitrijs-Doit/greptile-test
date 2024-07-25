package service

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/tiers/dal"
	userDalIface "github.com/doitintl/hello/scheduled-tasks/user/dal/iface"
	notificationcenter "github.com/doitintl/notificationcenter/pkg"
	tiersService "github.com/doitintl/tiers/service"
	"github.com/stretchr/testify/mock"
)

func TestTiersService_getLatestRelevantNotification(t *testing.T) {
	type fields struct {
		loggerProvider        logger.Provider
		Connection            *connection.Connection
		customerDal           customerDal.Customers
		usersDal              userDalIface.IUserFirestoreDAL
		trialNotificationsDal dal.TrialNotificationsDAL
		tiersSvc              tiersService.TierServiceIface
		notificationClient    notificationcenter.NotificationSender
	}
	type args struct {
		tierData      *startEndDates
		notifications []trialNotification
		now           time.Time
		lastDate      time.Time
	}

	now := time.Now().UTC()

	trialStartDate := now.AddDate(0, 0, -15)
	trialEndDate := now.AddDate(0, 0, 20)

	tierDates := &startEndDates{
		start: &trialStartDate,
		end:   &trialEndDate,
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		// TODO: Add test cases.
		{
			name: "Navigator Trial - 15 days into trial",
			args: args{
				tierData:      tierDates,
				notifications: activeTrialNotifications[pkg.NavigatorPackageTierType],
				now:           now,
				lastDate:      time.Time{},
			},
			want: 0,
		},
		{
			name: "Navigator Trial - week before end trial",
			args: args{
				tierData:      tierDates,
				notifications: activeTrialNotifications[pkg.NavigatorPackageTierType],
				now:           now.AddDate(0, 0, 13),
				lastDate:      now,
			},
			want: 1,
		},
		{
			name: "Navigator Trial - end of trial",
			args: args{
				tierData:      tierDates,
				notifications: activeTrialNotifications[pkg.NavigatorPackageTierType],
				now:           now.AddDate(0, 0, 20),
				lastDate:      now.AddDate(0, 0, 13),
			},
			want: 2,
		},
		{
			name: "Navigator Trial - should not send - too yearly",
			args: args{
				tierData:      tierDates,
				notifications: activeTrialNotifications[pkg.NavigatorPackageTierType],
				now:           now.AddDate(0, 0, 19),
				lastDate:      now.AddDate(0, 0, 13),
			},
			want: -1,
		},
		{
			name: "Navigator Trial - should not send - already sent",
			args: args{
				tierData:      tierDates,
				notifications: activeTrialNotifications[pkg.NavigatorPackageTierType],
				now:           now.AddDate(0, 0, 20),
				lastDate:      now.AddDate(0, 0, 20).Add(time.Minute),
			},
			want: -1,
		},
		{
			name: "Navigator Trial - should not send - older than 2 days",
			args: args{
				tierData:      tierDates,
				notifications: activeTrialNotifications[pkg.NavigatorPackageTierType],
				now:           now.AddDate(0, 0, 3),
				lastDate:      time.Time{},
			},
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &TiersService{
				loggerProvider:        tt.fields.loggerProvider,
				Connection:            tt.fields.Connection,
				customerDal:           tt.fields.customerDal,
				usersDal:              tt.fields.usersDal,
				trialNotificationsDal: tt.fields.trialNotificationsDal,
				tiersSvc:              tt.fields.tiersSvc,
				notificationClient:    tt.fields.notificationClient,
			}
			if got := s.getLatestRelevantNotification(tt.args.tierData, tt.args.notifications, tt.args.now, tt.args.lastDate); got != tt.want {
				t.Errorf("TiersService.getLatestRelevantNotification() = %v, want %v", got, tt.want)
			}
		})
	}
}

type testUsage struct{}

func (t *testUsage) hasUsage(ctx context.Context, s *TiersService, customerRef *firestore.DocumentRef) (bool, error) {
	return true, nil
}

type testNoUsage struct{}

func (t *testNoUsage) hasUsage(ctx context.Context, s *TiersService, customerRef *firestore.DocumentRef) (bool, error) {
	return false, nil
}

func TestTiersService_sendActiveTrialNotifications(t *testing.T) {
	type fields struct {
		loggerProvider *loggerMocks.ILogger
		Connection     *connection.Connection
		customerDal    *customerMocks.Customers
	}
	type args struct {
		ctx               context.Context
		customerRef       *firestore.DocumentRef
		tiers             map[string]*pkg.CustomerTier
		lastNotifications *dal.CustomerTrialNotifications
		users             []*common.User
		usageVerifier     usageVerifier
		now               time.Time
		dryRun            bool
	}

	now := time.Now().UTC()

	trialStartDate := now.AddDate(0, 0, -20)
	trialEndDate := now.AddDate(0, 0, 20)
	trialCanceledDate := now.AddDate(0, 0, -3)

	usageTrialStartDate := now.AddDate(0, 0, -12)

	customerRef := &firestore.DocumentRef{
		ID: "customerID",
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		on     func(*fields)
		want   bool
	}{
		{
			name: "Navigator Trial - should not send - Canceled trial",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
				tiers: map[string]*pkg.CustomerTier{
					string(pkg.NavigatorPackageTierType): {
						TrialStartDate:    &trialStartDate,
						TrialEndDate:      &trialEndDate,
						TrialCanceledDate: &trialCanceledDate,
					},
				},
				lastNotifications: &dal.CustomerTrialNotifications{
					LastSent: make(map[string]time.Time),
				},
				users:  nil,
				now:    now,
				dryRun: true,
			},
			want: false,
		},
		{
			name: "Navigator Trial - should not send - No trial start date",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
				tiers: map[string]*pkg.CustomerTier{
					string(pkg.NavigatorPackageTierType): {
						TrialEndDate: &trialEndDate,
					},
				},
				lastNotifications: &dal.CustomerTrialNotifications{
					LastSent: make(map[string]time.Time),
				},
				users:  nil,
				now:    now,
				dryRun: true,
			},
			want: false,
		},
		{
			name: "Navigator Trial - should not send - No trial end date",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
				tiers: map[string]*pkg.CustomerTier{
					string(pkg.NavigatorPackageTierType): {
						TrialStartDate: &trialStartDate,
					},
				},
				lastNotifications: &dal.CustomerTrialNotifications{
					LastSent: make(map[string]time.Time),
				},
				users:  nil,
				now:    now,
				dryRun: true,
			},
			want: false,
		},
		{
			name: "Navigator Trial - should send no attribution notification",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
				tiers: map[string]*pkg.CustomerTier{
					string(pkg.NavigatorPackageTierType): {
						TrialStartDate: &usageTrialStartDate,
						TrialEndDate:   &trialEndDate,
					},
				},
				lastNotifications: &dal.CustomerTrialNotifications{
					LastSent:  make(map[string]time.Time),
					UsageSent: make(map[string][]string),
				},
				users: []*common.User{{
					FirstName: "customerName",
					Email:     "team-navigator@doit.com",
				}},
				now:           now,
				dryRun:        true,
				usageVerifier: &testNoUsage{},
			},
			fields: fields{
				loggerProvider: &loggerMocks.ILogger{},
				customerDal:    &customerMocks.Customers{},
			},
			on: func(f *fields) {
				f.customerDal.On("GetCustomerAccountTeam", context.Background(), mock.AnythingOfType("string")).Return(nil, nil)
				f.loggerProvider.On("Debugf", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), "customerID", "team-navigator@doit.com", map[string]any{"name": "customerName"})
			},
			want: true,
		},
		{
			name: "Navigator Trial - not active trial, should not send no attribution notification",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
				tiers: map[string]*pkg.CustomerTier{
					string(pkg.NavigatorPackageTierType): {
						TrialStartDate: &usageTrialStartDate,
						TrialEndDate:   &trialEndDate,
					},
				},
				lastNotifications: &dal.CustomerTrialNotifications{
					LastSent:  make(map[string]time.Time),
					UsageSent: make(map[string][]string),
				},
				users:         nil,
				now:           trialEndDate.AddDate(0, 0, 5),
				dryRun:        true,
				usageVerifier: &testNoUsage{},
			},
			want: false,
		},
		{
			name: "Navigator Trial - has attributions, should not send no attribution notification",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
				tiers: map[string]*pkg.CustomerTier{
					string(pkg.NavigatorPackageTierType): {
						TrialStartDate: &usageTrialStartDate,
						TrialEndDate:   &trialEndDate,
					},
				},
				lastNotifications: &dal.CustomerTrialNotifications{
					LastSent:  make(map[string]time.Time),
					UsageSent: make(map[string][]string),
				},
				users:         nil,
				now:           now,
				dryRun:        true,
				usageVerifier: &testUsage{},
			},
			want: false,
		},
		{
			name: "Navigator Trial - should not send, already sent usage notifications",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
				tiers: map[string]*pkg.CustomerTier{
					string(pkg.NavigatorPackageTierType): {
						TrialStartDate: &usageTrialStartDate,
						TrialEndDate:   &trialEndDate,
					},
				},
				lastNotifications: &dal.CustomerTrialNotifications{
					LastSent:  make(map[string]time.Time),
					UsageSent: map[string][]string{string(pkg.NavigatorPackageTierType): {"noAttributions", "noAlertsOrBudgets"}},
				},
				now:           now,
				dryRun:        true,
				usageVerifier: &testNoUsage{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &TiersService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return tt.fields.loggerProvider
				},
				customerDal: tt.fields.customerDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			data := &notificationsData{
				customerRef:       tt.args.customerRef,
				customerName:      "customerName",
				tiersData:         tt.args.tiers,
				users:             tt.args.users,
				lastNotifications: tt.args.lastNotifications,
				dryRun:            tt.args.dryRun,
			}

			for packageType := range usageNotifications {
				for i := range usageNotifications[packageType] {
					usageNotifications[packageType][i].usageVerifier = tt.args.usageVerifier
				}
			}

			if got := s.sendCustomerActiveTrialNotification(tt.args.ctx, tt.args.now, data); got != tt.want {
				t.Errorf("TiersService.sendActiveTrialNotifications() = %v, want %v", got, tt.want)
			}
		})
	}
}
