package service

import (
	"context"
	"fmt"
	"math"
	"net/http/httptest"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal"
	mocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestBudgetsService_GetBudgetExternal(t *testing.T) {
	type fields struct {
		dal *mocks.Budgets
	}

	type args struct {
		ctx        context.Context
		email      string
		budgetID   string
		customerID string
	}

	customerID := "test_customer_id"
	customerIDTest := "test_customer_id2"
	emailTest := "test@example.com"
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	returnedBudget := &budget.Budget{
		Customer: &firestore.DocumentRef{
			ID: customerID,
		},
		Access: collab.Access{
			Collaborators: []collab.Collaborator{{
				Email: email,
				Role:  collab.CollaboratorRoleOwner,
			}},
		},
		Config: &budget.BudgetConfig{},
	}
	budgetWithoutConfig := &budget.Budget{
		Customer: &firestore.DocumentRef{
			ID: customerID,
		},
		Access: collab.Access{
			Collaborators: []collab.Collaborator{{
				Email: email,
				Role:  collab.CollaboratorRoleOwner,
			}},
		},
	}

	tests := []struct {
		name    string
		args    args
		wantErr error

		on func(*fields)
	}{
		{
			name: "Happy path",
			args: args{
				ctx:        ctx,
				email:      email,
				budgetID:   budgetID,
				customerID: customerID,
			},
			wantErr: nil,
			on: func(f *fields) {
				f.dal.
					On("GetBudget", ctx, budgetID).
					Return(returnedBudget, nil).
					Once()
			},
		},
		{
			name: "Budget not found",
			args: args{
				ctx:        ctx,
				email:      email,
				budgetID:   budgetID,
				customerID: customerID,
			},
			wantErr: web.ErrNotFound,
			on: func(f *fields) {
				f.dal.
					On("GetBudget", ctx, budgetID).
					Return(nil, doitFirestore.ErrNotFound).
					Once()
			},
		},
		{
			name: "Internal error getting budget",
			args: args{
				ctx:        ctx,
				email:      email,
				budgetID:   budgetID,
				customerID: customerID,
			},
			wantErr: web.ErrInternalServerError,
			on: func(f *fields) {
				f.dal.
					On("GetBudget", ctx, budgetID).
					Return(nil, web.ErrInternalServerError).
					Once()
			},
		},
		{
			name: "Budget doesn't belong to customer",
			args: args{
				ctx:        ctx,
				email:      email,
				budgetID:   budgetID,
				customerID: customerIDTest,
			},
			wantErr: web.ErrUnauthorized,
			on: func(f *fields) {
				f.dal.
					On("GetBudget", ctx, budgetID).
					Return(returnedBudget, nil).
					Once()
			},
		},
		{
			name: "User is not one of the collaborators",
			args: args{
				ctx:        ctx,
				email:      emailTest,
				budgetID:   budgetID,
				customerID: customerIDTest,
			},
			wantErr: web.ErrUnauthorized,
			on: func(f *fields) {
				f.dal.
					On("GetBudget", ctx, budgetID).
					Return(returnedBudget, nil).
					Once()
			},
		},
		{
			name: "Budget without config",
			args: args{
				ctx:        ctx,
				email:      email,
				budgetID:   budgetID,
				customerID: customerID,
			},
			wantErr: ErrMissingBudgetConfig,
			on: func(f *fields) {
				f.dal.
					On("GetBudget", ctx, budgetID).
					Return(budgetWithoutConfig, nil).
					Once()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				dal: &mocks.Budgets{},
			}
			s := &BudgetsService{
				dal: f.dal,
			}

			if tt.on != nil {
				tt.on(&f)
			}

			_, err := s.GetBudgetExternal(tt.args.ctx, tt.args.budgetID, tt.args.email, tt.args.customerID)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBudgetsService_ListBudgetExternal(t *testing.T) {
	type fields struct {
		dal *mocks.Budgets
	}

	type args struct {
		ctx           context.Context
		email         string
		budgetRequest BudgetsRequest
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	customerID := "test_customer_id"
	email := "test@test.com"
	timeYesterday := time.Now().AddDate(0, 0, -1)
	timeNowUnix := time.Now().UnixMilli()
	timeNow, _ := common.MsToTime(fmt.Sprint(timeNowUnix))
	timeTomorrow := time.Now().AddDate(0, 0, 1)

	tests := []struct {
		name       string
		args       args
		wantErr    error
		wantIntErr error
		wantResult *BudgetListResponse

		on func(*fields)
	}{
		{
			name: "Happy path with filter by owner and pagination",
			args: args{
				ctx:   ctx,
				email: email,
				budgetRequest: BudgetsRequest{
					MaxResults:      "2",
					MinCreationTime: "1685976601000",
					MaxCreationTime: "1688568601000",
					PageToken:       "YWIz",
					Filter:          "owner:test@test.com|owner:test1@test.com|lastModified:" + fmt.Sprint(timeNowUnix),
				},
			},
			wantErr: nil,
			wantResult: &BudgetListResponse{
				PageToken: "YWIx",
				RowCount:  2,
				Budgets: []BudgetListItem{
					{ID: "ab3", CreateTime: timeNowUnix, UpdateTime: timeNowUnix, Owner: "test@test.com", URL: "https://" + common.Domain + "/customers/" + customerID + "/analytics/budgets/" + "ab3"},
					{ID: "ab2", CreateTime: timeNowUnix, UpdateTime: timeNowUnix, Owner: "test1@test.com", URL: "https://" + common.Domain + "/customers/" + customerID + "/analytics/budgets/" + "ab2"},
				},
			},
			on: func(f *fields) {
				minCreationTime, _ := common.MsToTime("1685976601000")
				maxCreationTime, _ := common.MsToTime("1688568601000")
				pageToken := "YWIz"
				filter := dal.BudgetListFilter{
					Owners:       []string{"test@test.com", "test1@test.com"},
					TimeModified: &timeNow,
					OrderBy:      "updateTime",
				}

				f.dal.
					On("ListBudgets", ctx, &dal.ListBudgetsArgs{
						CustomerID:      customerID,
						Email:           email,
						MinCreationTime: &minCreationTime,
						MaxCreationTime: &maxCreationTime,
						IsDoitEmployee:  false,
						Filter:          &filter,
						MaxResults:      2,
						PageToken:       pageToken,
					}).
					Return([]budget.Budget{
						{ID: "ab1", TimeCreated: timeNow, TimeModified: timeNow,
							Access: collab.Access{
								Collaborators: []collab.Collaborator{
									{Email: "test@test.com", Role: collab.CollaboratorRoleOwner},
								}},
						},
						{
							ID:           "ab2",
							TimeCreated:  timeNow,
							TimeModified: timeNow,
							Access: collab.Access{
								Collaborators: []collab.Collaborator{
									{Email: "test1@test.com", Role: collab.CollaboratorRoleOwner},
								},
							},
						},
						{
							ID:           "ab3",
							TimeCreated:  timeNow,
							TimeModified: timeNow,
							Access: collab.Access{
								Collaborators: []collab.Collaborator{
									{Email: "test@test.com", Role: collab.CollaboratorRoleOwner},
								},
							},
						},
						{
							ID:           "ab4",
							TimeCreated:  timeNow,
							TimeModified: timeNow,
							Access: collab.Access{
								Collaborators: []collab.Collaborator{
									{Email: "test2@test.com", Role: collab.CollaboratorRoleOwner},
								},
							},
						},
						{
							ID:           "ab5",
							TimeCreated:  timeNow,
							TimeModified: timeNow,
							Access: collab.Access{
								Collaborators: []collab.Collaborator{
									{Email: "test1@test.com", Role: collab.CollaboratorRoleOwner},
								},
							},
						},
						{
							ID:           "ab6",
							TimeCreated:  timeYesterday,
							TimeModified: timeYesterday,
							Access: collab.Access{
								Collaborators: []collab.Collaborator{
									{Email: "test1@test.com", Role: collab.CollaboratorRoleOwner},
								},
							},
						},
						{
							ID:           "ab7",
							TimeCreated:  timeTomorrow,
							TimeModified: timeTomorrow,
							Access: collab.Access{
								Collaborators: []collab.Collaborator{
									{Email: "test@test.com", Role: collab.CollaboratorRoleOwner},
								},
							},
						},
					}, nil).
					Once()
			},
		},
		{
			name: "Validation error max result range",
			args: args{
				ctx:   ctx,
				email: email,
				budgetRequest: BudgetsRequest{
					MaxResults: "350",
				},
			},
			wantErr: ErrorParamMaxResultRange,
		},
		{
			name: "Validation error max results",
			args: args{
				ctx:   ctx,
				email: email,
				budgetRequest: BudgetsRequest{
					MaxResults: "e1685976601000",
				},
			},
			wantErr: fmt.Errorf(ErrorInvalidValue, "maxResults"),
		},
		{
			name: "Validation error min creation time",
			args: args{
				ctx:   ctx,
				email: email,
				budgetRequest: BudgetsRequest{
					MinCreationTime: "e1685976601000",
				},
			},
			wantErr: fmt.Errorf(ErrorInvalidValue, "minCreationTime"),
		},
		{
			name: "Validation error max creation time",
			args: args{
				ctx:   ctx,
				email: email,
				budgetRequest: BudgetsRequest{
					MaxCreationTime: "e1685976601000",
				},
			},
			wantErr: fmt.Errorf(ErrorInvalidValue, "maxCreationTime"),
		},
		{
			name: "Validation error filter lastModified",
			args: args{
				ctx:   ctx,
				email: email,
				budgetRequest: BudgetsRequest{
					Filter: "lastModified:e1685976601000",
				},
			},
			wantErr: fmt.Errorf(ErrorInvalidValue, "lastModified"),
		},
		{
			name: "Validation error invalid filter key",
			args: args{
				ctx:   ctx,
				email: email,
				budgetRequest: BudgetsRequest{
					Filter: "lastModified1:e1685976601000",
				},
			},
			wantErr: fmt.Errorf(ErrorInvalidFilterKey, "lastModified1"),
		},
		{
			name: "Internal Error",
			args: args{
				ctx:   ctx,
				email: email,
			},
			wantIntErr: ErrInternalError,
			on: func(f *fields) {
				f.dal.
					On("ListBudgets", ctx, &dal.ListBudgetsArgs{
						CustomerID:     customerID,
						Email:          email,
						IsDoitEmployee: false,
						MaxResults:     50,
						Filter:         &dal.BudgetListFilter{OrderBy: "createTime"},
					}).
					Return(nil, ErrInternalError).
					Once()
			},
		},
		{
			name: "+Inf gets ignored",
			args: args{
				ctx:   ctx,
				email: email,
			},
			wantResult: &BudgetListResponse{
				PageToken: "",
				RowCount:  1,
				Budgets: []BudgetListItem{
					{
						ID:                 "ab6",
						Owner:              "test1@test.com",
						CreateTime:         timeYesterday.UnixMilli(),
						UpdateTime:         timeYesterday.UnixMilli(),
						Amount:             0,
						Currency:           "",
						TimeInterval:       "",
						StartPeriod:        -62135596800000,
						EndPeriod:          -62135596800000,
						CurrentUtilization: 0,
						AlertThresholds:    []AlertThreshold{{Percentage: 0, Amount: 0}, {Percentage: 0, Amount: 0}},
						URL:                "https://dev-app.doit.com/customers/test_customer_id/analytics/budgets/ab6",
					},
				},
			},
			on: func(f *fields) {
				f.dal.
					On("ListBudgets", ctx, &dal.ListBudgetsArgs{
						CustomerID:     customerID,
						Email:          email,
						IsDoitEmployee: false,
						MaxResults:     50,
						Filter:         &dal.BudgetListFilter{OrderBy: "createTime"},
					}).
					Return([]budget.Budget{
						{
							ID:           "ab6",
							TimeCreated:  timeYesterday,
							TimeModified: timeYesterday,
							Access: collab.Access{
								Collaborators: []collab.Collaborator{{Email: "test1@test.com", Role: collab.CollaboratorRoleOwner}},
							},
							Config: &budget.BudgetConfig{
								Alerts: [3]budget.BudgetAlert{
									{Percentage: math.Inf(1), ForecastedDate: nil, Triggered: false},
									{Percentage: 0.0, ForecastedDate: nil, Triggered: false},
									{Percentage: 0.0, ForecastedDate: nil, Triggered: false}},
							},
						}}, nil).
					Once()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				dal: &mocks.Budgets{},
			}
			s := &BudgetsService{
				dal: f.dal,
			}

			if tt.on != nil {
				tt.on(&f)
			}

			res, paramErr, internalErr := s.ListBudgets(tt.args.ctx, &ExternalAPIListArgsReq{
				BudgetRequest:  &tt.args.budgetRequest,
				Email:          email,
				CustomerID:     customerID,
				IsDoitEmployee: false,
			})

			if tt.wantErr != nil {
				assert.EqualError(t, paramErr, tt.wantErr.Error())
			} else if tt.wantIntErr != nil {
				assert.EqualError(t, internalErr, tt.wantIntErr.Error())
			} else {
				assert.NoError(t, paramErr)
				assert.NoError(t, internalErr)
			}

			if tt.wantResult != nil {
				assert.Equal(t, tt.wantResult.PageToken, res.PageToken)
				assert.Equal(t, tt.wantResult.RowCount, res.RowCount)

				for index, budget := range tt.wantResult.Budgets {
					assert.Equal(t, budget, res.Budgets[index])
				}
			}
		})
	}
}
