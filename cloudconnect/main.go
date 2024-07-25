package cloudconnect

import (
	"context"
	"errors"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"

	reportStatus "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/statuses"
	dal "github.com/doitintl/hello/scheduled-tasks/cloudconnect/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/presentations/domain"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"google.golang.org/api/option"
)

type CloudConnectPermissions struct {
	Categories []Category `firestore:"categories"`
}

type GCPClient struct {
	Key []byte
	Doc common.GoogleCloudCredential
}

type Category struct {
	ID                      string   `firestore:"id"`
	Name                    string   `firestore:"name"`
	Permissions             []string `firestore:"permissions"`
	OrgLevelOnlyPermissions []string `firestore:"orgLevelOnlyPermissions"`
}

type WorkloadIdentityFederationConnectionCheckRequest struct {
	CustomerID     string `param:"customerID"`
	CloudConnectID string `json:"cloudConnectId"`
}

type WorkloadIdentityFederationConnectionCheckResponse struct {
	IsConnectionEstablished bool   `json:"isConnectionEstablished"`
	Error                   string `json:"error,omitempty"`
	ErrorDescription        string `json:"errorDescription,omitempty"`
}

type CloudConnectStatusType int

const (
	CloudConnectStatusTypeNotConfigured CloudConnectStatusType = iota
	CloudConnectStatusTypeHealthy
	CloudConnectStatusTypeUnhealthy
	CloudConnectStatusTypeCritical
	CloudConnectStatusTypePartial
)

type CloudConnectService struct {
	loggerProvider logger.Provider
	*connection.Connection
	reportStatusService *reportStatus.ReportStatusesService
	cloudconnectDal     dal.IGcpConnect
}

func NewCloudConnectService(loggerProvider logger.Provider, conn *connection.Connection) *CloudConnectService {
	reportStatusService, err := reportStatus.NewReportStatusesService(loggerProvider, conn)
	if err != nil {
		return nil
	}

	return &CloudConnectService{
		loggerProvider,
		conn,
		reportStatusService,
		dal.NewGcpConnectWithClient(conn.Firestore),
	}
}

func permissionsError(ctx *gin.Context, missingPermissionsArr []string, permission string) {
	missingPermissionsArr = append(missingPermissionsArr, permission)
	ctx.JSON(http.StatusOK, gin.H{
		"error":              "MissingPermissions",
		"missingPermissions": missingPermissionsArr,
	})
}

func (s *CloudConnectService) Health(ctx *gin.Context, customerID string) error {
	l := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)

	permissions, err := getRequiredPermissions(ctx, fs)
	if err != nil {
		return err
	}

	var googleDocSnaps []*firestore.DocumentSnapshot
	if customerID != "" {
		googleDocSnaps, err = fs.Collection("customers").Doc(customerID).Collection("cloudConnect").Where("cloudPlatform", "==", common.Assets.GoogleCloud).Documents(ctx).GetAll()
	} else {
		googleDocSnaps, err = fs.CollectionGroup("cloudConnect").
			Where("cloudPlatform", "==", common.Assets.GoogleCloud).
			Where("clientEmail", "!=", domain.CloudConnectClientEmail). // skip presentation customers service account
			Documents(ctx).GetAll()
	}

	if err != nil {
		return err
	}

	common.RunConcurrentJobsOnCollection(ctx, googleDocSnaps, 5, func(ctx context.Context, docSnap *firestore.DocumentSnapshot) {
		var cred common.GoogleCloudCredential
		if err := docSnap.DataTo(&cred); err != nil {
			docSnap.Ref.Update(ctx, []firestore.Update{
				{FieldPath: []string{"status"}, Value: common.CloudConnectStatusTypeCritical},
			})

			return
		}

		countHealtyStatus := 0
		newStatus := common.CloudConnectStatusTypeHealthy

		for _, k := range permissions.Categories {
			// cloud connect that are project scoped and can't have "sandboxes" permissions
			if cred.Scope == common.GCPScopeProject && k.ID == "sandboxes" {
				continue
			}

			categoryPermission, err := getGoogleCloudPermissions(ctx, fs, []string{k.ID}, cred.Scope)
			if err != nil {
				l.Errorf("failed fetching required permissions of category %s for %s, %s", k.ID, cred.Customer.ID, err)
				continue
			}

			categoryStatus, _, err := common.TestCloudConnectPermissions(ctx, k.ID, categoryPermission, &cred)
			if err != nil {
				l.Errorf("failed testing category %s status for %s, %s", k.ID, cred.Customer.ID, err)
				continue
			}

			if categoryStatus == common.CloudConnectStatusTypeUnhealthy {
				if k.ID == "core" {
					if cred.Scope == common.GCPScopeProject {
						//skipping the update unhealthy to core category
						continue
					} else {
						newStatus = common.CloudConnectStatusTypeUnhealthy
					}
				} else {
					categoryStatus = common.CloudConnectStatusTypeNotConfigured
				}
			}

			if k.ID == "bigquery-finops" && categoryStatus == common.CloudConnectStatusTypeHealthy && newStatus == common.CloudConnectStatusTypeHealthy {
				// check for finops sink
				if !isSinkExist(ctx, fs, cred.ClientID) && common.Production {
					l.Infof("customer id: ", cred.Customer.ID)

					form := RequestServiceAccount{
						Location: "US",
					}

					docID := common.CloudConnectDocID(common.Assets.GoogleCloud, cred.ClientID)
					if err := s.CreateSinkForCustomer(ctx.(*gin.Context), cred.Customer.ID, form, docID); err != nil {
						l.Errorf("create sink failed for customer %s, %s", cred.Customer.ID, err)
					}
				}
			}

			if err := common.UpdateCategoryStatus(ctx, cred.Customer, categoryStatus, cred.CloudPlatform, cred.ClientID, k.ID); err != nil {
				l.Errorf("failed updating category %s status for %s, %s", k.ID, cred.Customer.ID, err)
			}

			if categoryStatus == common.CloudConnectStatusTypeHealthy {
				countHealtyStatus++
			}
		}

		newWorkloadIdentityFederationStatus := getWorkloadIdentityFederationConnectionStatus(ctx, cred.ClientEmail)
		setWorkloadIdentityFederationStatus(ctx, cred.Customer, newWorkloadIdentityFederationStatus, cred.CloudPlatform, cred.ClientID)

		if newStatus != common.CloudConnectStatusTypeCritical && newWorkloadIdentityFederationStatus == common.CloudConnectStatusTypeCritical {
			newStatus = common.CloudConnectStatusTypeUnhealthy
		}

		if countHealtyStatus < len(permissions.Categories) && newStatus != common.CloudConnectStatusTypeUnhealthy {
			newStatus = common.CloudConnectStatusTypePartial
		}

		if newStatus != cred.Status {
			updateStatus(ctx, cred.Customer, newStatus, cred.CloudPlatform, cred.ClientID)
		}
	})

	return nil
}

func isSinkExist(ctx context.Context, fs *firestore.Client, clientID string) bool {
	docSnap, err := fs.Collection("superQuery").Doc("jobs-sinks").Collection("jobsSinksMetadata").Doc("google-cloud-" + clientID).Get(ctx)
	if err != nil {
		return false
	}

	if docSnap.Exists() {
		return true
	}

	return false
}

func getRequiredPermissions(ctx context.Context, fs *firestore.Client) (*common.CloudConnectPermissions, error) {
	docSnap, err := fs.Collection("app").Doc("cloud-connect").Get(ctx)
	if err != nil {
		return nil, err
	}

	var permissions common.CloudConnectPermissions
	if err := docSnap.DataTo(&permissions); err != nil {
		return nil, err
	}

	return &permissions, nil
}

// getProjectScopedRequiredPermissions returns cloud connect permissions without core/resourcemanager permissions and sandbox permissions
func getProjectScopedRequiredPermissions(ctx context.Context, fs *firestore.Client) (*common.CloudConnectPermissions, error) {
	docSnap, err := fs.Collection("app").Doc("cloud-connect").Get(ctx)
	if err != nil {
		return nil, err
	}

	var permissions common.CloudConnectPermissions
	if err := docSnap.DataTo(&permissions); err != nil {
		return nil, err
	}

	var core common.CloudConnectCategory

	var coreIndex int

	for i, category := range permissions.Categories {
		if category.ID == "core" {
			core = category
			coreIndex = i

			break
		}
	}

	partialCorePermissions := make([]string, 0)

	for _, permission := range core.Permissions {
		if core.OrgLevelOnlyPermissions != nil && !slice.Contains(core.OrgLevelOnlyPermissions, permission) {
			partialCorePermissions = append(partialCorePermissions, permission)
		}
	}

	core.Permissions = partialCorePermissions
	permissions.Categories[coreIndex] = core

	partialPermissions := common.CloudConnectPermissions{
		Categories: make([]common.CloudConnectCategory, 0),
	}

	for _, category := range permissions.Categories {
		if category.ID != "sandboxes" {
			partialPermissions.Categories = append(partialPermissions.Categories, category)
		}
	}

	return &partialPermissions, nil
}

func getGoogleCloudPermissions(ctx context.Context, fs *firestore.Client, categoriesID []string, scope common.GCPScope) ([]string, error) {
	var permissions *common.CloudConnectPermissions

	var err error
	if scope == common.GCPScopeProject {
		permissions, err = getProjectScopedRequiredPermissions(ctx, fs)
	} else {
		permissions, err = getRequiredPermissions(ctx, fs)
	}

	if err != nil {
		return nil, err
	}

	var arr []string

	for _, permission := range permissions.Categories {
		if slice.Contains(categoriesID, permission.ID) {
			arr = append(arr, permission.Permissions...)
		}
	}

	return arr, nil
}

func updateStatus(ctx context.Context, customerRef *firestore.DocumentRef, status common.CloudConnectStatusType, platform, clientID string) error {
	docID := common.CloudConnectDocID(platform, clientID)
	_, err := customerRef.Collection("cloudConnect").Doc(docID).Set(ctx, map[string]interface{}{
		"status": status,
	}, firestore.MergeAll)

	return err
}

func isUserAllowed(ctx *gin.Context) bool {
	fs := common.GetFirestoreClient(ctx)

	if !ctx.GetBool("doitEmployee") {
		var userID = ctx.GetString("userId")

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
	}

	return true
}

// GetCustomerGCPClient: get customer service accounts clients
func (s *CloudConnectService) GetCustomerGCPClient(ctx context.Context, customerID string) ([]common.GCPClient, error) {
	fs := s.Firestore(ctx)

	var clients []common.GCPClient

	gcpDocSnaps, err := fs.Collection("customers").Doc(customerID).Collection("cloudConnect").Where("cloudPlatform", "==", common.Assets.GoogleCloud).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, docSnap := range gcpDocSnaps {
		var cred common.GoogleCloudCredential
		if err := docSnap.DataTo(&cred); err != nil {
			continue
		}

		clients = append(clients, common.GCPClient{Doc: cred})
	}

	return clients, nil
}

func GetCustomerClients(ctx *gin.Context, customerID string, scope string) ([]*http.Client, []common.GoogleCloudCredential, []*google.Credentials, [][]byte, error) {
	var creds []common.GoogleCloudCredential

	fs := common.GetFirestoreClient(ctx)

	gcpDocSnaps, err := fs.Collection("customers").Doc(customerID).Collection("cloudConnect").
		Where("cloudPlatform", "==", common.Assets.GoogleCloud).Documents(ctx).GetAll()
	if err != nil {
		return nil, creds, nil, nil, err
	}

	clients := []*http.Client{}
	credFromJSONs := []*google.Credentials{}
	jsonKeys := [][]byte{}

	for _, docSnap := range gcpDocSnaps {
		var cred common.GoogleCloudCredential
		if err := docSnap.DataTo(&cred); err != nil {
			continue
		}

		creds = append(creds, cred)

		kmsD, err := common.DecryptSymmetric(cred.Key)
		if err != nil {
			log.Println(err)
		}

		credFromJSON, err := google.CredentialsFromJSON(ctx, kmsD, scope)
		if err != nil {
			continue
		}

		jsonKeys = append(jsonKeys, kmsD)
		credFromJSONs = append(credFromJSONs, credFromJSON)
		conf, err := google.JWTConfigFromJSON(kmsD, compute.CloudPlatformScope)
		c := conf.Client(context.Background())

		if err != nil {
			log.Println(err)
		}

		clients = append(clients, c)
	}

	return clients, creds, credFromJSONs, jsonKeys, nil
}

func (s *CloudConnectService) GetClientOptions(ctx context.Context, customerID string) ([]option.ClientOption, error) {
	customerCredentials, err := s.cloudconnectDal.GetCredentials(ctx, customerID)
	if err != nil {
		return nil, err
	}

	var clientOptionsArr []option.ClientOption

	for _, credential := range customerCredentials {
		customerCredentials := common.NewGcpCustomerAuthService(credential)

		clientOptions, err := customerCredentials.GetClientOption()
		if err != nil {
			return nil, err
		}

		clientOptionsArr = append(clientOptionsArr, clientOptions)
	}

	return clientOptionsArr, nil
}

func getWorkloadIdentityFederationConnectionStatus(ctx context.Context, clientEmail string) common.CloudConnectStatusType {
	if ok, _ := common.NewWorkloadIdentityFederationStrategy(clientEmail).IsConnectionEstablished(ctx); ok.IsConnectionEstablished {
		return common.CloudConnectStatusTypeHealthy
	}

	return common.CloudConnectStatusTypeCritical
}

func (s *CloudConnectService) CheckWorkloadIdentityFederationConnection(ctx *gin.Context, payload WorkloadIdentityFederationConnectionCheckRequest) (*common.WorkloadIdentityFederationConnectionStatus, error) {
	fs := s.Firestore(ctx)

	customerID := ctx.Param("customerID")
	customerDocRef := fs.Collection("customers").Doc(customerID)

	cloudConnectDocSnap, err := customerDocRef.Collection("cloudConnect").Doc(payload.CloudConnectID).Get(ctx)
	if err != nil {
		return nil, err
	}

	if !cloudConnectDocSnap.Exists() {
		return nil, errors.New("CloudConnect doc not found")
	}

	var cloudConnect common.GoogleCloudCredential
	if err := cloudConnectDocSnap.DataTo(&cloudConnect); err != nil {
		return nil, err
	}

	connectionStatus, err := common.NewWorkloadIdentityFederationStrategy(cloudConnect.ClientEmail).IsConnectionEstablished(ctx)
	if err != nil {
		return nil, err
	}

	workloadIdentityFederationStatus := common.CloudConnectStatusTypeCritical
	if connectionStatus.IsConnectionEstablished {
		workloadIdentityFederationStatus = common.CloudConnectStatusTypeHealthy
	}

	err = setWorkloadIdentityFederationStatus(ctx, customerDocRef, workloadIdentityFederationStatus, cloudConnect.CloudPlatform, cloudConnect.ClientID)
	if err != nil {
		return nil, err
	}

	return connectionStatus, nil
}

func setWorkloadIdentityFederationStatus(ctx context.Context, customerRef *firestore.DocumentRef, status common.CloudConnectStatusType, platform, clientID string) error {
	docID := common.CloudConnectDocID(platform, clientID)
	_, err := customerRef.Collection("cloudConnect").Doc(docID).Set(ctx, map[string]interface{}{
		"workloadIdentityFederationStatus": status,
	}, firestore.MergeAll)

	return err
}
