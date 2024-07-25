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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	testPackage "github.com/doitintl/tests"
)

func setupCustomers() (*CustomersFirestore, *mocks.DocumentsHandler) {
	fs, err := firestore.NewClient(context.Background(),
		common.TestProjectID,
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		panic(err)
	}

	dh := &mocks.DocumentsHandler{}

	return &CustomersFirestore{
		firestoreClientFun: func(ctx context.Context) *firestore.Client {
			return fs
		},
		documentsHandler: dh,
	}, dh
}

func TestNewFirestoreCustomersDAL(t *testing.T) {
	_, err := NewCustomersFirestore(context.Background(), common.TestProjectID)
	assert.NoError(t, err)

	d := NewCustomersFirestoreWithClient(nil)
	assert.NotNil(t, d)
}

func TestCustomersDAL_GetCustomer(t *testing.T) {
	ctx := context.Background()
	d, dh := setupCustomers()

	dh.
		On("Get", mock.Anything, mock.AnythingOfType("*firestore.DocumentRef")).
		Return(func() iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(nil)
			snap.On("Snapshot").Return(&firestore.DocumentSnapshot{})
			snap.On("ID").Return("testCustomerId")
			return snap
		}(), nil).
		Once()

	c, err := d.GetCustomer(ctx, "testCustomerId")

	assert.NoError(t, err)
	assert.NotNil(t, c)

	dh.
		On("Get", mock.Anything, mock.AnythingOfType("*firestore.DocumentRef")).
		Return(func() iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(fmt.Errorf("fail"))
			return snap
		}(), nil).
		Once()

	c, err = d.GetCustomer(ctx, "testCustomerId")
	assert.Nil(t, c)
	assert.Error(t, err)

	dh.
		On("Get", mock.Anything, mock.AnythingOfType("*firestore.DocumentRef")).
		Return(nil, fmt.Errorf("fail")).
		Once()

	c, err = d.GetCustomer(ctx, "testCustomerId")
	assert.Nil(t, c)
	assert.Error(t, err)

	dh.
		On("Get", mock.Anything, mock.AnythingOfType("*firestore.DocumentRef")).
		Return(nil, status.Error(codes.NotFound, "item not found, should fail")).
		Once()

	c, err = d.GetCustomer(ctx, "testCustomerId")
	assert.Nil(t, c)
	assert.ErrorIs(t, err, doitFirestore.ErrNotFound)

	c, err = d.GetCustomer(ctx, "")
	assert.Nil(t, c)
	assert.Error(t, err, "invalid customer id")
}

func TestCustomerFirestore_DeleteCustomer(t *testing.T) {
	type args struct {
		ctx        context.Context
		customerID string
	}

	if err := testPackage.LoadTestData("Customers"); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	d, err := NewCustomersFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedErr error
	}{
		{
			name: "err on empty customerID",
			args: args{
				ctx:        ctx,
				customerID: "",
			},
			wantErr:     true,
			expectedErr: ErrInvalidCustomerID,
		},
		{
			name: "success deleting customer",
			args: args{
				ctx:        ctx,
				customerID: "111_to_be_deleted",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := d.DeleteCustomer(tt.args.ctx, tt.args.customerID)
			if err != nil {
				if !tt.wantErr || err.Error() != tt.expectedErr.Error() {
					t.Errorf("CustomerFirestore.DeleteCustomer() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				return
			}

			_, err = d.GetCustomer(ctx, tt.args.customerID)
			assert.ErrorIs(t, err, doitFirestore.ErrNotFound)
		})
	}
}

func TestCustomerFirestore_GetMSAzureCustomers(t *testing.T) {
	if err := testPackage.LoadTestData("Customers"); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	d, err := NewCustomersFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Fatal(err)
	}

	docSnaps, err := d.GetMSAzureCustomers(ctx)
	if err != nil {
		t.Errorf("CustomerFirestore.GetMSAzureCustomers() error = %v", err)
		return
	}
	// successfully retrieves MS Azure customers
	assert.Equal(t, 1, len(docSnaps))
	assert.Equal(t, "QN5kJkJ0vCFcPILaLqOF", docSnaps[0].Ref.ID)
}

func TestCustomerFirestore_GetCustomerAccountTeam(t *testing.T) {
	if err := testPackage.LoadTestData("Customers"); err != nil {
		t.Fatal(err)
	}

	if err := testPackage.LoadTestData("AccountManagers"); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	d, err := NewCustomersFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Fatal(err)
	}

	accountManagerList, err := d.GetCustomerAccountTeam(ctx, "jRRyh8x04k1c29Nq7ywZ")
	if err != nil {
		t.Errorf("CustomerFirestore.GetCustomerAccountTeam() error = %v", err)
		return
	}

	assert.Equal(t, 4, len(accountManagerList))
}
