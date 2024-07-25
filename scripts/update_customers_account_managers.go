package scripts

import (
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type UpdateAccountManagerReq struct {
	OldAccountManagerEmail string `json:"oldAccountManagerEmail"`
	NewAccountManagerEmail string `json:"newAccountManagerEmail"`
}

/*
Replace all instances of the old account manager ref with the new account manager ref
In the accountTeam and accountManagers fields of all customers under the old account manager.

body:

	{
		oldAccountManagerEmail: "old@doit-intl.com"
		newAccountManagerEmail: "new@doit-intl.com"
	}
*/
func UpdateCustomersAM(ctx *gin.Context) []error {
	errors := []error{}

	var params UpdateAccountManagerReq
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	batch := doitFirestore.NewBatchProviderWithClient(fs, 300).Provide(ctx)

	// get all customers that need to be updated with the new AM
	oldAMSnap, err := fs.Collection("accountManagers").Where("email", "==", params.OldAccountManagerEmail).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	if len(oldAMSnap) < 1 {
		return []error{fmt.Errorf("cannot find account manager with email %s", params.OldAccountManagerEmail)}
	}

	newAMSnap, err := fs.Collection("accountManagers").Where("email", "==", params.NewAccountManagerEmail).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	if len(newAMSnap) < 1 {
		return []error{fmt.Errorf("cannot find account manager with email %s", params.NewAccountManagerEmail)}
	}

	var accountTeamMemberWithNotifications []common.AccountTeamMember
	for i := 0; i < 4; i++ {
		accountTeamMemberWithNotifications = append(accountTeamMemberWithNotifications, common.AccountTeamMember{
			Company:                  common.AccountManagerCompanyDoit,
			Ref:                      oldAMSnap[0].Ref,
			SupportNotificationLevel: int64(i),
		})
	}

	customersToUpdateSnaps, err := fs.Collection("customers").Where("accountTeam", "array-contains-any", accountTeamMemberWithNotifications).Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	for _, customerSnap := range customersToUpdateSnaps {
		var customer *common.Customer
		if err := customerSnap.DataTo(&customer); err != nil {
			return []error{err}
		}

		accountTeam, accountManagers, err := updateAccountManager(customer, oldAMSnap[0].Ref, newAMSnap[0].Ref)
		if err != nil {
			return []error{err}
		}

		if err := batch.Set(ctx, customerSnap.Ref, map[string]interface{}{
			"accountManagers": accountManagers,
			"accountTeam":     accountTeam,
		}, firestore.MergeAll); err != nil {
			errors = append(errors, err)
			continue
		}
	}

	if err := batch.Commit(ctx); err != nil {
		errors = append(errors, err)
	}

	return errors
}

func updateAccountManager(customer *common.Customer, oldAM *firestore.DocumentRef, newAM *firestore.DocumentRef) ([]*common.AccountTeamMember, map[string]*common.PlatformAccountManager, error) {
	for index, accountTeamMember := range customer.AccountTeam {
		if accountTeamMember.Ref.ID == oldAM.ID {
			customer.AccountTeam[index].Ref = newAM
		}
	}

	if customer.AccountManagers["doit"].AccountManager1 != nil &&
		customer.AccountManagers["doit"].AccountManager1.Ref != nil &&
		customer.AccountManagers["doit"].AccountManager1.Ref.ID == oldAM.ID {
		customer.AccountManagers["doit"].AccountManager1.Ref = newAM
	}

	if customer.AccountManagers["doit"].AccountManager2 != nil &&
		customer.AccountManagers["doit"].AccountManager2.Ref != nil &&
		customer.AccountManagers["doit"].AccountManager2.Ref.ID == oldAM.ID {
		customer.AccountManagers["doit"].AccountManager2.Ref = newAM
	}

	return customer.AccountTeam, customer.AccountManagers, nil
}
