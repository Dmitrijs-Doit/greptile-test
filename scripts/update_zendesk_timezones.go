package scripts

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

/*
This scripts updates document /integrations/zendesk field "timeZones" on firestore
Due to "ZenDesk timezone to country code spreadsheet":
	https://docs.google.com/spreadsheets/d/1PcRnigyGv2k6D8zFFPJ0U7MwR7XW3NWlBYf8N5n8uKc/edit#gid=1804660033

Should run whenever sheet is updated with new time zone

* run
	1. export sheet (link above) as CSV (make sure you have the latest version)
	2. name it file.csv
	3. put it under /server/services/scheduled-tasks/scripts
	4. run
*/

func UpdateZendeskTimeZones(ctx *gin.Context) []error {
	errors := []error{}
	success := 0
	fail := 0
	timeZones := make(map[string]string)

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

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

		timeZone := row[0]
		countryCode := row[1]

		if timeZone == "time_zone" || timeZone == "" {
			continue
		}

		timeZones[strings.ToLower(timeZone)] = strings.ToLower(countryCode)
		success++
	}

	docRef := fs.Collection("integrations").Doc("zendesk")

	updates := map[string]interface{}{
		"timeZones": timeZones,
	}
	_, err = docRef.Set(ctx, updates, firestore.MergeAll)
	errors = append(errors, err)

	fmt.Printf("updated %d time zone records successfully\n", success)
	fmt.Printf("failed to update %d time zones\n", fail)

	return errors
}
