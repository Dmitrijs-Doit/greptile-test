package receipt

import (
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

func Aggregate(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	now := time.Now()
	parentRef := fs.Collection("collection").Doc("account-receivables").Collection("dailyAccountReceivables")
	minDate := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)

	{
		month := make(map[string]map[string]float64)
		startDate := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)

		receiptsIter := fs.Collection("receipts").Where("CANCELED", "==", false).Where("PAYDATE", ">=", startDate).Documents(ctx)
		defer receiptsIter.Stop()

		for {
			docSnap, err := receiptsIter.Next()
			if err == iterator.Done {
				break
			} else if err != nil {
				l.Errorf("receipts error: %v", err)
				ctx.AbortWithError(http.StatusInternalServerError, err)

				return
			}

			dateObj, _ := docSnap.DataAt("PAYDATE")
			monthStr := dateObj.(time.Time).Format(times.YearMonthLayout)
			dateStr := dateObj.(time.Time).Format(times.YearMonthDayLayout)
			total, _ := docSnap.DataAt("USDTOTAL")

			if int(total.(float64)) > 0 {
				if month[monthStr] == nil {
					month[monthStr] = make(map[string]float64)
				}

				month[monthStr][dateStr] += total.(float64)
			}
		}

		batch := fs.Batch()

		for k, v := range month {
			month, _ := time.Parse(times.YearMonthLayout, k)
			batch.Set(parentRef.Doc(k), map[string]interface{}{
				"receivables": v,
				"month":       month,
			}, firestore.Merge([]string{"receivables"}, []string{"month"}))
		}

		if _, err := batch.Commit(ctx); err != nil {
			l.Error(err)
		}
	}

	{
		month := make(map[string]map[string]float64)

		invoicesIter := fs.Collection("invoices").
			Where("CANCELED", "==", false).
			Where("PAID", "==", false).
			Where("ESTPAYDATE", "<=", now.AddDate(0, 0, 60)).
			Documents(ctx)
		defer invoicesIter.Stop()

		for {
			docSnap, err := invoicesIter.Next()
			if err == iterator.Done {
				break
			} else if err != nil {
				l.Errorf("receipts error: %v", err)
				ctx.AbortWithError(http.StatusInternalServerError, err)

				return
			}

			var invoice invoices.FullInvoice
			if err := docSnap.DataTo(&invoice); err != nil {
				l.Error(err)
				continue
			}

			if invoice.EstimatedPayDate.Before(minDate) {
				continue
			}

			monthStr := invoice.EstimatedPayDate.Format(times.YearMonthLayout)
			dateStr := invoice.EstimatedPayDate.Format(times.YearMonthDayLayout)

			if month[monthStr] == nil {
				month[monthStr] = make(map[string]float64)
			}

			month[monthStr][dateStr] += invoice.USDTotal
		}

		batch := fs.Batch()

		for k, v := range month {
			month, _ := time.Parse(times.YearMonthLayout, k)
			batch.Set(parentRef.Doc(k), map[string]interface{}{
				"receivablesExpected": v,
				"month":               month,
			}, firestore.Merge([]string{"receivablesExpected"}, []string{"month"}))
		}

		if _, err := batch.Commit(ctx); err != nil {
			l.Error(err)
		}
	}
}
