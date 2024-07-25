package savingsreportfile

import (
	"context"
	"fmt"

	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	pdfGenerator "github.com/doitintl/hello/scheduled-tasks/pdftemplategenerator"
)

const (
	docTemplateID                string = "1ucJ7dSX7aGW7JCyC0gP4exAHJJ5JWJvE0BHb1nTS7GI"
	reportsFolderName            string = "Standalone Savings Reports"
	savingsReportDocName         string = "AWS Savings Report"
	annualSavingsPlaceHolder     string = "annual_savings"
	lastMonthPlaceHolder         string = "last_month_spend"
	spendWithFlexsavePlaceHolder string = "spend_with_flexsave"
)

type StandaloneSavingsReport struct {
	AnnualSavings           string `form:"annualSavings"`
	AnnualSavingsWithGrowth string `form:"annualSavingsWithGrowth"`
	LastMonthComputeSpend   string `form:"lastMonthComputeSpend"`
	LastMonthWithFlexsave   string `form:"lastMonthWithFlexsave"`
}

type Service struct {
	customersDAL customerDal.Customers
}

func NewService(conn *connection.Connection) *Service {
	return &Service{
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
	}
}

func (s *Service) CreateSavingsReportFile(ctx context.Context, customerID string, savingsReport StandaloneSavingsReport) ([]byte, error) {
	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	pdfGeneratorService, _ := pdfGenerator.NewService(ctx, customerID, docTemplateID, *customer.SharedDriveFolderID)

	changes := []pdfGenerator.PlaceHolderChange{
		{annualSavingsPlaceHolder, fmt.Sprintf("%s - %s", savingsReport.AnnualSavings, savingsReport.AnnualSavingsWithGrowth)},
		{lastMonthPlaceHolder, savingsReport.LastMonthComputeSpend},
		{spendWithFlexsavePlaceHolder, savingsReport.LastMonthWithFlexsave},
	}

	return pdfGeneratorService.GetTemplateFileWithReplacedValues(reportsFolderName, savingsReportDocName, changes)
}
