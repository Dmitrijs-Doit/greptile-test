package drive

import (
	"context"
	"fmt"
	"io"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

const (
	pdfMimeType    string = "application/pdf"
	folderMimeType string = "application/vnd.google-apps.folder"
	sheetsMimeType string = "application/vnd.google-apps.spreadsheet"

	// teamDriveID Target team drive ID (Finance in prod, Engineering in dev)
	teamDriveID    string = "0APq0GFH5dedFUk9PVA"
	devTeamDriveID string = "0AEaZfyctbKovUk9PVAz"

	// parentFolder for all invoices
	parentFolder    string = "17kfzlhsdla3ZtwaYXla_5F3VZ-kQxwBN"
	devParentFolder string = "1WU9fmEGnEtMny7X-YM5FbtNdUxUXPjPI"

	devDriveName string = "1WU9fmEGnEtMny7X-YM5FbtNdUxUXPjPI"
	email        string = "dror@doit.com"
)

var formatSheetWidths = []int64{25, 70, 50, 150, 160, 80, 175, 400, 40, 100, 40, 40, 50, 50, 50, 50, 200, 100, 40, 150, 100}

type SheetInfo struct {
	Id                    int64
	Title                 string
	FormatSheetWidths     bool
	IncludeInVerification bool
}
type service struct {
	sharedDriveFolderID string
	formatSheetWidths   []int64
	driveService        *drive.Service
	docsService         *docs.Service
	sheetsService       *sheets.Service
}

func NewGoogleDriveService(ctx context.Context, sharedDriveFolderID string) (Service, error) {
	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretGoogleDrive)
	if err != nil {
		return nil, err
	}

	serviceConfig, err := google.JWTConfigFromJSON(data, drive.DriveFileScope, drive.DriveScope, drive.DriveAppdataScope, drive.DriveMetadataScope)
	if err != nil {
		return nil, err
	}

	clientOpt := option.WithHTTPClient(serviceConfig.Client(ctx))

	driveService, err := drive.NewService(ctx, clientOpt)
	if err != nil {
		return nil, err
	}

	docsService, err := docs.NewService(ctx, clientOpt)
	if err != nil {
		return nil, err
	}

	sheetsService, err := sheets.NewService(ctx, clientOpt)
	if err != nil {
		return nil, err
	}

	return &service{
		sharedDriveFolderID,
		formatSheetWidths,
		driveService,
		docsService,
		sheetsService,
	}, nil
}

func (c *service) CopyFile(srcDocID string, destFolderID string, destFileName string) (string, error) {
	file, err := c.driveService.Files.Copy(srcDocID, &drive.File{
		Parents:     []string{destFolderID},
		TeamDriveId: teamDriveID,
		Name:        destFileName,
	}).SupportsAllDrives(true).Do()
	if err != nil {
		return "", err
	}

	return file.Id, nil
}

func (c *service) CreateFolder(parentFolderID string, folderName string) (string, error) {
	existFolderID, err := c.FindFolder(parentFolderID, folderName)
	if err != nil {
		return "", err
	}

	if len(existFolderID) > 0 {
		return existFolderID, nil
	}

	folder, err := c.driveService.Files.Create(&drive.File{
		Name:        folderName,
		MimeType:    folderMimeType,
		Parents:     []string{parentFolderID},
		TeamDriveId: teamDriveID,
	}).SupportsAllDrives(true).Do()
	if err != nil {
		return "", err
	}

	return folder.Id, nil
}

func (c *service) FindFolder(folderID string, folderName string) (string, error) {
	return c.findByName(folderID, folderName)
}

func (c *service) FindFile(folderID string, fileName string) (string, error) {
	return c.findByName(folderID, fileName)
}

func (c *service) ExportFileAsPDF(docID string) ([]byte, error) {
	pdfFile, err := c.driveService.Files.Export(docID, pdfMimeType).Download()
	if err != nil {
		return nil, err
	}

	bodyPdfByte, err := io.ReadAll(pdfFile.Body)
	if err != nil {
		return nil, err
	}

	return bodyPdfByte, nil
}

func (c *service) findByName(folderID string, name string) (string, error) {
	q := fmt.Sprintf("name = '%s' and '%s' in parents", name, folderID)

	fileList, err := c.driveService.Files.List().Q(q).Corpora("drive").SupportsAllDrives(true).IncludeItemsFromAllDrives(true).DriveId(teamDriveID).Do()
	if err != nil {
		return "", err
	}

	var foundID string

	for _, myFile := range fileList.Files {
		if myFile.Name == name && !myFile.Trashed {
			foundID = myFile.Id
			break
		}
	}

	return foundID, nil
}

func (c *service) ExecuteBatchUpdate(docID string, batchUpdate *docs.BatchUpdateDocumentRequest) error {
	_, err := c.docsService.Documents.BatchUpdate(docID, batchUpdate).Do()
	if err != nil {
		return err
	}

	return nil
}

// Invoicing
func getYearAndMonth(invoiceMonth string) (string, string) {
	parts := strings.Split(invoiceMonth, "-")
	if len(parts) != 2 {
		fmt.Println("Invalid format")
		return "", ""
	}
	year := parts[0]
	month := fmt.Sprintf("%s-%s", year, parts[1])

	return year, month
}

func GetInvoicingTargetSheetDestination(email string, devMode bool, devDriveName *string) (folder string, teamDrive string, writePermissionsUser string) {
	if devMode && devDriveName != nil {
		// dev mode
		return *devDriveName, devTeamDriveID, email
	} else if common.Production {
		// prod environment
		return parentFolder, teamDriveID, "dror@doit.com"
	}
	// dev environment
	return devParentFolder, devTeamDriveID, email
}

func (c *service) CreateYearMonthFolderStructure(parentFolderID string, invoiceMonth string) (string, error) {
	year, month := getYearAndMonth(invoiceMonth)
	// Create year folder
	yearFolderID, err := c.CreateFolder(parentFolderID, year)
	if err != nil {
		return "", fmt.Errorf("failed to create year folder: %v", err)
	}
	// Create month folder within year folder
	monthFolderID, err := c.CreateFolder(yearFolderID, month)
	if err != nil {
		return "", fmt.Errorf("failed to create month folder with root %v: %v", yearFolderID, err)
	}
	return monthFolderID, nil
}

func (c *service) CreateSingleInvoicesFolder(parentFolderID string, assetType string, invoiceMonth string) (string, error) {
	// Create year-month folder structure
	monthFolderID, err := c.CreateYearMonthFolderStructure(parentFolderID, invoiceMonth)
	if err != nil {
		return "", fmt.Errorf("failed to create year-month folder structure: %v", err)
	}
	// Create single-invoices folder within month folder
	singleInvoiceFolderID, err := c.CreateFolder(monthFolderID, "single-invoices")
	if err != nil {
		return "", fmt.Errorf("failed to create drafts folder: %v", err)
	}

	// Create asset type folder within single-invoices folder
	assetTypeFolderID, err := c.CreateFolder(singleInvoiceFolderID, assetType)
	if err != nil {
		return "", fmt.Errorf("failed to create drafts folder: %v", err)
	}

	return assetTypeFolderID, nil
}

func (s *service) CreateSheet(sheetName string, parentFolderID string, teamDriveID string) (*drive.File, error) {
	file, err := s.driveService.Files.Create(&drive.File{
		MimeType:    sheetsMimeType,
		Parents:     []string{parentFolderID},
		Name:        sheetName,
		TeamDriveId: teamDriveID,
	}).SupportsTeamDrives(true).Do()
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (s *service) GetSheetName(sheets []*sheets.Sheet, sheetID int64) (string, error) {
	var sheetName string

	for _, sheet := range sheets {
		if sheetID == sheet.Properties.SheetId {
			sheetName = sheet.Properties.Title
			break
		}
	}
	if sheetName == "" {
		return "", fmt.Errorf("failed to get sheet name by id: %v", sheetID)
	}

	return sheetName, nil
}

func (s *service) AddPermissionsToSheet(writePermissionsUser string, file *drive.File) error {
	// Grant write permissions
	var permissions *drive.Permission
	permissions = &drive.Permission{
		EmailAddress: writePermissionsUser,
		Role:         "writer",
		Type:         "user",
	}

	// Add permissions to sheet file
	if _, err := s.driveService.Permissions.Create(file.Id, permissions).SupportsTeamDrives(true).Do(); err != nil {
		return err
	}
	return nil
}

func (s *service) ProcessInvoiceSheets(file *drive.File, invoiceSheets []SheetInfo, extendedMode bool) (*sheets.Spreadsheet, map[int64]*[][]interface{}, error) {

	sheetRequests := make([]*sheets.Request, 0)
	rowData := make(map[int64]*[][]interface{})

	for _, sheet := range invoiceSheets {
		rowsArr := make([][]interface{}, 0)
		rowData[sheet.Id] = &rowsArr
	}

	// Add sheets and format widths
	for _, sheet := range invoiceSheets {
		sheetRequests = append(sheetRequests,
			&sheets.Request{AddSheet: &sheets.AddSheetRequest{
				Properties: &sheets.SheetProperties{SheetId: sheet.Id, Title: sheet.Title}},
			},
		)

		if !sheet.FormatSheetWidths {
			continue
		}

		for i, w := range formatSheetWidths {
			sheetRequests = append(sheetRequests, &sheets.Request{
				UpdateDimensionProperties: &sheets.UpdateDimensionPropertiesRequest{
					Range: &sheets.DimensionRange{
						SheetId:    sheet.Id,
						Dimension:  "COLUMNS",
						StartIndex: int64(i),
						EndIndex:   int64(i) + 1,
					},
					Properties: &sheets.DimensionProperties{PixelSize: w},
					Fields:     "pixelSize",
				}})
		}
	}

	// Delete the default spreadsheet sheet
	sheetRequests = append(sheetRequests,
		&sheets.Request{DeleteSheet: &sheets.DeleteSheetRequest{SheetId: 0}},
	)

	batchUpdateRequestInput := &sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: false,
		Requests:                     sheetRequests,
	}
	if _, err := s.sheetsService.Spreadsheets.BatchUpdate(file.Id, batchUpdateRequestInput).Do(); err != nil {
		return nil, nil, err
	}

	spreadsheet, err := s.sheetsService.Spreadsheets.Get(file.Id).Do()
	if err != nil {
		return nil, nil, err
	}

	return spreadsheet, rowData, nil
}

func (s *service) AddDataToSpreadsheet(spreadsheet *sheets.Spreadsheet, rowData map[int64]*[][]interface{}) error {
	for id, pValues := range rowData {
		var title string
		title, err := s.GetSheetName(spreadsheet.Sheets, id)
		if err != nil {
			return err
		}

		appendRange := fmt.Sprintf("%s!A1:V1", title)
		appendValuesRange := sheets.ValueRange{
			MajorDimension: "ROWS",
			Range:          appendRange,
			Values:         *pValues,
		}

		_, err = s.sheetsService.Spreadsheets.Values.
			Append(spreadsheet.SpreadsheetId, appendRange, &appendValuesRange).
			ValueInputOption("USER_ENTERED").
			InsertDataOption("INSERT_ROWS").
			IncludeValuesInResponse(false).
			Do()
		if err != nil {
			return err
		}
	}
	return nil
}
