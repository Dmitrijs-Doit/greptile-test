package aws

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"
	fsMocks "github.com/doitintl/firestore/mocks"
	"github.com/doitintl/firestore/pkg"
	sharedMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAwsStandaloneService_UpdateSavingsAndRecommendationsCSV2(t *testing.T) {
	var (
		hrCommitmentPurchase        = "0.242"
		savingsPercentage           = "24.34606234"
		savingsPlansCost            = "174.24"
		ROI                         = "39.24117443"
		estimatedCost               = "38.2275776661"
		estimatedMonthSavingsAmount = "69.323458733"
		utilization                 = "99.86111111"
		minSpend                    = "0"
		maxSpend                    = "0.3906"
		averageSpend                = "0.3900575"

		contextMock = mock.MatchedBy(func(_ context.Context) bool { return true })
		accountID   = "123456789"
		customerID  = "1234ABCDE"

		errDefault = errors.New("something went wrong")
	)

	customerRef := &firestore.DocumentRef{
		Parent: &firestore.CollectionRef{
			ID: "customers",
		},
		ID: customerID,
	}
	requestRecommendation := EstimationSummaryCSVRequest{
		OfferingID:                        "abcd-123-efg",
		HourlyCommitmentToPurchase:        hrCommitmentPurchase,
		EstimatedSavingsPlansCost:         savingsPlansCost,
		EstimatedOnDemandCost:             estimatedCost,
		CurrentAverageHourlyOnDemandSpend: averageSpend,
		CurrentMinimumHourlyOnDemandSpend: minSpend,
		CurrentMaximumHourlyOnDemandSpend: maxSpend,
		EstimatedAverageUtilization:       utilization,
		EstimatedMonthlySavingsAmount:     estimatedMonthSavingsAmount,
		EstimatedSavingsPercentage:        savingsPercentage,
		EstimatedROI:                      ROI,
	}

	type fields struct {
		customersDAL  sharedMocks.Customers
		standaloneDAL fsMocks.FlexsaveStandalone
	}

	tests := []struct {
		name              string
		on                func(*fields)
		recommendationCSV EstimationSummaryCSVRequest
		wantErr           error
	}{
		{
			name: "happy path create and update",
			on: func(f *fields) {
				f.customersDAL.On("GetRef", contextMock, customerID).Return(customerRef)

				f.standaloneDAL.On("InitStandaloneOnboarding",
					contextMock,
					fmt.Sprintf("amazon-web-services-%s",
						customerID),
					customerRef).Return(nil)

				f.standaloneDAL.On("UpdateStandaloneOnboarding",
					contextMock,
					pkg.ComposeStandaloneID(customerID, accountID, "AWS"),
					mock.MatchedBy(func(arg *pkg.AWSStandaloneOnboarding) bool {
						isMissingPermissions := arg.IsMissingAWSRole
						hasSavings := arg.Savings != nil
						hasRecommendations := arg.Recommendations[threeYears].SavingsPlansPurchaseRecommendationDetails != nil

						return isMissingPermissions && hasSavings && hasRecommendations
					})).Return(nil)
			},
			recommendationCSV: requestRecommendation,
		},
		{
			name: "happy path update",
			on: func(f *fields) {
				f.customersDAL.On("GetRef", contextMock, customerID).Return(customerRef)

				f.standaloneDAL.On("InitStandaloneOnboarding",
					contextMock, fmt.Sprintf("amazon-web-services-%s",
						customerID),
					customerRef).Return(pkg.ErrorAlreadyExist)

				f.standaloneDAL.On("UpdateStandaloneOnboarding",
					contextMock,
					pkg.ComposeStandaloneID(customerID, accountID, "AWS"),
					mock.MatchedBy(func(arg *pkg.AWSStandaloneOnboarding) bool {
						isMissingPermissions := arg.IsMissingAWSRole
						hasSavings := arg.Savings != nil
						hasRecommendations := arg.Recommendations[threeYears].SavingsPlansPurchaseRecommendationDetails != nil

						return isMissingPermissions && hasSavings && hasRecommendations
					})).Return(nil)
			},
			recommendationCSV: requestRecommendation,
		},
		{
			name: "failed to create",
			on: func(f *fields) {
				f.customersDAL.On("GetRef", contextMock, customerID).Return(customerRef)

				f.standaloneDAL.On("InitStandaloneOnboarding",
					contextMock, fmt.Sprintf("amazon-web-services-%s",
						customerID),
					customerRef).Return(errDefault)
			},
			recommendationCSV: requestRecommendation,
			wantErr:           errDefault,
		},
		{
			name: "failed to update",
			on: func(f *fields) {
				f.customersDAL.On("GetRef", contextMock, customerID).Return(customerRef)

				f.standaloneDAL.On("InitStandaloneOnboarding",
					contextMock, fmt.Sprintf("amazon-web-services-%s",
						customerID),
					customerRef).Return(pkg.ErrorAlreadyExist)

				f.standaloneDAL.On("UpdateStandaloneOnboarding",
					contextMock,
					pkg.ComposeStandaloneID(customerID, accountID, "AWS"),
					mock.MatchedBy(func(arg *pkg.AWSStandaloneOnboarding) bool {
						isMissingPermissions := arg.IsMissingAWSRole
						hasSavings := arg.Savings != nil
						hasRecommendations := arg.Recommendations[threeYears].SavingsPlansPurchaseRecommendationDetails != nil

						return isMissingPermissions && hasSavings && hasRecommendations
					})).Return(errDefault)
			},
			recommendationCSV: requestRecommendation,
			wantErr:           errDefault,
		},
		{
			name: "failed to update after successful create",
			on: func(f *fields) {
				f.customersDAL.On("GetRef", contextMock, customerID).Return(customerRef)

				f.standaloneDAL.On("InitStandaloneOnboarding",
					contextMock, fmt.Sprintf("amazon-web-services-%s",
						customerID),
					customerRef).Return(nil)

				f.standaloneDAL.On("UpdateStandaloneOnboarding",
					contextMock,
					pkg.ComposeStandaloneID(customerID, accountID, "AWS"),
					mock.MatchedBy(func(arg *pkg.AWSStandaloneOnboarding) bool {
						isMissingPermissions := arg.IsMissingAWSRole
						hasSavings := arg.Savings != nil
						hasRecommendations := arg.Recommendations[threeYears].SavingsPlansPurchaseRecommendationDetails != nil

						return isMissingPermissions && hasSavings && hasRecommendations
					})).Return(errDefault)
			},
			recommendationCSV: requestRecommendation,
			wantErr:           errDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &AwsStandaloneService{
				customersDAL:          &fields.customersDAL,
				flexsaveStandaloneDAL: &fields.standaloneDAL,
			}
			err := s.UpdateSavingsAndRecommendationsCSV(context.Background(), tt.recommendationCSV, customerID, accountID)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_getThreeYearsSavings(t *testing.T) {
	errFloatParse := errors.New("strconv.ParseFloat: parsing \"\": invalid syntax")
	tests := []struct {
		name              string
		recommendationCSV EstimationSummaryCSVRequest
		wantErr           error
		want              map[string]float64
	}{
		{
			name:    "failed to convert estimatedSavings",
			wantErr: errFloatParse,
		},
		{
			name:    "failed to convert utilization",
			wantErr: errFloatParse,
			recommendationCSV: EstimationSummaryCSVRequest{
				EstimatedMonthlySavingsAmount: "10.00",
			},
		},
		{
			name:    "failed to convert on demand spend",
			wantErr: errFloatParse,
			recommendationCSV: EstimationSummaryCSVRequest{
				EstimatedMonthlySavingsAmount: "10.00",
				EstimatedAverageUtilization:   "98.99",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getThreeYearsSavings(tt.recommendationCSV)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getThreeYearsSavings() = %v, want %v", got, tt.want)
			}
		})
	}
}
