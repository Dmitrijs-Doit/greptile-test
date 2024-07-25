package api

import (
	"errors"
	"regexp"

	"github.com/gin-gonic/gin"
)

const (
	ErrorUnAuthorized                     = "Unauthorized"
	ErrorForbidden                        = "User does not have required permissions for this action"
	ErrorNotFound                         = "Not Found"
	ErrorBadRequest                       = "Bad Request"
	ErrorInternalError                    = "Internal Server Error"
	ErrorParam                            = "Error In Param: " // Param name is dynamic
	ErrorParamMaxResultRange              = "maxResults must be lower than 250"
	ErrorUnknownFilterKey                 = "Invalid filter key: " // filter key is dynamic
	ErrorPageTokenNotFound                = "Invalid pageToken"
	ErrorInvoiceNotFound                  = "Invoice not found"
	ErrorFilterComplex                    = "Filter has more than one key that repeats more than once"
	ErrorKeyExists                        = "accessKey is Already exist"
	ErrorUserNotFound                     = "User Not Found"
	ErrorEmailIsEmpty                     = "Email is empty"
	ErrorUIDIsEmpty                       = "UID is empty"
	ErrorBudgetInvalidAmount              = "invalid amount"
	ErrorBudgetInvalidGrowthPerPeriod     = "invalid GrowthPerPeriod"
	ErrorBudgetInvalidCurrency            = "invalid currency"
	ErrorBudgetInvalidType                = "invalid budget type"
	ErrorBudgetInvalidTimeInterval        = "invalid time interval"
	ErrorBudgetInvalidStartPeriod         = "invalid start period"
	ErrorBudgetInvalidEndPeriod           = "invalid end period"
	ErrorBudgetInvalidScope               = "invalid scope (attributions)"
	ErrorBudgetInvalidName                = "invalid budget name"
	ErrorBudgetNoUpdates                  = "no updates found"
	ErrorBudgetInvalidCollaborator        = "invalid collaborator"
	ErrorBudgetUserNotConnectedToCustomer = ": not consolidated to customer" // user is dynamic
	ErrorBudgetInvalidRole                = "invalid role"
	ErrorBudgetInvalidOwnerRole           = "budget owner role must be of type owner"
	ErrorBudgetMissingOwner               = "no owner in collaborator list"
	ErrorBudgetRecurringWithEndPeriod     = "recurring budget can not have end period"
	ErrorBudgetFixedWithoutEndPeriod      = "fixed budget must have end period"
	ErrorBudgetInvalidDataSource          = "invalid budget data source"
)

// Limit for API list query parameter: maxResults
const maxResultsLimit = 250

var (
	errMaxResultRange = errors.New(ErrorParamMaxResultRange)
)

type ErrorResponse struct {
	Code  int
	Error error
	Msg   string
}

func AbortMsg(ctx *gin.Context, code int, err error, msg string) {
	ctx.JSON(code, gin.H{
		"error": msg,
	})

	// A custom error page with HTML templates can be shown by c.HTML()
	if err != nil {
		ctx.Error(err)
	}

	ctx.Abort()
}

func isEmailValid(e string) bool {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return emailRegex.MatchString(e)
}
