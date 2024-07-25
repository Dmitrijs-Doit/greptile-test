package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	ds "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/utils"
)

func (d *ServiceAccountsFirestore) GetEnvStatusRef(ctx context.Context) *firestore.DocumentRef {
	return d.GetOnboardingColRef(ctx).Doc(utils.EnvStatusDoc)
}

func (d *ServiceAccountsFirestore) GetEnvStatus(ctx context.Context) (*ds.EnvStatus, error) {
	envStatus, err := d.GetEnvStatusRef(ctx).Get(ctx)
	if err != nil {
		return nil, err
	}

	var data ds.EnvStatus

	err = envStatus.DataTo(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (d *ServiceAccountsFirestore) SetEnvStatus(ctx context.Context, e *ds.EnvStatus) error {
	_, err := d.documentsHandler.Set(ctx, d.GetEnvStatusRef(ctx), e)
	return err
}
