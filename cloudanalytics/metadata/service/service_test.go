package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/auth"
	attributionGroupsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/mocks"
	ag "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"

	metadataDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/iface"
	metadataDalMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	iface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func TestNewMetadataService(t *testing.T) {
	type args struct {
		conn *connection.Connection
		log  logger.Provider
	}

	ctx := context.Background()
	conn, err := connection.NewConnection(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args args
		want MetadataService
	}{
		{
			name: "TestNewMetadataService",
			args: args{
				log:  nil,
				conn: conn,
			},
			want: MetadataService{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewMetadataService(ctx, tt.args.log, tt.args.conn)
			assert.NotNil(t, got)
		})
	}
}

func TestMetadataService_ExternalAPIListDimensions(t *testing.T) {
	type fields struct {
		dal                  metadataDalMocks.Metadata
		customerDal          customerMocks.Customers
		attributionGroupsDAL attributionGroupsMocks.AttributionGroups
	}

	type args struct {
		customerID     string
		userID         string
		userEmail      string
		isDoitEmployee bool
	}

	customer := &common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: &firestore.DocumentRef{
				ID: "some-customer-id-1",
			},
		},
	}

	tests := []struct {
		name      string
		on        func(*fields)
		wantedErr error
		args      args
	}{
		{
			name: "successful return of external api list dimensions",
			args: args{
				customerID:     "some-customer-id-1",
				userID:         "some-user-id-1",
				userEmail:      "test1@doit.com",
				isDoitEmployee: true,
			},
			on: func(f *fields) {
				f.dal.On("ListMap", mock.MatchedBy(func(args metadataDalIface.ListArgs) bool {
					return args.CustomerRef.ID == "some-customer-id-1" && args.OrgRef.ID == metadata.DoitOrgID
				})).
					Return(map[metadata.MetadataFieldType][]metadataDalIface.ListItem{
						metadata.MetadataFieldTypeFixed: {
							{
								ID:        "test-id-1",
								Key:       "test-key-1",
								Type:      metadata.MetadataFieldTypeFixed,
								Label:     "test-label-1",
								Timestamp: time.Now(),
							},
						},
					}, nil)

				f.dal.On("FlatAndSortListMap", mock.Anything).Return([]metadataDalIface.ListItem{
					{
						ID:        "test-id-1",
						Key:       "test-key-1",
						Type:      metadata.MetadataFieldTypeFixed,
						Label:     "test-label-1",
						Timestamp: time.Now(),
					},
				})

				f.attributionGroupsDAL.On("List", mock.Anything, mock.Anything, "test1@doit.com").Return([]ag.AttributionGroup{}, nil)

				f.dal.On("GetPresetOrgRef", mock.Anything, mock.Anything).Return(&firestore.DocumentRef{
					ID: metadata.DoitOrgID,
				})

				f.customerDal.On("GetRef", mock.Anything, mock.Anything).Return(&firestore.DocumentRef{
					ID: "some-customer-id-1",
				})
				f.customerDal.On("GetCustomerOrPresentationModeCustomer", mock.Anything, mock.Anything).Return(customer, nil)
				f.customerDal.On("GetCustomer", mock.Anything, mock.Anything).Return(customer, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				dal:                  metadataDalMocks.Metadata{},
				customerDal:          customerMocks.Customers{},
				attributionGroupsDAL: attributionGroupsMocks.AttributionGroups{},
			}

			s := &MetadataService{
				dal:                  &fields.dal,
				customerDal:          &fields.customerDal,
				attributionGroupsDAL: &fields.attributionGroupsDAL,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/someRequest", nil)

			ctx.Set(auth.CtxKeyVerifiedCustomerID, tt.args.customerID)
			ctx.Set(common.CtxKeys.UserID, tt.args.userID)
			ctx.Set(common.CtxKeys.DoitEmployee, tt.args.isDoitEmployee)
			ctx.Set(common.CtxKeys.Email, tt.args.userEmail)

			retVal, apiErr := s.ExternalAPIList(iface.ExternalAPIListArgs{
				Ctx:            ctx,
				IsDoitEmployee: tt.args.isDoitEmployee,
				CustomerID:     tt.args.customerID,
				UserID:         tt.args.userID,
				UserEmail:      tt.args.userEmail,
			})
			if apiErr != nil && tt.wantedErr == nil {
				t.Errorf("Metadata.ExternalAPIListDimensions() error = %v, wantErr %v", retVal, tt.wantedErr)
			}

			if tt.wantedErr != nil {
				assert.Equal(t, tt.wantedErr.Error(), apiErr.Error())
			}
		})
	}
}

func TestMetadataService_ExternalAPIGetDimensions(t *testing.T) {
	type fields struct {
		dal                  metadataDalMocks.Metadata
		customerDal          customerMocks.Customers
		attributionGroupsDAL attributionGroupsMocks.AttributionGroups
	}

	type args struct {
		customerID     string
		userID         string
		userEmail      string
		keyFilter      string
		typeFilter     string
		isDoitEmployee bool
	}

	customer := &common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: &firestore.DocumentRef{
				ID: "some-customer-id-1",
			},
		},
	}

	customer2 := &common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: &firestore.DocumentRef{
				ID: "some-customer-id-2",
			},
		},
	}

	tests := []struct {
		name      string
		on        func(*fields)
		wantedErr error
		args      args
	}{
		{
			name: "successful return of external api get dimensions",
			args: args{
				customerID:     "some-customer-id-1",
				userID:         "some-user-id-1",
				userEmail:      "test1@doit.com",
				keyFilter:      "test-key-1",
				typeFilter:     "test-type-1",
				isDoitEmployee: true,
			},
			on: func(f *fields) {
				f.dal.On("Get", mock.MatchedBy(func(args metadataDalIface.GetArgs) bool {
					return args.CustomerRef.ID == "some-customer-id-1" && args.OrgRef.ID == metadata.DoitOrgID && args.KeyFilter == "test-key-1" && args.TypeFilter == "test-type-1"
				})).Return([]metadataDalIface.GetItem{
					{
						ID:    "test-id-1",
						Key:   "test-key-1",
						Type:  "test-type-1",
						Label: "test-label-1",
					},
				}, nil)

				f.attributionGroupsDAL.On("List", mock.Anything, mock.Anything, "test1@doit.com").Return([]ag.AttributionGroup{}, nil)

				f.dal.On("GetPresetOrgRef", mock.Anything, mock.Anything).Return(&firestore.DocumentRef{
					ID: metadata.DoitOrgID,
				})

				f.customerDal.On("GetRef", mock.Anything, mock.Anything).Return(&firestore.DocumentRef{
					ID: "some-customer-id-1",
				})
				f.customerDal.On("GetCustomerOrPresentationModeCustomer", mock.Anything, mock.Anything).Return(customer, nil)
				f.customerDal.On("GetCustomer", mock.Anything, mock.Anything).Return(customer, nil)
			},
		},
		{
			name: "not found error of external api get dimensions",
			args: args{
				customerID:     "some-customer-id-2",
				userID:         "some-user-id-2",
				userEmail:      "test2@doit.com",
				keyFilter:      "test-key-2",
				typeFilter:     "test-type-2",
				isDoitEmployee: true,
			},
			wantedErr: metadata.ErrNotFound,
			on: func(f *fields) {
				f.dal.On("Get", mock.MatchedBy(func(args metadataDalIface.GetArgs) bool {
					return args.CustomerRef.ID == "some-customer-id-2" && args.OrgRef.ID == metadata.DoitOrgID && args.KeyFilter == "test-key-2" && args.TypeFilter == "test-type-2"
				})).Return([]metadataDalIface.GetItem{}, nil)

				f.attributionGroupsDAL.On("List", mock.Anything, mock.Anything, "test2@doit.com").Return([]ag.AttributionGroup{}, nil)

				f.dal.On("GetPresetOrgRef", mock.Anything, mock.Anything).Return(&firestore.DocumentRef{
					ID: metadata.DoitOrgID,
				})

				f.customerDal.On("GetRef", mock.Anything, mock.Anything).Return(&firestore.DocumentRef{
					ID: "some-customer-id-2",
				})
				f.customerDal.On("GetCustomerOrPresentationModeCustomer", mock.Anything, mock.Anything).Return(customer2, nil)
				f.customerDal.On("GetCustomer", mock.Anything, mock.Anything).Return(customer2, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				dal:                  metadataDalMocks.Metadata{},
				customerDal:          customerMocks.Customers{},
				attributionGroupsDAL: attributionGroupsMocks.AttributionGroups{},
			}

			s := &MetadataService{
				dal:                  &fields.dal,
				customerDal:          &fields.customerDal,
				attributionGroupsDAL: &fields.attributionGroupsDAL,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/someRequest", nil)

			ctx.Set(auth.CtxKeyVerifiedCustomerID, tt.args.customerID)
			ctx.Set(common.CtxKeys.UserID, tt.args.userID)
			ctx.Set(common.CtxKeys.DoitEmployee, tt.args.isDoitEmployee)
			ctx.Set(common.CtxKeys.Email, tt.args.userEmail)

			retVal, apiErr := s.ExternalAPIGet(iface.ExternalAPIGetArgs{
				Ctx:            ctx,
				IsDoitEmployee: tt.args.isDoitEmployee,
				CustomerID:     tt.args.customerID,
				UserID:         tt.args.userID,
				UserEmail:      tt.args.userEmail,
				KeyFilter:      tt.args.keyFilter,
				TypeFilter:     tt.args.typeFilter,
			})
			if apiErr != nil && tt.wantedErr == nil {
				t.Errorf("Metadata.ExternalAPIGet() error = %v, wantErr %v", retVal, tt.wantedErr)
			}

			if tt.wantedErr != nil {
				assert.Equal(t, tt.wantedErr.Error(), apiErr.Error())
			}
		})
	}
}

func TestMetadataService_IsListItemUnique(t *testing.T) {
	type args struct {
		items     []iface.ExternalAPIListItem
		candidate iface.ExternalAPIListItem
	}

	testitems := []iface.ExternalAPIListItem{
		{
			ID:    "test-id-1",
			Type:  "test-type-1",
			Label: "test-label-1",
		},
		{
			ID:    "test-id-2",
			Type:  "test-type-2",
			Label: "test-label-2",
		},
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "item is unique",
			args: args{
				items: testitems,
				candidate: iface.ExternalAPIListItem{
					ID:    "test-id-3",
					Type:  "test-type-3",
					Label: "test-label-3",
				},
			},
			want: true,
		},

		{
			name: "item is unique",
			args: args{
				items: testitems,
				candidate: iface.ExternalAPIListItem{
					ID:    "test-id-1",
					Type:  "test-type-1",
					Label: "test-label-2",
				},
			},
			want: true,
		},
		{
			name: "item is not unique",
			args: args{
				items: testitems,
				candidate: iface.ExternalAPIListItem{
					ID:    "test-id-1",
					Type:  "test-type-1",
					Label: "test-label-1",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetadataService{}
			if got := s.isListItemUnique(tt.args.items, tt.args.candidate); got != tt.want {
				t.Errorf("MetadataService.isGetItemValueUniqueisGetItemValueUnique() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetadataService_IsGetItemValueUnique(t *testing.T) {
	type args struct {
		items     []iface.ExternalAPIGetValue
		candidate iface.ExternalAPIGetValue
	}

	testvalues := []iface.ExternalAPIGetValue{
		{
			Value: "test-value-1",
			Cloud: "test-cloud-1",
		},
		{
			Value: "test-value-2",
			Cloud: "test-cloud-2",
		},
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "value is unique",
			args: args{
				items: testvalues,
				candidate: iface.ExternalAPIGetValue{
					Value: "test-value-3",
					Cloud: "test-cloud-3",
				},
			},
			want: true,
		},
		{
			name: "value is unique",
			args: args{
				items: testvalues,
				candidate: iface.ExternalAPIGetValue{
					Value: "test-value-1",
					Cloud: "test-cloud-2",
				},
			},
			want: true,
		},
		{
			name: "value is not unique",
			args: args{
				items: testvalues,
				candidate: iface.ExternalAPIGetValue{
					Value: "test-value-1",
					Cloud: "test-cloud-1",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetadataService{}
			if got := s.isGetItemValueUnique(tt.args.items, tt.args.candidate); got != tt.want {
				t.Errorf("MetadataService.isGetItemValueUniqueisGetItemValueUnique() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetadataService_MapListItemOptionals(t *testing.T) {
	type args struct {
		input    map[metadata.MetadataFieldType][]metadataDalIface.ListItem
		expected map[metadata.MetadataFieldType][]metadataDalIface.ListItem
	}

	testInput := make(map[metadata.MetadataFieldType][]metadataDalIface.ListItem)
	testInput[metadata.MetadataFieldTypeDatetime] = []metadataDalIface.ListItem{
		{
			Key:   "test-key-1",
			Type:  metadata.MetadataFieldTypeDatetime,
			Label: "test-label-1",
		},
	}
	testInput[metadata.MetadataFieldTypeOptional] = []metadataDalIface.ListItem{
		{
			Key:     "test-key-2",
			SubType: metadata.MetadataFieldTypeProjectLabel,
			Type:    metadata.MetadataFieldTypeOptional,
			Values:  []string{"test-value-1", "test-value-2"},
		},
	}

	expectedOutput := make(map[metadata.MetadataFieldType][]metadataDalIface.ListItem)
	expectedOutput[metadata.MetadataFieldTypeDatetime] = []metadataDalIface.ListItem{testInput[metadata.MetadataFieldTypeDatetime][0]} // copy
	expectedOutput[metadata.MetadataFieldTypeProjectLabel] = []metadataDalIface.ListItem{
		{
			Type:  metadata.MetadataFieldTypeProjectLabel,
			Key:   "test-value-1",
			Label: "test-value-1",
		},
		{
			Type:  metadata.MetadataFieldTypeProjectLabel,
			Key:   "test-value-2",
			Label: "test-value-2",
		},
	}
	expectedOutput[metadata.MetadataFieldTypeOptional] = nil

	tests := []struct {
		name string
		args args
	}{
		{
			name: "map list item optionals",
			args: args{
				input:    testInput,
				expected: expectedOutput,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetadataService{}
			s.mapListItemOptionals(tt.args.input)
			assert.Equal(t, tt.args.expected, tt.args.input)
		})
	}
}

func TestMetadataService_FilterAttrGroups(t *testing.T) {
	type args struct {
		input      []*domain.OrgMetadataModel
		expected   []metadataDalIface.GetItem
		keyFilter  string
		typeFilter string
	}

	testInput := []*domain.OrgMetadataModel{
		{
			Key:    "test-key-1",
			Type:   metadata.MetadataFieldTypeAttributionGroup,
			Label:  "test-label-1",
			Values: []string{"test-value-1", "test-value-2"},
			Cloud:  "test-cloud-1",
		},
		{
			Key:    "test-key-2",
			Type:   metadata.MetadataFieldTypeAttributionGroup,
			Label:  "test-label-2",
			Values: []string{"test-value-2", "test-value-2"},
			Cloud:  "test-cloud-2",
		},
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "filter attr groups",
			args: args{
				keyFilter:  "test-key-1",
				typeFilter: string(metadata.MetadataFieldTypeAttributionGroup),
				input:      testInput,
				expected: []metadataDalIface.GetItem{
					{
						Key:    testInput[0].Key,
						Type:   testInput[0].Type,
						Label:  testInput[0].Label,
						Values: testInput[0].Values,
						Cloud:  testInput[0].Cloud,
					},
				},
			},
		},
		{
			name: "filter no attr groups",
			args: args{
				keyFilter:  "non-existing-key",
				typeFilter: string(metadata.MetadataFieldTypeAttributionGroup),
				input:      testInput,
				expected:   []metadataDalIface.GetItem{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetadataService{}
			filtered := s.filterAttrGroups(tt.args.input, tt.args.keyFilter, tt.args.typeFilter)
			assert.Equal(t, tt.args.expected, filtered)
		})
	}
}
