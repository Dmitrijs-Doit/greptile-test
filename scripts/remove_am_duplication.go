package scripts

import (
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

// const (
// 	oldDomain = "doit-intl.com"
// 	newDomain = "doit.com"
// )

func RemoveAccountManagersDuplications(ctx *gin.Context) []error {
	errors := []error{}
	deletions := 0

	var params UpdateAccountManagerReq
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	batch := doitFirestore.NewBatchProviderWithClient(fs, 1).Provide(ctx)

	allAMSnaps, err := fs.Collection("accountManagers").Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	if len(allAMSnaps) < 1 {
		return []error{fmt.Errorf("nothing found")}
	}

	for _, snap := range allAMSnaps {
		var am common.AccountManager
		if err := snap.DataTo(&am); err != nil {
			errors = append(errors, fmt.Errorf("couldnt parse %s", snap.Ref.ID))
		}

		if isOld(am.Email) {
			newEmail := toggleDoitDomain(am.Email)
			newSnaps, err := fs.Collection("accountManagers").Where("email", "==", newEmail).Limit(1).Documents(ctx).GetAll()

			if err != nil {
				return []error{fmt.Errorf("error fetching %s", newEmail)}
			}

			if len(newSnaps) < 1 {
				continue
			}

			if err := batch.Delete(ctx, newSnaps[0].Ref); err != nil {
				errors = append(errors, err)
				continue
			}

			deletions++
		}
	}

	if err := batch.Commit(ctx); err != nil {
		errors = append(errors, err)
	}

	fmt.Printf("deleted %d am with new domain", deletions)

	return errors
}

func toggleDoitDomain(email string) string {
	if isOld(email) {
		return strings.Replace(email, oldDomain, newDomain, -1)
	}

	if isNew(email) {
		return strings.Replace(email, newDomain, oldDomain, -1)
	}

	return email
}

func isOld(email string) bool {
	return strings.Contains(email, oldDomain)
}

func isNew(email string) bool {
	return strings.Contains(email, newDomain)
}
