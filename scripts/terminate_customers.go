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
This scripts relies on the data exported from this google sheet:
	https://docs.google.com/spreadsheets/d/161X9hmheQrYuVqDHf_nUC_rvFJVl92Fn0XTeITUdrmI/edit#gid=1706605200

* for each customer - update "classification" to "terminated"

* run
	1. export sheet (link above) as CSV (make sure you have the latest version)
	2. name it file.csv
	3. put it under /server/services/scheduled-tasks/scripts
	4. run
*/

func TerminateCustomers(ctx *gin.Context) []error {
	errors := []error{}
	success := 0
	fail := 0

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	batch := doitFirestore.NewBatchProviderWithClient(fs, 30).Provide(ctx)

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

		name := row[1]
		domain := strings.ToLower(row[3])

		if domain == "company domain" || domain == "" || name == "" {
			continue
		}

		customer, customerRef, err := getCustomerByDomainOrName(ctx, fs, domain, name)
		if err != nil {
			errors = append(errors, err)
			fail++

			continue
		}

		if err := batch.Set(ctx, customerRef, map[string]interface{}{
			"classification": "terminated",
		}, firestore.MergeAll); err != nil {
			errors = append(errors, err)
			fail++

			continue
		}

		fmt.Printf("updated customer [%s] previous classification: [%s]\n", customer.Name, customer.Classification)

		success++
	}

	if err := batch.Commit(ctx); err != nil {
		errors = append(errors, err)
	}

	fmt.Printf("updated %d customers successfully\n", success)
	fmt.Printf("failed to update %d customers\n", fail)

	return errors
}

func getCustomerByDomainOrName(ctx *gin.Context, fs *firestore.Client, domain, name string) (*common.Customer, *firestore.DocumentRef, error) {
	customerRefs, err := queryDomains(fs, domain).Documents(ctx).GetAll()
	if err != nil {
		return nil, nil, err
	}

	if len(customerRefs) < 1 {
		customerRefs, err = queryPrimaryDomain(fs, domain).Documents(ctx).GetAll()
		if err != nil {
			return nil, nil, err
		}

		if len(customerRefs) < 1 {
			customerRefs, err = queryName(fs, name).Documents(ctx).GetAll()
			if err != nil {
				return nil, nil, err
			}

			if len(customerRefs) < 1 {
				return nil, nil, fmt.Errorf("no customer found for domain [%s]", domain)
			}
		}
	}

	var customer *common.Customer
	if err := customerRefs[0].DataTo(&customer); err != nil {
		return nil, nil, err
	}

	return customer, customerRefs[0].Ref, nil
}

func queryDomains(fs *firestore.Client, domain string) firestore.Query {
	return fs.Collection("customers").Where("domains", "array-contains", domain).Limit(1)
}

func queryPrimaryDomain(fs *firestore.Client, domain string) firestore.Query {
	return fs.Collection("customers").Where("primaryDomain", "==", domain).Limit(1)
}

func queryName(fs *firestore.Client, name string) firestore.Query {
	return fs.Collection("customers").Where("name", "==", name).Limit(1)
}
