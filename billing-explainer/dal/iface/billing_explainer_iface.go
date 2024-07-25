package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/billing-explainer/domain"
)

//go:generate mockery --name BigQueryDAL --output ../mocks
type BigQueryDAL interface {
	GetInvoiceSummary(ctx context.Context, explainerParams domain.BillingExplainerParams, payerTable, accountIDString, PayerID, flexsaveCondition string) ([]domain.SummaryBQ, error)
	GetPayerIDFromAccountsHistory(ctx context.Context, startOfMonth string, customerID string) ([]domain.PayerAccountHistoryResult, error)
	GetServiceBreakdownData(ctx context.Context, explainerParams domain.BillingExplainerParams, payerTable, accountIDString, PayerID, flexsaveCondition string) ([]domain.ServiceRecord, error)
	GetAccountBreakdownData(ctx context.Context, explainerParams domain.BillingExplainerParams, payerTable, accountIDString, PayerID, flexsaveCondition string) ([]domain.AccountRecord, error)
}

//go:generate mockery --name FirestoreDAL --output ../mocks
type FirestoreDAL interface {
	UpdateEntityFirestoreDoc(ctx context.Context, isBackfill bool, yearMonth string, entityID string, invoicingMode string, summaryBqResults []domain.SummaryBQ, bucketName string, serviceBreakdownResults []domain.ServiceRecord, accountBreakdownResults []domain.AccountRecord) error
	GetPayerAccountDoc(ctx context.Context, payerID string) (map[string]interface{}, error)
}
