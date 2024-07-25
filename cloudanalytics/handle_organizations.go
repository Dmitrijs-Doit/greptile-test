package cloudanalytics

import (
	"context"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

// SetOrganizationFilters will add attribution filters on the report query request
// according to the organization reference that was provided
func SetOrganizationFilters(ctx context.Context, orgRef *firestore.DocumentRef) ([]*domainQuery.QueryRequestX, error) {
	rawAttrs, err := getOrgAttributions(ctx, orgRef)
	if err != nil {
		return nil, err
	}

	orgAttributions := processOrgAttributions(rawAttrs)

	return orgAttributions, nil
}

func getOrgAttributions(ctx context.Context, orgRef *firestore.DocumentRef) ([]attribution.Attribution, error) {
	organization, err := common.GetOrganization(ctx, orgRef)
	if err != nil {
		return nil, err
	}

	var attributions []attribution.Attribution

	for _, attr := range organization.Scope {
		attrSnap, err := attr.Get(ctx)
		if err != nil {
			return nil, err
		}

		var attribution attribution.Attribution
		if err := attrSnap.DataTo(&attribution); err != nil {
			return nil, err
		}

		attribution.ID = string(metadata.MetadataFieldTypeAttribution) + ":" + attrSnap.Ref.ID
		attributions = append(attributions, attribution)
	}

	return attributions, nil
}

func processOrgAttributions(rawAttributions []attribution.Attribution) []*domainQuery.QueryRequestX {
	attributions := make([]*domainQuery.QueryRequestX, 0, len(rawAttributions))

	for _, attr := range rawAttributions {
		attribution := &domainQuery.QueryRequestX{
			ID:              attr.ID,
			Type:            metadata.MetadataFieldTypeAttribution,
			Key:             attr.Name,
			IncludeInFilter: true,
			Formula:         attr.Formula,
			Composite:       make([]*domainQuery.QueryRequestX, 0, len(attr.Filters)),
		}

		for _, filter := range attr.Filters {
			composite := &domainQuery.QueryRequestX{
				Type:      filter.Type,
				ID:        filter.ID,
				Field:     filter.Field,
				Key:       filter.Key,
				AllowNull: filter.AllowNull,
				Inverse:   filter.Inverse,
				Regexp:    filter.Regexp,
				Values:    filter.Values,
			}

			attribution.Composite = append(attribution.Composite, composite)
		}

		attributions = append(attributions, attribution)
	}

	return attributions
}

func addOrgsToQuery(ctx context.Context, bq *bigquery.Client, OrgFP *domain.AttrFiltersParams, org *firestore.DocumentRef, filterData *filterData, rowsCols *rowData, filters *[]string) ([]string, error) {
	q := query.NewQuery(bq)

	orgAttrs, err := SetOrganizationFilters(ctx, org)
	if err != nil {
		return nil, err
	}

	if err := q.HandleAttributions(ctx, OrgFP, orgAttrs); err != nil {
		return nil, err
	}

	rowsCols.attrConditions = OrgFP.AttrConditions
	filterData.queryParams = append(filterData.queryParams, OrgFP.QueryParams...)

	// organization attributions are managed separately to create an "AND" relationship between them
	// and the other attributions that are used in a report/query
	if len(OrgFP.CompositeFilters) > 0 {
		*filters = append(*filters, handleCompositeFilters(OrgFP.CompositeFilters, filterData))
	}

	return OrgFP.CompositeFilters, nil
}
