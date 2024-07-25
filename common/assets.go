package common

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
)

type AssetType string

const (
	AssetTypeResold     AssetType = "resold"
	AssetTypeStandalone AssetType = "standalone"
)

type BaseAsset struct {
	// Those fields always need to be set, even if with no value
	AssetType string                 `firestore:"type"`
	Bucket    *firestore.DocumentRef `firestore:"bucket"`
	Contract  *firestore.DocumentRef `firestore:"contract"`
	Entity    *firestore.DocumentRef `firestore:"entity"`
	Customer  *firestore.DocumentRef `firestore:"customer"`
	Tags      []string               `firestore:"tags"`
	ID        string                 `firestore:"-"`
}

type AssetSettings struct {
	BaseAsset
	Settings    interface{} `firestore:"settings"`
	TimeCreated time.Time   `firestore:"timeCreated,omitempty"`
}

// GetAssetEntity : If there is only one assignable entity for an asset, return that entity
func GetAssetEntity(ctx *gin.Context, fs *firestore.Client, asCustomerRef, asEntityRef, customerRef *firestore.DocumentRef, cache map[string][]*firestore.DocumentRef) (*firestore.DocumentRef, bool) {
	if asCustomerRef != nil && asCustomerRef.Path == customerRef.Path {
		if asEntityRef == nil {
			if entities, err := getCustomerActiveEntities(ctx, fs, customerRef, cache); err != nil {
				fmt.Println("[ERROR] GetAssetEntity", err)
			} else if len(entities) == 1 {
				return entities[0], true
			} else {
				return nil, true
			}
		}
	} else {
		if entities, err := getCustomerActiveEntities(ctx, fs, customerRef, cache); err != nil {
			fmt.Println("[ERROR] GetAssetEntity", err)
		} else if len(entities) == 1 {
			return entities[0], true
		} else {
			return nil, true
		}
	}

	return nil, false
}

func getCustomerActiveEntities(ctx *gin.Context, fs *firestore.Client, customerRef *firestore.DocumentRef, cache map[string][]*firestore.DocumentRef) ([]*firestore.DocumentRef, error) {
	if cache != nil {
		if entities, ok := cache[customerRef.ID]; ok {
			return entities, nil
		}
	}

	entities := make([]*firestore.DocumentRef, 0)

	entitiesIterator := fs.Collection("entities").Where("customer", "==", customerRef).Where("active", "==", true).Documents(ctx)
	defer entitiesIterator.Stop()

	for {
		docSnap, err := entitiesIterator.Next()
		if err == iterator.Done {
			if cache != nil {
				cache[customerRef.ID] = entities
			}

			return entities, nil
		} else if err != nil {
			return nil, err
		}

		entities = append(entities, docSnap.Ref)
	}
}

type IAssetContract interface {
	GetCacheKey() string
	ModifyContractQuery(query *firestore.Query) firestore.Query
	ContractPredicate(contract *Contract) (match bool, save bool)
}

func GetAssetContract(ctx context.Context, fs *firestore.Client, asset IAssetContract, customerRef, entityRef *firestore.DocumentRef, cache map[string]*firestore.DocumentRef) (*firestore.DocumentRef, bool) {
	if entityRef == nil {
		return nil, true
	}

	cacheID := asset.GetCacheKey()
	if cache != nil {
		if contract, ok := cache[cacheID]; ok {
			return contract, true
		}
	}

	now := time.Now().UTC()

	query := fs.Collection("contracts").
		OrderBy("startDate", firestore.Desc).
		Where("startDate", "<=", now).
		Where("active", "==", true).
		Where("customer", "==", customerRef).
		Where("entity", "==", entityRef)

	query = asset.ModifyContractQuery(&query)

	docSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		fmt.Println("[ERROR] GetAssetContract", err)
		return nil, false
	}

	if len(docSnaps) == 0 {
		if cache != nil {
			cache[cacheID] = nil
		}

		return nil, true
	}

	for _, docSnap := range docSnaps {
		var contract Contract
		if err := docSnap.DataTo(&contract); err != nil {
			fmt.Println("[ERROR] GetAssetContract", err)
			return nil, false
		}

		if match, save := asset.ContractPredicate(&contract); match {
			if contract.IsCommitment {
				if contract.EndDate.IsZero() {
					fmt.Println("[ERROR] GetAssetContract", "invalid contract end date", docSnap.Ref.ID)
					return nil, false
				}

				if now.Before(contract.EndDate) {
					if save && cache != nil {
						cache[cacheID] = docSnap.Ref
					}

					return docSnap.Ref, true
				}
			} else {
				if save && cache != nil {
					cache[cacheID] = docSnap.Ref
				}

				return docSnap.Ref, true
			}
		}
	}

	return nil, true
}
