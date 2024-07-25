package service

import (
	"context"
	"fmt"
	"strings"

	reservation "cloud.google.com/go/bigquery/reservation/apiv1"
	"google.golang.org/api/option"

	crmIface "github.com/doitintl/cloudresourcemanager/iface"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Reservations struct {
	log             logger.Provider
	reservations    iface.Reservations
	resourceManager iface.CloudResourceManager
}

func NewReservations(log logger.Provider, reservations iface.Reservations, resourceManager iface.CloudResourceManager) *Reservations {
	return &Reservations{
		log:             log,
		reservations:    reservations,
		resourceManager: resourceManager,
	}
}

func (s *Reservations) NewClient(ctx context.Context, opts []option.ClientOption) (*reservation.Client, error) {
	return reservation.NewClient(ctx, opts...)
}

func (s *Reservations) GetProjectsWithReservations(
	ctx context.Context,
	customerID string,
	client *reservation.Client,
	crm crmIface.CloudResourceManager,
	billingProjectsWithReservations []domain.BillingProjectWithReservation,
) ([]string, []domain.ReservationAssignment) {
	l := s.log(ctx)

	var (
		projectsUnderReservations []string
		reservationAssignments    []domain.ReservationAssignment
	)

	for _, item := range billingProjectsWithReservations {
		reservations := s.listReservations(ctx, l, customerID, client, item.Project, item.Location)

		for _, reservation := range reservations {
			projectsUnderReservation := s.getReservationAssignments(ctx, l, customerID, client, reservation.Name, crm)

			if len(projectsUnderReservation) > 0 {
				projectsUnderReservations = append(projectsUnderReservations, projectsUnderReservation...)
				reservationAssignments = append(reservationAssignments, domain.ReservationAssignment{
					Reservation:  reservation,
					ProjectsList: projectsUnderReservation,
				})
			}
		}
	}

	return projectsUnderReservations, reservationAssignments
}

func (s *Reservations) getReservationAssignments(
	ctx context.Context,
	log logger.ILogger,
	customerID string,
	client *reservation.Client,
	reservationName string,
	crm crmIface.CloudResourceManager,
) []string {
	var projectsUnderReservations []string

	assignments, err := s.reservations.ListAssignments(ctx, client, reservationName)
	if err != nil {
		return handleReservationsError[string](log, "ListAssignments", customerID, err)
	}

	for _, assignment := range assignments {
		resourceType, resourceID := splitAssignee(assignment.Assignee)

		if resourceType == "projects" {
			projectsUnderReservations = append(projectsUnderReservations, resourceID)
		} else {
			filter := fmt.Sprintf("parent.type:%s parent.id:%s", singularise(resourceType), resourceID)

			projects, err := s.resourceManager.ListCustomerProjects(ctx, crm, filter)
			if err != nil {
				log.Errorf(wrapOperationError("ListCustomerProjects", customerID, err).Error())

				continue
			}

			projectsUnderReservations = append(projectsUnderReservations, projects...)
		}
	}

	return projectsUnderReservations
}

func (s *Reservations) listReservations(
	ctx context.Context,
	log logger.ILogger,
	customerID string,
	client *reservation.Client,
	projectID string,
	location string,
) []domain.Reservation {
	response, err := s.reservations.ListReservations(ctx, client, projectID, location)
	if err != nil {
		return handleReservationsError[domain.Reservation](log, "ListReservations", customerID, err)
	}

	var reservations []domain.Reservation

	for _, reservation := range response {
		reservations = append(reservations, domain.Reservation{
			Name:    reservation.GetName(),
			Edition: reservation.GetEdition(),
		})
	}

	return reservations
}

func (s *Reservations) GetCapacityCommitments(
	ctx context.Context,
	customerID string,
	client *reservation.Client,
	billingProjectsWithReservations []domain.BillingProjectWithReservation,
) map[string][]domain.CapacityCommitment {
	perProjectCapacityCommitments := make(map[string][]domain.CapacityCommitment)

	for _, item := range billingProjectsWithReservations {
		capacityCommitments := s.listCapacityCommitmentsForProject(
			ctx,
			customerID,
			client,
			item.Project,
			item.Location,
		)

		if len(capacityCommitments) > 0 {
			perProjectCapacityCommitments[item.Project] = capacityCommitments
		}
	}

	return perProjectCapacityCommitments
}

func (s *Reservations) listCapacityCommitmentsForProject(
	ctx context.Context,
	customerID string,
	client *reservation.Client,
	projectID string,
	location string,
) []domain.CapacityCommitment {
	response, err := s.reservations.ListCapacityCommitments(ctx, client, projectID, location)
	if err != nil {
		return handleReservationsError[domain.CapacityCommitment](s.log(ctx), "ListCapacityCommitments", customerID, err)
	}

	var commitments []domain.CapacityCommitment

	for _, commitment := range response {
		commitments = append(commitments, domain.CapacityCommitment{
			Name:                commitment.GetName(),
			SlotCount:           commitment.GetSlotCount(),
			Plan:                commitment.GetPlan(),
			CommitmentStartTime: commitment.GetCommitmentStartTime().AsTime(),
			CommitmentEndTime:   commitment.GetCommitmentEndTime().AsTime(),
			Edition:             commitment.GetEdition(),
		})
	}

	return commitments
}

func splitAssignee(assignee string) (resourceType, resourceID string) {
	parts := strings.SplitN(assignee, "/", 2)

	resourceType = parts[0]

	if len(parts) > 1 {
		resourceID = parts[1]
	}

	return
}

func singularise(input string) string {
	// For regular nouns, remove the trailing 's'
	if strings.HasSuffix(input, "s") {
		return input[:len(input)-1]
	}

	return input
}
