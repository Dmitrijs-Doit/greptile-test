package service

import (
	"context"
	"testing"

	"github.com/doitintl/hello/scheduled-tasks/common"
	csmDal "github.com/doitintl/hello/scheduled-tasks/csmengagement/dal"
	"github.com/doitintl/hello/scheduled-tasks/csmengagement/mocks"
	customerDalMock "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	customerDomain "github.com/doitintl/hello/scheduled-tasks/customer/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	userDalMock "github.com/doitintl/hello/scheduled-tasks/user/dal/mocks"
	userDomain "github.com/doitintl/hello/scheduled-tasks/user/domain"
	nc "github.com/doitintl/notificationcenter/pkg"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	ncMock "github.com/doitintl/notificationcenter/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_sendAttributionEmails(t *testing.T) {
	type fields struct {
		l      logger.ILogger
		nc     ncMock.NotificationSender
		csmDal mocks.ICsmEngagement
		cDal   customerDalMock.Customers
		uDal   userDalMock.IUserFirestoreDAL
	}

	type args struct {
		ctx          context.Context
		atrs         []csmDal.AttributionData
		onlyFirstAtr bool
		tracker      csmDal.SentNotificationsTracker
		dal          *mocks.IAttributionEmail
	}

	tracker := &mocks.SentNotificationsTracker{}
	tracker.On("GetSent", mock.Anything).Return(map[string]string{}, nil)
	tracker.On("UpdateSent", mock.Anything, mock.Anything).Return(nil)

	collab1 := collab.Collaborator{
		Email: "test@email.com",
		Role:  collab.CollaboratorRoleOwner,
	}

	atrs := []csmDal.AttributionData{
		{
			AttributionID: "attribution1",
			CustomerID:    "customer1",
			Collabs: []collab.Collaborator{
				collab1,
			},
		},
	}

	dal := &mocks.IAttributionEmail{}
	dal.On("GetAttributionsByDateRange", mock.Anything, mock.Anything, mock.Anything).Return(atrs, nil)
	dal.On("IsFirstAttribution", mock.Anything, "attribution1", mock.Anything).Return(true, nil)
	dal.On("HasBudgets", mock.Anything, collab1).Return(false, nil)
	dal.On("HasAlerts", mock.Anything, collab1).Return(false, nil)

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
		want    int
	}{
		{
			name: "first attribution",
			on: func(f *fields) {
				f.csmDal.On("GetCustomerMRR", mock.Anything, "customer1", true).Return(20100.00, nil)
				f.uDal.On("GetUserByEmail", mock.Anything, collab1.Email, "customer1").Return(&userDomain.User{
					FirstName: "John",
				}, nil)
				f.cDal.On("GetCustomerAccountTeam", mock.Anything, "customer1").Return([]customerDomain.AccountManagerListItem{}, nil)
				f.nc.On("Send", mock.Anything, nc.Notification{
					Template: "ZRDJ2MN79041EFQN5XF2MGYBTMY2",
					Email:    []string{"test@email.com"},
					BCC:      []string{},
					EmailFrom: nc.EmailFrom{
						Name:  "DoiT International",
						Email: "csm@doit-intl.com",
					},
					Data: map[string]interface{}{
						"subject": "Congrats on Your First Attribution!",
						"name":    "John",
						"first":   "true",
						"link":    getConsoleLink("customer1", "attribution1"),
					},
					Mock: !common.Production,
				}).Return("", nil)

			},
			args: args{
				ctx:          context.Background(),
				atrs:         atrs,
				onlyFirstAtr: true,
				tracker:      tracker,
				dal:          dal,
			},
			wantErr: nil,
			want:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				l:      &logger.Logger{},
				nc:     *ncMock.NewNotificationSender(t),
				csmDal: *mocks.NewICsmEngagement(t),
				uDal:   *userDalMock.NewIUserFirestoreDAL(t),
				cDal:   *customerDalMock.NewCustomers(t),
			}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				l:                  fields.l,
				notificationSender: &fields.nc,
				csmService:         &fields.csmDal,
				customerDAL:        &fields.cDal,
				userDAL:            &fields.uDal,
			}

			got, err := s.sendAttributionEmails(tt.args.ctx, tt.args.atrs, tt.args.onlyFirstAtr, tt.args.dal, tt.args.tracker)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if got != tt.want {
				t.Errorf("Service.sendAttributionEmails() = %v, want %v", got, tt.want)
			}
		})
	}
}
