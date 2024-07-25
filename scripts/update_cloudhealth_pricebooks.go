package scripts

import (
	"encoding/json"
	"errors"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
)

type UpdateCloudHealthPricebooksSettingsInput struct {
	ProjectID string `json:"projectId"`
}

type FirestoreCloudHealthPricebooksSettings struct {
	Operations []PricebookOption        `json:"operations" firestore:"operations"`
	Products   []PricebookProductOption `json:"products" firestore:"products"`
	Regions    []PricebookOption        `json:"regions" firestore:"regions"`
}

type PricebookProductOption struct {
	ProductName string            `json:"productName" firestore:"productName"`
	UsageTypes  []PricebookOption `json:"usageTypes,omitempty" firestore:"usageTypes,omitempty"`
}

type PricebookOption struct {
	Name string `json:"name" firestore:"name"`
}

// updateCloudHealthPricebooksOptions will generate the cloudhealth-pricebooks doc under "app"
func updateCloudHealthPricebooksOptions(ctx *gin.Context) []error {
	var params UpdateCloudHealthPricebooksSettingsInput

	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	if params.ProjectID == "" {
		err := errors.New("missing project id")
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, params.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	data, err := os.ReadFile("./scripts/data/cloudhealth_pricebooks_options.json")
	if err != nil {
		return []error{err}
	}

	var pricebookSettings FirestoreCloudHealthPricebooksSettings

	if err := json.Unmarshal(data, &pricebookSettings); err != nil {
		return []error{err}
	}

	if _, err := fs.Collection("app").
		Doc("cloudhealth-pricebooks").
		Set(ctx, pricebookSettings); err != nil {
		return []error{err}
	}

	return nil
}
