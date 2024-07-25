package dal

import (
	"context"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	testPackage "github.com/doitintl/tests"
)

func setupBudgets() (*BudgetsFirestore, error) {
	if err := testPackage.LoadTestData("Budgets"); err != nil {
		return nil, err
	}

	const projectID = "doitintl-cmp-dev"

	fs, err := firestore.NewClient(context.Background(), projectID)
	if err != nil {
		return nil, err
	}

	budgetsFirestore := NewBudgetsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		})

	return budgetsFirestore, nil
}

func TestNewBudgetsFirestoreDAL(t *testing.T) {
	budgetsFirestore, err := setupBudgets()
	if err != nil {
		t.Error(err)
	}

	assert.NotNil(t, budgetsFirestore)
}

func TestBudgetsFirestoreDAL_GetBudget(t *testing.T) {
	ctx := context.Background()

	budgetsFirestore, err := setupBudgets()
	if err != nil {
		t.Error(err)
	}

	c, err := budgetsFirestore.GetBudget(ctx, "PnOD7lsJWD2IseckPdHM")

	assert.NoError(t, err)
	assert.NotNil(t, c)

	c, err = budgetsFirestore.GetBudget(ctx, "no-existing-id")
	assert.Nil(t, c)
	assert.ErrorIs(t, err, doitFirestore.ErrNotFound)

	c, err = budgetsFirestore.GetBudget(ctx, "")
	assert.Nil(t, c)
	assert.Error(t, err, "invalid budget id")
}

func TestBudgetsFirestore_Share(t *testing.T) {
	type args struct {
		ctx           context.Context
		id            string
		collaborators []collab.Collaborator
		public        *collab.PublicAccess
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx: context.Background(),
				id:  "PnOD7lsJWD2IseckPdHM",
				collaborators: []collab.Collaborator{
					{
						Email: "newowner@a.com",
						Role:  collab.CollaboratorRoleOwner,
					},
				},
				public: nil,
			},
			wantErr: false,
		},
		{
			name: "error on update",
			args: args{
				ctx: context.Background(),
				id:  "invalid ID",
				collaborators: []collab.Collaborator{
					{
						Email: "test1@a.com",
						Role:  collab.CollaboratorRoleOwner,
					},
				},
				public: nil,
			},
			wantErr: true,
		},
	}

	budgetsFirestore, err := setupBudgets()
	if err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := budgetsFirestore.Share(tt.args.ctx, tt.args.id, tt.args.collaborators, tt.args.public); (err != nil) != tt.wantErr {
				t.Errorf("budgetsFirestore.Share() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBudgetsFirestore_UpdateBudgetRecipients(t *testing.T) {
	type args struct {
		ctx                     context.Context
		id                      string
		recipients              []string
		recipientsSlackChannels []common.SlackChannel
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx: context.Background(),
				id:  "PnOD7lsJWD2IseckPdHM",
				recipients: []string{
					"firstRecipient@a.com",
					"secondRecipient@a.com",
				},
				recipientsSlackChannels: []common.SlackChannel{
					{
						Name:       "moonactive-com",
						ID:         "C0123TZ9TLN",
						Shared:     true,
						Type:       "public",
						CustomerID: "2Gi0e4pPA3wsfJNOOohW",
						Workspace:  "",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "error on update recipients",
			args: args{
				ctx:                     context.Background(),
				id:                      "non-existing-id",
				recipients:              []string{},
				recipientsSlackChannels: []common.SlackChannel{},
			},
			wantErr: true,
		},
	}

	budgetsFirestore, err := setupBudgets()
	if err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := budgetsFirestore.UpdateBudgetRecipients(tt.args.ctx, tt.args.id, tt.args.recipients, tt.args.recipientsSlackChannels); (err != nil) != tt.wantErr {
				t.Errorf("budgetsFirestore.Share() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBudgetsFirestore_UpdateBudgetEnforcedByMetering(t *testing.T) {
	type args struct {
		ctx                context.Context
		id                 string
		enforcedByMetering bool
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx:                context.Background(),
				id:                 "PnOD7lsJWD2IseckPdHM",
				enforcedByMetering: true,
			},
			wantErr: false,
		},
		{
			name: "error on update",
			args: args{
				ctx:                context.Background(),
				id:                 "non-existing-id",
				enforcedByMetering: true,
			},
			wantErr: true,
		},
	}

	budgetsFirestore, err := setupBudgets()
	if err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := budgetsFirestore.UpdateBudgetEnforcedByMetering(tt.args.ctx, tt.args.id, tt.args.enforcedByMetering); (err != nil) != tt.wantErr {
				t.Errorf("budgetsFirestore.UpdateBudgetEnforcedByMetering() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
