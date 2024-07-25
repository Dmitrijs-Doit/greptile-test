package invoices

import (
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
)

type DailyCollection struct {
	Totals    map[string]float64 `firestore:"totals"`
	Weight    float64            `firestore:"weight"`
	Timestamp time.Time          `firestore:"timestamp,serverTimestamp"`
}

type GraphDataPoint struct {
	Severity map[string]*GraphSeverityDataPoint `firestore:"severity"`
	Date     time.Time                          `firestore:"date"`
}

type GraphSeverityDataPoint struct {
	Currency map[string]float64 `firestore:"currency"`
	Value    float64            `firestore:"value"`
}

func NewGraphSeverityDataPoint() *GraphSeverityDataPoint {
	return &GraphSeverityDataPoint{
		Value:    0.0,
		Currency: newTotalPerCurrency(),
	}
}

func DebtAnalyticsHandler(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	todayString := today.Format(dateFormat)

	severity20 := today.AddDate(0, 0, -35)
	severity30 := today.AddDate(0, 0, -60)
	severity40 := today.AddDate(0, 0, -90)

	collectionOverdue := DailyCollection{
		Totals: newTotalPerCurrency(),
		Weight: 0.0,
	}

	collectionOpen := DailyCollection{
		Totals: newTotalPerCurrency(),
		Weight: 0.0,
	}

	collectionGraphDataPoint := GraphDataPoint{
		Date: today,
		Severity: map[string]*GraphSeverityDataPoint{
			"0":  NewGraphSeverityDataPoint(),
			"10": NewGraphSeverityDataPoint(),
			"20": NewGraphSeverityDataPoint(),
			"30": NewGraphSeverityDataPoint(),
			"40": NewGraphSeverityDataPoint(),
		},
	}

	iter := fs.Collection("invoices").Where("PAID", "==", false).Documents(ctx)

	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		var invoice FullInvoice
		if err := docSnap.DataTo(&invoice); err != nil {
			l.Error(err)
			continue
		}

		debitUSD := invoice.Debit / invoice.USDExchangeRate

		var severity int

		switch {
		case invoice.PayDate.Before(severity40):
			severity = 40
		case invoice.PayDate.Before(severity30):
			severity = 30
		case invoice.PayDate.Before(severity20):
			severity = 20
		case invoice.PayDate.Before(today):
			severity = 10
		default:
			severity = 0
		}

		if severity > 0 {
			collectionOverdue.Totals[invoice.Symbol] += invoice.Debit
			collectionOverdue.Weight += debitUSD
		}

		collectionOpen.Totals[invoice.Symbol] += invoice.Debit
		collectionOpen.Weight += debitUSD
		collectionGraphDataPoint.Severity[strconv.Itoa(severity)].Value += debitUSD
		collectionGraphDataPoint.Severity[strconv.Itoa(severity)].Currency[invoice.Symbol] += invoice.Debit
	}

	collectionRef := fs.Collection("collection")
	if _, err := collectionRef.Doc("daily-collection").Collection("dailyCollection").Doc(todayString).Set(ctx, collectionOverdue); err != nil {
		l.Error(err)
	}

	if _, err := collectionRef.Doc("daily-collection-open").Collection("dailyCollection").Doc(todayString).Set(ctx, collectionOpen); err != nil {
		l.Error(err)
	}

	u := make(map[string]interface{})
	u[todayString] = collectionGraphDataPoint
	u["timestamp"] = firestore.ServerTimestamp

	if _, err := collectionRef.Doc("debt-analytics").Collection("debtAnalyticsGraph").Doc(today.Format("2006-01")).Set(ctx, u, firestore.MergeAll); err != nil {
		l.Error(err)
	}
}

func newTotalPerCurrency() map[string]float64 {
	newMap := make(map[string]float64)
	for _, c := range fixer.Currencies {
		newMap[string(c)] = 0.0
	}

	return newMap
}
