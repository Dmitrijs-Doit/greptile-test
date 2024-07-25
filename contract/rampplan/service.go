package rampplan

import (
	"context"
	"fmt"
	"math"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	reportDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	attributionGroupDal "github.com/doitintl/hello/scheduled-tasks/contract/attributiongroup/dal"
	contractDAL "github.com/doitintl/hello/scheduled-tasks/contract/dal"
	contractDALIface "github.com/doitintl/hello/scheduled-tasks/contract/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/contract/rampplan/dal"
	rampPlansDal "github.com/doitintl/hello/scheduled-tasks/contract/rampplan/dal"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Service struct {
	*logger.Logging
	conn                *connection.Connection
	cloudAnalytics      cloudanalytics.CloudAnalytics
	RampPlansDal        rampPlansDal.RampPlans
	contractsDal        contractDALIface.ContractFirestore
	AttributionGroupDal attributionGroupDal.AttributionGroup
}

func NewRampPlanService(log *logger.Logging, conn *connection.Connection) (*Service, error) {
	contractDAL := contractDAL.NewContractFirestoreWithClient(conn.Firestore)
	rampPlansDAL := dal.NewRampPlansFirestoreWithClient(conn.Firestore)
	attributionGroupDAL := attributionGroupDal.NewContractsFirestoreWithClient(conn.Firestore)

	customerDal := customerDAL.NewCustomersFirestoreWithClient(conn.Firestore)
	reportDal := reportDAL.NewReportsFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(logger.FromContext, conn, reportDal, customerDal)
	if err != nil {
		return nil, err
	}

	return &Service{
		log,
		conn,
		cloudAnalytics,
		rampPlansDAL,
		contractDAL,
		attributionGroupDAL,
	}, nil
}

func (s *Service) UpdateUsage(ctx context.Context, planDoc *firestore.DocumentSnapshot) error {
	var plan pkg.RampPlan
	if err := planDoc.DataTo(&plan); err != nil {
		return err
	}

	plan.Ref = planDoc.Ref

	periodsSpends, err := s.GetActualMonthlySpend(ctx, plan)
	if err != nil {
		return err
	}

	updatedCommitmentPeriods, err := addActualSpendsToPeriods(plan.CommitmentPeriods, periodsSpends)
	if err != nil {
		return err
	}

	totalActual := 0.0

	for _, period := range updatedCommitmentPeriods {
		for _, actual := range period.Actuals {
			totalActual += actual
		}
	}

	attainmentPercent := math.Min((totalActual/plan.TargetAmount)*100, 100)

	if _, err := plan.Ref.Update(ctx, []firestore.Update{
		{Path: "attainment", Value: attainmentPercent},
		{Path: "commitmentPeriods", Value: updatedCommitmentPeriods},
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) CreateRampPlan(ctx context.Context, customerID, contractID, rampPlanName string) error {
	log := s.Logger(ctx)

	contract, err := s.contractsDal.GetCustomerContractByID(ctx, customerID, contractID)
	if err != nil {
		log.Errorf("CreateRampPlan: contract not found customerID: %s, contractID: %s", customerID, contractID)
		return err
	}

	name := rampPlanName
	if name == "" {
		name = fmt.Sprintf("%s Ramp Plan %s - %s", contract.Type, contract.StartDate.Format("2006"), contract.EndDate.Format("2006"))
	}

	log.Infof("CreateRampPlan name: %s, contract :%s, customer: %s", rampPlanName, contract.ID, contract.Customer.ID)

	if !IsEligible(contract) {
		log.Infof("Contract %s (customer: %s) not eligible\n", contract.ID, customerID)
		return nil
	}

	attributionGroupSnaps, err := s.AttributionGroupDal.GetRampPlanEligibleSpendAttributionGroup(ctx)
	if err != nil || len(attributionGroupSnaps) == 0 {
		log.Errorf("Error getting attribution group , %s", err)
		return err
	}

	rampPlan := CreateRampPlan(contract, attributionGroupSnaps[0].Ref, name)

	ref, _, err := s.RampPlansDal.AddRampPlan(ctx, rampPlan)
	if err != nil {
		log.Errorf("Error creating ramp plan, %s", err)
		return err
	}

	log.Infof("Created rampPlan %s, docRef: %s, customer: %s, contrat: %s", rampPlan.Name, ref.ID, rampPlan.Customer.ID, rampPlan.ContractID)

	return nil
}

func (s *Service) ProcessSingleContract(ctx context.Context, logger logger.ILogger, contractSnap *firestore.DocumentSnapshot, attributionGroupRef *firestore.DocumentRef, channels ProcessContractChannels) {
	var contract pkg.Contract
	if err := contractSnap.DataTo(&contract); err != nil {
		logger.Errorf("ProcessSingleContract: Error reading contract: %s", err)
		channels.Errors <- &contract

		return
	}

	contract.ID = contractSnap.Ref.ID

	if !IsEligible(&contract) {
		logger.Infof("ProcessSingleContract: contractID: %s not eligible\n", contract.ID)
		channels.NotEligible <- &contract

		return
	}

	rampPlans, err := s.RampPlansDal.GetRampPlansByContractID(ctx, contract.ID)
	if err != nil {
		logger.Errorf("ProcessSingleContract: Error reading ramp plans: %s", err)
		channels.Errors <- &contract

		return
	}

	if len(rampPlans) > 0 {
		logger.Infof("ProcessSingleContract: contractID: %s matchedContracts RampPlan: %s(%d)\n", contract.ID, rampPlans[0].Ref.ID, len(rampPlans))
		channels.Matched <- &contract

		return
	}

	rampPlan := CreateRampPlan(&contract, attributionGroupRef, "")

	ref, _, err := s.RampPlansDal.AddRampPlan(ctx, rampPlan)
	if err != nil {
		logger.Errorf("ProcessSingleContract: Error adding contract: %s, err: %s", contract.ID, err)
		channels.Errors <- &contract

		return
	}

	logger.Infof("ProcessSingleContract: Created rampPlan %s, docRef: %s, customer: %s, contract: %s\n", rampPlan.Name, ref.ID, rampPlan.Customer.ID, rampPlan.ContractID)
	channels.New <- &contract
}

func (s *Service) CreateRampPlans(ctx context.Context) error {
	log := s.Logger(ctx)

	contractSnaps, err := s.contractsDal.GetActiveContracts(ctx)
	if err != nil {
		log.Errorf("CreateRampPlans: Error getting contractSnaps, %s", err)
	}

	attributionGroupSnaps, err := s.AttributionGroupDal.GetRampPlanEligibleSpendAttributionGroup(ctx)
	if err != nil || len(attributionGroupSnaps) == 0 {
		log.Errorf("CreateRampPlans: Error getting attribution group , %s", err)
		return err
	}

	attributionGroupRef := attributionGroupSnaps[0].Ref

	maxConcurrentGoRoutine := 5
	channels := ProcessContractChannels{
		make(chan *pkg.Contract, maxConcurrentGoRoutine),
		make(chan *pkg.Contract, maxConcurrentGoRoutine),
		make(chan *pkg.Contract, maxConcurrentGoRoutine),
		make(chan *pkg.Contract, maxConcurrentGoRoutine),
	}

	var (
		notEligibleCount int
		errorCount       int
		newCount         int
		matchCount       int
	)

	common.RunConcurrentJobsOnCollection(ctx, contractSnaps, maxConcurrentGoRoutine, func(ctx context.Context, contractSnap *firestore.DocumentSnapshot) {
		s.ProcessSingleContract(ctx, log, contractSnap, attributionGroupRef, channels)
		select {
		case <-channels.Errors:
			errorCount++
		case <-channels.Matched:
			matchCount++
		case <-channels.New:
			newCount++
		case <-channels.NotEligible:
			notEligibleCount++
		}
	})

	close(channels.Errors)
	close(channels.Matched)
	close(channels.New)
	close(channels.NotEligible)

	log.Infof("notEligibleContracts: %d, matchedContracts: %d , newRampPlans: %d, errors: %d ,total: %d\n", notEligibleCount, matchCount, newCount, errorCount, len(contractSnaps))

	return nil
}
