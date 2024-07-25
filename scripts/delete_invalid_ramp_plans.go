package scripts

import (
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func DeleteInvalidRampPlans(ctx *gin.Context) []error {
	dryRun := ctx.Query("dryRun") == "true"
	l := logger.FromContext(ctx)

	var errors []error

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		errors = append(errors, fmt.Errorf("error creating firestore client"))
		return errors
	}

	defer fs.Close()

	// get all ramp plans that are not associated with an attribution group

	rampPlans := fs.Collection("rampPlans").Documents(ctx)

	for {
		doc, err := rampPlans.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			errors = append(errors, fmt.Errorf("error iterating ramp plans: %v", err))
		}

		if doc.Data()["attributionGroupRef"] != nil {
			continue
		}

		l.Infof("dry run: %v: deleting ramp plan %s", dryRun, doc.Ref.ID)

		if dryRun {
			continue
		}

		_, err = doc.Ref.Delete(ctx)
		if err != nil {
			errors = append(errors, fmt.Errorf("error deleting ramp plan: %v", err))
		}
	}

	return errors
}
