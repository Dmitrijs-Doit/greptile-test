package service

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"google.golang.org/api/iterator"
)

//go:generate mockery --name CSMEngagement --output ./mocks
type CSMEngagement interface {
	GetCustomerMRR(ctx context.Context, customerID string, excludeStandalone bool) (float64, error)
	GetNewCustomersIDs(ctx context.Context, createdAfter time.Time) ([]string, error)
	CustomerHasAssets(ctx context.Context, customerID string) (bool, error)
	IsCustomerResold(ctx context.Context, customerID string) (bool, error)
}

type CsmEngagement struct {
	fs *firestore.Client
}

func NewCsmEngagement(fs *firestore.Client) *CsmEngagement {
	return &CsmEngagement{
		fs: fs,
	}
}

func (d *CsmEngagement) IsCustomerResold(ctx context.Context, customerID string) (bool, error) {
	now := time.Now()
	activeContracts, err := d.fs.Collection("contracts").
		Where("customer", "==", d.fs.Collection("customers").Doc(customerID)).
		Where("active", "==", true).
		Where("startDate", "<=", now).
		Documents(ctx).GetAll()

	if err != nil {
		return false, err
	}

	for _, contract := range activeContracts {
		contractType, err := contract.DataAt("type")
		if err != nil {
			return false, err
		}

		if contractType != common.Assets.AmazonWebServicesStandalone && contractType != common.Assets.GoogleCloudStandalone {
			return true, nil
		}
	}

	return false, nil
}

func (d *CsmEngagement) GetCustomerMRR(ctx context.Context, customerID string, excludeStandalone bool) (float64, error) {
	now := time.Now()
	activeContractsIter := d.fs.Collection("contracts").
		Where("customer", "==", d.fs.Collection("customers").Doc(customerID)).
		Where("active", "==", true).
		Where("startDate", "<=", now).
		Documents(ctx)

	var totalValue float64

	for {
		doc, err := activeContractsIter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return 0, err
		}

		contractType, err := doc.DataAt("type")
		if err != nil {
			return 0, err
		}

		if excludeStandalone && (contractType == common.Assets.AmazonWebServicesStandalone || contractType == common.Assets.GoogleCloudStandalone) {
			continue
		}

		estimatedValue, err := doc.DataAt("estimatedValue")
		if err != nil {
			return 0, err
		}

		estimatedValueInt, ok := estimatedValue.(int64)
		if !ok {
			if estimatedValueFloat, ok := estimatedValue.(float64); ok {
				totalValue += estimatedValueFloat
			}
		} else {
			totalValue += float64(estimatedValueInt)
		}
	}

	return totalValue / 12, nil
}

func (d *CsmEngagement) GetNewCustomersIDs(ctx context.Context, createdAfter time.Time) ([]string, error) {
	iter := d.fs.Collection("customers").
		Where("timeCreated", ">=", createdAfter).
		Documents(ctx)

	var ids []string

	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return nil, err
		}

		ids = append(ids, doc.Ref.ID)
	}

	return ids, nil
}

func (d *CsmEngagement) CustomerHasAssets(ctx context.Context, customerID string) (bool, error) {
	assetsSnaps, err := d.fs.Collection("assets").
		Where("customer", "==", d.fs.Collection("customers").Doc(customerID)).Limit(1).Documents(ctx).GetAll()

	if err != nil {
		return false, err
	}

	if len(assetsSnaps) == 0 {
		return false, nil
	}

	return true, nil
}
