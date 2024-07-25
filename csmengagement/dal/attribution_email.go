package dal

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	alertsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal"
	atrDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	budgetDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
)

type IAttributionEmail interface {
	GetAttributionsByDateRange(ctx context.Context, from time.Time, to time.Time) ([]AttributionData, error)
	IsFirstAttribution(ctx context.Context, attributionID string, coll collab.Collaborator) (bool, error)
	HasBudgets(ctx context.Context, coll collab.Collaborator) (bool, error)
	HasAlerts(ctx context.Context, coll collab.Collaborator) (bool, error)
}

type AttributionEmail struct {
	fs *firestore.Client
}

func NewAttributionEmail(fs *firestore.Client) *AttributionEmail {
	return &AttributionEmail{
		fs: fs,
	}
}

type AttributionData struct {
	AttributionID string
	CustomerID    string
	Collabs       []collab.Collaborator
}

func (a *AttributionEmail) GetAttributionsByDateRange(ctx context.Context, from time.Time, to time.Time) ([]AttributionData, error) {
	docs, err := a.fs.Collection(atrDal.AttributionsCollection).Where(
		"timeCreated", ">=", from,
	).Where(
		"timeCreated", "<=", to,
	).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	data := make([]AttributionData, len(docs))

	for i, doc := range docs {
		atr := &attribution.Attribution{}
		if err := doc.DataTo(atr); err != nil {
			return nil, err
		}

		// presets don't have a customer
		if atr.Customer == nil || atr.Customer.ID == "" {
			continue
		}

		data[i] = AttributionData{
			AttributionID: doc.Ref.ID,
			CustomerID:    atr.Customer.ID,
			Collabs:       atr.Collaborators,
		}
	}

	return data, nil
}

func (a *AttributionEmail) IsFirstAttribution(ctx context.Context, attributionID string, coll collab.Collaborator) (bool, error) {
	docs, err := a.fs.Collection(atrDal.AttributionsCollection).Where(
		"collaborators", "array-contains", coll,
	).Documents(ctx).GetAll()

	if err != nil {
		return false, err
	}

	if len(docs) == 0 {
		return false, fmt.Errorf("attribution not found %s", attributionID)
	}

	if len(docs) == 1 {
		return true, nil
	}

	var firstAtr *attribution.Attribution

	var firstAtrID string

	for _, doc := range docs {
		atr := &attribution.Attribution{}
		if err := doc.DataTo(atr); err != nil {
			return false, err
		}

		if firstAtr == nil || atr.TimeCreated.Before(firstAtr.TimeCreated) {
			firstAtr = atr
			firstAtrID = doc.Ref.ID
		}
	}

	if firstAtrID != attributionID {
		return false, nil
	}

	return true, nil
}

func (a *AttributionEmail) HasBudgets(ctx context.Context, coll collab.Collaborator) (bool, error) {
	docs, err := a.fs.Collection(budgetDal.BudgetsCollection).Where(
		"collaborators", "array-contains", coll,
	).Limit(1).Documents(ctx).GetAll()

	if err != nil {
		return false, err
	}

	if len(docs) > 0 {
		return true, nil
	}

	return false, nil
}

func (a *AttributionEmail) HasAlerts(ctx context.Context, coll collab.Collaborator) (bool, error) {
	docs, err := a.fs.Collection(alertsDal.AlertsCollection).Where(
		"collaborators", "array-contains", coll,
	).Limit(1).Documents(ctx).GetAll()

	if err != nil {
		return false, err
	}

	if len(docs) > 0 {
		return true, nil
	}

	return false, nil
}
