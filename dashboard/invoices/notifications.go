package invoices

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/doitintl/hello/scheduled-tasks/common"
	csm_service "github.com/doitintl/hello/scheduled-tasks/csmengagement/service"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	rsDomain "github.com/doitintl/notificationcenter/domain"
	notificationcenter "github.com/doitintl/notificationcenter/pkg"
	recipientsService "github.com/doitintl/notificationcenter/service"
)

type NotificationTask struct {
	CustomerID string `json:"customer_id"`
	EntityID   string `json:"entity_id"`
}

type InvoiceNotificationData struct {
	ID       string `json:"id"`
	Products string `json:"products"`
	PayDate  string `json:"payDate"`
	Amount   string `json:"amount"`
	LinkURL  string `json:"url"`
}

func NotificationsHandler(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	// TODO: check if there are any new invoices?
	invoiceDocSnaps, err := fs.Collection("invoices").
		Where("CANCELED", "==", false).
		Where("notification.sent", "==", false).
		Limit(1).Documents(ctx).GetAll()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if len(invoiceDocSnaps) <= 0 {
		l.Info("no new invoices were found")
		ctx.Status(http.StatusOK)

		return
	}

	docSnaps, err := fs.Collection("entities").
		Where("active", "==", true).
		Documents(ctx).
		GetAll()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	for _, docSnap := range docSnaps {
		var entity common.Entity
		if err := docSnap.DataTo(&entity); err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		t := NotificationTask{
			CustomerID: entity.Customer.ID,
			EntityID:   docSnap.Ref.ID,
			// Contact:    entity.Contact,
		}

		taskBody, err := json.Marshal(t)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		config := common.CloudTaskConfig{
			Method:       cloudtaskspb.HttpMethod_POST,
			Path:         "/tasks/invoices/notifications",
			Queue:        common.TaskQueueSendgrid,
			Body:         taskBody,
			ScheduleTime: nil,
		}

		_, err = common.CreateCloudTask(ctx, &config)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
}

// func NotificationsWorker(ctx *gin.Context, t NotificationTask) {
func NotificationsWorker(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	var t NotificationTask
	if err := ctx.ShouldBindJSON(&t); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	l.Info(t)

	fs := common.GetFirestoreClient(ctx)

	minCreateTime := time.Now().UTC().Truncate(24 * time.Hour).Add(-72 * time.Hour)
	printer := message.NewPrinter(language.English)
	customerRef := fs.Collection("customers").Doc(t.CustomerID)
	entityRef := fs.Collection("entities").Doc(t.EntityID)

	invoiceDocSnaps, err := fs.Collection("invoices").
		Where("CANCELED", "==", false).
		Where("customer", "==", customerRef).
		Where("entity", "==", entityRef).
		Where("notification.sent", "==", false).
		Where("notification.created", ">=", minCreateTime).
		Limit(20).
		Documents(ctx).GetAll()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	prevInvoiceDocSnaps, err := fs.Collection("invoices").
		Where("CANCELED", "==", false).
		Where("customer", "==", customerRef).
		Where("notification.sent", "==", true).
		Limit(1).
		Documents(ctx).GetAll()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	batch := fs.Batch()
	invoices := make([]InvoiceNotificationData, 0)
	attachments := make([]notificationcenter.EmailAttachment, 0)

	for _, docSnap := range invoiceDocSnaps {
		var invoice FullInvoice
		if err := docSnap.DataTo(&invoice); err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if invoice.Notification == nil || invoice.Notification.Created.IsZero() || invoice.Notification.Created.Before(minCreateTime) {
			continue
		}

		var linkURL string

		if invoice.ExternalFilesSubForm != nil {
			for _, externalFile := range invoice.ExternalFilesSubForm {
				if externalFile.URL != nil {
					linkURL = *externalFile.URL
					break
				}
			}
		}

		var products []string

		for _, p := range invoice.Products {
			productLabel := common.FormatAssetType(p)
			if productLabel != "" {
				products = append(products, productLabel)
			}
		}

		attachment, err := getInvoicePDFencoded(ctx, invoice)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		attachments = append(attachments, notificationcenter.EmailAttachment{
			Filename: fmt.Sprintf("invoice-%s.pdf", invoice.ID),
			Type:     "application/pdf",
			Content:  attachment,
		})

		invoices = append(invoices, InvoiceNotificationData{
			ID:       invoice.ID,
			PayDate:  invoice.PayDate.Format("02 Jan, 2006"),
			LinkURL:  linkURL,
			Products: strings.Join(products, ", "),
			Amount:   formatAmount(printer, invoice.TotalTax, invoice.Symbol),
		})

		batch.Update(docSnap.Ref, []firestore.Update{
			{FieldPath: []string{"notification", "sent"}, Value: true},
		})
	}

	if len(invoices) <= 0 {
		l.Info("entity has no new invoices")
		ctx.Status(http.StatusOK)

		return
	}

	client, err := notificationcenter.NewClient(ctx, common.ProjectID)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	courierMessage, err := getInvoiceNotifications(ctx, fs, entityRef, invoices, attachments, customerRef)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if courierMessage != nil {
		task, err := client.CreateSendTask(ctx, *courierMessage)
		if err != nil {
			l.Errorf("failed to send notification to %s", courierMessage.Email)
			return
		}

		l.Infof("notification task %s to %s created", task.GetName(), courierMessage.Email)
	}

	// updates invoices notification.sent status whether or not there are any recipients
	if _, err := batch.Commit(ctx); err != nil {
		var invoiceIds []string
		for _, invoice := range invoices {
			invoiceIds = append(invoiceIds, invoice.ID)
		}

		l.Errorf("failed to update notification.sent status for invoices: %s", strings.Join(invoiceIds, ", "))
		ctx.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	isFirstInvoice := len(prevInvoiceDocSnaps) == 0
	if isFirstInvoice {
		csmService := csm_service.NewService(ctx, fs, nil)
		if err := csmService.SendFirstInvoiceEmail(ctx, customerRef.ID); err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
}

func getInvoiceNotifications(
	ctx *gin.Context,
	fs *firestore.Client,
	entityRef *firestore.DocumentRef,
	invoices []InvoiceNotificationData,
	attachments []notificationcenter.EmailAttachment,
	customerRef *firestore.DocumentRef,
) (*notificationcenter.Notification, error) {
	rs := recipientsService.NewRecipientsService(fs)

	configs, err := rs.GetNotificationRecipientsForCustomer(
		ctx,
		customerRef,
		rsDomain.NotificationNewInvoices,
		rsDomain.WithInvoiceFilter(entityRef, true),
	)
	if err != nil {
		return nil, err
	}

	if len(configs) <= 0 {
		return nil, nil
	}

	customerNotification := rs.GetCustomersNotifications(configs)[customerRef.ID]

	return populateNewInvoiceNotification(ctx, fs, customerNotification, invoices, attachments, customerRef), nil
}

func getInvoicePDFencoded(ctx context.Context, invoice FullInvoice) (string, error) {
	r, err := invoice.getPDFReader(ctx)
	if err != nil {
		return "", err
	}
	defer r.Close()

	file, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(file), nil
}

func populateNewInvoiceNotification(
	ctx *gin.Context,
	fs *firestore.Client,
	notification *notificationcenter.Notification,
	invoices []InvoiceNotificationData,
	attachments []notificationcenter.EmailAttachment,
	customerRef *firestore.DocumentRef) *notificationcenter.Notification {
	ccs := getAccountTeamCCs(fs, ctx, customerRef.ID)

	// CCs are currently disabled for new invoices because AMs cannot yet opt out and were receiving an unpleasant level of traffic.
	// https://doitintl.atlassian.net/browse/CMP-17832 has been created to resolve.
	disableCCs := true

	isEmailNotification := len(notification.Email) > 0

	notification.Mock = !common.Production
	notification.Template = notificationcenter.NewInvoiceTemplate
	notification.Data = map[string]interface{}{
		"invoices":   invoices,
		"supportUrl": fmt.Sprintf("https://console.doit.com/customers/%s/support", customerRef.ID),
		"profileUrl": fmt.Sprintf("https://console.doit.com/customers/%s/notifications", customerRef.ID),
	}

	if isEmailNotification {
		notification.EmailFrom = notificationcenter.EmailFrom{
			Email: notificationcenter.BillingSenderAddress,
			Name:  notificationcenter.SenderName,
		}

		if len(attachments) > 0 {
			notification.EmailAttachments = attachments
		}

		if len(ccs) > 0 && !disableCCs {
			notification.CC = ccs
		}
	}

	return notification
}

func formatAmount(p *message.Printer, amount float64, code string) string {
	if fixer.SupportedCurrency(code) {
		return p.Sprintf("%s%.2f", fixer.FromString(code).Symbol(), amount)
	}

	return p.Sprintf("%s %.2f", code, amount)
}

func getAccountTeamCCs(fs *firestore.Client, ctx *gin.Context, customerID string) []string {
	customerRef := fs.Collection("customers").Doc(customerID)
	emails := make([]string, 0)
	snap, err := customerRef.Get(ctx)

	if err != nil {
		return emails
	}

	var customer common.Customer
	if err := snap.DataTo(&customer); err != nil {
		return emails
	}

	emails = getAccountTeamMembersEmails(ctx, customer, common.AccountManagerRoleSAM)

	if len(emails) == 0 {
		emails = getAccountTeamMembersEmails(ctx, customer, common.AccountManagerRoleFSR)
	}

	return emails
}

func getAccountTeamMembersEmails(ctx *gin.Context, customer common.Customer, role common.AccountManagerRole) []string {
	emails := make([]string, 0)

	for _, at := range customer.AccountTeam {
		if at.Company != "doit" {
			continue
		}

		amSnap, err := at.Ref.Get(ctx)
		if err != nil {
			continue
		}

		var am common.AccountManager
		if err := amSnap.DataTo(&am); err != nil {
			continue
		}

		if am.Role == role && !contains(emails, am.Email) {
			emails = append(emails, am.Email)
		}
	}

	return emails
}

func contains(slice []string, element string) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}

	return false
}
