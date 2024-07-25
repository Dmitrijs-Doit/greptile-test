package service

import (
	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func toAttributionAPIItem(attr *attribution.Attribution) *attribution.AttributionAPI {
	return &attribution.AttributionAPI{
		ID:               attr.ID,
		Name:             attr.Name,
		Description:      attr.Description,
		CreateTime:       attr.TimeCreated.UnixMilli(),
		UpdateTime:       attr.TimeModified.UnixMilli(),
		Type:             attr.Type,
		AnomalyDetection: attr.AnomalyDetection,
		Filters:          toAttributionComponentAPI(attr),
		Formula:          attr.Formula,
	}
}

func ToAttributionList(attrList []attribution.Attribution) []customerapi.SortableItem {
	apiAttr := make([]customerapi.SortableItem, len(attrList))

	for i, a := range attrList {
		item := attribution.AttributionListItem{
			ID:          a.ID,
			Name:        a.Name,
			Description: a.Description,
			Type:        a.Type,
			CreateTime:  a.TimeCreated.UnixMilli(),
			UpdateTime:  a.TimeModified.UnixMilli(),
		}

		for _, r := range a.Collaborators {
			if r.Role == collab.CollaboratorRoleOwner {
				item.Owner = r.Email
				break
			}
		}

		apiAttr[i] = item
	}

	return apiAttr
}

func toAttributionComponentAPI(attrs *attribution.Attribution) []attribution.AttributionComponent {
	var componentsResponse []attribution.AttributionComponent
	for _, component := range attrs.Filters {
		componentsResponse = append(componentsResponse, attribution.AttributionComponent{
			Key:       component.Key,
			Type:      attribution.AttributionComponentType(component.Type),
			Values:    component.Values,
			AllowNull: component.AllowNull,
			Inverse:   component.Inverse,
			Regexp:    component.Regexp,
		})
	}

	return componentsResponse
}

func getAttributionFilters(filters []report.BaseConfigFilter) ([]report.BaseConfigFilter, error) {
	var result []report.BaseConfigFilter

	for _, filter := range filters {
		id := metadata.ToInternalID(filter.Type, filter.Key)

		md, key, err := cloudanalytics.ParseID(id)
		if err != nil {
			return nil, err
		}

		f := report.BaseConfigFilter{
			Key:       key,
			Type:      md.Type,
			Values:    filter.Values,
			ID:        id,
			Field:     md.Field,
			Inverse:   filter.Inverse,
			Regexp:    filter.Regexp,
			AllowNull: filter.AllowNull,
		}

		result = append(result, f)
	}

	return result, nil
}
