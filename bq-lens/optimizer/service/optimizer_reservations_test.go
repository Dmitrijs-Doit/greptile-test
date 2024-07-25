package service

import (
	"context"
	"fmt"
	"testing"

	reservation "cloud.google.com/go/bigquery/reservation/apiv1"
	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	crmMocks "github.com/doitintl/cloudresourcemanager/mocks"
	"github.com/doitintl/errors"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

var (
	customerID          = "mock-customer-id"
	someErr             = errors.New("some error")
	errPermissionDenied = status.Errorf(codes.PermissionDenied, "access denied")
	mockProjectID       = "doitintl-cmp-dev"
	mockDatasetID       = "mock-dataset-id"
	mockLocation        = "us-central1"
	mockTableID         = "mock-table-id"
	mockTableIDBase     = "mock-table-id-base"
	reservationName     = "mock-reservation-name"
	mockEdition         = reservationpb.Edition_ENTERPRISE
	assignment1         = "organizations/1234"
	assignment2         = "folder/5678"
	assignment3         = "projects/9876"

	mockCapacityCommitment = "capacityCommitment-1"
	mockCommitmentPlan     = reservationpb.CapacityCommitment_ANNUAL
)

func TestReservations_GetProjectsWithReservations(t *testing.T) {
	var (
		ctx       = context.Background()
		rsvClient = &reservation.Client{}
	)

	type fields struct {
		log             loggerMocks.ILogger
		reservations    mocks.Reservations
		resourceManager mocks.CloudResourceManager
	}

	type args struct {
		billingProjectsWithReservations []domain.BillingProjectWithReservation
	}

	tests := []struct {
		name                       string
		on                         func(*fields)
		args                       args
		wantProjects               []string
		wantReservationAssignments []domain.ReservationAssignment
	}{
		{
			name: "get projects with reservations",
			args: args{
				billingProjectsWithReservations: []domain.BillingProjectWithReservation{
					{Project: mockProjectID, Location: mockLocation},
				},
			},
			on: func(f *fields) {
				f.reservations.On("ListReservations", ctx, rsvClient, mockProjectID, mockLocation).
					Return([]*reservationpb.Reservation{
						{
							Name:    reservationName,
							Edition: mockEdition,
						},
					}, nil)

				f.reservations.On(
					"ListAssignments",
					ctx,
					rsvClient,
					reservationName,
				).Return([]*reservationpb.Assignment{
					{Assignee: assignment1},
				}, nil)

				f.resourceManager.On(
					"ListCustomerProjects",
					ctx,
					mock.AnythingOfType("*mocks.CloudResourceManager"),
					"parent.type:organization parent.id:1234",
				).Return([]string{"project1", "project2"}, nil).Once()
			},
			wantProjects: []string{"project1", "project2"},
			wantReservationAssignments: []domain.ReservationAssignment{
				{
					Reservation: domain.Reservation{
						Name:    reservationName,
						Edition: mockEdition,
					},
					ProjectsList: []string{"project1", "project2"},
				},
			},
		},
		{
			name: "get projects with no reservations names",
			args: args{
				billingProjectsWithReservations: []domain.BillingProjectWithReservation{
					{Project: mockProjectID, Location: mockLocation},
				},
			},
			on: func(f *fields) {
				f.reservations.On("ListReservations", ctx, rsvClient, mockProjectID, mockLocation).
					Return(nil, nil)
			},
		},
		{
			name: "get projects with no assignments",
			args: args{
				billingProjectsWithReservations: []domain.BillingProjectWithReservation{
					{Project: mockProjectID, Location: mockLocation},
				},
			},
			on: func(f *fields) {
				f.reservations.On("ListReservations", ctx, rsvClient, mockProjectID, mockLocation).
					Return([]*reservationpb.Reservation{
						{Name: reservationName},
					}, nil)

				f.reservations.On(
					"ListAssignments",
					ctx,
					rsvClient,
					reservationName,
				).Return(nil, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &Reservations{
				log: func(ctx context.Context) logger.ILogger {
					return &fields.log
				},
				reservations:    &fields.reservations,
				resourceManager: &fields.resourceManager,
			}

			gotProjectsUnderReservations, reservationAssignments := s.GetProjectsWithReservations(
				ctx,
				customerID,
				rsvClient,
				crmMocks.NewCloudResourceManager(t),
				tt.args.billingProjectsWithReservations,
			)

			assert.Equal(t, tt.wantProjects, gotProjectsUnderReservations)
			assert.Equal(t, tt.wantReservationAssignments, reservationAssignments)
		})
	}
}

func TestReservations_getReservationAssignments(t *testing.T) {
	var (
		ctx       = context.Background()
		rsvClient = &reservation.Client{}
	)

	type fields struct {
		log             loggerMocks.ILogger
		reservations    mocks.Reservations
		resourceManager mocks.CloudResourceManager
	}

	type args struct {
		reservationName string
	}

	tests := []struct {
		name string
		on   func(*fields)
		args args
		want []string
	}{
		{
			name: "get assignments without 'projects' resource type",
			args: args{
				reservationName: reservationName,
			},
			on: func(f *fields) {
				f.reservations.On(
					"ListAssignments",
					ctx,
					rsvClient,
					reservationName,
				).Return([]*reservationpb.Assignment{
					{Assignee: assignment1}, {Assignee: assignment2},
				}, nil)

				f.resourceManager.On(
					"ListCustomerProjects",
					ctx,
					mock.AnythingOfType("*mocks.CloudResourceManager"),
					fmt.Sprintf("parent.type:organization parent.id:1234"),
				).Return([]string{"project1", "project2"}, nil).Once()

				f.resourceManager.On(
					"ListCustomerProjects",
					ctx,
					mock.AnythingOfType("*mocks.CloudResourceManager"),
					fmt.Sprintf("parent.type:folder parent.id:5678"),
				).Return([]string{"project3"}, nil).Once()
			},
			want: []string{"project1", "project2", "project3"},
		},
		{
			name: "get assignments with 'projects' resource type",
			args: args{
				reservationName: reservationName,
			},
			on: func(f *fields) {
				f.reservations.On(
					"ListAssignments",
					ctx,
					rsvClient,
					reservationName,
				).Return([]*reservationpb.Assignment{
					{Assignee: assignment1}, {Assignee: assignment3},
				}, nil)

				f.resourceManager.On(
					"ListCustomerProjects",
					ctx,
					mock.AnythingOfType("*mocks.CloudResourceManager"),
					fmt.Sprintf("parent.type:organization parent.id:1234"),
				).Return([]string{"project1", "project2"}, nil).Once()
			},
			want: []string{"project1", "project2", "9876"},
		},
		{
			name: "failed to get assignments",
			args: args{
				reservationName: reservationName,
			},
			on: func(f *fields) {
				f.reservations.On(
					"ListAssignments",
					ctx,
					rsvClient,
					reservationName,
				).Return([]*reservationpb.Assignment{
					{Assignee: assignment1}, {Assignee: assignment2},
				}, someErr)

				f.log.On("Errorf", wrapOperationError("ListAssignments", customerID, someErr).Error())
			},
			want: nil,
		},
		{
			name: "permission denied error",
			args: args{
				reservationName: reservationName,
			},
			on: func(f *fields) {
				f.reservations.On(
					"ListAssignments",
					ctx,
					rsvClient,
					reservationName,
				).Return([]*reservationpb.Assignment{
					{Assignee: assignment1}, {Assignee: assignment2},
				}, errPermissionDenied)

				f.log.On("Warning", wrapPermissionDeniedError("ListAssignments", customerID, errPermissionDenied).Error())
			},
			want: nil,
		},
		{
			name: "get assignments without 'projects' resource type",
			args: args{
				reservationName: reservationName,
			},
			on: func(f *fields) {
				f.reservations.On(
					"ListAssignments",
					ctx,
					rsvClient,
					reservationName,
				).Return([]*reservationpb.Assignment{
					{Assignee: assignment1}, {Assignee: assignment2},
				}, nil)

				f.resourceManager.On(
					"ListCustomerProjects",
					ctx,
					mock.AnythingOfType("*mocks.CloudResourceManager"),
					fmt.Sprintf("parent.type:organization parent.id:1234"),
				).Return([]string{"project1", "project2"}, someErr).Once()

				f.log.On("Errorf", wrapOperationError("ListCustomerProjects", customerID, someErr).Error())

				f.resourceManager.On(
					"ListCustomerProjects",
					ctx,
					mock.AnythingOfType("*mocks.CloudResourceManager"),
					fmt.Sprintf("parent.type:folder parent.id:5678"),
				).Return([]string{"project3"}, nil).Once()
			},
			want: []string{"project3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &Reservations{
				log: func(ctx context.Context) logger.ILogger {
					return &fields.log
				},
				reservations:    &fields.reservations,
				resourceManager: &fields.resourceManager,
			}

			got := s.getReservationAssignments(
				ctx,
				s.log(ctx),
				customerID,
				rsvClient,
				tt.args.reservationName,
				crmMocks.NewCloudResourceManager(t),
			)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_splitAssignee(t *testing.T) {
	type args struct {
		assignee string
	}

	tests := []struct {
		name             string
		args             args
		wantResourceType string
		wantResourceId   string
	}{
		{
			name: "typical case",
			args: args{
				assignee: "projects/12345",
			},
			wantResourceType: "projects",
			wantResourceId:   "12345",
		},
		{
			name: "single part",
			args: args{
				assignee: "projects",
			},
			wantResourceType: "projects",
			wantResourceId:   "",
		},
		{
			name: "empty string",
			args: args{
				assignee: "",
			},
			wantResourceType: "",
			wantResourceId:   "",
		},
		{
			name: "multiple separators",
			args: args{
				assignee: "project/subproject/123",
			},
			wantResourceType: "project",
			wantResourceId:   "subproject/123",
		},
		{
			name: "leading separator",
			args: args{
				assignee: "/12345",
			},
			wantResourceType: "",
			wantResourceId:   "12345",
		},
		{
			name: "trailing separator",
			args: args{
				assignee: "projects/",
			},
			wantResourceType: "projects",
			wantResourceId:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResourceType, gotResourceId := splitAssignee(tt.args.assignee)
			assert.Equalf(t, tt.wantResourceType, gotResourceType, "splitAssignee(%v)", tt.args.assignee)
			assert.Equalf(t, tt.wantResourceId, gotResourceId, "splitAssignee(%v)", tt.args.assignee)
		})
	}
}

func TestReservations_listReservations(t *testing.T) {
	var (
		ctx       = context.Background()
		rsvClient = &reservation.Client{}
	)

	type fields struct {
		log             loggerMocks.ILogger
		reservations    mocks.Reservations
		resourceManager mocks.CloudResourceManager
	}

	type args struct {
		projectID string
		location  string
	}

	tests := []struct {
		name string
		on   func(*fields)
		args args
		want []domain.Reservation
	}{
		{
			name: "reservation list with values",
			args: args{
				projectID: mockProjectID,
				location:  mockLocation,
			},
			on: func(f *fields) {
				f.reservations.On("ListReservations", ctx, rsvClient, mockProjectID, mockLocation).
					Return([]*reservationpb.Reservation{
						{
							Name:    reservationName,
							Edition: mockEdition,
						},
					}, nil)
			},
			want: []domain.Reservation{
				{
					Name:    reservationName,
					Edition: mockEdition,
				},
			},
		},
		{
			name: "empty list",
			args: args{
				projectID: mockProjectID,
				location:  mockLocation,
			},
			on: func(f *fields) {
				f.reservations.On("ListReservations", ctx, rsvClient, mockProjectID, mockLocation).
					Return(nil, nil)
			},
			want: nil,
		},
		{
			name: "failed to get list",
			args: args{
				projectID: mockProjectID,
				location:  mockLocation,
			},
			on: func(f *fields) {
				f.reservations.On("ListReservations", ctx, rsvClient, mockProjectID, mockLocation).
					Return(nil, someErr)

				f.log.On("Errorf", wrapOperationError("ListReservations", customerID, someErr).Error())
			},
			want: nil,
		},
		{
			name: "permission denied error",
			args: args{
				projectID: mockProjectID,
				location:  mockLocation,
			},
			on: func(f *fields) {
				f.reservations.On("ListReservations", ctx, rsvClient, mockProjectID, mockLocation).
					Return(nil, errPermissionDenied)

				f.log.On("Warning", wrapPermissionDeniedError("ListReservations", customerID, errPermissionDenied).Error())
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &Reservations{
				log: func(ctx context.Context) logger.ILogger {
					return &fields.log
				},
				reservations:    &fields.reservations,
				resourceManager: &fields.resourceManager,
			}

			assert.Equal(
				t,
				tt.want,
				s.listReservations(ctx, s.log(ctx), customerID, rsvClient, tt.args.projectID, tt.args.location))
		})
	}
}

func TestReservations_listCapacityCommitmentsForProject(t *testing.T) {
	var (
		ctx       = context.Background()
		rsvClient = &reservation.Client{}
	)

	type fields struct {
		log             loggerMocks.ILogger
		reservations    mocks.Reservations
		resourceManager mocks.CloudResourceManager
	}

	type args struct {
		projectID string
		location  string
	}

	tests := []struct {
		name string
		on   func(*fields)
		args args
		want []domain.CapacityCommitment
	}{
		{
			name: "capacity commitment list with values, active",
			args: args{
				projectID: mockProjectID,
				location:  mockLocation,
			},
			on: func(f *fields) {
				f.reservations.On("ListCapacityCommitments", ctx, rsvClient, mockProjectID, mockLocation).
					Return([]*reservationpb.CapacityCommitment{
						{
							Name:      mockCapacityCommitment,
							Edition:   mockEdition,
							SlotCount: 100,
							Plan:      mockCommitmentPlan,
							State:     reservationpb.CapacityCommitment_ACTIVE,
						},
					}, nil)
			},
			want: []domain.CapacityCommitment{
				{
					Name:      mockCapacityCommitment,
					SlotCount: 100,
					Plan:      mockCommitmentPlan,
					Edition:   mockEdition,
				},
			},
		},
		{
			name: "empty list",
			args: args{
				projectID: mockProjectID,
				location:  mockLocation,
			},
			on: func(f *fields) {
				f.reservations.On("ListCapacityCommitments", ctx, rsvClient, mockProjectID, mockLocation).
					Return(nil, nil)
			},
			want: nil,
		},
		{
			name: "failed to get list",
			args: args{
				projectID: mockProjectID,
				location:  mockLocation,
			},
			on: func(f *fields) {
				f.reservations.On("ListCapacityCommitments", ctx, rsvClient, mockProjectID, mockLocation).
					Return(nil, someErr)

				f.log.On("Errorf", wrapOperationError("ListCapacityCommitments", customerID, someErr).Error())
			},
			want: nil,
		},
		{
			name: "permission denied error",
			args: args{
				projectID: mockProjectID,
				location:  mockLocation,
			},
			on: func(f *fields) {
				f.reservations.On("ListCapacityCommitments", ctx, rsvClient, mockProjectID, mockLocation).
					Return(nil, errPermissionDenied)

				f.log.On("Warning", wrapPermissionDeniedError("ListCapacityCommitments", customerID, errPermissionDenied).Error())
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &Reservations{
				log: func(ctx context.Context) logger.ILogger {
					return &fields.log
				},
				reservations:    &fields.reservations,
				resourceManager: &fields.resourceManager,
			}

			got := s.listCapacityCommitmentsForProject(ctx, customerID, rsvClient, tt.args.projectID, tt.args.location)

			// Serialising and deserialising time introduces tiny differences that would make the comparison fail.
			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreFields(domain.CapacityCommitment{}, "CommitmentStartTime", "CommitmentEndTime")); diff != "" {
				t.Errorf("listCapacityCommitmentsForProject() mismatch; diff -want +got:\n %s", diff)
			}
		})
	}
}
