package scripts

import (
	"encoding/csv"
	"io"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func AddGCPCustomerSegment(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	tags := []string{"enterprise", "corporate"}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	for _, tag := range tags {
		errs := processCSV(ctx, fs, tag)
		if len(errs) > 0 {
			return errs
		}

		l.Infof("successfully processed GCP Enterprise_Corporate tags - %s.csv", tag)
	}

	return nil
}

func processCSV(ctx *gin.Context, fs *firestore.Client, tag string) []error {
	l := logger.FromContext(ctx)

	f, err := os.Open("./GCP Enterprise_Corporate tags - " + tag + ".csv")
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}
	defer f.Close()

	batch := fb.NewAutomaticWriteBatch(fs, 100)
	csvReader := csv.NewReader(f)

	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
			return []error{err}
		}

		l.Info(row)
		name := row[0]

		_, customerRef, err := getCustomer(ctx, fs, name)
		if err != nil {
			l.Info(err)
			continue
		}

		batch.Update(customerRef, []firestore.Update{
			{
				Path:  "gcpCustomerSegment",
				Value: tag,
			},
		})
	}

	errs := batch.Commit(ctx)

	if len(errs) > 0 {
		return errs
	}

	return nil
}
