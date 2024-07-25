package service

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	"github.com/doitintl/cloudtasks/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/entity/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type EntitiesService struct {
	loggerProvider logger.Provider
	*connection.Connection
	entityDal       dal.Entites
	cloudTaskClient iface.CloudTaskClient
}

func NewEntitiesService(log logger.Provider, conn *connection.Connection) *EntitiesService {
	return &EntitiesService{
		log,
		conn,
		dal.NewEntitiesFirestoreWithClient(conn.Firestore),
		conn.CloudTaskClient,
	}
}

func (s *EntitiesService) SyncEntitiesInvoiceAttributions(ctx context.Context, forceUpdate bool) error {
	logger := s.loggerProvider(ctx)

	entities, err := s.entityDal.GetEntities(ctx)
	if err != nil {
		return err
	}

	for _, entity := range entities {
		if !shouldUpdate(ctx, entity, forceUpdate) {
			continue
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf("/tasks/analytics/customers/%s/entities/%s/sync-invoice-attributions", entity.Customer.ID, entity.Snapshot.Ref.ID),
			Queue:  common.TaskQueueEntityInvoiceAttributionsSync,
		}
		conf := config.Config(nil)

		if _, err = s.cloudTaskClient.CreateTask(ctx, conf); err != nil {
			logger.Errorf("failed to create invoice attributions synchronization task for entity %s with error: %s", entity.Snapshot.Ref.ID, err)
			continue
		}
	}

	return nil
}

func shouldUpdate(ctx context.Context, e *common.Entity, forceUpdate bool) bool {
	customerSnap, err := e.Customer.Get(ctx)
	if err != nil {
		return false
	}

	classification, err := customerSnap.DataAt("classification")
	if err != nil || classification.(string) == string(common.CustomerClassificationTerminated) {
		return false
	}

	twentyFourHours := 24 * time.Hour

	return forceUpdate || e.Snapshot.UpdateTime.After(time.Now().UTC().Add(-twentyFourHours)) // is within the last 24 hours
}
