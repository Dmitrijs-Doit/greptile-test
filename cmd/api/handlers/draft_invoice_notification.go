package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	ncDomain "github.com/doitintl/notificationcenter/domain"
	nc "github.com/doitintl/notificationcenter/pkg"
	ncService "github.com/doitintl/notificationcenter/service"
)

type DraftInvoicesNotificationRequest struct {
	EntityID   string `json:"entityId"`
	CustomerID string `json:"customerId"`
	YearMonth  string `json:"yearMonth"`
}

var (
	ErrCanRetry = errors.New("can retry")
)

type DraftInvoicesNotification struct {
	loggerProvider logger.Provider
	fsProvider     func(ctx context.Context) *firestore.Client
}

func NewDraftInvoicesNotification(loggerProvider logger.Provider, conn *connection.Connection) *DraftInvoicesNotification {
	return &DraftInvoicesNotification{
		loggerProvider: loggerProvider,
		fsProvider:     conn.Firestore,
	}
}

// DraftInvoicesCreatedNotification sends a notification when draft invoices are first created for a given entity and year_month.
func (d *DraftInvoicesNotification) DraftInvoicesCreatedNotification(ctx *gin.Context) error {
	l := d.loggerProvider(ctx)

	var req DraftInvoicesNotificationRequest
	if err := ctx.BindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := SendDraftInvoicesCreatedNotification(ctx, req, d.fsProvider(ctx), l); err != nil {
		l.Errorf("failed to send draft invoices notification: %v", err)

		if errors.Is(err, ErrCanRetry) {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func SendDraftInvoicesCreatedNotification(ctx context.Context, req DraftInvoicesNotificationRequest, fs *firestore.Client, l logger.ILogger) error {
	if !isValidYearMonth(req.YearMonth) {
		return fmt.Errorf("year_month format should be YYYY-MM")
	}

	l.Debugf("SendDraftInvoicesCreatedNotification: request received: %v", req)

	entityRef := fs.Collection("entities").Doc(req.EntityID)
	customerRef := fs.Collection("customers").Doc(req.CustomerID)

	rs := ncService.NewRecipientsService(fs)
	configs, err := rs.GetNotificationRecipientsForCustomer(
		ctx,
		customerRef,
		ncDomain.NotificationNewProformaInvoices,
		ncDomain.WithInvoiceFilter(entityRef, true),
	)

	if err != nil {
		return fmt.Errorf("%w: %v", ErrCanRetry, err)
	}

	if len(configs) <= 0 {
		l.Debug("no recipients found")
		return nil
	}

	var entity common.Entity

	entitySnap, err := entityRef.Get(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCanRetry, err)
	}

	if err := entitySnap.DataTo(&entity); err != nil {
		return err
	}

	month := strings.Split(req.YearMonth, "-")[1]

	monthInt, err := strconv.Atoi(month)
	if err != nil {
		return err
	}

	monthName := time.Month(monthInt).String()

	notification := rs.GetCustomersNotifications(configs)[customerRef.ID]
	notification.Mock = !common.Production
	notification.Template = "AA2TGWMJXPMEN8QB5150CTA47TC6"
	notification.Data = map[string]interface{}{
		"month":  monthName,
		"url":    customerURL(req.CustomerID),
		"entity": entity.Name,
	}

	c, err := nc.NewClient(ctx, common.ProjectID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCanRetry, err)
	}

	res, err := c.Send(ctx, *notification)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCanRetry, err)
	}

	l.Infof("notification sent for request: %v, response: %v", req, res)

	return nil
}

func isValidYearMonth(input string) bool {
	pattern := `^\d{4}-(0[1-9]|1[0-2])$`
	regex := regexp.MustCompile(pattern)

	return regex.MatchString(input)
}

func customerURL(customerID string) string {
	var sub string
	if common.Production {
		sub = "console"
	} else {
		sub = "dev-app"
	}

	return fmt.Sprintf("https://%s.doit.com/customers/%s", sub, customerID)
}
