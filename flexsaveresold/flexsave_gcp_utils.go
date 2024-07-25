package flexsaveresold

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	. "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	devFlexsaveGCPUsageTable     = "doitintl-cmp-global-data-dev.gcp_custom_billing.usage_agg_60_days"
	prodFlexsaveGCPUsageTable    = "doitintl-cmp-global-data.gcp_custom_billing.usage_agg_60_days"
	requiredFlexsaveGCPUsageDays = 7
)

// CanEnableFlexSaveGCP returns whether a specific customer has the requirements to enable flexsave gcp.
func (s *Service) CanEnableFlexsaveGCP(ctx context.Context, customerID string, userID string, doitEmployee bool) (*CloudEnablementDetails, error) {
	fs := s.Firestore(ctx)

	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	if slice.Contains(customer.EarlyAccessFeatures, "FSGCP Disabled") {
		return nil, ErrFlexsaveDisabled
	}

	if !doitEmployee {
		if err := assertPermissions(ctx, fs, userID); err != nil {
			return nil, err
		}
	}

	isAlreadyEnabled, err := s.assertIsAlreadyEnabled(ctx, customerID)
	if err != nil {
		return nil, err
	}

	if isAlreadyEnabled {
		errMessage := ErrAlreadyEnabled.Error()
		return &CloudEnablementDetails{ReasonCantEnable: &errMessage}, nil
	}

	return s.GetGCPFlexSaveEnablementDetails(ctx, customerID)
}

func (s *Service) GetGCPFlexSaveEnablementDetails(ctx context.Context, customerID string) (*CloudEnablementDetails, error) {
	err := s.assertGCPContract(ctx, customerID)
	if err != nil {
		if err == ErrNoBillingProfile || err == ErrNoContract {
			errMessage := err.Error()
			return &CloudEnablementDetails{ReasonCantEnable: &errMessage}, nil
		}

		return nil, err
	}

	savingsSummary, err := s.assertComputeEngineSpend(ctx, customerID)
	if err != nil {
		if err == ErrNoSpend {
			errMessage := err.Error()
			return &CloudEnablementDetails{ReasonCantEnable: &errMessage}, nil
		}

		return nil, err
	}

	var data CloudEnablementDetails

	if savingsSummary == nil {
		return nil, errors.New("missing savings summary")
	}

	if savingsSummary.NextMonth.OnDemandSpend > 0 {
		data.OnDemandSpend = &savingsSummary.NextMonth.OnDemandSpend
	}

	if savingsSummary.NextMonth.Savings > 0 {
		data.Savings = &savingsSummary.NextMonth.Savings
	}

	if savingsSummary.NextMonth.SavingsRate > 0 {
		data.SavingsRate = &savingsSummary.NextMonth.SavingsRate
	}

	return &data, nil
}

// assertIsAlreadyEnabled checks if customer is already FlexSave enabled
func (s *Service) assertIsAlreadyEnabled(ctx context.Context, customerID string) (bool, error) {
	fs := s.Firestore(ctx)

	docSnap, err := fs.Collection("integrations").Doc("flexsave").Collection("configuration").Doc(customerID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}

		return false, err
	}

	if !docSnap.Exists() {
		return false, nil
	}

	var config struct {
		GCP FlexSaveSavingsMetrics `json:"gcp"`
	}

	err = docSnap.DataTo(&config)
	if err := docSnap.DataTo(&config); err != nil {
		return false, nil
	}

	if config.GCP.Enabled {
		return true, nil
	}

	return false, nil
}

// assertGCPContract checks whether the customer has an active gcp contact.
func (s *Service) assertGCPContract(ctx context.Context, customerID string) error {
	fs := s.Firestore(ctx)

	customerRef := fs.Collection("customers").Doc(customerID)

	contractSnaps, err := fs.Collection("contracts").Where("customer", "==", customerRef).Where("type", "==", common.Assets.GoogleCloud).Where("active", "==", true).Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	for i := range contractSnaps {
		var contract common.Contract
		if err := contractSnaps[i].DataTo(&contract); err != nil {
			return err
		}

		if !contract.EndDate.IsZero() && contract.EndDate.Before(now) {
			continue
		}

		hasBillingProfiles, err := s.assertGCPBillingProfiles(ctx, contractSnaps[i].Ref)
		if err != nil && i == len(contractSnaps)-1 {
			return err
		}

		if hasBillingProfiles {
			return nil
		}
	}

	return ErrNoContract
}

// assertGCPBillingProfiles checks whether the customer has google-cloud assets.
func (s *Service) assertGCPBillingProfiles(ctx context.Context, contractRef *firestore.DocumentRef) (bool, error) {
	fs := s.Firestore(ctx)

	assetsSnaps, err := fs.Collection("assets").Where("type", "==", common.Assets.GoogleCloud).Where("contract", "==", contractRef).Documents(ctx).GetAll()
	if err != nil {
		return false, err
	}

	if len(assetsSnaps) == 0 {
		return false, ErrNoBillingProfile
	}

	return true, nil
}

// assertComputeEngineSpend returns whether a specific customer has the required usage days
func (s *Service) assertComputeEngineSpend(ctx context.Context, customerID string) (*SavingsSummary, error) {
	bq := s.Bigquery(ctx)

	query := bq.Query(s.buildFlexSaveGCPUsageDaysQuery(customerID))

	it, err := query.Read(ctx)
	if err != nil {
		return nil, err
	}

	var count int

	for {
		var res struct {
			Count int `bigquery:"count"`
		}

		err = it.Next(&res)
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}

		count = res.Count
	}

	if count < requiredFlexsaveGCPUsageDays {
		return nil, ErrNoSpend
	}

	fs := s.Firestore(ctx)

	docSnap, err := fs.Collection("integrations").Doc("flexsave").Collection("configuration").Doc(customerID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNoSpend
		}

		return nil, err
	}

	var config struct {
		GCP FlexSaveSavingsMetrics `json:"gcp"`
	}

	err = docSnap.DataTo(&config)
	if err != nil {
		return nil, err
	}

	return config.GCP.SavingsSummary, nil
}

func (s *Service) buildFlexSaveGCPUsageDaysQuery(customerID string) string {
	return `
		SELECT
  			COUNT(*) count
		FROM
  		(
    		SELECT
      			DATE(usage_start_time) date_start_time
    		FROM ` + s.flexsaveGCPUsageTable + `
    		WHERE
      			customer_id = "` + customerID + `"
    		GROUP BY
      			date_start_time
		)
	`
}
