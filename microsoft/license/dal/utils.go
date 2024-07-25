package dal

import (
	"log"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/assets"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
	"github.com/doitintl/hello/scheduled-tasks/microsoft/license/domain"
)

const timeLayout = "2006-01-02T15:04:05Z"

func makeAsset(props *microsoft.CreateAssetProps, s *microsoft.SubscriptionWithStatus, item *domain.CatalogItem, customerRef, entityRef *firestore.DocumentRef) (*microsoft.Asset, *assets.AssetSettings) {
	assetType := "office-365"

	var settings *assets.Settings

	if item.Plan == "ANNUAL" {
		var err error

		var endDate, startDate time.Time

		startDate, err = time.Parse(timeLayout, s.Subscription.StartDate)

		if err != nil {
			log.Println(err.Error())

			startDate = time.Now()
		}

		endDate, err = time.Parse(timeLayout, s.Subscription.EndDate)

		if err != nil {
			log.Println(err.Error())

			endDate = startDate.AddDate(1, 0, -1)
		}

		settings = &assets.Settings{
			Plan: &assets.SubscriptionPlan{
				PlanName:         item.Plan,
				IsCommitmentPlan: true,
				CommitmentInterval: &assets.SubscriptionPlanCommitmentInterval{
					StartTime: startDate.UnixNano() / 1000000,
					EndTime:   endDate.UnixNano() / 1000000,
				},
			},
			Payment: item.Payment,
		}
	}

	properties := &microsoft.AssetProperties{
		CustomerDomain: props.LicenseCustomerDomain,
		CustomerID:     props.LicenseCustomerID,
		Subscription:   s.Subscription,
		Reseller:       props.Reseller,
		Settings:       settings,
	}

	asset := &microsoft.Asset{
		BaseAsset: pkg.BaseAsset{
			AssetType: assetType,
			Customer:  customerRef,
			Entity:    entityRef,
			Bucket:    nil,
			Contract:  nil,
		},
		Properties: properties,
	}

	assetSettings := &assets.AssetSettings{
		AssetType: assetType,
		Customer:  customerRef,
		Entity:    entityRef,
		Bucket:    nil,
		Contract:  nil,
		Settings:  settings,
	}

	return asset, assetSettings
}
