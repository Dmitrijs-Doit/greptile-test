package scripts

import (
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

func AddPerksTimestamps(ctx *gin.Context) []error {
	fs, err := firestore.NewClient(ctx, common.ProjectID)

	if err != nil {
		return []error{err}
	}

	defer fs.Close()

	batch := fb.NewAutomaticWriteBatch(fs, 500)
	docSnaps, err := fs.Collection("perks").Documents(ctx).GetAll()

	if err != nil {
		return []error{err}
	}

	timeNow := time.Now().UTC()
	docsProcessed := 0

	for _, docSnap := range docSnaps {
		fields, err := docSnap.DataAt("fields")
		if err != nil {
			log.Printf("An error has occurred: %s", err)
			continue
		}

		fieldsMap, ok := fields.(map[string]interface{})
		if !ok {
			log.Printf("An error has occurred: %s", err)
			continue
		}

		active, ok := fieldsMap["active"].(bool)
		if !ok {
			log.Printf("An error has occurred: %s", err)
			continue
		}

		var timePublished *time.Time

		if active {
			timePublished = &timeNow
		}

		batch.Update(docSnap.Ref, []firestore.Update{
			{Path: "timeCreated", Value: timeNow},
			{Path: "timeModified", Value: timeNow},
			{Path: "timePublished", Value: timePublished},
		})

		docsProcessed++
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs
	}

	log.Printf("Timestamps were added to %d out of %d perks.\n", docsProcessed, len(docSnaps))

	return nil
}
