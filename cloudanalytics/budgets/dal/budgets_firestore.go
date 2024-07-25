package dal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	BudgetsCollection    = "cloudAnalytics/budgets/cloudAnalyticsBudgets"
	BudgetsNotifications = "cloudAnalyticsBudgetsNotifications"
	customersCollection  = "customers"
)

// BudgetsFirestore is used to interact with cloud analytics budgets stored on Firestore.
type BudgetsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewBudgetsFirestoreWithClient returns a new BudgetsFirestore using given client.
func NewBudgetsFirestoreWithClient(fun connection.FirestoreFromContextFun) *BudgetsFirestore {
	return &BudgetsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *BudgetsFirestore) GetRef(ctx context.Context, budgetID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(BudgetsCollection).Doc(budgetID)
}

func (d *BudgetsFirestore) GetBudget(ctx context.Context, budgetID string) (*budget.Budget, error) {
	if budgetID == "" {
		return nil, errors.New("missing budget id")
	}

	doc := d.GetRef(ctx, budgetID)

	snap, err := d.documentsHandler.Get(ctx, doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var b budget.Budget

	if err := snap.DataTo(&b); err != nil {
		return nil, err
	}

	return &b, nil
}

func (d *BudgetsFirestore) Share(ctx context.Context, budgetID string, collaborators []collab.Collaborator, public *collab.PublicAccess) error {
	budgetRef := d.GetRef(ctx, budgetID)

	if _, err := budgetRef.Update(ctx, []firestore.Update{
		{
			FieldPath: []string{"collaborators"},
			Value:     collaborators,
		}, {
			FieldPath: []string{"public"},
			Value:     public,
		}}); err != nil {
		return err
	}

	return nil
}

func (d *BudgetsFirestore) UpdateBudgetRecipients(ctx context.Context, budgetID string, newRecipients []string, newRecipientsSlackChannels []common.SlackChannel) error {
	budgetRef := d.GetRef(ctx, budgetID)

	if newRecipients == nil {
		newRecipients = []string{}
	}

	if _, err := budgetRef.Update(ctx, []firestore.Update{
		{
			FieldPath: []string{"recipients"},
			Value:     newRecipients,
		}, {
			FieldPath: []string{"recipientsSlackChannels"},
			Value:     newRecipientsSlackChannels,
		}}); err != nil {
		return err
	}

	return nil
}

func (d *BudgetsFirestore) UpdateBudgetEnforcedByMetering(ctx context.Context, budgetID string, enforcedByMetering bool) error {
	fs := d.firestoreClientFun(ctx)

	return fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		budgetRef := d.GetRef(ctx, budgetID)

		doc, err := tx.Get(budgetRef)
		if err != nil {
			return err
		}

		currentEnforcedByMetering, err := doc.DataAt("enforcedByMetering")
		if err == nil && currentEnforcedByMetering == enforcedByMetering {
			return nil
		}

		return tx.Update(budgetRef, []firestore.Update{
			{
				FieldPath: []string{"enforcedByMetering"},
				Value:     enforcedByMetering,
			}})
	})
}

func (d *BudgetsFirestore) ListBudgets(ctx context.Context, args *ListBudgetsArgs) ([]budget.Budget, error) {
	var docSnaps []*firestore.DocumentSnapshot

	var err error

	customerRef := d.firestoreClientFun(ctx).Collection(customersCollection).Doc(args.CustomerID)
	customerQuery := d.firestoreClientFun(ctx).
		Collection(BudgetsCollection).
		Where("customer", "==", customerRef)

	customerQuery = addTimeCreatedFilter(customerQuery, args)

	if args.IsDoitEmployee {
		iter := customerQuery.Documents(ctx)
		docSnaps, err = iter.GetAll()
	} else {
		sharedBudgetsQuery := customerQuery.Where("public", "==", nil).
			Where("collaborators", common.ArrayContainsAny, []collab.Collaborator{
				{Email: args.Email, Role: collab.CollaboratorRoleOwner},
				{Email: args.Email, Role: collab.CollaboratorRoleEditor},
				{Email: args.Email, Role: collab.CollaboratorRoleViewer},
			})

		publicBudgetsQuery := customerQuery.Where("public", common.In,
			[]collab.PublicAccess{collab.PublicAccessView, collab.PublicAccessEdit})

		docSnaps, err = firebase.ExecuteQueries(ctx, []firestore.Query{publicBudgetsQuery, sharedBudgetsQuery})
	}

	if err != nil {
		return nil, err
	}

	var items []budget.Budget

	for _, doc := range docSnaps {
		var item budget.Budget
		if err := doc.DataTo(&item); err != nil {
			return nil, err
		}

		item.ID = doc.Ref.ID
		items = append(items, item)
	}

	return items, nil
}

func (d *BudgetsFirestore) SaveNotification(ctx context.Context, notification *budget.BudgetNotification) error {
	docID := fmt.Sprintf("%s_%s", notification.BudgetID, notification.Type)
	ref := d.GetRef(ctx, notification.BudgetID).Collection(BudgetsNotifications).Doc(docID)

	if notification.Type == budget.BudgetNotificationTypeForecast && notification.ForcastedDate == nil {
		return errors.New("forecasted date is required for forecast notification")
	}

	fs := d.firestoreClientFun(ctx)

	return fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		created := notification.AlertDate

		doc, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}
		} else {
			createdFromField, err := doc.DataAt("created")
			if err != nil {
				return err
			}

			createdTime, ok := createdFromField.(time.Time)
			if !ok {
				return errors.New("created field is not a time.Time")
			}

			created = createdTime
		}

		notification.Created = &created
		err = tx.Set(ref, notification)

		return err

	})
}

func addTimeCreatedFilter(query firestore.Query, args *ListBudgetsArgs) firestore.Query {
	if args.MinCreationTime != nil {
		query = query.Where("timeCreated", ">=", args.MinCreationTime)
	}

	if args.MaxCreationTime != nil {
		query = query.Where("timeCreated", "<=", args.MaxCreationTime)
	}

	return query.OrderBy("timeCreated", firestore.Desc)
}

func (d *BudgetsFirestore) GetByCustomerAndAttribution(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	attrRef *firestore.DocumentRef,
) ([]*budget.Budget, error) {
	allDocs, err := d.firestoreClientFun(ctx).Collection(BudgetsCollection).
		Where("customer", "==", customerRef).
		Where("attributions", common.ArrayContains, attrRef).Documents(ctx).GetAll()

	if err != nil {
		return nil, err
	}

	var budgets []*budget.Budget

	for _, doc := range allDocs {
		var budget *budget.Budget
		if err := doc.DataTo(&budget); err != nil {
			return nil, err
		}

		budget.ID = doc.Ref.ID

		budgets = append(budgets, budget)
	}

	return budgets, nil
}
