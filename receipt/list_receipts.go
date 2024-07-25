package receipt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/priority"
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
	serviceIface "github.com/doitintl/hello/scheduled-tasks/priority/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

const (
	dateTimeFormat = "2006-01-02T15:04:05-07:00"

	StatusCanceled = "Canceled"
	StatusApproved = "Approved"
)

type CustomerTask struct {
	EntityID        string `json:"entity_id"`
	PriorityID      string `json:"priority_id"`
	PriorityCompany string `json:"priority_company"`
}

func SyncReceipts(ctx context.Context) error {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	iter := fs.Collection("entities").Select("priorityId", "priorityCompany").Documents(ctx)
	defer iter.Stop()

	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return err
		}

		var entity common.Entity

		if err := docSnap.DataTo(&entity); err != nil {
			l.Errorf("entity: %v", err)
			continue
		}

		t := CustomerTask{
			EntityID:        docSnap.Ref.ID,
			PriorityID:      entity.PriorityID,
			PriorityCompany: entity.PriorityCompany,
		}

		taskBody, err := json.Marshal(t)
		if err != nil {
			return err
		}

		config := common.CloudTaskConfig{
			Method:       cloudtaskspb.HttpMethod_POST,
			Path:         "/tasks/receipts",
			Queue:        common.TaskQueueInvoicesSync,
			Body:         taskBody,
			ScheduleTime: nil,
		}

		_, err = common.CreateCloudTask(ctx, &config)
		if err != nil {
			return err
		}
	}

	return nil
}

func SyncCustomerReceipts(ctx *gin.Context, priorityService serviceIface.Service) error {
	l := logger.FromContext(ctx)

	if !fixer.CurrencyHistoricalTimeseriesInitialized {
		return errors.New("currency historical timeseries is not available")
	}

	var t CustomerTask

	if err := ctx.ShouldBindJSON(&t); err != nil {
		return err
	}

	l.Info(t)

	fs := common.GetFirestoreClient(ctx)

	entityRef := fs.Collection("entities").Doc(t.EntityID)

	entityDocSnap, err := entityRef.Get(ctx)
	if err != nil {
		return err
	}

	var entity common.Entity

	if err := entityDocSnap.DataTo(&entity); err != nil {
		return err
	}

	customerDocSnap, err := entity.Customer.Get(ctx)
	if err != nil {
		return err
	}

	var customer common.Customer

	if err := customerDocSnap.DataTo(&customer); err != nil {
		return err
	}

	receipts, err := priorityService.ListCustomerReceipts(ctx, priority.CompanyCode(entity.PriorityCompany), entity.PriorityID)
	if err != nil {
		return err
	}

	l.Infof("total receipts: %d", len(receipts.Value))

	batch := fs.Batch()
	batchCount := 0

	metadata := map[string]interface{}{
		"customer": map[string]string{
			"name":          customer.Name,
			"primaryDomain": customer.PrimaryDomain,
		},
		"entity": map[string]string{
			"name": entity.Name,
		},
	}

	for _, receipt := range receipts.Value {
		receipt.Customer = entity.Customer
		receipt.Entity = entityRef
		receipt.Metadata = metadata
		receipt.Company = entity.PriorityCompany
		receipt.Canceled = receipt.Status == StatusCanceled
		receipt.Symbol = fixer.CodeToLabel(receipt.Currency)

		if receipt.DateString != "" {
			if t, err := time.Parse(dateTimeFormat, receipt.DateString); err != nil {
				l.Warningf("invalid date %s: %v", receipt.ID, err)
			} else {
				receipt.Date = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			}
		}

		if receipt.PayDateString != "" {
			if t, err := time.Parse(dateTimeFormat, receipt.PayDateString); err != nil {
				l.Warningf("invalid pay date %s: %v", receipt.ID, err)
				receipt.USDExchangeRate = 1
			} else {
				receipt.PayDate = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
				if convDay, ok := getExchangeRates(receipt.PayDate); ok {
					if exchRate, prs := convDay[receipt.Symbol]; prs {
						receipt.USDExchangeRate = exchRate
					} else {
						receipt.USDExchangeRate = 1
					}
				} else {
					receipt.USDExchangeRate = 1
				}
			}

			receipt.USDTotal = receipt.Total / receipt.USDExchangeRate
		}

		for _, tfncItem := range receipt.TFNCItemsSubform {
			if invoiceID := findInvoice(tfncItem.FNCIREF1, receipts.Value); invoiceID != "" {
				invoiceRef := fs.Collection("invoices").Doc(fmt.Sprintf("%s-%s-%s", receipt.Company, receipt.PriorityID, invoiceID))
				receipt.Invoices = append(receipt.Invoices, invoiceRef)
				receipt.InvoicesPaid = append(receipt.InvoicesPaid, tfncItem)
			}
		}

		receiptRef := fs.Collection("receipts").Doc(fmt.Sprintf("%s-%s-%s", receipt.Company, receipt.PriorityID, receipt.ID))
		batch.Set(receiptRef, receipt)

		batchCount++

		if batchCount >= 50 {
			if _, err := batch.Commit(ctx); err != nil {
				l.Errorf("batch.Commit: %s", err.Error())
			}

			batch = fs.Batch()
			batchCount = 0
		}
	}

	if batchCount > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			l.Errorf("batch.Commit: %s", err.Error())
		}
	}

	return nil
}

func getExchangeRates(date time.Time) (map[string]float64, bool) {
	if convDay, prs := fixer.CurrencyHistoricalTimeseries[date.Year()][date.Format(times.YearMonthDayLayout)]; prs {
		return convDay, true
	} else if date.Before(fixer.TimeseriesStartDate) {
		defaultConvDay := fixer.CurrencyHistoricalTimeseries[fixer.TimeseriesStartDate.Year()][fixer.TimeseriesStartDate.Format(times.YearMonthDayLayout)]
		return defaultConvDay, true
	} else if date.After(fixer.TimeseriesEndDate) {
		defaultConvDay := fixer.CurrencyHistoricalTimeseries[fixer.TimeseriesEndDate.Year()][fixer.TimeseriesEndDate.Format(times.YearMonthDayLayout)]
		return defaultConvDay, true
	}

	return nil, false
}

// findInvoice returns the invoice ID for the given receipt ID
func findInvoice(tid string, receipts []*priorityDomain.TInvoice) string {
	if tid == "" {
		return ""
	}

	if strings.HasPrefix(tid, "RC") {
		for _, r := range receipts {
			if r.ID == tid {
				if len(r.TFNCItemsSubform) == 1 {
					return findInvoice(r.TFNCItemsSubform[0].FNCIREF1, receipts)
				}
			}
		}
	}

	return tid
}
