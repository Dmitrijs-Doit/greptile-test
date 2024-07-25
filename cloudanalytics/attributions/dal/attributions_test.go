package dal

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/doitintl/customerapi"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	labelsDALIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
	labelsDALMocks "github.com/doitintl/hello/scheduled-tasks/labels/dal/mocks"
	testPackage "github.com/doitintl/tests"
)

var ctx = context.Background()

func NewFirestoreWithMockLabels(labelsMock labelsDALIface.Labels) (*AttributionsFirestore, error) {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	fun := func(ctx context.Context) *firestore.Client {
		return fs
	}

	return &AttributionsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
		labelsDal:          labelsMock,
	}, nil
}

func setupAttributions() (*AttributionsFirestore, *mocks.DocumentsHandler) {
	fs, err := firestore.NewClient(context.Background(),
		common.TestProjectID,
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		panic(err)
	}

	dh := &mocks.DocumentsHandler{}

	return &AttributionsFirestore{
		firestoreClientFun: func(ctx context.Context) *firestore.Client {
			return fs
		},
		documentsHandler: dh,
	}, dh
}

func TestNewFirestoreAttributionsDAL(t *testing.T) {
	_, err := NewAttributionsFirestore(context.Background(), common.TestProjectID)
	assert.NoError(t, err)

	d := NewAttributionsFirestoreWithClient(nil)
	assert.NotNil(t, d)
}

func TestAttributionsDAL_GetAttribution(t *testing.T) {
	ctx := context.Background()
	d, dh := setupAttributions()

	dh.
		On("Get", mock.Anything, mock.AnythingOfType("*firestore.DocumentRef")).
		Return(func() iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(nil)
			snap.On("ID").Return("testAttributionId")
			return snap
		}(), nil).
		Once()

	r, err := d.GetAttribution(ctx, "testAttributionId")
	assert.NoError(t, err)
	assert.NotNil(t, r)

	dh.
		On("Get", mock.Anything, mock.AnythingOfType("*firestore.DocumentRef")).
		Return(func() iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(fmt.Errorf("fail"))
			return snap
		}(), nil).
		Once()

	r, err = d.GetAttribution(ctx, "testAttributionId")
	assert.Nil(t, r)
	assert.Error(t, err)

	dh.
		On("Get", mock.Anything, mock.AnythingOfType("*firestore.DocumentRef")).
		Return(nil, fmt.Errorf("fail")).
		Once()

	r, err = d.GetAttribution(ctx, "testAttributionId")
	assert.Nil(t, r)
	assert.Error(t, err)

	dh.
		On("Get", mock.Anything, mock.AnythingOfType("*firestore.DocumentRef")).
		Return(nil, status.Error(codes.NotFound, "attribution not found, should fail")).
		Once()

	r, err = d.GetAttribution(ctx, "testAttributionId")
	assert.Nil(t, r)
	assert.Error(t, err, attribution.ErrNotFound)

	r, err = d.GetAttribution(ctx, "")
	assert.Nil(t, r)
	assert.Error(t, err, attribution.ErrInvalidAttributionID)
}

func TestAttributionsDAL_ListAttributions(t *testing.T) {
	ctx := context.Background()
	d, dh := setupAttributions()
	customerRef := &firestore.DocumentRef{ID: "test-customer-id"}
	email := "requester@example.com"
	public := collab.PublicAccessEdit
	presetAttr := attribution.Attribution{
		ID:   "preset-attribution-id",
		Type: "preset",
		Access: collab.Access{
			Public: &public,
		},
	}

	customAttr := attribution.Attribution{
		ID:   "preset-attribution-id",
		Type: "custom",
		Access: collab.Access{
			Public: &public,
		},
	}

	req := &customerapi.Request{
		MaxResults: 100,
		Filters: []customerapi.Filter{
			{
				Field:    "type",
				Operator: "==",
				Value:    "custom",
			},
		},
		NextPageToken: "",
		SortBy:        firestore.DocumentID,
		SortOrder:     firestore.Asc,
		CustomerID:    "test-customer-id",
		Email:         email,
	}

	dh.On("GetAll", mock.Anything).
		Return(func() []iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(fmt.Errorf("fail"))
			return []iface.DocumentSnapshot{
				snap,
			}
		}(), nil).Twice()

	//error in DataTo
	r, err := d.ListAttributions(ctx, req, customerRef)
	assert.Nil(t, r)
	assert.Error(t, err)

	dh.On("GetAll", mock.Anything).
		Return(func() []iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				arg := args.Get(0).(*attribution.Attribution)
				*arg = presetAttr
			})
			snap.On("ID", mock.Anything).Return("preset-attribution-id")
			snap.On("Snapshot", mock.Anything).Return(&firestore.DocumentSnapshot{})

			return []iface.DocumentSnapshot{
				snap,
			}
		}(), nil).Once()

	dh.On("GetAll", mock.Anything).
		Return(func() []iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				arg := args.Get(0).(*attribution.Attribution)
				*arg = customAttr
			})
			snap.On("ID", mock.Anything).Return("custom-attribution-id")
			snap.On("Snapshot", mock.Anything).Return(&firestore.DocumentSnapshot{})

			return []iface.DocumentSnapshot{
				snap,
			}
		}(), nil).Once()

	req.Filters = []customerapi.Filter{}

	//happy path no error
	r, err = d.ListAttributions(ctx, req, customerRef)
	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, 2, len(r))
	assert.ElementsMatch(t, []string{"custom-attribution-id", "preset-attribution-id"}, []string{r[0].ID, r[1].ID})
}

func TestAttributionsDAL_GetAttributions(t *testing.T) {
	ctx := context.Background()

	type args struct {
		ctx             context.Context
		attributionsIDs []string
	}

	tests := []struct {
		name        string
		args        args
		wantRole    collab.CollaboratorRole
		wantErr     bool
		expectedErr error
	}{
		{
			name: "err on empty attributions ids",
			args: args{
				ctx:             ctx,
				attributionsIDs: nil,
			},
			wantErr:     true,
			expectedErr: attribution.ErrEmptyAttributionRefsList,
		},
		{
			name: "success getting attributions",
			args: args{
				ctx:             ctx,
				attributionsIDs: []string{"fBPGqV4qQKg51J9CoyqA", "GL0dPtG7mID22qW6tkA4", "KJFo9UpYbHrY5g28Dfno"},
			},
			wantErr:  false,
			wantRole: collab.CollaboratorRoleOwner,
		},
	}

	d, err := NewAttributionsFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Attributions"); err != nil {
		t.Error(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attributionsRefs []*firestore.DocumentRef
			for _, id := range tt.args.attributionsIDs {
				attributionsRefs = append(attributionsRefs, d.GetRef(ctx, id))
			}

			got, err := d.GetAttributions(tt.args.ctx, attributionsRefs)
			if (err != nil) != tt.wantErr || err != tt.expectedErr {
				t.Errorf("AttributionsFirestore.GetAttributions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				assert.Equal(t, 3, len(got))
			}
		})
	}
}

func TestAttributionsDAL_Delete(t *testing.T) {
	type fields struct {
		labelsDal *labelsDALMocks.Labels
	}

	type args struct {
		ctx           context.Context
		attributionID string
	}

	if err := testPackage.LoadTestData("Attributions"); err != nil {
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
			name: "success deleting attribution",
			args: args{
				ctx:           ctx,
				attributionID: "fBPGqV4qQKg51J9CoyqA",
			},
			wantErr: false,
			on: func(f *fields) {
				f.labelsDal.On("DeleteObjectWithLabels", ctx, mock.AnythingOfType("*firestore.DocumentRef")).Return(nil)
			},
		},
		{
			name: "error - delete object with labels error",
			args: args{
				ctx:           ctx,
				attributionID: "fBPGqV4qQKg51J9CoyqA",
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

			err = d.DeleteAttribution(tt.args.ctx, tt.args.attributionID)

			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionsFirestore.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("AttributionsFirestore.Delete() error = %v, expectedError %v", err, tt.expectedErr)
			}
		})
	}
}
