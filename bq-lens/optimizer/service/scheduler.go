package service

import (
	"context"
	"fmt"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	updateCustomerTaskPathTemplate = "/tasks/bq-lens/optimizer/%s"
	presentationCustomer           = "presentationcustomerAWSAzureGCP"
)

func (s *OptimizerService) Schedule(ctx context.Context) (taskErrors []error, _ error) {
	bqLensCustomers, err := s.cloudConnect.GetBQLensCustomers(ctx)
	if err != nil {
		return taskErrors, err
	}

	// These queries run against our own tables, hence the use
	// of the default BQ client.
	bq := s.conn.Bigquery(ctx)

	customerDiscounts, err := s.serviceBQ.GetCustomerDiscounts(ctx, bq)
	if err != nil {
		return taskErrors, err
	}

	allCustomerBillingProjectsWithEditions, err := s.serviceBQ.GetBillingProjectsWithEditions(ctx, bq)
	if err != nil {
		return taskErrors, err
	}

	for _, customerID := range bqLensCustomers {
		if skipCustomer(customerID) {
			continue
		}

		customerDiscount := 1.0
		if _, ok := customerDiscounts[customerID]; ok {
			customerDiscount = customerDiscounts[customerID]
		}

		projectsWithEditions := allCustomerBillingProjectsWithEditions[customerID]
		if projectsWithEditions == nil {
			projectsWithEditions = []domain.BillingProjectWithReservation{}
		}

		payload := domain.Payload{
			Discount:                      customerDiscount,
			BillingProjectWithReservation: projectsWithEditions,
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf(updateCustomerTaskPathTemplate, customerID),
			Queue:  common.TaskQueueBQLensOptimizer,
			DispatchDeadline: &durationpb.Duration{
				Seconds: 1800,
			},
		}

		if _, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(payload)); err != nil {
			taskErrors = append(taskErrors, fmt.Errorf("failed to create task for customer %s: %w", customerID, err))
		}
	}

	return taskErrors, nil
}

func skipCustomer(customerID string) bool {
	return customerID == presentationCustomer
}
