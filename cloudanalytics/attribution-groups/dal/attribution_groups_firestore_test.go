package dal

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	labelsDALIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
	labelsDALMocks "github.com/doitintl/hello/scheduled-tasks/labels/dal/mocks"
	testPackage "github.com/doitintl/tests"
)

var ctx = context.Background()

var attributionGroupID = "PVF999AAAAIErQXpfDcf"

const pathForDocRefs = "projects/1/databases/1/documents/1/1"

func NewFirestoreWithMockLabels(labelsMock labelsDALIface.Labels) (*AttributionGroupsFirestore, error) {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	fun := func(ctx context.Context) *firestore.Client {
		return fs
	}

	return &AttributionGroupsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
		batchProvider:      doitFirestore.NewBatchProvider(fun, 0),
		labelsDal:          labelsMock,
	}, nil
}

func TestNewFirestoreAttributionGroupsDAL(t *testing.T) {
	_, err := NewAttributionGroupsFirestore(ctx, common.TestProjectID)
	assert.NoError(t, err)

	d := NewAttributionGroupsFirestoreWithClient(nil)
	assert.NotNil(t, d)
}

func TestAttributionGroupsFirestore_Get(t *testing.T) {
	type args struct {
		ctx                context.Context
		attributionGroupID string
	}

	tests := []struct {
		name        string
		args        args
		wantRole    collab.CollaboratorRole
		wantErr     bool
		expectedErr error
	}{
		{
			name: "err on empty attributionGroupID",
			args: args{
				ctx:                ctx,
				attributionGroupID: "",
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrNoAttributionGroupID,
		},
		{
			name: "err attributionGroup not found",
			args: args{
				ctx:                ctx,
				attributionGroupID: "invalidID",
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrNotFound,
		},
		{
			name: "success getting attributionGroup",
			args: args{
				ctx:                ctx,
				attributionGroupID: attributionGroupID,
			},
			wantErr:  false,
			wantRole: collab.CollaboratorRoleOwner,
		},
	}

	d, err := NewAttributionGroupsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("AttributionGroups"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.Get(tt.args.ctx, tt.args.attributionGroupID)
			if (err != nil) != tt.wantErr || err != tt.expectedErr {
				t.Errorf("AttributionGroupsFirestore.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if got.Access.Collaborators[0].Role != tt.wantRole {
					t.Errorf("AttributionGroupsFirestore.Get() = %v, want %v", got, tt.wantRole)
				}
			}
		})
	}
}

func TestAttributionsGroupsFirestore_GetAll(t *testing.T) {
	ctx := context.Background()

	type args struct {
		ctx                 context.Context
		attributionGroupIDs []string
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
		expectedLen int
	}{
		{
			name: "success getting attribution groups",
			args: args{
				ctx: ctx,
				attributionGroupIDs: []string{
					"5XMpfSAWFXbENFK7hVRc",
					"E0V1NkVK7LcIe3b0HRNf",
					"non-existing",
				},
			},
			expectedLen: 2,
			wantErr:     false,
		},
		{
			name: "err on empty attribution groups ids",
			args: args{
				ctx:                 ctx,
				attributionGroupIDs: nil,
			},
			wantErr:     true,
			expectedErr: domainAttributions.ErrEmptyAttributionRefsList,
		},
	}

	d, err := NewAttributionGroupsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("AttributionGroups"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attributionGroupIDs []*firestore.DocumentRef
			for _, id := range tt.args.attributionGroupIDs {
				attributionGroupIDs = append(attributionGroupIDs, d.GetRef(ctx, id))
			}

			got, err := d.GetAll(tt.args.ctx, attributionGroupIDs)
			if (err != nil) != tt.wantErr || err != tt.expectedErr {
				t.Errorf("AttributionGroupsFirestore.GetAttributions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				assert.Equal(t, tt.expectedLen, len(got))
			}
		})
	}
}

func TestAttributionGroupsFirestore_Share(t *testing.T) {
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
				ctx: ctx,
				id:  attributionGroupID,
				collaborators: []collab.Collaborator{
					{
						Email: "test1@a.com",
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
				ctx: ctx,
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

	if err := testPackage.LoadTestData("AttributionGroups"); err != nil {
		t.Error(err)
	}

	d, err := NewAttributionGroupsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := d.Share(tt.args.ctx, tt.args.id, tt.args.collaborators, tt.args.public); (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsFirestore.Share() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAttributionGroupsFirestore_Create(t *testing.T) {
	type args struct {
		ctx              context.Context
		attributionGroup *attributiongroups.AttributionGroup
	}

	d, err := NewAttributionGroupsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("AttributionGroups"); err != nil {
		t.Error(err)
	}

	public := collab.PublicAccessEdit

	testAttributionGroup := &attributiongroups.AttributionGroup{
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{
					Email: "test@test.com",
					Role:  collab.CollaboratorRoleOwner,
				},
			},
			Public: &public,
		},
		Customer: &firestore.DocumentRef{
			ID:   "customerID",
			Path: pathForDocRefs,
		},
		Name: "test",
		Attributions: []*firestore.DocumentRef{
			{
				ID:   "attributionID",
				Path: pathForDocRefs,
			},
		},
		Type: domainAttributions.ObjectTypeCustom,
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "err on empty attributionGroup",
			args: args{
				ctx:              ctx,
				attributionGroup: nil,
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrInvalidAttributionGroup,
		},
		{
			name: "success creating attributionGroup",
			args: args{
				ctx:              ctx,
				attributionGroup: testAttributionGroup,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := d.Create(tt.args.ctx, tt.args.attributionGroup); (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsFirestore.Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAttributionGroupsFirestore_Update(t *testing.T) {
	type args struct {
		ctx                context.Context
		attributionGroupID string
		attributionGroup   *attributiongroups.AttributionGroup
	}

	d, err := NewAttributionGroupsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("AttributionGroups"); err != nil {
		t.Error(err)
	}

	existingAttributionGroup, _ := d.Get(ctx, attributionGroupID)

	updatedExistingAttributionGroup := existingAttributionGroup
	updatedExistingAttributionGroup.Name = "updated name"

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "err on empty attributionGroupID",
			args: args{
				ctx:                ctx,
				attributionGroupID: "",
				attributionGroup:   &attributiongroups.AttributionGroup{},
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrNoAttributionGroupID,
		},
		{
			name: "err attributionGroup not found",
			args: args{
				ctx:                ctx,
				attributionGroupID: attributionGroupID,
				attributionGroup:   &attributiongroups.AttributionGroup{},
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrInvalidAttributionGroup,
		},
		{
			name: "success updating attributionGroup",
			args: args{
				ctx:                ctx,
				attributionGroupID: attributionGroupID,
				attributionGroup:   updatedExistingAttributionGroup,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := d.Update(tt.args.ctx, tt.args.attributionGroupID, tt.args.attributionGroup); err != nil {
				if !tt.wantErr || err != tt.expectedErr {
					t.Errorf("AttributionGroupsFirestore.Update() actual error = %v, expected error = %v, wantErr %v", err, tt.expectedErr, tt.wantErr)
				}
			}
		})
	}
}

func TestAttributionGroupsFirestore_Delete(t *testing.T) {
	type fields struct {
		labelsDal *labelsDALMocks.Labels
	}

	type args struct {
		ctx                context.Context
		attributionGroupID string
	}

	if err := testPackage.LoadTestData("AttributionGroups"); err != nil {
		t.Error(err)
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
		fields      fields
		on          func(f *fields)
	}{
		{
			name: "success deleting attributionGroup",
			args: args{
				ctx:                ctx,
				attributionGroupID: attributionGroupID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.labelsDal.On("DeleteObjectWithLabels", ctx, mock.AnythingOfType("*firestore.DocumentRef")).Return(nil)
			},
		},
		{
			name: "err on empty attributionGroupID",
			args: args{
				ctx:                ctx,
				attributionGroupID: "",
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrNoAttributionGroupID,
		},
		{
			name: "error - delete object with labels error",
			args: args{
				ctx:                ctx,
				attributionGroupID: attributionGroupID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.labelsDal.On("DeleteObjectWithLabels", ctx, mock.AnythingOfType("*firestore.DocumentRef")).Return(errors.New("error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				labelsDal: &labelsDALMocks.Labels{},
			}

			d, err := NewFirestoreWithMockLabels(tt.fields.labelsDal)
			if err != nil {
				t.Error(err)
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err = d.Delete(tt.args.ctx, tt.args.attributionGroupID)

			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsFirestore.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("AttributionGroupsFirestore.Delete() error = %v, expectedError %v", err, tt.expectedErr)
			}
		})
	}
}

func TestAttributionGroupsFirestore_List(t *testing.T) {
	pathForCustomerRef := "projects/doitintl-cmp-dev/databases/(default)/documents/customers/ImoC9XkrutBysJvyqlBm"

	type args struct {
		ctx         context.Context
		customerRef *firestore.DocumentRef
	}

	d, err := NewAttributionGroupsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("AttributionGroups"); err != nil {
		t.Error(err)
	}

	tests := []struct {
		name  string
		args  args
		email string
	}{
		{
			name: "success listing preset, custom and managed attributionGroups",
			args: args{
				ctx:         ctx,
				customerRef: &firestore.DocumentRef{ID: "ImoC9XkrutBysJvyqlBm", Path: pathForCustomerRef},
			},
			email: "requester@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := d.List(tt.args.ctx, tt.args.customerRef, tt.email)

			assert.Len(t, got, 4, "Provided slice is not of length 4")
		})
	}
}
