package flexsaveresold

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

// A bare metal instance is the same size as the largest instance within the same instance family.
// For example, the largest instance with in the i3 instance family is the i3.16xlarge, so the the normalization
// factor for i3.metal is the same as that for 16xlarge.
var normalizationFactorsForMetals = map[string]float64{
	"mac1.metal":   24, //3
	"a1.metal":     32, //4
	"m5zn.metal":   96, //12
	"z1d.metal":    96,
	"x2iezn.metal": 96,
	"g4dn.metal":   96,
	"i3.metal":     128, //16
	"c6g.metal":    128,
	"c6gd.metal":   128,
	"c6gn.metal":   128,
	"m6g.metal":    128,
	"m6gd.metal":   128,
	"r6g.metal":    128,
	"r6gd.metal":   128,
	"x2gd.metal":   128,
	"g5g.metal":    128,
	"c5n.metal":    144,
	"c5.metal":     192, //24
	"c5d.metal":    192,
	"i3en.metal":   192,
	"m5.metal":     192,
	"m5d.metal":    192,
	"m5n.metal":    192,
	"m5dn.metal":   192,
	"r5.metal":     192,
	"r5b.metal":    192,
	"r5d.metal":    192,
	"r5n.metal":    192,
	"r5dn.metal":   192,
	"c6i.metal":    256, //32
	"c6id.metal":   256,
	"m6i.metal":    256,
	"m6id.metal":   256,
	"r6i.metal":    256,
	"r6id.metal":   256,
	"x2idn.metal":  256,
	"x2iedn.metal": 256,
	"c6a.metal":    384, //48
	"m6a.metal":    384,
	"r6a.metal":    384,
}

var normalizationFactors = map[string]float64{
	"nano":     0.25,
	"micro":    0.5,
	"small":    1,
	"medium":   2,
	"large":    4,
	"xlarge":   8,
	"2xlarge":  16,
	"3xlarge":  24,
	"4xlarge":  32,
	"5xlarge":  40,
	"6xlarge":  48,
	"7xlarge":  56,
	"8xlarge":  64,
	"9xlarge":  72,
	"10xlarge": 80,
	"12xlarge": 96,
	"16xlarge": 128,
	"18xlarge": 144,
	"24xlarge": 192,
	"32xlarge": 256,
	"48xlarge": 384,
}

func InstanceFamilyNormalizationFactor(orderInstanceType string) (string, float64, error) {
	instanceType := strings.ToLower(strings.TrimSpace(orderInstanceType))
	instance := strings.SplitN(instanceType, ".", 2)

	if len(instance) != 2 {
		return "", 0, fmt.Errorf("invalid instance type: %v", orderInstanceType)
	}

	instanceFamily, instanceSize := instance[0], instance[1]

	if instanceSize == "metal" {
		factor, ok := normalizationFactorsForMetals[instanceType]
		if !ok {
			return instanceFamily, 0, fmt.Errorf("invalid instance size: %v", orderInstanceType)
		}

		return instanceFamily, factor, nil
	}

	factor, ok := normalizationFactors[instanceSize]
	if !ok {
		return instanceFamily, 0, fmt.Errorf("invalid instance size: %v", orderInstanceType)
	}

	return instanceFamily, factor, nil
}

// getDiscounts retrieves rates for the AWS account. The order we use to determine eligible discount is:
//
// - If there is a valid contract attached to this account that contains overrides, use those.
//
// - If there is no valid contract for this asset, but customer has only one active contract for AWS in general, and if that contract has an override, use that.
//
// - If none of the above works, use MPA default for FlexSave override and 0.0 for discount.
//
// We never try to use rates from contracts in other Billing Profiles if there is more than one active contract.
//
// Returns FlexSave discount percentage, On-Demand Discount Rate, error in case something is wrong
func GetDiscounts(ctx context.Context, fs *firestore.Client, asset amazonwebservices.Asset) (float64, float64, error) {
	if asset.Properties.OrganizationInfo == nil {
		return 0, 0, fmt.Errorf("account %v doesn't have payer", asset.Properties.AccountID)
	}

	payerAccountID := asset.Properties.OrganizationInfo.PayerAccount.AccountID

	payerAccounts, err := dal.GetMasterPayerAccountsByPayerIDs(ctx, fs, payerAccountID)
	if err != nil {
		return 0, 0, errors.New("failed to read master payer accounts data")
	}

	payerAccount, ok := payerAccounts.Accounts[payerAccountID]
	if !ok {
		return 0, 0, fmt.Errorf("payer %v for account %v not under doit", payerAccountID, asset.Properties.AccountID)
	}

	var activeContract *common.Contract

	if asset.Contract != nil {
		contractSnap, err := asset.Contract.Get(ctx)
		if err != nil {
			return 0, 0, err
		}

		var contract common.Contract
		if err := contractSnap.DataTo(&contract); err != nil {
			return 0, 0, nil
		}

		now := time.Now().UTC()
		if contract.Active && contract.StartDate.Before(now) && (contract.EndDate.IsZero() || !contract.EndDate.Before(now)) {
			activeContract = &contract
		}
	}

	if activeContract == nil {
		activeContract, err = GetDefaultContract(ctx, fs, asset.Customer)
		if err != nil {
			return 0, 0, err
		}
	}

	mpaDefault := payerAccount.DefaultAwsFlexSaveDiscountRate

	if activeContract != nil {
		percentage, found := activeContract.GetFloatProperty("awsFlexSaveOverwrite", 0.0)
		if !found || percentage == 0.0 {
			percentage = mpaDefault
		}

		return percentage, activeContract.Discount, nil
	}

	return mpaDefault, 0.0, nil
}

// getDefaultContract See if customer has only one active AWS contract that is still valid, which we can treat as default.
// If there are more than one valid AWS contract, we haven't found a default.
// Returns default contract if found, nil if there is nothing we can treat as default.
func GetDefaultContract(ctx context.Context, fs *firestore.Client, customerRef *firestore.DocumentRef) (*common.Contract, error) {
	contracts, err := fs.Collection("contracts").Where("type", "==", "amazon-web-services").Where("customer", "==", customerRef).Where("active", "==", true).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	var validContract *common.Contract

	for _, contractSnap := range contracts {
		var contract common.Contract
		if err := contractSnap.DataTo(&contract); err != nil {
			return nil, err
		}

		if contract.StartDate.After(now) || (!contract.EndDate.IsZero() && contract.EndDate.Before(now)) {
			continue
		}

		// more than one valid AWS contract found, this means there is no default we can use
		if validContract != nil {
			return nil, nil
		}

		validContract = &contract
	}

	return validContract, nil
}

func getApplicableMonths(date time.Time, numberOfMonths int) []string {
	var months []string

	for i := 0; i < numberOfMonths; i++ {
		dateString := formatMonthFromDate(date, -i)
		months = append(months, dateString)
	}

	return months
}

func formatMonthFromDate(date time.Time, monthNumber int) string {
	firstOfNextMonth := time.Date(date.Year(), date.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	month := firstOfNextMonth.AddDate(0, monthNumber, -1).Month()
	year := firstOfNextMonth.AddDate(0, monthNumber, -1).Year()

	return fmt.Sprint(int(month)) + "_" + fmt.Sprint(year)
}
