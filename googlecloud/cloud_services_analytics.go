package googlecloud

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
)

type CloudServiceRow struct {
	BillingAccountID string  `bigquery:"billing_account_id"`
	ServiceID        string  `bigquery:"service_id"`
	ServiceName      string  `bigquery:"service_description"`
	TotalCost        float64 `bigquery:"total_cost"`
	InvoiceMonth     string  `bigquery:"invoice_month"`
}

type CustomerData struct {
	Customer *firestore.DocumentRef `firestore:"customer"`
	Entity   *firestore.DocumentRef `firestore:"entity"`
	Type     string                 `firestore:"type"`
}

type ServiceItem struct {
	Customer           *firestore.DocumentRef `firestore:"customer"`
	Entity             *firestore.DocumentRef `firestore:"entity"`
	Type               string                 `firestore:"type"`
	ServiceName        string                 `firestore:"serviceName"`
	CostCurrentMonth   float64                `firestore:"costCurrentMonth"`
	CostLastMonth      float64                `firestore:"costLastMonth"`
	CostLastThreeMonth float64                `firestore:"costLastThreeMonth"`
}

func ImportCloudServicesToFS(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	bq, err := bigquery.NewClient(ctx, common.ProjectID)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("loadToBQ New Client: %s", err.Error()))
		return
	}
	defer bq.Close()

	query := bq.Query(getCloudServicesQuery())
	currentTime := time.Now().UTC().Format("2006-01-02")
	beforeThreeMonth := time.Now().UTC().AddDate(0, -3, -time.Now().Day()+1).Format("2006-01-") + "01"

	query.Parameters = []bigquery.QueryParameter{
		{Name: "start_date", Value: beforeThreeMonth},
		{Name: "end_date", Value: currentTime},
	}

	itr, err := query.Read(ctx)
	if err != nil {
		l.Errorf("failed to read query: %s", err)
		return
	}

	currentBillingAccount := ""

	var currentService CloudServiceRow

	var customerData CustomerData

	customersServices := make(map[string]map[string]*ServiceItem) //[customerIID][serviceID]
	allCosts := make(map[string]float64)

	for {
		var row CloudServiceRow

		err := itr.Next(&row)
		if err == iterator.Done {
			break
		}

		if currentBillingAccount != row.BillingAccountID {
			currentBillingAccount = row.BillingAccountID

			tmpData, err := fs.Collection("assets").Doc("google-cloud-" + currentBillingAccount).Get(ctx)
			if err != nil {
				l.Errorf("failed to get asset for billing account %s: %s", currentBillingAccount, err)

				customerData.Customer = nil
			} else {
				if err := tmpData.DataTo(&customerData); err != nil {
					l.Errorf("failed to populate customer data for billing account %s: %s", currentBillingAccount, err)
				}
			}
		}

		if customerData.Customer != nil {
			if currentService.ServiceID != row.ServiceID || currentService.BillingAccountID != row.BillingAccountID {
				if currentService.ServiceID != "" && allCosts != nil {
					if customersServices[customerData.Customer.ID] == nil {
						customersServices[customerData.Customer.ID] = make(map[string]*ServiceItem)
					}

					if customersServices[customerData.Customer.ID][currentService.ServiceID] == nil {
						customersServices[customerData.Customer.ID][currentService.ServiceID] = &ServiceItem{
							Customer:           customerData.Customer,
							Entity:             customerData.Entity,
							ServiceName:        currentService.ServiceName,
							CostCurrentMonth:   allCosts["currentMonth"],
							CostLastMonth:      allCosts["lastMonth"],
							CostLastThreeMonth: allCosts["lastThreeMonth"],
							Type:               "google-cloud",
						}
					} else {
						customersServices[customerData.Customer.ID][currentService.ServiceID].CostCurrentMonth += allCosts["currentMonth"]
						customersServices[customerData.Customer.ID][currentService.ServiceID].CostLastMonth += allCosts["lastMonth"]
						customersServices[customerData.Customer.ID][currentService.ServiceID].CostLastThreeMonth += allCosts["lastThreeMonth"]
					}

					allCosts = make(map[string]float64)
				}

				currentService = row
			}

			currentMonth := time.Now().Format("200601")
			lastMonth := time.Now().UTC().AddDate(0, -1, -time.Now().Day()+1).Format("200601")

			if currentMonth == row.InvoiceMonth {
				allCosts["currentMonth"] = row.TotalCost
			}

			if lastMonth == row.InvoiceMonth {
				allCosts["lastMonth"] += row.TotalCost
			}

			if currentMonth != row.InvoiceMonth {
				allCosts["lastThreeMonth"] += row.TotalCost
			}
		}
	}

	batch := fs.Batch()
	batchCounter := 0

	for customerID, servicesMap := range customersServices {
		for serviceID, service := range servicesMap {
			batchCounter++

			batch.Set(fs.Collection("customers").Doc(customerID).Collection("cloudServices").Doc(serviceID), service)

			if batchCounter > 100 {
				batchCounter = 0

				if _, err := batch.Commit(ctx); err != nil {
					log.Println(err.Error())
				}

				batch = nil
				batch = fs.Batch()
			}
		}
	}

	if batchCounter > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			log.Println(err.Error())
		}
	}
}

func getCloudServicesQuery() string {
	return `
	CREATE TEMP FUNCTION TO_INVOICE_MONTH(x DATE) AS (FORMAT_DATE("%Y%m", x));
	SELECT
	  billing_account_id,
	  service.id AS service_id,
	  service.description AS service_description,
	  SUM(cost + IFNULL((SELECT SUM(IF(credit.name LIKE "Committed Usage Discount%" OR credit.name = "Sustained Usage Discount", credit.amount, 0)) FROM UNNEST(credits) AS credit), 0)) AS total_cost,
	  invoice.month AS invoice_month
	FROM
	  billing-explorer.gcp.gcp_billing_export_v1_0033B9_BB2726_9A3CB4
	WHERE
	  DATE(_PARTITIONTIME) BETWEEN @start_date AND @end_date
	  AND invoice.month BETWEEN TO_INVOICE_MONTH(@start_date) AND TO_INVOICE_MONTH(@end_date)
	GROUP BY
	  billing_account_id,
	  service.id,
	  service.description,
	  invoice.month
	HAVING
	  total_cost > 0
	ORDER BY
	  billing_account_id,
	  service.id
	`
}
