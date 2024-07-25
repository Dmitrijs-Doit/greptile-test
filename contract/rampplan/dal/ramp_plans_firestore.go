package dal

import (
	"context"
	"time"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/contract/dal"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	rampPlansCollection = "rampPlans"
)

type RampPlansFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewRampPlansFirestore returns a new RampPlansFirestore instance with given project id.
func NewRampPlansFirestore(ctx context.Context, projectID string) (*dal.ContractFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return dal.NewContractFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewRampPlansFirestoreWithClient returns a new RampPlansFirestore using given client.
func NewRampPlansFirestoreWithClient(fun connection.FirestoreFromContextFun) *RampPlansFirestore {
	return &RampPlansFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (r *RampPlansFirestore) GetRampPlan(ctx context.Context, PlanID string) (*firestore.DocumentSnapshot, error) {
	return r.firestoreClientFun(ctx).Collection(rampPlansCollection).Doc(PlanID).Get(ctx)
}

func (r *RampPlansFirestore) GetAllActiveRampPlans(ctx context.Context) ([]*firestore.DocumentSnapshot, error) {
	return r.firestoreClientFun(ctx).Collection(rampPlansCollection).Where("origEstEndDate", ">", time.Now()).Documents(ctx).GetAll()
}

func (r *RampPlansFirestore) GetRampPlansByContractID(ctx context.Context, contractID string) ([]*firestore.DocumentSnapshot, error) {
	return r.firestoreClientFun(ctx).Collection("rampPlans").
		Where("contractId", "==", contractID).
		Documents(ctx).GetAll()
}

func (r *RampPlansFirestore) AddRampPlan(ctx context.Context, rampPlan *pkg.RampPlan) (*firestore.DocumentRef, *firestore.WriteResult, error) {
	return r.firestoreClientFun(ctx).Collection("rampPlans").Add(ctx, rampPlan)
}
