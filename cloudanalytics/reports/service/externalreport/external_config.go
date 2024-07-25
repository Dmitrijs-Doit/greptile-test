package externalreport

import (
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func (s *Service) LoadExternalConfigFilters(
	externalConfig *domainExternalReport.ExternalConfig,
	filters []*domainReport.ConfigFilter,
) ([]errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	for _, filter := range filters {
		skip := false
		externalConfigFilter, externalConfigFilterValidationErrors := domainExternalReport.NewExternalConfigFilterFromInternal(filter)

		for _, e := range externalConfigFilterValidationErrors {
			if e.Message != domainExternalReport.ErrConfigFilterIsFilterForLimits {
				validationErrors = append(validationErrors, e)
			} else {
				skip = true
			}
		}

		if !skip {
			externalConfig.Filters = append(externalConfig.Filters, externalConfigFilter)
		}

		for _, group := range externalConfig.Groups {
			if group.GetInternalID() == filter.ID {
				groupLoadFilterValidationError, err := s.GroupLoadFilter(group, filter)
				if err != nil {
					return nil, err
				}

				if groupLoadFilterValidationError != nil {
					validationErrors = append(validationErrors, groupLoadFilterValidationError...)
				}
			}
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors, nil
	}

	return nil, nil
}
