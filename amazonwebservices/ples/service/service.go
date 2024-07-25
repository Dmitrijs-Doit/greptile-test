package service

import (
	"context"

	mpaAccountsDAL "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/ples/dal"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/ples/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type PLESService struct {
	loggerProvider logger.Provider
	assetsDal      assetsDal.AssetSettings
	mpaDal         mpaAccountsDAL.MasterPayerAccounts
	flexsaveAPI    flexapi.FlexAPI
	plesDal        dal.PlesBigQueryDalIface
}

func NewPLESService(log logger.Provider, conn *connection.Connection) *PLESService {
	flexAPIService, err := flexapi.NewFlexAPIService()
	if err != nil {
		panic(err)
	}

	return &PLESService{
		log,
		assetsDal.NewAssetSettingsFirestoreWithClient(conn.Firestore),
		mpaAccountsDAL.NewMasterPayerAccountDALWithClient(conn.Firestore(context.Background())),
		flexAPIService,
		&dal.PlesBigQueryDal{},
	}
}

func (s *PLESService) UpdatePLESAccounts(ctx context.Context, accounts []domain.PLESAccount, forceUpdate bool) []error {
	if !forceUpdate {
		if errs := s.validatePLESAccounts(ctx, accounts); len(errs) > 0 {
			return errs
		}
	}

	file, err := createCsvFile(accounts)
	if err != nil {
		return []error{err}
	}

	if err := s.plesDal.UpdatePlesAccounts(ctx, file, accounts[0].InvoiceMonth.Format("20060102")); err != nil {
		return []error{err}
	}

	return nil
}
