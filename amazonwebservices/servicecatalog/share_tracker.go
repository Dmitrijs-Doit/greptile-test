package servicecatalog

import "context"

type ShareTracker interface {
	SaveAccountID(ctx context.Context, accountID string) error
	ListAccountIDs(ctx context.Context) (map[string]bool, error)
}

type FSShareTracker struct {
	collPath   string
	fsProvider fsProvider
}

func (st FSShareTracker) SaveAccountID(ctx context.Context, accountID string) error {
	fs := st.fsProvider(ctx)
	_, err := fs.Collection(st.collPath).Doc(accountID).Set(ctx, map[string]interface{}{})

	return err
}

func (st FSShareTracker) ListAccountIDs(ctx context.Context) (map[string]bool, error) {
	fs := st.fsProvider(ctx)
	accountIDs := make(map[string]bool)

	docs, err := fs.Collection(st.collPath).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, doc := range docs {
		accountIDs[doc.Ref.ID] = true
	}

	return accountIDs, nil
}
