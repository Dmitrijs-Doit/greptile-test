package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	slackgo "github.com/slack-go/slack"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	awsTablemgmt "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/services/tablemanagement"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal"
	attributionsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/attributiontier"
	attributionTierServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/attributiontier/iface"
	backfill "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/handlers"
	budgetsSvc "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/service"
	domainAnalyticsGCP "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	gcpBillingTableMgmt "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/service"
	gcpBillingTableMgmtIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/service/iface"
	domainHighCharts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/domain"
	highchartsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/service"
	highchartsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/service/iface"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	reportDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reporttier"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reporttier/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/stats"
	reportStatsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/stats/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/schedule"
	tabelMgmtSvc "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget"
	domainWidget "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	dashboardDAL "github.com/doitintl/hello/scheduled-tasks/dashboard/dal"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slack/service/slack"
	slackIface "github.com/doitintl/hello/scheduled-tasks/slack/service/slack/iface"
	"github.com/doitintl/hello/scheduled-tasks/times"
	tier "github.com/doitintl/tiers/service"
)

const (
	queryParamAllPartitions = "allPartitions"
	queryParamFrom          = "from"
	queryParamNumPartitions = "numPartitions"
	queryParamInterval      = "interval"
	queryParamAccountID     = "accountID"
)

// CloudAnalytics cloud analytics handlers
type CloudAnalytics struct {
	loggerProvider               logger.Provider
	conn                         *connection.Connection
	customerDal                  customerDAL.Customers
	cloudAnalytics               cloudanalytics.CloudAnalytics
	scheduledReports             *schedule.ScheduledReportsService
	widgetService                *widget.WidgetService
	budgets                      *budgetsSvc.BudgetsService
	slack                        slackIface.Slack
	scheduledWidgetUpdateService *widget.ScheduledWidgetUpdateService
	highcharts                   highchartsIface.IHighcharts
	GCPBillingTables             gcpBillingTableMgmtIface.BillingTableManagementService
	backfill                     *backfill.Backfill
	AWSBillingTables             awsTablemgmt.IBillingTableManagementService
	reportStatsService           reportStatsIface.ReportStatsService
	reportTierService            iface.ReportTierService
	attributionTierService       attributionTierServiceIface.AttributionTierService
}

// NewCloudAnalytics init new CloudAnalytics handlers
func NewCloudAnalytics(loggerProvider logger.Provider, conn *connection.Connection) *CloudAnalytics {
	reportDAL := reportDAL.NewReportsFirestoreWithClient(conn.Firestore)
	customerDAL := customerDAL.NewCustomersFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(loggerProvider, conn, reportDAL, customerDAL)
	if err != nil {
		panic(err)
	}

	budgets, err := budgetsSvc.NewBudgetsService(loggerProvider, conn)
	if err != nil {
		panic(err)
	}

	highcharts, err := highchartsService.NewHighcharts(loggerProvider, conn, budgets)
	if err != nil {
		panic(err)
	}

	scheduledReports, err := schedule.NewScheduledReportsService(loggerProvider, conn, highcharts)
	if err != nil {
		panic(err)
	}

	widgetService, err := widget.NewWidgetService(loggerProvider, conn)
	if err != nil {
		panic(err)
	}

	slack, err := slack.NewSlackService(loggerProvider, conn)
	if err != nil {
		panic(err)
	}

	dashboardAccessMetadataDAL := dashboardDAL.NewDashboardAccessMetadataFirestoreWithClient(conn.Firestore)

	reportStatsService, err := stats.NewReportStatsService(loggerProvider, reportDAL)
	if err != nil {
		panic(err)
	}

	tierService := tier.NewTiersService(conn.Firestore)

	attributionDAL := attributionsDal.NewAttributionsFirestoreWithClient(conn.Firestore)

	attrGroupDAL := dal.NewAttributionGroupsFirestoreWithClient(conn.Firestore)

	doitEmployeesService := doitemployees.NewService(conn)

	reportTierService := reporttier.NewReportTierService(
		loggerProvider,
		reportDAL,
		customerDAL,
		attributionDAL,
		attrGroupDAL,
		tierService,
		doitEmployeesService,
	)

	scheduledWidgetUpdateService, err := widget.NewScheduledWidgetUpdateService(
		loggerProvider,
		conn,
		widgetService,
		dashboardAccessMetadataDAL,
		reportStatsService,
	)
	if err != nil {
		panic(err)
	}

	awsBillingTables, err := awsTablemgmt.NewBillingTableManagementService(loggerProvider, conn, customerDAL)
	if err != nil {
		panic(err)
	}

	gcpBillingTables := gcpBillingTableMgmt.NewBillingTableManagementService(loggerProvider, conn)
	backfill := backfill.NewBackfill(loggerProvider, conn)

	attributionTierService := attributiontier.NewAttributionTierService(
		loggerProvider,
		attributionDAL,
		tierService,
		doitEmployeesService,
	)

	return &CloudAnalytics{
		loggerProvider,
		conn,
		customerDAL,
		cloudAnalytics,
		scheduledReports,
		widgetService,
		budgets,
		slack,
		scheduledWidgetUpdateService,
		highcharts,
		gcpBillingTables,
		backfill,
		awsBillingTables,
		reportStatsService,
		reportTierService,
		attributionTierService,
	}
}

func (h *CloudAnalytics) UpdateCurrenciesTable(ctx *gin.Context) error {
	err := h.cloudAnalytics.UpdateCurrenciesTable(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateAllCustomerDashboardReportWidgetsHandler(ctx *gin.Context) error {
	if err := h.scheduledWidgetUpdateService.UpdateAllCustomerDashboardReportWidgets(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateCustomerDashboardReportWidgetsHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	orgID := ctx.Query("orgId")

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginWidgets)
	ctx.Set(domainWidget.IsPrioritized, ctx.Query(domainWidget.IsPrioritized))

	if err := h.scheduledWidgetUpdateService.UpdateCustomerDashboardReportWidgetsHandler(ctx, customerID, orgID); err != nil {
		if err == widget.ErrTerminatedCustomer {
			return web.NewRequestError(err, http.StatusUnprocessableEntity)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) DeleteStaleDraftReportsHandler(ctx *gin.Context) error {
	if err := h.cloudAnalytics.DeleteStaleDraftReports(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) RefreshAllBudgetsHandler(ctx *gin.Context) error {
	if err := h.budgets.RefreshAllBudgets(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) TriggerBudgetsAlertsHandler(ctx *gin.Context) error {
	slackAlerts, err := h.budgets.TriggerBudgetsAlerts(ctx) //	send emails & return slack payloads
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	parsedPayload := h.attachImagesToSlackAlerts(ctx, slackAlerts)
	h.slack.PostMessages(ctx, parsedPayload)

	return web.Respond(ctx, nil, http.StatusOK)
}

// TODO remove after highcharts directory is refactored (CMP-2399)
func (h *CloudAnalytics) attachImagesToSlackAlerts(ctx *gin.Context, alerts map[*budgetsSvc.BudgetSlackAlert][]common.SlackChannel) map[*slackgo.MsgOption][]common.SlackChannel {
	l := h.loggerProvider(ctx)

	slackAlerts := make(map[*slackgo.MsgOption][]common.SlackChannel)

	for budgetSlackAlert, channels := range alerts {
		_, imageURLForecasted, err := h.highcharts.GetBudgetImages(ctx, budgetSlackAlert.BudgetID, channels[0].CustomerID, &domainHighCharts.SlackUnfurlFontSettings)
		if err != nil {
			l.Errorf("parseAlertsToSlack() error generating highcharts image for budget %s - reason: %s", budgetSlackAlert.BudgetID, err.Error())
		}

		msgOption := h.budgets.GetSlackFinalBlocks(ctx, imageURLForecasted, budgetSlackAlert.SlackBlocks)

		slackAlerts[&msgOption] = append(slackAlerts[&msgOption], channels...)
	}

	return slackAlerts
}

func (h *CloudAnalytics) TriggerForecastedDateAlertsHandler(ctx *gin.Context) error {
	if err := h.budgets.TriggerForecastedDateAlerts(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) RefreshBudgetUsageDataHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	email := ctx.GetString("email")
	customerID := ctx.Param("customerID")
	budgetID := ctx.Param("budgetID")

	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"budgetId":             budgetID,
	})

	if budgetID == "" {
		return web.NewRequestError(budgetsSvc.ErrMissingBudgetID, http.StatusInternalServerError)
	}

	if customerID == "" {
		return web.NewRequestError(budgetsSvc.ErrMissingCustomerID, http.StatusInternalServerError)
	}

	customer, err := h.customerDal.GetCustomer(ctx, customerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusNotFound)
	}

	if customer.Inactive() || customer.Terminated() {
		l.Warningf("customer %s is inactive or terminated, skipping budget refresh", customerID)
		return web.Respond(ctx, nil, http.StatusOK)
	}

	budget, err := h.budgets.GetBudget(ctx, budgetID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	// if customer is in presentation mode
	if customer.PresentationMode != nil && customer.PresentationMode.Enabled {
		// check if budget belongs to customer's presentation mode dataset
		if budget.Customer.ID != customer.PresentationMode.CustomerID {
			return web.NewRequestError(budgetsSvc.ErrUnauthorized, http.StatusForbidden)
		}

		// run budget refresh in presentation mode customer ID context
		customerID = customer.PresentationMode.CustomerID

		l.SetLabel(logger.LabelCustomerID, customerID)
	}

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginBudgets)

	if err := h.budgets.UpdateEnforcedByMeteringField(ctx, budgetID, budget.Collaborators, budget.Recipients, budget.Public); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.budgets.RefreshBudgetUsageData(ctx, customerID, budgetID, email); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateCustomersInfoTableHandler(ctx *gin.Context) error {
	if err := h.cloudAnalytics.UpdateCustomersInfoTable(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateCSPAccounts(ctx *gin.Context) error {
	allPartitions := ctx.Query(queryParamAllPartitions) == "true"
	fromDate := ctx.Query(queryParamFrom)

	var numPartitions int

	if v := ctx.Query(queryParamNumPartitions); v != "" {
		var err error

		numPartitions, err = strconv.Atoi(v)
		if err != nil || numPartitions < 0 {
			numPartitions = 0
		}
	}

	if err := h.AWSBillingTables.UpdateCSPAccounts(
		ctx,
		allPartitions,
		fromDate,
		numPartitions,
	); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateCSPAggregatedTableAWS(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtAws)

	// allPartitions == "true" -> Update all table partitions
	// allPartitions == "" (or any other value) -> (default) Update last partition based on current time
	allPartitions := ctx.Query(queryParamAllPartitions) == "true"

	fromDate := ctx.Query(queryParamFrom)

	var numPartitions int

	if v := ctx.Query(queryParamNumPartitions); v != "" {
		var err error

		numPartitions, err = strconv.Atoi(v)
		if err != nil || numPartitions < 0 {
			numPartitions = 0
		}
	}

	if err := h.AWSBillingTables.UpdateCSPAggregatedTable(
		ctx,
		allPartitions,
		fromDate,
		numPartitions,
	); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateAggregateAllAWSAccounts(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtAws)

	allPartitions := ctx.Query(queryParamAllPartitions) == "true"

	if errs := h.AWSBillingTables.UpdateAllAggregatedTablesAllCustomers(ctx, allPartitions); errs != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errors": errs})
		return nil
	}

	return nil
}

func (h *CloudAnalytics) UpdateAllAWSAggregatedTables(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtAws)

	suffix := ctx.Param(queryParamAccountID)
	if suffix == "" {
		return web.NewRequestError(awsTablemgmt.ErrSuffixIsEmpty, http.StatusBadRequest)
	}

	allPartitions := ctx.Query(queryParamAllPartitions) == "true"
	if errs := h.AWSBillingTables.UpdateAllAggregatedTables(ctx, suffix, allPartitions); errs != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errors": errs})
		return nil
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateAggregatedTableAWS(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtAws)

	interval := ctx.Query(queryParamInterval)
	if interval == "" {
		return web.NewRequestError(errors.New("interval query param is required"), http.StatusBadRequest)
	}

	suffix := ctx.Param(queryParamAccountID)
	if suffix == "" {
		return web.NewRequestError(errors.New("accountID is required"), http.StatusBadRequest)
	}

	// allPartitions == "true" -> Update all table partitions
	// allPartitions == "" (or any other value) -> (default) Update last partition based on current time
	allPartitions := ctx.Query(queryParamAllPartitions) == "true"
	if err := h.AWSBillingTables.UpdateAggregatedTable(ctx, suffix, interval, allPartitions); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *CloudAnalytics) InitRawTable(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	if err := h.GCPBillingTables.InitRawTable(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) InitRawResourceTable(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	if err := h.GCPBillingTables.InitRawResourceTable(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateRawTableLastPartition(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	if err := h.GCPBillingTables.UpdateRawTableLastPartition(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateRawResourceTableLastPartition(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	if err := h.GCPBillingTables.UpdateRawResourceTableLastPartition(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateDiscountsAWS(ctx *gin.Context) error {
	if err := h.AWSBillingTables.UpdateDiscounts(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateDiscountsGCP(ctx *gin.Context) error {
	if err := h.GCPBillingTables.UpdateDiscounts(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) ScheduledBillingAccountsTableUpdate(ctx *gin.Context) error {
	if err := h.GCPBillingTables.ScheduledBillingAccountsTableUpdate(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) StandaloneBillingUpdateEvents(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	if err := h.GCPBillingTables.StandaloneBillingUpdateEvents(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateAllBillingAccountsTableHandler(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	var input domainAnalyticsGCP.UpdateBillingAccountsTableInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.GCPBillingTables.UpdateBillingAccountsTable(ctx, input); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateCustomerBillingAccountTable(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	uri := ctx.Request.RequestURI

	billingAccountID, err := h.parseBillingAccountID(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	allPartitions := h.parseAllParitions(ctx)
	refreshMetadata := ctx.Query("refreshMetadata") == "true"
	assetType := ctx.Query("assetType")
	fromDate := h.parseFromDate(ctx)

	numPartitions, err := h.parseNumPartitions(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.GCPBillingTables.UpdateBillingAccountTable(ctx, uri, billingAccountID, allPartitions, refreshMetadata,
		assetType, fromDate, numPartitions); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func UpdateOrganizationIAMResources(ctx *gin.Context) error {
	googlecloud.UpdateOrganizationIAMResources(ctx)

	return nil
}

func (h *CloudAnalytics) UpdateIAMResources(ctx *gin.Context) error {
	if err := h.GCPBillingTables.UpdateIAMResources(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CloudAnalytics) UpdateCSPBillingAccounts(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	var params domainAnalyticsGCP.UpdateCspTaskParams
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	allPartitions := h.parseAllParitions(ctx)

	fromDate := h.parseFromDate(ctx)
	if fromDate != "" {
		if allPartitions {
			return web.NewRequestError(errors.New("conflict between from date and all partitions params"), http.StatusBadRequest)
		}

		if _, err := time.Parse(times.YearMonthDayLayout, fromDate); err != nil {
			return web.NewRequestError(fmt.Errorf("failed parsing from date with error: %s", err), http.StatusBadRequest)
		}
	}

	numPartitions, err := h.parseNumPartitions(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.GCPBillingTables.UpdateCSPBillingAccounts(ctx, params, numPartitions, allPartitions, fromDate); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *CloudAnalytics) AppendToTempCSPBillingAccountTable(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	billingAccountID := ctx.Param("billingAccountID")

	// updateAll == "true" -> this request is part of updating all accounts process
	// updateAll == "" (or any other value) -> otherwise
	updateAll := ctx.Query("updateAll") == "true"

	allPartitions := h.parseAllParitions(ctx)
	fromDate := h.parseFromDate(ctx)

	numPartitions, err := h.parseNumPartitions(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.GCPBillingTables.AppendToTempCSPBillingAccountTable(ctx, billingAccountID, updateAll, allPartitions, numPartitions, fromDate); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *CloudAnalytics) UpdateCSPTableAndDeleteTemp(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	billingAccountID := ctx.Query("account")

	allPartitions := h.parseAllParitions(ctx)
	fromDate := h.parseFromDate(ctx)

	numPartitions, err := h.parseNumPartitions(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.GCPBillingTables.UpdateCSPTableAndDeleteTemp(ctx, billingAccountID, allPartitions, fromDate, numPartitions); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *CloudAnalytics) JoinCSPTempTable(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	billingAccountID := ctx.Query("account")

	idx, err := strconv.Atoi(ctx.Query("idx"))
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	allPartitions := h.parseAllParitions(ctx)
	fromDate := h.parseFromDate(ctx)

	numPartitions, err := h.parseNumPartitions(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.GCPBillingTables.JoinCSPTempTable(ctx, billingAccountID, idx, allPartitions, fromDate, numPartitions); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *CloudAnalytics) UpdateCSPAggregatedTableGCP(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	billingAccountID := ctx.Query("account")

	allPartitions := h.parseAllParitions(ctx)

	if err := h.GCPBillingTables.UpdateCSPAggregatedTable(ctx, billingAccountID, allPartitions); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

type updateAggregatedTableParams struct {
	billingAccountID string
	interval         string
	fromDate         string
	numPartitions    int
	allPartitions    bool
}

func (h *CloudAnalytics) parseAllParitions(ctx *gin.Context) bool {
	// allPartitions == "true" -> Update all table partitions
	// allPartitions == "" (or any other value) -> (default) Update last partition or from specific date
	allPartitions := ctx.Query("allPartitions") == "true"

	return allPartitions
}

func (h *CloudAnalytics) parseBillingAccountID(ctx *gin.Context) (string, error) {
	billingAccountID := ctx.Param("billingAccountID")
	if billingAccountID == "" {
		return "", errors.New("billing account id is empty")
	}

	return billingAccountID, nil
}

func (h *CloudAnalytics) parseFromDate(ctx *gin.Context) string {
	// from == "YYYY-MM-DD" -> Update all partitions from a specific date (when not "allPartitions")
	return ctx.Query("from")
}

func (h *CloudAnalytics) parseNumPartitions(ctx *gin.Context) (int, error) {
	var numPartitions int

	if v := ctx.Query("numPartitions"); v != "" {
		var err error

		numPartitions, err = strconv.Atoi(v)
		if err != nil || numPartitions < 0 {
			return 0, errors.New("invalid number of partitions")
		}
	}

	return numPartitions, nil
}

func (h *CloudAnalytics) parseUpdateAggregatedTableParams(ctx *gin.Context) (*updateAggregatedTableParams, error) {
	billingAccountID := ctx.Param("billingAccountID")
	if billingAccountID == "" {
		return nil, errors.New("billing account id is empty")
	}

	interval := ctx.Query(queryParamInterval)
	if interval == "" {
		return nil, errors.New("interval query param is required")
	}

	allPartitions := h.parseAllParitions(ctx)

	fromDate := h.parseFromDate(ctx)

	numPartitions, err := h.parseNumPartitions(ctx)
	if err != nil {
		return nil, err
	}

	return &updateAggregatedTableParams{
		billingAccountID: billingAccountID,
		interval:         interval,
		fromDate:         fromDate,
		numPartitions:    numPartitions,
		allPartitions:    allPartitions,
	}, nil
}

func (h *CloudAnalytics) UpdateAggregatedTableGCP(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	params, err := h.parseUpdateAggregatedTableParams(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err = h.GCPBillingTables.UpdateAggregatedTable(
		ctx, params.billingAccountID, params.interval, params.fromDate, params.numPartitions, params.allPartitions)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *CloudAnalytics) UpdateAllGCPAggregatedTables(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtGcp)

	billingAccountID := ctx.Param("billingAccountID")
	allPartitions := h.parseAllParitions(ctx)
	fromDate := h.parseFromDate(ctx)

	numPartitions, err := h.parseNumPartitions(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	h.GCPBillingTables.UpdateAllAggregatedTables(
		ctx, billingAccountID, fromDate, numPartitions, allPartitions,
	)

	return nil
}

func (h *CloudAnalytics) HandleCustomerBackfill(ctx *gin.Context) error {
	return h.backfill.HandleCustomer(ctx)
}

func (h *CloudAnalytics) Query(ctx *gin.Context) error {
	var params cloudanalytics.RunQueryInput

	customer, err := h.customerDal.GetCustomerOrPresentationModeCustomer(ctx, ctx.Param("customerID"))
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}
	//if the customer is a demo customer (predefined customer used for presentation mode), we will run the query on their behalf
	if customer.PresentationMode != nil && customer.PresentationMode.IsPredefined {
		params.PresentationModeEnabled = true
	}

	params.CustomerID = customer.Snapshot.Ref.ID
	params.ReportID = ctx.Param("reportID")
	params.Email = ctx.GetString("email")

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      params.Email,
		logger.LabelCustomerID: params.CustomerID,
	})

	var qr cloudanalytics.QueryRequest

	if err := ctx.ShouldBindJSON(&qr); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if qr.Type == cloudanalytics.QueryRequestTypeAttribution {
		accessDeniedErr, err := h.attributionTierService.CheckAccessToQueryRequest(
			ctx,
			params.CustomerID,
			&qr,
		)
		if err != nil {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}

		if accessDeniedErr != nil {
			return web.Respond(ctx, accessDeniedErr, http.StatusForbidden)
		}
	}

	accessDeniedErr, err := h.reportTierService.CheckAccessToQueryRequest(
		ctx,
		params.CustomerID,
		&qr,
	)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if accessDeniedErr != nil {
		return web.Respond(ctx, accessDeniedErr, http.StatusForbidden)
	}

	qr.Origin = domainOrigin.QueryOriginFromContextOrFallback(ctx, domainOrigin.QueryOriginClient)

	result, err := h.cloudAnalytics.RunQuery(ctx, &qr, params)
	if err != nil {
		if errors.Is(err, tabelMgmtSvc.ErrReportOrganization) {
			return web.NewRequestError(err, http.StatusUnauthorized)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if result.Error != nil {
		status := result.Error.Status
		if status == 0 {
			status = http.StatusInternalServerError
		}

		return web.Respond(ctx, result, status)
	}

	if qr.Type == "report" {
		if err := h.reportStatsService.UpdateReportStats(ctx, qr.ID, qr.Origin, result.Details); err != nil {
			l.Errorf("failed to update report stats for report %s with error %s", qr.ID, err)
		}
	}

	return web.Respond(ctx, result, http.StatusOK)
}
