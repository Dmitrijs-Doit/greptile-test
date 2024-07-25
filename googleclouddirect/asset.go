package googleclouddirect

import (
	"errors"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AssetService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
}

func NewAssetService(loggerProvider logger.Provider, conn *connection.Connection) *AssetService {
	return &AssetService{
		loggerProvider,
		conn,
	}
}

var (
	ErrorAssetsNotFound             = errors.New("asset not found")
	ErrorAssetStateIsOtherThanError = errors.New("asset is not in an error state")
)

type CopyJobMetadata struct {
	Progress float64 `firestore:"progress" binding:"required"`
	Reason   string  `firestore:"reason" binding:"required"`
	Status   string  `firestore:"status" binding:"required"`
	Action   string  `firestore:"action"`
}
