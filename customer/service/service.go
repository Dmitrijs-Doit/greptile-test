package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	originDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
	contractDalIface "github.com/doitintl/hello/scheduled-tasks/contract/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/customer/domain"
	entitiesDALIface "github.com/doitintl/hello/scheduled-tasks/entity/dal"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/dal/invoices"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	marketplaceGCPDal "github.com/doitintl/hello/scheduled-tasks/marketplace/dal"
	marketplaceGCPIface "github.com/doitintl/hello/scheduled-tasks/marketplace/dal/iface"
	userIface "github.com/doitintl/hello/scheduled-tasks/user/dal/iface"
)

type Service struct {
	loggerProvider    logger.Provider
	conn              *connection.Connection
	cloudAnalytics    cloudanalytics.CloudAnalytics
	userDAL           userIface.IUserFirestoreDAL
	customerDAL       dal.Customers
	entitiesDAL       entitiesDALIface.Entites
	assetDAL          assetsDal.Assets
	contractDAL       contractDalIface.ContractFirestore
	invoiceDAL        invoices.InvoicesDAL
	marketplaceGCPDAL marketplaceGCPIface.IAccountFirestoreDAL
}

func NewCustomerService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	cloudAnalytics cloudanalytics.CloudAnalytics,
	userDAL userIface.IUserFirestoreDAL,
	customerDAL dal.Customers,
	entitiesDAL entitiesDALIface.Entites,
	assetDAL assetsDal.Assets,
	contractDAL contractDalIface.ContractFirestore,
	invoiceDAL invoices.InvoicesDAL,
	marketplaceGCPDAL marketplaceGCPIface.IAccountFirestoreDAL,
) (*Service, error) {
	return &Service{
		loggerProvider,
		conn,
		cloudAnalytics,
		userDAL,
		customerDAL,
		entitiesDAL,
		assetDAL,
		contractDAL,
		invoiceDAL,
		marketplaceGCPDAL,
	}, nil
}

func (s *Service) ClearCustomerUsersNotifications(ctx context.Context, customerID string) error {
	users, err := s.userDAL.GetCustomerUsersWithNotifications(ctx, customerID, false)
	if err != nil {
		return err
	}

	var errs error

	for _, user := range users {
		err = s.userDAL.ClearUserNotifications(ctx, user)
		if err != nil {
			errs = fmt.Errorf("%w, %v", errs, err)
		}
	}

	if errs != nil {
		return errs
	}

	return nil
}

func (s *Service) RestoreCustomerUsersNotifications(ctx context.Context, customerID string) error {
	users, err := s.userDAL.GetCustomerUsersWithNotifications(ctx, customerID, true)
	if err != nil {
		return err
	}

	var errs error

	for _, user := range users {
		err = s.userDAL.RestoreUserNotifications(ctx, user)
		if err != nil {
			errs = fmt.Errorf("%w, %v", errs, err)
		}
	}

	if errs != nil {
		return errs
	}

	return nil
}

func (s *Service) Delete(
	ctx context.Context,
	customerID string,
	execute bool,
) error {
	const itemLimit = 1

	customer, err := s.customerDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	customerRef := customer.Snapshot.Ref

	entities, err := s.entitiesDAL.GetCustomerEntities(ctx, customerRef)
	if err != nil {
		return err
	}

	if len(entities) > 0 {
		return ErrCustomerHasBillingProfiles
	}

	contracts, err := s.contractDAL.ListContracts(ctx, customerRef, itemLimit)
	if err != nil {
		return err
	}

	if len(contracts) > 0 {
		return ErrCustomerHasContracts
	}

	assets, err := s.assetDAL.ListBaseAssetsForCustomer(ctx, customerRef, itemLimit)
	if err != nil {
		return err
	}

	if len(assets) > 0 {
		return ErrCustomerHasAssets
	}

	invoices, err := s.invoiceDAL.ListInvoices(ctx, customerRef, itemLimit)
	if err != nil {
		return err
	}

	if len(invoices) > 0 {
		return ErrCustomerHasInvoices
	}

	users, err := s.userDAL.ListUsers(ctx, customerRef, itemLimit)
	if err != nil {
		return err
	}

	if len(users) > 0 {
		return ErrCustomerHasUsers
	}

	gcpMarketplaceAccount, err := s.marketplaceGCPDAL.GetAccountByCustomer(ctx, customerID)
	if err != nil && !errors.Is(err, marketplaceGCPDal.ErrAccountNotFound) {
		return err
	}

	if gcpMarketplaceAccount != nil {
		return ErrCustomerHasGCPMarketplaceAccounts
	}

	if !execute {
		return nil
	}

	err = s.customerDAL.DeleteCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) ListAccountManagers(ctx context.Context, customerID string) (*domain.AccountManagerListAPI, error) {
	accountTeam, err := s.customerDAL.GetCustomerAccountTeam(ctx, customerID)
	if err != nil {
		return nil, err
	}

	return &domain.AccountManagerListAPI{
		AccountManagers: accountTeam,
	}, err
}

func getSegmentQueryRequestCols() []*queryDomain.QueryRequestX {
	field := "T.usage_date_time"

	return []*queryDomain.QueryRequestX{
		{
			Field:     field,
			ID:        "datetime:year",
			Key:       "year",
			AllowNull: false,
			Position:  queryDomain.QueryFieldPositionCol,
			Type:      "datetime",
		},
		{
			Field:     field,
			ID:        "datetime:month",
			Key:       "month",
			AllowNull: false,
			Position:  queryDomain.QueryFieldPositionCol,
			Type:      "datetime",
		},
	}
}

func (s *Service) getSegmentQueryRequest(ctx context.Context, customerID string, now time.Time) (*cloudanalytics.QueryRequest, error) {
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, -90)
	timeSettings := &cloudanalytics.QueryRequestTimeSettings{
		Interval: "month",
		From:     &from,
		To:       &to,
	}
	qr := cloudanalytics.QueryRequest{
		CloudProviders: &[]string{common.Assets.AmazonWebServices, common.Assets.GoogleCloud, common.Assets.MicrosoftAzure},
		Origin:         originDomain.QueryOriginOthers,
		Currency:       fixer.USD,
		Type:           "report",
		TimeSettings:   timeSettings,
		Cols:           getSegmentQueryRequestCols(),
	}

	var err error

	qr.Accounts, err = s.cloudAnalytics.GetAccounts(ctx, customerID, nil, []*report.ConfigFilter{})
	if err != nil {
		return nil, err
	}

	return &qr, nil
}

const (
	averageSpendFor3Month = 3
	investLimit           = 10000
	incubateLimit         = 100000
	currentSegment        = "currentSegment"
	customerSegment       = "customerSegment"

	updateCustomerSegmentPathTemplate = "/tasks/segment/%s"
)

func getSegmentValue(avgVal float64) common.SegmentValue {
	switch {
	case avgVal < investLimit:
		return common.Invest
	case avgVal < incubateLimit:
		return common.Incubate
	default:
		return common.Accelerate
	}
}

func getCustomerSegmentValue(rows [][]bigquery.Value, metricOffset int) (float64, error) {
	var sum float64

	for _, row := range rows {
		val, ok := row[metricOffset].(float64)
		if !ok {
			return 0, errors.New("error parsing row value")
		}

		sum += val
	}

	return sum / averageSpendFor3Month, nil
}

func (s *Service) UpdateSegment(ctx context.Context, customerID string) error {
	queryRequest, err := s.getSegmentQueryRequest(ctx, customerID, time.Now())
	if err != nil {
		return err
	}

	queryResult, err := s.cloudAnalytics.GetQueryResult(ctx, queryRequest, customerID, "")
	if err != nil {
		if errors.As(err, &service.ErrNoTablesFound{}) {
			return nil
		}

		return err
	}

	metricOffset := len(queryRequest.Rows) + len(queryRequest.Cols)

	avgVal, err := getCustomerSegmentValue(queryResult.Rows, metricOffset)
	if err != nil {
		return err
	}

	customer, err := s.customerDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	segmentValue := getSegmentValue(avgVal)

	segment := common.CustomerSegment{
		CurrentSegment: segmentValue,
	}

	if customer.CustomerSegment != nil && customer.CustomerSegment.OverrideSegment != "" {
		segment.OverrideSegment = customer.CustomerSegment.OverrideSegment
	}

	return s.customerDAL.UpdateCustomerFieldValue(ctx, customerID, customerSegment, segment)
}

func (s *Service) UpdateAllCustomersSegment(ctx context.Context) (taskErrors []error, _ error) {
	var allAssets []*pkg.BaseAsset

	assetTypes := []string{
		common.Assets.AmazonWebServices,
		common.Assets.GoogleCloud,
		common.Assets.MicrosoftAzure,
		common.Assets.AmazonWebServicesStandalone,
		common.Assets.GoogleCloudStandalone,
	}

	for _, assetType := range assetTypes {
		assets, err := s.assetDAL.ListBaseAssets(ctx, assetType)
		if err != nil {
			return nil, fmt.Errorf("failed to list all %s assets: %w", assetType, err)
		}

		allAssets = append(allAssets, assets...)
	}

	uniqCustomers := make(map[string]bool)

	for _, asset := range allAssets {
		if asset.Customer == nil {
			continue
		}

		customerID := asset.Customer.ID

		if _, ok := uniqCustomers[customerID]; ok {
			continue
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf(updateCustomerSegmentPathTemplate, customerID),
			Queue:  common.TaskQueueUpdateCustomersSegment,
		}

		if _, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(nil)); err != nil {
			taskErrors = append(taskErrors, fmt.Errorf("failed to create task for customer %s: %w", customerID, err))
		}

		uniqCustomers[asset.Customer.ID] = true
	}

	return taskErrors, nil
}
