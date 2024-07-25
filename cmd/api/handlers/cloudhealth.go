package handlers

import (
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/gin-gonic/gin"
)

func SyncCloudhealthCustomers(ctx *gin.Context) error {
	amazonwebservices.SyncCloudhealthCustomers(ctx)

	return nil
}
