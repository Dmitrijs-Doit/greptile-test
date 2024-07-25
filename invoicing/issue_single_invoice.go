package invoicing

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

const Issue JobType = "issue"
const issueInvoicelogName = "manualOperations - issue"

func (s *InvoicingService) IssueSingleCustomerInvoice(ctx *gin.Context, request IssueRecalculateRequest) error {
	input := request.Input
	l := logger.FromContext(ctx)

	l.Infof(FormatLogEntryInfo(issueInvoicelogName, input.CustomerID, input.InvoiceMonth, "Starting issue single invoice"))

	customer, err := s.customersDAL.GetCustomer(ctx, input.CustomerID)

	if err != nil {
		return fmt.Errorf(FormatLogEntryError(issueInvoicelogName, input.CustomerID, input.InvoiceMonth, "customerId not found: %v", err))
	}

	if !utils.IsIssuableAssetType(input.AssetType) {
		return fmt.Errorf(FormatLogEntryError(issueInvoicelogName, input.CustomerID, input.InvoiceMonth, "assetType is not issuable"))
	}

	_, err = time.Parse("2006-01-02", input.InvoiceMonth)

	if err != nil {
		return fmt.Errorf(FormatLogEntryError(issueInvoicelogName, input.CustomerID, input.InvoiceMonth, "wrong InvoiceMonth format: %v", err))
	}

	jobID, err := s.createJobDoc(ctx, request, Issue)
	if err != nil {
		return fmt.Errorf(FormatLogEntryError(issueInvoicelogName, input.CustomerID, input.InvoiceMonth, "createJobDoc failed: %v", err))
	}

	go s.processIssueSingleCustomerInvoice(ctx, jobID, request, customer.PrimaryDomain)

	return nil
}

func (s *InvoicingService) processIssueSingleCustomerInvoice(ctx *gin.Context, jobID string, request IssueRecalculateRequest, primaryDomain string) {
	l := logger.FromContext(ctx)
	input := request.Input

	l.Infof(FormatLogEntryInfo(issueInvoicelogName, input.CustomerID, input.InvoiceMonth, "Step 1: MBDA started"))

	if err := s.awsAnalyticsService.AmazonWebServicesInvoicingDataWorker(ctx, input.CustomerID, input.InvoiceMonth, false); err != nil {
		s.UpdateJobDoc(ctx, jobID, StatusFailed, err)
		return
	}

	l.Infof(FormatLogEntryInfo(issueInvoicelogName, input.CustomerID, input.InvoiceMonth, "Step 1: MBDA completed"))
	l.Infof(FormatLogEntryInfo(issueInvoicelogName, input.CustomerID, input.InvoiceMonth, "Step 2: Draft invoice started"))

	inputDraftInvoice := ProcessInvoicesInput{
		InvoiceMonth:  input.InvoiceMonth,
		PrimaryDomain: primaryDomain,
		TimeIndex:     "-2",
	}

	processWithCloudTask := false

	if err := s.ProcessCustomersInvoices(ctx, &inputDraftInvoice, processWithCloudTask); err != nil {
		s.UpdateJobDoc(ctx, jobID, StatusFailed, err)
		return
	}

	l.Infof(FormatLogEntryInfo(issueInvoicelogName, input.CustomerID, input.InvoiceMonth, "Step 2: Draft invoice completed"))
	l.Infof(FormatLogEntryInfo(issueInvoicelogName, input.CustomerID, input.InvoiceMonth, "Step 3: Export invoice started"))

	year, month := splitYearMonth(input.InvoiceMonth)
	inputExportInvoice := ExportInvoicesRequest{
		CustomerID: &input.CustomerID,
		Year:       year,
		Month:      month,
		Types:      []string{input.AssetType},
		Override:   input.Override,
	}

	devDriveName, _ := s.getDevDriveName(ctx, request.DevMode)

	if _, err := s.ExportInvoices(ctx, &inputExportInvoice, request.UID, request.Email, request.DevMode, devDriveName); err != nil {
		s.UpdateJobDoc(ctx, jobID, StatusFailed, err)
		return
	}

	l.Infof(FormatLogEntryInfo(issueInvoicelogName, input.CustomerID, input.InvoiceMonth, "Step 3: Export invoice completed"))

	s.UpdateJobDoc(ctx, jobID, StatusCompleted, nil)

	l.Infof(FormatLogEntryInfo(issueInvoicelogName, input.CustomerID, input.InvoiceMonth, "Finished issue single invoice"))
}

func splitYearMonth(invoiceMonth string) (int64, int64) {
	parts := strings.Split(invoiceMonth, "-")
	yearStr := parts[0]
	monthStr := parts[1]
	year, _ := strconv.Atoi(yearStr)
	month, _ := strconv.Atoi(monthStr)
	return int64(year), int64(month)
}

func (s *InvoicingService) getDevDriveName(ctx *gin.Context, devMode bool) (devDriveName *string, err error) {
	if !devMode {
		return nil, nil
	}

	fs := s.Firestore(ctx)
	isAllowed, devExportDrive, err := checkDevModeAllowed(ctx, fs)

	if err != nil || !isAllowed {
		return &devExportDrive, err
	}

	return &devExportDrive, nil
}
