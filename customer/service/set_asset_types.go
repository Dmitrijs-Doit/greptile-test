package service

import (
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/presentations/service"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

type customerAssets struct {
	customer *firestore.DocumentRef
	assets   []string
}

func (s *Service) SetCustomerAssetTypes(ctx *gin.Context) {
	fs := s.conn.Firestore(ctx)

	assetTypes := []string{
		common.Assets.GoogleCloud,
		common.Assets.GoogleCloudStandalone,
		common.Assets.AmazonWebServices,
		common.Assets.AmazonWebServicesStandalone,
		common.Assets.Office365,
		common.Assets.GSuite,
		common.Assets.MicrosoftAzure,
	}

	assetDocSnaps, err := fs.Collection("assets").
		Where("type", "in", assetTypes).
		Select("customer", "type").
		Documents(ctx).GetAll()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	batch := fb.NewAutomaticWriteBatch(fs, 500)
	uniqueCustomerMap := make(map[string]*customerAssets)

	for _, assetDocSnap := range assetDocSnaps {
		var ca common.BaseAsset
		if err := assetDocSnap.DataTo(&ca); err != nil {
			ctx.Error(err)
			continue
		}

		if ca.Customer == nil || ca.Customer.ID == domainQuery.CSPCustomerID {
			continue
		}

		if service.IsPresentationCustomer(ca.Customer.ID) {
			continue
		}

		if val, ok := uniqueCustomerMap[ca.Customer.ID]; !ok {
			uniqueCustomerMap[ca.Customer.ID] = &customerAssets{
				customer: ca.Customer,
				assets:   []string{ca.AssetType},
			}
		} else if !slice.Contains(uniqueCustomerMap[ca.Customer.ID].assets, ca.AssetType) {
			val.assets = append(val.assets, ca.AssetType)
		}
	}

	// Override CSP assets
	uniqueCustomerMap[domainQuery.CSPCustomerID] = &customerAssets{
		customer: fs.Collection("customers").Doc(domainQuery.CSPCustomerID),
		assets:   []string{common.Assets.GoogleCloud, common.Assets.AmazonWebServices},
	}

	for _, customerMap := range uniqueCustomerMap {
		batch.Update(customerMap.customer, []firestore.Update{
			{Path: "assets", Value: customerMap.assets},
		})
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		for _, err := range errs {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
}
