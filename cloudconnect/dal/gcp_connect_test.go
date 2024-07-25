package dal

import (
	"context"
	"errors"
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

func setupServices() (*GcpConnect, *mocks.DocumentsHandler) {
	fs, err := firestore.NewClient(context.Background(),
		common.TestProjectID,
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		panic(err)
	}

	dh := &mocks.DocumentsHandler{}

	return &GcpConnect{
			firestoreClientFun: func(ctx context.Context) *firestore.Client {
				return fs
			},
			documentsHandler: dh,
		},
		dh
}

func TestNewGcpConnect(t *testing.T) {
	_, err := NewGcpConnect(context.Background(), common.TestProjectID)
	assert.NoError(t, err)

	d := NewGcpConnectWithClient(nil)
	assert.NotNil(t, d)
}

func Test_GetCredentials(t *testing.T) {
	ctx := context.Background()
	d, dh := setupServices()

	dh.
		On("GetAll", mock.Anything).
		Return(nil, errors.New("not found")).
		Once()

	cred, err := d.GetCredentials(ctx, "customer-id")
	assert.Error(t, err)
	assert.Nil(t, cred)

	dh.
		On("GetAll", mock.Anything).
		Return(func() []iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(nil)
			return []iface.DocumentSnapshot{
				snap,
			}
		}(), nil).
		Once()

	cred, err = d.GetCredentials(ctx, "customer-id")
	assert.NoError(t, err)
	assert.NotNil(t, cred)
}

func Test_GetCredentialByOrg(t *testing.T) {
	ctx := context.Background()
	d, dh := setupServices()

	dh.
		On("Get", mock.Anything, mock.Anything).
		Return(nil, errors.New("not found")).
		Once()

	cred, err := d.GetCredentialByOrg(ctx, nil)
	assert.Error(t, err)
	assert.Nil(t, cred)

	dh.
		On("Get", mock.Anything, mock.Anything).
		Return(func() iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(nil)
			return snap
		}(), nil).Once()

	cred, err = d.GetCredentialByOrg(ctx, nil)
	assert.NoError(t, err)
	assert.NotNil(t, cred)
}

func Test_GetConnectDetails(t *testing.T) {
	ctx := context.Background()
	d, dh := setupServices()

	dh.
		On("Get", mock.Anything, mock.Anything).
		Return(nil, errors.New("not found")).
		Once()

	creds, err := d.GetClientOption(ctx, nil)
	assert.Error(t, err)
	assert.Nil(t, creds)

	dh.
		On("Get", mock.Anything, mock.Anything).
		Return(func() iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(nil)
			return snap
		}(), nil).Once()

	creds, err = d.GetClientOption(ctx, nil)
	// decryptSymmetric error
	assert.Error(t, err)
	assert.Nil(t, creds)
}

func Test_GetSinkParams(t *testing.T) {
	d, _ := setupServices()
	res := d.GetSinkParams("sink-destination")
	assert.NotNil(t, res)
	assert.Equal(t, sinkFilter, res.Filter)
	assert.Equal(t, true, res.BigqueryOptions.UsePartitionedTables)
}
