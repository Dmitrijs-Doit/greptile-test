package service

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/customerapi"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attributionGroupTierServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service/attributiongrouptier/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	attributionsDalMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/mocks"
	caOwnerCheckersMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/caownerchecker/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	collabMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab/mocks"
	reportMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	domainResource "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/resource/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	testTools "github.com/doitintl/hello/scheduled-tasks/common/test_tools"
	customerDalMock "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
)

var (
	email              = "requester@example.com"
	attributionGroupID = "my_attributionGroup_id"
	attributionID1     = "attribution_id_1"
	attributionID2     = "attribution_id_2"
	customerID         = "customer_id"
	userID             = "my_user_id"

	nilCustomerRef *firestore.DocumentRef
)

const pathForDocRefs = "projects/1/databases/1/documents/1/1"

func TestAnalyticsAttributionGroupsService_ShareAttributionGroup(t *testing.T) {
	type fields struct {
		attributionGroupsDal *mocks.AttributionGroups
		collab               *collabMock.Icollab
		caOwnerChecker       *caOwnerCheckersMock.CheckCAOwnerInterface
	}

	type args struct {
		ctx                context.Context
		collaboratorsReq   []collab.Collaborator
		public             *collab.PublicAccess
		email              string
		attributionGroupID string
		userID             string
	}

	ctx := context.Background()

	attributionGroup := &attributiongroups.AttributionGroup{
		Access: collab.Access{
			Collaborators: []collab.Collaborator{},
		},
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error

		on func(*fields)
	}{
		{
			name: "successfully share attributionGroup",
			args: args{
				ctx:                ctx,
				collaboratorsReq:   []collab.Collaborator{},
				public:             nil,
				email:              email,
				attributionGroupID: attributionGroupID,
				userID:             userID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroup, nil).
					Once()
				f.collab.
					On("ShareAnalyticsResource", ctx, mock.AnythingOfType("[]collab.Collaborator"), mock.AnythingOfType("[]collab.Collaborator"), mock.AnythingOfType("*collab.PublicAccess"), attributionGroupID, email, mock.Anything, true).
					Return(nil).
					Once()
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()
			},
			expectedErr: nil,
		}, {
			name: "Get returns error",
			args: args{
				ctx:                ctx,
				collaboratorsReq:   []collab.Collaborator{},
				public:             nil,
				email:              email,
				attributionGroupID: attributionGroupID,
				userID:             userID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroup, errors.New("error")).
					Once()
				f.collab.
					On("ShareAnalyticsResource", ctx, mock.AnythingOfType("[]collab.Collaborator"), mock.AnythingOfType("[]collab.Collaborator"), mock.AnythingOfType("*collab.PublicAccess"), attributionGroupID, email, mock.Anything, true).
					Return(nil).
					Once()
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()
			},
			expectedErr: errors.New("error"),
		}, {
			name: "ShareAnalyticsResource returns error",
			args: args{
				ctx:                ctx,
				collaboratorsReq:   []collab.Collaborator{},
				public:             nil,
				email:              email,
				attributionGroupID: attributionGroupID,
				userID:             userID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroup, nil).
					Once()
				f.collab.
					On("ShareAnalyticsResource", ctx, mock.AnythingOfType("[]collab.Collaborator"), mock.AnythingOfType("[]collab.Collaborator"), mock.AnythingOfType("*collab.PublicAccess"), attributionGroupID, email, mock.Anything, true).
					Return(errors.New("error2")).
					Once()
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(true, nil).Once()
			},
			expectedErr: errors.New("error2"),
		}, {
			name: "ShareAnalyticsResource returns error if CheckCAOwner throwing error",
			args: args{
				ctx:                ctx,
				collaboratorsReq:   []collab.Collaborator{},
				public:             nil,
				email:              email,
				attributionGroupID: attributionGroupID,
				userID:             userID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.caOwnerChecker.On("CheckCAOwner", ctx, mock.Anything, userID, email).Return(false, errors.New("error")).Once()
			},
			expectedErr: errors.New("error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				attributionGroupsDal: &mocks.AttributionGroups{},
				collab:               &collabMock.Icollab{},
				caOwnerChecker:       &caOwnerCheckersMock.CheckCAOwnerInterface{},
			}
			s := &AttributionGroupsService{
				attributionGroupsDAL: tt.fields.attributionGroupsDal,
				collab:               tt.fields.collab,
				caOwnerChecker:       tt.fields.caOwnerChecker,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.ShareAttributionGroup(tt.args.ctx, tt.args.collaboratorsReq, tt.args.public, tt.args.attributionGroupID, tt.args.userID, tt.args.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsService.ShareAttributionGroup() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

func TestAnalyticsAttributionGroupsService_CreateAttributionGroup(t *testing.T) {
	type fields struct {
		attributionGroupsDal *mocks.AttributionGroups
		attributionsDal      *attributionsDalMock.Attributions
		customersDal         *customerDalMock.Customers
	}

	type args struct {
		ctx            context.Context
		customerID     string
		requestedEmail string
		attributionReq *attributiongroups.AttributionGroupRequest
	}

	ctx := context.Background()

	nullFallback := "stuff"

	attributionGroupReq := &attributiongroups.AttributionGroupRequest{
		Name:        "attribution group",
		Description: "description",
		Attributions: []string{
			attributionID1,
			attributionID2,
		},
		NullFallback: &nullFallback,
	}

	attribution1, attribution2, nonResourceAttribution, attributionRef1, attributionRef2, customerRef := setupAttributionData()

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error

		on func(*fields)
	}{
		{
			name: "successfully create attribution group",
			args: args{
				ctx:            ctx,
				customerID:     customerID,
				requestedEmail: email,
				attributionReq: attributionGroupReq,
			},
			wantErr: false,
			on: func(f *fields) {
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, customerRef, "attribution group").
					Return(nil, attributiongroups.ErrNotFound).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, nilCustomerRef, "attribution group").
					Return(nil, attributiongroups.ErrNotFound).
					Once()
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID1).
					Return(attribution1, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID1).
					Return(attributionRef1).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID2).
					Return(attribution2, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID2).
					Return(attributionRef2).
					Once()
				f.attributionGroupsDal.
					On("Create", ctx, mock.AnythingOfType("*attributiongroups.AttributionGroup")).
					Return(attributionGroupID, nil).
					Once()
			},
			expectedErr: nil,
		},
		{
			name: "Create returns error",
			args: args{
				ctx:            ctx,
				customerID:     customerID,
				requestedEmail: email,
				attributionReq: attributionGroupReq,
			},
			wantErr: true,
			on: func(f *fields) {
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, customerRef, "attribution group").
					Return(nil, attributiongroups.ErrNotFound).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, nilCustomerRef, "attribution group").
					Return(nil, attributiongroups.ErrNotFound).
					Once()
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID1).
					Return(attribution1, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID1).
					Return(attributionRef1).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID2).
					Return(attribution2, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID2).
					Return(attributionRef2).
					Once()
				f.attributionGroupsDal.
					On("Create", ctx, mock.AnythingOfType("*attributiongroups.AttributionGroup")).
					Return("", errors.New("error")).
					Once()
			},
			expectedErr: errors.New("error"),
		},
		{
			name: "attribution not from same customer",
			args: args{
				ctx:            ctx,
				customerID:     customerID,
				requestedEmail: email,
				attributionReq: attributionGroupReq,
			},
			wantErr: true,
			on: func(f *fields) {
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, customerRef, "attribution group").
					Return(nil, attributiongroups.ErrNotFound).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, nilCustomerRef, "attribution group").
					Return(nil, attributiongroups.ErrNotFound).
					Once()
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID1).
					Return(attribution1, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID1).
					Return(attributionRef1).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID2).
					Return(nonResourceAttribution, nil).
					Once()
			},
			expectedErr: attributiongroups.ErrForbiddenAttribution,
		},
		{
			name: "attribution group already exists",
			args: args{
				ctx:            ctx,
				customerID:     customerID,
				requestedEmail: email,
				attributionReq: attributionGroupReq,
			},
			wantErr: true,
			on: func(f *fields) {
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, customerRef, "attribution group").
					Return(nil, nil).
					Once()
			},
			expectedErr: attributiongroups.ErrNameAlreadyExists,
		},
		{
			name: "preset attribution group already exists",
			args: args{
				ctx:            ctx,
				customerID:     customerID,
				requestedEmail: email,
				attributionReq: attributionGroupReq,
			},
			wantErr: true,
			on: func(f *fields) {
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, customerRef, "attribution group").
					Return(nil, attributiongroups.ErrNotFound).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, nilCustomerRef, "attribution group").
					Return(nil, nil).
					Once()
			},
			expectedErr: attributiongroups.ErrPresetNameAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				attributionGroupsDal: &mocks.AttributionGroups{},
				attributionsDal:      &attributionsDalMock.Attributions{},
				customersDal:         &customerDalMock.Customers{},
			}

			s := &AttributionGroupsService{
				attributionGroupsDAL: tt.fields.attributionGroupsDal,
				attributionsDAL:      tt.fields.attributionsDal,
				customersDAL:         tt.fields.customersDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.CreateAttributionGroup(tt.args.ctx, tt.args.customerID, tt.args.requestedEmail, tt.args.attributionReq)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsService.CreateAttributionGroup() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else if !tt.wantErr {
				assert.Equal(t, attributionGroupID, got)
			}
		})
	}
}

func TestAnalyticsAttributionGroupsService_UpdateAttributionGroup(t *testing.T) {
	type fields struct {
		attributionGroupsDal *mocks.AttributionGroups
		attributionsDal      *attributionsDalMock.Attributions
		customersDal         *customerDalMock.Customers
	}

	type args struct {
		ctx                  context.Context
		customerID           string
		requesterEmail       string
		attributionUpdateReq *attributiongroups.AttributionGroupUpdateRequest
	}

	ctx := context.Background()

	nullFallback := "stuff"

	attributionGroupUpdateReq := &attributiongroups.AttributionGroupUpdateRequest{
		Name:        "attribution group",
		Description: "description",
		Attributions: []string{
			attributionID1,
			attributionID2,
		},
		NullFallback: &nullFallback,
	}

	attribution1, attribution2, nonResourceAttribution, attributionRef1, attributionRef2, customerRef := setupAttributionData()

	access := collab.Access{
		Collaborators: []collab.Collaborator{
			{
				Email: email,
				Role:  collab.CollaboratorRoleOwner,
			},
		},
	}

	attributionGroup := &attributiongroups.AttributionGroup{
		Customer: &firestore.DocumentRef{
			ID: customerID,
		},
		ID:          attributionGroupID,
		Name:        "some attribution group name",
		Description: "description",
		Attributions: []*firestore.DocumentRef{
			attributionRef1,
			attributionRef2,
		},
		Type: domainAttributions.ObjectTypeCustom,
		Access: collab.Access{
			Collaborators: access.Collaborators,
		},
		NullFallback: &nullFallback,
	}

	attributionGroupWithoutPermissions := *attributionGroup
	attributionGroupWithoutPermissions.Collaborators = []collab.Collaborator{}

	attributionGroupSameName := &attributiongroups.AttributionGroup{
		Customer: &firestore.DocumentRef{
			ID: customerID,
		},
		ID:          attributionGroupID,
		Name:        "attribution group",
		Description: "description",
		Attributions: []*firestore.DocumentRef{
			attributionRef1,
			attributionRef2,
		},
		Type: domainAttributions.ObjectTypeCustom,
		Access: collab.Access{
			Collaborators: access.Collaborators,
		},
	}

	attributionGroupPreset := &attributiongroups.AttributionGroup{
		Customer:    nil,
		ID:          attributionGroupID,
		Name:        "attribution group",
		Description: "description",
		Attributions: []*firestore.DocumentRef{
			attributionRef1,
		},
		Type: domainAttributions.ObjectTypePreset,
		Access: collab.Access{
			Collaborators: access.Collaborators,
		},
	}

	attributionGroupManaged := &attributiongroups.AttributionGroup{
		Customer:    nil,
		ID:          attributionGroupID,
		Name:        "attribution group",
		Description: "description",
		Attributions: []*firestore.DocumentRef{
			attributionRef1,
		},
		Type: domainAttributions.ObjectTypeManaged,
		Access: collab.Access{
			Collaborators: access.Collaborators,
		},
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successfully update attribution group",
			args: args{
				ctx:                  ctx,
				customerID:           customerID,
				requesterEmail:       email,
				attributionUpdateReq: attributionGroupUpdateReq,
			},
			wantErr: false,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroup, nil).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID1).
					Return(attribution1, nil).
					Once()
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, customerRef, "attribution group").
					Return(nil, attributiongroups.ErrNotFound).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, nilCustomerRef, "attribution group").
					Return(nil, attributiongroups.ErrNotFound).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID1).
					Return(attributionRef1).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID2).
					Return(attribution2, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID2).
					Return(attributionRef2).
					Once()
				f.attributionGroupsDal.
					On("Update", ctx, attributionGroupID, attributionGroup).
					Return(nil).
					Once()
			},
		},
		{
			name: "no need to check attribution group name if user does not update the name",
			args: args{
				ctx:                  ctx,
				customerID:           customerID,
				requesterEmail:       email,
				attributionUpdateReq: attributionGroupUpdateReq,
			},
			wantErr: false,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroupSameName, nil).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID1).
					Return(attribution1, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID1).
					Return(attributionRef1).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID2).
					Return(attribution2, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID2).
					Return(attributionRef2).
					Once()
				f.attributionGroupsDal.
					On("Update", ctx, attributionGroupID, attributionGroupSameName).
					Return(nil).
					Once()
			},
		},
		{
			name: "fail to update attribution group - attribution group not found",
			args: args{
				ctx:                  ctx,
				customerID:           customerID,
				requesterEmail:       email,
				attributionUpdateReq: attributionGroupUpdateReq,
			},
			wantErr:     true,
			expectedErr: errors.New("attribution group not found"),
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(nil, errors.New("attribution group not found")).
					Once()
			},
		},
		{
			name: "fail - user without required permissions",
			args: args{
				ctx:                  ctx,
				customerID:           customerID,
				requesterEmail:       email,
				attributionUpdateReq: attributionGroupUpdateReq,
			},
			wantErr:     true,
			expectedErr: errors.New("user does not have required permissions to update this attribution group"),
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(&attributionGroupWithoutPermissions, nil).
					Once()
			},
		},
		{
			name: "fail - update attribution group returns error",
			args: args{
				ctx:                  ctx,
				customerID:           customerID,
				requesterEmail:       email,
				attributionUpdateReq: attributionGroupUpdateReq,
			},
			wantErr:     true,
			expectedErr: errors.New("failed to update attribution group"),
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroup, nil).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID1).
					Return(attribution1, nil).
					Once()
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, customerRef, "attribution group").
					Return(nil, attributiongroups.ErrNotFound).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID1).
					Return(attributionRef1).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID2).
					Return(attribution2, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID2).
					Return(attributionRef2).
					Once()
				f.attributionGroupsDal.
					On("Update", ctx, attributionGroupID, attributionGroup).
					Return(errors.New("failed to update attribution group")).
					Once()
			},
		},
		{
			name: "fail - customer id does not match resource",
			args: args{
				ctx:                  ctx,
				customerID:           "other_customer_id",
				requesterEmail:       email,
				attributionUpdateReq: attributionGroupUpdateReq,
			},
			wantErr:     true,
			expectedErr: errors.New("user does not have required permissions to update this attribution group"),
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroup, nil).
					Once()
			},
		},
		{
			name: "fail - attribution is not from customer",
			args: args{
				ctx:                  ctx,
				customerID:           customerID,
				requesterEmail:       email,
				attributionUpdateReq: attributionGroupUpdateReq,
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrForbiddenAttribution,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroup, nil).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID1).
					Return(attribution1, nil).
					Once()
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, customerRef, "attribution group").
					Return(nil, attributiongroups.ErrNotFound).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID1).
					Return(attributionRef1).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID2).
					Return(nonResourceAttribution, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID2).
					Return(attributionRef2).
					Once()
			},
		},
		{
			name: "fail - attribution group is a preset",
			args: args{
				ctx:                  ctx,
				customerID:           customerID,
				requesterEmail:       email,
				attributionUpdateReq: attributionGroupUpdateReq,
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrForbidden,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroupPreset, nil).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID1).
					Return(attribution1, nil).
					Once()
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, customerRef, "attribution group").
					Return(attributionGroupPreset, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID1).
					Return(attributionRef1).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID2).
					Return(nonResourceAttribution, nil).
					Once()
			},
		},
		{
			name: "fail - attribution group is managed",
			args: args{
				ctx:                  ctx,
				customerID:           customerID,
				requesterEmail:       email,
				attributionUpdateReq: attributionGroupUpdateReq,
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrForbidden,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroupManaged, nil).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID1).
					Return(attribution1, nil).
					Once()
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef, nil).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, customerRef, "attribution group").
					Return(attributionGroupPreset, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID1).
					Return(attributionRef1).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID2).
					Return(nonResourceAttribution, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				attributionGroupsDal: &mocks.AttributionGroups{},
				attributionsDal:      &attributionsDalMock.Attributions{},
				customersDal:         &customerDalMock.Customers{},
			}

			s := &AttributionGroupsService{
				attributionGroupsDAL: tt.fields.attributionGroupsDal,
				attributionsDAL:      tt.fields.attributionsDal,
				customersDAL:         tt.fields.customersDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := s.UpdateAttributionGroup(tt.args.ctx, tt.args.customerID, attributionGroupID, tt.args.requesterEmail, tt.args.attributionUpdateReq); err != nil {
				if !tt.wantErr || tt.expectedErr.Error() != err.Error() {
					t.Errorf("AttributionGroupsService.UpdateAttributionGroup() actual error = %v, expected error = %v, wantErr %v", err, tt.expectedErr, tt.wantErr)
				}
			}
		})
	}
}

func TestAttributionGroupsService_validateAttributions(t *testing.T) {
	type fields struct {
		attributionsDal *attributionsDalMock.Attributions
	}

	type args struct {
		ctx            context.Context
		attributionIDs []string
		customerID     string
	}

	ctx := context.Background()

	attribution1, attribution2, nonResourceAttribution, attributionRef1, attributionRef2, customerRef := setupAttributionData()

	managedAttribution := &attribution.Attribution{
		ID:       "managed-attribution",
		Customer: customerRef,
		Type:     string(domainAttributions.ObjectTypeManaged),
	}

	managedRef := &firestore.DocumentRef{
		ID:   managedAttribution.ID,
		Path: pathForDocRefs,
	}

	attributionRefs := []*firestore.DocumentRef{attributionRef1, attributionRef2}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "success - validate attributions from customer",
			args: args{
				ctx:            ctx,
				attributionIDs: []string{attributionID1, attributionID2},
				customerID:     customerID,
			},
			on: func(f *fields) {
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID1).
					Return(attribution1, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID1).
					Return(attributionRef1).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID2).
					Return(attribution2, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID2).
					Return(attributionRef2).
					Once()
			},
		},
		{
			name: "fail - attribution is not from customer",
			args: args{
				ctx:            ctx,
				attributionIDs: []string{attributionID1, attributionID2},
				customerID:     customerID,
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrForbiddenAttribution,
			on: func(f *fields) {
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID1).
					Return(attribution1, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID1).
					Return(attributionRef1).
					Once()
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID2).
					Return(nonResourceAttribution, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, attributionID2).
					Return(attributionRef2).
					Once()
			},
		},
		{
			name: "fail - attribution does not exist",
			args: args{
				ctx:            ctx,
				attributionIDs: []string{attributionID1, attributionID2},
				customerID:     customerID,
			},
			wantErr:     true,
			expectedErr: errors.New("attribution does not exist"),
			on: func(f *fields) {
				f.attributionsDal.
					On("GetAttribution", ctx, attributionID1).
					Return(nil, errors.New("attribution does not exist")).
					Once()
			},
		},
		{
			name: "fail - attribution is of type managed",
			args: args{
				ctx:            ctx,
				attributionIDs: []string{managedAttribution.ID},
				customerID:     customerID,
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrManagedAttributionTypeInvalid,
			on: func(f *fields) {
				f.attributionsDal.
					On("GetAttribution", ctx, managedAttribution.ID).
					Return(managedAttribution, nil).
					Once()
				f.attributionsDal.
					On("GetRef", ctx, managedAttribution.ID).
					Return(managedRef).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				attributionsDal: &attributionsDalMock.Attributions{},
			}

			s := &AttributionGroupsService{
				attributionsDAL: tt.fields.attributionsDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.validateAttributions(tt.args.ctx, tt.args.attributionIDs, tt.args.customerID)
			if err != nil {
				if !tt.wantErr || tt.expectedErr.Error() != err.Error() {
					t.Errorf("AttributionGroupsService.validateAttributions() actual error = %v, expected error = %v, wantErr %v", err, tt.expectedErr, tt.wantErr)
				}
			}

			if !tt.wantErr {
				assert.EqualValues(t, attributionRefs, got)
			}
		})
	}
}

func TestAnalyticsAttributionGroupsService_DeleteAttributionGroup(t *testing.T) {
	type fields struct {
		attributionGroupsDal *mocks.AttributionGroups
		reportDal            *reportMocks.Reports
	}

	type args struct {
		ctx                context.Context
		customerID         string
		requesterEmail     string
		attributionGroupID string
	}

	ctx := context.Background()
	publicAccessView := collab.PublicAccessView

	attributionGroup := &attributiongroups.AttributionGroup{
		Customer: &firestore.DocumentRef{
			ID: customerID,
		},
		Access: collab.Access{
			Public: &publicAccessView,
			Collaborators: []collab.Collaborator{
				{
					Email: email,
					Role:  collab.CollaboratorRoleOwner,
				},
			},
		},
		Type: domainAttributions.ObjectTypeCustom,
	}

	attributionGroupPreset := &attributiongroups.AttributionGroup{
		Customer: nil,
		Access: collab.Access{
			Public:        &publicAccessView,
			Collaborators: []collab.Collaborator{},
		},
		Type: domainAttributions.ObjectTypePreset,
	}

	attributionGroupNotOwner := &attributiongroups.AttributionGroup{
		Customer: &firestore.DocumentRef{
			ID: customerID,
		},
		Access: collab.Access{
			Public: &publicAccessView,
			Collaborators: []collab.Collaborator{
				{
					Email: email,
					Role:  collab.CollaboratorRoleEditor,
				},
			},
		},
		Type: domainAttributions.ObjectTypeCustom,
	}

	errorOnDelete := errors.New("error on delete")

	tests := []struct {
		name                      string
		fields                    fields
		args                      args
		wantErr                   bool
		expectedErr               error
		expectedBlockingResources []domainResource.Resource
		on                        func(*fields)
	}{
		{
			name: "successfully delete attribution group when user is owner",
			args: args{
				ctx:                ctx,
				customerID:         customerID,
				requesterEmail:     email,
				attributionGroupID: attributionGroupID,
			},
			wantErr:     false,
			expectedErr: nil,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroup, nil).
					Once()
				f.reportDal.
					On("GetCustomerReports", ctx, customerID).
					Return([]*report.Report{}, nil).
					Once()
				f.attributionGroupsDal.
					On("Delete", ctx, attributionGroupID).
					Return(nil).
					Once()
			},
		},
		{
			name: "error on delete attribution group when it is not custom type",
			args: args{
				ctx:                ctx,
				customerID:         customerID,
				requesterEmail:     email,
				attributionGroupID: attributionGroupID,
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrForbidden,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroupPreset, nil).
					Once()
			},
		},
		{
			name: "error on delete attribution group when it belongs to a different customer",
			args: args{
				ctx:                ctx,
				customerID:         "some-other-customer",
				requesterEmail:     email,
				attributionGroupID: attributionGroupID,
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrForbidden,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroup, nil).
					Once()
			},
		},
		{
			name: "error on delete attribution group when user is not owner",
			args: args{
				ctx:                ctx,
				customerID:         customerID,
				requesterEmail:     email,
				attributionGroupID: attributionGroupID,
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrForbidden,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroupNotOwner, nil).
					Once()
			},
		},
		{
			name: "error on delete when attribution group used in report",
			args: args{
				ctx:                ctx,
				customerID:         customerID,
				requesterEmail:     email,
				attributionGroupID: attributionGroupID,
			},
			wantErr:     true,
			expectedErr: attributiongroups.ErrAttrGroupExistReports,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroup, nil).
					Once()
				f.reportDal.
					On("GetCustomerReports", ctx, customerID).
					Return(nil, attributiongroups.ErrAttrGroupExistReports).
					Once()
			},
		},
		{
			name: "error on delete attribution group when delete operation fails",
			args: args{
				ctx:                ctx,
				customerID:         customerID,
				requesterEmail:     email,
				attributionGroupID: attributionGroupID,
			},
			wantErr:     true,
			expectedErr: errorOnDelete,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("Get", ctx, attributionGroupID).
					Return(attributionGroup, nil).
					Once()
				f.reportDal.
					On("GetCustomerReports", ctx, customerID).
					Return([]*report.Report{}, nil).
					Once()
				f.attributionGroupsDal.
					On("Delete", ctx, attributionGroupID).
					Return(errorOnDelete).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				attributionGroupsDal: &mocks.AttributionGroups{},
				reportDal:            &reportMocks.Reports{},
			}
			s := &AttributionGroupsService{
				attributionGroupsDAL: tt.fields.attributionGroupsDal,
				reportDAL:            tt.fields.reportDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			blockingResources, err := s.DeleteAttributionGroup(
				tt.args.ctx,
				tt.args.customerID,
				tt.args.requesterEmail,
				tt.args.attributionGroupID,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsService.DeleteAttributionGroup() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}

			if tt.expectedBlockingResources != nil {
				assert.Equal(t, tt.expectedBlockingResources, blockingResources)
			}
		})
	}
}

func TestAnalyticsAttributionGroupsService_ListAttributionGroups(t *testing.T) {
	type fields struct {
		attributionGroupsDal        *mocks.AttributionGroups
		customerDal                 *customerDalMock.Customers
		attributionGroupTierService *attributionGroupTierServiceMocks.AttributionGroupTierService
	}

	type args struct {
		ctx context.Context
		req *customerapi.Request
	}

	ctx := context.Background()
	publicAccessView := collab.PublicAccessView
	access := collab.Access{
		Public: &publicAccessView,
		Collaborators: []collab.Collaborator{
			{
				Email: email,
				Role:  collab.CollaboratorRoleViewer,
			},
		},
	}

	cRef := &firestore.DocumentRef{
		ID: customerID,
	}

	customer := &common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: cRef,
		},
	}
	groups := []attributiongroups.AttributionGroup{
		{
			ID:          "1",
			Name:        "1_Group",
			Description: "1_Group_Description",
			Access:      access,
			Customer:    cRef,
			Type:        domainAttributions.ObjectTypeCustom,
			Cloud:       nil,
		},
		{
			ID:          "2",
			Name:        "2_Group",
			Description: "2_Group_Description",
			Access:      access,
			Customer:    cRef,
			Type:        domainAttributions.ObjectTypePreset,
			Cloud:       []string{"google-cloud-standalone"},
		},
		{
			ID:          "3",
			Name:        "3_Group",
			Description: "3_Group_Description",
			Access:      access,
			Customer:    cRef,
			Type:        domainAttributions.ObjectTypeCustom,
			Cloud:       nil,
		},
		{
			ID:          "4",
			Name:        "4_Group",
			Description: "4_Group_Description",
			Access:      access,
			Customer:    cRef,
			Type:        domainAttributions.ObjectTypeManaged,
			Cloud:       nil,
		},
		{
			ID:          "5",
			Name:        "5_Group",
			Description: "5_Group_Description",
			Access:      access,
			Customer:    cRef,
			Type:        domainAttributions.ObjectTypeManaged,
			Cloud:       nil,
		},
	}

	sortGroups := func(groups []customerapi.SortableItem, sortBy string, sortOrder firestore.Direction) {
		switch sortBy {
		case "name":
			if sortOrder == firestore.Asc {
				sort.Slice(groups, func(p, q int) bool {
					return groups[p].(attributiongroups.AttributionGroupListItemExternal).Name < groups[q].(attributiongroups.AttributionGroupListItemExternal).Name
				})
			} else {
				sort.Slice(groups, func(p, q int) bool {
					return groups[p].(attributiongroups.AttributionGroupListItemExternal).Name > groups[q].(attributiongroups.AttributionGroupListItemExternal).Name
				})
			}

		case "description":
			if sortOrder == firestore.Asc {
				sort.Slice(groups, func(p, q int) bool {
					return groups[p].(attributiongroups.AttributionGroupListItemExternal).Description < groups[q].(attributiongroups.AttributionGroupListItemExternal).Description
				})
			} else {
				sort.Slice(groups, func(p, q int) bool {
					return groups[p].(attributiongroups.AttributionGroupListItemExternal).Description > groups[q].(attributiongroups.AttributionGroupListItemExternal).Description
				})
			}

		default:
			t.Fatalf("invalid sort by field: %s", sortBy)
		}
	}

	listError := errors.New("error listing attribution groups")

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successfully list attribution groups sort by name ascending",
			args: args{
				ctx: ctx,
				req: &customerapi.Request{
					MaxResults: 3,
					SortBy:     "name",
					SortOrder:  firestore.Asc,
					Email:      email,
					CustomerID: customerID,
				},
			},
			wantErr:     false,
			expectedErr: nil,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("List", ctx, cRef, email).
					Return(groups, nil).
					Once()
				f.customerDal.
					On("GetCustomer", ctx, customerID).
					Return(customer, nil).
					Once()
				f.attributionGroupTierService.
					On("CheckAccessToCustomAttributionGroup", ctx, customerID).
					Return(nil, nil).
					Once()
				f.attributionGroupTierService.
					On("CheckAccessToPresetAttributionGroup", ctx, customerID).
					Return(nil, nil).
					Once()
			},
		},
		{
			name: "successfully list attribution groups sort by name descending",
			args: args{
				ctx: ctx,
				req: &customerapi.Request{
					MaxResults: 3,
					SortBy:     "name",
					SortOrder:  firestore.Desc,
					Email:      email,
					CustomerID: customerID,
				},
			},
			wantErr:     false,
			expectedErr: nil,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("List", ctx, cRef, email).
					Return(groups, nil).
					Once()
				f.customerDal.
					On("GetCustomer", ctx, customerID).
					Return(customer, nil).
					Once()
				f.attributionGroupTierService.
					On("CheckAccessToCustomAttributionGroup", ctx, customerID).
					Return(nil, nil).
					Once()
				f.attributionGroupTierService.
					On("CheckAccessToPresetAttributionGroup", ctx, customerID).
					Return(nil, nil).
					Once()
			},
		},
		{
			name: "successfully list attribution groups max result 1",
			args: args{
				ctx: ctx,
				req: &customerapi.Request{
					MaxResults: 1,
					SortBy:     "name",
					SortOrder:  firestore.Desc,
					Email:      email,
					CustomerID: customerID,
				},
			},
			wantErr:     false,
			expectedErr: nil,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("List", ctx, cRef, email).
					Return(groups, nil).
					Once()
				f.customerDal.
					On("GetCustomer", ctx, customerID).
					Return(customer, nil).
					Once()
				f.attributionGroupTierService.
					On("CheckAccessToCustomAttributionGroup", ctx, customerID).
					Return(nil, nil).
					Once()
				f.attributionGroupTierService.
					On("CheckAccessToPresetAttributionGroup", ctx, customerID).
					Return(nil, nil).
					Once()
			},
		},
		{
			name: "error on list attribution groups",
			args: args{
				ctx: ctx,
				req: &customerapi.Request{
					MaxResults: 3,
					SortBy:     "name",
					SortOrder:  firestore.Desc,
					Email:      email,
					CustomerID: customerID,
				},
			},
			wantErr:     true,
			expectedErr: listError,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("List", ctx, cRef, email).
					Return(nil, listError).
					Once()
				f.customerDal.
					On("GetCustomer", ctx, customerID).
					Return(customer, nil).
					Once()
			},
		},
		{
			name: "error on list attribution groups invalid customer",
			args: args{
				ctx: ctx,
				req: &customerapi.Request{
					MaxResults: 3,
					SortBy:     "name",
					SortOrder:  firestore.Desc,
					Email:      email,
					CustomerID: customerID,
				},
			},
			wantErr:     true,
			expectedErr: doitFirestore.ErrNotFound,
			on: func(f *fields) {
				f.attributionGroupsDal.
					On("List", ctx, cRef, email).
					Return(nil, listError).
					Once()
				f.customerDal.
					On("GetCustomer", ctx, customerID).
					Return(nil, doitFirestore.ErrNotFound).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				attributionGroupsDal:        &mocks.AttributionGroups{},
				customerDal:                 &customerDalMock.Customers{},
				attributionGroupTierService: &attributionGroupTierServiceMocks.AttributionGroupTierService{},
			}

			s := &AttributionGroupsService{
				attributionGroupsDAL:        tt.fields.attributionGroupsDal,
				customersDAL:                tt.fields.customerDal,
				attributionGroupTierService: tt.fields.attributionGroupTierService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			page, err := s.ListAttributionGroupsExternal(
				tt.args.ctx,
				tt.args.req,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsService.ListAttributionGroupsExternal() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}

			if page != nil {
				cpy := make([]customerapi.SortableItem, len(page.AttributionGroups))
				copy(cpy, page.AttributionGroups)
				sortGroups(cpy, tt.args.req.SortBy, tt.args.req.SortOrder)

				if !reflect.DeepEqual(page.AttributionGroups, cpy) {
					t.Errorf("AttributionGroupsFirestore.List() sort error got = %v, want %v", page.AttributionGroups, cpy)
				}

				assert.Equal(t, tt.args.req.MaxResults, len(page.AttributionGroups))
			}
		})
	}
}

func TestAnalyticsAttributionGroupsService_validateNotInReports(t *testing.T) {
	type fields struct {
		reportsDal *reportMocks.Reports
	}

	ctx := context.Background()

	var reportJSON report.Report
	if err := testTools.ConvertJSONFileIntoStruct("testData", "report.json", &reportJSON); err != nil {
		t.Fatalf("could not convert json test file into struct. error %s", err)
	}

	var configJSON report.Config
	if err := testTools.ConvertJSONFileIntoStruct("testData", "report_config.json", &configJSON); err != nil {
		t.Fatalf("could not convert json test file into struct. error %s", err)
	}

	fullReport := reportJSON
	fullReport.ID = "111"
	fullReport.Name = "report name"

	reportErr := errors.New("error")

	tests := []struct {
		name              string
		fields            fields
		wantErr           bool
		expectedErr       error
		expectedResources []domainResource.Resource

		on func(*fields)
	}{
		{
			name:        "error retrieving reports",
			wantErr:     true,
			expectedErr: reportErr,
			on: func(f *fields) {
				f.reportsDal.
					On("GetCustomerReports", ctx, customerID).
					Return(nil, reportErr).
					Once()
			},
		},
		{
			name: "successfully validate not in reports",
			on: func(f *fields) {
				f.reportsDal.
					On("GetCustomerReports", ctx, customerID).
					Return([]*report.Report{&fullReport}, nil).
					Once()
			},
		},
		{
			name: "attr group in filters",
			on: func(f *fields) {
				f.reportsDal.
					On("GetCustomerReports", ctx, customerID).
					Return([]*report.Report{&fullReport}, nil).
					Once()
			},
			expectedResources: []domainResource.Resource{
				{
					ID:    "111",
					Name:  "report name",
					Owner: "chaim@doit.com",
				},
			},
		},
		{
			name: "attr group in rows",
			on: func(f *fields) {
				f.reportsDal.
					On("GetCustomerReports", ctx, customerID).
					Return([]*report.Report{&fullReport}, nil).
					Once()
			},
			expectedResources: []domainResource.Resource{
				{
					ID:    "111",
					Name:  "report name",
					Owner: "chaim@doit.com",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				reportsDal: &reportMocks.Reports{},
			}

			s := &AttributionGroupsService{
				reportDAL: tt.fields.reportsDal,
			}

			if tt.name == "attr group in filters" {
				fullReport.Config = &configJSON
				fullReport.Config.Rows = nil
			} else if tt.name == "attr group in rows" {
				fullReport.Config = &configJSON
				fullReport.Config.Filters = nil
				fullReport.Config.Rows = []string{"attribution_group:zY0EMISuSUs8FpMahUfD"}
			} else if tt.name == "successfully validate not in reports" {
				fullReport.Config = nil
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			inReport, err := s.isUsedByReport(ctx, customerID, "user@doit.com", "zY0EMISuSUs8FpMahUfD")
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsService.validateNotInReports() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}

			if err == nil {
				assert.Equal(t, tt.expectedResources, inReport)
			}
		})
	}
}

func TestAnalyticsAttributionGroupsService_validateAttributionGroupName(t *testing.T) {
	type fields struct {
		customersDal         *customerDalMock.Customers
		attributionGroupsDal *mocks.AttributionGroups
	}

	type args struct {
		attributionName string
	}

	ctx := context.Background()

	attributionName := "attribution group"

	customerRef := &firestore.DocumentRef{
		ID:   customerID,
		Path: pathForDocRefs,
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error

		on func(*fields)
	}{
		{
			name:    "attribution name is available when attribution group not found",
			wantErr: false,
			args: args{
				attributionName: attributionName,
			},
			on: func(f *fields) {
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, customerRef, attributionName).
					Return(nil, attributiongroups.ErrNotFound).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, nilCustomerRef, attributionName).
					Return(nil, attributiongroups.ErrNotFound).
					Once()
			},
		},
		{
			name:        "attribution name is invalid when attribution group name is empty string",
			wantErr:     true,
			expectedErr: attributiongroups.ErrInvalidAttributionGroupName,
			args: args{
				attributionName: "",
			},
		},
		{
			name:        "attribution name is not available when attribution group already exists",
			wantErr:     true,
			expectedErr: attributiongroups.ErrNameAlreadyExists,
			args: args{
				attributionName: attributionName,
			},
			on: func(f *fields) {
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, customerRef, attributionName).
					Return(nil, nil).
					Once()
			},
		},
		{
			name:        "attribution name is not available when preset attribution group already exists",
			wantErr:     true,
			expectedErr: attributiongroups.ErrPresetNameAlreadyExists,
			args: args{
				attributionName: attributionName,
			},
			on: func(f *fields) {
				f.customersDal.
					On("GetRef", ctx, customerID).
					Return(customerRef).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, customerRef, attributionName).
					Return(nil, attributiongroups.ErrNotFound).
					Once()
				f.attributionGroupsDal.
					On("GetByName", ctx, nilCustomerRef, attributionName).
					Return(nil, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				customersDal:         &customerDalMock.Customers{},
				attributionGroupsDal: &mocks.AttributionGroups{},
			}

			s := &AttributionGroupsService{
				customersDAL:         tt.fields.customersDal,
				attributionGroupsDAL: tt.fields.attributionGroupsDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.validateAttributionGroupName(ctx, customerID, tt.args.attributionName)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsService.validateAttributionGroupName() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

func TestAnalyticsAttributionGroupsService_GetAll(t *testing.T) {
	type fields struct {
		dal *mocks.AttributionGroups
	}

	type args struct {
		ctx                  context.Context
		attributionGroupsIDs []string
	}

	ctx := context.Background()

	attrRef1 := firestore.DocumentRef{}
	attrRef2 := firestore.DocumentRef{}

	var attributionGroups []*attributiongroups.AttributionGroup

	someDalError := errors.New("some dal error")

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		expectedRes []*attributiongroups.AttributionGroup

		on func(*fields)
	}{
		{
			name: "return list of attribution groups",
			args: args{
				ctx:                  ctx,
				attributionGroupsIDs: []string{"111", "222"},
			},
			wantErr: false,
			on: func(f *fields) {
				f.dal.On("GetRef", ctx, "111").Return(&attrRef1)
				f.dal.On("GetRef", ctx, "222").Return(&attrRef2)
				f.dal.On("GetAll", ctx, []*firestore.DocumentRef{
					&attrRef1,
					&attrRef2,
				}).Return(attributionGroups, nil)
			},
			expectedErr: nil,
			expectedRes: attributionGroups,
		},
		{
			name: "return error on dal error",
			args: args{
				ctx:                  ctx,
				attributionGroupsIDs: []string{"111", "222"},
			},
			wantErr: true,
			on: func(f *fields) {
				f.dal.On("GetRef", ctx, "111").Return(&attrRef1)
				f.dal.On("GetRef", ctx, "222").Return(&attrRef2)
				f.dal.On("GetAll", ctx, []*firestore.DocumentRef{
					&attrRef1,
					&attrRef2,
				}).Return(nil, someDalError)
			},
			expectedErr: someDalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				dal: &mocks.AttributionGroups{},
			}

			s := &AttributionGroupsService{
				attributionGroupsDAL: tt.fields.dal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.GetAttributionGroups(tt.args.ctx, tt.args.attributionGroupsIDs)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionGroupsService.GetAttributionGroups() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else if !tt.wantErr {
				assert.Equal(t, tt.expectedRes, got)
			}
		})
	}
}

func setupAttributionData() (*attribution.Attribution, *attribution.Attribution, *attribution.Attribution, *firestore.DocumentRef, *firestore.DocumentRef, *firestore.DocumentRef) {
	customerRef := &firestore.DocumentRef{
		ID:   customerID,
		Path: pathForDocRefs,
	}

	otherCustomerRef := &firestore.DocumentRef{
		ID:   "otherCustomerID",
		Path: pathForDocRefs,
	}

	attributionRef1 := &firestore.DocumentRef{
		ID:   attributionID1,
		Path: pathForDocRefs,
	}

	attributionRef2 := &firestore.DocumentRef{
		ID:   attributionID2,
		Path: pathForDocRefs,
	}

	attribution1 := &attribution.Attribution{
		ID:       attributionID1,
		Customer: customerRef,
	}

	attribution2 := &attribution.Attribution{
		ID:       attributionID2,
		Customer: customerRef,
	}

	nonResourceAttribution := &attribution.Attribution{
		ID:       attributionID2,
		Customer: otherCustomerRef,
		Type:     "custom",
	}

	return attribution1, attribution2, nonResourceAttribution, attributionRef1, attributionRef2, customerRef
}
