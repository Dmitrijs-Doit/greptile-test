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

func setupServices() (*AppFirestore, *mocks.DocumentsHandler) {
	fs, err := firestore.NewClient(context.Background(),
		common.TestProjectID,
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		panic(err)
	}

	dh := &mocks.DocumentsHandler{}

	return &AppFirestore{
			firestoreClientFun: func(ctx context.Context) *firestore.Client {
				return fs
			},
			documentsHandler: dh,
		},
		dh
}

func TestNewFirestoreCustomersDAL(t *testing.T) {
	_, err := NewAppFirestore(context.Background(), common.TestProjectID)
	assert.NoError(t, err)

	d := NewAppFirestoreWithClient(nil)
	assert.NotNil(t, d)
}

func TestAppDAL_GetServicesPlatformVersion(t *testing.T) {
	ctx := context.Background()
	d, dh := setupServices()

	dh.
		On("GetAll", mock.Anything).
		Return(nil, errors.New("fail")).
		Once()

	c, err := d.GetServicesPlatformVersion(ctx, "platform")

	assert.Error(t, err)
	assert.NotNil(t, err)
	assert.Equal(t, int64(0), c)

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

	c, err = d.GetServicesPlatformVersion(ctx, "platform")

	assert.NoError(t, err)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), c)
	assert.NotNil(t, c)
}
