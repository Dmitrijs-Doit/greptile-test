package scripts

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

type shouldSetCustomerTierFn func(customerID string, tierData *pkg.CustomerTierData) bool
type newTierRefFn func(customerID string) *firestore.DocumentRef

type SetCustomersTierRequest struct {
	Package string `json:"package"`
	Trial   bool   `json:"trial"`
	DryRun  bool   `json:"dryRun"`
}

type TierName string

const (
	heritageTierName   = "last10"
	standardTierName   = "standard"
	enhancedTierName   = "enhanced"
	premiumTierName    = "premium"
	enterpriseTierName = "enterprise"
)

var paidTiersNames = []string{heritageTierName, standardTierName, enhancedTierName, premiumTierName, enterpriseTierName}

// customers that have already purchased a subscription
var purchasedTiers = map[string]map[string]string{
	string(pkg.NavigatorPackageTierType): {
		"TzQOGuWJuK7junJo2ZSn": standardTierName,
		"J6FN75wg4CzKQkKY3K1v": standardTierName,
		"GnnFMff9PzV9jH1UueMA": standardTierName,
		"btdgoI4wRaXSmOOUHDtK": enhancedTierName,
		"fOjoMysJgzX0eP2kKZnN": enterpriseTierName,
		"24zPfOExSds0ETyQbO7t": premiumTierName,
		"fSjxgVOn6vkMJgwbdPGt": premiumTierName,
	},
	string(pkg.SolvePackageTierType): {
		"TzQOGuWJuK7junJo2ZSn": enhancedTierName,
		"btdgoI4wRaXSmOOUHDtK": enhancedTierName,
		"TJotky6uRyOBMqpKQd27": enhancedTierName,
		"24zPfOExSds0ETyQbO7t": enhancedTierName,
		"J6FN75wg4CzKQkKY3K1v": enhancedTierName,
		"fW1FPEHg0f6Sv90kee0m": enhancedTierName,
		"fSjxgVOn6vkMJgwbdPGt": enhancedTierName,
		"GnnFMff9PzV9jH1UueMA": enhancedTierName,
	},
}

func getTierDataPath(packageType string) string {
	return fmt.Sprintf("tiers.%s", packageType)
}

func SetCustomersTier(ctx *gin.Context) []error {
	params, err := getRequestParams(ctx)
	if err != nil {
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	if params.Trial {
		return setCustomersTrialTier(ctx, fs, params)
	}

	return setCustomersHeritageTier(ctx, fs, params)
}

func setCustomersTrialTier(ctx *gin.Context, fs *firestore.Client, params *SetCustomersTierRequest) []error {
	trialTierDoc, err := fs.Collection("tiers").
		Where("trialTier", "==", true).
		Where("packageType", "==", params.Package).
		Documents(ctx).Next()
	if err != nil {
		return []error{err}
	}

	customers, err := fs.Collection("customers").Where(fmt.Sprintf("%s.trialStartDate", getTierDataPath(params.Package)), "!=", "").Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	fmt.Println("Number of customers found", len(customers))

	return updateCustomersTierRef(ctx, fs, customers,
		func(customerID string) *firestore.DocumentRef {
			return trialTierDoc.Ref
		}, params.Package,
		func(customerID string, tierData *pkg.CustomerTierData) bool {
			if tierData.Tiers[params.Package].Tier != nil {
				fmt.Printf("CustomerID %s: Tier reference already set to %s", customerID, tierData.Tiers[params.Package].Tier.ID)
				return false
			}

			if tierData.Tiers[params.Package].TrialCanceledDate != nil {
				fmt.Printf("CustomerID %s: Trial canceled on %s", customerID, tierData.Tiers[params.Package].TrialCanceledDate)
				return false
			}

			return true
		}, params.DryRun)
}

func setCustomersHeritageTier(ctx *gin.Context, fs *firestore.Client, params *SetCustomersTierRequest) []error {
	allPackageTierSnaps, err := fs.Collection("tiers").
		Where("name", "in", paidTiersNames).
		Where("packageType", "==", params.Package).
		Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	allPackageTiers := make(map[string]*firestore.DocumentSnapshot)
	for _, snap := range allPackageTierSnaps {
		tierName := snap.Data()["name"].(string)

		allPackageTiers[tierName] = snap
	}

	customers, err := fs.Collection("customers").Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	return updateCustomersTierRef(ctx, fs, customers,
		func(customerID string) *firestore.DocumentRef {
			if tierName, ok := purchasedTiers[params.Package][customerID]; ok {
				return allPackageTiers[tierName].Ref
			}
			return allPackageTiers[heritageTierName].Ref
		}, params.Package,
		func(customerID string, tierData *pkg.CustomerTierData) bool {
			return tierData.Tiers == nil || tierData.Tiers[params.Package] == nil ||
				(tierData.Tiers[params.Package].Tier == nil && tierData.Tiers[params.Package].TrialStartDate == nil &&
					tierData.Tiers[params.Package].TrialEndDate == nil)
		}, params.DryRun)
}

func getRequestParams(ctx *gin.Context) (*SetCustomersTierRequest, error) {
	var params SetCustomersTierRequest
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

func updateCustomersTierRef(ctx context.Context, fs *firestore.Client, customers []*firestore.DocumentSnapshot, tierRefFn newTierRefFn, packageType string, filterFn shouldSetCustomerTierFn, dryRun bool) []error {
	errs := []error{}

	if tierRefFn == nil {
		return []error{errors.New("no tier reference callback provided")}
	}

	batch := fb.NewAutomaticWriteBatch(fs, 250)

	for _, customer := range customers {
		var tierData pkg.CustomerTierData
		err := customer.DataTo(&tierData)
		if err != nil {
			return []error{err}
		}

		if filterFn != nil && !filterFn(customer.Ref.ID, &tierData) {
			fmt.Printf("CustomerID %s: Skipping, already assinged tier: %s\n", customer.Ref.ID, tierData.Tiers[packageType].Tier.ID)
			continue
		}

		tierRef := tierRefFn(customer.Ref.ID)

		if tierRef == nil {
			continue
		}

		if dryRun {
			fmt.Printf("CustomerID %s: Setting tier reference to %s\n", customer.Ref.ID, tierRef.ID)
			continue
		}

		batch.Update(customer.Ref, []firestore.Update{
			{Path: fmt.Sprintf("%s.tier", getTierDataPath(packageType)), Value: tierRef},
		})
	}

	if newErrs := batch.Commit(ctx); len(newErrs) > 0 {
		errs = append(errs, newErrs...)
	}

	return errs
}
