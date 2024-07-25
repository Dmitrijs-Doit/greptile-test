package invoicing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type CancelIssuedInvoicesReq struct {
	EntityIDs 		[]string `json:"entityIds"`
	Month     		string   `json:"month"`
	Types    	 	[]string `json:"types"`
	Reason  		string   `json:"cancellationReason"`
}

const (
	entityInvoicesPath = "billing/invoicing/invoicingMonths/%s/monthInvoices/%s/entityInvoices"
	issuedAt           = "issuedAt"
	canceledAt         = "canceledAt"
	note               = "note"
	cancellationReason     = "cancellationReason"
)

func (s *InvoicingService) CancelIssuedInvoices(ctx context.Context, request *CancelIssuedInvoicesReq, email string) error {
	logger := s.Logger(ctx)
	fs := s.Firestore(ctx)

	var errorList []string

	requestTime := time.Now().UTC()
	batch := fb.NewAutomaticWriteBatch(fs, 250)
	processedInvoices := 0

	logger.Infof("CancelIssuedInvoices: request params %+v", request)

	for _, entity := range request.EntityIDs {
		entityInvoicePath := fmt.Sprintf(entityInvoicesPath, request.Month, entity)

		query := fs.Collection(entityInvoicePath).Where("issuedAt", "!=", "")

		if len(request.Types) != 0 {
			query = query.Where("type", "in", request.Types)
		}

		invoices, err := query.Documents(ctx).GetAll()
		if err != nil {
			return err
		}

		for _, invoice := range invoices {
			logger.Infof("CancelIssuedInvoices: updating entity invoice, setting issuedAt = nil for entityId:%s invoiceId:%s", entity, invoice.Ref.ID)

			currentIssuedAt, err := invoice.DataAt(issuedAt)
			if err != nil {
				errorList = addErrorToList(errorList, logger, fmt.Sprintf("updating entity invoice errored while reading issuedAt, for entityId:%s invoiceId:%s error:%+v", entity, invoice.Ref.ID, err))
				continue
			}

			currentIssuedAtTime, ok := currentIssuedAt.(time.Time)
			if !ok {
				errorList = addErrorToList(errorList, logger, fmt.Sprintf("updating entity invoice errored while casting issuedAt, for entityId:%s invoiceId:%s error:%+v", entity, invoice.Ref.ID, err))
				continue
			}

			batch.Update(invoice.Ref, []firestore.Update{{
				Path:  issuedAt,
				Value: nil,
			}, {
				Path:  cancellationReason,
				Value: request.Reason,
			}, {
				Path:  canceledAt,
				Value: requestTime,
			}, {
				Path:  note,
				Value: fmt.Sprintf("Previously issued on %+v. Canceled by %s", currentIssuedAtTime.Format(time.RFC822), email),
			}})

			processedInvoices++
		}
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		allErrors := fmt.Errorf("error: commit failed: %+v", errors.Join(errs...).Error())
		logger.Errorf(allErrors.Error())

		return allErrors
	}

	if len(errorList) != 0 {
		allErrors := fmt.Errorf("warning: some entityIds failed update: %s", strings.Join(errorList, ";"))
		return allErrors
	}

	logger.Infof("CancelIssuedInvoices: updating entity invoice, updated %d invoice documents", processedInvoices)

	if processedInvoices == 0 {
		return fmt.Errorf("No invoices to cancel found")
	}

	return nil
}

func addErrorToList(errorList []string, logger logger.ILogger, errDesc string) []string {
	logger.Errorf(errDesc)
	return append(errorList, errDesc)
}
