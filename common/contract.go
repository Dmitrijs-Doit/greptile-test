package common

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
)

type Contract struct {
	Type               string                     `firestore:"type"`
	Customer           *firestore.DocumentRef     `firestore:"customer"`
	Entity             *firestore.DocumentRef     `firestore:"entity"`
	Assets             []*firestore.DocumentRef   `firestore:"assets"`
	Active             bool                       `firestore:"active"`
	IsCommitment       bool                       `firestore:"isCommitment"`
	CommitmentPeriods  []ContractCommitmentPeriod `firestore:"commitmentPeriods"`
	CommitmentRollover bool                       `firestore:"commitmentRollover"`
	PLPSPercent        float64                    `firestore:"plpsPercent"`
	Discount           float64                    `firestore:"discount"`
	DiscountEndDate    *time.Time                 `firestore:"discountEndDate"`
	StartDate          time.Time                  `firestore:"startDate"`
	EndDate            time.Time                  `firestore:"endDate"`
	Properties         map[string]interface{}     `firestore:"properties"`
}

type ContractCommitmentPeriod struct {
	Value     float64   `firestore:"value"`
	StartDate time.Time `firestore:"startDate"`
	EndDate   time.Time `firestore:"endDate"`
	Discount  float64   `firestore:"discount"`
}

// ShouldUseCommitmentPeriodDiscounts checks if a contract should use variable commitment period
// discounts. Only relevant for Commitment GCP contracts that has at least one value of
// commitment period discount
func (c *Contract) ShouldUseCommitmentPeriodDiscounts() bool {
	if c.IsCommitment && c.Type == Assets.GoogleCloud && len(c.CommitmentPeriods) > 0 {
		for _, cp := range c.CommitmentPeriods {
			if cp.Discount != 0 {
				return true
			}
		}
	}

	return false
}

// GetFloatProperty extract float64 value from contract properties
func (c *Contract) GetFloatProperty(key string, defaultValue float64) (float64, bool) {
	if c.Properties != nil && len(c.Properties) > 0 {
		if v, prs := c.Properties[key]; prs {
			switch t := v.(type) {
			case int64:
				return float64(t), true
			case float64:
				return float64(t), true
			default:
			}
		}
	}

	return defaultValue, false
}

// GetBoolProperty extract bool value from contract properties
func (c *Contract) GetBoolProperty(key string, defaultValue bool) (bool, bool) {
	if c.Properties != nil && len(c.Properties) > 0 {
		if v, prs := c.Properties[key]; prs {
			switch t := v.(type) {
			case bool:
				return bool(t), true
			default:
			}
		}
	}

	return defaultValue, false
}

func GetContract(ctx context.Context, ref *firestore.DocumentRef) (*Contract, error) {
	if ref == nil {
		return nil, errors.New("invalid nil contract ref")
	}

	docSnap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	var contract Contract
	if err := docSnap.DataTo(&contract); err != nil {
		return nil, err
	}

	return &contract, nil
}

func IsThereActiveSignedContract(ctx context.Context, customerID string, entityID string, typeFilter []string) (bool, error) {
	customerRef := fs.Collection("customers").Doc(customerID)
	entityRef := fs.Collection("entities").Doc(entityID)

	docs, err := fs.Collection("contracts").
		Where("customer", "==", customerRef).
		Where("entity", "==", entityRef).
		Where("active", "==", true).
		Where("type", "in", typeFilter).
		Documents(ctx).GetAll()
	if err != nil {
		return false, err
	}

	return len(docs) > 0, nil
}
