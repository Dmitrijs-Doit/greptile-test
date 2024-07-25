package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/pkg"
)

type RampPlans interface {
	GetRampPlan(ctx context.Context, PlanID string) (*firestore.DocumentSnapshot, error)
	GetAllActiveRampPlans(ctx context.Context) ([]*firestore.DocumentSnapshot, error)
	GetRampPlansByContractID(ctx context.Context, contractID string) ([]*firestore.DocumentSnapshot, error)
	AddRampPlan(ctx context.Context, rampPlan *pkg.RampPlan) (*firestore.DocumentRef, *firestore.WriteResult, error)
}
