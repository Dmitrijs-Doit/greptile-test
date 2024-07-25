package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type generateAssetFilterResult struct {
	filterType string
	value      string
	err        error
}

const (
	CloudProjectFilter    string = "cloudProjectFilter"
	FlexsaveProjectFilter string = "flexsaveProjectFilter"
	BillingAccountFilter  string = "BillingAccountFilter"
	AWSPayerAccountFilter string = "PayerAccountFilter"
)

func (s *AttributionsService) CreateBucketAttribution(ctx context.Context, req *SyncBucketAttributionRequest) (*firestore.DocumentRef, error) {
	attributionRef, err := s.getOrCreateBucketAttributionRef(ctx, req)
	if err != nil {
		return nil, err
	}

	entityNameWithoutQuotes := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(req.Entity.Name, "`", ""), "\"", ""), "'", "")
	bucketNameWithouQuotes := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(req.Bucket.Name, "`", ""), "\"", ""), "'", "")
	attributionName := "[" + req.Entity.PriorityID + "] " + entityNameWithoutQuotes + " - " + bucketNameWithouQuotes

	assets := req.Assets
	if req.Entity.Invoicing.Default != nil && req.Entity.Invoicing.Default.ID == req.Bucket.Ref.ID {
		assets, err = s.getDefaultBucketAssets(ctx, req)
		if err != nil {
			return nil, err
		}
	}

	attributionFilters, err := s.generateInvoiceAttributionFilters(ctx, assets)
	if err != nil {
		return nil, err
	}

	var attributionCloudType string

	if len(assets) > 0 {
		switch assets[0].AssetType {
		case common.Assets.GoogleCloud, common.Assets.GoogleCloudProject:
			attributionCloudType = common.Assets.GoogleCloud
		case common.Assets.AmazonWebServices:
			attributionCloudType = common.Assets.AmazonWebServices
		}
	}

	publicAccessView := collab.PublicAccessView

	if err = s.dal.UpdateAttribution(ctx, attributionRef.ID, []firestore.Update{
		{Path: "name", Value: attributionName},
		{Path: "filters", Value: attributionFilters},
		{Path: "formula", Value: getAttributionFormula(attributionFilters)},
		{Path: "cloud", Value: []string{attributionCloudType}},
		{Path: "type", Value: "managed"},
		{Path: "classification", Value: "invoice"},
		{Path: "hidden", Value: true},
		{Path: "collaborators", Value: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}}},
		{Path: "public", Value: &publicAccessView},
	}); err != nil {
		return nil, err
	}

	return attributionRef, nil
}

func (s *AttributionsService) getOrCreateBucketAttributionRef(ctx context.Context, req *SyncBucketAttributionRequest) (*firestore.DocumentRef, error) {
	attributionRef := req.Bucket.Attribution

	var attributionSnap *firestore.DocumentSnapshot

	var err error
	if attributionRef != nil {
		attributionSnap, err = attributionRef.Get(ctx)
		if err != nil && status.Code(err) != codes.NotFound {
			return nil, err
		}
	}

	if attributionRef == nil || !attributionSnap.Exists() {
		publicAccessView := collab.PublicAccessView
		a, err := s.dal.CreateAttribution(ctx, &attribution.Attribution{
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
		})

		if err != nil {
			return nil, err
		}

		attributionRef = a.Ref

		if err = s.bucketsDal.UpdateBucket(ctx, req.Entity.Snapshot.Ref.ID, req.Bucket.Ref.ID, []firestore.Update{
			{Path: "attribution", Value: attributionRef},
		}); err != nil {
			return nil, err
		}
	}

	return attributionRef, nil
}

func getAttributionFormula(filters []report.BaseConfigFilter) string {
	letters := []string{}
	skipUntilIndex := -1

	for i, filter := range filters {
		if filter.Key == metadata.MetadataFieldKeyAwsPayerAccountID {
			letters = append(letters, fmt.Sprintf("(%c AND %c)", rune(65+i), rune(65+i+1)))
			skipUntilIndex = i + 1
		} else if filter.Key == metadata.MetadataFieldKeyBillingAccountID {
			letters = append(letters, fmt.Sprintf("(%c AND %c AND %c)", rune(65+i), rune(65+i+1), rune(65+i+2)))
			skipUntilIndex = i + 2
		} else if i > skipUntilIndex {
			letters = append(letters, string(rune(65+i)))
		}
	}

	return strings.Join(letters, " OR ")
}

func (s *AttributionsService) getDefaultBucketAssets(ctx context.Context, req *SyncBucketAttributionRequest) ([]*pkg.BaseAsset, error) {
	entityAssets, err := s.assetsDal.GetAssetsInEntity(ctx, req.Entity.Snapshot.Ref)
	if err != nil {
		return nil, err
	}

	if len(entityAssets) == 0 {
		return req.Assets, nil
	}

	assets := req.Assets

	// if bucket isn't empty add all unassigned assets of same type
	if len(assets) != 0 {
		return getNonEmptyDefaultBucketAssets(entityAssets, req, assets)
	}

	return getEmptyDefaultBucketAssets(entityAssets, assets, req)
}

func getEmptyDefaultBucketAssets(entityAssets []*pkg.BaseAsset, assets []*pkg.BaseAsset, req *SyncBucketAttributionRequest) ([]*pkg.BaseAsset, error) {
	var bucketType string

	for _, asset := range entityAssets {
		if asset.Bucket == nil && (asset.AssetType == common.Assets.AmazonWebServices || asset.AssetType == common.Assets.GoogleCloud || asset.AssetType == common.Assets.GoogleCloudProject) {
			// if bucket is empty make sure that all unassigned assets have the same type
			if bucketType == "" {
				bucketType = asset.AssetType
			}

			if !common.IsSameCloudAssetType(asset.AssetType, bucketType) {
				// return req.Assets if unassigned asset have different types
				return req.Assets, nil
			}

			assets = append(assets, asset)
		}
	}

	return assets, nil
}

func getNonEmptyDefaultBucketAssets(entityAssets []*pkg.BaseAsset, req *SyncBucketAttributionRequest, assets []*pkg.BaseAsset) ([]*pkg.BaseAsset, error) {
	for _, asset := range entityAssets {
		if asset.Bucket == nil && common.IsSameCloudAssetType(asset.AssetType, req.Assets[0].AssetType) {
			assets = append(assets, asset)
		}
	}

	return assets, nil
}

func (s *AttributionsService) generateInvoiceAttributionFilters(ctx context.Context, assets []*pkg.BaseAsset) ([]report.BaseConfigFilter, error) {
	numJobs := len(assets)
	jobs := make(chan *pkg.BaseAsset, numJobs)
	results := make(chan generateAssetFilterResult, numJobs)

	wg := sync.WaitGroup{}
	wg.Add(numJobs)

	for i := 0; i < 100; i++ {
		go s.generateAssetFilterValueWorker(ctx, jobs, results, &wg)
	}

	for _, asset := range assets {
		jobs <- asset
	}

	go func() {
		wg.Wait()
		close(results)
		close(jobs)
	}()

	var cloudProjectsFilterValues, flexsaveProjectsFilterValues, cloudBillingAccounts, awsPayerAccounts = []string{}, []string{}, []string{}, []string{}

	for result := range results {
		if result.err != nil {
			return nil, result.err
		}

		switch result.filterType {
		case CloudProjectFilter:
			cloudProjectsFilterValues = append(cloudProjectsFilterValues, result.value)
		case FlexsaveProjectFilter:
			flexsaveProjectsFilterValues = append(flexsaveProjectsFilterValues, result.value)
		case BillingAccountFilter:
			cloudBillingAccounts = append(cloudBillingAccounts, result.value)
		case AWSPayerAccountFilter:
			awsPayerAccounts = append(awsPayerAccounts, result.value)
		}
	}

	filters := []report.BaseConfigFilter{}

	if len(cloudProjectsFilterValues) > 0 {
		filters = append(filters, report.BaseConfigFilter{
			Key:    metadata.MetadataFieldKeyProjectID,
			Type:   metadata.MetadataFieldTypeFixed,
			Values: &cloudProjectsFilterValues,
		})
	}

	if len(flexsaveProjectsFilterValues) > 0 {
		filters = append(filters, report.BaseConfigFilter{
			Key:    metadata.MetadataFieldKeyCmpFlexsaveProject,
			Type:   metadata.MetadataFieldTypeSystemLabel,
			Values: &flexsaveProjectsFilterValues,
		})
	}

	if len(cloudBillingAccounts) > 0 {
		filters = append(filters, report.BaseConfigFilter{
			Key:    metadata.MetadataFieldKeyBillingAccountID,
			Type:   metadata.MetadataFieldTypeFixed,
			Values: &cloudBillingAccounts,
		})
		filters = append(filters, report.BaseConfigFilter{
			Key:       metadata.MetadataFieldKeyProjectID,
			Type:      metadata.MetadataFieldTypeFixed,
			AllowNull: true,
		})

		filters = append(filters, report.BaseConfigFilter{
			Key:     metadata.MetadataFieldKeyServiceDescription,
			Type:    metadata.MetadataFieldTypeFixed,
			Values:  &[]string{"Looker"},
			Inverse: true,
		})
	}

	if len(awsPayerAccounts) > 0 {
		filters = append(filters, report.BaseConfigFilter{
			Key:    metadata.MetadataFieldKeyAwsPayerAccountID,
			Type:   metadata.MetadataFieldTypeSystemLabel,
			Values: &awsPayerAccounts,
		})
		filters = append(filters, report.BaseConfigFilter{
			Key:    metadata.MetadataFieldKeyProjectName,
			Type:   metadata.MetadataFieldTypeFixed,
			Values: &[]string{"Flexsave"},
		})
	}

	result, err := getAttributionFilters(filters)
	if err != nil {
		return nil, err
	}

	return result, err
}

func (s *AttributionsService) generateAssetFilterValueWorker(ctx context.Context, assets <-chan *pkg.BaseAsset, results chan<- generateAssetFilterResult, wg *sync.WaitGroup) {
	for asset := range assets {
		ID := strings.ReplaceAll(asset.ID, asset.AssetType+"-", "")

		if asset.AssetType == common.Assets.GoogleCloud {
			results <- generateAssetFilterResult{filterType: BillingAccountFilter, value: ID, err: nil}

			wg.Done()

			continue
		}

		if strings.HasPrefix(ID, "doitintl-fs") {
			results <- generateAssetFilterResult{filterType: FlexsaveProjectFilter, value: ID, err: nil}

			wg.Done()

			continue
		}

		results <- generateAssetFilterResult{filterType: CloudProjectFilter, value: ID, err: nil}

		if asset.AssetType == common.Assets.AmazonWebServices {
			awsAsset, err := s.assetsDal.GetAWSAsset(ctx, asset.ID)
			if err != nil {
				results <- generateAssetFilterResult{filterType: "", value: "", err: err}
			}

			if awsAsset != nil &&
				awsAsset.Properties != nil &&
				awsAsset.Properties.OrganizationInfo != nil &&
				awsAsset.Properties.OrganizationInfo.PayerAccount != nil &&
				awsAsset.Properties.OrganizationInfo.PayerAccount.AccountID != "" &&
				awsAsset.Properties.OrganizationInfo.PayerAccount.AccountID == awsAsset.Properties.AccountID {
				results <- generateAssetFilterResult{filterType: AWSPayerAccountFilter, value: ID, err: nil}
			}
		}

		wg.Done()
	}
}
