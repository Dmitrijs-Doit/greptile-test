package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	ds "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/utils"
)

func (d *ServiceAccountsFirestore) GetProjectsRef(ctx context.Context) *firestore.DocumentRef {
	return d.GetOnboardingColRef(ctx).Doc(utils.GetProjectsDocName())
}

func (d *ServiceAccountsFirestore) GetProjects(ctx context.Context) (*ds.Projects, error) {
	project, err := d.GetProjectsRef(ctx).Get(ctx)
	if err != nil {
		return nil, err
	}

	var data ds.Projects

	err = project.DataTo(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (d *ServiceAccountsFirestore) GetCurrentProject(ctx context.Context) (string, error) {
	data, err := d.GetProjects(ctx)
	if err != nil {
		return "", err
	}

	return data.CurrentProject, nil
}

func (d *ServiceAccountsFirestore) SetProjects_w_Transaction(ctx context.Context, fn utils.TransactionFunc, aux interface{}) (interface{}, error) {
	return d.tx.executeTransaction(ctx, d.GetProjectsRef(ctx), fn, aux)
}
