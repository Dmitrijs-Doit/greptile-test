package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/priority"
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
)

func (s *service) ListCustomerReceipts(ctx context.Context, priorityCompany priority.CompanyCode, customerName string) (priorityDomain.TInvoices, error) {
	return s.priorityReaderWriter.ListCustomerReceipts(ctx, priorityCompany, customerName)
}
