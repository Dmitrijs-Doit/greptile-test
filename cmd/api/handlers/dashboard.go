package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/contract"
	"github.com/doitintl/hello/scheduled-tasks/dashboard"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/renewals"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/gsuite"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
)

func DebtAnalyticsHandler(ctx *gin.Context) error {
	invoices.DebtAnalyticsHandler(ctx)

	return nil
}

func UpdateCommitmentContracts(ctx *gin.Context) error {
	contract.UpdateCommitmentContracts(ctx)

	return nil
}

func GSuiteRenewalsHandler(ctx *gin.Context) error {
	renewals.GSuiteRenewalsHandler(ctx)

	return nil
}

func Office365RenewalsHandler(ctx *gin.Context) error {
	renewals.Office365RenewalsHandler(ctx)

	return nil
}

func ZendeskRenewalsHandler(ctx *gin.Context) error {
	renewals.ZendeskRenewalsHandler(ctx)

	return nil
}

func BetterCloudRenewalsHandler(ctx *gin.Context) error {
	renewals.BetterCloudRenewalsHandler(ctx)

	return nil
}

func CopyLicenseToDashboardGSuite(ctx *gin.Context) error {
	gsuite.CopyLicenseToDashboard(ctx)

	return nil
}

func CopyLicenseToDashboardMicrosoft(ctx *gin.Context) error {
	if err := microsoft.CopyLicenseToDashboard(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func GetCustomerDashboards(ctx *gin.Context) error {
	dashboard.GetCustomerDashboards(ctx)

	return nil
}
