package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/cloudtasks/iface"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	reportDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/invoicing"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/flexsave/dal"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/flexsave/domain/billing"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

var (
	ErrInvalidData = errors.New("invalid data returned from query")
)

type FlexsaveInvoiceService struct {
	loggerProvider     logger.Provider
	invoiceMonthParser invoicing.InvoiceMonthParser
	dal                dal.FlexsaveStandalone
	assetDAL           assetsDal.Assets
	custometrDAL       customerDal.Customers
	cloudTaskClient    iface.CloudTaskClient
	billingData        BillingData
}

func NewFlexsaveInvoiceService(loggerProvider logger.Provider, conn *connection.Connection) (*FlexsaveInvoiceService, error) {
	parser := invoicing.DefaultInvoiceMonthParser{InvoicingDaySwitchOver: 10}

	customerDal := customerDal.NewCustomersFirestoreWithClient(conn.Firestore)
	reportDal := reportDal.NewReportsFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(loggerProvider, conn, reportDal, customerDal)
	if err != nil {
		return nil, err
	}

	billingData := &BillingDataService{
		logger.FromContext,
		&invoicing.BillingDataQueryBuilder{},
		cloudAnalytics,
	}

	firestoreFun := conn.Firestore
	cloudTaskClient := conn.CloudTaskClient

	return &FlexsaveInvoiceService{
		loggerProvider,
		&parser,
		dal.NewFlexsaveStandaloneFirestoreWithClient(firestoreFun),
		assetsDal.NewAssetsFirestoreWithClient(firestoreFun),
		customerDal,
		cloudTaskClient,
		billingData,
	}, nil
}

// UpdateFlexsaveInvoicingData processes all customer aws fssa assets
func (s *FlexsaveInvoiceService) UpdateFlexsaveInvoicingData(ctx context.Context, invoiceMonthInput string, provider string) error {
	logger := s.loggerProvider(ctx)

	invoiceMonth, err := s.invoiceMonthParser.GetInvoiceMonth(invoiceMonthInput)
	if err != nil {
		return err
	}

	logger.Infof("invoiceMonth %v", invoiceMonth)

	assets, err := s.assetDAL.ListBaseAssets(ctx, provider)
	if err != nil {
		return err
	}

	uniqueCustomers := make(map[string]bool)
	yearMonthDay := invoiceMonth.Format(times.YearMonthDayLayout)

	for _, asset := range assets {
		t := billing.InvoicingTask{
			InvoiceMonth: yearMonthDay,
			Provider:     provider,
		}

		// only create new task for unique customer not each asset
		if ok := uniqueCustomers[asset.Customer.ID]; ok {
			continue
		}

		uniqueCustomers[asset.Customer.ID] = true

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf("/tasks/invoicing/%s/customer/%s", provider, asset.Customer.ID),
			Queue:  common.TaskQueueInvoicingAnalytics,
		}

		if _, err = s.cloudTaskClient.CreateTask(ctx, config.Config(t)); err != nil {
			logger.Errorf("UpdateFlexsaveInvoicingData, failed to create task for customer: %s, error: %v", asset.Customer.ID, err)
			return err
		}
	}

	return nil
}

func (s *FlexsaveInvoiceService) FlexsaveDataWorker(ginCtx *gin.Context, customerID string, invoiceMonthInput string, provider string) error {
	logger := s.loggerProvider(ginCtx)

	invoiceMonth, err := s.invoiceMonthParser.GetInvoiceMonth(invoiceMonthInput)
	if err != nil {
		return err
	}

	logger.Infof("invoiceMonth %v", invoiceMonth)
	logger.Infof("Starting Analytics %sStandalone InvoicingDataWorker for customer %s", provider, customerID)

	cloudProvider := strings.TrimSuffix(provider, "-standalone")

	rows, err := s.billingData.GetCustomerBillingRows(ginCtx, customerID, invoiceMonth, cloudProvider)
	if err != nil {
		return err
	}

	assetInvoiceMap := make(map[string]map[string]float64)
	customerRef := s.custometrDAL.GetRef(ginCtx, customerID)

	assetMap := make(map[string]pkg.MonthlyBillingFlexsaveStandalone)

	for _, row := range rows {
		// row structure ["account_id", "billing_account_id" (gcp)/"aws/payer_account_id" (aws), cost_metric, usage_metric, savings_metric]
		// e.g. ["12356789012", "456", 23, 89, 0]
		accountID, ok := row[0].(string)
		if !ok || accountID == "" {
			return ErrInvalidData
		}

		assetID, ok := row[1].(string)
		if !ok || assetID == "" {
			return ErrInvalidData
		}

		rowSpend, ok := row[2].(float64)
		if !ok {
			return ErrInvalidData
		}

		if _, ok := assetInvoiceMap[assetID]; !ok {
			assetInvoiceMap[assetID] = make(map[string]float64)
		}

		assetInvoiceMap[assetID][accountID] += rowSpend
	}

	yearMonth := invoiceMonth.Format(times.YearMonthLayout)

	for assetID, spend := range assetInvoiceMap {
		monthlyBillingFlexsave := pkg.MonthlyBillingFlexsaveStandalone{
			Customer:     customerRef,
			Spend:        spend,
			InvoiceMonth: yearMonth,
			Type:         provider,
		}

		assetDocID := fmt.Sprintf("%s-%s", provider, assetID)
		assetMap[assetDocID] = monthlyBillingFlexsave
	}

	if err := s.dal.BatchSetFlexsaveBillingData(ginCtx, assetMap); err != nil {
		return err
	}

	return nil
}
