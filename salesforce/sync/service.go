package sync

import (
	"context"
	"fmt"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	customerPKG "github.com/doitintl/hello/scheduled-tasks/salesforce/sync/customer"
)

type Service struct {
	*logger.Logging
	*connection.Connection
	*customerPKG.Service
}

func NewService(log *logger.Logging, conn *connection.Connection) (*Service, error) {
	return &Service{
		log,
		conn,
		customerPKG.NewService(log, conn),
	}, nil
}

func (s *Service) Sync(ctx context.Context) error {
	docSnaps, err := s.Firestore(ctx).Collection("customers").
		Documents(ctx).GetAll()
	if err != nil {
		s.Logger(ctx).Printf("could not retrieve customers. error %s", err)
		return err
	}

	for _, docSnap := range docSnaps {
		var customer common.Customer
		if err := docSnap.DataTo(&customer); err != nil {
			return err
		}

		// Skip customers without billing profile (they don't have any assets, services, invoices etc.) unless they are saas
		if len(customer.Entities) == 0 && (customer.Type == nil || *customer.Type != "standalone") {
			continue
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_GET,
			Path:   fmt.Sprintf("/tasks/salesforce/customer/%s", docSnap.Ref.ID),
			Queue:  common.TaskQueueSalesforce,
		}

		_, err = common.CreateCloudTask(ctx, &config)
		if err != nil {
			return err
		}
	}

	return nil
}
