package payermanager

import (
	"context"
	"fmt"
	"time"

	"github.com/doitintl/errors"
	"github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers"
	computestate "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/compute"
	rdsstate "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/rds"
	sagemakerstate "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/sagemaker"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

//go:generate mockery --name Service --output ./mocks --packageprefix mock
type Service interface {
	ProcessPayerStatusTransition(ctx context.Context, accountID, customerID string, initialStatus, targetStatus string, flexsaveType utils.FlexsaveType) error
	GetPayer(ctx context.Context, accountID string) (types.PayerConfig, error)
	UpdateNonStatusPayerConfigFields(ctx context.Context, payer types.PayerConfig, entry FormEntry) error
}

type service struct {
	loggerProvider logger.Provider
	payers         payers.Service
	integrations   firestore.Integrations
	compute        computestate.Service
	rds            rdsstate.Service
	sagemaker      sagemakerstate.Service
}

func NewService(log logger.Provider, conn *connection.Connection) Service {
	payersService, err := payers.NewService()
	if err != nil {
		panic(err)
	}

	return &service{
		loggerProvider: log,
		payers:         payersService,
		integrations:   firestore.NewIntegrationsDALWithClient(conn.Firestore(context.Background())),
		compute:        computestate.NewService(log, conn),
		rds:            rdsstate.NewService(log, conn),
		sagemaker:      sagemakerstate.NewService(log, conn),
	}
}

func (s *service) GetPayer(ctx context.Context, accountID string) (types.PayerConfig, error) {
	payer, err := s.payers.GetPayerConfig(ctx, accountID)
	if err != nil {
		return types.PayerConfig{}, err
	}

	if payer == nil {
		return types.PayerConfig{}, fmt.Errorf("payer '%s' not found", accountID)
	}

	return *payer, nil
}

func (s *service) UpdateNonStatusPayerConfigFields(ctx context.Context, payer types.PayerConfig, entry FormEntry) error {
	now := time.Now()

	_, err := s.payers.UpdatePayerConfigsForCustomer(ctx, []types.PayerConfig{{
		CustomerID:                  payer.CustomerID,
		AccountID:                   payer.AccountID,
		PrimaryDomain:               payer.PrimaryDomain,
		FriendlyName:                payer.FriendlyName,
		Name:                        payer.Name,
		Status:                      payer.Status,
		Type:                        payer.Type,
		Managed:                     entry.Managed,
		LastUpdated:                 &now,
		TargetPercentage:            entry.TargetPercentage,
		MinSpend:                    entry.MinSpend,
		MaxSpend:                    entry.MaxSpend,
		KeepActiveEvenWhenOnCredits: entry.KeepActiveEvenWhenOnCredits,
		DiscountDetails:             mergeDiscounts(payer.DiscountDetails, entry.Discount),
		Seasonal:                    getPointerOrDefault(payer.Seasonal, entry.Seasonal),
		RDSTargetPercentage:         entry.RDSTargetPercentage,
	}})
	if err != nil {
		return errors.Wrapf(err, "UpdatePayerConfigsForCustomer() failed for payer '%s'", payer.AccountID)
	}

	return nil
}

func (s *service) ProcessPayerStatusTransition(ctx context.Context, accountID, customerID, initialStatus, targetStatus string, flexsaveType utils.FlexsaveType) error {
	switch flexsaveType {
	case utils.ComputeFlexsaveType:
		return s.compute.ProcessPayerStatusTransition(ctx, accountID, customerID, initialStatus, targetStatus)
	case utils.SageMakerFlexsaveType:
		return s.sagemaker.ProcessPayerStatusTransition(ctx, accountID, customerID, initialStatus, targetStatus)
	case utils.RDSFlexsaveType:
		return s.rds.ProcessPayerStatusTransition(ctx, accountID, customerID, initialStatus, targetStatus)
	default:
		return errors.New("unsupported FlexsaveType")
	}
}
