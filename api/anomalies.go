package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// swagger:parameters idOfAnomalies
type AnomalyParams struct {
	// Min value for the anomaly detection time
	MinCreationTime int64 `json:"minCreationTime"`
	// Max value for the anomaly detection time
	MaxCreationTime int64 `json:"maxCreationTime"`
	// An expression for filtering the results of the request
	Filter string `json:"filter"`
	// The maximum number of results to return in a single page
	MaxResults int `json:"maxResults"`
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken"`
}

const AnomalyChartKeyFormat = "2006-01-02 15:04:05 UTC"
const UsageStartTime = "metadata.usage_start_time"

type AnomalyItemDetails struct {
	AnomalyChartURL string `json:"anomalyChartUrl"`
	Anomaly
}

type AnomalyItem struct {
	ID string `json:"id"`
	Anomaly
}

type AnomalyChartDataPoint struct {
	High float64 `json:"high"`
	Low  float64 `json:"low"`

	// new fields
	SnapshotValue float64 `json:"snapshot_value"`
	UpdatedValue  float64 `json:"updated_value"`
	Status        string  `json:"status"`

	// deprecated fields
	SkuCosts   []interface{} `json:"sku_costs"`
	SkuNames   []interface{} `json:"sku_names"`
	ActualCost float64       `json:"actual_cost"`
}

type AnomalySKU struct {
	SKUName string  `json:"name"`
	SKUCost float64 `json:"cost"`
}

type AnomalySKUArray []AnomalySKU

type Anomaly struct {
	// Billing account ID
	// required: true
	BillingAccount string `json:"billingAccount"`
	// Attribution ID
	// required: true
	Attribution string `json:"attribution"`
	// Cost of the anomaly over and above the expected normal cost
	// required: true
	CostOfAnomaly float64 `json:"costOfAnomaly"`
	// Cloud Provider name
	// required: true
	CloudProvider string `json:"platform"`
	// Scope: Project or Account
	// required: true
	Scope string `json:"scope"`
	// Service name
	// required: true
	ServiceName string `json:"serviceName"`
	// Top 3 SKUs contributing to the anomaly
	// required: true
	TopSKUs AnomalySKUArray `json:"top3SKUs"`
	// Severity level: Information, Warning or Critical
	// required: true
	SeverityLevel string `json:"severityLevel"`
	// Timeframe: Daily or Hourly
	// required: true
	TimeFrame string `json:"timeFrame"`
	// Usage start time of the anomaly
	// required: true
	StartTime int64 `json:"startTime"`
}

// swagger:parameters idOfAnomaly
type AnomalyRequest struct {
	// anomaly id, uniquely identifying the anomaly
	// in:path
	ID string `json:"id"`
}

type AnomaliesResponse struct {
	PageToken string        `json:"pageToken,omitempty"`
	RowCount  int           `json:"rowCount"`
	Anomalies []AnomalyItem `json:"anomalies"`
}

type AnomaliesMetadata struct {
	AttributionID    string `json:"attribution"`
	AlertID          string `json:"alert_id"`
	AnomalyChartURL  string `json:"anomaly_chart_url"`
	BillingAccountID string `json:"billing_account_id"`
	Platform         string `json:"platform"`
	ProjectID        string `json:"project_id"`
	ServiceName      string `json:"service_name"`

	// new fields
	SkuName   string    `json:"sku_name"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Unit      string    `json:"unit"`
	Frequency string    `json:"frequency"`
	Value     float64   `json:"value"`
	Excess    float64   `json:"excess"`
	Severity  int       `json:"severity"`

	// deprecated fields
	Context       string                 `json:"context"`
	SeverityLevel map[string]interface{} `json:"level"`
	TimeFrame     map[string]interface{} `json:"explorated_level"`
	StartTime     string                 `json:"usage_start_time"`
}

func (metadata *AnomaliesMetadata) getTimeFrame() string {
	if metadata.Frequency != "" {
		return metadata.Frequency
	}

	timeFrameMap := map[string]string{
		"RISING_DAILY_COSTS":       "daily",
		"MULTIPLE_ALERTS_IN_A_DAY": "hourly",
		"HOURLY_ALERT":             "hourly",
		"":                         "hourly",
	}

	return timeFrameMap[metadata.Context]
}

func (metadata *AnomaliesMetadata) getStartTime() (time.Time, error) {
	if metadata.Timestamp != (time.Time{}) {
		return metadata.Timestamp, nil
	}

	t, err := time.Parse(AnomalyChartKeyFormat, metadata.StartTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse date: %v", err)
	}

	return t, nil
}

func (metadata *AnomaliesMetadata) getSeverityLevel() string {
	if metadata.Severity > 0 {
		return severityMap[strconv.Itoa(metadata.Severity)]
	}

	return severityMap[metadata.SeverityLevel["rules_model"].(string)]
}

type AnomalyUpdate struct {
	Value    float64 `json:"value"`
	Excess   float64 `json:"excess"`
	Severity int     `json:"severity"`
	Status   string  `json:"status"`
}

type anomalyHelper struct {
	MetaData  AnomaliesMetadata                `json:"metadata"`
	ChartData map[string]AnomalyChartDataPoint `json:"chart_data"`
	Updates   map[string]AnomalyUpdate         `json:"updates"`
}

func ListAnomalies(ctx *gin.Context, conn *connection.Connection) {
	l := logger.FromContext(ctx)
	fs := conn.Firestore(ctx)

	params, hasErrors := getParams(ctx)
	if hasErrors {
		return
	}

	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	customerRef := fs.Collection("customers").Doc(customerID)

	query := fs.CollectionGroup("billingAnomalies").Where("customer", "==", customerRef)

	docSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)
		return
	}

	anomalies, shouldReturn := filterAnomalies(ctx, l, docSnaps, params)
	if shouldReturn {
		return
	}

	sort.Slice(anomalies, func(a, b int) bool {
		startTimeA, _ := anomalies[a].MetaData.getStartTime()
		startTimeB, _ := anomalies[b].MetaData.getStartTime()

		return startTimeA.After(startTimeB)
	})

	response, shouldReturn1 := getListAnomaliesResponse(ctx, l, fs, anomalies, params)
	if shouldReturn1 {
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func getListAnomaliesResponse(ctx *gin.Context, l logger.ILogger, fs *firestore.Client, anomalies []*anomalyHelper, params AnomalyParams) (AnomaliesResponse, bool) {
	var response AnomaliesResponse

	for _, anomalyHelper := range anomalies {
		if params.MaxResults > 0 && len(response.Anomalies) == params.MaxResults {
			break
		}

		if params.PageToken == anomalyHelper.MetaData.AlertID {
			params.PageToken = ""
			continue
		}

		if params.PageToken != "" {
			continue
		}

		anomaly := new(Anomaly)

		if err := mapFields(ctx, l, fs, anomaly, anomalyHelper); err != nil {
			l.Info(err)
			AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)

			return AnomaliesResponse{}, true
		}

		anomalyItem := AnomalyItem{
			ID:      anomalyHelper.MetaData.AlertID,
			Anomaly: *anomaly,
		}

		response.Anomalies = append(response.Anomalies, anomalyItem)
	}

	if params.MaxResults > 0 && len(response.Anomalies) == params.MaxResults {
		response.PageToken = customerapi.EncodePageToken(response.Anomalies[len(response.Anomalies)-1].ID)
	} else {
		response.PageToken = ""
	}

	response.RowCount = len(response.Anomalies)

	return response, false
}

func filterAnomalies(ctx *gin.Context, l logger.ILogger, docSnaps []*firestore.DocumentSnapshot, params AnomalyParams) ([]*anomalyHelper, bool) {
	var anomalies []*anomalyHelper

	if len(docSnaps) == 0 {
		return anomalies, false
	}

	for _, docSnap := range docSnaps {
		anomaly, err := getAnomalyHelperFromDocSnap(docSnap, l)
		if err != nil {
			AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)
			return nil, true
		}

		if ok, err := shouldReturnAnomaly(params, anomaly); ok {
			anomalies = append(anomalies, anomaly)
		} else {
			if err != nil {
				AbortMsg(ctx, http.StatusBadRequest, err, err.Error())
				return nil, true
			}
		}
	}

	return anomalies, false
}

func getParams(ctx *gin.Context) (AnomalyParams, bool) {
	var (
		err             error
		params          AnomalyParams
		maxResults      = 0
		minCreationTime int64
		maxCreationTime int64 = math.MaxInt64
	)

	maxResultsStr := ctx.Request.URL.Query().Get("maxResults")
	if maxResultsStr != "" {
		maxResults, err = strconv.Atoi(maxResultsStr)
		if err != nil {
			AbortMsg(ctx, http.StatusBadRequest, err, ErrorParam+"maxResults")
			return params, true
		}
	}

	minCreationTimeStr := ctx.Request.URL.Query().Get("minCreationTime")
	if minCreationTimeStr != "" {
		minCreationTime, err = strconv.ParseInt(minCreationTimeStr, 10, 64)
		if err != nil {
			AbortMsg(ctx, http.StatusBadRequest, err, ErrorParam+"minCreationTime")
			return params, true
		}
	}

	maxCreationTimeStr := ctx.Request.URL.Query().Get("maxCreationTime")
	if maxCreationTimeStr != "" {
		maxCreationTime, err = strconv.ParseInt(maxCreationTimeStr, 10, 64)
		if err != nil {
			AbortMsg(ctx, http.StatusBadRequest, err, ErrorParam+"maxCreationTime")
			return params, true
		}
	}

	params.MaxResults = maxResults
	params.MinCreationTime = minCreationTime
	params.MaxCreationTime = maxCreationTime
	params.Filter = ctx.Request.URL.Query().Get("filter")
	decodedPageToken, err := customerapi.DecodePageToken(ctx.Request.URL.Query().Get("pageToken"))

	if err != nil {
		AbortMsg(ctx, http.StatusBadRequest, err, err.Error())
		return params, true
	}

	params.PageToken = decodedPageToken

	return params, false
}

func getAnomalyHelperFromDocSnap(docSnap *firestore.DocumentSnapshot, l logger.ILogger) (*anomalyHelper, error) {
	data := docSnap.Data()

	combinedData := struct {
		Metadata  map[string]interface{} `json:"metadata"`
		ChartData map[string]interface{} `json:"chart_data"`
	}{
		Metadata:  data["metadata"].(map[string]interface{}),
		ChartData: data["chart_data"].(map[string]interface{}),
	}
	anomaly := new(anomalyHelper)

	if data != nil {
		if num, ok := combinedData.Metadata["billing_account_id"].(int64); ok {
			combinedData.Metadata["billing_account_id"] = strconv.FormatInt(num, 10)
		}

		jsonData, err := json.Marshal(combinedData)
		if err != nil {
			l.Info(err)
			return anomaly, err
		}

		if err := json.Unmarshal(jsonData, anomaly); err != nil {
			l.Info(err)
			return anomaly, err
		}
	}

	return anomaly, nil
}

func shouldReturnAnomaly(params AnomalyParams, anomaly *anomalyHelper) (bool, error) {
	startTime, err := anomaly.MetaData.getStartTime()
	if err != nil {
		return false, err
	}

	startTimeInMillis := common.ToUnixMillis(startTime)

	if params.MinCreationTime > startTimeInMillis {
		return false, nil
	}

	if params.MaxCreationTime < startTimeInMillis {
		return false, nil
	}

	if params.Filter != "" {
		dict := strings.Split(params.Filter, "|")
		if len(dict) > 0 {
			for _, value := range dict {
				keyVale := strings.Split(value, ":")

				if len(keyVale) > 0 {
					if keyVale[0] == "serviceName" && anomaly.MetaData.ServiceName != keyVale[1] {
						return false, nil
					} else if keyVale[0] == "billingAccount" && anomaly.MetaData.BillingAccountID != keyVale[1] {
						return false, nil
					} else if keyVale[0] == "severityLevel" && anomaly.MetaData.getSeverityLevel() != keyVale[1] {
						return false, nil
					} else if keyVale[0] == "platform" && anomaly.MetaData.Platform != keyVale[1] {
						return false, nil
					} else if keyVale[0] != "serviceName" && keyVale[0] != "billingAccount" && keyVale[0] != "severityLevel" && keyVale[0] != "platform" {
						return false, errors.New(ErrorUnknownFilterKey + keyVale[0])
					}
				}
			}
		}
	}

	return true, nil
}

func prepareAnomaly(ctx *gin.Context, fs *firestore.Client, docSnap *firestore.DocumentSnapshot, l logger.ILogger) (*Anomaly, error) {
	data := docSnap.Data()

	combinedData := struct {
		Metadata  map[string]interface{} `json:"metadata"`
		ChartData map[string]interface{} `json:"chart_data"`
	}{
		Metadata:  data["metadata"].(map[string]interface{}),
		ChartData: data["chart_data"].(map[string]interface{}),
	}

	anomalyHelper := new(anomalyHelper)
	anomaly := new(Anomaly)

	if data != nil {
		jsonData, err := json.Marshal(combinedData)
		if err != nil {
			l.Info(err)
			return anomaly, err
		}

		if err := json.Unmarshal(jsonData, anomalyHelper); err != nil {
			l.Info(err)
			return anomaly, err
		}

		if err := mapFields(ctx, l, fs, anomaly, anomalyHelper); err != nil {
			l.Info(err)
			return anomaly, err
		}
	}

	return anomaly, nil
}

var severityMap = map[string]string{
	"1": "information",
	"2": "warning",
	"3": "critical",
}

func getLastChartDataPoint(l logger.ILogger, anoHelp *anomalyHelper) AnomalyChartDataPoint {
	var timestamps []time.Time

	var chartDataArray = anoHelp.ChartData

	for timestampStr := range chartDataArray {
		timestamp, err := time.Parse(AnomalyChartKeyFormat, timestampStr)
		if err != nil {
			l.Errorf("Error parsing timestamp %s: %s", timestampStr, err)
			return AnomalyChartDataPoint{}
		}

		timestamps = append(timestamps, timestamp)
	}

	// Sort data points by timestamp
	sort.SliceStable(timestamps, func(i, j int) bool {
		return timestamps[i].Before(timestamps[j])
	})

	var lastDataPoint AnomalyChartDataPoint

	// Calculate cost of anomaly using the last data point in the chartData array, and subtracting
	// the high estimate from the actual_cost
	for _, timestamp := range timestamps {
		lastDataPoint = chartDataArray[timestamp.Format(AnomalyChartKeyFormat)]
	}

	return lastDataPoint
}

func calculateCostOfAnomalyFromChartData(l logger.ILogger, anoHelp *anomalyHelper) float64 {
	if anoHelp.MetaData.Excess > 0 {
		return anoHelp.MetaData.Excess
	}

	lastDataPoint := getLastChartDataPoint(l, anoHelp)

	return lastDataPoint.ActualCost - lastDataPoint.High
}

func getTopSKUs(l logger.ILogger, anoHelp *anomalyHelper) AnomalySKUArray {
	var skuArray AnomalySKUArray

	if anoHelp.MetaData.SkuName != "" {
		skuArray = append(skuArray, AnomalySKU{
			SKUName: anoHelp.MetaData.SkuName,
			SKUCost: anoHelp.MetaData.Excess,
		})

		return skuArray
	}

	var lastDataPoint = getLastChartDataPoint(l, anoHelp)

	skuNames := lastDataPoint.SkuNames
	skuCosts := lastDataPoint.SkuCosts

	for i := 0; i < len(skuNames); i++ {
		skuName := skuNames[i].(string)
		skuCost := skuCosts[i].(float64)

		skuArray = append(skuArray, AnomalySKU{
			SKUName: skuName,
			SKUCost: skuCost,
		})
	}

	return skuArray
}

func mapFields(ctx *gin.Context, l logger.ILogger, fs *firestore.Client, anomaly *Anomaly, anoHelp *anomalyHelper) error {
	anomaly.TopSKUs = getTopSKUs(l, anoHelp)

	anomaly.Attribution, _ = getAttributionByID(ctx, fs, anoHelp.MetaData.AttributionID)
	anomaly.CostOfAnomaly = calculateCostOfAnomalyFromChartData(l, anoHelp)
	anomaly.BillingAccount = anoHelp.MetaData.BillingAccountID
	anomaly.CloudProvider = anoHelp.MetaData.Platform
	anomaly.Scope = anoHelp.MetaData.ProjectID
	anomaly.ServiceName = anoHelp.MetaData.ServiceName
	anomaly.SeverityLevel = anoHelp.MetaData.getSeverityLevel()

	anomaly.TimeFrame = anoHelp.MetaData.getTimeFrame()

	t, err := anoHelp.MetaData.getStartTime()
	if err != nil {
		return err
	}

	anomaly.StartTime = common.ToUnixMillis(t)

	return nil
}

func getAttributionByID(ctx context.Context, fs *firestore.Client, attributionID string) (string, error) {
	// Call firestore with the attribution document reference and get back the name of the attribution
	query := fs.Collection("dashboards").Doc("google-cloud-reports").Collection("attributions").Doc(attributionID)

	// Get a single document from Firestore using the query.
	docSnap, err := query.Get(ctx)
	if err != nil {
		return "", err
	}

	data := docSnap.Data()

	return data["name"].(string), nil
}

func GetAnomaly(ctx *gin.Context, conn *connection.Connection) {
	l := logger.FromContext(ctx)
	fs := conn.Firestore(ctx)

	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	customerRef := fs.Collection("customers").Doc(customerID)

	alertID := ctx.Param("id")

	docSnaps, err := fs.CollectionGroup("billingAnomalies").Where("customer", "==", customerRef).Where("metadata.alert_id", "==", alertID).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)
		return
	}

	if len(docSnaps) == 1 {
		anomaly, err := prepareAnomaly(ctx, fs, docSnaps[0], l)
		if err != nil {
			AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)
			return
		}

		url := "https://storage.googleapis.com/" + common.ProjectID + "-gcp-anomalies/" + alertID + ".png"
		anomalyItemDetails := AnomalyItemDetails{AnomalyChartURL: url, Anomaly: *anomaly}
		ctx.JSON(http.StatusOK, anomalyItemDetails)
	} else {
		AbortMsg(ctx, http.StatusNotFound, nil, ErrorNotFound)
	}
}
