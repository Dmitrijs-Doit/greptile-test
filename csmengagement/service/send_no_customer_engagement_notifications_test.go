package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	fsDalMocks "github.com/doitintl/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/csmengagement/dal"
	csmManagementMock "github.com/doitintl/hello/scheduled-tasks/csmengagement/dal/mocks"
	csmManagementServiceMock "github.com/doitintl/hello/scheduled-tasks/csmengagement/service/mocks"
	customerDALMock "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/customer/domain"
	logger "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	userDALMock "github.com/doitintl/hello/scheduled-tasks/user/dal/mocks"
	notificationcenter "github.com/doitintl/notificationcenter/mocks"
	notificationCenter "github.com/doitintl/notificationcenter/pkg"
	testUtils "github.com/doitintl/tests"
)

func Test_getCustomersWithNoEngagement(t *testing.T) {
	type args struct {
		allCustomerIDs            []string
		usersWithRecentEngagement []common.User
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "gets customers with no engagement",
			args: args{
				allCustomerIDs: []string{"customer1", "customer2", "customer3"},
				usersWithRecentEngagement: []common.User{
					{
						Customer: common.UserCustomer{
							Ref: &firestore.DocumentRef{
								ID: "customer1",
							},
						}},
					{
						Customer: common.UserCustomer{
							Ref: &firestore.DocumentRef{
								ID: "customer3",
							},
						},
					}},
			},
			want: []string{"customer2"},
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, getCustomersWithNoEngagement(
				tt.args.allCustomerIDs, tt.args.usersWithRecentEngagement),
				"getCustomersWithNoEngagement(%v, %v)",
				tt.args.allCustomerIDs, tt.args.usersWithRecentEngagement)
		})
	}
}

func Test_wasUserEngagedAfterEmail(t *testing.T) {
	type args struct {
		lastUserEngagement   *time.Time
		customerID           string
		allEngagementDetails map[string]dal.EngagementDetails
	}

	longTimeAgo := time.Now().Add(-time.Hour)
	recently := time.Now().Add(-time.Minute)

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should skip as no activity from user",
			args: args{
				lastUserEngagement: &longTimeAgo,
				customerID:         "test",
				allEngagementDetails: map[string]dal.EngagementDetails{
					"test": {
						NotifiedDates: []time.Time{recently},
					},
				},
			},
			want: true,
		},

		{
			name: "recent activity detected",
			args: args{
				lastUserEngagement: &recently,
				customerID:         "test",
				allEngagementDetails: map[string]dal.EngagementDetails{
					"test": {
						NotifiedDates: []time.Time{longTimeAgo},
					},
				},
			},
			want: false,
		},

		{
			name: "no engagement in map",
			args: args{
				lastUserEngagement:   &longTimeAgo,
				customerID:           "test",
				allEngagementDetails: nil,
			},
			want: false,
		},
		{
			name: "no user details",
			args: args{
				lastUserEngagement:   nil,
				customerID:           "test",
				allEngagementDetails: nil,
			},
			want: false,
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, skipDueToNoEngagementSinceEmail(
				tt.args.lastUserEngagement, tt.args.customerID, tt.args.allEngagementDetails),
				"skipDueToNoEngagementSinceEmail(%v, %v, %v)",
				tt.args.lastUserEngagement, tt.args.customerID, tt.args.allEngagementDetails)
		})
	}
}

func Test_service_SendNoCustomerEngagementNotifications(t *testing.T) {
	longTimeAgo := time.Now().Add(-time.Hour * 24 * 1000)

	type fields struct {
		l                  logger.ILogger
		notificationSender notificationcenter.NotificationSender
		csmService         csmManagementServiceMock.CSMEngagement
		csmEngagementDAL   csmManagementMock.CSMEngagementDAL
		customerDAL        customerDALMock.Customers
		userDAL            userDALMock.IUserFirestoreDAL
		customerTypeDal    *fsDalMocks.CustomerTypeIface
	}

	tests := []struct {
		name        string
		expectedErr string
		on          func(f *fields)
	}{
		{
			name:        "user dal returns error when getting users with recent engagement",
			expectedErr: "user dal error",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return(nil, errors.New("user dal error"))
			},
		},

		{
			name:        "customer dal returns error when getting all customer ids",
			expectedErr: "customer dal error",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{{
						ID: "user1",
						Customer: common.UserCustomer{
							Ref: &firestore.DocumentRef{
								ID: "customer1",
							},
						},
					}}, nil)

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return(nil, errors.New("customer dal error"))
			},
		},

		{
			name:        "getting engagement details returns error when getting engagement details",
			expectedErr: "engagement details error",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{}, nil)

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return([]string{}, nil)

				f.csmEngagementDAL.On("GetCustomerEngagementDetailsByCustomerID", context.Background()).
					Return(nil, errors.New("engagement details error"))
			},
		},

		{
			name:        "will attempt to process multiple customers picking correct ones",
			expectedErr: "",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{
						{
							ID: "user1",
							Customer: common.UserCustomer{
								Ref: &firestore.DocumentRef{
									ID: "customer1",
								},
							},
						},
						{
							ID: "user2",
							Customer: common.UserCustomer{
								Ref: &firestore.DocumentRef{
									ID: "customer1",
								},
							},
						},
						{
							ID: "user3",
							Customer: common.UserCustomer{
								Ref: &firestore.DocumentRef{
									ID: "customer3",
								},
							},
						},
					}, nil).Once()

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return([]string{"customer1", "customer2", "customer3", "customer4"}, nil).Once()

				f.csmEngagementDAL.On("GetCustomerEngagementDetailsByCustomerID", context.Background()).
					Return(map[string]dal.EngagementDetails{}, nil).Once()

				f.customerDAL.On("GetCustomer", context.Background(), "customer2").
					Return(nil, errors.New("customer error")).Once()
				f.customerDAL.On("GetCustomer", context.Background(), "customer4").
					Return(nil, errors.New("customer error")).Once()

				f.l.On("Error", mock.Anything)
			},
		},

		{
			name:        "will skip standalone customer",
			expectedErr: "",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{}, nil).Once()

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return([]string{"customer1"}, nil).Once()

				f.csmEngagementDAL.On("GetCustomerEngagementDetailsByCustomerID", context.Background()).
					Return(map[string]dal.EngagementDetails{}, nil).Once()

				f.customerDAL.On("GetCustomer", context.Background(), "customer1").Return(&common.Customer{
					ID: "customer1"}, nil).Once()
				f.customerTypeDal.On("IsProductOnlyCustomerType", context.Background(), "customer1").
					Return(true, nil).Once()
			},
		},

		{
			name:        "will skip terminated customer",
			expectedErr: "",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{}, nil).Once()

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return([]string{"customer1"}, nil).Once()

				f.csmEngagementDAL.On("GetCustomerEngagementDetailsByCustomerID", context.Background()).
					Return(map[string]dal.EngagementDetails{}, nil).Once()

				f.customerDAL.On("GetCustomer", context.Background(), "customer1").Return(&common.Customer{
					Classification: common.CustomerClassificationTerminated}, nil).Once()
			},
		},

		{
			name:        "will skip as customer was notified within last month",
			expectedErr: "",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{}, nil).Once()

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return([]string{"customer1"}, nil).Once()

				f.csmEngagementDAL.On("GetCustomerEngagementDetailsByCustomerID", context.Background()).
					Return(map[string]dal.EngagementDetails{
						"customer1": {
							NotifiedDates: []time.Time{
								time.Now().Add(-time.Hour * 24 * 20),   // 20 days ago
								time.Now().Add(-time.Hour * 24 * 1000), // long time ago
							},
						},
					}, nil).Once()

				f.customerDAL.On("GetCustomer", context.Background(), "customer1").
					Return(&common.Customer{ID: "customer1"}, nil).Once()
				f.customerTypeDal.On("IsProductOnlyCustomerType", context.Background(), "customer1").
					Return(false, nil).Once()
			},
		},

		{
			name:        "returns error getting mrr",
			expectedErr: "",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{}, nil).Once()

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return([]string{"customer1"}, nil).Once()

				f.csmEngagementDAL.On("GetCustomerEngagementDetailsByCustomerID", context.Background()).
					Return(map[string]dal.EngagementDetails{
						"customer1": {
							NotifiedDates: []time.Time{longTimeAgo},
						},
					}, nil).Once()

				f.customerDAL.On("GetCustomer", context.Background(), "customer1").
					Return(&common.Customer{ID: "customer1"}, nil).Once()

				f.customerTypeDal.On("IsProductOnlyCustomerType", context.Background(), "customer1").
					Return(false, nil).Once()

				f.csmService.On("GetCustomerMRR", context.Background(), "customer1", true).
					Return(0.0, errors.New("mrr error")).Once()

				f.l.On("Error", mock.Anything)
			},
		},

		{
			name:        "will skip with low mrr",
			expectedErr: "",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{}, nil).Once()

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return([]string{"customer1"}, nil).Once()

				f.csmEngagementDAL.On("GetCustomerEngagementDetailsByCustomerID", context.Background()).
					Return(map[string]dal.EngagementDetails{
						"customer1": {
							NotifiedDates: []time.Time{longTimeAgo},
						},
					}, nil).Once()

				f.customerDAL.On("GetCustomer", context.Background(), "customer1").
					Return(&common.Customer{ID: "customer1"}, nil).Once()

				f.customerTypeDal.On("IsProductOnlyCustomerType", context.Background(), "customer1").
					Return(false, nil).Once()

				f.csmService.On("GetCustomerMRR", context.Background(), "customer1", true).
					Return(200.0, nil).Once()
			},
		},

		{
			name:        "will continue when getting last activity returns error",
			expectedErr: "",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{}, nil).Once()

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return([]string{"customer1"}, nil).Once()

				f.csmEngagementDAL.On("GetCustomerEngagementDetailsByCustomerID", context.Background()).
					Return(map[string]dal.EngagementDetails{
						"customer1": {
							NotifiedDates: []time.Time{longTimeAgo},
						},
					}, nil).Once()

				f.customerDAL.On("GetCustomer", context.Background(), "customer1").
					Return(&common.Customer{ID: "customer1"}, nil).Once()

				f.customerTypeDal.On("IsProductOnlyCustomerType", context.Background(), "customer1").
					Return(false, nil).Once()

				f.csmService.On("GetCustomerMRR", context.Background(), "customer1", true).
					Return(10000000.0, nil).Once()

				f.userDAL.On("GetLastUserEngagementTimeForCustomer", context.Background(), "customer1").
					Return(nil, errors.New("last user engagement error")).Once()

				f.l.On("Error", mock.Anything)
			},
		},

		{
			name:        "will continue when no recent activity from customer",
			expectedErr: "",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{}, nil).Once()

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return([]string{"customer1"}, nil).Once()

				f.csmEngagementDAL.On("GetCustomerEngagementDetailsByCustomerID", context.Background()).
					Return(map[string]dal.EngagementDetails{
						"customer1": {
							NotifiedDates: []time.Time{longTimeAgo},
						},
					}, nil).Once()

				f.customerDAL.On("GetCustomer", context.Background(), "customer1").
					Return(&common.Customer{ID: "customer1"}, nil).Once()

				f.customerTypeDal.On("IsProductOnlyCustomerType", context.Background(), "customer1").
					Return(false, nil).Once()

				f.csmService.On("GetCustomerMRR", context.Background(), "customer1", true).
					Return(10000000.0, nil).Once()

				f.userDAL.On("GetLastUserEngagementTimeForCustomer", context.Background(), "customer1").
					Return(nil, nil).Once()
			},
		},

		{
			name:        "will continue if error getting account team",
			expectedErr: "",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{}, nil).Once()

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return([]string{"customer1"}, nil).Once()

				f.csmEngagementDAL.On("GetCustomerEngagementDetailsByCustomerID", context.Background()).
					Return(map[string]dal.EngagementDetails{
						"customer1": {
							NotifiedDates: []time.Time{time.Now().Add(-time.Hour * 24 * 1001)},
						},
					}, nil).Once()

				f.customerDAL.On("GetCustomer", context.Background(), "customer1").
					Return(&common.Customer{ID: "customer1"}, nil).Once()

				f.customerTypeDal.On("IsProductOnlyCustomerType", context.Background(), "customer1").
					Return(false, nil).Once()

				f.csmService.On("GetCustomerMRR", context.Background(), "customer1", true).
					Return(10000000.0, nil).Once()

				f.userDAL.On("GetLastUserEngagementTimeForCustomer", context.Background(), "customer1").
					Return(&longTimeAgo, nil).Once()

				f.customerDAL.On("GetCustomerAccountTeam", context.Background(), "customer1").
					Return(nil, errors.New("account team error")).Once()

				f.l.On("Error", mock.Anything)
			},
		},

		{
			name:        "will continue if no matches from account team",
			expectedErr: "",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{}, nil).Once()

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return([]string{"customer1"}, nil).Once()

				f.csmEngagementDAL.On("GetCustomerEngagementDetailsByCustomerID", context.Background()).
					Return(map[string]dal.EngagementDetails{
						"customer1": {
							NotifiedDates: []time.Time{time.Now().Add(-time.Hour * 24 * 1001)},
						},
					}, nil).Once()

				f.customerDAL.On("GetCustomer", context.Background(), "customer1").
					Return(&common.Customer{ID: "customer1"}, nil).Once()

				f.customerTypeDal.On("IsProductOnlyCustomerType", context.Background(), "customer1").
					Return(false, nil).Once()

				f.csmService.On("GetCustomerMRR", context.Background(), "customer1", true).Return(10000000.0, nil).Once()

				f.userDAL.On("GetLastUserEngagementTimeForCustomer", context.Background(), "customer1").Return(&longTimeAgo, nil).Once()

				f.customerDAL.On("GetCustomerAccountTeam", context.Background(), "customer1").Return([]domain.AccountManagerListItem{
					{Role: common.AccountManagerRolePSE},
				}, nil).Once()
			},
		},

		{
			name:        "will send notification and update firestore with last engagement time",
			expectedErr: "",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{}, nil).Once()

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return([]string{"customer1"}, nil).Once()

				f.csmEngagementDAL.On("GetCustomerEngagementDetailsByCustomerID", context.Background()).
					Return(map[string]dal.EngagementDetails{
						"customer1": {
							NotifiedDates: []time.Time{time.Now().Add(-time.Hour * 24 * 1001)},
						},
					}, nil).Once()

				f.customerDAL.On("GetCustomer", context.Background(), "customer1").
					Return(&common.Customer{
						ID:   "customer1",
						Name: "customer1 name",
					}, nil).Once()

				f.customerTypeDal.On("IsProductOnlyCustomerType", context.Background(), "customer1").
					Return(false, nil).Once()

				f.csmService.On("GetCustomerMRR", context.Background(), "customer1", true).
					Return(10000000.0, nil).Once()

				f.userDAL.On("GetLastUserEngagementTimeForCustomer", context.Background(), "customer1").
					Return(&longTimeAgo, nil).Once()

				f.customerDAL.On("GetCustomerAccountTeam", context.Background(), "customer1").
					Return([]domain.AccountManagerListItem{
						{
							Role:  common.AccountManagerRoleFSR,
							Name:  "dr pepper",
							Email: "dr.pepper@doit.com",
						},
					}, nil).Once()

				f.notificationSender.On("Send", context.Background(), notificationCenter.Notification{
					Template: notificationCenter.NoRecentUserActivity,
					Email:    []string{"dr.pepper@doit.com"},
					EmailFrom: notificationCenter.EmailFrom{
						Name:  "DoiT International",
						Email: "csm@doit-intl.com",
					},
					Data: map[string]interface{}{
						"CUSTOMER":            "customer1 name",
						"account_team_member": "dr pepper",
						"number_of":           "1000",
					},
					Mock: true,
				}).Return("requestID", nil).Once()

				f.csmEngagementDAL.On(
					"AddLastCustomerEngagementTime", context.Background(), "customer1", mock.AnythingOfType("time.Time")).
					Return(nil)
			},
		},

		{
			name:        "will log if saving last engagement time fails",
			expectedErr: "",
			on: func(f *fields) {
				f.userDAL.On("GetUsersWithRecentEngagement", context.Background()).
					Return([]common.User{}, nil).Once()

				f.customerDAL.On("GetAllCustomerIDs", context.Background()).
					Return([]string{"customer1"}, nil).Once()

				f.csmEngagementDAL.On("GetCustomerEngagementDetailsByCustomerID", context.Background()).
					Return(map[string]dal.EngagementDetails{
						"customer1": {
							NotifiedDates: []time.Time{time.Now().Add(-time.Hour * 24 * 1001)},
						},
					}, nil).Once()

				f.customerDAL.On("GetCustomer", context.Background(), "customer1").
					Return(&common.Customer{
						ID:   "customer1",
						Name: "customer1 name",
					}, nil).Once()

				f.customerTypeDal.On("IsProductOnlyCustomerType", context.Background(), "customer1").
					Return(false, nil).Once()

				f.csmService.On("GetCustomerMRR", context.Background(), "customer1", true).
					Return(10000000.0, nil).Once()

				f.userDAL.On("GetLastUserEngagementTimeForCustomer", context.Background(), "customer1").
					Return(&longTimeAgo, nil).Once()

				f.customerDAL.On("GetCustomerAccountTeam", context.Background(), "customer1").
					Return([]domain.AccountManagerListItem{{Role: common.AccountManagerRoleFSR}}, nil).Once()

				f.notificationSender.On("Send", context.Background(), mock.Anything).
					Return("requestID", nil).Once()

				f.csmEngagementDAL.On(
					"AddLastCustomerEngagementTime", context.Background(), "customer1", mock.AnythingOfType("time.Time")).
					Return(errors.New("error updating engagement time")).Once()

				f.l.On("Error", mock.Anything)
			},
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				notificationSender: notificationcenter.NotificationSender{},
				csmService:         csmManagementServiceMock.CSMEngagement{},
				csmEngagementDAL:   csmManagementMock.CSMEngagementDAL{},
				customerDAL:        customerDALMock.Customers{},
				userDAL:            userDALMock.IUserFirestoreDAL{},
				customerTypeDal:    &fsDalMocks.CustomerTypeIface{},
			}

			tt.on(&f)

			s := &service{
				l:                  &f.l,
				notificationSender: &f.notificationSender,
				csmService:         &f.csmService,
				csmEngagementDAL:   &f.csmEngagementDAL,
				customerDAL:        &f.customerDAL,
				userDAL:            &f.userDAL,
				customerTypeDal:    f.customerTypeDal,
			}

			gotErr := s.SendNoCustomerEngagementNotifications(context.Background())
			testUtils.WantErr(t, gotErr, tt.expectedErr)

			f.notificationSender.AssertExpectations(t)
			f.csmService.AssertExpectations(t)
			f.csmEngagementDAL.AssertExpectations(t)
			f.customerDAL.AssertExpectations(t)
			f.userDAL.AssertExpectations(t)
			f.customerTypeDal.AssertExpectations(t)
			f.l.AssertExpectations(t)
		})
	}
}
