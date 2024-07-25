package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/mock"

	dal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	caOwnerCheckersMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/caownerchecker/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	collabMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	labelsMocks "github.com/doitintl/hello/scheduled-tasks/labels/dal/mocks"
)

var (
	email    = "requester@example.com"
	budgetID = "my_budget_id"
	userID   = "my_user_id"
)

func TestBudgetsService_ShareBudget(t *testing.T) {
	type fields struct {
		dal            *dal.Budgets
		collab         *collabMock.Icollab
		caOwnerChecker *caOwnerCheckersMock.CheckCAOwnerInterface
	}

	type args struct {
		ctx                context.Context
		shareBudgetRequest ShareBudgetRequest
		email              string
		budgetID           string
		userID             string
	}

	ctx := context.Background()

	budget := &budget.Budget{
		Access: collab.Access{
			Collaborators: []collab.Collaborator{},
		},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool

		on func(*fields)
	}{
		{
			name: "Happy path",
			args: args{
				ctx: ctx,
				shareBudgetRequest: ShareBudgetRequest{
					Collaborators:           []collab.Collaborator{},
					PublicAccess:            nil,
					Recipients:              []string{},
					RecipientsSlackChannels: []common.SlackChannel{},
				},
				email:    email,
				budgetID: budgetID,
				userID:   userID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.dal.
					On("GetBudget", ctx, budgetID).
					Return(budget, nil).
					Once()
				f.collab.
					On("ShareAnalyticsResource", mock.Anything, mock.Anything, mock.Anything, mock.Anything, budgetID, email, mock.Anything, true).
					Return(nil).
					Once()
				f.dal.
					On("UpdateBudgetRecipients", ctx, budgetID, mock.Anything, mock.Anything).
					Return(nil).
					Once()
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()
				f.dal.
					On("UpdateBudgetEnforcedByMetering", ctx, budgetID, false).
					Return(nil).
					Once()
			},
		}, {
			name: "GetBudget returns error",
			args: args{
				ctx: ctx,
				shareBudgetRequest: ShareBudgetRequest{
					Collaborators:           []collab.Collaborator{},
					PublicAccess:            nil,
					Recipients:              []string{},
					RecipientsSlackChannels: []common.SlackChannel{},
				},
				email:    email,
				budgetID: budgetID,
				userID:   userID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.dal.
					On("GetBudget", ctx, budgetID).
					Return(budget, errors.New("error")).
					Once()
				f.collab.
					On("ShareAnalyticsResource", mock.Anything, mock.Anything, mock.Anything, mock.Anything, budgetID, email, mock.Anything, true).
					Return(nil).
					Once()
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()
			},
		}, {
			name: "ShareAnalyticsResource returns error",
			args: args{
				ctx: ctx,
				shareBudgetRequest: ShareBudgetRequest{
					Collaborators:           []collab.Collaborator{},
					PublicAccess:            nil,
					Recipients:              []string{},
					RecipientsSlackChannels: []common.SlackChannel{},
				},
				email:    email,
				budgetID: budgetID,
				userID:   userID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.dal.
					On("GetBudget", ctx, budgetID).
					Return(budget, nil).
					Once()
				f.collab.
					On("ShareAnalyticsResource", mock.Anything, mock.Anything, mock.Anything, mock.Anything, budgetID, email, mock.Anything, true).
					Return(errors.New("error")).
					Once()
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()
			},
		}, {
			name: "ShareAnalyticsResource returns error if CheckCAOwner throwing error",
			args: args{
				ctx: ctx,
				shareBudgetRequest: ShareBudgetRequest{
					Collaborators:           []collab.Collaborator{},
					PublicAccess:            nil,
					Recipients:              []string{},
					RecipientsSlackChannels: []common.SlackChannel{},
				},
				email:    email,
				budgetID: budgetID,
				userID:   userID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(false, errors.New("error")).Once()
			},
		}, {
			name: "UpdateBudgetRecipients returns error",
			args: args{
				ctx: ctx,
				shareBudgetRequest: ShareBudgetRequest{
					Collaborators:           []collab.Collaborator{},
					PublicAccess:            nil,
					Recipients:              []string{},
					RecipientsSlackChannels: []common.SlackChannel{},
				},
				email:    email,
				budgetID: budgetID,
				userID:   userID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.dal.
					On("GetBudget", ctx, budgetID).
					Return(budget, nil).
					Once()
				f.collab.
					On("ShareAnalyticsResource", mock.Anything, mock.Anything, mock.Anything, mock.Anything, budgetID, email, mock.Anything, true).
					Return(nil).
					Once()
				f.dal.
					On("UpdateBudgetRecipients", ctx, budgetID, mock.Anything, mock.Anything).
					Return(errors.New("error")).
					Once()
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()
			},
		}, {
			name: "UpdateBudgetEnforcedByMetering returns error",
			args: args{
				ctx: ctx,
				shareBudgetRequest: ShareBudgetRequest{
					Collaborators:           []collab.Collaborator{},
					PublicAccess:            nil,
					Recipients:              []string{},
					RecipientsSlackChannels: []common.SlackChannel{},
				},
				email:    email,
				budgetID: budgetID,
				userID:   userID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.dal.
					On("GetBudget", ctx, budgetID).
					Return(budget, nil).
					Once()
				f.collab.
					On("ShareAnalyticsResource", mock.Anything, mock.Anything, mock.Anything, mock.Anything, budgetID, email, mock.Anything, true).
					Return(nil).
					Once()
				f.dal.
					On("UpdateBudgetRecipients", ctx, budgetID, mock.Anything, mock.Anything).
					Return(nil).
					Once()
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()
				f.dal.
					On("UpdateBudgetEnforcedByMetering", ctx, budgetID, false).
					Return(errors.New("error")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				dal:            &dal.Budgets{},
				collab:         &collabMock.Icollab{},
				caOwnerChecker: &caOwnerCheckersMock.CheckCAOwnerInterface{},
			}
			s := &BudgetsService{
				dal:            tt.fields.dal,
				collab:         tt.fields.collab,
				caOwnerChecker: tt.fields.caOwnerChecker,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := s.ShareBudget(tt.args.ctx, tt.args.shareBudgetRequest, tt.args.budgetID, tt.args.userID, tt.args.email); (err != nil) != tt.wantErr {
				t.Errorf("BudgetService.ShareBudget() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBudgetsService_DeleteMany(t *testing.T) {
	type fields struct {
		dal            *dal.Budgets
		collab         *collabMock.Icollab
		caOwnerChecker *caOwnerCheckersMock.CheckCAOwnerInterface
		labelsMock     *labelsMocks.Labels
	}

	type args struct {
		ctx       context.Context
		email     string
		budgetIDs []string
	}

	ctx := context.Background()

	budgetWithOwner := &budget.Budget{
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{Email: email, Role: collab.CollaboratorRoleOwner},
			},
		},
	}

	budgetWithoutOwner := &budget.Budget{
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{Email: email, Role: collab.CollaboratorRoleViewer},
			},
		},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool

		on func(*fields)
	}{
		{
			name: "Happy path",
			args: args{
				ctx:       ctx,
				email:     email,
				budgetIDs: []string{budgetID},
			},
			wantErr: false,
			on: func(f *fields) {
				f.dal.
					On("GetBudget", ctx, budgetID).
					Return(budgetWithOwner, nil).
					Once()
				f.dal.
					On("GetRef", ctx, budgetID).
					Return(&firestore.DocumentRef{ID: budgetID}, nil).
					Once()
				f.labelsMock.
					On("DeleteManyObjectsWithLabels", ctx, []*firestore.DocumentRef{{ID: budgetID}}).
					Return(nil).
					Once()
			},
		}, {
			name: "Unauthorized to delete budget",
			args: args{
				ctx:       ctx,
				email:     email,
				budgetIDs: []string{budgetID},
			},
			wantErr: true,
			on: func(f *fields) {
				f.dal.
					On("GetBudget", ctx, budgetID).
					Return(budgetWithoutOwner, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				dal:            &dal.Budgets{},
				collab:         &collabMock.Icollab{},
				caOwnerChecker: &caOwnerCheckersMock.CheckCAOwnerInterface{},
				labelsMock:     &labelsMocks.Labels{},
			}
			s := &BudgetsService{
				dal:            tt.fields.dal,
				collab:         tt.fields.collab,
				caOwnerChecker: tt.fields.caOwnerChecker,
				labelsDal:      tt.fields.labelsMock,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := s.DeleteMany(tt.args.ctx, tt.args.email, tt.args.budgetIDs); (err != nil) != tt.wantErr {
				t.Errorf("BudgetService.ShareBudget() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBudgetsService_validateRecipients(t *testing.T) {
	type args struct {
		recipients    []string
		collaborators []collab.Collaborator
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Happy path when user is a valid collaborator",
			args: args{
				recipients: []string{"someUser@doit-intl.com"},
				collaborators: []collab.Collaborator{
					{
						Email: "someUser@doit-intl.com",
						Role:  collab.CollaboratorRoleViewer,
					},
					{
						Email: "SecondUser@doit-intl.com",
						Role:  collab.CollaboratorRoleEditor,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Happy path when user is a valid slack user",
			args: args{
				recipients:    []string{"someUser@someDomain.slack.com"},
				collaborators: []collab.Collaborator{},
			},
			wantErr: false,
		},
		{
			name: "Returns error when user is not a collaborator and not a slack user",
			args: args{
				recipients: []string{"badUser@doit-intl.com"},
				collaborators: []collab.Collaborator{
					{
						Email: "someUser@doit-intl.com",
						Role:  collab.CollaboratorRoleViewer,
					},
					{
						Email: "SecondUser@doit-intl.com",
						Role:  collab.CollaboratorRoleEditor,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateRecipients(tt.args.recipients, tt.args.collaborators); (err != nil) != tt.wantErr {
				t.Errorf("validateRecipients() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBudgetService_UpdateEnforcedByMeteringField(t *testing.T) {
	type fields struct {
		dal            *dal.Budgets
		collab         *collabMock.Icollab
		caOwnerChecker *caOwnerCheckersMock.CheckCAOwnerInterface
	}

	type args struct {
		ctx           context.Context
		budgetID      string
		collaborators []collab.Collaborator
		recipients    []string
	}

	ctx := context.Background()
	doerEmail := "test@doit.com"
	customerEmail := "customer@test.com"

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool

		on func(*fields)
	}{
		{
			name: "if customer is in collaborators set enforcedByMetering true",
			args: args{
				ctx:      ctx,
				budgetID: budgetID,
				collaborators: []collab.Collaborator{
					{Email: doerEmail, Role: collab.CollaboratorRoleOwner},
					{Email: customerEmail, Role: collab.CollaboratorRoleViewer},
				},
				recipients: []string{doerEmail},
			},
			wantErr: false,
			on: func(f *fields) {
				f.dal.
					On("UpdateBudgetEnforcedByMetering", ctx, budgetID, true).
					Return(nil).
					Once()
			},
		}, {
			name: "if customer is in recipients set enforcedByMetering true",
			args: args{
				ctx:      ctx,
				budgetID: budgetID,
				collaborators: []collab.Collaborator{
					{Email: doerEmail, Role: collab.CollaboratorRoleOwner},
				},
				recipients: []string{doerEmail, customerEmail},
			},
			wantErr: false,
			on: func(f *fields) {
				f.dal.
					On("UpdateBudgetEnforcedByMetering", ctx, budgetID, true).
					Return(nil).
					Once()
			},
		}, {
			name: "if no customer in collaborators and recipients set enforcedByMetering false",
			args: args{
				ctx:      ctx,
				budgetID: budgetID,
				collaborators: []collab.Collaborator{
					{Email: doerEmail, Role: collab.CollaboratorRoleOwner},
				},
				recipients: []string{doerEmail},
			},
			wantErr: false,
			on: func(f *fields) {
				f.dal.
					On("UpdateBudgetEnforcedByMetering", ctx, budgetID, false).
					Return(nil).
					Once()
			},
		}, {
			name: "UpdateBudgetEnforcedByMetering returns error",
			args: args{
				ctx:           ctx,
				budgetID:      budgetID,
				collaborators: []collab.Collaborator{},
				recipients:    []string{},
			},
			wantErr: true,
			on: func(f *fields) {
				f.dal.
					On("UpdateBudgetEnforcedByMetering", ctx, budgetID, false).
					Return(errors.New("error")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				dal:            &dal.Budgets{},
				collab:         &collabMock.Icollab{},
				caOwnerChecker: &caOwnerCheckersMock.CheckCAOwnerInterface{},
			}
			s := &BudgetsService{
				dal:            tt.fields.dal,
				collab:         tt.fields.collab,
				caOwnerChecker: tt.fields.caOwnerChecker,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := s.UpdateEnforcedByMeteringField(tt.args.ctx, tt.args.budgetID, tt.args.collaborators, tt.args.recipients, nil); (err != nil) != tt.wantErr {
				t.Errorf("BudgetService.UpdateBudgetEnforcedByMetering() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
