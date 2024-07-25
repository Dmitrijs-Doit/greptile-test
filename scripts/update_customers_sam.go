package scripts

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

/*
This scripts relies on the data exported from 2 google sheets:
	* FSR:			https://docs.google.com/spreadsheets/d/1dczhkGZxWe4x5KguzqYIWfhKGyE-GEQckxrHDK1TdeU/edit#gid=0
	* SAM: 			https://docs.google.com/spreadsheets/d/14w49iCZoV3jdQ6fIjZ_lXPK5kQq53rkSQ0QypeYBqD8/edit#gid=0

* body:
 	* use {am}.index={0-9} for cell index of a given role
 	* use {am}.notificationLevel={1-4} for Support notification level
	{
		"add": bool,
		"domainIndexes":int (omit if customer domain is on index 0)
		"fsr":	{ index: int; notificationLevel: int } (omit if not relevant)
		"sam":	{ index: int; notificationLevel: int } (omit if not relevant)
		"tam":	{ index: int; notificationLevel: int } (omit if not relevant)
	}

* for each customer - update AM value to the one under on sheets above:
	* updates AM record inside AccountTeam (array) with new value OR adds it if not exist
	* sets AccountManagers.doit with new value

* run
	1. export sheets (link above) as CSV (make sure you have the latest version)
	2. name it file.csv
	3. put it under /server/services/scheduled-tasks/scripts
	4. fill in all the sheet indexes you would like to use (use the  convention from above)
	5. run
	6. do the same for the 2nd sheet
*/

type accountManagerUpdate struct {
	Index             int `json:"index"`
	NotificationLevel int `json:"notificationLevel"`
}
type request struct {
	Add    bool                  `json:"add"`
	Domain int                   `json:"domainIndex"`
	FSR    *accountManagerUpdate `json:"fsr"`
	SAM    *accountManagerUpdate `json:"sam"`
	TAM    *accountManagerUpdate `json:"tam"`
}

func UpdateCustomersSAM(ctx *gin.Context) []error {
	errors := []error{}
	success := 0
	fail := 0

	var req request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	batch := doitFirestore.NewBatchProviderWithClient(fs, 50).Provide(ctx)

	file, err := os.Open("./scripts/file.csv")
	if err != nil {
		return []error{err}
	}
	defer file.Close()

	csvReader := csv.NewReader(file)

	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			errors = append(errors, err)
			fail++

			continue
		}

		domain := adjustString(row[req.Domain])
		newFsrEmail := getAmEmail(row, req.FSR)
		newSamEmail := getAmEmail(row, req.SAM)
		newTamEmail := getAmEmail(row, req.TAM)

		if domain == "" || adjustString(domain) == "domain" || adjustString(domain) == "customer" || (newFsrEmail == "" && newSamEmail == "" && newTamEmail == "") {
			continue
		}

		fmt.Printf("\n%s --> ", domain)

		customer, customerRef, err := getCustomer(ctx, fs, domain)
		if err != nil {
			errors = append(errors, err)
			fail++

			continue
		}

		// fsr
		if newFsrEmail != "" && strings.Contains(newFsrEmail, "@") {
			fmt.Printf("FSR %s, ", newFsrEmail)

			err = updateAccountManagerFields(ctx, fs, customer, newFsrEmail, common.AccountManagerRoleFSR, req.FSR.NotificationLevel, req.Add)
			if err != nil {
				errors = append(errors, err)
				fail++

				continue
			}
		}

		// sam
		if newSamEmail != "" && strings.Contains(newSamEmail, "@") {
			fmt.Printf("SAM %s, ", newSamEmail)

			err = updateAccountManagerFields(ctx, fs, customer, newSamEmail, common.AccountManagerRoleSAM, req.SAM.NotificationLevel, req.Add)
			if err != nil {
				errors = append(errors, err)
				fail++

				continue
			}
		}

		// tam
		if newTamEmail != "" && strings.Contains(newTamEmail, "@") {
			fmt.Printf("TAM %s, ", newTamEmail)

			err = updateAccountManagerFields(ctx, fs, customer, newTamEmail, common.AccountManagerRoleTAM, req.TAM.NotificationLevel, req.Add)
			if err != nil {
				errors = append(errors, err)
				fail++

				continue
			}
		}

		if err := batch.Set(ctx, customerRef, map[string]interface{}{
			"accountManagers": customer.AccountManagers,
			"accountTeam":     customer.AccountTeam,
		}, firestore.MergeAll); err != nil {
			errors = append(errors, err)
			fail++

			continue
		}

		success++
	}

	if err := batch.Commit(ctx); err != nil {
		errors = append(errors, err)
	}

	fmt.Printf("updated %d customers successfully\n", success)
	fmt.Printf("failed to update %d customers\n", fail)

	return errors
}

func updateAccountManagerFields(ctx *gin.Context, fs *firestore.Client, customer *common.Customer, newAm string, amType common.AccountManagerRole, supportNotificationLevel int, add bool) error {
	newAccountTeamMember, err := getAccountManagerByType(ctx, fs, newAm, amType, supportNotificationLevel)
	if err != nil {
		return err
	}

	var foundExisting bool

	if !add {
		foundExisting, err = replaceExistingAccountManager(ctx, customer, amType, newAccountTeamMember)
		if err != nil {
			return err
		}
	}

	shouldPush := add || !foundExisting

	if shouldPush { //	if no AM of the given role yet - add it
		customer.AccountTeam = append(customer.AccountTeam, newAccountTeamMember)
	}

	return nil
}

func replaceExistingAccountManager(ctx *gin.Context, customer *common.Customer, amType common.AccountManagerRole, newAccountTeamMember *common.AccountTeamMember) (bool, error) {
	var foundExisting bool

	// update AM on the relevant structure
	for index, accountTeamMember := range customer.AccountTeam {
		docSnap, err := accountTeamMember.Ref.Get(ctx)
		if err != nil {
			return false, err
		}

		var accountManager *common.AccountManager
		if err := docSnap.DataTo(&accountManager); err != nil {
			return false, err
		}

		if accountManager.Role == amType { //	update existing AM of the given role to the new one
			customer.AccountTeam[index] = newAccountTeamMember
			foundExisting = true
			break
		}
	}

	// update AM on the legacy structure
	if customer.AccountManagers["doit"] != nil {
		switch amType {
		case common.AccountManagerRoleFSR:
			customer.AccountManagers["doit"].AccountManager1 = &common.AccountManagerRef{
				Ref:          newAccountTeamMember.Ref,
				Notification: newAccountTeamMember.SupportNotificationLevel,
			}
		case common.AccountManagerRoleSAM:
			customer.AccountManagers["doit"].AccountManager2 = &common.AccountManagerRef{
				Ref:          newAccountTeamMember.Ref,
				Notification: newAccountTeamMember.SupportNotificationLevel,
			}
		}
	}

	return foundExisting, nil
}

func getAccountManagerByType(ctx *gin.Context, fs *firestore.Client, email string, amType common.AccountManagerRole, supportNotificationLevel int) (*common.AccountTeamMember, error) {
	newAmRef, err := getAccountManagerRef(ctx, fs, "email", email, amType)
	if err != nil {
		return nil, err
	}

	if supportNotificationLevel == 0 {
		supportNotificationLevel = 3
	}

	return &common.AccountTeamMember{
		Company:                  common.AccountManagerCompanyDoit,
		Ref:                      newAmRef,
		SupportNotificationLevel: int64(supportNotificationLevel),
	}, nil
}

func getAccountManagerRef(ctx *gin.Context, fs *firestore.Client, valueToSearch, value string, amType common.AccountManagerRole) (*firestore.DocumentRef, error) {
	value = adjustString(value)

	accountManagerRefs, err := fs.Collection("accountManagers").Where("role", "==", amType).Where(valueToSearch, "==", value).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	if len(accountManagerRefs) < 1 {
		err := fmt.Errorf("no account manager found with %s %s", valueToSearch, value)
		return nil, err
	}

	return accountManagerRefs[0].Ref, nil
}

func getCustomer(ctx *gin.Context, fs *firestore.Client, domain string) (*common.Customer, *firestore.DocumentRef, error) {
	customerRefs, err := fs.Collection("customers").Where("domains", "array-contains", domain).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return nil, nil, err
	}

	if len(customerRefs) < 1 {
		err := fmt.Errorf("no customer found for domain %s", domain)
		return nil, nil, err
	}

	var customer *common.Customer
	if err := customerRefs[0].DataTo(&customer); err != nil {
		return nil, nil, err
	}

	return customer, customerRefs[0].Ref, nil
}

func getAmEmail(row []string, amUpdate *accountManagerUpdate) string {
	if amUpdate != nil {
		return adjustString(row[amUpdate.Index])
	}

	return ""
}

func adjustString(str string) string {
	return strings.TrimSpace(strings.ToLower(string(str)))
}
