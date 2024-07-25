package googlecloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
	recommender "google.golang.org/api/recommender/v1beta1"
)

type Recommender struct {
	ProjectID          string `firestore:"projectId"`
	ProjectNumber      int64  `firestore:"projectNumber"`
	Zone               string `firestore:"zone"`
	Name               string `firestore:"name"`
	ReplaceResource    string `firestore:"replaceResource"`
	Description        string `firestore:"description"`
	RecommenderSubtype string `firestore:"recommenderSubtype"`
	LastRefreshTime    string `firestore:"lastRefreshTime"`
	Saving             string `firestore:"saving"`
	Duration           string `firestore:"duration"`
	Category           string `firestore:"category"`
	State              string `firestore:"state"`
	Instance           string `firestore:"instance"`
	CurrentInstance    string `firestore:"currentInstance"`
	Etag               string `firestore:"etag"`
}

type ReqBody struct {
	ProjectID      string `json:"projectId"`
	Zone           string `json:"zone"`
	Instance       string `json:"instance"`
	MachineType    string `json:"machineType"`
	Name           string `json:"name"`
	Etag           string `json:"etag"`
	CustomerID     string `json:"customerId"`
	IsDoitEmployee bool   `json:"isDoitEmployee"`
	UserID         string `json:"userId"`
}

type ResponseBody struct {
	Data string `json:"data"`
}
type TaskBody struct {
	CustomerID string `json:"customerId"`
}

const (
	TestAction    string = "test"
	ReplaceAction string = "replace"
	SuccessState  string = "SUCCEEDED"
)

var (
	ErrorNoPermission    = errors.New("user does not have manage settings permission")
	ErrorGeneric         = errors.New("argh! something went wrong")
	ErrorNoInstanceFound = errors.New("no instance found")
)

func (s *GoogleCloudService) GetCustomersRecommendations(ctx context.Context) error {
	fs := s.Firestore(ctx)
	l := s.loggerProvider(ctx)

	googleDocSnaps, err := fs.CollectionGroup("cloudConnect").Where("cloudPlatform", "==", common.Assets.GoogleCloud).Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, docSnap := range googleDocSnaps {
		var cred common.GoogleCloudCredential
		if err := docSnap.DataTo(&cred); err != nil {
			continue
		}

		if cred.CategoriesStatus["rightsizing-recommendation"] == common.CloudConnectStatusTypeHealthy && cred.Scope != common.GCPScopeProject {
			if err := createRecommenderTask(ctx, cred.Customer.ID); err != nil {
				l.Error(err)
			}
		}
	}

	return nil
}

func (s *GoogleCloudService) UpdateCustomerRecommendations(ctx context.Context, customerID string) error {
	l := s.loggerProvider(ctx)

	clients, err := s.CloudConnect.GetCustomerGCPClient(ctx, customerID)
	if err != nil {
		return err
	}

	var allRecommendations []Recommender

	for _, client := range clients {
		if client.Doc.CategoriesStatus["rightsizing-recommendation"] == common.CloudConnectStatusTypeHealthy {
			orgRecommendation, err := s.GetRecommendations(ctx, client.Doc)
			if err != nil {
				l.Error(err)
				continue
			}

			allRecommendations = append(allRecommendations, orgRecommendation...)
		}
	}

	return s.UpdateCustomerRecommendationsOnFS(ctx, allRecommendations, customerID)
}

func (s *GoogleCloudService) UpdateCustomerRecommendationsOnFS(ctx context.Context, allRecommendations []Recommender, customerID string) error {
	fs := s.Firestore(ctx)
	_, err := fs.Collection("integrations").Doc("google-cloud").Collection("recommender").Doc(customerID).Set(ctx, map[string]interface{}{
		"recommendations":  allRecommendations,
		"customer":         customerID,
		"isServiceEnabled": true,
	})

	return err
}

func (s *GoogleCloudService) GetRecommendations(ctx context.Context, cred common.GoogleCloudCredential) ([]Recommender, error) {
	fs := s.Firestore(ctx)

	customerCredentials := common.NewGcpCustomerAuthService(&cred)

	clientOptions, err := customerCredentials.GetClientOption()
	if err != nil {
		return nil, err
	}

	cloudresourcemanagerService, err := cloudresourcemanager.NewService(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	allCustomerProjects, err := cloudresourcemanagerService.Projects.List().Do()
	if err != nil {
		return nil, err
	}

	if len(allCustomerProjects.Projects) == 0 {
		return nil, errors.New("no projects found")
	}

	if !isServiceEnabled(ctx, cred, allCustomerProjects.Projects[0].ProjectNumber) {
		if common.Production {
			errorMsg := ""
			isEnabeld, err := s.EnableService(ctx, cred.Customer.ID, allCustomerProjects.Projects)

			if err != nil {
				errorMsg = err.Error()
			}

			if _, err := fs.Collection("integrations").Doc("google-cloud").Collection("recommender").Doc(cred.Customer.ID).Set(ctx, map[string]interface{}{
				"customer":         cred.Customer,
				"isServiceEnabled": isEnabeld,
				"errorMsg":         errorMsg,
			}); err != nil {
				return nil, err
			}

			if !isEnabeld {
				return nil, errors.New("not enabeld")
			}
		}
	}

	return getAllProjectsRecommendations(ctx, cred, allCustomerProjects)
}

func (s *GoogleCloudService) ChangeMachineType(ctx context.Context, req ReqBody) (*compute.Operation, error) {
	fs := s.Firestore(ctx)
	l := s.loggerProvider(ctx)

	if ok := req.hasPermissions(ctx, fs); !ok {
		return nil, ErrorNoPermission
	}

	clients, err := s.CloudConnect.GetCustomerGCPClient(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}

	for _, c := range clients {
		customerCredentials := common.NewGcpCustomerAuthService(&c.Doc)

		clientOptions, err := customerCredentials.GetClientOption()
		if err != nil {
			l.Error(err)
			continue
		}

		computeService, err := compute.NewService(ctx, clientOptions)
		if err != nil {
			l.Error(err)
			continue
		}

		l.Println(req.ProjectID)

		rb := &compute.InstancesSetMachineTypeRequest{
			MachineType: req.MachineType,
		}

		resp, err := computeService.Instances.SetMachineType(req.ProjectID, req.Zone, req.Instance, rb).Context(ctx).Do()
		if err != nil {
			l.Error(err)
			continue
		}

		if err := s.UpdateCustomerRecommendations(ctx, req.CustomerID); err != nil {
			l.Error(err)
			continue
		}

		res, err := s.runInstance(ctx, req, req.CustomerID)
		if err != nil || res == nil {
			l.Error(err)
			continue
		}

		return resp, nil
	}

	return nil, ErrorGeneric
}

func (s *GoogleCloudService) GetInstanceStatus(ctx context.Context, req ReqBody) (*compute.Instance, error) {
	fs := s.Firestore(ctx)

	if ok := req.hasPermissions(ctx, fs); !ok {
		return nil, ErrorNoPermission
	}

	clients, err := s.CloudConnect.GetCustomerGCPClient(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}

	for _, c := range clients {
		customerCredentials := common.NewGcpCustomerAuthService(&c.Doc)

		clientOptions, err := customerCredentials.GetClientOption()
		if err != nil {
			continue
		}

		computeService, err := compute.NewService(ctx, clientOptions)
		if err != nil {
			continue
		}

		resp, err := computeService.Instances.Get(req.ProjectID, req.Zone, req.Instance).Context(ctx).Do()
		if err != nil {
			return nil, err
		}

		return resp, nil
	}

	return nil, ErrorGeneric
}

func (s *GoogleCloudService) StopInstance(ctx context.Context, req ReqBody) (*compute.Operation, error) {
	fs := s.Firestore(ctx)

	if ok := req.hasPermissions(ctx, fs); !ok {
		return nil, ErrorNoPermission
	}

	clients, err := s.CloudConnect.GetCustomerGCPClient(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}

	for _, c := range clients {
		customerCredentials := common.NewGcpCustomerAuthService(&c.Doc)

		clientOptions, err := customerCredentials.GetClientOption()
		if err != nil {
			continue
		}

		computeService, err := compute.NewService(ctx, clientOptions)
		if err != nil {
			continue
		}

		resp, err := computeService.Instances.Stop(req.ProjectID, req.Zone, req.Instance).Context(ctx).Do()
		if err != nil {
			continue
		}

		return resp, nil
	}

	return nil, ErrorGeneric
}

func (s *GoogleCloudService) StartInstance(ctx context.Context, req ReqBody) (*compute.Operation, error) {
	fs := s.Firestore(ctx)

	if ok := req.hasPermissions(ctx, fs); !ok {
		return nil, ErrorNoPermission
	}

	res, err := s.runInstance(ctx, req, req.CustomerID)

	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrorNoInstanceFound
	}

	return res, nil
}

func createRecommenderTask(ctx context.Context, customerID string) error {
	taskBody := TaskBody{
		CustomerID: customerID,
	}

	taskBodyAsJSON, err := json.Marshal(taskBody)
	if err != nil {
		return err
	}

	URI := "/tasks/recommender/updateCustomer"

	config := common.CloudTaskConfig{
		Method:       cloudtaskspb.HttpMethod_PUT,
		Path:         URI,
		Queue:        common.TaskQueueGCPRecommenderRightsizing,
		Body:         taskBodyAsJSON,
		ScheduleTime: nil,
	}

	_, err = common.CreateCloudTask(ctx, &config)

	return err
}

func getAllProjectsRecommendations(ctx context.Context, cred common.GoogleCloudCredential, allCustomerProjects *cloudresourcemanager.ListProjectsResponse) ([]Recommender, error) {
	var recommenderArray []Recommender

	customerCredentials := common.NewGcpCustomerAuthService(&cred)

	clientOptions, err := customerCredentials.GetClientOption()
	if err != nil {
		return nil, err
	}

	computeService, err := compute.NewService(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	recommenderService, err := recommender.NewService(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	for _, project := range allCustomerProjects.Projects {
		projectRecommendations, err := getRecommendationsInstanceByProject(ctx, computeService, recommenderService, project)
		if err != nil {
			continue
		}

		recommenderArray = append(recommenderArray, projectRecommendations...)
	}

	return recommenderArray, nil
}

func getRecommendationsInstanceByProject(ctx context.Context, computeService *compute.Service, recommenderService *recommender.Service, project *cloudresourcemanager.Project) ([]Recommender, error) {
	var recommenderArray []Recommender

	req := computeService.Regions.List(project.ProjectId)
	if err := req.Pages(ctx, func(page *compute.RegionList) error {
		for _, region := range page.Items {

			isUsedLocation := isRegionUsed(region.Quotas)
			if !isUsedLocation {
				continue
			}

			for _, zoneResource := range region.Zones {
				zoneIndex := strings.LastIndex(zoneResource, "/") + 1
				zone := zoneResource[zoneIndex:]

				resourceID := fmt.Sprintf("projects/%d/locations/%s/recommenders/google.compute.instance.MachineTypeRecommender", project.ProjectNumber, zone)

				recommendationRes, err := recommenderService.Projects.Locations.Recommenders.Recommendations.List(resourceID).Do()
				if err != nil {
					continue
				}

				recommendationsByZone := getRecommendationsInstanceByZone(ctx, computeService, zone, recommendationRes.Recommendations, project)
				recommenderArray = append(recommenderArray, recommendationsByZone...)

			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return recommenderArray, nil
}

func getRecommendationsInstanceByZone(ctx context.Context, computeService *compute.Service, zone string, recommendations []*recommender.GoogleCloudRecommenderV1beta1Recommendation, project *cloudresourcemanager.Project) []Recommender {
	var recommenderArray []Recommender

	for _, recommend := range recommendations {
		if recommend.PrimaryImpact.Category != "COST" {
			continue
		}

		replaceResource, instanceResource, currentInstance := getInstanceAction(recommend.Content.OperationGroups)

		f := recommend.PrimaryImpact.CostProjection.Cost.Nanos * int64(-1)
		nanos := strconv.FormatInt(f, 10)

		currencyCode := "$"
		if recommend.PrimaryImpact.CostProjection.Cost.CurrencyCode != "USD" {
			currencyCode = recommend.PrimaryImpact.CostProjection.Cost.CurrencyCode
		}

		isRoundUp := 0
		i, _ := strconv.Atoi(nanos[:1])

		if i > 4 {
			isRoundUp = 1
		}

		currentInstanceResourceIndex := strings.LastIndex(instanceResource, "/") + 1

		instanceRes, err := computeService.Instances.Get(project.ProjectId, zone, instanceResource[currentInstanceResourceIndex:]).Context(ctx).Do()
		if err != nil {
			continue
		}

		zoneIndex := strings.LastIndex(instanceRes.MachineType, "zones/")

		state := recommend.StateInfo.State

		if replaceResource == instanceRes.MachineType[zoneIndex:] {
			state = SuccessState
		}

		recommenderArray = append(recommenderArray, Recommender{
			ProjectID:          project.ProjectId,
			ProjectNumber:      project.ProjectNumber,
			Zone:               zone,
			Name:               recommend.Name,
			ReplaceResource:    replaceResource,
			Description:        recommend.Description,
			RecommenderSubtype: recommend.RecommenderSubtype,
			LastRefreshTime:    recommend.LastRefreshTime,
			Saving:             fmt.Sprintf("%s%d", currencyCode, recommend.PrimaryImpact.CostProjection.Cost.Units*-1+int64(isRoundUp)),
			Duration:           recommend.PrimaryImpact.CostProjection.Duration,
			Category:           recommend.PrimaryImpact.Category,
			State:              state,
			Instance:           instanceResource,
			CurrentInstance:    currentInstance,
			Etag:               recommend.Etag,
		})
	}

	return recommenderArray
}

func getInstanceAction(operationGroups []*recommender.GoogleCloudRecommenderV1beta1OperationGroup) (string, string, string) {
	var replaceResource, instanceResource, currentInstance string

	for _, op := range operationGroups {
		for _, operation := range op.Operations {
			if operation.Action == TestAction {
				currentInstance = operation.ValueMatcher.MatchesPattern
			}

			if operation.Action == ReplaceAction {
				instanceResource = operation.Resource
				replaceResource = operation.Value.(string)
			}
		}
	}

	return replaceResource, instanceResource, currentInstance
}

func isRegionUsed(quotas []*compute.Quota) bool {
	for _, q := range quotas {
		if q.Usage > 0 {
			return true
		}
	}

	return false
}

func isServiceEnabled(ctx context.Context, cred common.GoogleCloudCredential, projectNumber int64) bool {
	customerCredentials := common.NewGcpCustomerAuthService(&cred)

	clientOptions, err := customerCredentials.GetClientOption()
	if err != nil {
		return false
	}

	recommenderService, err := recommender.NewService(ctx, clientOptions)
	if err != nil {
		return false
	}

	recommendationRes, err := recommenderService.Projects.Locations.Recommenders.Recommendations.List("projects/" + fmt.Sprintf("%d", projectNumber) + "/locations/us-central1-a/recommenders/google.compute.instance.MachineTypeRecommender").Do()
	if err != nil {
		return !strings.Contains(err.Error(), "Recommender API has not been used")
	}

	if recommendationRes != nil {
		return true
	}

	return false
}

func (s *GoogleCloudService) runInstance(ctx context.Context, req ReqBody, customerID string) (*compute.Operation, error) {
	clients, err := s.CloudConnect.GetCustomerGCPClient(ctx, customerID)

	if err != nil {
		return nil, err
	}

	for _, c := range clients {
		customerCredentials := common.NewGcpCustomerAuthService(&c.Doc)

		clientOptions, err := customerCredentials.GetClientOption()
		if err != nil {
			continue
		}

		computeService, err := compute.NewService(ctx, clientOptions)
		if err != nil {
			continue
		}

		resp, err := computeService.Instances.Start(req.ProjectID, req.Zone, req.Instance).Context(ctx).Do()
		if err != nil {
			continue
		}

		return resp, nil
	}

	return nil, nil
}

func (s *GoogleCloudService) EnableService(ctx context.Context, customerID string, projects []*cloudresourcemanager.Project) (bool, error) {
	l := s.loggerProvider(ctx)

	clients, err := s.CloudConnect.GetClientOptions(ctx, customerID)
	if err != nil {
		return false, err
	}

	if len(clients) == 0 || len(projects) == 0 {
		return false, nil
	}

	for _, project := range projects {
		for _, c := range clients {
			if err := common.EnableService(ctx, project.ProjectNumber, "recommender", c); err != nil {
				l.Warningf("recommender - service usage enable error: %s", err.Error())
			} else {
				l.Infof("recommender - service usage enabled")
				break
			}
		}
	}

	return true, nil
}

func (req *ReqBody) hasPermissions(ctx context.Context, fs *firestore.Client) bool {
	if req.IsDoitEmployee {
		return false
	}

	var userID = req.UserID

	var userRef = fs.Collection("users").Doc(userID)

	var user common.User

	docSnap, err := userRef.Get(ctx)
	if err != nil {
		return false
	}

	if err := docSnap.DataTo(&user); err != nil {
		return false
	}

	if !user.HasManageSettingsPermission(ctx) {
		return false
	}

	return true
}
