package scripts

import (
	"cloud.google.com/go/firestore" //nolint:gci
	"errors"
	"fmt"
	"github.com/doitintl/hello/scheduled-tasks/common" //nolint:gci

	FS "github.com/doitintl/firestore/pkg" //nolint:gci
	"github.com/gin-gonic/gin"             //nolint:gci
)

type UpdateAccountManagerCompanyInput struct {
	AccountManagerID string                   `json:"accountManagerId"`
	Company          FS.AccountManagerCompany `json:"company"`
}

func getAccountManager(ctx *gin.Context, accountManagerID string, fs *firestore.Client) (*common.AccountManager, error) {
	am, err := fs.Collection("accountManagers").Doc(accountManagerID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var accountManager common.AccountManager
	if err := am.DataTo(&accountManager); err != nil {
		return nil, err
	}

	return &accountManager, nil
}

func updateAccountManagerCompany(ctx *gin.Context, amRef *firestore.DocumentRef, data UpdateAccountManagerCompanyInput, company FS.AccountManagerCompany) error {
	_, err := amRef.Update(ctx, []firestore.Update{{FieldPath: []string{"company"}, Value: company}})
	if err != nil {
		fmt.Printf("Error while updating account manager %s company to %s", data.AccountManagerID, data.Company)
		return err
	}

	return nil
}

func updateCustomerAccountManagers(ctx *gin.Context, amRef *firestore.DocumentRef, data UpdateAccountManagerCompanyInput, oldCompany common.AccountManagerCompany, fs *firestore.Client) error {
	accountManagerCustomers := fs.Collection("Customers").Where(fmt.Sprintf("accountManagers.%s.account_manager.ref", oldCompany), "==", amRef)

	amCustomers, err := accountManagerCustomers.Documents(ctx).GetAll()
	if err != nil {
		fmt.Printf("Error while getting customers for account manager %s", data.AccountManagerID)
		return err
	}

	for _, amCustomer := range amCustomers {
		oldCompanyPath := fmt.Sprintf("accountManagers.%s.account_manager", oldCompany)
		updatePath := fmt.Sprintf("accountManagers.%s.account_manager", data.Company)

		var customerData common.Customer
		if err := amCustomer.DataTo(customerData); err != nil {
			fmt.Printf("Notice: Unable to get customer data for customer %s but account manager update can continue.", amCustomer.Ref.ID)
			continue
		}

		// save account_manager info from old path
		companyAccountManagerData := customerData.AccountManagers[string(oldCompany)].AccountManager1

		_, err := amCustomer.Ref.Update(ctx, []firestore.Update{
			{FieldPath: []string{oldCompanyPath}, Value: nil},
			{FieldPath: []string{updatePath}, Value: companyAccountManagerData},
		})
		if err != nil {
			fmt.Printf("Error while updating customer's account managers. Attempted to move value %v from the account_manager property of %s to %s", companyAccountManagerData, oldCompany, data.Company)
			return err
		}

		fmt.Printf("Successfully moved value %v from the account_manager property of %s to %s", companyAccountManagerData, oldCompany, data.Company)
	}

	return nil
}

func ChangeAccountManagerCompany(ctx *gin.Context) []error {
	var data UpdateAccountManagerCompanyInput

	if err := ctx.ShouldBindJSON(&data); err != nil {
		fmt.Println("Error while binding request body json")
		return []error{err}
	}

	if data.AccountManagerID == "" {
		fmt.Println("Missing account manager id. It must be present")

		err := errors.New("missing account manager id")

		return []error{err}
	}

	if data.Company == "" {
		fmt.Println("Missing company. It must be present and must be one of doit, google_cloud_platform, or amazon_web_services")

		err := errors.New("missing company")

		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		fmt.Printf("Error while creating firestore client")
		return []error{err}
	}

	amRef := fs.Collection("accountManagers").Doc(data.AccountManagerID)

	accountManager, err := getAccountManager(ctx, data.AccountManagerID, fs)
	if err != nil {
		fmt.Printf("Error while getting account manager %s", data.AccountManagerID)
		return []error{err}
	}

	oldCompany := accountManager.Company

	err = updateAccountManagerCompany(ctx, amRef, data, data.Company)
	if err != nil {
		return []error{err}
	}

	err = updateCustomerAccountManagers(ctx, amRef, data, oldCompany, fs)
	if err != nil {
		return []error{err}
	}

	return nil
}
