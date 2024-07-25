package scripts

import (
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

func checkIfContainsElement(arr []interface{}, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}

	return false
}

func flexsaveCapitalizationFix(ctx *gin.Context) []error {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	customersSnaps, err := fs.Collection("customers").Documents(ctx).GetAll()

	metaCollectionGroups := []string{"reportMetadata", "reportOrgMetadata"}

	for _, collectionGroup := range metaCollectionGroups {
		for _, customerSnap := range customersSnaps {
			querySnap, err := fs.CollectionGroup(collectionGroup).
				Where("customer", "==", customerSnap.Ref).
				Where("type", "in", []string{
					"fixed",
				}).
				Where("key", "==", "cost_type").
				Documents(ctx).GetAll()
			if err != nil {
				fmt.Printf("There was problem getting snaps :(")
			}

			for _, metadataSnap := range querySnap {
				data := metadataSnap.Data()

				_, ok := data["key"]
				if !ok {
					continue
				}

				if data["values"] == nil {
					continue
				}

				valuesSlice := data["values"].([]interface{})

				if !checkIfContainsElement(valuesSlice, "FlexSave") {
					continue
				}

				var newValues []interface{}

				for _, v := range valuesSlice {
					if v != "FlexSave" {
						newValues = append(newValues, v)
					}
				}

				if !checkIfContainsElement(newValues, "Flexsave") {
					newValues = append(newValues, "Flexsave")
				}

				_, err := metadataSnap.Ref.Update(ctx, []firestore.Update{
					{Path: "values", Value: newValues},
				})

				if err == nil {
					fmt.Printf("updated document: %s, %v\n", metadataSnap.Ref.Path, checkIfContainsElement(newValues, "Flexsave"))
				} else {
					fmt.Println(err)
				}
			}
		}
	}

	return nil
}
