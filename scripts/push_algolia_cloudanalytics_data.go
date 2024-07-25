package scripts

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"cloud.google.com/go/firestore"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type algoliaParams struct {
	AppID          string         `json:"appId"`
	WriteKey       string         `json:"writeKey"`
	CollectionData CollectionData `json:"collectionData"`
}

type CollectionData struct {
	IndexName      string          `json:"indexName"`
	CollectionName string          `json:"collectionName"`
	Fields         []AlgoliaFields `json:"fields"`
}

type AlgoliaFields struct {
	FieldName    string      `json:"fieldName"`
	FieldPath    string      `json:"fieldPath"`
	FixedValue   interface{} `json:"fixedValue"`
	DefaultValue interface{} `json:"defaultValue"`
	InTags       bool        `json:"inTags"`
}

/*
Body request for each cloud analytics objects at the bottom
When running the script a prompt will appear to confirm the action
Before confirming "server/services/scheduled-tasks/algolia-cloud-analytics-update" will be generated with preview of the data that will be pushed to algolia
*/

func PushDataToAlgolia(ctx *gin.Context) []error {
	var params algoliaParams

	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	f, err := os.Create("algolia-cloud-analytics-update")
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	paramsJSON, _ := json.MarshalIndent(params, "", "  ")
	log.Printf("Request Body: %v", string(paramsJSON))

	config := search.Configuration{
		AppID:        params.AppID,
		APIKey:       params.WriteKey,
		MaxBatchSize: 5000,
	}

	client := search.NewClientWithConfig(config)
	fs, err := firestore.NewClient(ctx, common.ProjectID)

	if err != nil {
		return []error{err}
	}

	defer fs.Close()

	if err != nil {
		return []error{err}
	}

	cdata := params.CollectionData
	docs, err := fs.Collection(cdata.CollectionName).Documents(ctx).GetAll()

	if err != nil {
		log.Printf("error getting documents from collection: %s, error: %v", cdata.CollectionName, err)
		return []error{err}
	}

	index := client.InitIndex(cdata.IndexName)

	if index == nil {
		log.Printf("error getting index: %s", cdata.IndexName)
		return []error{err}
	}

	outChan := make(chan map[string]interface{})
	errChan := make(chan error)
	done := make(chan bool)
	elems := make([]map[string]interface{}, 0)
	wg := &sync.WaitGroup{}

	for _, doc := range docs {
		go processDoc(ctx, doc, cdata, fs, f, outChan, errChan, wg)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	for i := 0; i < len(docs); i++ {
		select {
		case elem := <-outChan:
			elems = append(elems, elem)
		case err := <-errChan:
			log.Printf("error processing document: %v", err)
		case <-done:
			break
		}
	}

	log.Printf("Firestore collection size: %d", len(docs))
	log.Printf("Total number of objects to write to Algolia : %d", len(elems))
	j, _ := json.MarshalIndent(elems, "", "  ")

	_, _ = f.WriteString("Cloud analytics objects: \n" + string(j))

	isConfirmed := confirmAction()
	if !isConfirmed {
		log.Printf("writing to index: %s skipped", cdata.IndexName)
		return nil
	}

	res, err := index.SaveObjects(elems)
	if err != nil {
		log.Printf("error saving objects to index: %s, error: %v", cdata.IndexName, err)
		return []error{err}
	}

	log.Printf("writing to index: %s finished, total: %d", cdata.IndexName, len(res.ObjectIDs()))

	return nil
}

func processDoc(ctx *gin.Context, doc *firestore.DocumentSnapshot, cdata CollectionData, fs *firestore.Client, f *os.File, outChan chan map[string]interface{}, errChan chan error, wg *sync.WaitGroup) {
	wg.Add(1)

	defer func() {
		wg.Done()
	}()

	var tags = []interface{}{"cloudAnalytics"}

	data := doc.Data()
	caObj := make(map[string]interface{})

	caObj["_indexName"] = cdata.IndexName
	caObj["objectID"] = doc.Ref.ID
	caObj["customerId"] = nil

	customerRef, err := doc.DataAt("customer")
	if err != nil {
		errChan <- err
	}

	cRef, ok := customerRef.(*firestore.DocumentRef)

	if ok {
		var customer common.Customer

		customerSnap, err := fs.Collection("customers").Doc(cRef.ID).Get(ctx)
		if err != nil {
			errChan <- err
			return
		}

		if err = customerSnap.DataTo(&customer); err != nil {
			errChan <- err
			return
		}

		caObj["customerId"] = cRef.ID
		caObj["customerName"] = customer.Name
		caObj["customerDomain"] = customer.PrimaryDomain
	} else {
		if data["type"] == "managed" || data["type"] == "custom" {
			log.Printf("managed or custom type does not have a customer assigned doc: %s, error: %v", doc.Ref.ID, err)
			return
		}
	}

	for _, field := range cdata.Fields {
		// Fixed Value
		if field.FixedValue != nil {
			// Value in tags array
			if field.InTags {
				tags = append(tags, field.FixedValue)

				continue
			}

			caObj[field.FieldName] = field.FixedValue

			continue
		}

		// Default Value
		if common.IsNil(data[field.FieldPath]) && !common.IsNil(field.DefaultValue) {
			caObj[field.FieldName] = field.DefaultValue

			continue
		}

		if common.IsNil(data[field.FieldPath]) && common.IsNil(field.DefaultValue) {
			_, _ = f.WriteString(fmt.Sprintf("document: %s has field: %s nil \n", doc.Ref.ID, field.FieldPath))
			caObj[field.FieldName] = nil

			continue
		}

		// Value from firestore
		caObj[field.FieldName] = data[field.FieldPath]
	}

	caObj["_tags"] = tags

	outChan <- caObj
}

func confirmAction() bool {
	var s string

	fmt.Printf("do you want to save objects to Algolias records DB? (y/N): ")
	_, err := fmt.Scanln(&s)

	if err != nil {
		panic(err)
	}

	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	if s == "y" || s == "yes" {
		return true
	}

	return false
}

/*
//Attributions: around 9400 objects on dev
{
    "appId": "",
    "writeKey": "",
    "collectionData": {
        "indexName": "attributions",
        "collectionName": "dashboards/google-cloud-reports/attributions",
        "fields": [
            {
                "fieldName": "name",
                "fieldPath": "name",
                "defaultValue": ""
            },
            {
                "fieldName": "description",
                "fieldPath": "description",
                "defaultValue": ""
            },
            {
                "fieldName": "collaborators",
                "fieldPath": "collaborators"
            },
            {
                "fieldName": "public",
                "fieldPath": "public"
            },
            {
                "fieldName": "type",
                "fieldPath": "type"
            },
            {
                "fieldName": "cloud",
                "fieldPath": "cloud"
            },
            {
                "inTags": true,
                "fixedValue": "hasAccessControl"
            }
        ]
    }
}
// Attribution Groups: around 277 elements on dev
{
    "appId": "",
    "writeKey": "",
     "collectionData": {
        "indexName": "attributionGroups",
        "collectionName": "cloudAnalytics/attribution-groups/cloudAnalyticsAttributionGroups",
        "fields": [
            {
                "fieldName": "name",
                "fieldPath": "name",
                "defaultValue": ""
            },
            {
                "fieldName": "description",
                "fieldPath": "description",
                "defaultValue": ""

            },
            {
                "fieldName": "collaborators",
                "fieldPath": "collaborators"
            },
            {
                "fieldName": "public",
                "fieldPath": "public"
            },
            {
                "fieldName": "type",
                "fieldPath": "type"
            },
            {
                "fieldName": "cloud",
                "fieldPath": "cloud"
            },
            {
                "inTags": true,
                "fixedValue": "hasAccessControl"
            }
        ]
    }
}

// Reports: around 26000 elements on dev
{
    "appId": "",
    "writeKey": "",
    "collectionData": {
        "indexName": "reports",
        "collectionName": "dashboards/google-cloud-reports/savedReports",
         "fields": [
            {
                "fieldName": "name",
                "fieldPath": "name",
                "defaultValue": ""
            },
            {
                "fieldName": "description",
                "fieldPath": "description",
                "defaultValue": ""
            },
            {
                "fieldName": "collaborators",
                "fieldPath": "collaborators"
            },
            {
                "fieldName": "public",
                "fieldPath": "public"
            },
            {
                "fieldName": "type",
                "fieldPath": "type"
            },
            {
                "fieldName": "cloud",
                "fieldPath": "cloud"
            },
            {
                "inTags": true,
                "fixedValue": "hasAccessControl"
            }
        ]
    }
}

//Budgets around 2600 elements on dev
{
    "appId": "",
    "writeKey": "",
    "collectionData": {
        "indexName": "budgets",
        "collectionName": "cloudAnalytics/budgets/cloudAnalyticsBudgets",
         "fields": [
            {
                "fieldName": "name",
                "fieldPath": "name",
                "defaultValue": ""
            },
            {
                "fieldName": "description",
                "fieldPath": "description",
                "defaultValue": ""
            },
            {
                "fieldName": "collaborators",
                "fieldPath": "collaborators"
            },
            {
                "fieldName": "public",
                "fieldPath": "public"
            },
            {
                "inTags": true,
                "fixedValue": "hasAccessControl"
            }
        ]
    }
}

//Metrics around 2025 elements on dev
{
    "appId": "",
    "writeKey": "",
    "collectionData": {
        "indexName": "metrics",
        "collectionName": "cloudAnalytics/metrics/cloudAnalyticsMetrics",
        "fields": [
            {
                "fieldName": "name",
                "fieldPath": "name",
                "defaultValue": ""
            },
            {
                "fieldName": "description",
                "fieldPath": "description",
                "defaultValue": ""
            },
            {
                "fieldName": "type",
                "fieldPath": "type"
            },
            {
                "fieldName": "public",
                "fixedValue": "viewer"
            }
        ]
    }
}

// Alerts around 500 elements on dev
{
    "appId": "",
    "writeKey": "",
    "collectionData": {
        "indexName": "alerts",
        "collectionName": "cloudAnalytics/alerts/cloudAnalyticsAlerts",
        "fields": [
            {
                "fieldName": "name",
                "fieldPath": "name",
                "defaultValue": ""
            },
            {
                "fieldName": "description",
                "fixedValue": ""
            },
            {
                "fieldName": "collaborators",
                "fieldPath": "collaborators"
            },
            {
                "fieldName": "public",
                "fieldPath": "public"
            },
            {
                "inTags": true,
                "fixedValue": "hasAccessControl"
            }
        ]
    }
}
*/
