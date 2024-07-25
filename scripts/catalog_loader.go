package scripts

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

var googleWorkspacePrevPriceEndDate = time.Date(2023, 4, 11, 0, 0, 0, 0, time.UTC)

type catalogLoaderParams struct {
	Type string `json:"type"`
}

func CatalogLoader(ctx *gin.Context) []error {
	var params catalogLoaderParams
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	if params.Type != common.Assets.GSuite && params.Type != common.Assets.Office365 {
		err := fmt.Errorf("invalid type: %s", params.Type)
		return []error{err}
	}

	f, err := os.Open("./" + params.Type + ".csv")
	if err != nil {
		return []error{err}
	}
	defer f.Close()

	batch := fb.NewAutomaticWriteBatch(fs, 100)
	catalogCollection := fs.Collection("catalog").Doc(params.Type).Collection("services")
	csvReader := csv.NewReader(f)

	for i := 0; ; i++ {
		row, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				return batch.Commit(ctx)
			}

			return []error{err}
		}

		fmt.Println(row)

		if len(row) != 13 {
			err := errors.New("invalid row length")
			return []error{err}
		}

		skuName := row[0]
		skuID := row[1]
		plan := row[2]
		payment := row[3]
		itemType := row[4]
		usd, _ := strconv.ParseFloat(row[5], 64)
		eur, _ := strconv.ParseFloat(row[6], 64)
		gbp, _ := strconv.ParseFloat(row[7], 64)
		aud, _ := strconv.ParseFloat(row[8], 64)
		brl, _ := strconv.ParseFloat(row[9], 64)
		nok, _ := strconv.ParseFloat(row[10], 64)
		dkk, _ := strconv.ParseFloat(row[11], 64)
		skuPriority := row[12]

		if plan != "ANNUAL" && plan != "FLEXIBLE" {
			err := errors.New("invalid plan type")
			return []error{err}
		}

		if payment != "YEARLY" && payment != "MONTHLY" {
			err := errors.New("invalid payment type")
			return []error{err}
		}

		if itemType != "ONLINE" && itemType != "OFFLINE" {
			err := errors.New("invalid item type")
			return []error{err}
		}

		docSnaps, err := catalogCollection.
			Where("skuId", "==", skuID).
			Where("plan", "==", plan).
			Where("payment", "==", payment).
			Documents(ctx).GetAll()
		if err != nil {
			return []error{err}
		}

		if len(docSnaps) > 1 {
			err := fmt.Errorf("more than one document found for skuId: %s, plan: %s, payment: %s", skuID, plan, payment)
			return []error{err}
		}

		data := map[string]interface{}{
			"id":      i,
			"skuName": skuName,
			"skuId":   skuID,
			"plan":    plan,
			"payment": payment,
			"type":    itemType,
			"price": map[string]float64{
				"USD": usd,
				"EUR": eur,
				"GBP": gbp,
				"AUD": aud,
				"BRL": brl,
				"NOK": nok,
				"DKK": dkk,
			},
			"skuPriority": skuPriority,
		}

		// If there is a document with the same skuId, plan and payment, then we need to update the prevPriceEndDate and prevPrice
		// Otherwise, we need to create a new document with prevPriceEndDate and prevPrice set to nil
		if len(docSnaps) == 1 {
			// update existing document
			docSnap := docSnaps[0]

			if params.Type == common.Assets.GSuite {
				prevPrice, err := docSnap.DataAt("price")
				if err != nil {
					err = fmt.Errorf("error getting prevPrice for doc: %s", docSnap.Ref.ID)
					return []error{err}
				}

				data["prevPriceEndDate"] = googleWorkspacePrevPriceEndDate
				data["prevPrice"] = prevPrice
			}

			batch.Set(docSnap.Ref, data)
		} else {
			// create new document, no prev price
			data["prevPriceEndDate"] = nil
			data["prevPrice"] = nil

			batch.Create(catalogCollection.NewDoc(), data)
		}
	}
}
