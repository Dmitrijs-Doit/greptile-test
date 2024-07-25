package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cloudTaskMocks "github.com/doitintl/cloudtasks/mocks"
	fsMocks "github.com/doitintl/firestore/mocks"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	cloudAnalyticsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/mocks"
	originDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	contractDalMocks "github.com/doitintl/hello/scheduled-tasks/contract/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/contract/domain"
	customerDalMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	entityDalMocks "github.com/doitintl/hello/scheduled-tasks/entity/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tierDalMocks "github.com/doitintl/tiers/dal/mocks"
)

func TestCreateContract(t *testing.T) {
	contextMock := mock.MatchedBy(func(_ context.Context) bool { return true })
	// Define test cases
	const (
		entityID   = "entityID"
		customerID = "customerID"
		startDate  = "2024-02-15T00:04:00Z"
		tierID     = "TierID"
		typeName   = "navigator"
	)

	customerRef := firestore.DocumentRef{ID: "customerRefID"}

	entityRef := firestore.DocumentRef{ID: "entityRefID"}

	tierRef := firestore.DocumentRef{ID: "tierRefID"}

	accountManagerRef := firestore.DocumentRef{ID: "tierRefID"}

	layout := time.RFC3339

	startDateTime, _ := time.Parse(layout, startDate)

	type fields struct {
		customerDAL        customerDalMocks.Customers
		contractsDAL       contractDalMocks.ContractFirestore
		entityDAL          entityDalMocks.Entites
		accountManagersDAL fsMocks.AccountManagers
		tiersDAL           tierDalMocks.TierEntitlementsIface
		cloudTaskClient    cloudTaskMocks.CloudTaskClient
	}

	tests := []struct {
		name    string
		req     domain.ContractInputStruct
		on      func(f *fields)
		wantErr error
	}{
		{
			name: "success",
			req:  domain.ContractInputStruct{CustomerID: customerID, EntityID: entityID, StartDate: startDate, Tier: tierID, Type: typeName},
			on: func(f *fields) {
				f.entityDAL.On("GetRef", contextMock, entityID).
					Return(&entityRef)

				f.customerDAL.On("GetRef", contextMock, customerID).
					Return(&customerRef)

				f.tiersDAL.On("GetTierRef", contextMock, tierID).
					Return(&tierRef)

				f.accountManagersDAL.On("GetRef", contextMock, tierID).
					Return(&accountManagerRef)

				f.contractsDAL.On("CreateContract", contextMock, pkg.Contract{Customer: &customerRef, Entity: &entityRef, StartDate: &startDateTime, Tier: &tierRef, Type: typeName, Active: true}).
					Return(nil)

				f.tiersDAL.On("GetTier", contextMock, tierRef.ID).
					Return(&pkg.Tier{}, nil)

				f.cloudTaskClient.On("CreateTask", contextMock, mock.AnythingOfType("*iface.Config")).
					Return(nil, nil)
			},
			wantErr: nil,
		},
		{
			name: "failure - saving contract in FS",
			req:  domain.ContractInputStruct{StartDate: startDate},
			on: func(f *fields) {
				f.entityDAL.On("GetRef", contextMock, "").
					Return(&entityRef)

				f.customerDAL.On("GetRef", contextMock, "").
					Return(&customerRef)

				f.tiersDAL.On("GetTierRef", contextMock, "").
					Return(&tierRef)

				f.accountManagersDAL.On("GetRef", contextMock, "").
					Return(&accountManagerRef)

				f.contractsDAL.On("CreateContract", contextMock, pkg.Contract{Customer: &customerRef, StartDate: &startDateTime, Type: "", Active: true}).
					Return(errors.New("Mising required fields, Fail to create contract"))
			},
			wantErr: errors.New("Mising required fields, Fail to create contract"),
		},
		{
			name:    "failure - start date validation failed",
			req:     domain.ContractInputStruct{},
			wantErr: errors.New("parsing time \"\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"\" as \"2006\""),
		},
		{
			name:    "failure - end date validation failed",
			req:     domain.ContractInputStruct{StartDate: startDate, IsCommitment: true},
			wantErr: errors.New("validation failed: either 'CommitmentMonths' or 'EndDate' must be specified, but both are missing"),
		},
		{
			name:    "failure - missing entity",
			req:     domain.ContractInputStruct{StartDate: startDate, ChargePerTerm: 100},
			wantErr: errors.New("entityId must be specified when chargePerTerm is specified"),
		},
		{
			name: "failure - incorrect end date",
			req:  domain.ContractInputStruct{StartDate: startDate, EndDate: "abc"},
			on: func(f *fields) {
				f.entityDAL.On("GetRef", contextMock, "").
					Return(&entityRef)

				f.customerDAL.On("GetRef", contextMock, "").
					Return(&customerRef)

				f.tiersDAL.On("GetTierRef", contextMock, "").
					Return(&tierRef)

				f.accountManagersDAL.On("GetRef", contextMock, "").
					Return(&accountManagerRef)
			},
			wantErr: errors.New("parsing time \"abc\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"abc\" as \"2006\""),
		},
		{
			name: "failure - incorrect contract file",
			req:  domain.ContractInputStruct{StartDate: startDate, ContractFile: &pkg.ContractFile{}},
			on: func(f *fields) {
				f.entityDAL.On("GetRef", contextMock, "").
					Return(&entityRef)

				f.customerDAL.On("GetRef", contextMock, "").
					Return(&customerRef)

				f.tiersDAL.On("GetTierRef", contextMock, "").
					Return(&tierRef)

				f.accountManagersDAL.On("GetRef", contextMock, "").
					Return(&accountManagerRef)
			},
			wantErr: errors.New("contract file invalid"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock setup
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &ContractService{
				loggerProvider: logger.FromContext,
				contractsDAL:   &fields.contractsDAL,
				customerDAL:    &fields.customerDAL,
				entityDAL:      &fields.entityDAL,
				tiersDAL:       &fields.tiersDAL,
				conn: &connection.Connection{
					CloudTaskClient: &fields.cloudTaskClient,
				},
			}

			err := s.CreateContract(context.Background(), tt.req)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestCancelContract(t *testing.T) {
	contextMock := mock.MatchedBy(func(_ context.Context) bool { return true })
	customerID := "customerRefID"
	customerRef := firestore.DocumentRef{ID: customerID}

	type fields struct {
		contractsDAL    contractDalMocks.ContractFirestore
		tiersDAL        tierDalMocks.TierEntitlementsIface
		cloudTaskClient cloudTaskMocks.CloudTaskClient
	}

	tests := []struct {
		name       string
		contractID string
		on         func(f *fields)
		wantErr    error
	}{
		{
			name:       "success",
			contractID: "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(
					&pkg.Contract{
						Customer: &customerRef,
						Tier:     &firestore.DocumentRef{ID: "tierID"},
						Type:     string(pkg.NavigatorPackageTierType),
					},
					nil,
				)
				f.contractsDAL.On("CancelContract", contextMock, "contractID").Return(nil)
				f.tiersDAL.On("GetTier", contextMock, "tierID").Return(&pkg.Tier{TrialTier: false}, nil)
				f.cloudTaskClient.On("CreateTask", contextMock, mock.AnythingOfType("*iface.Config")).
					Return(nil, nil)
			},
			wantErr: nil,
		},
		{
			name:       "success - update trial dates",
			contractID: "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(
					&pkg.Contract{
						Customer: &customerRef,
						Tier:     &firestore.DocumentRef{ID: "tierID"},
						Type:     string(pkg.NavigatorPackageTierType),
					},
					nil,
				)
				f.contractsDAL.On("CancelContract", contextMock, "contractID").Return(nil)

				f.tiersDAL.On("GetTier", contextMock, "tierID").Return(&pkg.Tier{TrialTier: true}, nil)
				f.tiersDAL.On(
					"UpdateCustomerTier",
					contextMock,
					&customerRef,
					pkg.NavigatorPackageTierType,
					mock.AnythingOfType("*pkg.CustomerTier")).
					Return(nil)

				f.cloudTaskClient.On("CreateTask", contextMock, mock.AnythingOfType("*iface.Config")).
					Return(nil, nil)
			},
			wantErr: nil,
		},
		{
			name:       "fail",
			contractID: "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(
					&pkg.Contract{Type: string(pkg.NavigatorPackageTierType)},
					nil,
				)
				f.contractsDAL.On("CancelContract", contextMock, "contractID").
					Return(errors.New("some error"))
			},
			wantErr: errors.New("some error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock setup
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &ContractService{
				loggerProvider: logger.FromContext,
				contractsDAL:   &fields.contractsDAL,
				tiersDAL:       &fields.tiersDAL,
				conn: &connection.Connection{
					CloudTaskClient: &fields.cloudTaskClient,
				},
			}

			err := s.CancelContract(context.Background(), tt.contractID)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestContractService_RefreshCustomerTiers(t *testing.T) {
	const (
		customerID  = "customerID"
		packageType = "navigator"
		isActive    = true
		startDate   = "2024-02-15T00:04:00Z"
	)

	layout := time.RFC3339
	startDateTime, _ := time.Parse(layout, startDate)

	var (
		contextMock                  = mock.MatchedBy(func(_ context.Context) bool { return true })
		customerRef                  = firestore.DocumentRef{ID: "customerRefID"}
		customerRef2                 = firestore.DocumentRef{ID: "customerRefID2"}
		tierRef                      = firestore.DocumentRef{ID: "tierRefID"}
		zeroEntitlementsTierRef      = firestore.DocumentRef{ID: "zero-entitlements"}
		heritageTierRef              = firestore.DocumentRef{ID: "heritage"}
		trialTier                    = pkg.Tier{ID: tierRef.ID, TrialTier: true}
		customerContract             = pkg.Contract{StartDate: &startDateTime, Tier: &tierRef, Type: "navigator", Customer: &customerRef, Active: true}
		customerContract2            = pkg.Contract{StartDate: &startDateTime, Tier: &tierRef, Type: "navigator", Customer: &customerRef2, Active: true}
		futureDateStart              = time.Now().Add(5 * time.Hour)
		futureDateEnd                = time.Now().Add(8 * time.Hour)
		now                          = time.Now()
		futureContract               = pkg.Contract{StartDate: &futureDateStart, EndDate: &futureDateEnd, Tier: &tierRef, Type: "navigator", Customer: &customerRef2, Active: false}
		cancelledFutureContract      = pkg.Contract{StartDate: &futureDateStart, EndDate: &now, Tier: &tierRef, Type: "navigator", Customer: &customerRef2, Active: false}
		expiredDate                  = time.Now().Add(-5 * time.Hour)
		expiredCustomerContract      = pkg.Contract{ID: "nav-id", Active: true, StartDate: &time.Time{}, EndDate: &expiredDate, Tier: &tierRef, Type: "navigator", Customer: &customerRef}
		advantageTierRef             = firestore.DocumentRef{ID: "legacy"}
		expiredCustomerSolveContract = pkg.Contract{ID: "solve-id", Active: true, StartDate: &time.Time{}, EndDate: &expiredDate, Tier: &tierRef, Type: "solve", Customer: &customerRef}
	)

	type fields struct {
		customerDAL  customerDalMocks.Customers
		contractsDAL contractDalMocks.ContractFirestore
		tiersDAL     tierDalMocks.TierEntitlementsIface
	}

	tests := []struct {
		name       string
		customerID string
		on         func(f *fields)
		wantErr    error
	}{
		{
			name:       "success - no contracts - set back to default tier",
			customerID: customerID,
			on: func(f *fields) {
				f.customerDAL.On("GetRef", contextMock, customerID).
					Return(&customerRef)

				f.contractsDAL.On("ListCustomerNext10Contracts", contextMock, &customerRef).
					Return([]pkg.Contract{}, nil)

				f.contractsDAL.On("GetContractsByType",
					contextMock,
					&customerRef,
					domain.ContractTypeAWS,
					domain.ContractTypeGoogleCloud,
					domain.ContractTypeAzure,
				).Return([]common.Contract{}, nil)

				f.tiersDAL.On(
					"UpdateCustomerTier",
					contextMock,
					customerContract.Customer,
					pkg.PackageTierType(customerContract.Type),
					mock.AnythingOfType("*pkg.CustomerTier")).
					Return(nil)

				f.tiersDAL.On("GetZeroEntitlementsTierRef", contextMock, mock.AnythingOfType("pkg.PackageTierType")).Return(&firestore.DocumentRef{ID: "zero-entitlements"}, nil)
			},
			wantErr: nil,
		},
		{
			name:       "success - contract found",
			customerID: customerID,
			on: func(f *fields) {
				f.customerDAL.On("GetRef", contextMock, customerID).
					Return(&customerRef)

				f.contractsDAL.On("ListCustomerNext10Contracts", contextMock, &customerRef).
					Return([]pkg.Contract{customerContract}, nil)

				f.contractsDAL.On("SetActiveFlag", contextMock, customerContract.ID, isActive).
					Return(nil)

				f.tiersDAL.On("GetTier", contextMock, tierRef.ID).Return(&pkg.Tier{TrialTier: true}, nil)

				f.customerDAL.On("UpdateCustomerFieldValueDeep", contextMock, customerRef.ID, mock.AnythingOfType("[]string"), mock.AnythingOfType("bool")).Return(nil, nil)

				f.tiersDAL.On("UpdateCustomerTier", contextMock, customerContract.Customer, pkg.PackageTierType(customerContract.Type), mock.AnythingOfType("*pkg.CustomerTier")).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "success - contracts found - only update active tier once",
			customerID: customerID,
			on: func(f *fields) {
				f.customerDAL.On("GetRef", contextMock, customerID).
					Return(&customerRef)

				f.contractsDAL.On("ListCustomerNext10Contracts", contextMock, &customerRef).
					Return([]pkg.Contract{customerContract, customerContract2}, nil)

				f.contractsDAL.On("SetActiveFlag", contextMock, customerContract.ID, isActive).
					Return(nil)

				f.tiersDAL.On("GetTier", contextMock, tierRef.ID).Return(&pkg.Tier{TrialTier: true}, nil)

				f.customerDAL.On("UpdateCustomerFieldValueDeep", contextMock, customerRef.ID, mock.AnythingOfType("[]string"), mock.AnythingOfType("bool")).Return(nil, nil)

				f.tiersDAL.On("UpdateCustomerTier", contextMock, customerContract.Customer, pkg.PackageTierType(customerContract.Type), mock.AnythingOfType("*pkg.CustomerTier")).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "success - expired navigator tier is reset - set back to heritage tier",
			customerID: customerID,
			on: func(f *fields) {
				f.customerDAL.On("GetRef", contextMock, customerID).
					Return(&customerRef)

				f.contractsDAL.On("ListCustomerNext10Contracts", contextMock, &customerRef).
					Return([]pkg.Contract{expiredCustomerContract}, nil)

				f.contractsDAL.On("SetActiveFlag", contextMock, expiredCustomerContract.ID, false).
					Return(nil)

				f.contractsDAL.On("GetContractsByType",
					contextMock,
					&customerRef,
					domain.ContractTypeAWS,
					domain.ContractTypeGoogleCloud,
					domain.ContractTypeAzure,
				).Return([]common.Contract{{StartDate: time.Date(2023, time.January, 1, 12, 0, 0, 0, time.UTC)}}, nil)

				f.tiersDAL.On("GetHeritageTierRef", contextMock, mock.AnythingOfType("pkg.PackageTierType")).Return(&heritageTierRef, nil)

				f.tiersDAL.On("GetTier", contextMock, expiredCustomerContract.Tier.ID).Return(&pkg.Tier{}, nil)

				f.customerDAL.On("UpdateCustomerFieldValueDeep", contextMock, customerRef.ID, mock.AnythingOfType("[]string"), mock.AnythingOfType("bool")).Return(nil, nil)

				f.tiersDAL.On(
					"UpdateCustomerTier",
					contextMock,
					customerContract.Customer,
					pkg.PackageTierType(expiredCustomerContract.Type),
					&pkg.CustomerTier{Tier: &heritageTierRef}).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "success - expired navigator tier is reset - set back to zero-entitlements tier with trial dates updated",
			customerID: customerID,
			on: func(f *fields) {
				f.customerDAL.On("GetRef", contextMock, customerID).
					Return(&customerRef)

				f.contractsDAL.On("ListCustomerNext10Contracts", contextMock, &customerRef).
					Return([]pkg.Contract{expiredCustomerContract}, nil)

				f.contractsDAL.On("SetActiveFlag", contextMock, expiredCustomerContract.ID, false).
					Return(nil)

				f.contractsDAL.On("GetContractsByType",
					contextMock,
					&customerRef,
					domain.ContractTypeAWS,
					domain.ContractTypeGoogleCloud,
					domain.ContractTypeAzure,
				).Return([]common.Contract{}, nil)

				f.tiersDAL.On("GetZeroEntitlementsTierRef", contextMock, mock.AnythingOfType("pkg.PackageTierType")).Return(&zeroEntitlementsTierRef, nil)

				f.tiersDAL.On("GetTier", contextMock, tierRef.ID).Return(&trialTier, nil)

				f.customerDAL.On("UpdateCustomerFieldValueDeep", contextMock, customerRef.ID, mock.AnythingOfType("[]string"), mock.AnythingOfType("bool")).Return(nil, nil)

				f.tiersDAL.On(
					"UpdateCustomerTier",
					contextMock,
					customerContract.Customer,
					pkg.PackageTierType(expiredCustomerContract.Type),
					&pkg.CustomerTier{Tier: &zeroEntitlementsTierRef, TrialStartDate: expiredCustomerContract.StartDate, TrialEndDate: expiredCustomerContract.EndDate}).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "contract start date is nil - set back to default tier",
			customerID: customerID,
			on: func(f *fields) {
				f.customerDAL.On("GetRef", contextMock, customerID).
					Return(&customerRef)

				f.contractsDAL.On("ListCustomerNext10Contracts", contextMock, &customerRef).
					Return([]pkg.Contract{{}}, nil)

				f.contractsDAL.On("GetContractsByType",
					contextMock,
					&customerRef,
					domain.ContractTypeAWS,
					domain.ContractTypeGoogleCloud,
					domain.ContractTypeAzure,
				).Return([]common.Contract{}, nil)

				f.tiersDAL.On(
					"UpdateCustomerTier",
					contextMock,
					customerContract.Customer,
					pkg.PackageTierType(customerContract.Type),
					mock.AnythingOfType("*pkg.CustomerTier")).
					Return(nil)

				f.tiersDAL.On("GetZeroEntitlementsTierRef", contextMock, mock.AnythingOfType("pkg.PackageTierType")).Return(&firestore.DocumentRef{ID: "zero-entitlements"}, nil)
			},
			wantErr: nil,
		},
		{
			name:       "success - expired solve tier is reset - set to advantage tier",
			customerID: customerID,
			on: func(f *fields) {
				f.customerDAL.On("GetRef", contextMock, customerID).
					Return(&customerRef)

				f.contractsDAL.On("ListCustomerNext10Contracts", contextMock, &customerRef).
					Return([]pkg.Contract{expiredCustomerSolveContract, customerContract}, nil)

				f.contractsDAL.On("SetActiveFlag", contextMock, expiredCustomerSolveContract.ID, false).
					Return(nil)

				f.tiersDAL.On("GetTier", contextMock, tierRef.ID).Return(&pkg.Tier{TrialTier: true}, nil)

				f.customerDAL.On("UpdateCustomerFieldValueDeep", contextMock, customerRef.ID, mock.AnythingOfType("[]string"), mock.AnythingOfType("bool")).Return(nil, nil)

				f.tiersDAL.On("UpdateCustomerTier", contextMock, customerContract.Customer, pkg.PackageTierType(customerContract.Type), mock.AnythingOfType("*pkg.CustomerTier")).
					Return(nil)

				f.tiersDAL.On("GetTierRefByName", contextMock, "advantage-only", pkg.SolvePackageTierType).Return(&advantageTierRef, nil)

				f.tiersDAL.On(
					"UpdateCustomerTier",
					contextMock,
					customerContract.Customer,
					pkg.PackageTierType(expiredCustomerSolveContract.Type),
					&pkg.CustomerTier{Tier: &advantageTierRef}).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "success - future contract found - only update active tier once",
			customerID: customerID,
			on: func(f *fields) {
				f.customerDAL.On("GetRef", contextMock, customerID).
					Return(&customerRef)

				f.contractsDAL.On("ListCustomerNext10Contracts", contextMock, &customerRef).
					Return([]pkg.Contract{futureContract, customerContract2}, nil)

				f.tiersDAL.On("GetTier", contextMock, tierRef.ID).Return(&pkg.Tier{TrialTier: true}, nil)

				f.tiersDAL.On("UpdateCustomerTier",
					contextMock,
					futureContract.Customer,
					pkg.PackageTierType(customerContract.Type),
					&pkg.CustomerTier{
						TrialEndDate:   futureContract.EndDate,
						TrialStartDate: futureContract.StartDate,
					},
				).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:       "success - cancelled future contract found - skipped",
			customerID: customerID,
			on: func(f *fields) {
				f.customerDAL.On("GetRef", contextMock, customerID).
					Return(&customerRef)

				f.contractsDAL.On("ListCustomerNext10Contracts", contextMock, &customerRef).
					Return([]pkg.Contract{cancelledFutureContract}, nil)
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock setup
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &ContractService{
				loggerProvider: logger.FromContext,
				contractsDAL:   &fields.contractsDAL,
				customerDAL:    &fields.customerDAL,
				tiersDAL:       &fields.tiersDAL,
			}

			err := s.RefreshCustomerTiers(context.Background(), tt.customerID)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestContractService_AggregateInvoiceData(t *testing.T) {
	contextMock := mock.MatchedBy(func(_ context.Context) bool { return true })

	ctx := context.Background()

	const (
		invalidInvoiceMonth                   = "20240102"
		entityID                              = "entityID"
		startDateStr                          = "2024-01-01"
		endDateStr                            = "2024-01-31"
		customerID                            = "customerID"
		invoiceMonth                          = "2024-02-01"
		contractStartDate                     = "2024-02-28"
		marchContractStartDate                = "2024-03-01"
		contractEndDate                       = "2024-11-27"
		contractEndDate3Months                = "2024-05-27"
		contractSolveAcceleratorStartDate     = "2024-03-01"
		contractSolveAcceleratorEndDate       = "2024-04-26"
		contractSolveAcceleratorFutureEndDate = "2026-04-26"
		invoiceMonthSolveAccelerator          = "2024-04-01"
		marchInvoiceMonth                     = "2024-03-01"
	)

	startDate, _ := time.Parse("2006-01-02", startDateStr)

	endDate, _ := time.Parse("2006-01-02", endDateStr)

	customerRef := firestore.DocumentRef{ID: "customerRefID"}

	entityRef := firestore.DocumentRef{ID: "entityID"}

	tierRef := firestore.DocumentRef{ID: "tierID"}

	invoiceMonthStart, invoiceMonthEnd, _ := getMonthStartAndEnd(invoiceMonth)

	contractStartTime, _ := time.Parse("2006-01-02", contractStartDate)

	contractEndTime, _ := time.Parse("2006-01-02", contractEndDate)

	contractSolveAcceleratorStartTime, _ := time.Parse("2006-01-02", contractSolveAcceleratorStartDate)

	contractSolveAcceleratorEndTime, _ := time.Parse("2006-01-02", contractSolveAcceleratorEndDate)

	contractSolveAcceleratorFutureEndTime, _ := time.Parse("2006-01-02", contractSolveAcceleratorFutureEndDate)

	contractEndTime3Months, _ := time.Parse("2006-01-02", contractEndDate3Months)

	firstBillableDayTime, _ := time.Parse("2006-01-02", contractStartDate)

	contractActiveInMarch, _ := time.Parse("2006-01-02", marchContractStartDate)

	var cloudProvider *[]string = nil

	currency := "USD"

	entity := common.Entity{Currency: &currency}

	params := cloudanalytics.RunQueryInput{CustomerID: "customerRefID"}

	contractSolveActive := pkg.Contract{ID: "contractID", StartDate: &startDate, EndDate: &endDate, Type: "solve", PaymentTerm: "monthly", MonthlyFlatRate: 2, ChargePerTerm: 100}

	contractSolve := pkg.Contract{ID: "contractID", StartDate: &contractStartTime, EndDate: &invoiceMonthEnd, Type: "solve", PaymentTerm: "monthly", MonthlyFlatRate: 2, Entity: &entityRef, Customer: &customerRef, Tier: &tierRef, ChargePerTerm: 100, Discount: 1.0}

	contractSolveStartingEndOfBillingMonth := pkg.Contract{ID: "contractID", StartDate: &contractActiveInMarch, EndDate: &contractEndTime3Months, Type: "solve"}

	contractNavAnnual := pkg.Contract{ID: "contractID", StartDate: &contractStartTime, EndDate: &contractEndTime, Type: "navigator", PaymentTerm: "annual", ChargePerTerm: 1000, CommitmentMonths: 9, IsCommitment: true}

	contractSolveAnnual := pkg.Contract{ID: "contractID", StartDate: &contractStartTime, EndDate: &contractEndTime3Months, Type: "solve", PaymentTerm: "annual", MonthlyFlatRate: 2, Entity: &entityRef, Customer: &customerRef, Tier: &tierRef, ChargePerTerm: 500, Discount: 1.0, CommitmentMonths: 3, IsCommitment: true}

	contractList := []pkg.Contract{contractSolveActive, contractSolve}

	contractSolveAccelerator := pkg.Contract{ID: "contractID", StartDate: &contractSolveAcceleratorStartTime, EndDate: &contractSolveAcceleratorEndTime, Type: "solve-accelerator", ChargePerTerm: 1000, Customer: &customerRef}

	contractSolveAcceleratorEstimatedFunding := pkg.Contract{ID: "contractID", StartDate: &contractSolveAcceleratorStartTime, EndDate: &contractSolveAcceleratorEndTime, Type: "solve-accelerator", ChargePerTerm: 1000, Customer: &customerRef, Properties: map[string]interface{}{"estimatedFunding": 1.0}}

	contractSolveAcceleratorFuture := pkg.Contract{ID: "contractID", StartDate: &contractSolveAcceleratorStartTime, EndDate: &contractSolveAcceleratorFutureEndTime, Type: "solve-accelerator", ChargePerTerm: 1000, Customer: &customerRef, Properties: map[string]interface{}{"estimatedFunding": 1.0}}

	billingMonth := invoiceMonthStart.Format(domain.BillingMonthLayout)

	qr, _ := createQueryRequest(firstBillableDayTime, invoiceMonthEnd)

	qr.Accounts = []string{"account1"}

	qr.Currency = fixer.FromString(currency)

	queryResult := cloudanalytics.QueryResult{}

	result := cloudanalytics.QueryResult{Rows: [][]bigquery.Value{{"google-cloud", "temp_id", "2024", "02", 259692.1199288222, 27186318281.221176}}}

	type fields struct {
		contractsDAL contractDalMocks.ContractFirestore
		entityDAL    entityDalMocks.Entites
		tiersDAL     tierDalMocks.TierEntitlementsIface

		cloudAnalyticsService cloudAnalyticsMocks.CloudAnalytics
		cloudTaskClient       cloudTaskMocks.CloudTaskClient
	}

	tests := []struct {
		name         string
		invoiceMonth string
		contractID   string
		on           func(f *fields)
		wantErr      error
	}{
		{
			name:         "happy case - contract not active for billing month",
			invoiceMonth: invoiceMonth,
			contractID:   "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractSolveStartingEndOfBillingMonth, nil)

				f.entityDAL.On("GetEntity", contextMock, "entityID").
					Return(&entity, nil)
			},
			wantErr: nil,
		},

		{
			name:         "happy case",
			invoiceMonth: invoiceMonth,
			contractID:   "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractSolve, nil)

				f.entityDAL.On("GetEntity", contextMock, "entityID").
					Return(&entity, nil)

				f.cloudAnalyticsService.On("GetAccounts", ctx, "customerRefID", cloudProvider, []*report.ConfigFilter{}).Return([]string{"account1"}, nil)

				f.cloudAnalyticsService.On("RunQuery", ctx, &qr, params).Return(&queryResult, nil)

				f.contractsDAL.On("WriteBillingDataInContracts", ctx, domain.ContractBillingAggregatedData{BaseFee: 6.827586206896552}, billingMonth, contractSolve.ID, time.Now().Format("2006-01-02"), true).Return(nil)
			},
			wantErr: nil,
		},

		{
			name:         "happy case with query result-- monthly",
			invoiceMonth: invoiceMonth,
			contractID:   "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractSolve, nil)

				f.entityDAL.On("GetEntity", contextMock, "entityID").
					Return(&entity, nil)

				f.cloudAnalyticsService.On("GetAccounts", ctx, "customerRefID", cloudProvider, []*report.ConfigFilter{}).Return([]string{"account1"}, nil)

				f.cloudAnalyticsService.On("RunQuery", ctx, &qr, params).Return(&result, nil)

				f.contractsDAL.On("WriteBillingDataInContracts", ctx, domain.ContractBillingAggregatedData{BaseFee: 6.827586206896552, Consumption: []pkg.ConsumptionStruct{{Cloud: "google-cloud", Currency: "USD", Final: true, VariableFee: 5193.842398576445}}}, billingMonth, contractSolve.ID, time.Now().Format("2006-01-02"), true).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:         "happy case with query result -- annual solve contract",
			invoiceMonth: invoiceMonth,
			contractID:   "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractSolveAnnual, nil)

				f.entityDAL.On("GetEntity", contextMock, "entityID").
					Return(&entity, nil)

				f.cloudAnalyticsService.On("GetAccounts", ctx, "customerRefID", cloudProvider, []*report.ConfigFilter{}).Return([]string{"account1"}, nil)

				f.cloudAnalyticsService.On("RunQuery", ctx, &qr, params).Return(&result, nil)

				var consumptionList []pkg.ConsumptionStruct

				f.contractsDAL.On("WriteBillingDataInContracts", ctx, domain.ContractBillingAggregatedData{BaseFee: 1000, Consumption: consumptionList}, billingMonth, contractSolve.ID, time.Now().Format("2006-01-02"), true).Return(nil)

				f.contractsDAL.On("WriteBillingDataInContracts", ctx, domain.ContractBillingAggregatedData{BaseFee: 495, Consumption: []pkg.ConsumptionStruct{{Cloud: "google-cloud", Currency: "USD", Final: true, VariableFee: 5193.842398576445}}}, billingMonth, contractSolve.ID, time.Now().Format("2006-01-02"), true).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:         "happy case with query result -- annual navigator contract",
			invoiceMonth: invoiceMonth,
			contractID:   "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractNavAnnual, nil)

				f.entityDAL.On("GetEntity", contextMock, "entityID").
					Return(&entity, nil)

				f.cloudAnalyticsService.On("GetAccounts", ctx, "customerRefID", cloudProvider, []*report.ConfigFilter{}).Return([]string{"account1"}, nil)

				f.cloudAnalyticsService.On("RunQuery", ctx, &qr, params).Return(&result, nil)

				var consumptionList []pkg.ConsumptionStruct

				f.contractsDAL.On("WriteBillingDataInContracts", ctx, domain.ContractBillingAggregatedData{BaseFee: 1000, Consumption: consumptionList}, billingMonth, contractSolve.ID, time.Now().Format("2006-01-02"), true).Return(nil)

				f.contractsDAL.On("WriteBillingDataInContracts", ctx, domain.ContractBillingAggregatedData{BaseFee: 495, Consumption: []pkg.ConsumptionStruct{{Cloud: "google-cloud", Currency: "USD", Final: true, VariableFee: 5193.842398576445}}}, billingMonth, contractSolve.ID, time.Now().Format("2006-01-02"), true).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:         "zero base fee for annual navigator contract",
			invoiceMonth: marchInvoiceMonth,
			contractID:   "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractNavAnnual, nil)

				f.entityDAL.On("GetEntity", contextMock, "entityID").
					Return(&entity, nil)

				f.cloudAnalyticsService.On("GetAccounts", ctx, "customerRefID", cloudProvider, []*report.ConfigFilter{}).Return([]string{"account1"}, nil)

				f.cloudAnalyticsService.On("RunQuery", ctx, &qr, params).Return(&result, nil)

				var consumptionList []pkg.ConsumptionStruct

				f.contractsDAL.On("WriteBillingDataInContracts", ctx, domain.ContractBillingAggregatedData{BaseFee: -0.01, Consumption: consumptionList}, "2024-03", contractNavAnnual.ID, time.Now().Format("2006-01-02"), true).Return(nil)

			},
			wantErr: nil,
		},
		{
			name:         "happy case with query result -- solve accelerator contract",
			invoiceMonth: invoiceMonthSolveAccelerator,
			contractID:   "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractSolveAccelerator, nil)

				f.tiersDAL.On("GetCustomerTier", contextMock, contractSolveAccelerator.Customer, pkg.SolvePackageTierType).
					Return(&pkg.Tier{Name: "standard", PackageType: "solve"}, nil)

				f.entityDAL.On("GetEntity", contextMock, "entityID").
					Return(&entity, nil)

				f.contractsDAL.On("WriteBillingDataInContracts", ctx, domain.ContractBillingAggregatedData{BaseFee: 1000}, "2024-04", contractSolveAccelerator.ID, time.Now().Format("2006-01-02"), true).Return(nil)

			},
			wantErr: nil,
		},
		{
			name:         "solve accelerator contract with premium tier",
			invoiceMonth: invoiceMonthSolveAccelerator,
			contractID:   "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractSolveAccelerator, nil)

				f.tiersDAL.On("GetCustomerTier", contextMock, contractSolveAccelerator.Customer, pkg.SolvePackageTierType).
					Return(&pkg.Tier{Name: "premium", PackageType: "solve"}, nil)

				f.entityDAL.On("GetEntity", contextMock, "entityID").
					Return(&entity, nil)

				f.contractsDAL.On("WriteBillingDataInContracts", ctx, domain.ContractBillingAggregatedData{BaseFee: -0.01}, "2024-04", contractSolveAccelerator.ID, time.Now().Format("2006-01-02"), true).Return(nil)

			},
			wantErr: nil,
		},
		{
			name:         "solve accelerator contract with EstimatedFunding = 1",
			invoiceMonth: invoiceMonthSolveAccelerator,
			contractID:   "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractSolveAcceleratorEstimatedFunding, nil)

				f.tiersDAL.On("GetCustomerTier", contextMock, contractSolveAccelerator.Customer, pkg.SolvePackageTierType).
					Return(&pkg.Tier{Name: "standard", PackageType: "solve"}, nil)

				f.entityDAL.On("GetEntity", contextMock, "entityID").
					Return(&entity, nil)

				f.contractsDAL.On("WriteBillingDataInContracts", ctx, domain.ContractBillingAggregatedData{BaseFee: -0.01}, "2024-04", contractSolveAccelerator.ID, time.Now().Format("2006-01-02"), true).Return(nil)

			},
			wantErr: nil,
		},
		{
			name:         "solve accelerator contract enddate not reached",
			invoiceMonth: invoiceMonthSolveAccelerator,
			contractID:   "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractSolveAcceleratorFuture, nil)

				f.tiersDAL.On("GetCustomerTier", contextMock, contractSolveAccelerator.Customer, pkg.SolvePackageTierType).
					Return(&pkg.Tier{Name: "standard", PackageType: "solve"}, nil)

				f.entityDAL.On("GetEntity", contextMock, "entityID").
					Return(&entity, nil)

				f.contractsDAL.On("WriteBillingDataInContracts", ctx, domain.ContractBillingAggregatedData{BaseFee: -0.01}, "2024-04", contractSolveAccelerator.ID, time.Now().Format("2006-01-02"), true).Return(nil)

			},
			wantErr: nil,
		},
		{
			name:         "Run all contracts",
			invoiceMonth: invoiceMonth,
			contractID:   "",
			on: func(f *fields) {
				f.contractsDAL.On("GetNavigatorAndSolveContracts", contextMock).Return(contractList, nil)

				f.cloudTaskClient.On("CreateTask", contextMock, mock.AnythingOfType("*iface.Config")).
					Return(nil, nil)
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock setup
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &ContractService{
				loggerProvider:        logger.FromContext,
				contractsDAL:          &fields.contractsDAL,
				tiersDAL:              &fields.tiersDAL,
				entityDAL:             &fields.entityDAL,
				cloudAnalyticsService: &fields.cloudAnalyticsService,
				conn: &connection.Connection{
					CloudTaskClient: &fields.cloudTaskClient,
				},
			}

			err := s.AggregateInvoiceData(context.Background(), tt.invoiceMonth, tt.contractID)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestCloudSpendQueryRequest(t *testing.T) {
	now := time.Now()

	invoiceMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	invoiceMonthEnd := invoiceMonthStart.AddDate(0, 1, 0).Add(-time.Second)

	timeSettings := &cloudanalytics.QueryRequestTimeSettings{
		Interval: "month",
		From:     &invoiceMonthStart,
		To:       &invoiceMonthEnd,
	}

	expected := cloudanalytics.QueryRequest{
		CloudProviders: &[]string{common.Assets.AmazonWebServices, common.Assets.GoogleCloud, common.Assets.MicrosoftAzure},
		Origin:         originDomain.QueryOriginOthers,
		Type:           "report",
		TimeSettings:   timeSettings,
		Cols:           getCloudSpendQueryRequestCols(),
		Rows:           getCloudSpendQueryRequestRows(),
		Filters:        getCloudSpendQueryRequestFilter(),
	}

	qr, err := createQueryRequest(invoiceMonthStart, invoiceMonthEnd)

	assert.Equal(t, expected, qr)
	assert.Equal(t, nil, err)
}

func TestCalculateFixedMBDForNavigatorSolve(t *testing.T) {
	const (
		startDateStr         = "2024-01-01"
		endDateStr           = "2024-08-31"
		midMonth             = "2024-02-18"
		invoiceMonthStartStr = "2024-02-01"
		invoiceMonthEndStr   = "2024-02-29"
	)

	invoiceMonthStartTime, invoiceMonthEndTime, _ := getMonthStartAndEnd("")

	startDateContract := time.Now().AddDate(1, 0, 0)

	startDate, _ := time.Parse("2006-01-02", startDateStr)

	endDate, _ := time.Parse("2006-01-02", endDateStr)

	midMonthDate, _ := time.Parse("2006-01-02", midMonth)

	invoiceMonthStart, _ := time.Parse("2006-01-02", invoiceMonthStartStr)

	invoiceMonthEnd, _ := time.Parse("2006-01-02", invoiceMonthEndStr)

	// Contract active for the whole month

	mockContract := pkg.Contract{ChargePerTerm: 3000, Discount: 0, StartDate: &startDate, EndDate: &endDate}

	startForProrating, endForProrating := getBillableDays(mockContract, invoiceMonthStart, invoiceMonthEnd)

	charge := CalculateFixedMBDForNavigatorSolve(mockContract, startForProrating, endForProrating, invoiceMonthEnd)

	assert.Equal(t, float64(3000), charge)

	// Contract starting mid month, 11 days to bill for

	mockContractMidMonth := pkg.Contract{ChargePerTerm: 3000, Discount: 0, StartDate: &midMonthDate, EndDate: &endDate}

	startForProrating, endForProrating = getBillableDays(mockContractMidMonth, invoiceMonthStart, invoiceMonthEnd)

	charge = CalculateFixedMBDForNavigatorSolve(mockContractMidMonth, startForProrating, endForProrating, invoiceMonthEnd)

	assert.Equal(t, float64(1241.3793103448277), charge)

	// Contract ending mid month, 18 days to bill for, contract is inactive

	mockContractInactive := pkg.Contract{ChargePerTerm: 3000, Discount: 0, StartDate: &startDate, EndDate: &midMonthDate}

	startForProrating, endForProrating = getBillableDays(mockContractInactive, invoiceMonthStart, invoiceMonthEnd)

	charge = CalculateFixedMBDForNavigatorSolve(mockContractInactive, startForProrating, endForProrating, invoiceMonthEnd)

	assert.Equal(t, float64(1862.0689655172414), charge)

	// contract starting on the last day of the month

	mockContractLastDay := pkg.Contract{ChargePerTerm: 1000, Discount: 0, StartDate: &invoiceMonthEnd, EndDate: &invoiceMonthEnd}

	startForProrating, endForProrating = getBillableDays(mockContractLastDay, invoiceMonthStart, invoiceMonthEnd)

	charge = CalculateFixedMBDForNavigatorSolve(mockContractLastDay, startForProrating, endForProrating, invoiceMonthEnd)

	assert.Equal(t, float64(34.48275862068966), charge)

	// Contract StartDate in future and endDate in currentInvoiceMonth

	mockContractFutureStartDate := pkg.Contract{ChargePerTerm: 3000, Discount: 0, StartDate: &startDateContract, EndDate: &endDate}

	startForProrating, endForProrating = getBillableDays(mockContractFutureStartDate, invoiceMonthStartTime, invoiceMonthEndTime)

	charge = CalculateFixedMBDForNavigatorSolve(mockContract, startForProrating, endForProrating, invoiceMonthEndTime)

	assert.Equal(t, float64(-0.01), charge)
}

func TestUpdateContract(t *testing.T) {
	contextMock := mock.MatchedBy(func(_ context.Context) bool { return true })
	// Define test cases
	const (
		entityID          = "entityID"
		customerID        = "customerID"
		startDate         = "2024-02-15T00:00:00.000Z"
		tierID            = "TierID"
		navigatorContract = "navigator"
		invoiceMonth      = "2024-02-01"
	)

	isCommitmentTrue := true

	customerRef := firestore.DocumentRef{ID: "customerRefID"}

	entityRef := firestore.DocumentRef{ID: "entityRefID"}

	tierRef := firestore.DocumentRef{ID: "tierRefID"}

	updatedTierRef := firestore.DocumentRef{ID: "TierID"}

	contractStartTime, _ := time.Parse(time.RFC3339, startDate)

	invoiceMonthStart, invoiceMonthEnd, _ := getMonthStartAndEnd(invoiceMonth)

	updates := []firestore.Update{{Path: "startDate", Value: &contractStartTime}, {Path: "type", Value: "navigator"}, {Path: "tier", Value: &updatedTierRef}, {Path: "discount", Value: 1.0}, {Path: "timestamp", Value: firestore.ServerTimestamp}, {Path: "updatedBy", Value: pkg.ContractUpdatedBy{Email: "test@doit.com", Name: "test"}}}

	contractSolve := pkg.Contract{ID: "contractID", StartDate: &invoiceMonthStart, EndDate: &invoiceMonthEnd, Type: "solve", PaymentTerm: "monthly", MonthlyFlatRate: 2, Entity: &entityRef, Customer: &customerRef, Tier: &tierRef, ChargePerTerm: 100, Discount: 1.0}

	contractSolveStartDateUpdated := pkg.Contract{ID: "contractID", StartDate: &contractStartTime, EndDate: &invoiceMonthEnd, Type: "solve", PaymentTerm: "monthly", MonthlyFlatRate: 2, Entity: &entityRef, Customer: &customerRef, Tier: &tierRef, ChargePerTerm: 100, Discount: 1.0}

	type fields struct {
		customerDAL     customerDalMocks.Customers
		contractsDAL    contractDalMocks.ContractFirestore
		entityDAL       entityDalMocks.Entites
		tiersDAL        tierDalMocks.TierEntitlementsIface
		cloudTaskClient cloudTaskMocks.CloudTaskClient
	}

	tests := []struct {
		name       string
		req        domain.ContractUpdateInputStruct
		email      string
		userName   string
		contractID string
		on         func(f *fields)
		wantErr    error
	}{
		{
			name:       "success",
			req:        domain.ContractUpdateInputStruct{StartDate: startDate, Tier: tierID, Type: navigatorContract, Discount: 1.0},
			email:      "test@doit.com",
			userName:   "test",
			contractID: "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractSolve, nil)

				f.tiersDAL.On("GetTierRef", contextMock, tierID).
					Return(&updatedTierRef)

				f.contractsDAL.On("UpdateContract", contextMock, "contractID", updates).
					Return(nil)

				f.cloudTaskClient.On("CreateTask", contextMock, mock.AnythingOfType("*iface.Config")).
					Return(nil, nil)
			},
			wantErr: nil,
		},
		{
			name:       "failure - start date validation failed",
			req:        domain.ContractUpdateInputStruct{StartDate: "2024-01-01"},
			email:      "test@doit.com",
			userName:   "test",
			contractID: "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractSolveStartDateUpdated, nil)
			},

			wantErr: errors.New("parsing time \"2024-01-01\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"\" as \"T\""),
		},
		{
			name:       "failure - end date validation failed",
			req:        domain.ContractUpdateInputStruct{StartDate: startDate, IsCommitment: &isCommitmentTrue},
			email:      "test@doit.com",
			userName:   "test",
			contractID: "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractSolveStartDateUpdated, nil)
			},
			wantErr: errors.New("validation failed: either 'CommitmentMonths' or 'EndDate' must be specified, but both are missing"),
		},
		{
			name:       "failure - incorrect end date",
			req:        domain.ContractUpdateInputStruct{StartDate: startDate, EndDate: "abc"},
			email:      "test@doit.com",
			userName:   "test",
			contractID: "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractSolveStartDateUpdated, nil)
			},
			wantErr: errors.New("parsing time \"abc\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"abc\" as \"2006\""),
		},
		{
			name:       "failure - incorrect contract file",
			req:        domain.ContractUpdateInputStruct{ContractFile: &pkg.ContractFile{}},
			email:      "test@doit.com",
			userName:   "test",
			contractID: "contractID",
			on: func(f *fields) {
				f.contractsDAL.On("GetContractByID", contextMock, "contractID").Return(&contractSolveStartDateUpdated, nil)
			},
			wantErr: errors.New("contract file invalid"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock setup
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &ContractService{
				loggerProvider: logger.FromContext,
				contractsDAL:   &fields.contractsDAL,
				customerDAL:    &fields.customerDAL,
				entityDAL:      &fields.entityDAL,
				tiersDAL:       &fields.tiersDAL,
				conn: &connection.Connection{
					CloudTaskClient: &fields.cloudTaskClient,
				},
			}

			err := s.UpdateContract(context.Background(), tt.contractID, tt.req, tt.email, tt.userName)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestContractService_getBillingDataOfContract(t *testing.T) {
	const (
		contractID          = "contractID"
		lastUpdateDate      = "2024-04-10"
		billingMonth        = "2024-03"
		lastUpdateDateField = "lastUpdateDate"
		finalField          = "final"
		finalValue          = true
		cloudAWS            = "amazon-web-services"
		currencyEUR         = "EUR"
		variableFee         = 298.97
	)

	singleDayAggEmptyConsumption := map[string]interface{}{"baseFee": 1000}

	consumption := []interface{}{
		map[string]interface{}{"final": true, "currency": "EUR", "cloud": "amazon-web-services", "variableFee": 298.97},
	}

	singleDayAgg := map[string]interface{}{"baseFee": 1000, "consumption": consumption}

	type fields struct {
		customerDAL  customerDalMocks.Customers
		contractsDAL contractDalMocks.ContractFirestore
		tiersDAL     tierDalMocks.TierEntitlementsIface
	}

	tests := []struct {
		name           string
		contractID     string
		rawBillingData map[string]map[string]interface{}
		on             func(f *fields)
		want           []pkg.BillingDataBigQuery
		wantErr        error
	}{
		{
			name:           "fail - missing lastUpdateDate",
			contractID:     contractID,
			rawBillingData: map[string]map[string]interface{}{billingMonth: {}},
			on: func(f *fields) {
			},
			wantErr: errors.New("missing lastUpdateDate in contract billingData contractID"),
		},
		{
			name:           "fail - incorrect lastUpdateDate",
			contractID:     contractID,
			rawBillingData: map[string]map[string]interface{}{billingMonth: {lastUpdateDateField: false}},
			on: func(f *fields) {
			},
			wantErr: errors.New("missing/incorrect lastUpdateDate in contract billingData contractID"),
		},
		{
			name:           "fail - no billing data for lastUpdateDate",
			contractID:     contractID,
			rawBillingData: map[string]map[string]interface{}{billingMonth: {lastUpdateDateField: lastUpdateDate}},
			on: func(f *fields) {
			},
			wantErr: errors.New("missing/incorrect billingData record in contract billingData contractID"),
		},
		{
			name:           "fail - missing final flag",
			contractID:     contractID,
			rawBillingData: map[string]map[string]interface{}{billingMonth: {lastUpdateDateField: lastUpdateDate, lastUpdateDate: ""}},
			on: func(f *fields) {
			},
			wantErr: errors.New("incorrect billingData record in contract billingData contractID for date 2024-04-10"),
		},
		{
			name:           "fail - missing final flag",
			contractID:     contractID,
			rawBillingData: map[string]map[string]interface{}{billingMonth: {lastUpdateDateField: lastUpdateDate, lastUpdateDate: nil}},
			on: func(f *fields) {
			},
			wantErr: errors.New("missing final flag in contract billingData contractID"),
		},
		{
			name:           "fail - missing final flag",
			contractID:     contractID,
			rawBillingData: map[string]map[string]interface{}{billingMonth: {lastUpdateDateField: lastUpdateDate, lastUpdateDate: nil, finalField: "true"}},
			on: func(f *fields) {
			},
			wantErr: errors.New("missing/incorrect final flag in contract billingData contractID"),
		},
		{
			name:           "success - empty data",
			contractID:     contractID,
			rawBillingData: map[string]map[string]interface{}{billingMonth: {lastUpdateDateField: lastUpdateDate, lastUpdateDate: nil, finalField: finalValue}},
			on: func(f *fields) {
			},
			wantErr: nil,
			want:    []pkg.BillingDataBigQuery{{Month: billingMonth, BaseFee: 0, Consumption: []pkg.ConsumptionStruct(nil), Final: true, LastUpdateDate: lastUpdateDate}},
		},
		{
			name:           "success - empty consumption",
			contractID:     contractID,
			rawBillingData: map[string]map[string]interface{}{billingMonth: {lastUpdateDateField: lastUpdateDate, lastUpdateDate: singleDayAggEmptyConsumption, finalField: finalValue}},
			on: func(f *fields) {
			},
			wantErr: nil,
			want:    []pkg.BillingDataBigQuery{{Month: billingMonth, BaseFee: 1000, Consumption: []pkg.ConsumptionStruct(nil), Final: finalValue, LastUpdateDate: lastUpdateDate}},
		},
		{
			name:           "success - full data",
			contractID:     contractID,
			rawBillingData: map[string]map[string]interface{}{billingMonth: {lastUpdateDateField: lastUpdateDate, lastUpdateDate: singleDayAgg, finalField: finalValue}},
			on: func(f *fields) {
			},
			wantErr: nil,
			want:    []pkg.BillingDataBigQuery{{Month: billingMonth, BaseFee: 1000, Consumption: []pkg.ConsumptionStruct{{Cloud: cloudAWS, Currency: currencyEUR, Final: finalValue, VariableFee: variableFee}}, Final: finalValue, LastUpdateDate: lastUpdateDate}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &ContractService{
				loggerProvider: logger.FromContext,
				contractsDAL:   &fields.contractsDAL,
				customerDAL:    &fields.customerDAL,
				tiersDAL:       &fields.tiersDAL,
			}

			returned, err := s.getBillingDataOfContract(context.Background(), tt.rawBillingData, tt.contractID)

			if tt.want != nil {
				assert.Equal(t, tt.want, returned)
			}

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestContractService_DeleteContract(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider logger.Provider
		contractsDAL   *contractDalMocks.ContractFirestore
	}

	type args struct {
		ctx        context.Context
		contractID string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "delete contract success",
			args: args{
				ctx:        ctx,
				contractID: "contractID",
			},
			on: func(f *fields) {
				f.contractsDAL.On("DeleteContract", ctx, "contractID").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "delete contract error",
			args: args{
				ctx:        ctx,
				contractID: "contractID",
			},
			on: func(f *fields) {
				f.contractsDAL.On("DeleteContract", ctx, "contractID").Return(errors.New("error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				contractsDAL:   &contractDalMocks.ContractFirestore{},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &ContractService{
				loggerProvider: tt.fields.loggerProvider,
				contractsDAL:   tt.fields.contractsDAL,
			}
			if err := s.DeleteContract(tt.args.ctx, tt.args.contractID); (err != nil) != tt.wantErr {
				t.Errorf("ContractService.DeleteContract() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
