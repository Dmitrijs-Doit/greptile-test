package assets

import (
	"context"

	"github.com/doitintl/cloudtasks/iface"
	accessService "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/access"
	accessIface "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/access/iface"
	mpaDal "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	entityDal "github.com/doitintl/hello/scheduled-tasks/entity/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

//go:generate mockery --name IAWSAssetsService --output ./mocks --outpkg mocks --case=underscore

type IAWSAssetsService interface {
	GetAssetFromAccountNumber(ctx context.Context, accountNumber string) (*pkg.AWSAsset, error)
	UpdateAssetsAllMPA(ctx context.Context) error
	UpdateAssetsMPA(ctx context.Context, mpaID string) error
	UpdateManualAssetsMPA(ctx context.Context, mpaID string) error
	UpdateStandaloneAssets(ctx context.Context, customerID, accountID string) error
	ClearAllFlexsaveAssets(ctx context.Context) error
}

type AWSAssetsService struct {
	loggerProvider   logger.Provider
	conn             *connection.Connection
	assetSettingsDAL assetsDal.AssetSettings
	assetsDAL        assetsDal.Assets
	customersDAL     customerDal.Customers
	entitiesDAL      entityDal.Entites
	mpaDAL           mpaDal.MasterPayerAccounts
	flexsaveAPI      flexapi.FlexAPI
	awsAccessService accessIface.Access
	cloudTaskClient  iface.CloudTaskClient
}

func NewAWSAssetsService(log logger.Provider, conn *connection.Connection, cloudTaskClient iface.CloudTaskClient) (*AWSAssetsService, error) {
	firestoreFun := conn.Firestore

	flexAPIService, err := flexapi.NewFlexAPIService()
	if err != nil {
		return nil, err
	}

	return &AWSAssetsService{
		log,
		conn,
		assetsDal.NewAssetSettingsFirestoreWithClient(firestoreFun),
		assetsDal.NewAssetsFirestoreWithClient(firestoreFun),
		customerDal.NewCustomersFirestoreWithClient(firestoreFun),
		entityDal.NewEntitiesFirestoreWithClient(firestoreFun),
		mpaDal.NewMasterPayerAccountDALWithClient(firestoreFun(context.Background())),
		flexAPIService,
		accessService.New(),
		cloudTaskClient,
	}, nil
}
