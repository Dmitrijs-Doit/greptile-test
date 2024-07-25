package iface

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/bqutils"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/schema"
	"github.com/doitintl/hello/scheduled-tasks/looker/domain"
)

type AssetsServiceIface interface {
	LoadLookerContractsToBQ(ctx *gin.Context, request domain.UpdateTableInterval) error
	CreateLookerRows(ctx *gin.Context, contracts []*pkg.Contract, updateTableInterval []time.Time) (map[time.Time][]schema.BillingRow, error)
	GetTableLoaderPayload(ctx context.Context, billingRows []schema.BillingRow) (*bqutils.BigQueryTableLoaderParams, error)
	LookerBigQueryTableLoader(ctx context.Context, loadAttributes bqutils.BigQueryTableLoaderParams, partition time.Time, tableExists bool) error
}
