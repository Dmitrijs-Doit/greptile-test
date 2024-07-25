package invoicing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	contractDalIface "github.com/doitintl/hello/scheduled-tasks/contract/dal/iface"
	customersDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	lookerIface "github.com/doitintl/hello/scheduled-tasks/invoicing/looker/iface"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/priority"
	"github.com/stretchr/testify/assert"
)

/*
To run tests: /opt/homebrew/bin/go test -timeout 60s -run ^Test_ExportInvoices$ github.com/doitintl/hello/scheduled-tasks/invoicing
To set env variable: export GOOGLE_CLOUD_PROJECT=doitintl-cmp-dev
*/

type InvoicingServiceTest struct {
	*logger.Logging
	*connection.Connection
	fixerService           *fixer.FixerService
	contractDAL            contractDalIface.ContractFirestore
	customersDAL           customersDal.Customers
	customerAssetInvoice   CustomerAssetInvoice
	lookerInvoicingService lookerIface.InvoicingService
}

type InvoicingTest struct {
	*logger.Logging
	service               *InvoicingService
	awsCloudHealthService *CloudHealthAWSInvoicingService
}

func Test_ExportInvoices(t *testing.T) {

	testCases := []struct {
		name         string
		year         int64
		month        int64
		types        []string
		override     bool
		uid          string
		email        string
		devMode      bool
		devDriveName *string
	}{
		{
			name:         "Valid parameters - test navigator",
			types:        []string{"navigator"},
			year:         2024,
			month:        1,
			override:     true, // do not change
			uid:          "uid",
			email:        "test@test.com",
			devMode:      true,
			devDriveName: func(s string) *string { return &s }(""),
		},
		{
			name:         "Valid parameters - test solve",
			types:        []string{"solve"},
			year:         2024,
			month:        1,
			override:     true,
			uid:          "uid",
			email:        "test@test.com",
			devMode:      true,
			devDriveName: func(s string) *string { return &s }(""),
		},
		{
			name:         "Valid parameters - test solve",
			types:        []string{"solve-accelerator"},
			year:         2024,
			month:        1,
			override:     true,
			uid:          "uid",
			email:        "laura.p@doit.com",
			devMode:      true,
			devDriveName: func(s string) *string { return &s }(""),
		},
		{
			name:         "Valid parameters - test amazon-web-services",
			types:        []string{"amazon-web-services"},
			year:         2024,
			month:        1,
			override:     true,
			uid:          "uid",
			email:        "test@test.com",
			devMode:      true,
			devDriveName: func(s string) *string { return &s }(""),
		},
		{
			name:         "Valid parameters - google-cloud",
			types:        []string{"google-cloud"},
			year:         2024,
			month:        1,
			override:     true,
			uid:          "uid",
			email:        "test@test.com",
			devMode:      true,
			devDriveName: func(s string) *string { return &s }(""),
		},
		{
			name:         "Valid parameters - amazon-web-services-standalone",
			types:        []string{"amazon-web-services-standalone"},
			year:         2024,
			month:        1,
			override:     true,
			uid:          "uid",
			email:        "",
			devMode:      true,
			devDriveName: func(s string) *string { return &s }(""),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			log, err := logger.NewLogging(ctx)
			if err != nil {
				// Fail the test if logger creation fails
				t.Fatalf("Failed to create logger: %v", err)
			}

			conn, err := connection.NewConnection(ctx, log)
			if err != nil {
				t.Fatalf("Failed to create connection: %v", err)
			}

			// Create an InvoicingServiceTest instance with the logger and connection
			s := InvoicingServiceTest{
				Logging:    log,
				Connection: conn,
			}

			fs := s.Firestore(ctx)
			// Get drive folder name
			_, devDriveName, _ := checkDevModeAllowed(ctx, fs)

			// Create a new InvoicingService instance
			service, _ := NewInvoicingService(log, conn)

			// Call the ExportInvoices method with the test case parameters
			channel, err := service.ExportInvoices(ctx, &ExportInvoicesRequest{
				Types: tc.types,
				Year:  tc.year,
				Month: tc.month,
			}, tc.uid, tc.email, tc.devMode, &devDriveName,
			)

			// Assert that no error occurred during the call
			assert.NoError(t, err)
			// Assert that the channel is not nil
			assert.NotNil(t, channel)
		})
	}
}

func TestExportUtils(t *testing.T) {
	t.Run("getDefaultRevenueAndTargetAccounts", func(t *testing.T) {
		testCases := []struct {
			name               string
			invoiceType        string
			expectedTargetAcc  string
			expectedRevenueAcc string
			companyCode        string
		}{
			{
				name:               "Valid parameters - invoice type is 'gsuite', company code is 'US'",
				invoiceType:        "gsuite",
				expectedTargetAcc:  "404-1",
				expectedRevenueAcc: "108-0",
				companyCode:        "US",
			},
			{
				name:               "Valid parameters - invoice type is 'gsuite', company code is 'NO_US'",
				invoiceType:        "gsuite",
				expectedTargetAcc:  "404-1",
				expectedRevenueAcc: "108-0",
				companyCode:        "NO_US",
			},
			{
				name:               "Valid parameters - invoice type is 'navigator', company code is 'US'",
				invoiceType:        "navigator",
				expectedTargetAcc:  "430",
				expectedRevenueAcc: "108-0",
				companyCode:        "US",
			},
			{
				name:               "Valid parameters - invoice type is 'solve', company code is 'US'",
				invoiceType:        "solve",
				expectedTargetAcc:  "429",
				expectedRevenueAcc: "108-0",
				companyCode:        "US",
			},
			{
				name:               "Valid parameters - invoice type is 'solve-accelerator', company code is 'US'",
				invoiceType:        "solve-accelerator",
				expectedTargetAcc:  "431",
				expectedRevenueAcc: "108-0",
				companyCode:        "US",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Call the getDefaultRevenueAndTargetAccounts method with the test case parameters
				revenue, target := getDefaultRevenueAndTargetAccounts(tc.invoiceType)

				// Assert that the returned revenue account is equal to the expected revenue account
				if revenue != tc.expectedRevenueAcc {
					t.Errorf("Revenue account mismatch: expected %s, got %s", tc.expectedRevenueAcc, revenue)
				}
				// Assert that the returned target account is equal to the expected target account
				if target != tc.expectedTargetAcc {
					t.Errorf("Target account mismatch: expected %s, got %s", tc.expectedTargetAcc, target)
				}
			})
		}
	})

	t.Run("getISRRevenueAndTargetAccounts", func(t *testing.T) {
		testCases := []struct {
			name               string
			invoiceType        string
			expectedTargetAcc  string
			expectedRevenueAcc string
			companyCode        string
			isExportCustomer   bool
		}{
			{
				name:               "Valid parameters - invoice type is 'gsuite', company code is 'ISR'",
				invoiceType:        "gsuite",
				expectedTargetAcc:  "6100",
				expectedRevenueAcc: "2682",
				companyCode:        "ISR",
				isExportCustomer:   false,
			},
			{
				name:               "Valid parameters - invoice type is 'gsuite', company code is 'NO_ISR'",
				invoiceType:        "gsuite",
				expectedTargetAcc:  "6200",
				expectedRevenueAcc: "2684",
				companyCode:        "NO_ISR",
				isExportCustomer:   true,
			},
			{
				name:               "Valid parameters - invoice type is 'navigator', company code is 'ISR'",
				invoiceType:        "navigator",
				expectedTargetAcc:  "430",
				expectedRevenueAcc: "2682",
				companyCode:        "ISR",
				isExportCustomer:   false,
			},
			{
				name:               "Valid parameters - invoice type is 'navigator', company code is 'NO_ISR'",
				invoiceType:        "navigator",
				expectedTargetAcc:  "430-1",
				expectedRevenueAcc: "2684",
				companyCode:        "NO_ISR",
				isExportCustomer:   true,
			},
			{
				name:               "Valid parameters - invoice type is 'solve', company code is 'ISR'",
				invoiceType:        "solve",
				expectedTargetAcc:  "429",
				expectedRevenueAcc: "2682",
				companyCode:        "ISR",
				isExportCustomer:   false,
			},
			{
				name:               "Valid parameters - invoice type is 'solve', company code is 'NO_ISR'",
				invoiceType:        "solve",
				expectedTargetAcc:  "429-1",
				expectedRevenueAcc: "2684",
				companyCode:        "NO_ISR",
				isExportCustomer:   true,
			},
			{
				name:               "Valid parameters - invoice type is 'solve-accelerator', company code is 'ISR'",
				invoiceType:        "solve-accelerator",
				expectedTargetAcc:  "431",
				expectedRevenueAcc: "2682",
				companyCode:        "ISR",
				isExportCustomer:   false,
			},
			{
				name:               "Valid parameters - invoice type is 'solve-accelerator', company code is 'NO_ISR'",
				invoiceType:        "solve-accelerator",
				expectedTargetAcc:  "431-1",
				expectedRevenueAcc: "2684",
				companyCode:        "NO_ISR",
				isExportCustomer:   true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Call the getISRRevenueAndTargetAccounts method with the test case parameters
				revenue, target := getISRRevenueAndTargetAccounts(tc.isExportCustomer, tc.invoiceType)

				// Assert that the returned revenue account is equal to the expected revenue account
				if revenue != tc.expectedRevenueAcc {
					t.Errorf("Revenue account mismatch: expected %s, got %s", tc.expectedRevenueAcc, revenue)
				}
				// Assert that the returned target account is equal to the expected target account
				if target != tc.expectedTargetAcc {
					t.Errorf("Target account mismatch: expected %s, got %s", tc.expectedTargetAcc, target)
				}
			})
		}
	})

	t.Run("mapPriorityCompanyToSheetID", func(t *testing.T) {
		testCases := []struct {
			name        string
			companyCode priority.CompanyCode
			expectedID  int64
		}{
			{
				name:        "Valid parameters - company code is 'UK'",
				companyCode: "doituk",
				expectedID:  9,
			},
			{
				name:        "Valid parameters - company code is 'ISR'",
				companyCode: "doit",
				expectedID:  5,
			},
			{
				name:        "Valid parameters - company code is 'NO_ISR'",
				companyCode: "doitee",
				expectedID:  29,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Call the mapPriorityCompanyToSheetID method with the test case parameters
				id := mapPriorityCompanyToSheetID(tc.companyCode)
				// Assert that the returned ID is equal to the expected ID
				if id != tc.expectedID {
					t.Errorf("Sheet ID mismatch: expected %d, got %d", tc.expectedID, id)
				}
			})
		}
	})

	t.Run("addFirestoreMonthErrorsDoc", func(t *testing.T) {
		testCases := []struct {
			name         string
			invoiceMonth string
			customerRef  *firestore.DocumentRef
			error        string
			invoiceType  string
		}{
			{
				name:         "Valid parameters - test navigator",
				invoiceMonth: "2024-01",
				customerRef:  nil,
				error:        "export_test.go testing",
				invoiceType:  "navigator",
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {

				ctx := context.Background()
				log, _ := logger.NewLogging(ctx)
				conn, _ := connection.NewConnection(ctx, log)

				s := InvoicingServiceTest{
					Logging:    log,
					Connection: conn,
				}

				fs := s.Firestore(ctx)

				res, _ := addFirestoreMonthErrorsDoc(ctx, fs, ErrorDoc{
					InvoiceMonth: tc.invoiceMonth,
					CustomerRef:  tc.customerRef,
					Error:        tc.error,
					Type:         tc.invoiceType,
					Timestamp:    time.Now(),
				})

				assert.NotEmpty(t, res)
				assert.IsType(t, "", res)
			})

		}
	})

	t.Run("addErrorDocSnapsToErrorSheet", func(t *testing.T) {
		testCases := []struct {
			name    string
			docData map[string]interface{}
		}{
			{
				name: "Valid parameters - test navigator",
				docData: map[string]interface{}{
					"customer":  nil,
					"error":     "export_test.go testing",
					"timestamp": time.Date(2024, time.April, 18, 16, 43, 42, 0, time.FixedZone("", 2*60*60)),
					"type":      "navigator",
				},
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {

				docSnapshot := []*firestore.DocumentSnapshot{}

				rowData := make(map[int64]*[][]interface{})
				rowData[sheetERROR] = &[][]interface{}{}

				err := addErrorDocSnapsToErrorSheet(docSnapshot, rowData)

				assert.NoError(t, err)
				assert.IsType(t, nil, err)
				assert.NotNil(t, rowData)
			})

		}
	})

	t.Run("getBillingProfileParams", func(t *testing.T) {
		currency := "ILS"
		country := "Israel"

		pCurrency := &currency
		pCountry := &country

		testCases := []struct {
			name                    string
			invoiceType             string
			entity                  *common.Entity
			expectedPriorityCompany string
			expectedSheetID         int64
		}{
			{
				name:        "Valid parameters - test navigator",
				invoiceType: "navigator",
				entity: &common.Entity{
					PriorityCompany: "doit",
					Country:         pCountry,
					Currency:        pCurrency,
				},
				expectedPriorityCompany: "doit",
				expectedSheetID:         sheetISR,
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Call the getBillingProfileParams method with the test case parameters
				entityParam, err := getBillingProfileParams(tc.entity, tc.invoiceType)

				assert.NoError(t, err)
				assert.NotEmpty(t, entityParam)
				assert.Equal(t, tc.expectedPriorityCompany, fmt.Sprint(entityParam.priorityCompany))
				assert.Equal(t, tc.expectedSheetID, entityParam.sheetID)
			})

		}
	})

}

func TestIsInvoiceIssuable(t *testing.T) {
	testCases := []struct {
		invoice     Invoice
		expectedRes bool
	}{
		{
			invoice:     Invoice{Type: common.Assets.AmazonWebServices},
			expectedRes: true,
		},
		{
			invoice:     Invoice{Type: utils.NavigatorType},
			expectedRes: true,
		},
		{
			invoice:     Invoice{Type: utils.SolveType},
			expectedRes: true,
		},
		{
			invoice:     Invoice{Type: utils.SolveAcceleratorType},
			expectedRes: true,
		},
		{
			invoice:     Invoice{Type: common.Assets.GoogleCloud},
			expectedRes: false,
		},
		{
			invoice:     Invoice{Type: common.Assets.GSuite},
			expectedRes: false,
		},
		{
			invoice:     Invoice{Type: common.Assets.Looker},
			expectedRes: false,
		},
		{
			invoice:     Invoice{Type: common.Assets.MicrosoftAzure},
			expectedRes: false,
		},
		{
			invoice:     Invoice{Type: common.Assets.Office365},
			expectedRes: false,
		},
		{
			invoice:     Invoice{Type: common.Assets.AmazonWebServicesStandalone},
			expectedRes: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.invoice.Type, func(t *testing.T) {
			assert.Equal(t, tc.expectedRes, isInvoiceIssuable(tc.invoice))
		})
	}
}

func TestGetIssuedSheet(t *testing.T) {
	issuingDate := "2024-05-16"

	testCases := []struct {
		firstIssuingDate string
		issuingDate      string
		expectedSheet    int64
	}{
		{
			firstIssuingDate: "2024-05-01",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED,
		},
		{
			firstIssuingDate: "2024-05-02",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED,
		},
		{
			firstIssuingDate: "2024-05-03",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED3,
		},
		{
			firstIssuingDate: "2024-05-04",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED4,
		},
		{
			firstIssuingDate: "2024-05-05",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED5,
		},
		{
			firstIssuingDate: "2024-05-06",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED6,
		},
		{
			firstIssuingDate: "2024-05-07",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED7,
		},
		{
			firstIssuingDate: "2024-05-08",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED8,
		},
		{
			firstIssuingDate: "2024-05-09",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED9,
		},
		{
			firstIssuingDate: "2024-05-10",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED10,
		},
		{
			firstIssuingDate: "2024-05-11",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED11,
		},
		{
			firstIssuingDate: "2024-05-12",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED12,
		},
		{
			firstIssuingDate: "2024-05-13",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED,
		},
		{
			firstIssuingDate: "2024-05-20",
			issuingDate:      issuingDate,
			expectedSheet:    sheetISSUED,
		},
		{
			firstIssuingDate: "",
			issuingDate:      "2024-05-20",
			expectedSheet:    sheetISSUED,
		},
		{
			firstIssuingDate: "",
			issuingDate:      "2024-05-03",
			expectedSheet:    sheetISSUED3,
		},
		{
			firstIssuingDate: "",
			issuingDate:      "2024-05-01",
			expectedSheet:    sheetISSUED,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			issuingTimestamp, _ := time.Parse(time.DateOnly, tc.issuingDate)
			firstIssuingTimestamp, _ := time.Parse(time.DateOnly, tc.firstIssuingDate)
			assert.Equal(t, tc.expectedSheet, getIssuedSheet(issuingTimestamp, firstIssuingTimestamp))
		})
	}
}

func TestGetProductLabels(t *testing.T) {
	testCases := []struct {
		assetTypes    []string
		productLabels []string
		extendedMode  bool
		err           error
	}{
		{
			assetTypes:    []string{common.Assets.AmazonWebServices, common.Assets.GoogleCloud},
			productLabels: []string{"AWS", "GCP"},
			extendedMode:  true,
			err:           nil,
		},
		{
			assetTypes:    []string{common.Assets.GoogleCloud, common.Assets.Office365},
			productLabels: []string{"GCP", "Office"},
			extendedMode:  false,
			err:           nil,
		},
		{
			assetTypes:    []string{utils.NavigatorType},
			productLabels: []string{"Navigator"},
			extendedMode:  true,
			err:           nil,
		},
		{
			assetTypes:    []string{utils.SolveType},
			productLabels: []string{"Solve"},
			extendedMode:  true,
			err:           nil,
		},
		{
			assetTypes:    []string{utils.SolveAcceleratorType},
			productLabels: []string{"Solve Accelerator"},
			extendedMode:  true,
			err:           nil,
		},
		{
			assetTypes:    []string{"some type"},
			productLabels: nil,
			extendedMode:  false,
			err:           fmt.Errorf("exportInvoices failed, invalid product type some type"),
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			productLabels, extendedMode, err := getProductLabels(tc.assetTypes)
			assert.Equal(t, tc.productLabels, productLabels)
			assert.Equal(t, tc.extendedMode, extendedMode)
			assert.Equal(t, tc.err, err)
		})
	}
}
