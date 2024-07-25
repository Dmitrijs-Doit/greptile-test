package scripts

import (
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

const (
	oldDomain = "doit-intl.com"
	newDomain = "doit.com"
)

/*
Following moving to a new domain doit.com from doit-intl.com it's required to duplicate tenants
with a new email and the same tenantId tenants/{project}/emailToTenant
*/
func DuplicateTenantsWithNewDomainEmails(ctx *gin.Context) []error {
	fs, err := firestore.NewClient(ctx, common.ProjectID)

	if err != nil {
		return []error{err}
	}

	defer fs.Close()

	batch := fb.NewAutomaticWriteBatch(fs, 500)
	docSnaps, err := fs.Collection("tenants").Doc(common.ProjectID).Collection("emailToTenant").Documents(ctx).GetAll()

	if err != nil {
		return []error{err}
	}

	tenantsProcessed := 0

	for _, docSnap := range docSnaps {
		data := docSnap.Data()
		tenantID := data["tenantId"]
		currentEmail := docSnap.Ref.ID
		emailParts := strings.Split(currentEmail, "@")
		prefix := emailParts[0]
		emailDomain := emailParts[1]

		if emailDomain == oldDomain {
			newDomainEmail := fmt.Sprintf("%s@%s", prefix, newDomain)
			newDocRef := fs.Collection("tenants").Doc(common.ProjectID).Collection("emailToTenant").Doc(newDomainEmail)
			batch.Set(newDocRef, map[string]interface{}{
				"tenantId": tenantID,
			}, firestore.MergeAll)

			tenantsProcessed++
		}
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs
	}

	log.Printf("Tenants processed: %d out of %d\n", tenantsProcessed, len(docSnaps))

	return nil
}
