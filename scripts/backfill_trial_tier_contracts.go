package scripts

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	reportDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/contract/domain"
	contracts "github.com/doitintl/hello/scheduled-tasks/contract/service"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tiers "github.com/doitintl/tiers/service"
	"github.com/gin-gonic/gin"
)

const (
	defaultTrialDays = 45

	successMessageFormat = "created %s trial contract for customer %s, start: %s, end: %s"
)

type TiersScripts struct {
	*connection.Connection
	logger logger.Provider

	customerDal      customerDal.Customers
	contractsDal     fsdal.Contracts
	tiersService     *tiers.TiersService
	contractsService *contracts.ContractService
}

func NewTiersScripts(log logger.Provider, conn *connection.Connection) *TiersScripts {
	ctx := context.Background()

	reportDal := reportDal.NewReportsFirestoreWithClient(conn.Firestore)
	customersDal := customerDal.NewCustomersFirestoreWithClient(conn.Firestore)
	cloudAnalyticsService, err := cloudanalytics.NewCloudAnalyticsService(log, conn, reportDal, customersDal)
	if err != nil {
		panic(err)
	}

	return &TiersScripts{
		conn,
		log,
		customersDal,
		fsdal.NewContractsDALWithClient(conn.Firestore(ctx)),
		tiers.NewTiersService(conn.Firestore),
		contracts.NewContractService(log, conn, cloudAnalyticsService),
	}
}

type BackfilTrialContractsRequest struct {
	Package string `json:"package"`
	DryRun  bool   `json:"dryRun"`
}

func (h *TiersScripts) BackfilTrialTierContracts(ctx *gin.Context) []error {
	params, err := h.validateParams(ctx)
	if err != nil {
		return []error{err}
	}

	customersDocs, err := h.getCustomersForBackfill(ctx, params.Package)
	if err != nil {
		return []error{err}
	}

	trialTierRef, err := h.tiersService.GetTrialTierRef(ctx, pkg.PackageTierType(params.Package))
	if err != nil {
		return []error{err}
	}

	errs := []error{}

	for _, customerDoc := range customersDocs {
		if err := h.handleCustomer(ctx, customerDoc, trialTierRef.ID, params.Package, params.DryRun); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (h *TiersScripts) validateParams(ctx *gin.Context) (*BackfilTrialContractsRequest, error) {
	var params BackfilTrialContractsRequest
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return nil, err
	}

	if params.Package == "" {
		return nil, fmt.Errorf("invalid input parameters: package is empty")
	}

	if params.Package != string(pkg.NavigatorPackageTierType) && params.Package != string(pkg.SolvePackageTierType) {
		return nil, fmt.Errorf("invalid input parameters: package must be either 'navigator' or 'solve'")
	}

	return &params, nil
}

func (h *TiersScripts) getCustomersForBackfill(ctx context.Context, pkgType string) ([]*firestore.DocumentSnapshot, error) {
	fs := h.Firestore(ctx)

	docsWithEndDate, err := fs.Collection("customers").
		Where(fmt.Sprintf("%s.trialEndDate", getTierDataPath(pkgType)), "!=", "").
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	docsWithCanceledDate, err := fs.Collection("customers").
		Where(fmt.Sprintf("%s.trialCanceledDate", getTierDataPath(pkgType)), "!=", "").
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	allDocs := docsWithEndDate

	for _, doc1 := range docsWithCanceledDate {
		found := false
		for _, doc2 := range docsWithEndDate {
			if doc1.Ref.ID == doc2.Ref.ID {
				found = true
				break
			}
		}

		if !found {
			allDocs = append(allDocs, doc1)
		}
	}

	return allDocs, nil
}

func (h *TiersScripts) handleCustomer(ctx context.Context, customerDoc *firestore.DocumentSnapshot, trialTierID, pkgType string, dryRun bool) error {
	var customer common.Customer
	if err := customerDoc.DataTo(&customer); err != nil {
		return err
	}

	tierData := customer.Tiers[pkgType]

	if backfill, err := h.shouldBackfill(ctx, customerDoc.Ref, tierData, pkgType, trialTierID); err != nil || !backfill {
		return err
	}

	startDate, endDate := getTrialDates(tierData)

	c := domain.ContractInputStruct{
		CustomerID: customerDoc.Ref.ID,
		Type:       pkgType,
		Tier:       trialTierID,
		StartDate:  startDate.Format(time.RFC3339),
		EndDate:    endDate.Format(time.RFC3339),
	}

	if dryRun {
		h.logger(ctx).Infof(fmt.Sprintf("dry run: %s", successMessageFormat), pkgType, c.CustomerID, c.StartDate, c.EndDate)
		return nil
	}

	if err := h.contractsService.CreateContract(ctx, c); err != nil {
		return err
	}

	h.logger(ctx).Debugf(successMessageFormat, pkgType, c.CustomerID, c.StartDate, c.EndDate)

	return nil
}

func (h *TiersScripts) shouldBackfill(ctx context.Context, customerRef *firestore.DocumentRef, tierData *pkg.CustomerTier, pkgType, trialTierID string) (bool, error) {
	if tierData.TrialCanceledDate == nil && (tierData.TrialStartDate == nil || tierData.TrialEndDate == nil) {
		h.logger(ctx).Errorf("customer %s has invalid trial dates", customerRef.ID)
		return false, nil
	}

	allContracts, err := h.contractsDal.ListCustomerContracts(ctx, customerRef)
	if err != nil {
		return false, err
	}

	for _, contract := range allContracts {
		if contract.Type == pkgType && contract.Tier.ID == trialTierID {
			return false, nil
		}
	}

	return true, nil
}

func getTrialDates(tierData *pkg.CustomerTier) (*time.Time, *time.Time) {
	startDate := tierData.TrialStartDate
	if startDate == nil {
		date := time.Time{}

		if tierData.TrialEndDate != nil {
			date = tierData.TrialEndDate.AddDate(0, 0, -1*defaultTrialDays)
		} else {
			date = tierData.TrialCanceledDate.AddDate(0, 0, -1*defaultTrialDays)
		}

		startDate = &date
	}

	endDate := tierData.TrialEndDate
	if tierData.TrialCanceledDate != nil {
		endDate = tierData.TrialCanceledDate
	}

	return startDate, endDate
}
