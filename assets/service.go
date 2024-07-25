package assets

import (
	"errors"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AssetService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
}

var (
	ErrInvalidAssetID     = errors.New("invalid asset id")
	ErrInvalidRequestBody = errors.New("invalid request body")
)

func NewAssetService(loggerProvider logger.Provider, conn *connection.Connection) *AssetService {
	return &AssetService{
		loggerProvider,
		conn,
	}
}
