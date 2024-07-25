package api

import (
	"errors"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/sosodev/duration"

	"github.com/doitintl/auth"
	"github.com/doitintl/customerapi"
	domainAPI "github.com/doitintl/hello/scheduled-tasks/api/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	externalAPIService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/service"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	reportDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	reportDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/iface"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	reportTierServiceiface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reporttier/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/stats/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

type ReportAPIService struct {
	loggerProvider     logger.Provider
	conn               *connection.Connection
	cloudAnalytics     cloudanalytics.CloudAnalytics
	reportStatsService iface.ReportStatsService
	reportDAL          reportDalIface.Reports
	reportTierService  reportTierServiceiface.ReportTierService
}

func NewReportAPIService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	reportStatsService iface.ReportStatsService,
	reportTierService reportTierServiceiface.ReportTierService,
) (*ReportAPIService, error) {
	customerDal := customerDAL.NewCustomersFirestoreWithClient(conn.Firestore)
	reportDal := reportDAL.NewReportsFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(logger.FromContext, conn, reportDal, customerDal)
	if err != nil {
		return nil, err
	}

	return &ReportAPIService{
		loggerProvider,
		conn,
		cloudAnalytics,
		reportStatsService,
		reportDal,
		reportTierService,
	}, nil
}

func (s *ReportAPIService) ListReports(ctx *gin.Context, conn *connection.Connection) {
	l := s.loggerProvider(ctx)

	email := ctx.GetString("email")

	l.SetLabel("email", email)

	fs := s.conn.Firestore(ctx)

	// Verify customerID path param
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	request := domainAPI.ReportsRequest{
		MaxResults:      ctx.Request.URL.Query().Get("maxResults"),
		PageToken:       ctx.Request.URL.Query().Get("pageToken"),
		Filter:          ctx.Request.URL.Query().Get("filter"),
		MinCreationTime: ctx.Request.URL.Query().Get("minCreationTime"),
		MaxCreationTime: ctx.Request.URL.Query().Get("maxCreationTime"),
		SortOrder:       "desc",
		SortBy:          "createTime",
		CustomerID:      customerID,
		Email:           email,
	}

	// Parse request
	parsedRequest, err := customerapi.NewAPIRequest(request)
	if err != nil {
		errMsg := "Error parsing request with: " + err.Error()
		l.Error(errMsg)
		ctx.AbortWithStatus(http.StatusBadRequest)

		return
	}

	customerRef := fs.Collection("customers").Doc(customerID)

	// permissions: get reports that which the user is the owner, editor or viewer
	reportCollaboratorArr := []collab.Collaborator{
		{Email: email, Role: collab.CollaboratorRoleOwner},
		{Email: email, Role: collab.CollaboratorRoleEditor},
		{Email: email, Role: collab.CollaboratorRoleViewer},
	}

	reportsCollectionRef := fs.Collection("dashboards").Doc("google-cloud-reports").Collection("savedReports")

	// preset reports
	presetReportsQuery := reportsCollectionRef.
		Where("type", "==", "preset").
		Where("customer", "==", nil)

	// public reports
	publicReportsQuery := reportsCollectionRef.
		Where("type", "==", "custom").
		Where("customer", "==", customerRef).
		Where("public", common.In, []collab.CollaboratorRole{
			collab.CollaboratorRoleViewer,
			collab.CollaboratorRoleEditor,
		})

	// user reports
	sharedReportsQuery := reportsCollectionRef.
		Where("type", "==", "custom").
		Where("customer", "==", customerRef).
		Where("public", "==", nil).
		Where("collaborators", "array-contains-any", reportCollaboratorArr)

	// extracting owner from filters
	ownerFilter, _ := parsedRequest.GetFilterValue("owner").(string)

	// extracting type from filters
	typeFilter, _ := parsedRequest.GetFilterValue("type").(string)
	if typeFilter != "" && typeFilter != "preset" && typeFilter != "custom" {
		l.Error("invalid type filter: " + typeFilter)
		AbortMsg(ctx, http.StatusBadRequest, err, ErrorParam+"type")

		return
	}

	// extracting updateTime from filters
	updateTimeFilterStr, _ := parsedRequest.GetFilterValue("updateTime").(string)

	var updateTimeFilter int64

	if updateTimeFilterStr != "" {
		updateTimeFilterTime, err := common.MsToTime(updateTimeFilterStr)
		if err != nil {
			l.Error(err)
			AbortMsg(ctx, http.StatusBadRequest, err, ErrorParam+"updateTime")

			return
		}

		updateTimeFilter = updateTimeFilterTime.UnixMilli()
	}

	// extracting reportName from filters
	reportNameFilter, _ := parsedRequest.GetFilterValue("reportName").(string)

	if parsedRequest.MinCreationTime != nil {
		sharedReportsQuery = sharedReportsQuery.Where("timeCreated", ">=", parsedRequest.MinCreationTime)
		presetReportsQuery = presetReportsQuery.Where("timeCreated", ">=", parsedRequest.MinCreationTime)
		publicReportsQuery = publicReportsQuery.Where("timeCreated", ">=", parsedRequest.MinCreationTime)
		l.Infof("minCreationTime: %#v ", parsedRequest.MinCreationTime)
	}

	if parsedRequest.MaxCreationTime != nil {
		sharedReportsQuery = sharedReportsQuery.Where("timeCreated", "<=", parsedRequest.MaxCreationTime)
		presetReportsQuery = presetReportsQuery.Where("timeCreated", "<=", parsedRequest.MaxCreationTime)
		publicReportsQuery = publicReportsQuery.Where("timeCreated", "<=", parsedRequest.MaxCreationTime)
		l.Infof("maxCreationTime: %#v ", parsedRequest.MaxCreationTime)
	}

	var docSnaps, sharedReportsSnaps, presetReportsSnaps, publicReportsSnaps []*firestore.DocumentSnapshot

	accessDeniedCustomReportErr, err := s.reportTierService.CheckAccessToCustomReport(
		ctx,
		customerID,
	)
	if err != nil {
		if respErr := ctx.AbortWithError(http.StatusInternalServerError, err); respErr != nil {
			l.Error(respErr)
		}

		return
	}

	accessDeniedPresetReportErr, err := s.reportTierService.CheckAccessToPresetReport(
		ctx,
		customerID,
	)
	if err != nil {
		if respErr := ctx.AbortWithError(http.StatusInternalServerError, err); respErr != nil {
			l.Error(respErr)
		}

		return
	}

	if typeFilter == domainReport.ReportTypeCustom && accessDeniedCustomReportErr != nil {
		if err := web.Respond(ctx, accessDeniedCustomReportErr.PublicError(), http.StatusForbidden); err != nil {
			l.Error(err)
		}

		return
	}

	if (typeFilter == domainReport.ReportTypeCustom || typeFilter == "") && accessDeniedCustomReportErr == nil {
		// get the user reports
		sharedReportsSnaps, err = sharedReportsQuery.Documents(ctx).GetAll()
		if err != nil {
			AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)
			return
		}

		docSnaps = append(docSnaps, sharedReportsSnaps...)

		// get the public reports
		publicReportsSnaps, err = publicReportsQuery.Documents(ctx).GetAll()
		if err != nil {
			AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)
			return
		}

		docSnaps = append(docSnaps, publicReportsSnaps...)
	}

	// get the preset reports
	if typeFilter == domainReport.ReportTypePreset || typeFilter == "" {
		presetReportsSnaps, err = presetReportsQuery.Documents(ctx).GetAll()
		if err != nil {
			AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)
			return
		}

		docSnaps = append(docSnaps, presetReportsSnaps...)
	}

	// construct the response
	reportsListFromSnaps := make([]domainAPI.ReportListItem, 0)

	customerEntitlements, err := s.reportTierService.GetCustomerEntitlementIDs(ctx, customerID)
	if err != nil {
		l.Error(err)
		AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)

		return
	}

	for _, docSnap := range docSnaps {
		var tempReport domainReport.Report

		if err := docSnap.DataTo(&tempReport); err != nil {
			l.Error(err)
			AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)

			return
		}

		if tempReport.Type == domainReport.ReportTypePreset {
			if len(tempReport.Entitlements) > 0 {
				if !slice.ContainsAny(tempReport.Entitlements, customerEntitlements) {
					continue
				}
			} else if accessDeniedPresetReportErr != nil {
				continue
			}
		}

		report := domainAPI.ReportListItem{
			ID:           docSnap.Ref.ID,
			ReportName:   tempReport.Name,
			Type:         tempReport.Type,
			TimeCreated:  tempReport.TimeCreated.UnixMilli(),
			LastModified: tempReport.TimeModified.UnixMilli(),
		}
		// find Owner: loop over collaborator
		for _, collaborator := range tempReport.Collaborators {
			if collaborator.Role == collab.CollaboratorRoleOwner {
				report.Owner = collaborator.Email
				break
			}
		}
		// find URL
		report.URL = "https://" + common.Domain + "/customers/" + customerID + "/analytics/reports/" + docSnap.Ref.ID
		reportsListFromSnaps = append(reportsListFromSnaps, report)
	}

	filteredReportsList := make([]domainAPI.ReportListItem, 0)

	for _, report := range reportsListFromSnaps {
		// filter by owner
		if ownerFilter != "" && ownerFilter != report.Owner {
			continue
		}
		// filter by reportName
		if reportNameFilter != "" && report.ReportName != reportNameFilter {
			continue
		}

		// filter by updateTime
		if updateTimeFilter != 0 {
			if report.LastModified < updateTimeFilter {
				continue
			}
		}

		filteredReportsList = append(filteredReportsList, report)
	}

	// converting to generic sortable list for sorting and paging
	sortableReportList := make([]customerapi.SortableItem, len(filteredReportsList))
	for i, report := range filteredReportsList {
		sortableReportList[i] = report
	}

	sortedReportList, err := customerapi.SortAPIList(sortableReportList, parsedRequest.SortBy, parsedRequest.SortOrder)
	if err != nil {
		l.Error(err)
		AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)

		return
	}

	// paging
	page, token, err := customerapi.GetEncodedAPIPage(parsedRequest.MaxResults, parsedRequest.NextPageToken, sortedReportList)
	if err != nil {
		l.Error(err)
		AbortMsg(ctx, http.StatusBadRequest, err, ErrorBadRequest)

		return
	}

	ctx.JSON(http.StatusOK, domainAPI.ReportsList{
		Reports:   page,
		PageToken: token,
		RowCount:  len(page),
	})
}

func (s *ReportAPIService) RunReport(ctx *gin.Context, conn *connection.Connection) {
	email := ctx.GetString("email")

	l := logger.FromContext(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail: email,
	})

	// Set origin
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginReportsAPI)

	// Verify customerID path param
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	// get the reportID
	reportID := ctx.Param("id")
	if reportID == "" {
		l.Error("missing report id")
		ctx.AbortWithStatus(http.StatusBadRequest)

		return
	}

	fs := conn.Firestore(ctx)

	// get the report details
	docSnap, err := fs.Collection("dashboards").Doc("google-cloud-reports").Collection("savedReports").Doc(reportID).Get(ctx)
	if err != nil {
		l.Error(err)
		AbortMsg(ctx, 404, err, ErrorNotFound)

		return
	}

	var resp domainAPI.Report

	var tempReport domainReport.Report

	if err := docSnap.DataTo(&tempReport); err != nil {
		l.Error(err)
		AbortMsg(ctx, 500, err, ErrorInternalError)

		return
	}

	accessDeniedErr, err := s.reportTierService.CheckAccessToReport(
		ctx,
		customerID,
		&tempReport,
	)
	if err != nil {
		if respErr := ctx.AbortWithError(http.StatusInternalServerError, err); respErr != nil {
			l.Error(respErr)
		}

		return
	}

	if accessDeniedErr != nil {
		if respErr := web.Respond(ctx, accessDeniedErr.PublicError(), http.StatusForbidden); respErr != nil {
			l.Error(respErr)
		}

		return
	}

	resp.ID = docSnap.Ref.ID
	resp.ReportName = tempReport.Name
	resp.Type = tempReport.Type
	resp.TimeCreated = tempReport.TimeCreated.UnixMilli()
	resp.LastModified = tempReport.TimeModified.UnixMilli()

	isDoitEmployee := ctx.GetBool(common.CtxKeys.DoitEmployee)

	hasPermission := false

	if isDoitEmployee {
		hasPermission = true
	}

	// find Owner: loop over collaborator
	if tempReport.Public != nil {
		hasPermission = true
	}

	for _, collaborator := range tempReport.Collaborators {
		if collaborator.Email == email {
			hasPermission = true
		}

		if collaborator.Role == collab.CollaboratorRoleOwner {
			resp.Owner = collaborator.Email
		}
	}

	if !hasPermission {
		l.Error(err)
		AbortMsg(ctx, 401, err, ErrorUnAuthorized)

		return
	}

	// find URL
	resp.URL = "https://" + common.Domain + "/customers/" + customerID + "/analytics/reports/" + docSnap.Ref.ID

	queryRequest, report, err := s.cloudAnalytics.GetQueryRequest(ctx, customerID, reportID)
	if err != nil {
		if respErr := ctx.AbortWithError(http.StatusBadRequest, err); respErr != nil {
			l.Error(respErr)
		}

		return
	}

	if err := handleCustomTimeRange(ctx, queryRequest); err != nil {
		if respErr := ctx.AbortWithError(http.StatusBadRequest, err); respErr != nil {
			l.Error(respErr)
		}

		return
	}

	result, err := s.cloudAnalytics.GetQueryResult(ctx, queryRequest, customerID, email)
	if err != nil {
		if respErr := ctx.AbortWithError(http.StatusInternalServerError, err); respErr != nil {
			l.Error(respErr)
		}

		return
	}

	if queryRequest.Type == "report" && report.Type == domainReport.ReportTypeCustom {
		if err := s.reportStatsService.UpdateReportStats(
			ctx,
			queryRequest.ID,
			queryRequest.Origin,
			result.Details,
		); err != nil {
			l.Errorf("failed to update report stats for report %s with error %s", queryRequest.ID, err)
		}

		err = s.reportDAL.UpdateTimeLastRun(ctx, reportID, domainOrigin.QueryOriginReportsAPI)
		if err != nil {
			l.Errorf("failed to update last time run for report %v; %s", reportID, err)
		}
	}

	externalAPIService := externalAPIService.NewExternalAPIService()
	resp.Result = externalAPIService.ProcessResult(queryRequest, report, result)
	ctx.JSON(http.StatusOK, resp)
}

func handleCustomTimeRange(ctx *gin.Context, qr *cloudanalytics.QueryRequest) error {
	timeRange := ctx.Query("timeRange")
	startDate := ctx.Query("startDate")
	endDate := ctx.Query("endDate")

	if timeRange == "" && startDate == "" && endDate == "" {
		return nil
	}

	if timeRange != "" && (startDate != "" || endDate != "") {
		return errors.New("custom time range can be se by either timeRange or start and end date, not both")
	}

	if timeRange == "" && (startDate == "" || endDate == "") {
		return errors.New("custom time by date range must include both start and end date")
	}

	if timeRange != "" {
		d, err := duration.Parse(timeRange)
		if err != nil {
			return err
		}

		dur := d.ToTimeDuration()
		to := time.Now()
		from := to.Add(-dur)

		qr.TimeSettings.From = &from
		qr.TimeSettings.To = &to
	}

	if startDate != "" && endDate != "" {
		format := "2006-01-02"

		from, err := time.Parse(format, startDate)
		if err != nil {
			return err
		}

		to, err := time.Parse(format, endDate)
		if err != nil {
			return err
		}

		qr.TimeSettings.From = &from
		qr.TimeSettings.To = &to
	}

	return nil
}

func checkPresetAccess(entitlements []string, customerEntitlementsMap map[string]bool) bool {
	if len(entitlements) == 0 {
		return false
	}

	for _, entitlement := range entitlements {
		if _, ok := customerEntitlementsMap[entitlement]; ok {
			return true
		}
	}

	return false
}
