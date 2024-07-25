package service

import (
	"context"

	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
)

func (s *AnalyticsAlertsService) GetAlert(ctx context.Context, alertID string) (*AlertAPI, error) {
	a, err := s.alertsDal.GetAlert(ctx, alertID)

	if err != nil {
		return nil, err
	}

	alert, err := toAlertAPI(a)
	if err != nil {
		return nil, err
	}

	return alert, nil
}

func (s *AnalyticsAlertsService) ListAlerts(ctx context.Context, args ExternalAPIListArgsReq) (*ExternalAlertList, error) {
	customerRef := s.customersDAL.GetRef(ctx, args.CustomerID)
	alerts, err := s.alertsDal.GetAlertsByCustomer(ctx, &iface.AlertsByCustomerArgs{
		CustomerRef: customerRef,
		Email:       args.Email,
	})

	if err != nil {
		return nil, err
	}

	apiAlerts, err := toListAlertAPI(alerts)
	if err != nil {
		return nil, err
	}

	filteredAlerts := customerapi.FilterAPIList(apiAlerts, args.Filters)

	sortedAlerts, err := customerapi.SortAPIList(filteredAlerts, args.SortBy, args.SortOrder)
	if err != nil {
		return nil, err
	}

	page, token, err := customerapi.GetEncodedAPIPage(args.MaxResults, args.NextPageToken, sortedAlerts)
	if err != nil {
		return nil, err
	}

	return &ExternalAlertList{
		PageToken: token,
		RowCount:  len(page),
		Alerts:    page,
	}, err
}

func (s *AnalyticsAlertsService) DeleteAlert(
	ctx context.Context,
	customerID string,
	email string,
	alertID string,
) error {
	alert, err := s.alertsDal.GetAlert(ctx, alertID)
	if err != nil {
		return err
	}

	if customerID != alert.Customer.ID {
		return domain.ErrForbidden
	}

	var isOwner bool

	for _, collaborator := range alert.Collaborators {
		if collaborator.Email == email {
			isOwner = collaborator.Role == collab.CollaboratorRoleOwner
			break
		}
	}

	if !isOwner {
		return domain.ErrorUnAuthorized
	}

	return s.alertsDal.DeleteAlert(ctx, alertID)
}

func (s *AnalyticsAlertsService) CreateAlert(ctx context.Context, args ExternalAPICreateUpdateArgsReq) ExternalAPICreateUpdateResp {
	validatedAlert, validationErrors := s.validateCreateAlertRequest(ctx, args)

	if len(validationErrors) != 0 {
		return ExternalAPICreateUpdateResp{ValidationErrors: validationErrors, Error: domain.ErrValidationErrors}
	}

	collab.SetAccess(validatedAlert, args.Email, nil)

	a, err := s.alertsDal.CreateAlert(ctx, validatedAlert)
	if err != nil {
		return ExternalAPICreateUpdateResp{Error: domain.ErrValidationErrors}
	}

	alert, err := toAlertAPI(a)
	if err != nil {
		return ExternalAPICreateUpdateResp{Error: domain.ErrValidationErrors}
	}

	return ExternalAPICreateUpdateResp{Alert: alert}
}

func (s *AnalyticsAlertsService) UpdateAlert(ctx context.Context, alertID string, args ExternalAPICreateUpdateArgsReq) ExternalAPICreateUpdateResp {
	currentAlert, err := s.alertsDal.GetAlert(ctx, alertID)
	if err != nil {
		return ExternalAPICreateUpdateResp{
			Error: err,
		}
	}

	if !currentAlert.Access.CanEdit(args.Email) {
		return ExternalAPICreateUpdateResp{
			Error: domain.ErrForbidden,
		}
	}

	updates, validationErrors := s.validateUpdateAlertRequest(ctx, args)

	if len(validationErrors) != 0 {
		return ExternalAPICreateUpdateResp{
			ValidationErrors: validationErrors,
			Error:            domain.ErrValidationErrors,
		}
	}

	err = s.alertsDal.UpdateAlert(ctx, alertID, updates)
	if err != nil {
		return ExternalAPICreateUpdateResp{
			Error: err,
		}
	}

	a, err := s.alertsDal.GetAlert(ctx, alertID)
	if err != nil {
		return ExternalAPICreateUpdateResp{
			Error: err,
		}
	}

	updatedAlert, _ := toAlertAPI(a)

	return ExternalAPICreateUpdateResp{
		Alert: updatedAlert,
	}
}
