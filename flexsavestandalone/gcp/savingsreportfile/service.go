package savingsreportfile

import (
	"context"
	"fmt"

	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	pdfGenerator "github.com/doitintl/hello/scheduled-tasks/pdftemplategenerator"
)

const (
	docTemplateID            string = "1FvtaxW9bShYN8ngNyerqQiaPlYs8SLcDXD1ilRlKLIw"
	reportsFolderName        string = "Standalone Savings Reports"
	savingsReportDocName     string = "GCP Savings Report"
	annualSavingsPlaceHolder string = "annual_savings"
)

type StandaloneSavingsReport struct {
	AnnualSavings           string `form:"annualSavings"`
	AnnualSavingsWithGrowth string `form:"annualSavingsWithGrowth"`
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
	}

	return pdfGeneratorService.GetTemplateFileWithReplacedValues(reportsFolderName, savingsReportDocName, changes)
}
