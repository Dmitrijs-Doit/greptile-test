package dal

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/utils"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// OnBoardingFirestore
type OnBoardingFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
	tx                 *firestoreTransaction
}

func NewOnBoardingFirestoreWithClient(log logger.Provider, conn *connection.Connection) *OnBoardingFirestore {
	return &OnBoardingFirestore{
		firestoreClientFun: conn.Firestore,
		documentsHandler:   doitFirestore.DocumentHandler{},
		tx:                 newFirestoreTransaction(log, conn),
	}
}

func (d *OnBoardingFirestore) GetOnboardingColRef(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(utils.AppCollection).Doc(utils.GCPFlexsaveStandaloneDoc).Collection(utils.OnboardingCollection)
}

func (d *OnBoardingFirestore) GetProjectsRef(ctx context.Context) *firestore.DocumentRef {
	return d.GetOnboardingColRef(ctx).Doc(utils.GetProjectsDocName())
}

func (d *OnBoardingFirestore) GetServiceAccountRef(ctx context.Context) *firestore.DocumentRef {
	return d.GetOnboardingColRef(ctx).Doc(utils.GetServiceAccountsDocName())
}

func (d *OnBoardingFirestore) GetEnvStatusRef(ctx context.Context) *firestore.DocumentRef {
	return d.GetOnboardingColRef(ctx).Doc(utils.EnvStatusDoc)
}

func (d *OnBoardingFirestore) GetProjects(ctx context.Context) (*dataStructures.Projects, error) {
	project, err := d.GetProjectsRef(ctx).Get(ctx)
	if err != nil {
		return nil, err
	}

	var data dataStructures.Projects

	err = project.DataTo(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (d *OnBoardingFirestore) GetCurrentProject(ctx context.Context) (string, error) {
	data, err := d.GetProjects(ctx)
	if err != nil {
		return "", err
	}

	return data.CurrentProject, nil
}

func (d *OnBoardingFirestore) GetServiceAccountsPool(ctx context.Context) (*dataStructures.ServiceAccountsPool, error) {
	doc, err := d.documentsHandler.Get(ctx, d.GetServiceAccountRef(ctx))
	if err != nil {
		return nil, err
	}

	var pool dataStructures.ServiceAccountsPool
	if err := doc.DataTo(&pool); err != nil {
		return nil, err
	}

	return &pool, nil
}

func (d *OnBoardingFirestore) AddNewProject(ctx context.Context, projectID string) error {
	projectsRef := d.GetProjectsRef(ctx)

	data, err := d.GetProjects(ctx)
	if err != nil {
		return err
	}

	data.Projects[projectID] = 0

	_, err = projectsRef.Set(ctx, data)

	if err != nil {
		return err
	}

	return nil
}

func (d *OnBoardingFirestore) SetServiceAccountsPool(ctx context.Context, p *dataStructures.ServiceAccountsPool) error {
	_, err := d.documentsHandler.Set(ctx, d.GetServiceAccountRef(ctx), p)
	return err
}

func (d *OnBoardingFirestore) SetServiceAccountsPool_w_Transaction(ctx context.Context, fn utils.TransactionFunc, aux interface{}) (interface{}, error) {
	return d.tx.executeTransaction(ctx, d.GetServiceAccountRef(ctx), fn, aux)
}

func (d *OnBoardingFirestore) GetNextFreeServiceAccount(ctx context.Context, customerRef *firestore.DocumentRef) (saEmail string, shouldCreateNewSA bool, err error) {
	err = d.firestoreClientFun(ctx).RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		var pool *dataStructures.ServiceAccountsPool

		docRef, err := tx.Get(d.GetServiceAccountRef(ctx))
		if err != nil {
			return err
		}

		if docRef.DataTo(&pool) != nil {
			return err
		}

		var saMetadata dataStructures.ServiceAccountMetadata

		for key, value := range pool.Free {
			saEmail = key
			saMetadata = value

			break
		}

		saMetadata.Customer = customerRef

		delete(pool.Free, saEmail)

		if len(pool.Reserved) == 0 {
			pool.Reserved = make(map[string]dataStructures.ServiceAccountMetadata)
		}

		pool.Reserved[saEmail] = saMetadata

		shouldCreateNewSA = false
		if len(pool.Free) <= utils.FreeServiceAccountsThreshold {
			shouldCreateNewSA = true
		}

		err = tx.Set(docRef.Ref, pool)
		if err != nil {
			return err
		}

		return nil
	}, firestore.MaxAttempts(20))
	if err != nil {
		return "", false, err
	}

	return saEmail, shouldCreateNewSA, nil
}

func (d *OnBoardingFirestore) SetProjects_w_Transaction(ctx context.Context, fn utils.TransactionFunc, aux interface{}) (interface{}, error) {
	return d.tx.executeTransaction(ctx, d.GetProjectsRef(ctx), fn, aux)
}

func (d *OnBoardingFirestore) GetEnvStatus(ctx context.Context) (*dataStructures.EnvStatus, error) {
	envStatus, err := d.GetEnvStatusRef(ctx).Get(ctx)
	if err != nil {
		return nil, err
	}

	var data dataStructures.EnvStatus

	err = envStatus.DataTo(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (d *OnBoardingFirestore) SetEnvStatus(ctx context.Context, e *dataStructures.EnvStatus) error {
	_, err := d.documentsHandler.Set(ctx, d.GetEnvStatusRef(ctx), e)
	return err
}
