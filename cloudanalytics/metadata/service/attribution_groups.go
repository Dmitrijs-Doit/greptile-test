package service

import (
	"context"
	"fmt"

	agDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

var (
	unallocated = "Unallocated"
)

func (s *MetadataService) AttributionGroupsMetadata(ctx context.Context, customerID, email string) ([]*domain.OrgMetadataModel, error) {
	customer, err := s.customerDal.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	customerOrPresentationModeCustomer, err := s.customerDal.GetCustomerOrPresentationModeCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	attributionGroups, err := s.attributionGroupsDAL.List(ctx, customerOrPresentationModeCustomer.Snapshot.Ref, email)
	if err != nil {
		return nil, err
	}

	if customer.PresentationMode != nil && customer.PresentationMode.Enabled {
		genuineCustomAttrGroups, err := s.attributionGroupsDAL.GetByType(ctx, customer.Snapshot.Ref, domainAttributions.ObjectTypeCustom)
		if err != nil {
			return nil, err
		}

		for _, attrGroup := range genuineCustomAttrGroups {
			attributionGroups = append(attributionGroups, *attrGroup)
		}

	}

	attributionGroups = filterAttributionGroupsByCloud(attributionGroups, customerOrPresentationModeCustomer.Assets)

	attrGroupsMetadata := make([]*domain.OrgMetadataModel, 0, len(attributionGroups))

	for _, attrGroup := range attributionGroups {
		attrGroupMetadata := convertToOrgMetadataModel(&attrGroup)
		attrGroupsMetadata = append(attrGroupsMetadata, attrGroupMetadata)
	}

	return attrGroupsMetadata, nil
}

func convertToOrgMetadataModel(attrGroup *agDomain.AttributionGroup) *domain.OrgMetadataModel {
	attrGroupMetadata := &domain.OrgMetadataModel{
		Key:                 attrGroup.ID,
		Label:               attrGroup.Name,
		Timestamp:           attrGroup.TimeModified,
		Type:                metadata.MetadataFieldTypeAttributionGroup,
		Values:              make([]string, 0, len(attrGroup.Attributions)),
		DisableRegexpFilter: true,
		ObjectType:          attrGroup.Type,
	}

	if attrGroup.NullFallback != nil {
		attrGroupMetadata.NullFallback = attrGroup.NullFallback
	} else {
		attrGroupMetadata.NullFallback = &unallocated
	}

	for _, docRef := range attrGroup.Attributions {
		attrGroupMetadata.Values = append(attrGroupMetadata.Values, docRef.ID)
	}

	attrGroupMetadata.ID = fmt.Sprintf("%s:%s", attrGroupMetadata.Type, attrGroupMetadata.Key)
	attrGroupMetadata.Plural = fmt.Sprintf("%s values", attrGroupMetadata.Label)

	return attrGroupMetadata
}

// filterAttributionGroupsByCloud filters out preset attribution groups that are not relevant
// to the customer based on the customer assets (cloud types) information.
// All custom attribution groups are returned.
func filterAttributionGroupsByCloud(attributionGroups []agDomain.AttributionGroup, customerAssets []string) []agDomain.AttributionGroup {
	// Return all attribution groups if there is no customer assets information
	if len(customerAssets) == 0 {
		return attributionGroups
	}

	filteredAttributionGroups := make([]agDomain.AttributionGroup, 0)

	for _, ag := range attributionGroups {
		if ag.Type != domainAttributions.ObjectTypePreset || len(ag.Cloud) == 0 || slice.ContainsAny(ag.Cloud, customerAssets) {
			filteredAttributionGroups = append(filteredAttributionGroups, ag)
		}
	}

	return filteredAttributionGroups
}
