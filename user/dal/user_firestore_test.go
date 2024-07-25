package dal

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/iface"
	testUtils "github.com/doitintl/tests"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/stretchr/testify/assert"

	testPackage "github.com/doitintl/tests"
)

func TestUserFirestoreDAL_NewUserFirestoreDAL(t *testing.T) {
	_, err := NewUserFirestoreDAL(context.Background(), "doitintl-cmp-dev")
	assert.NoError(t, err)

	d := NewUserFirestoreDALWithClient(nil)
	assert.NotNil(t, d)
}

func TestUserFirestoreDAL_GetUserByEmail(t *testing.T) {
	ctx := context.Background()

	if err := testPackage.LoadTestData("Users"); err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Customers"); err != nil {
		t.Error(err)
	}

	userFirestoreDAL, _ := NewUserFirestoreDAL(ctx, "doitintl-cmp-dev")

	customerID := "4RvPLdzUE8lBmGzzgGPa"

	wrongCustomerID := "jRRyh8x04k1c29Nq7ywZ"

	existingEmail := "dror@carodox.com"
	nonExistingEmail := "non-existing-email@doit.com"

	type args struct {
		ctx        context.Context
		email      string
		customerID string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "get existing user when email and customerRef exist",
			args: args{
				ctx:        ctx,
				email:      existingEmail,
				customerID: customerID,
			},
			wantErr: false,
		},
		{
			name: "get error when email does not match customer",
			args: args{
				ctx:        ctx,
				email:      existingEmail,
				customerID: wrongCustomerID,
			},
			wantErr: true,
		},
		{
			name: "get error when email does not exist",
			args: args{
				ctx:        ctx,
				email:      nonExistingEmail,
				customerID: customerID,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := userFirestoreDAL.GetUserByEmail(
				tt.args.ctx,
				tt.args.email,
				tt.args.customerID); (err != nil) != tt.wantErr {
				t.Errorf("accountFirestoreDAL.GetUserByEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserFirestoreDAL_Get(t *testing.T) {
	ctx := context.Background()

	if err := testPackage.LoadTestData("Users"); err != nil {
		t.Error(err)
	}

	userFirestoreDAL, _ := NewUserFirestoreDAL(ctx, "doitintl-cmp-dev")

	existingID := "yW7LrCcVHZGEaDLu7gkf"
	nonExistingID := "non-existing-id"

	type args struct {
		ctx context.Context
		id  string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "get existing user when ID exists",
			args: args{
				ctx: ctx,
				id:  existingID,
			},
			wantErr: false,
		},
		{
			name: "get error when ID does not exist",
			args: args{
				ctx: ctx,
				id:  nonExistingID,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := userFirestoreDAL.Get(tt.args.ctx, tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("accountFirestoreDAL.Get() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserFirestoreDAL_GetUsersWithRecentEngagement(t *testing.T) {
	type fields struct {
		fsClient   firestore.Client
		docHandler mocks.DocumentsHandler
	}

	tests := []struct {
		name    string
		fields  fields
		want    []common.User
		wantErr bool
		on      func(f *fields)
	}{
		{
			name: "get users with recent engagement",
			fields: fields{
				fsClient:   firestore.Client{},
				docHandler: mocks.DocumentsHandler{},
			},
			want:    []common.User{{ID: "user-1"}},
			wantErr: false,
			on: func(f *fields) {
				f.docHandler.On("GetAll", mock.Anything).
					Return(func() []iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*common.User)
								*arg = common.User{
									ID: "user-1",
								}
							}).Once()
						return []iface.DocumentSnapshot{
							snap,
						}
					}(), nil).
					Once()

			},
		},
		{
			name: "doc handler returns error",
			fields: fields{
				fsClient:   firestore.Client{},
				docHandler: mocks.DocumentsHandler{},
			},
			want:    nil,
			wantErr: true,
			on: func(f *fields) {
				f.docHandler.On("GetAll", mock.Anything).
					Return(nil, errors.New("err")).
					Once()
			},
		},
		{
			name: "unable to decode document",
			fields: fields{
				fsClient:   firestore.Client{},
				docHandler: mocks.DocumentsHandler{},
			},
			want:    nil,
			wantErr: true,
			on: func(f *fields) {
				f.docHandler.On("GetAll", mock.Anything).
					Return(func() []iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("DataTo", mock.Anything).Return(errors.New("err"))
						return []iface.DocumentSnapshot{
							snap,
						}
					}(), nil).
					Once()

			},
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)

			d := &UserFirestoreDAL{
				firestoreClientFun: func(ctx context.Context) *firestore.Client {
					return &tt.fields.fsClient
				},
				documentsHandler: &tt.fields.docHandler,
			}

			got, err := d.GetUsersWithRecentEngagement(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equalf(t, tt.want, got, "GetUsersWithRecentEngagement")
		})
	}
}

func TestUserFirestoreDAL_GetLastUserEngagementTimeForCustomer(t *testing.T) {
	longTimeAgo := time.Now().Add(-time.Hour)
	recently := time.Now().Add(-time.Minute)

	type fields struct {
		fsClient   firestore.Client
		docHandler mocks.DocumentsHandler
	}

	tests := []struct {
		name    string
		fields  fields
		want    *time.Time
		wantErr string
		on      func(f *fields)
	}{
		{
			name:    "gets error from doc handler",
			fields:  fields{},
			want:    nil,
			wantErr: "error from doc handler",
			on: func(f *fields) {
				f.docHandler.On("GetAll", mock.Anything).
					Return(nil, errors.New("error from doc handler")).
					Once()
			},
		},

		{
			name: "returns nil as no activity found",
			fields: fields{
				fsClient:   firestore.Client{},
				docHandler: mocks.DocumentsHandler{},
			},
			want:    nil,
			wantErr: "",
			on: func(f *fields) {
				f.docHandler.On("GetAll", mock.Anything).
					Return(nil, nil).
					Once()
			},
		},

		{
			name: "returns most recent activity time",
			fields: fields{
				fsClient:   firestore.Client{},
				docHandler: mocks.DocumentsHandler{},
			},
			want:    &recently,
			wantErr: "",
			on: func(f *fields) {
				f.docHandler.On("GetAll", mock.Anything).
					Return(func() []iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*common.User)
								*arg = common.User{
									ID:        "user-1",
									LastLogin: longTimeAgo,
								}
							}).Once()
						snap2 := &mocks.DocumentSnapshot{}
						snap2.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*common.User)
								*arg = common.User{
									ID:        "user-2",
									LastLogin: recently,
								}
							}).Once()
						snap3 := &mocks.DocumentSnapshot{}
						snap3.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*common.User)
								*arg = common.User{
									ID:        "user-3",
									LastLogin: longTimeAgo,
								}
							}).Once()

						return []iface.DocumentSnapshot{
							snap, snap2, snap3,
						}
					}(), nil).
					Once()

			},
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)

			d := &UserFirestoreDAL{
				firestoreClientFun: func(ctx context.Context) *firestore.Client {
					return &tt.fields.fsClient
				},
				documentsHandler: &tt.fields.docHandler,
			}
			got, gotErr := d.GetLastUserEngagementTimeForCustomer(context.Background(), "customer1")
			testUtils.WantErr(t, gotErr, tt.wantErr)

			assert.Equalf(t, tt.want, got, "GetLastUserEngagementTimeForCustomer")
		})
	}
}
