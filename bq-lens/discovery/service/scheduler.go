package service

import (
	"context"
	"fmt"
	"slices"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/protobuf/types/known/durationpb"

	cloudResourceManagerDomain "github.com/doitintl/cloudresourcemanager/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	maxChunkSize                   = 25
	updateCustomerTaskPathTemplate = "/tasks/bq-lens/discovery/%s"
	doitCustomer                   = "EE8CtpzYiKp0dVAESVrB"
	dreamData                      = "nRWYx1dfdTv9cCl0auRS"
)

type TablesDiscoveryPayload struct {
	Projects []*cloudResourceManagerDomain.Project `json:"projects" validate:"gt=0,required"`
}

func (s *DiscoveryService) Schedule(ctx context.Context) (taskErrors []error, _ error) {
	l := s.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		"house":   "adoption",
		"feature": "bq-lens",
		"module":  "discovery",
		"service": "discovery",
	})

	err := s.processExceptionCustomer(ctx, doitCustomer)
	if err != nil {
		taskErrors = append(taskErrors, err)
	}

	bqLensCustomers, err := s.cloudConnect.GetBQLensCustomers(ctx)
	if err != nil {
		return taskErrors, err
	}

	for _, customerID := range bqLensCustomers {
		if slices.Contains([]string{doitCustomer, dreamData}, customerID) {
			continue
		}

		connect, _, err := s.cloudConnect.NewGCPClients(ctx, customerID)
		if err != nil {
			l.Errorf("failed to create clients for customer %s: %v", customerID, err)
			return
		}

		crm := connect.CRM

		customerProjects, err := s.listCustomerProjects(ctx, crm)
		if err != nil {
			l.Errorf("failed to list customer projects for customer %s: %v", customerID, err)
			return
		}

		payload := TablesDiscoveryPayload{
			Projects: customerProjects,
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf(updateCustomerTaskPathTemplate, customerID),
			Queue:  common.TaskQueueBQLensTablesDiscovery,
			DispatchDeadline: &durationpb.Duration{
				Seconds: 1800,
			},
		}

		if _, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(payload)); err != nil {
			taskErrors = append(taskErrors, fmt.Errorf("failed to create task for customer %s: %w, ", customerID, err))
		}

		// close bq client as it is initialised in the NewGCPClients
		bq := connect.BQ.BigqueryService
		bq.Close()
	}

	err = s.processExceptionCustomer(ctx, dreamData)
	if err != nil {
		taskErrors = append(taskErrors, err)
	}

	return taskErrors, nil
}

func (s *DiscoveryService) processExceptionCustomer(ctx context.Context, customerID string) error {
	connect, _, err := s.cloudConnect.NewGCPClients(ctx, customerID)
	if err != nil {
		return fmt.Errorf("failed to create clients for customer %s: %v", customerID, err)
	}

	crm := connect.CRM

	customerProjects, err := s.listCustomerProjects(ctx, crm)
	if err != nil {
		return fmt.Errorf("failed to list customer projects for customer %s: %v", customerID, err)
	}

	numChunks := len(customerProjects) / maxChunkSize

	// if customer is not doit, we need to send all projects in one chunk
	if customerID != doitCustomer {
		numChunks = 1
	}

	for i := 0; i <= numChunks; i++ {
		start := i * maxChunkSize
		end := (i + 1) * maxChunkSize

		if end > len(customerProjects) {
			end = len(customerProjects)
		}

		chunk := customerProjects[start:end]

		payload := TablesDiscoveryPayload{
			Projects: chunk,
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf(updateCustomerTaskPathTemplate, doitCustomer),
			Queue:  common.TaskQueueBQLensTablesDiscovery,
			DispatchDeadline: &durationpb.Duration{
				Seconds: 1800,
			},
		}

		_, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(payload))
		if err != nil {
			return fmt.Errorf("failed to create task for customer %s: %w. chunk %d/%d, ", customerID, err, i, numChunks)
		}

	}

	// close bq client as it is initialised in the NewGCPClients
	bq := connect.BQ.BigqueryService
	bq.Close()

	return nil
}
