package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type UserService struct {
	loggerProvider logger.Provider
	firestoreFn    connection.FirestoreFromContextFun
}

func NewUserService(loggerProvider logger.Provider, firestoreFn connection.FirestoreFromContextFun) *UserService {
	return &UserService{
		loggerProvider,
		firestoreFn,
	}
}

// DoitMigration migrates a Doer cloud analytics resources from the doit-intl.com domain
// to the new doit.com domain
func (s *UserService) DoitMigration(ctx context.Context, email string) error {
	p := strings.Split(email, "@")
	if len(p) != 2 {
		return errors.New("invalid email")
	}

	if p[1] != "doit.com" {
		return errors.New("invalid doit domain email")
	}

	oldEmail := fmt.Sprintf("%s@doit-intl.com", p[0])
	newEmail := email

	// Migrate cloud analytics objects
	if err := s.doitMigrateCloudAnalytics(ctx, oldEmail, newEmail); err != nil {
		return err
	}

	// Migrate doer roles
	if err := s.doitMigrateRoles(ctx, oldEmail, newEmail); err != nil {
		return err
	}

	// Add here other calls to migrate other objects if needed, for example Account manager object.

	return nil
}

func (s *UserService) doitMigrateCloudAnalytics(ctx context.Context, oldEmail, newEmail string) error {
	fs := s.firestoreFn(ctx)
	logger := s.loggerProvider(ctx)
	batch := doitFirestore.NewBatchProviderWithClient(fs, 100).Provide(ctx)

	input := collab.UpdateUserEmailInput{
		OldEmail: oldEmail,
		NewEmail: newEmail,
		Collections: []string{
			"dashboards/google-cloud-reports/savedReports",
			"dashboards/google-cloud-reports/attributions",
			"cloudAnalytics/alerts/cloudAnalyticsAlerts",
			"cloudAnalytics/budgets/cloudAnalyticsBudgets",
		},
		CustomerID: "", // update objects in all customers tenants
	}

	return collab.NewCollab().UpdateCollabEmail(ctx, logger, fs, batch, input)
}

// doitMigrateRoles will migrate the old user email domain in the doit roles collection to the new email dp,aom
func (s *UserService) doitMigrateRoles(ctx context.Context, oldEmail, newEmail string) error {
	fs := s.firestoreFn(ctx)
	batch := doitFirestore.NewBatchProviderWithClient(fs, 100).Provide(ctx)

	docSnaps, err := fs.Collection("app/doit-employees/doitRoles").
		Where("users", "array-contains", oldEmail).
		Documents(ctx).
		GetAll()
	if err != nil {
		return err
	}

	for _, docSnap := range docSnaps {
		docRef := docSnap.Ref

		if err := batch.Update(ctx, docRef, []firestore.Update{
			{
				Path:  "users",
				Value: firestore.ArrayUnion(newEmail),
			},
		}); err != nil {
			return err
		}

		if err := batch.Update(ctx, docRef, []firestore.Update{
			{
				Path:  "users",
				Value: firestore.ArrayRemove(oldEmail),
			},
		}); err != nil {
			return err
		}
	}

	return batch.Commit(ctx)
}
