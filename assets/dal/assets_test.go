package dal

import (
	"context"
	"errors"
	"fmt"
	"log"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customers "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/tests"
	testPackage "github.com/doitintl/tests"
)

func setupAssets() (*AssetsFirestore, *mocks.DocumentsHandler) {
	fs, err := firestore.NewClient(context.Background(),
		common.TestProjectID,
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		panic(err)
	}

	dh := &mocks.DocumentsHandler{}

	return &AssetsFirestore{
		firestoreClientFun: func(ctx context.Context) *firestore.Client {
			return fs
		},
		documentsHandler: dh,
	}, dh
}

func TestNewAssetsDAL(t *testing.T) {
	_, err := NewAssetsFirestore(context.Background(), common.TestProjectID)
	assert.NoError(t, err)

	a := NewAssetsFirestoreWithClient(nil)
	assert.NotNil(t, a)
}

func TestAssetsDAL_ListGCPAssets(t *testing.T) {
	ctx := context.Background()
	a, dh := setupAssets()

	dh.
		On("GetAll", mock.Anything).
		Return(nil, errors.New("fail")).
		Once()

	c, err := a.ListGCPAssets(ctx)

	assert.Error(t, err)
	assert.Nil(t, c)

	dh.
		On("GetAll", mock.Anything).
		Return([]iface.DocumentSnapshot{}, nil).
		Once()

	c, err = a.ListGCPAssets(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, c)

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

	c, err = a.ListGCPAssets(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, c)
	assert.Len(t, c, 1)

	dh.
		On("GetAll", mock.Anything).
		Return(func() []iface.DocumentSnapshot {
			snap := &mocks.DocumentSnapshot{}
			snap.On("DataTo", mock.Anything).Return(errors.New("fail"))
			return []iface.DocumentSnapshot{
				snap,
			}
		}(), nil).
		Once()

	c, err = a.ListGCPAssets(ctx)

	assert.Error(t, err)
	assert.Nil(t, c)
}

func NewAssetsFirestoreWithClientMock(ctx context.Context) *AssetsFirestore {
	logging, err := logger.NewLogging(ctx)
	if err != nil {
		log.Printf("main: could not initialize logging. error %s", err)
		return nil
	}

	// Initialize db connections clients
	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		log.Printf("main: could not initialize db connections. error %s", err)
		return nil
	}

	return NewAssetsFirestoreWithClient(conn.Firestore)
}

func TestListBaseAssets(t *testing.T) {
	ctx := context.Background()
	s := NewAssetsFirestoreWithClientMock(ctx)

	if err := tests.LoadTestData("Assets"); err != nil {
		t.Error(err)
	}

	result, err := s.ListBaseAssets(ctx, common.Assets.AmazonWebServicesStandalone)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, result[0].Customer.ID, "s5x8qLHg0EGIJVBFayAb")
	assert.Equal(t, result[1].Customer.ID, "bWBwhhI3gIRZre40dhz8")
}

func TestAssetsFirestore_ListStandaloneGCPAssets(t *testing.T) {
	ctx := context.Background()
	s := NewAssetsFirestoreWithClientMock(ctx)

	if err := tests.LoadTestData("Assets"); err != nil {
		t.Error(err)
	}

	result, err := s.ListStandaloneGCPAssets(ctx)
	if err != nil {
		t.Error(err)
	}

	for _, asset := range result {
		assert.Equal(t, asset.Customer.ID, "s5x8qLHg0EGIJVBFayAb")
		assert.Equal(t, asset.Contract.ID, "OBkkNX9LuY3p7czXVdTE")
		assert.Equal(t, asset.Entity.ID, "kWYuwEYQfZHwnM0uCqhI")
		assert.Equal(t, asset.AssetType, "google-cloud-standalone")
	}
}

func TestAssetsFirestore_GetCustomerGCPAssetsWithTypes(t *testing.T) {
	ctx := context.Background()
	s := NewAssetsFirestoreWithClientMock(ctx)

	if err := tests.LoadTestData("Assets"); err != nil {
		t.Error(err)
	}

	customerDAL, err := customers.NewCustomersFirestore(ctx, "doitintl-cmp-dev")
	if err != nil {
		t.Error(err)
	}

	customerRef := customerDAL.GetRef(ctx, "s5x8qLHg0EGIJVBFayAb")

	result, err := s.GetCustomerGCPAssetsWithTypes(ctx, customerRef, []string{pkg.AssetGoogleCloud, pkg.AssetStandaloneGoogleCloud})
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(result), 1)

	_, err = s.GetCustomerGCPAssetsWithTypes(ctx, customerRef, []string{})
	assert.Equal(t, err, ErrGCPTypesIsEmpty)

	_, err = s.GetCustomerGCPAssetsWithTypes(ctx, customerRef, []string{pkg.AssetGoogleCloud, "someInvalidType"})
	assert.Equal(t, err, ErrGCPTypeInvalid)
}

func TestAssetsFirestore_GetCustomerAWSAssets(t *testing.T) {
	ctx := context.Background()
	s := NewAssetsFirestoreWithClientMock(ctx)

	if err := tests.LoadTestData("Assets"); err != nil {
		t.Error(err)
	}

	type inOut struct {
		customerID   string
		expectedID   string
		resultLength int
	}

	correctID := "ppVGgqCVzBsOVfwL8wRi"
	unknownID := "fake"
	standaloneID := "s5x8qLHg0EGIJVBFayAb"

	input_outputs := []inOut{
		{
			customerID:   correctID,
			expectedID:   "ppVGgqCVzBsOVfwL8wRi",
			resultLength: 1,
		},
		{
			customerID:   unknownID,
			expectedID:   "",
			resultLength: 0,
		},
		{
			customerID:   standaloneID,
			expectedID:   "",
			resultLength: 0,
		},
	}

	for _, input_output := range input_outputs {
		result, err := s.GetCustomerAWSAssets(ctx, input_output.customerID)
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, input_output.resultLength, len(result))

		for _, asset := range result {
			assert.Equal(t, asset.Customer.ID, input_output.expectedID)
			assert.Equal(t, asset.AssetType, pkg.AssetAWS)
		}
	}
}

func TestAssetsFirestore_getAssets(t *testing.T) {
	ctx := context.Background()
	s := NewAssetsFirestoreWithClientMock(ctx)

	customerDAL, err := customers.NewCustomersFirestore(ctx, "doitintl-cmp-dev")
	if err != nil {
		t.Error(err)
	}

	if err := tests.LoadTestData("Assets"); err != nil {
		t.Error(err)
	}

	standaloneRef := customerDAL.GetRef(ctx, "s5x8qLHg0EGIJVBFayAb")

	result, err := s.getAssets(ctx, standaloneRef, "amazon-web-services-standalone")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 1, len(result))

	doesNotExist := customerDAL.GetRef(ctx, "example")

	result, err = s.getAssets(ctx, doesNotExist, "")
	assert.Errorf(t, err, "asset type not provided")
	assert.Nil(t, result)
}

func TestAssetsFirestore_GetAWSStandaloneAssets(t *testing.T) {
	ctx := context.Background()
	s := NewAssetsFirestoreWithClientMock(ctx)

	customerDAL, err := customers.NewCustomersFirestore(ctx, "doitintl-cmp-dev")
	if err != nil {
		t.Error(err)
	}

	if err := tests.LoadTestData("Assets"); err != nil {
		t.Error(err)
	}

	standaloneRef := customerDAL.GetRef(ctx, "s5x8qLHg0EGIJVBFayAb")

	result, err := s.GetAWSStandaloneAssets(ctx, standaloneRef)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 1, len(result))

	doesNotExist := customerDAL.GetRef(ctx, "example")

	result, err = s.GetAWSStandaloneAssets(ctx, doesNotExist)
	assert.Equal(t, 0, len(result))
}

func TestAssetsFirestore_getAWSAssetsFromIterator(t *testing.T) {
	a, dh := setupAssets()

	dh.
		On("GetAll", mock.Anything).
		Return(nil, errors.New("err")).
		Once()

	snap := mocks.DocumentSnapshot{}

	snap.On("DataTo", mock.Anything).Return(errors.New("err"))

	dh.
		On("GetAll", mock.Anything).
		Return([]iface.DocumentSnapshot{&snap}, nil).
		Once()

	type args struct {
		iter *firestore.DocumentIterator
	}

	tests := []struct {
		name    string
		args    args
		want    []*pkg.AWSAsset
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"iterator fails",
			args{},
			nil,
			func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
		{
			"data to fails",
			args{},
			nil,
			func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := a.getAWSAssetsFromIterator(tt.args.iter)
			if !tt.wantErr(t, err, fmt.Sprintf("getAWSAssetsFromIterator(%v)", tt.args.iter)) {
				return
			}

			assert.Equalf(t, tt.want, got, "getAWSAssetsFromIterator(%v)", tt.args.iter)
		})
	}
}

func TestAssetsDAL_HasSharedPayerAWSAssets(t *testing.T) {
	iter := mock.AnythingOfType("*firestore.DocumentIterator")

	errExample := errors.New("things have gone terribly wrong")
	customerRef := &firestore.DocumentRef{
		ID: "mr_customer",
	}

	cloudhealthProperties := pkg.CloudHealthAccountInfo{
		CustomerName: "mr_customer",
		CustomerID:   666,
	}

	sharedPayerProperties := pkg.AWSProperties{
		CloudHealth: &cloudhealthProperties,
		OrganizationInfo: &pkg.OrganizationInfo{
			PayerAccount: &domain.PayerAccount{
				AccountID: "1",
			},
		},
	}

	sharedPayerAsset := pkg.AWSAsset{
		BaseAsset: pkg.BaseAsset{
			AssetType: pkg.AssetAWS,
			Customer:  customerRef,
		},
		Properties: &sharedPayerProperties,
	}

	dedicatedPayerAsset := pkg.AWSAsset{
		BaseAsset: pkg.BaseAsset{
			AssetType: pkg.AssetAWS,
			Customer:  customerRef,
		},
	}

	type fields struct {
		fsClient   firestore.Client
		docHandler mocks.DocumentsHandler
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
		want    bool
	}{
		{
			name: "returns true if AWS asset with CloudHealth ID present",
			on: func(f *fields) {
				f.docHandler.On("GetAll", mock.Anything).
					Return(func() []iface.DocumentSnapshot {
						snap := &mocks.DocumentSnapshot{}
						snap.On("ID").Return("mr_customer")
						snap.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*pkg.AWSAsset)
								*arg = sharedPayerAsset
							}).Once()
						return []iface.DocumentSnapshot{
							snap,
						}
					}(), nil).
					Once()
			},
			want: true,
		},
		{
			name: "returns true if AWS asset with CloudHealth ID present and dedicated payer asset present",
			on: func(f *fields) {
				f.docHandler.On("GetAll", mock.Anything).
					Return(func() []iface.DocumentSnapshot {
						snap2 := &mocks.DocumentSnapshot{}
						snap2.On("ID").Return("mr_customer_2")
						snap2.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*pkg.AWSAsset)
								*arg = dedicatedPayerAsset
							}).Once()
						snap := &mocks.DocumentSnapshot{}
						snap.On("ID").Return("mr_customer")
						snap.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*pkg.AWSAsset)
								*arg = sharedPayerAsset
							}).Once()
						return []iface.DocumentSnapshot{
							snap, snap2,
						}
					}(), nil).
					Once()
			},
			want: true,
		},
		{
			name: "returns false if assets have no CloudHealth properties",
			on: func(f *fields) {
				f.docHandler.On("GetAll", mock.Anything).
					Return(func() []iface.DocumentSnapshot {
						snap2 := &mocks.DocumentSnapshot{}
						snap2.On("ID").Return("mr_customer_2")
						snap2.On("DataTo", mock.Anything).Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*pkg.AWSAsset)
								*arg = dedicatedPayerAsset
							}).Once()
						return []iface.DocumentSnapshot{
							snap2,
						}
					}(), nil).
					Once()
			},
			want: false,
		},
		{
			name: "failed to get snapshot from iterator",
			on: func(f *fields) {
				f.docHandler.On("GetAll", iter).
					Return(nil, errExample).Once()
			},
			wantErr: errExample,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &AssetsFirestore{
				firestoreClientFun: func(ctx context.Context) *firestore.Client {
					return &fields.fsClient
				},
				documentsHandler: &fields.docHandler,
			}

			got, err := s.HasSharedPayerAWSAssets(context.Background(), customerRef)

			assert.Equal(t, got, tt.want)

			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAssetsFirestore_GetAWSAssetFromAccountNumber(t *testing.T) {
	ctx := context.Background()
	s := NewAssetsFirestoreWithClientMock(ctx)

	if err := testPackage.LoadTestData("Assets"); err != nil {
		t.Error(err)
	}

	type args struct {
		accountNumber string
		opts          []QueryOption
	}

	tcs := []struct {
		name    string
		args    args
		want    *pkg.AWSAsset // only asset properties are tested though
		wantErr bool
	}{
		{
			name: "aws-standalone",
			args: args{
				accountNumber: fixtureAssetAWSStandalone().Properties.AccountID,
			},
			want: fixtureAssetAWSStandalone(),
		},
		{
			name: "aws-resold",
			args: args{
				accountNumber: fixtureAssetAWSResold().Properties.AccountID,
			},
			want: fixtureAssetAWSResold(),
		},
		{
			name: "aws-resold-with-query-opts",
			args: args{
				accountNumber: fixtureAssetAWSResold().Properties.AccountID,
				opts: []QueryOption{
					WithWhereQuery(WhereQuery{
						Path:  "type",
						Op:    "==",
						Value: pkg.AssetAWS,
					}),
				},
			},
			want: fixtureAssetAWSResold(),
		},
		{
			name: "expected-error-null-results",
			args: args{
				accountNumber: fixtureAssetAWSResold().Properties.AccountID,
				opts: []QueryOption{
					WithWhereQuery(WhereQuery{
						Path:  "type",
						Op:    "==",
						Value: pkg.AssetStandaloneAWS,
					}),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tcs {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.GetAWSAssetFromAccountNumber(ctx, tt.args.accountNumber, tt.args.opts...)
			if err != nil {
				if tt.wantErr {
					return
				}

				t.Error(err)
			}

			if got.Properties == nil {
				t.Errorf("unexpected nil 'Properties' attributes, got: %+v", got)
			}

			if !cmp.Equal(tt.want.Properties, got.Properties) {
				t.Errorf(cmp.Diff(tt.want.Properties, got.Properties))
			}
		})
	}
}
