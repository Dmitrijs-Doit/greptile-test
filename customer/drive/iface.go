//go:generate mockery --output=./mocks --all
package drive

import (
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

// Service Allows to make operation on the Google Drive of the customer
type Service interface {
	CopyFile(srcDocID string, destFolderID string, destFileName string) (string, error)
	CreateFolder(parentFolderID string, folderName string) (string, error)
	FindFolder(folderID string, folderName string) (string, error)
	FindFile(folderID string, fileName string) (string, error)
	ExportFileAsPDF(docID string) ([]byte, error)
	ExecuteBatchUpdate(docID string, batchUpdate *docs.BatchUpdateDocumentRequest) error
	CreateSheet(sheetName string, parentFolderID string, teamDriveID string) (*drive.File, error)
	GetSheetName(sheet []*sheets.Sheet, sheetID int64) (string, error)
	AddPermissionsToSheet(writePermissionsUser string, file *drive.File) error
	ProcessInvoiceSheets(file *drive.File, invoiceSheets []SheetInfo, extendedMode bool) (*sheets.Spreadsheet, map[int64]*[][]interface{}, error)
	AddDataToSpreadsheet(spreadsheet *sheets.Spreadsheet, rowData map[int64]*[][]interface{}) error
	CreateYearMonthFolderStructure(parentFolderID string, invoiceMonth string) (string, error)
	CreateSingleInvoicesFolder(parentFolderID string, assetType string, invoiceMonth string) (string, error)
}
