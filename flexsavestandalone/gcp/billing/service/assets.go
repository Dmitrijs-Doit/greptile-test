package service

import (
	"context"

	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Assets struct {
	loggerProvider logger.Provider
	*connection.Connection
	assetsDAL *assetsDal.AssetsFirestore
}

func NewAssets(log logger.Provider, conn *connection.Connection) *Assets {
	return &Assets{
		loggerProvider: log,
		Connection:     conn,
		assetsDAL:      assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
	}
}

func (a *Assets) GetAssets(ctx context.Context) ([]*pkg.GCPAsset, error) {
	return a.assetsDAL.ListStandaloneGCPAssets(ctx)
}
