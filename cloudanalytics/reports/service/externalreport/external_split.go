package externalreport

import (
	"context"
	"fmt"

	"golang.org/x/exp/maps"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainSplit "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
)

func (s *Service) NewExternalSplitToInternal(
	ctx context.Context,
	externalSplits []*domainExternalReport.ExternalSplit,
) ([]domainSplit.Split, []errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	var splits []domainSplit.Split

	attributionGroupIDs := make(map[string]bool)
	attributionsIDs := make(map[string]bool)

	isSplitParsingValid := true

	for _, externalSplit := range externalSplits {
		split, splitValidationErrors := externalSplit.ToInternal()
		if splitValidationErrors != nil {
			validationErrors = append(validationErrors, splitValidationErrors...)
			isSplitParsingValid = false
		} else {
			splits = append(splits, *split)

			attributionGroupID, err := getAttributionGroupID(split.ID)
			if err != nil {
				validationErrors = append(validationErrors, errormsg.ErrorMsg{
					Field:   domainExternalReport.SplitField,
					Message: err.Error(),
				})
				isSplitParsingValid = false
			} else {
				attributionGroupIDs[attributionGroupID] = true
			}

			attributionID, err := getAttributionID(split.Origin)
			if err != nil {
				validationErrors = append(validationErrors, errormsg.ErrorMsg{
					Field:   domainExternalReport.SplitField,
					Message: err.Error(),
				})
				isSplitParsingValid = false
			} else {
				attributionsIDs[attributionID] = true
			}

			for _, target := range split.Targets {
				attributionID, err := getAttributionID(target.ID)
				if err != nil {
					validationErrors = append(validationErrors, errormsg.ErrorMsg{
						Field:   domainExternalReport.SplitField,
						Message: err.Error(),
					})
					isSplitParsingValid = false
				} else {
					attributionsIDs[attributionID] = true
				}
			}
		}
	}

	if isSplitParsingValid {
		validateReqErrors := s.splittingService.ValidateSplitsReq(&splits)
		for _, validateReqError := range validateReqErrors {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainExternalReport.SplitField,
				Message: validateReqError.Error(),
			})
		}

		validateAttributionsErr, err := s.validateAttributions(ctx, attributionsIDs)
		if err != nil {
			return nil, nil, err
		}

		if validateAttributionsErr != nil {
			validationErrors = append(validationErrors, validateAttributionsErr...)
		}

		validateAttributionGroupsErr, err := s.validateAttributionGroups(ctx, attributionGroupIDs)
		if err != nil {
			return nil, nil, err
		}

		if validateAttributionGroupsErr != nil {
			validationErrors = append(validationErrors, validateAttributionGroupsErr...)
		}
	}

	return splits, validationErrors, nil
}

func (s *Service) validateAttributions(
	ctx context.Context,
	attributionsIDs map[string]bool,
) ([]errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	existingAttributions, err := s.attributionService.GetAttributions(ctx, maps.Keys(attributionsIDs))
	if err != nil {
		return nil, err
	}

	existingAttributionsMap := make(map[string]*attribution.Attribution)
	for _, existingAttribution := range existingAttributions {
		existingAttributionsMap[existingAttribution.ID] = existingAttribution
	}

	for attributionID, _ := range attributionsIDs {
		if _, ok := existingAttributionsMap[attributionID]; !ok {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainExternalReport.SplitField,
				Message: fmt.Sprintf("%s: %s", ErrAttributionDoesNotExistMsg, attributionID),
			})
		}
	}

	return validationErrors, nil
}

func (s *Service) validateAttributionGroups(
	ctx context.Context,
	attributionGroupsIDs map[string]bool,
) ([]errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	existingAttributionGroups, err := s.attributionGroupService.GetAttributionGroups(ctx, maps.Keys(attributionGroupsIDs))
	if err != nil {
		return nil, err
	}

	existingAttributionGroupsMap := make(map[string]*attributiongroups.AttributionGroup)
	for _, existingAttributionGroup := range existingAttributionGroups {
		existingAttributionGroupsMap[existingAttributionGroup.ID] = existingAttributionGroup
	}

	for attributionGroupID, _ := range attributionGroupsIDs {
		if _, ok := existingAttributionGroupsMap[attributionGroupID]; !ok {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainExternalReport.SplitField,
				Message: fmt.Sprintf("%s: %s", ErrAttributionGroupDoesNotExistMsg, attributionGroupID),
			})
		}
	}

	return validationErrors, nil
}
