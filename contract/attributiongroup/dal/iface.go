package dal

import (
	"context"

	"cloud.google.com/go/firestore"
)

type AttributionGroup interface {
	GetRampPlanEligibleSpendAttributionGroup(ctx context.Context) ([]*firestore.DocumentSnapshot, error)
}
