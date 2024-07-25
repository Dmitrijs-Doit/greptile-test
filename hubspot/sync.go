package hubspot

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	fsDal "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/firebase/tenant"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type HubspotService struct {
	*logger.Logging
	*connection.Connection
	*tenant.TenantService
	customerTypeDal fsDal.CustomerTypeIface
}

func NewHubspotService(log *logger.Logging, conn *connection.Connection) (*HubspotService, error) {
	tenantService, err := tenant.NewTenantsService(conn)
	if err != nil {
		return nil, err
	}

	return &HubspotService{
		log,
		conn,
		tenantService,
		fsDal.NewCustomerTypeDALWithClient(conn.Firestore(context.Background())),
	}, nil
}

func (s *HubspotService) Sync(ctx context.Context) error {
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

		// Skip customers without billing profile (they don't have any assets, services, invoices etc.) unless they are product only
		if len(customer.Entities) == 0 {
			productOnly, err := s.customerTypeDal.IsProductOnlyCustomerType(ctx, customer.ID)
			if err != nil {
				return err
			}

			if !productOnly {
				continue
			}
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_GET,
			Path:   fmt.Sprintf("/tasks/hubspot/companies/%s", docSnap.Ref.ID),
			Queue:  common.TaskQueueHubspot,
		}

		_, err = common.CreateCloudTask(ctx, &config)
		if err != nil {
			return err
		}

		userDocSnaps, err := s.Firestore(ctx).Collection("users").
			WherePath([]string{"customer", "ref"}, "==", docSnap.Ref).
			Limit(1).Documents(ctx).GetAll()
		if err != nil {
			return err
		}

		// create a contacts task if there is at least 1 user
		if len(userDocSnaps) > 0 {
			scheduleTime := time.Now().UTC().Add(time.Hour * 2)

			config = common.CloudTaskConfig{
				Method:       cloudtaskspb.HttpMethod_GET,
				Path:         fmt.Sprintf("/tasks/hubspot/contacts/%s", docSnap.Ref.ID),
				Queue:        common.TaskQueueHubspot,
				ScheduleTime: common.TimeToTimestamp(scheduleTime),
			}

			_, err = common.CreateCloudTask(ctx, &config)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
