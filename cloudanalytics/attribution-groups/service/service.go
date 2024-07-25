package service

import (
	"context"
	"fmt"
	"regexp"

	"github.com/doitintl/customerapi"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	bucketsDal "github.com/doitintl/hello/scheduled-tasks/buckets/dal"
	bucketsService "github.com/doitintl/hello/scheduled-tasks/buckets/service"
	bucketsServiceIface "github.com/doitintl/hello/scheduled-tasks/buckets/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal"
	attributionGroupsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionGroupTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service/attributiongrouptier"
	attributionGroupIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service/attributiongrouptier/iface"
	attributionsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal/iface"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	attributionsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/attributiontier"
	attributionsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/caownerchecker/service"
	caownercheckerIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/caownerchecker/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	reportDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	reportIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/iface"
	domainResource "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/resource/domain"
	contractDal "github.com/doitintl/hello/scheduled-tasks/contract/dal"
	contractDalIface "github.com/doitintl/hello/scheduled-tasks/contract/dal/iface"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	entityDal "github.com/doitintl/hello/scheduled-tasks/entity/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tier "github.com/doitintl/tiers/service"
)

type AttributionGroupsService struct {
	loggerProvider logger.Provider
	*connection.Connection
	attributionsDAL             iface.Attributions
	attributionGroupsDAL        attributionGroupsDal.AttributionGroups
	collab                      collab.Icollab
	customersDAL                customerDal.Customers
	reportDAL                   reportIface.Reports
	employeeService             doitemployees.ServiceInterface
	caOwnerChecker              caownercheckerIface.CheckCAOwnerInterface
	entityDal                   entityDal.Entites
	bucketsDal                  bucketsDal.Buckets
	attributionsService         attributionsIface.AttributionsIface
	assetsDal                   assetsDal.Assets
	bucketsService              bucketsServiceIface.BucketsIface
	contractsDAL                contractDalIface.ContractFirestore
	attributionGroupTierService attributionGroupIface.AttributionGroupTierService
}

const (
	attributionGroupsValidNameRegex = `^[0-9a-zA-Z-_.,:()\[\]\s%]+$`
)

func NewAttributionGroupsService(ctx context.Context, log logger.Provider, conn *connection.Connection) *AttributionGroupsService {
	attributionGroupDAL := dal.NewAttributionGroupsFirestoreWithClient(conn.Firestore)

	attributionDAL := attributionsDal.NewAttributionsFirestoreWithClient(conn.Firestore)

	tierService := tier.NewTiersService(conn.Firestore)

	doitEmployeesService := doitemployees.NewService(conn)

	attributionTierService := attributiontier.NewAttributionTierService(
		log,
		attributionDAL,
		tierService,
		doitEmployeesService,
	)

	attributionGroupTierService := attributionGroupTier.NewAttributionGroupTierService(
		log,
		attributionGroupDAL,
		tierService,
		attributionTierService,
		doitEmployeesService,
	)

	return &AttributionGroupsService{
		log,
		conn,
		attributionDAL,
		attributionGroupDAL,
		&collab.Collab{},
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		reportDal.NewReportsFirestoreWithClient(conn.Firestore),
		doitEmployeesService,
		service.NewCAOwnerChecker(conn),
		entityDal.NewEntitiesFirestoreWithClient(conn.Firestore),
		bucketsDal.NewBucketsFirestoreWithClient(conn.Firestore),
		attributionsService.NewAttributionsService(ctx, log, conn),
		assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
		bucketsService.NewBucketsService(log, conn),
		contractDal.NewContractFirestoreWithClient(conn.Firestore),
		attributionGroupTierService,
	}
}

func (s *AttributionGroupsService) ShareAttributionGroup(ctx context.Context, newCollabs []collab.Collaborator, public *collab.PublicAccess, attributionGroupID, userID, requesterEmail string) error {
	isCAOwner, err := s.caOwnerChecker.CheckCAOwner(ctx, s.employeeService, userID, requesterEmail)
	if err != nil {
		return err
	}

	attributionGroup, err := s.attributionGroupsDAL.Get(ctx, attributionGroupID)
	if err != nil {
		return err
	}

	if err := s.collab.ShareAnalyticsResource(ctx, attributionGroup.Access.Collaborators, newCollabs, public, attributionGroupID, requesterEmail, s.attributionGroupsDAL, isCAOwner); err != nil {
		return err
	}

	return nil
}

func (s *AttributionGroupsService) CreateAttributionGroup(
	ctx context.Context,
	customerID string,
	requesterEmail string,
	attributionGroupRequest *attributiongroups.AttributionGroupRequest,
) (string, error) {
	err := s.validateAttributionGroupName(ctx, customerID, attributionGroupRequest.Name)
	if err != nil {
		return "", err
	}

	attributionGroup := attributionGroupRequest.ToAttributionGroup()
	attributionGroup.Type = domainAttributions.ObjectTypeCustom

	publicAccessView := collab.PublicAccessView
	attributionGroup.Public = &publicAccessView
	attributionGroup.Customer = s.customersDAL.GetRef(ctx, customerID)
	attributionGroup.Collaborators = []collab.Collaborator{
		{
			Email: requesterEmail,
			Role:  collab.CollaboratorRoleOwner,
		},
	}

	attributionGroup.Attributions, err = s.validateAttributions(ctx, attributionGroupRequest.Attributions, customerID)
	if err != nil {
		return "", err
	}

	attributionGroup.NullFallback = attributionGroupRequest.NullFallback

	return s.attributionGroupsDAL.Create(ctx, &attributionGroup)
}

// UpdateAttributionGroup updates an attribution group with the update data provided in the request
// This method only updates the `name`, `description` and `attributions` of the group
func (s *AttributionGroupsService) UpdateAttributionGroup(
	ctx context.Context,
	customerID string,
	attributionGroupID string,
	requesterEmail string,
	attributionGroupUpdateRequest *attributiongroups.AttributionGroupUpdateRequest,
) error {
	attributionGroup, err := s.attributionGroupsDAL.Get(ctx, attributionGroupID)
	if err != nil {
		return err
	}

	if attributionGroup.Type == domainAttributions.ObjectTypeManaged || attributionGroup.Type == domainAttributions.ObjectTypePreset {
		return attributiongroups.ErrForbidden
	}

	if customerID != attributionGroup.Customer.ID {
		return attributiongroups.ErrForbidden
	}

	if !attributionGroup.Access.CanEdit(requesterEmail) {
		return attributiongroups.ErrForbidden
	}

	if attributionGroupUpdateRequest.Name != "" {
		if attributionGroup.Name != attributionGroupUpdateRequest.Name {
			err = s.validateAttributionGroupName(ctx, customerID, attributionGroupUpdateRequest.Name)
			if err != nil {
				return err
			}
		}

		attributionGroup.Name = attributionGroupUpdateRequest.Name
	}

	if attributionGroupUpdateRequest.Description != "" {
		attributionGroup.Description = attributionGroupUpdateRequest.Description
	}

	if len(attributionGroupUpdateRequest.Attributions) > 0 {
		attributionGroup.Attributions, err = s.validateAttributions(ctx, attributionGroupUpdateRequest.Attributions, customerID)
		if err != nil {
			return err
		}
	}

	attributionGroup.NullFallback = attributionGroupUpdateRequest.NullFallback

	return s.attributionGroupsDAL.Update(ctx, attributionGroupID, attributionGroup)
}

func (s *AttributionGroupsService) DeleteAttributionGroup(
	ctx context.Context,
	customerID string,
	requesterEmail string,
	attributionGroupID string,
) ([]domainResource.Resource, error) {
	if attributionGroupID == "" {
		return nil, attributiongroups.ErrNoAttributionGroupID
	}

	attributionGroup, err := s.attributionGroupsDAL.Get(ctx, attributionGroupID)
	if err != nil {
		return nil, err
	}

	if attributionGroup.Type != domainAttributions.ObjectTypeCustom {
		return nil, attributiongroups.ErrForbidden
	}

	if customerID != attributionGroup.Customer.ID {
		return nil, attributiongroups.ErrForbidden
	}

	if !attributionGroup.Access.IsOwner(requesterEmail) {
		return nil, attributiongroups.ErrForbidden
	}

	// Validate that the attribution group is not used in reports
	resourcesUsedInReport, err := s.isUsedByReport(ctx, customerID, requesterEmail, attributionGroupID)
	if err != nil {
		return nil, err
	}

	if len(resourcesUsedInReport) > 0 {
		return resourcesUsedInReport, nil
	}

	return nil, s.attributionGroupsDAL.Delete(ctx, attributionGroupID)
}

func (s *AttributionGroupsService) ListAttributionGroupsExternal(ctx context.Context, req *customerapi.Request) (*attributiongroups.AttributionGroupsListExternal, error) {
	c, err := s.customersDAL.GetCustomer(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}

	attributionGroups, err := s.attributionGroupsDAL.List(ctx, c.Snapshot.Ref, req.Email)
	if err != nil {
		return nil, err
	}

	accessDeniedCustomAttributionGroupErr, err := s.attributionGroupTierService.CheckAccessToCustomAttributionGroup(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}

	accessDeniedPresetAttributionGroupErr, err := s.attributionGroupTierService.CheckAccessToPresetAttributionGroup(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}

	filteredAttributionGroups := make([]attributiongroups.AttributionGroup, 0)

	for _, attrGroup := range attributionGroups {
		if attrGroup.Type == domainAttributions.ObjectTypeCustom && accessDeniedCustomAttributionGroupErr != nil {
			continue
		}

		if attrGroup.Type == domainAttributions.ObjectTypePreset && accessDeniedPresetAttributionGroupErr != nil {
			continue
		}

		filteredAttributionGroups = append(attributionGroups, attrGroup)
	}

	sorted, err := customerapi.SortAPIList(toAttributionGroupsListExternal(filteredAttributionGroups), req.SortBy, req.SortOrder)
	if err != nil {
		return nil, err
	}

	page, token, err := customerapi.GetEncodedAPIPage(req.MaxResults, req.NextPageToken, sorted)
	if err != nil {
		return nil, err
	}

	return &attributiongroups.AttributionGroupsListExternal{RowCount: len(page), AttributionGroups: page, PageToken: token}, nil
}

func (s *AttributionGroupsService) GetAttributionGroupExternal(ctx context.Context, attributionGroupID string) (*attributiongroups.AttributionGroupGetExternal, error) {
	attr, err := s.attributionGroupsDAL.Get(ctx, attributionGroupID)
	if err != nil {
		return nil, err
	}

	return toAttributionGroupGetExternal(ctx, attr)
}

func (s *AttributionGroupsService) isUsedByReport(
	ctx context.Context,
	customerID,
	requesterEmail string,
	attributionGroupID string,
) ([]domainResource.Resource, error) {
	var usedInReportResources []domainResource.Resource

	reports, err := s.reportDAL.GetCustomerReports(ctx, customerID)
	if err != nil {
		return nil, err
	}

	agMetadataID := fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttributionGroup, attributionGroupID)

	for _, report := range reports {
		if report.Config.IsUsingDimension(agMetadataID) {
			var reportOwner string

			if !report.Access.CanView(requesterEmail) {
				reportOwner = report.Access.GetOwner()
			}

			usedInReportResources = append(
				usedInReportResources,
				domainResource.Resource{
					ID:    report.ID,
					Name:  report.Name,
					Owner: reportOwner,
				},
			)
		}
	}

	return usedInReportResources, nil
}

// validateAttributionGroupName checks if attribution group name contains valid characters and already exists
// returns nil if name is not taken, otherwise returns error
func (s *AttributionGroupsService) validateAttributionGroupName(ctx context.Context, customerID, name string) error {
	nameRegexp := regexp.MustCompile(attributionGroupsValidNameRegex)

	if !nameRegexp.MatchString(name) {
		return attributiongroups.ErrInvalidAttributionGroupName
	}

	customerRef := s.customersDAL.GetRef(ctx, customerID)

	_, err := s.attributionGroupsDAL.GetByName(ctx, customerRef, name)
	if err == nil {
		return attributiongroups.ErrNameAlreadyExists
	} else if err != attributiongroups.ErrNotFound {
		return attributiongroups.ErrValidationsFailed
	}

	_, err = s.attributionGroupsDAL.GetByName(ctx, nil, name)
	if err == nil {
		return attributiongroups.ErrPresetNameAlreadyExists
	} else if err != attributiongroups.ErrNotFound {
		return attributiongroups.ErrValidationsFailed
	}

	return nil
}
