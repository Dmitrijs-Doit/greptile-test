package amazonwebservices

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	_ "embed"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
)

//go:embed get_aws_cloudservices.bqsql
var getAWSCloudServicesQuery string

type CloudServiceRow struct {
	BillingAccountID string   `bigquery:"billing_account_id"`
	Services         []string `bigquery:"services"`
}

func ImportAWSCloudServicesToFS(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	bq, err := bigquery.NewClient(ctx, common.ProjectID)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("loadToBQ New Client: %s", err.Error()))
		return
	}
	defer bq.Close()

	query := bq.Query(getAWSCloudServicesQuery)

	itr, err := query.Read(ctx)
	if err != nil {
		l.Errorf("failed to read query: %s", err)
		return
	}

	batch := fs.Batch()
	batchCounter := 0

	for {
		var row CloudServiceRow

		err := itr.Next(&row)
		if err == iterator.Done {
			break
		}

		customer, err := fs.Collection("customers").Doc(row.BillingAccountID).Get(ctx)
		if err != nil {
			continue
		}

		for _, service := range row.Services {
			serviceDetails := strings.Split(service, ",") // id,name
			batchCounter++

			batch.Set(customer.Ref.Collection("cloudServices").Doc(serviceDetails[0]), map[string]interface{}{
				"serviceName": serviceDetails[1],
				"type":        "amazon-web-services",
			}, firestore.MergeAll)

			if batchCounter > 200 {
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
