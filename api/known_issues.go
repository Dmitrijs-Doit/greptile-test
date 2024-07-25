package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/knownissues"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// Known Issues OutputMap
type KnownIssuesOutputMap struct {
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// Known issues rows count
	RowCount int `json:"rowCount"`
	// Array of known issues
	KnownIssues []KnownIssueListItem `json:"knownIssues"`
}

// Array of known issues
type KnownIssues []KnownIssue

// Cloud Incidents OutputMap
type CloudIncidentsOutputMap struct {
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// Known issues rows count
	RowCount int `json:"rowCount"`
	// Array of known issues
	KnownIssues []KnownIssueListItem `json:"incidents"`
}

type KnownIssueListItem struct {
	// known issue id, uniquely identifying the known issue
	Id string `json:"id"`
	// The time when this known issue was created, in milliseconds since the epoch.
	Date int64 `json:"createTime"`
	// The cloud Platform
	// enum: google-cloud,amazon-web-services,microsoft-azure
	Platform string `json:"platform"`
	// The name of the product affected by the known issue
	AffectedProduct string `json:"product"`
	// Known issue name as provided by cloud platform vendor
	Title string `json:"title"`
	// The Status of the issue
	// enum: active,archived
	Status string `json:"status"`
}

// Known Issue as reported by the cloud providers such as Google Cloud or Amazon Web Services
type KnownIssue struct {
	// known issue id, uniquely identifying the known issue
	Id string `json:"id"`
	// The time when this known issue was created, in milliseconds since the epoch.
	Date int64 `json:"createTime"`
	// The cloud Platform
	// enum: google-cloud,amazon-web-services,microsoft-azure
	Platform string `json:"platform"`
	// The name of the product affected by the known issue
	AffectedProduct string `json:"product"`
	// Known issue name as provided by cloud platform vendor
	Title string `json:"title"`
	// The Status of the issue
	// enum: active,archived
	Status string `json:"status"`
	// known issue description in a summarised form
	Summary string `json:"summary"`
	// Detailed explanation on the known issue.
	Description string `json:"description"`
	// Known issue symptoms, if available
	Symptoms string `json:"symptoms"`
	// Mitigation workaround for the known issue, if available.
	Workaround string `json:"workaround"`
}

type KnownIssuesFilter struct {
	// Filter by cloud platform
	// enum: all,amazon-web-services,google-cloud
	// default: all
	Platform string `json:"platform"`
	// Filter by product
	Product string `json:"product"`
	// Filter by status
	// enum:  all,ongoing,archived
	// default: all
	Status string `json:"status"`
}

// swagger:parameters idOfKnownIssues
type KnownIssuesRequest struct {
	// The maximum number of results to return in a single page. Leverage the page tokens to iterate through the entire collection.
	// default: 50
	MaxResults int `json:"maxResults"`
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// An expression for filtering the results of the request. The syntax is "key:[<value>]". Multiple filters can be connected using a pipe |. Note that using different keys in the same filter results in “AND,” while using the same key multiple times in the same filter results in “OR”.
	Filter string `json:"filter"`
	// Min value for the known issue creation time, in milliseconds since the POSIX epoch. If set, only known issues created after or at this timestamp are returned.
	MinCreationTime string `json:"minCreationTime"`
	// Max value for the known issue creation time, in milliseconds since the POSIX epoch. If set, only known issues created before or at this timestamp are returned.
	MaxCreationTime string `json:"maxCreationTime"`
}

// swagger:parameters idOfKnownIssue
type KnownIssueRequest struct {
	// known issue id, uniquely identifying the known issue
	// in:path
	ID string `json:"id"`
}

func ListKnownIssues(ctx *gin.Context, conn *connection.Connection) {
	email := ctx.GetString("email")

	l := logger.FromContext(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail: email,
	})

	maxResults := 50

	maxResultsStr := ctx.Request.URL.Query().Get("maxResults")
	if maxResultsStr != "" {
		var err error

		maxResults, err = strconv.Atoi(maxResultsStr)
		if err != nil {
			errMsg := ErrorParam + "maxResults"
			AbortMsg(ctx, http.StatusBadRequest, errors.New(errMsg), errMsg)

			return
		}

		if maxResults > maxResultsLimit {
			l.Info(ErrorParamMaxResultRange)
			AbortMsg(ctx, http.StatusBadRequest, errMaxResultRange, errMaxResultRange.Error())

			return
		}

		if maxResults <= 0 {
			maxResults = maxResultsLimit
		}
	}

	minCreationTime := ctx.Request.URL.Query().Get("minCreationTime")
	maxCreationTime := ctx.Request.URL.Query().Get("maxCreationTime")
	encodedPageToken := ctx.Request.URL.Query().Get("pageToken")

	decodedPageToken, err := customerapi.DecodePageToken(encodedPageToken)
	if err != nil {
		AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)
		return
	}

	pageToken := decodedPageToken
	filterStr := ctx.Request.URL.Query().Get("filter")

	fs := conn.Firestore(ctx)

	var knownIssuesOutputMap KnownIssuesOutputMap
	knownIssuesOutputMap.KnownIssues = make([]KnownIssueListItem, 0)
	knownIssuesRef := fs.Collection("knownIssues")
	knownIssuesQuery := knownIssuesRef.Select("issueId", "dateTime", "affectedProduct", "platform", "title", "symptoms", "summary", "status", "products").OrderBy("dateTime", firestore.Desc).Limit(maxResults)

	if filterStr != "" {
		// parse the filter string and split into array. loop over each key:
		filterArr := strings.Split(filterStr, "|")

		var productArr []string

		var platformArr []string

		var statusArr []string

		for _, param := range filterArr {
			splitParam := strings.Split(param, ":")
			if len(splitParam) == 2 {
				key := splitParam[0]
				value := splitParam[1]

				switch key {
				case "product":
					productArr = append(productArr, value)
				case "platform":
					if value == "google-cloud" {
						value = "google-cloud-project"
					}

					platformArr = append(platformArr, value)
				case "status":
					if value == "active" {
						value = "ongoing"
					}

					statusArr = append(statusArr, value)
				default:
					l.Info(ErrorUnknownFilterKey + param)
					AbortMsg(ctx, http.StatusBadRequest, errors.New(ErrorUnknownFilterKey+param), ErrorUnknownFilterKey+param)

					return
				}
			} else {
				l.Info(ErrorUnknownFilterKey + param)
				AbortMsg(ctx, http.StatusBadRequest, errors.New(ErrorUnknownFilterKey+param), ErrorUnknownFilterKey+param)

				return
			}
		}

		if len(productArr) > 0 {
			if len(productArr) > 1 {
				knownIssuesQuery = knownIssuesQuery.Where("affectedProduct", "in", productArr)
			} else {
				knownIssuesQuery = knownIssuesQuery.Where("affectedProduct", "==", productArr[0])
			}
		}

		if len(platformArr) > 0 {
			if len(platformArr) > 1 {
				knownIssuesQuery = knownIssuesQuery.Where("platform", "in", platformArr)
			} else {
				knownIssuesQuery = knownIssuesQuery.Where("platform", "==", platformArr[0])
			}
		}

		if len(statusArr) > 0 {
			if len(statusArr) > 1 {
				knownIssuesQuery = knownIssuesQuery.Where("status", "in", statusArr)
			} else {
				knownIssuesQuery = knownIssuesQuery.Where("status", "==", statusArr[0])
			}
		}
	}

	if minCreationTime != "" {
		t, err := common.MsToTime(minCreationTime)
		if err != nil {
			l.Error(err)
			AbortMsg(ctx, 400, err, ErrorParam+"minCreationTime")

			return
		}

		knownIssuesQuery = knownIssuesQuery.Where("dateTime", ">=", t)
		l.Infof("minCreationTime: %#v", t)
	}

	if maxCreationTime != "" {
		t, err := common.MsToTime(maxCreationTime)
		if err != nil {
			l.Error(err)
			AbortMsg(ctx, 400, err, ErrorParam+"maxCreationTime")

			return
		}

		knownIssuesQuery = knownIssuesQuery.Where("dateTime", "<=", t)
		l.Infof("maxCreationTime: %#v", t)
	}
	//paging
	if pageToken != "" {
		docSnap, err := fs.Collection("knownIssues").Doc(pageToken).Get(ctx)
		knownIssuesQuery = knownIssuesQuery.StartAfter(docSnap)

		if err != nil {
			AbortMsg(ctx, 404, err, ErrorPageTokenNotFound)
			return
		}
	}

	docSnaps, err := knownIssuesQuery.Documents(ctx).GetAll()
	if err != nil {
		if strings.Contains(err.Error(), "InvalidArgument") && strings.Contains(err.Error(), "Only a single") {
			AbortMsg(ctx, 400, err, ErrorFilterComplex)
			return
		}

		AbortMsg(ctx, 400, err, ErrorInternalError)

		return
	}

	for _, docSnap := range docSnaps {
		knownIssuelistItem := KnownIssueListItem{}
		knownIssuelistItem.Id = docSnap.Ref.ID

		if docSnap.Data()["platform"].(string) == "google-cloud-project" {
			var gcpKnownIssue knownissues.GCPKnownIssue
			if err := docSnap.DataTo(&gcpKnownIssue); err != nil {
				l.Error(err)
				AbortMsg(ctx, 500, err, ErrorInternalError)

				return
			}

			if len(gcpKnownIssue.Products) > 0 {
				knownIssuelistItem = KnownIssueListItem{
					Id:              docSnap.Ref.ID,
					Platform:        common.Assets.GoogleCloud,
					Date:            gcpKnownIssue.DateTime.UnixMilli(),
					AffectedProduct: strings.Join(gcpKnownIssue.Products, ","),
					Title:           gcpKnownIssue.Title,
					Status:          gcpKnownIssue.Status,
				}
			} else {
				knownIssuelistItem = KnownIssueListItem{
					Id:              docSnap.Ref.ID,
					Platform:        common.Assets.GoogleCloud,
					Date:            gcpKnownIssue.DateTime.UnixMilli(),
					AffectedProduct: gcpKnownIssue.Product,
					Title:           gcpKnownIssue.Title,
					Status:          gcpKnownIssue.Status,
				}
			}
		} else {
			var awsKnownIssue knownissues.AWSKnownIssue
			if err := docSnap.DataTo(&awsKnownIssue); err != nil {
				l.Error(err)
				AbortMsg(ctx, 500, err, ErrorInternalError)

				return
			}

			knownIssuelistItem = KnownIssueListItem{
				Id:              docSnap.Ref.ID,
				Platform:        awsKnownIssue.Platform,
				Date:            awsKnownIssue.DateTime.UnixMilli(),
				AffectedProduct: awsKnownIssue.Product,
				Title:           awsKnownIssue.Title,
				Status:          awsKnownIssue.Status,
			}
		}

		if knownIssuelistItem.Status == "ongoing" {
			knownIssuelistItem.Status = "active"
		}

		knownIssuelistItem.Id = docSnap.Ref.ID
		knownIssuesOutputMap.KnownIssues = append(knownIssuesOutputMap.KnownIssues, knownIssuelistItem)
	}

	if len(docSnaps) == maxResults {
		knownIssuesOutputMap.PageToken = customerapi.EncodePageToken(docSnaps[len(docSnaps)-1].Ref.ID)
	} else {
		knownIssuesOutputMap.PageToken = ""
	}

	knownIssuesOutputMap.RowCount = len(knownIssuesOutputMap.KnownIssues)

	// if request path is cloud incidents, then return cloud incidents
	if strings.Contains(ctx.Request.URL.Path, "cloudincidents") {
		ctx.JSON(http.StatusOK, CloudIncidentsOutputMap(knownIssuesOutputMap))
	} else {
		ctx.JSON(http.StatusOK, knownIssuesOutputMap)
	}
}

func GetKnownIssue(ctx *gin.Context, conn *connection.Connection) {
	email := ctx.GetString("email")

	l := logger.FromContext(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail: email,
	})

	// get the issue id
	knownIssueId := ctx.Param("id")

	if knownIssueId == "" {
		l.Error("missing issueId")
		ctx.AbortWithStatus(http.StatusBadRequest)

		return
	}

	fs := conn.Firestore(ctx)

	knownIssueSnap, err := fs.Collection("knownIssues").Doc(knownIssueId).Get(ctx)
	if err != nil {
		l.Error(err)
		AbortMsg(ctx, http.StatusNotFound, err, ErrorNotFound)

		return
	}

	var knownIssue KnownIssue

	if knownIssueSnap.Data()["platform"].(string) == "google-cloud-project" {
		var gcpKnownIssue knownissues.GCPKnownIssue
		if err := knownIssueSnap.DataTo(&gcpKnownIssue); err != nil {
			l.Error(err)
			AbortMsg(ctx, 500, err, ErrorInternalError)

			return
		}

		knownIssue = KnownIssue{
			Id:              knownIssueSnap.Ref.ID,
			Platform:        common.Assets.GoogleCloud,
			Date:            gcpKnownIssue.DateTime.UnixMilli(),
			AffectedProduct: gcpKnownIssue.Product,
			Title:           gcpKnownIssue.Title,
			Status:          gcpKnownIssue.Status,
			Summary:         gcpKnownIssue.Summary,
			Description:     gcpKnownIssue.OutageDescription,
			Symptoms:        gcpKnownIssue.Symptoms,
			Workaround:      gcpKnownIssue.Workaround,
		}
	} else {
		var awsKnownIssue knownissues.AWSKnownIssue
		if err := knownIssueSnap.DataTo(&awsKnownIssue); err != nil {
			l.Error(err)
			AbortMsg(ctx, 500, err, ErrorInternalError)

			return
		}

		knownIssue = KnownIssue{
			Id:              knownIssueSnap.Ref.ID,
			Platform:        awsKnownIssue.Platform,
			Date:            awsKnownIssue.DateTime.UnixMilli(),
			AffectedProduct: awsKnownIssue.Product,
			Title:           awsKnownIssue.Title,
			Status:          awsKnownIssue.Status,
			Description:     awsKnownIssue.OutageDescription,
		}
	}

	ctx.JSON(http.StatusOK, knownIssue)
}
