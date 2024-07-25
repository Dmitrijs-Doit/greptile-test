package iface

import (
	"context"

	"github.com/gin-gonic/gin"
)

type FlexsaveStandalone interface {
	UpdateFlexsaveInvoicingData(ctx context.Context, invoiceMonthInput string, provider string) error
	FlexsaveDataWorker(ginCtx *gin.Context, customerID string, invoiceMonthInput string, provider string) error
}
