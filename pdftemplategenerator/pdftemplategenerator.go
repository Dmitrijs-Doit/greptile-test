package pdftemplategenerator

import (
	"context"
	"fmt"

	"google.golang.org/api/docs/v1"

	customerDrive "github.com/doitintl/hello/scheduled-tasks/customer/drive"
)

type Service struct {
	customerID           string
	docTemplateID        string
	sharedDriveFolderID  string
	customerDriveService customerDrive.Service
}

type PlaceHolderChange struct {
	PlaceHolder string
	TextReplace string
}

func NewService(ctx context.Context, customerID string, docTemplateID string, sharedDriveFolderID string) (*Service, error) {
	customerDriveService, err := customerDrive.NewGoogleDriveService(ctx, sharedDriveFolderID)
	if err != nil {
		return nil, err
	}

	return &Service{
		customerID,
		docTemplateID,
		sharedDriveFolderID,
		customerDriveService,
	}, nil
}

func (s *Service) GetTemplateFileWithReplacedValues(folderName string, fileName string, changes []PlaceHolderChange) ([]byte, error) {
	fileID, err := s.copyTemplateDocFileToCustomerFolder(folderName, fileName)
	if err != nil {
		return nil, err
	}

	err = s.replaceValuesInTemplateDocument(fileID, changes)
	if err != nil {
		return nil, err
	}

	return s.customerDriveService.ExportFileAsPDF(fileID)
}

func (s *Service) copyTemplateDocFileToCustomerFolder(folderName, fileName string) (string, error) {
	// create or return an existing folder ID by name
	folderID, err := s.customerDriveService.CreateFolder(s.sharedDriveFolderID, folderName)
	if err != nil {
		return "", err
	}

	// copy new template into the folder
	fileID, err := s.customerDriveService.CopyFile(s.docTemplateID, folderID, fileName)
	if err != nil {
		return "", err
	}

	return fileID, nil
}

func (s *Service) replaceValuesInTemplateDocument(docID string, changes []PlaceHolderChange) error {
	var requests []*docs.Request

	for _, change := range changes {
		request := &docs.Request{
			ReplaceAllText: &docs.ReplaceAllTextRequest{
				ContainsText: &docs.SubstringMatchCriteria{
					MatchCase: false,
					Text:      fmt.Sprintf("{{%s}}", change.PlaceHolder),
				},
				ReplaceText: change.TextReplace,
			},
		}
		requests = append(requests, request)
	}

	batchUpdateRequest := &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	}

	return s.customerDriveService.ExecuteBatchUpdate(docID, batchUpdateRequest)
}
