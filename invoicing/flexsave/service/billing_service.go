package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/invoicing"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	flexSaveCharges = "FlexsaveCharges"
)

type BillingData interface {
	GetFlexsaveQuery(ctx context.Context, invoiceMonth time.Time, accounts []string, provider string) (*cloudanalytics.QueryRequest, error)
	GetCustomerBillingRows(ctx *gin.Context, customerID string, invoiceMonth time.Time, provider string) ([][]bigquery.Value, error)
}

type BillingDataService struct {
	LoggerProvider logger.Provider
	QueryBuilder   invoicing.BillingDataQuery
	CloudAnalytics cloudanalytics.CloudAnalytics
}

func (b *BillingDataService) GetCustomerBillingRows(ctx *gin.Context, customerID string, invoiceMonth time.Time, provider string) ([][]bigquery.Value, error) {
	logger := b.LoggerProvider(ctx)

	accounts, err := b.getAccounts(ctx, customerID, provider)
	if err != nil {
		return nil, err
	}

	queryRequest, err := b.GetFlexsaveQuery(ctx, invoiceMonth, accounts, provider)
	if err != nil {
		return nil, err
	}

	billingQueryResult, err := b.CloudAnalytics.GetQueryResult(ctx, queryRequest, customerID, "")
	if err != nil {
		return nil, err
	}

	if billingQueryResult.Error != nil {
		return nil, fmt.Errorf("monthly billing data query failed for customer: %s and invoiceMonth: %v resulted in error: %#v", customerID, invoiceMonth, *billingQueryResult.Error)
	}

	if len(billingQueryResult.Rows) == 0 {
		logger.Debugf("customer %s: billing data query returned 0 rows", customerID)
		return nil, nil
	}

	return billingQueryResult.Rows, nil
}

func (b *BillingDataService) GetFlexsaveQuery(ctx context.Context, invoiceMonth time.Time, accounts []string, provider string) (*cloudanalytics.QueryRequest, error) {
	filters, err := b.QueryBuilder.GetBillingQueryFilters(provider, []string{flexSaveCharges}, false, "", nil, false)
	if err != nil {
		return nil, err
	}

	var rows []*domainQuery.QueryRequestX

	accountID, err := domainQuery.NewRow(domainQuery.FieldProjectID)
	if err != nil {
		return nil, err
	}

	rows = append(rows, accountID)

	switch provider {
	case common.Assets.AmazonWebServices:
		payerAccountID, err := domainQuery.NewRowConstituentField("system_labels", "aws/payer_account_id")
		if err != nil {
			return nil, err
		}

		rows = append(rows, payerAccountID)
	case common.Assets.GoogleCloud:
		billingAccountID, err := domainQuery.NewRow(domainQuery.FieldBillingAccountID)
		if err != nil {
			return nil, err
		}

		rows = append(rows, billingAccountID)
	default:
		return nil, errors.New("invalid provider")
	}

	qr := cloudanalytics.QueryRequest{
		Origin:         domainOrigin.QueryOriginFromContext(ctx),
		Type:           "report",
		CloudProviders: &[]string{provider},
		Accounts:       accounts,
		TimeSettings:   b.QueryBuilder.GetTimeSettings(invoiceMonth),
		Rows:           rows,
		Filters:        filters,
		Currency:       fixer.USD,
		NoAggregate:    true,
	}

	return &qr, nil
}

func (b *BillingDataService) getAccounts(ctx *gin.Context, customerID string, provider string) ([]string, error) {
	switch provider {
	case common.Assets.AmazonWebServices:
		return []string{customerID}, nil
	case common.Assets.GoogleCloud:
		return b.CloudAnalytics.GetAccounts(ctx, customerID, &[]string{provider}, nil)
	default:
		return nil, errors.New("invalid provider")
	}
}
