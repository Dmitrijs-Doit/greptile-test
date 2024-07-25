package service

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/dal"
	ds "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/utils"
	"github.com/doitintl/serviceusage"
	cloudresourcemanagerv1 "google.golang.org/api/cloudresourcemanager/v1"
)

type ProjectService struct {
	loggerProvider logger.Provider
	*connection.Connection
	dal          *dal.ServiceAccountsFirestore
	crmv1        *cloudresourcemanagerv1.Service
	serviceUsage *serviceusage.Service
}

func NewProjectService(log logger.Provider, conn *connection.Connection) *ProjectService {
	ctx := context.Background()

	crmv1, err := cloudresourcemanagerv1.NewService(ctx)
	if err != nil {
		return nil
	}

	serviceUsage, err := serviceusage.NewService(ctx, time.Minute)
	if err != nil {
		return nil
	}

	return &ProjectService{
		log,
		conn,
		dal.NewServiceAccountsFirestoreWithClient(log, conn),
		crmv1,
		serviceUsage,
	}
}

func (p *ProjectService) createNewProject(ctx context.Context, folderID string) (*cloudresourcemanagerv1.Project, error) {
	projectID := utils.GetNewProjectName()
	parent := &cloudresourcemanagerv1.ResourceId{
		Id:   folderID,
		Type: "folder",
	}

	projectCreateReq := &cloudresourcemanagerv1.Project{
		ProjectId: projectID,
		Name:      projectID,
		Parent:    parent,
	}

	operation, err := p.crmv1.Projects.Create(projectCreateReq).Do()
	if err != nil {
		return nil, err
	}

	for !operation.Done {
		time.Sleep(2 * time.Second)

		op, err := p.crmv1.Operations.Get(operation.Name).Do()
		if err != nil {
			return nil, err
		}

		operation = op
	}

	if operation.Error != nil {
		return nil, fmt.Errorf("%d: %s", operation.Error.Code, operation.Error.Message)
	}

	project, err := p.crmv1.Projects.Get(projectID).Do()
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (p *ProjectService) CreateNewProject(ctx context.Context, folderID string) (*cloudresourcemanagerv1.Project, error) {
	newProject, err := p.createNewProject(ctx, folderID)
	if err != nil {
		return nil, err
	}

	if err := p.enableRequiredGoogleAPIs(ctx, newProject.ProjectId); err != nil {
		return nil, err
	}

	_, err = p.dal.SetProjects_w_Transaction(ctx, addNewProject, newProject.ProjectId)

	if err != nil {
		return nil, err
	}

	return newProject, nil
}

func addNewProject(docSnap *firestore.DocumentSnapshot, aux interface{}) (interface{}, error) {
	var projects ds.Projects
	if err := docSnap.DataTo(&projects); err != nil {
		return "", err
	}

	if projects.Projects == nil {
		projects.Projects = make(map[string]int, 0)
	}

	newProjectID := aux.(string)

	projects.Projects[newProjectID] = 0
	projects.NextProject = newProjectID

	if projects.CurrentProject == "" {
		projects.CurrentProject = newProjectID
		projects.NextProject = ""
	}

	return projects, nil
}

func (p *ProjectService) enableRequiredGoogleAPIs(ctx context.Context, projectID string) error {
	for _, apiName := range utils.RequiredAPIs {
		if err := p.enableAPI(ctx, utils.GetResourceName(projectID, apiName)); err != nil {
			return err
		}
	}

	return nil
}

func (p *ProjectService) enableAPI(ctx context.Context, resoursceName string) error {
	operation, err := p.serviceUsage.Enable(ctx, resoursceName)
	if err != nil {
		return err
	}

	tickerDuration := 30 * time.Second

	return p.serviceUsage.WaitForOperation(ctx, operation, tickerDuration)
}

func (p *ProjectService) GetCurrentProject(ctx context.Context) (string, error) {
	currentProject, err := p.dal.GetCurrentProject(ctx)
	if err != nil {
		return "", err
	}

	return currentProject, nil
}

func (p *ProjectService) AddServiceAccountToProjects(ctx context.Context, projectID string) error {
	folder := utils.GetFolderID()

	projects, err := p.dal.SetProjects_w_Transaction(ctx, incCurrentProjectSANum, projectID)
	if err != nil {
		return err
	}

	ps := projects.(*ds.Projects)

	if ps.NextProject == "" && (ps.CurrentProject == "" ||
		ps.Projects[ps.CurrentProject] >= utils.MaxServiceAccountsInProject-utils.ServiceAccountsInProjectThreshold) {
		_, err = p.CreateNewProject(ctx, folder)
		if err != nil {
			return err
		}
	}

	return err
}

func incCurrentProjectSANum(docSnap *firestore.DocumentSnapshot, aux interface{}) (interface{}, error) {
	var projects ds.Projects
	if err := docSnap.DataTo(&projects); err != nil {
		return "", err
	}

	pID := aux.(string)

	projects.Projects[pID]++

	if projects.Projects[pID] >= utils.MaxServiceAccountsInProject {
		projects.CurrentProject = projects.NextProject
		projects.NextProject = ""
	}

	return &projects, nil
}
