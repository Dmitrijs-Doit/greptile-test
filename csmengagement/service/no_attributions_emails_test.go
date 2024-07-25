package service

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/csmengagement/mocks"
	customerDalMock "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/customer/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ncMock "github.com/doitintl/notificationcenter/mocks"
)

func TestService_sendNoAttributionsEmails(t *testing.T) {
	type fields struct {
		l      logger.ILogger
		nc     ncMock.NotificationSender
		csmDal mocks.ICsmEngagement
		attDal mocks.INoAttributionsEmail
		cDal   customerDalMock.Customers
	}

	type args struct {
		ctx context.Context
	}

	customerWithCustomAttributions := firestore.DocumentRef{
		ID: "customer1",
	}
	customerWithoutCustomAttributions := firestore.DocumentRef{
		ID: "customer2",
	}
	customerWithoutCustomAttributionsSnap := firestore.DocumentSnapshot{
		Ref: &customerWithoutCustomAttributions,
	}
	customAttribution1 := attribution.Attribution{
		ID:       "attribution1",
		Customer: &customerWithCustomAttributions,
	}

	eligibleUser := common.User{
		ID: "user1",
		Permissions: []string{
			string(common.PermissionAttributionsManager),
		},
		Email:     "user1@customer2.com",
		LastLogin: time.Now(),
	}

	accountManagerItem := domain.AccountManagerListItem{
		ID:    "Manager1",
		Email: "manager1@manager1.com",
		Name:  "Manager1",
		Role:  common.AccountManagerRoleFSR,
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
	}{
		{
			name: "Customer has no custom attributions and has one eligible user",
			on: func(f *fields) {
				tracker := &mocks.SentNotificationsTracker{}
				tracker.On("GetSent", mock.Anything).Return(map[string]string{}, nil)
				tracker.On("UpdateSent", mock.Anything, mock.Anything).Return(nil)
				f.attDal.On("GetTracker").Return(tracker)
				f.attDal.On("GetAllCustomAttributions", mock.Anything).Return([]attribution.Attribution{customAttribution1}, nil)
				f.attDal.On("GetCustomersNewerThanThirtyDays", mock.Anything).Return([]*firestore.DocumentSnapshot{&customerWithoutCustomAttributionsSnap}, nil)
				f.csmDal.On("IsCustomerResold", mock.Anything, "customer2").Return(true, nil)
				f.csmDal.On("GetCustomerMRR", mock.Anything, "customer2", true).Return(float64(21000), nil)
				f.attDal.On("GetRequiredRolePermissions", mock.Anything, "Standard User").Return([]*firestore.DocumentRef{
					{
						ID: string(common.PermissionAttributionsManager),
					},
				}, nil)
				f.attDal.On("GetEligibleUsersForCustomer", mock.Anything, &customerWithoutCustomAttributions).Return([]*common.User{&eligibleUser}, nil)
				f.attDal.On("GetAttributionsForUser", mock.Anything, collab.Collaborator{
					Email: eligibleUser.Email,
					Role:  collab.CollaboratorRoleOwner,
				}).Return([]*firestore.DocumentRef{}, nil)
				f.cDal.On("GetCustomerAccountTeam", mock.Anything, customerWithoutCustomAttributions.ID).Return([]domain.AccountManagerListItem{accountManagerItem}, nil)
				f.attDal.On("UserHasRequiredPermissions", mock.Anything, &eligibleUser, []common.Permission{common.PermissionAttributionsManager}).Return(nil)
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			fields := fields{
				l:      &logger.Logger{},
				nc:     *ncMock.NewNotificationSender(t),
				csmDal: *mocks.NewICsmEngagement(t),
				attDal: *mocks.NewINoAttributionsEmail(t),
				cDal:   *customerDalMock.NewCustomers(t),
			}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				l:                  fields.l,
				notificationSender: &fields.nc,
				csmService:         &fields.csmDal,
				noAttrsDAL:         &fields.attDal,
				customerDAL:        &fields.cDal,
			}

			emailsToSend, err := s.GetNoAttributionsEmails(tt.args.ctx)
			if err != nil {
				t.Errorf("GetNoAttributionsEmails() error = %v", err)
				return
			}

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.Equal(t, 1, len(emailsToSend))
			}
		})
	}
}
