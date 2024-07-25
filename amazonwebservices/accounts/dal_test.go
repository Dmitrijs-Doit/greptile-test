package accounts

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func setupServices() (Dal, *mocks.DocumentsHandler) {
	fs, err := firestore.NewClient(context.Background(),
		common.TestProjectID,
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		panic(err)
	}

	dh := &mocks.DocumentsHandler{}

	return &dal{
		firestoreClient: func(ctx context.Context) *firestore.Client {
			return fs
		},
		documentsHandler: dh,
	}, dh
}

func TestAccountsDal_findAccountByID(t *testing.T) {
	contextMock := mock.MatchedBy(func(_ context.Context) bool { return true })
	ref := mock.AnythingOfType("*firestore.DocumentRef")
	err := errors.New("oh no")

	d, dh := setupServices()

	tests := []struct {
		name    string
		on      func()
		wantErr error
		want    *Account
		params  string
	}{
		{
			name: "dal returns error",
			on: func() {
				dh.On("Get", contextMock, ref).Return(nil, err).Once()
			},
			wantErr: err,
			want:    nil,
			params:  "id1",
		},
		{
			name: "dal returns items",
			on: func() {
				dh.On("Get", contextMock, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(func() iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("DataTo",
							mock.Anything).
							Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*Account)
								*arg = Account{}
							})
						return snap
					}(), nil).Once()

			},
			wantErr: nil,
			want:    &Account{},
			params:  "id1",
		},
		{
			name: "data to returns error",
			on: func() {

				dh.On("Get", contextMock, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(func() iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("DataTo",
							mock.Anything).
							Return(err)
						return snap
					}(), nil).Once()

			},
			wantErr: err,
			want:    nil,
			params:  "id1",
		},
		{
			name: "dal returns error if no account found",
			on: func() {
				dh.On("Get", contextMock, mock.AnythingOfType("*firestore.DocumentRef")).
					Return(nil, errNotFound)
			},
			wantErr: errNotFound,
			want:    nil,
			params:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.on != nil {
				tt.on()
			}

			got, err := d.findAccountByID(context.Background(), tt.params)

			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AccountsDal_findAccountByID = %v, want %v", got, tt.want)
			}
		})
	}
}
