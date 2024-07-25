package contract

import (
	"math"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func UpdateCommitmentContracts(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	fs := common.GetFirestoreClient(ctx)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	contractDocSnaps, err := fs.Collection("contracts").
		Where("active", "==", true).
		Where("isCommitment", "==", true).
		Where("type", "in", []string{common.Assets.GoogleCloud, common.Assets.AmazonWebServices}).
		Where("endDate", ">=", today.AddDate(0, -2, 0)).
		OrderBy("endDate", firestore.Asc).
		Documents(ctx).
		GetAll()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	for _, contractDocSnap := range contractDocSnaps {
		var contract common.Contract
		if err := contractDocSnap.DataTo(&contract); err != nil {
			ctx.Error(err)
			continue
		}

		if contract.CommitmentPeriods == nil || len(contract.CommitmentPeriods) <= 0 {
			continue
		}

		if err := processCommitmentContract(ctx, fs, contractDocSnap, &contract, today); err != nil {
			l.Errorf("contract %s: %s", contractDocSnap.Ref.ID, err.Error())
		}
	}
}

func processCommitmentContract(ctx *gin.Context, fs *firestore.Client, docSnap *firestore.DocumentSnapshot, contract *common.Contract, today time.Time) error {
	rollover := make([]float64, len(contract.CommitmentPeriods)+1)
	data := make([]interface{}, len(contract.CommitmentPeriods))

	for i, cp := range contract.CommitmentPeriods {
		ended := today.After(cp.EndDate)
		current := today.After(cp.StartDate) && !ended
		totalDays := cp.EndDate.Sub(cp.StartDate).Hours() / 24
		daysPassed := today.Sub(cp.StartDate).Hours() / 24

		if totalDays <= 0 {
			totalDays = 1
		}

		if daysPassed < 0 {
			daysPassed = 0
		}

		r := math.Min(daysPassed/totalDays, 1)
		useLastQ := current && r > 0.25
		lastQ := today.Add(time.Hour * time.Duration((totalDays+1)*24*-0.25))
		total := 0.0
		est := 0.0

		if daysPassed > 0 {
			invoiceDocSnaps, err := fs.Collection("invoices").
				Where("CANCELED", "==", false).
				Where("customer", "==", contract.Customer).
				Where("entity", "==", contract.Entity).
				Where("PRODUCTS", "array-contains", contract.Type).
				Where("IVDATE", ">=", cp.StartDate).
				Where("IVDATE", "<=", cp.EndDate).
				OrderBy("IVDATE", firestore.Asc).
				Documents(ctx).
				GetAll()
			if err != nil {
				return err
			}

			var lastInvoiced time.Time

			for _, invoiceDocSnap := range invoiceDocSnaps {
				var invoice invoices.FullInvoice
				if err := invoiceDocSnap.DataTo(&invoice); err != nil {
					return err
				}

				if invoice.Date.After(lastInvoiced) {
					lastInvoiced = invoice.Date
				}

				total += invoice.USDTotal

				if current {
					if useLastQ {
						if invoice.Date.After(lastQ) {
							est += invoice.USDTotal
						}
					} else {
						est += invoice.USDTotal
					}
				}
			}

			if current {
				daysInvoiced := lastInvoiced.Sub(cp.StartDate).Hours() / 24
				est = total / (daysInvoiced / totalDays)
			}

			if contract.CommitmentRollover && ended && total > cp.Value {
				rollover[i+1] = total - cp.Value
			}
		}

		data[i] = map[string]interface{}{
			"value":     cp.Value,
			"startDate": cp.StartDate,
			"endDate":   cp.EndDate,
			"total":     total,
			"estimated": est,
			"current":   current,
			"ended":     ended,
			"rollover":  rollover[i],
		}
	}

	fs.Collection("dashboards").Doc("commitment-contracts").Collection("commitmentContracts").Doc(docSnap.Ref.ID).Set(ctx, map[string]interface{}{
		"entity":             contract.Entity,
		"customer":           contract.Customer,
		"timestamp":          firestore.ServerTimestamp,
		"type":               contract.Type,
		"commitmentRollover": contract.CommitmentRollover,
		"commitmentPeriods":  data,
	})

	return nil
}
