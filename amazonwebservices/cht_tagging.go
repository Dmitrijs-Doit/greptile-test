package amazonwebservices

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudhealth"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type (
	// Tag is a cloudhealth tag
	Tag struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	// TagGroup is an array cloudhealth tags
	TagGroup struct {
		AssetType string  `json:"asset_type"`
		Ids       []int64 `json:"ids"`
		Tags      []Tag   `json:"tags"`
	}

	// ChTags is an array of cloudhealth tag groups
	ChTags struct {
		TagGroups []*TagGroup `json:"tag_groups"`
	}
)

func TagCustomers(ctx *gin.Context) {
	fs := common.GetFirestoreClient(ctx)

	awsAccounts := make(map[int64]*cloudhealth.AwsAccount)
	if err := cloudhealth.ListAccounts(1, awsAccounts, nil); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	t := &ChTags{
		TagGroups: make([]*TagGroup, 0),
	}

	customers := make(map[string]*common.Customer)

	for _, awsAccount := range awsAccounts {
		if awsAccount.OwnerID == "" {
			continue
		}

		docID := fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, awsAccount.OwnerID)

		asset, err := fs.Collection("assets").Doc(docID).Get(ctx)
		if err != nil {
			log.Println(err)
			continue
		}

		var assetDoc Asset

		var customerDoc common.Customer

		err = asset.DataTo(&assetDoc)
		if err != nil {
			log.Println(err)
			continue
		}

		customerRef := assetDoc.Customer
		if v, prs := customers[customerRef.ID]; prs {
			customerDoc = *v
		} else {
			customer, err := assetDoc.Customer.Get(ctx)
			if err != nil {
				log.Println(err)
				continue
			}

			err = customer.DataTo(&customerDoc)
			if err != nil {
				log.Println(err)
				continue
			}

			customers[customerRef.ID] = &customerDoc
		}

		customerClassification := common.CustomerClassificationBusiness
		if len(customerDoc.Classification) > 0 {
			customerClassification = customerDoc.Classification
		}

		t.TagGroups = append(t.TagGroups, &TagGroup{
			AssetType: "AwsAccount",
			Ids:       []int64{awsAccount.ID},
			Tags: []Tag{
				{Key: "cmp_type", Value: string(customerClassification)},
				{Key: "cmp_domain", Value: customerDoc.PrimaryDomain},
			},
		})

		if len(t.TagGroups) >= 50 {
			if err := doTag(t); err != nil {
				log.Println(err)
			}

			t = &ChTags{
				TagGroups: make([]*TagGroup, 0),
			}
		}
	}

	if len(t.TagGroups) > 0 {
		if err := doTag(t); err != nil {
			log.Println(err)
		}
	}
}

func doTag(t *ChTags) error {
	reqBody, err := json.Marshal(t)
	if err != nil {
		return err
	}

	path := "/v1/custom_tags"
	if _, err := cloudhealth.Client.Post(path, nil, []byte(reqBody)); err != nil {
		return err
	}

	return nil
}
