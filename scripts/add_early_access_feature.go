package scripts

import (
	"encoding/csv"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

/*
This scripts relies on the data exported from this google sheet:
	https://docs.google.com/spreadsheets/d/1bXDgdR4qKVLK2ArNMK6SC5HJJM5fl7NYaN9J2gdl6iM/edit#gid=824670558
payload:
	{
		"featureName": "Disable Credit Card Fees",
		"project": "hello-dev"
		"csvDomainIndex": 0,
		"path": "absolute path.../file.csv"
	}

* run
	1. export sheet (link above) as CSV
	3. put it under /server/services/scheduled-tasks/scripts (or wherever you want)
	4. run
*/

// AddEarlyAccessFeatureRequest is the payload for the AddEarlyAccessFeature function
type AddEarlyAccessFeatureRequest struct {
	FeatureName    string `json:"featureName"`
	Project        string `json:"project"`
	CsvDomainIndex int    `json:"csvDomainIndex"`
	Path           string `json:"path"`
}

// AddEarlyAccessFeature adds a new early access feature to a list of customers in Firestore based on a CSV file
func AddEarlyAccessFeature(ctx *gin.Context) []error {
	// handle the payload from the request
	var requestBody AddEarlyAccessFeatureRequest
	if err := ctx.BindJSON(&requestBody); err != nil {
		return []error{err}
	}

	rows, err := readCSVFile(requestBody.Path)
	if err != nil {
		return []error{err}
	}

	// Initialize the Firestore client
	fs, err := firestore.NewClient(ctx, requestBody.Project)
	if err != nil {
		return []error{err}
	}

	var domainList []string

	domains := make(map[string]bool)

	for _, row := range rows {
		if len(row) < 1 {
			continue
		}

		domain := row[requestBody.CsvDomainIndex] // the column where the domain is located

		if _, exists := domains[domain]; !exists {
			domains[domain] = true
		}

		if domains[domain] {
			domainList = append(domainList, domain)
		}
	}

	chunks := [][]string{domainList}

	if len(domainList) >= 30 {
		chunks = chunkSlice(domainList, 29)
	}

	var docSnaps []*firestore.DocumentSnapshot

	for _, chunk := range chunks {
		// Collect all the matching customer documents with a single query
		query := fs.Collection("customers").
			Where("primaryDomain", "in", chunk)

		chunkDocSnaps, err := query.Documents(ctx).GetAll()
		if err != nil {
			return []error{err}
		}

		docSnaps = append(docSnaps, chunkDocSnaps...)
	}

	// Loop through each document and add the feature to the customer's earlyAccessFeatures field
	batch := fs.Batch()

	for _, docSnap := range docSnaps {
		var customer common.Customer
		if err := docSnap.DataTo(&customer); err != nil {
			return []error{err}
		}

		domain := customer.PrimaryDomain
		featureName := requestBody.FeatureName

		for _, feature := range customer.EarlyAccessFeatures {
			if feature == featureName {
				delete(domains, domain) // Remove the domain from the map to prevent duplicates
				continue
			}
		}

		if domains[domain] {
			batch.Update(docSnap.Ref, []firestore.Update{
				{Path: "earlyAccessFeatures", Value: firestore.ArrayUnion(featureName)},
			})
			delete(domains, domain) // Remove the domain from the map to prevent duplicates
		}
	}

	// Commit the batch write operation
	if _, err = batch.Commit(ctx); err != nil {
		return []error{err}
	}

	return nil
}

func readCSVFile(path string) ([][]string, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	reader := csv.NewReader(file)

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func chunkSlice(input []string, chunkSize int) [][]string {
	var output [][]string

	for i := 0; i < len(input); i += chunkSize {
		end := i + chunkSize
		if end > len(input) {
			end = len(input)
		}

		output = append(output, input[i:end])
	}

	return output
}
