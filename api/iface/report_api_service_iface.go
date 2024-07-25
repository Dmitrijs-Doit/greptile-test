//go:generate mockery --output=../mocks --all
package iface

import (
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

type ReportAPIService interface {
	ListReports(ctx *gin.Context, conn *connection.Connection)
	RunReport(ctx *gin.Context, conn *connection.Connection)
}
