package dal

import (
	"context"
	"fmt"
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

func setupAwsServices() (*AwsConnect, *mocks.DocumentsHandler) {
	fs, err := firestore.NewClient(context.Background(),
		common.TestProjectID,
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		panic(err)
	}

	dh := mocks.DocumentsHandler{}

	return &AwsConnect{
			firestoreClientFun: func(ctx context.Context) *firestore.Client {
				return fs
			},
			documentsHandler: &dh,
		},
		&dh
}

func TestNewAwsConnect(t *testing.T) {
	_, err := NewAwsConnect(context.Background())
	assert.NoError(t, err)

	d := NewAwsConnectWithClient(nil)
	assert.NotNil(t, d)
}

func TestAwsConnect_GetSpot0CustomerFlags(t *testing.T) {
	a, dh := setupAwsServices()
	dh.
		On("Get", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("not found")).
		Once()

	spot0CustomerFlags, err := a.GetSpot0CustomerFlags(context.Background(), "")
	assert.Error(t, err)
	assert.Nil(t, spot0CustomerFlags)

	dh.
		On("Get", mock.Anything, mock.Anything).
		Return(func() iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(nil, nil)
			return snap
		}(), nil).Once()

	spot0CustomerFlags, err = a.GetSpot0CustomerFlags(context.Background(), "")
	assert.NoError(t, err)
	assert.NotNil(t, spot0CustomerFlags)
}

func TestAwsConnect_SetSpot0CustomerFlags(t *testing.T) {
	a, dh := setupAwsServices()
	dh.
		On("Set", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()

	err := a.SetSpot0CustomerFlags(context.Background(), "")
	assert.NoError(t, err)

	dh.
		On("Set", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("error")).
		Once()

	err = a.SetSpot0CustomerFlags(context.Background(), mock.Anything)
	assert.Error(t, err)
}

func TestAwsConnect_GetCustomerAdmins(t *testing.T) {
	a, dh := setupAwsServices()
	dh.On("GetAll", mock.Anything).
		Return(func() []iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(nil)
			return []iface.DocumentSnapshot{
				snap,
			}
		}(), nil).Once()

	admins, err := a.GetCustomerAdmins(context.Background(), "")
	assert.NotNil(t, admins)
	assert.NoError(t, err)

	dh.On("GetAll", mock.Anything).
		Return(nil, fmt.Errorf("error")).Once()

	admins, err = a.GetCustomerAdmins(context.Background(), "")
	assert.Error(t, err)
	assert.Nil(t, admins)
}
