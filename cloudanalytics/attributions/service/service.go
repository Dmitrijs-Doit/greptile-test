package service

import (
	"context"
	"fmt"
	"regexp"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/customerapi"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	bucketsDal "github.com/doitintl/hello/scheduled-tasks/buckets/dal"
	alertDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal"
	alertDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal/iface"
	attributionGroupsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal"
	attributionGroupsDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	attributionsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal/iface"
	attributionsDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/attributiontier"
	attributionTierServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/attributiontier/iface"
	budgetDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/caownerchecker/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metadataService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service"
	metadataIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/iface"
	metricsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal"
	metricsDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	attributionsQueryIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/iface"
	reportsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	reportsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/iface"
	domainResource "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/resource/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	entityDal "github.com/doitintl/hello/scheduled-tasks/entity/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	tier "github.com/doitintl/tiers/service"
)

type AttributionsService struct {
	loggerProvider         logger.Provider
	conn                   *connection.Connection
	dal                    iface.Attributions
	customerDal            customerDal.Customers
	reportsDal             reportsIface.Reports
	metadataService        metadataIface.MetadataIface
	attributionQuery       attributionsQueryIface.IAttributionQuery
	employeeService        doitemployees.ServiceInterface
	bucketsDal             bucketsDal.Buckets
	assetsDal              assetsDal.Assets
	entityDal              entityDal.Entites
	attributionGroupsDal   attributionGroupsDalIface.AttributionGroups
	attributionsDal        attributionsDalIface.Attributions
	metricsDal             metricsDalIface.Metrics
	budgetDal              budgetDal.Budgets
	alertDal               alertDalIface.Alerts
	attributionTierService attributionTierServiceIface.AttributionTierService
}

func NewAttributionsService(ctx context.Context, log logger.Provider, conn *connection.Connection) *AttributionsService {
	tierService := tier.NewTiersService(conn.Firestore)

	doitEmployeesService := doitemployees.NewService(conn)

	attributionDAL := attributionsDal.NewAttributionsFirestoreWithClient(conn.Firestore)

	attributionTierService := attributiontier.NewAttributionTierService(
		log,
		attributionDAL,
		tierService,
		doitEmployeesService,
	)

	return &AttributionsService{
		log,
		conn,
		dal.NewAttributionsFirestoreWithClient(conn.Firestore),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		reportsDal.NewReportsFirestoreWithClient(conn.Firestore),
		metadataService.NewMetadataService(ctx, log, conn),
		query.NewAttributionQuery(),
		doitemployees.NewService(conn),
		bucketsDal.NewBucketsFirestoreWithClient(conn.Firestore),
		assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
		entityDal.NewEntitiesFirestoreWithClient(conn.Firestore),
		attributionGroupsDal.NewAttributionGroupsFirestoreWithClient(conn.Firestore),
		attributionDAL,
		metricsDal.NewMetricsFirestoreWithClient(conn.Firestore),
		budgetDal.NewBudgetsFirestoreWithClient(conn.Firestore),
		alertDal.NewAlertsFirestoreWithClient(conn.Firestore),
		attributionTierService,
	}
}

const (
	googleCloudReports  = "google-cloud-reports"
	attrNameMaxLength   = 64
	attrDescMaxLength   = 100
	attrFiltersMaxItems = 26
	attrValidNameRegex  = "^[0-9a-zA-Z_\\-.,:()\\[\\] %&]*$"
)

func (s *AttributionsService) GetAttribution(ctx context.Context, attributionID string, isDoitEmployee bool, customerID string, userEmail string) (*attribution.AttributionAPI, error) {
	attr, err := s.dal.GetAttribution(ctx, attributionID)
	if err != nil {
		return nil, err
	}

	if attr.Type == "custom" && attr.Customer.ID != customerID {
		return nil, ErrWrongCustomer
	}

	// validating the user has access to this attribution
	if !isDoitEmployee && !doesUserHaveViewAccessToAttr(attr, userEmail) {
		return nil, ErrMissingPermissions
	}

	// map attribution to API response
	return toAttributionAPIItem(attr), nil
}

func (s *AttributionsService) ListAttributions(ctx context.Context, req *customerapi.Request) (*attribution.AttributionsList, error) {
	cRef := s.customerDal.GetRef(ctx, req.CustomerID)
	attributionList, err := s.dal.ListAttributions(ctx, req, cRef)
	if err != nil {
		return nil, err
	}

	accessDeniedCustomAttributionsErr, err := s.attributionTierService.CheckAccessToCustomAttribution(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}

	accessDeniedPresetAttributionsErr, err := s.attributionTierService.CheckAccessToPresetAttribution(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}

	filteredAttributionList := make([]attribution.Attribution, 0)

	for _, attr := range attributionList {
		if attr.Type == string(attribution.ObjectTypeCustom) && accessDeniedCustomAttributionsErr != nil {
			continue
		}

		if attr.Type == string(attribution.ObjectTypePreset) && accessDeniedPresetAttributionsErr != nil {
			continue
		}

		filteredAttributionList = append(filteredAttributionList, attr)
	}

	extAttrList := customerapi.FilterAPIList(
		ToAttributionList(filteredAttributionList),
		req.Filters,
	)
	sorted, err := customerapi.SortAPIList(extAttrList, req.SortBy, req.SortOrder)

	if err != nil {
		return nil, err
	}

	page, token, err := customerapi.GetEncodedAPIPage(req.MaxResults, req.NextPageToken, sorted)

	if err != nil {
		return nil, err
	}

	return &attribution.AttributionsList{
		RowCount:     len(page),
		PageToken:    token,
		Attributions: page,
	}, nil
}

func (s *AttributionsService) CreateAttribution(ctx context.Context, req *CreateAttributionRequest) (*attribution.AttributionAPI, error) {
	l := s.loggerProvider(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	// validate user if not doitEmployee
	isDoitEmployee, ok := ctx.Value(common.DoitEmployee).(bool)
	if ok && !isDoitEmployee {
		user, err := s.GetCurrentUser(ctx, req.UserID)
		if err != nil {
			return nil, err
		}

		if !user.HasAttributionsPermission(ctx) {
			return nil, ErrForbidden
		}
	}

	// validate filters
	if err := s.ValidateAttributionFilters(ctx, req.Attribution, req.UserID, req.CustomerID); err != nil {
		return nil, err
	}

	// validate formula
	if err := s.attributionQuery.ValidateFormula(ctx, bq, len(req.Attribution.Filters), req.Attribution.Formula); err != nil {
		return nil, err
	}

	// Add customer ref
	customer, err := s.customerDal.GetCustomer(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}

	req.Attribution.Customer = customer.Snapshot.Ref
	publicAccessView := collab.PublicAccessView
	collab.SetAccess(&req.Attribution, req.Email, &publicAccessView)

	if err = s.validateName(req.Attribution.Name); err != nil {
		return nil, err
	}

	if err = s.validateDescription(req.Attribution.Description); err != nil {
		return nil, err
	}

	// Create attribution in firestore
	a, err := s.dal.CreateAttribution(ctx, &req.Attribution)
	if err != nil {
		return nil, err
	}

	return toAttributionAPIItem(a), nil
}

func (s *AttributionsService) UpdateAttribution(ctx context.Context, req *UpdateAttributionRequest) (*attribution.AttributionAPI, error) {
	currentAttribution, err := s.GetAttributionByID(ctx, req.Attribution.ID)
	// validate user if not doitEmployee
	if v, ok := ctx.Value(common.DoitEmployee).(bool); ok && !v {
		user, err := s.GetCurrentUser(ctx, req.UserID)
		if err != nil {
			return nil, err
		}

		if user.Customer.Ref.ID != req.CustomerID ||
			!currentAttribution.Access.CanEdit(user.Email) {
			return nil, ErrForbidden
		}
	}

	if err != nil {
		return nil, err
	}

	if currentAttribution == nil {
		return nil, ErrNotFound
	}

	updates, err := s.getAttributionUpdates(ctx, &req.Attribution, currentAttribution, req.UserID)
	if err != nil {
		return nil, err
	}

	if len(updates) == 0 {
		return nil, ErrBadRequest
	}

	// Update attribution in firestore
	err = s.dal.UpdateAttribution(ctx, req.Attribution.ID, updates)
	if err != nil {
		return nil, err
	}

	updatedAttribution, err := s.GetAttributionByID(ctx, req.Attribution.ID)
	if err != nil {
		return nil, err
	}

	return toAttributionAPIItem(updatedAttribution), nil
}

func (s *AttributionsService) UpdateAttributions(ctx context.Context, customerID string, attributions []*attribution.Attribution, userID string) ([]*attribution.AttributionAPI, error) {
	var updatedAttributions []*attribution.AttributionAPI

	for _, attr := range attributions {
		updatedAttr, err := s.UpdateAttribution(ctx, &UpdateAttributionRequest{customerID, *attr, userID})
		if err != nil {
			return nil, err
		}

		updatedAttributions = append(updatedAttributions, updatedAttr)
	}

	return updatedAttributions, nil
}

func (s *AttributionsService) GetAttributionByID(ctx context.Context, attributionID string) (*attribution.Attribution, error) {
	attr, err := s.dal.GetAttribution(ctx, attributionID)
	if err != nil {
		return nil, err
	}

	return attr, nil
}

func (s *AttributionsService) GetAttributions(ctx context.Context, attributionsIDs []string) ([]*attribution.Attribution, error) {
	var docRefs []*firestore.DocumentRef

	for _, attributionID := range attributionsIDs {
		docRef := s.dal.GetRef(ctx, attributionID)
		docRefs = append(docRefs, docRef)
	}

	attributions, err := s.dal.GetAttributions(ctx, docRefs)
	if err != nil {
		return nil, err
	}

	return attributions, nil
}

func (s *AttributionsService) handleShareAttributions(ctx context.Context, req *ShareAttributionRequest) error {
	fs := s.conn.Firestore(ctx)

	attrRef := fs.Collection("dashboards").Doc(googleCloudReports).Collection("attributions").Doc(req.AttributionID)

	if _, err := attrRef.Update(ctx, []firestore.Update{
		{
			FieldPath: []string{"collaborators"},
			Value:     req.Collaborators,
		}}); err != nil {
		return err
	}

	if req.Role != nil {
		if _, err := attrRef.Update(ctx, []firestore.Update{
			{
				FieldPath: []string{"public"},
				Value:     *req.Role,
			}}); err != nil {
			return err
		}
	}

	return nil
}

func (s *AttributionsService) handleDeleteAttributions(ctx context.Context, req *DeleteAttributionsRequest) ([]AttributionDeleteValidation, error) {
	fs := s.conn.Firestore(ctx)

	customerRef := fs.Collection("customers").Doc(req.CustomerID)

	requesterEmail := req.Email

	var validations []AttributionDeleteValidation

	for _, attrID := range req.AttributionsIDs {
		v := AttributionDeleteValidation{
			ID:        attrID,
			Resources: map[domainResource.ResourceType][]domainResource.Resource{},
		}

		attrRef := s.dal.GetRef(ctx, attrID)

		attr, err := s.dal.GetAttribution(ctx, attrID)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				v.Error = ErrNotFound
				validations = append(validations, v)

				continue
			}

			return nil, err
		}

		if attr.Type != string(attribution.ObjectTypeCustom) {
			v.Error = ErrCannotDeleteNonCustom
			validations = append(validations, v)

			continue
		}

		if !attr.Access.IsOwner(req.Email) {
			v.Error = ErrCannotDeleteNotOwner
			validations = append(validations, v)

			continue
		}

		// Validate that the attribution is not used in reportsResources
		reportsResources, err := s.validateNotInReports(ctx, req.CustomerID, requesterEmail, attrID)
		if err != nil {
			v.Error = ErrAttrDeleteValidationFailed
			validations = append(validations, v)

			continue
		}

		if len(reportsResources) > 0 {
			v.Resources[domainResource.Reports] = reportsResources
		}

		attributionGroups, err := s.attributionGroupsDal.GetByCustomer(ctx, customerRef, attrRef)
		if err != nil {
			v.Error = ErrAttrDeleteValidationFailed
			validations = append(validations, v)

			continue
		}

		if len(attributionGroups) > 0 {
			v.Resources[domainResource.AttributionGroups] = domainResource.NewResourcesFromAttributionGroups(
				requesterEmail,
				attributionGroups,
			)
		}

		// Validate that attribution is not used by any budget
		budgets, err := s.budgetDal.GetByCustomerAndAttribution(ctx, customerRef, attrRef)
		if err != nil {
			v.Error = ErrAttrDeleteValidationFailed
			validations = append(validations, v)

			continue
		}

		if len(budgets) > 0 {
			v.Resources[domainResource.Budgets] = domainResource.NewResourcesFromBudgets(requesterEmail, budgets)
		}

		// Validate that attribution is not used by any alert
		alerts, err := s.alertDal.GetByCustomerAndAttribution(ctx, customerRef, attrRef)
		if err != nil {
			v.Error = ErrAttrDeleteValidationFailed
			validations = append(validations, v)

			continue
		}

		if len(alerts) > 0 {
			v.Resources[domainResource.Alerts] = domainResource.NewResourcesFromAlerts(requesterEmail, alerts)
		}

		// Validate that attribution is not used by any custom metric
		metrics, err := s.metricsDal.GetMetricsUsingAttr(ctx, attrRef)
		if err != nil {
			v.Error = ErrAttrDeleteValidationFailed
			validations = append(validations, v)

			continue
		}

		if len(metrics) > 0 {
			v.Resources[domainResource.Metrics] = domainResource.NewResourcesFromMetrics(metrics)
		}

		// Validate that attribution is not used by any organization
		customerOrganizations, err := s.customerDal.GetCustomerOrgs(ctx, customerRef.ID, "")
		if err != nil {
			v.Error = ErrAttrDeleteValidationFailed
			validations = append(validations, v)

			continue
		}

		var orgsWithAttribution []*common.Organization

		if len(customerOrganizations) > 0 {
			for _, org := range customerOrganizations {
				for _, scopeRef := range org.Scope {
					if scopeRef.ID == attrRef.ID {
						orgsWithAttribution = append(orgsWithAttribution, org)
						break
					}
				}
			}
		}

		if len(orgsWithAttribution) > 0 {
			v.Resources[domainResource.Organizations] = domainResource.NewResourcesFromOrgs(orgsWithAttribution)
		}

		if len(v.Resources) > 0 {
			v.Error = ErrAttrUsedInOneOrMoreResources
			validations = append(validations, v)

			continue
		}

		// Attribution is not used by any other object, delete it
		if err = s.attributionsDal.DeleteAttribution(ctx, attrID); err != nil {
			return nil, ErrFailedToDeleteAttribution
		}
	}

	return validations, nil
}

func (s *AttributionsService) GetCurrentUser(ctx context.Context, userID string) (*common.User, error) {
	fs := s.conn.Firestore(ctx)
	userRef := fs.Collection("users").Doc(userID)
	user, err := common.GetUser(ctx, userRef)

	if err != nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (s *AttributionsService) DeleteAttributions(ctx context.Context, req *DeleteAttributionsRequest) ([]AttributionDeleteValidation, error) {
	// validate user if not doitEmployee
	isDoitEmployee, ok := ctx.Value(common.CtxKeys.DoitEmployee).(bool)
	if !ok || !isDoitEmployee {
		user, err := s.GetCurrentUser(ctx, req.UserID)
		if err != nil {
			return nil, err
		}

		if !user.HasAttributionsPermission(ctx) {
			return nil, ErrForbidden
		}
	}

	// Delete attributions from firestore
	return s.handleDeleteAttributions(ctx, req)
}

func (s *AttributionsService) ValidateAttributionFilters(ctx context.Context, attribution attribution.Attribution, userID string, customerID string) error {
	isDoitEmployee, ok := ctx.Value(common.DoitEmployee).(bool)
	if !ok {
		return ErrBadRequest
	}

	userEmail, ok := ctx.Value(common.CtxKeys.Email).(string)
	if !ok {
		return ErrEmailNotFound
	}

	if len(attribution.Filters) > attrFiltersMaxItems {
		return ErrFiltersTooLong
	}

	cID := customerID

	if customerID != "" {
		customer, err := s.customerDal.GetCustomerOrPresentationModeCustomer(ctx, customerID)
		if err != nil {
			return err
		}

		cID = customer.Snapshot.Ref.ID
	}

	dimensions, err := s.metadataService.ExternalAPIList(
		metadataIface.ExternalAPIListArgs{
			Ctx:            ctx,
			IsDoitEmployee: isDoitEmployee,
			UserID:         userID,
			CustomerID:     cID,
			UserEmail:      userEmail,
		},
	)
	if err != nil {
		return err
	}

	for i, filter := range attribution.Filters {
		if filter.Type == metadata.MetadataFieldTypeAttributionGroup || filter.Type == metadata.MetadataFieldTypeAttribution {
			return fmt.Errorf("filter %d is not valid", i+1)
		}

		filterExists := false

		for _, dimension := range dimensions {
			if filter.Key == dimension.ID && filter.Type == dimension.Type {
				filterExists = true
				break
			}
		}

		if !filterExists {
			return fmt.Errorf("filter %d is not valid", i+1)
		}
	}

	return nil
}

func (s *AttributionsService) getAttributionUpdates(ctx context.Context, att *attribution.Attribution, currentAttribution *attribution.Attribution, userID string) ([]firestore.Update, error) {
	updates := make([]firestore.Update, 0)

	if att.Name != "" {
		if err := s.validateName(att.Name); err != nil {
			return nil, err
		}

		nameUpdate := firestore.Update{
			Path:  "name",
			Value: att.Name,
		}
		updates = append(updates, nameUpdate)
	}

	if att.Description != "" {
		if err := s.validateDescription(att.Description); err != nil {
			return nil, err
		}

		descriptionUpdate := firestore.Update{
			Path:  "description",
			Value: att.Description,
		}
		updates = append(updates, descriptionUpdate)
	}

	if att.Formula != "" || att.Filters != nil {
		// validate formula
		err := s.validateFormula(ctx, att, currentAttribution)

		if err != nil {
			return nil, err
		}

		if att.Filters != nil {
			// validate filters
			if err := s.ValidateAttributionFilters(ctx, *att, userID, currentAttribution.Customer.ID); err != nil {
				return nil, err
			}

			filtersUpdate := firestore.Update{
				Path:  "filters",
				Value: att.Filters,
			}
			updates = append(updates, filtersUpdate)
		}

		if att.Formula != "" {
			formulaUpdate := firestore.Update{
				Path:  "formula",
				Value: att.Formula,
			}
			updates = append(updates, formulaUpdate)
		}
	}

	if att.Draft != nil {
		draftUpdate := firestore.Update{
			Path:  "draft",
			Value: att.Draft,
		}
		expireByUpdate := firestore.Update{
			Path:  "expireBy",
			Value: att.ExpireBy,
		}
		updates = append(updates, draftUpdate, expireByUpdate)
	}

	return updates, nil
}

func (s *AttributionsService) validateFormula(ctx context.Context, att *attribution.Attribution, currentAttribution *attribution.Attribution) error {
	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		s.loggerProvider(ctx).Warningf("no bq client found for origin. Using default")
	}

	var err error
	if att.Formula == "" {
		err = s.attributionQuery.ValidateFormula(ctx, bq, len(att.Filters), currentAttribution.Formula)
	} else if att.Filters == nil {
		err = s.attributionQuery.ValidateFormula(ctx, bq, len(currentAttribution.Filters), att.Formula)
	} else {
		err = s.attributionQuery.ValidateFormula(ctx, bq, len(att.Filters), att.Formula)
	}

	return err
}

func (s *AttributionsService) HandleErrors(err error) error {
	switch status.Code(err) {
	case codes.NotFound:
		return ErrNotFound
	case codes.PermissionDenied:
		return ErrForbidden
	default:
		return ErrInternalServerError
	}
}

func (s *AttributionsService) ShareAttributions(ctx context.Context, req *ShareAttributionRequest, email string, userID string) error {
	caOwnerChecker := service.NewCAOwnerChecker(s.conn)
	isCAOwner, err := caOwnerChecker.CheckCAOwner(ctx, s.employeeService, userID, email)

	if err != nil {
		return err
	}

	attr, err := s.GetAttributionByID(ctx, req.AttributionID)
	if err != nil {
		return err
	}

	if err := ValidateCollaborators(attr, req.Collaborators, email, isCAOwner); err != nil {
		return err
	}

	if err := s.handleShareAttributions(ctx, req); err != nil {
		return err
	}

	return nil
}

func ValidateCollaborators(attr *attribution.Attribution, collaborators []collab.Collaborator, email string, isCAOwner bool) error {
	var newOwner string

	var owners int

	for _, collaborator := range collaborators {
		if collaborator.Role == collab.CollaboratorRoleOwner {
			owners++

			newOwner = collaborator.Email
		}

		if !slice.Contains([]string{
			string(collab.CollaboratorRoleOwner),
			string(collab.CollaboratorRoleEditor),
			string(collab.CollaboratorRoleViewer),
		}, string(collaborator.Role)) {
			err := ErrBadCollaborator
			return err
		}
	}

	if owners != 1 {
		err := ErrMultipleOwners
		return err
	}

	var currentOwner string

	for _, collaborator := range attr.Collaborators {
		if collaborator.Role == collab.CollaboratorRoleOwner {
			currentOwner = collaborator.Email
		}
	}

	if newOwner != currentOwner && currentOwner != email && !isCAOwner {
		return ErrNonOwner
	}

	return nil
}

func (s *AttributionsService) validateNotInReports(
	ctx context.Context,
	customerID string,
	requesterEmail string,
	attributionID string,
) ([]domainResource.Resource, error) {
	var resources []domainResource.Resource

	if customerID == "" {
		return nil, ErrCustomerIDRequired
	}

	if attributionID == "" {
		return nil, ErrAttributionIDRequired
	}

	reports, err := s.reportsDal.GetCustomerReports(ctx, customerID)
	if err != nil {
		return nil, err
	}

	for _, report := range reports {
		if report.Config == nil || report.Config.Filters == nil {
			continue
		}

		for _, filter := range report.Config.Filters {
			if filter.ID == "attribution:attribution" && filter.Values != nil {
				if slice.Contains(*filter.Values, attributionID) {
					var reportOwner string

					if !report.Access.CanView(requesterEmail) {
						reportOwner = report.Access.GetOwner()
					}

					resources = append(
						resources,
						domainResource.Resource{
							ID:    report.ID,
							Name:  report.Name,
							Owner: reportOwner,
						})
				}
			}
		}
	}

	return resources, nil
}

func doesUserHaveViewAccessToAttr(attr *attribution.Attribution, userEmail string) bool {
	if attr.Type == "preset" {
		return true
	}

	if *attr.Public == collab.PublicAccessView || *attr.Public == collab.PublicAccessEdit {
		return true
	}

	for _, collaborator := range attr.Collaborators {
		if collaborator.Email == userEmail {
			return true
		}
	}

	return false
}

func (s *AttributionsService) validateName(name string) error {
	if len(name) > attrNameMaxLength {
		return ErrNameTooLong
	}

	if !regexp.MustCompile(attrValidNameRegex).MatchString(name) {
		return ErrInvalidName
	}

	return nil
}

func (s *AttributionsService) validateDescription(description string) error {
	if len(description) > attrDescMaxLength {
		return ErrDescriptionTooLong
	}

	return nil
}
