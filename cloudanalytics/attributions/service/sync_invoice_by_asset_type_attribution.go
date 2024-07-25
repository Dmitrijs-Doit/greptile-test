package service

import (
	"context"
	"strings"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func (s *AttributionsService) CreateAttributionsForInvoiceAssetTypes(ctx context.Context, req SyncInvoiceByAssetTypeAttributionRequest) ([]*firestore.DocumentRef, error) {
	attributions, err := s.getAttributionsInGroup(ctx, req.AttributionGroup)
	if err != nil {
		return nil, err
	}

	gcpAttribution, awsAttribution := getExistingAttributionsForEntity(attributions, req.Entity)

	if gcpAttribution == nil {
		gcpAttribution, err = s.createAttributionForType(ctx, req, common.Assets.GoogleCloud)

		if err != nil {
			return nil, err
		}
	}

	if awsAttribution == nil {
		awsAttribution, err = s.createAttributionForType(ctx, req, common.Assets.AmazonWebServices)
		if err != nil {
			return nil, err
		}
	}

	entityAssets, err := s.assetsDal.GetAssetsInEntity(ctx, req.Entity.Snapshot.Ref)
	if err != nil {
		return nil, err
	}

	gcpAssets, awsAssets := getGCPandAWSassets(entityAssets)

	err = s.updateAttribution(ctx, gcpAssets, gcpAttribution, common.Assets.GoogleCloud)
	if err != nil {
		return nil, err
	}

	err = s.updateAttribution(ctx, awsAssets, awsAttribution, common.Assets.AmazonWebServices)
	if err != nil {
		return nil, err
	}

	return []*firestore.DocumentRef{
		gcpAttribution.Ref,
		awsAttribution.Ref,
	}, nil
}

func getExistingAttributionsForEntity(attributions []*attribution.Attribution, entity *common.Entity) (*attribution.Attribution, *attribution.Attribution) {
	var gcpAttribution *attribution.Attribution

	var awsAttribution *attribution.Attribution

	entityNameWithoutQuotes := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(entity.Name, "`", ""), "\"", ""), "'", "")

	for _, attribution := range attributions {
		if attribution.Name == "["+entity.PriorityID+"] "+entityNameWithoutQuotes+" - Google Cloud" {
			gcpAttribution = attribution
			continue
		}

		if attribution.Name == "["+entity.PriorityID+"] "+entityNameWithoutQuotes+" - Amazon Web Services" {
			awsAttribution = attribution
			continue
		}
	}

	return gcpAttribution, awsAttribution
}

func (s *AttributionsService) updateAttribution(ctx context.Context, assets []*pkg.BaseAsset, attribution *attribution.Attribution, attributionCloud string) error {
	filters, err := s.generateInvoiceAttributionFilters(ctx, assets)
	if err != nil {
		return err
	}

	publicAccessView := collab.PublicAccessView
	err = s.dal.UpdateAttribution(ctx, attribution.Ref.ID, []firestore.Update{
		{Path: "filters", Value: filters},
		{Path: "formula", Value: getAttributionFormula(filters)},
		{Path: "type", Value: "managed"},
		{Path: "classification", Value: "invoice"},
		{Path: "hidden", Value: true},
		{Path: "collaborators", Value: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}}},
		{Path: "public", Value: &publicAccessView},
		{Path: "cloud", Value: []string{attributionCloud}},
	})

	if err != nil {
		return err
	}

	return nil
}

func getGCPandAWSassets(entityAssets []*pkg.BaseAsset) ([]*pkg.BaseAsset, []*pkg.BaseAsset) {
	var gcpAssets, awsAssets []*pkg.BaseAsset

	for _, asset := range entityAssets {
		if asset.AssetType == common.Assets.GoogleCloud || asset.AssetType == common.Assets.GoogleCloudProject {
			gcpAssets = append(gcpAssets, asset)
			continue
		}

		if asset.AssetType == common.Assets.AmazonWebServices {
			awsAssets = append(awsAssets, asset)
			continue
		}
	}

	return gcpAssets, awsAssets
}

func (s *AttributionsService) getAttributionsInGroup(ctx context.Context, attributionGroup *attributiongroups.AttributionGroup) ([]*attribution.Attribution, error) {
	if len(attributionGroup.Attributions) == 0 {
		return nil, nil
	}

	attributions, err := s.dal.GetAttributions(ctx, attributionGroup.Attributions)
	if err != nil {
		return nil, err
	}

	return attributions, nil
}

func (s *AttributionsService) createAttributionForType(ctx context.Context, req SyncInvoiceByAssetTypeAttributionRequest, assetsType string) (*attribution.Attribution, error) {
	publicAccessView := collab.PublicAccessView
	entityNameWithoutQuotes := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(req.Entity.Name, "`", ""), "\"", ""), "'", "")

	attribution, err := s.dal.CreateAttribution(ctx, &attribution.Attribution{
		Type:           "managed",
		Classification: "invoice",
		Hidden:         true,
		Access: collab.Access{
			Collaborators: []collab.Collaborator{
				{Email: "doit.com", Role: collab.CollaboratorRoleOwner},
			},
			Public: &publicAccessView,
		},
		Customer: req.Customer.Snapshot.Ref,
		Name:     "[" + req.Entity.PriorityID + "] " + entityNameWithoutQuotes + " - " + common.FormatAssetType(assetsType),
		Cloud:    []string{assetsType},
	})

	if err != nil {
		return nil, err
	}

	return attribution, nil
}
