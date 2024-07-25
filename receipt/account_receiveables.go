package receipt

import (
	"net/http"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type QueryResult struct {
	EntityID        string `bigquery:"ENTITY_ID"`
	AverageDays     int64  `bigquery:"AVG_AR_DAYS"`
	AverageLateDays int64  `bigquery:"AVG_LATE_AR_DAYS"`
}

func AccountReceiveables(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	bq, err := bigquery.NewClient(ctx, common.ProjectID)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer bq.Close()

	query := "SELECT ENTITY_ID, AVG_AR_DAYS, AVG_LATE_AR_DAYS FROM `me-doit-intl-com.stored_queries.average_account_receivables`"
	queryJob := bq.Query(query)

	iter, err := queryJob.Read(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	batch := fs.Batch()
	batchSize := 0

	for {
		var row QueryResult

		err := iter.Next(&row)
		if err != nil {
			if err == iterator.Done {
				break
			} else {
				ctx.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		}

		entityRef := fs.Collection("entities").Doc(row.EntityID)
		batch.Set(entityRef.Collection("entityMetadata").Doc("account-receivables"), map[string]interface{}{
			"avgDays":     row.AverageDays,
			"avgLateDays": row.AverageLateDays,
			"entity":      entityRef,
		}, firestore.MergeAll)

		batchSize++

		if batchSize >= 50 {
			if _, err := batch.Commit(ctx); err != nil {
				l.Error(err)
			}

			batch = fs.Batch()
			batchSize = 0
		}
	}

	if batchSize > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			l.Error(err)
		}
	}
}
