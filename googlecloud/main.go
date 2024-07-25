package googlecloud

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	cloudbilling "google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

const (
	DoiTCommitmentProjectNamePrefix = "doitintl-fs"
)

type job struct {
	Customer       *common.Customer
	CustomerRef    *firestore.DocumentRef
	BillingAccount BillingAccount
	Policy         *cloudbilling.Policy
	Projects       []*cloudbilling.ProjectBillingInfo
}

type Asset struct {
	common.BaseAsset
	Properties           *AssetProperties            `firestore:"properties"`
	StandaloneProperties *StandaloneProperties       `firestore:"standaloneProperties,omitempty"`
	Snapshot             *firestore.DocumentSnapshot `firestore:"-"`
}

type AssetProperties struct {
	Etag             string   `firestore:"etag"`
	BillingAccountID string   `firestore:"billingAccountId"`
	DisplayName      string   `firestore:"displayName"`
	Admins           []string `firestore:"admins"`
	Projects         []string `firestore:"projects"`
	NumProjects      int64    `firestore:"numProjects"`
}

type StandaloneProperties struct {
	BillingReady bool `firestore:"billingReady"`
}

type ProjectAsset struct {
	AssetType  string                  `firestore:"type"`
	Properties *ProjectAssetProperties `firestore:"properties"`
	Bucket     *firestore.DocumentRef  `firestore:"bucket"`
	Contract   *firestore.DocumentRef  `firestore:"contract"`
	Customer   *firestore.DocumentRef  `firestore:"customer"`
	Entity     *firestore.DocumentRef  `firestore:"entity"`
	Tags       []string                `firestore:"tags"`
}

type ProjectAssetProperties struct {
	BillingAccountID string `firestore:"billingAccountId"`
	ProjectID        string `firestore:"projectId"`
}

type BillingAccount struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Name        string `json:"name"`
}

type AssetUpdateRequest struct {
	CustomerID     string         `json:"customerId"`
	BillingAccount BillingAccount `json:"billingAccount"`
}

type AssetsUpdateRequest struct {
	Assets []AssetUpdateRequest `json:"assets"`
	Type   string               `json:"type"`
}

const (
	validationsBillingAccount         = "011C60-22D535-69D740"
	ValidationsBillingAccountResource = "billingAccounts/" + validationsBillingAccount
	masterBillingAccount              = "0033B9-BB2726-9A3CB4"
	masterBillingAccountResource      = "billingAccounts/" + masterBillingAccount
	pageSize                          = 20
)

type GoogleCloudService struct {
	loggerProvider logger.Provider
	*connection.Connection
	CloudConnect *cloudconnect.CloudConnectService
}

func NewGoogleCloudService(loggerProvider logger.Provider, conn *connection.Connection) *GoogleCloudService {
	cloudconnect := cloudconnect.NewCloudConnectService(loggerProvider, conn)

	return &GoogleCloudService{
		loggerProvider,
		conn,
		cloudconnect,
	}
}

func (a Asset) GetCacheKey() string {
	return fmt.Sprintf("%s-%s-%s", a.AssetType, a.Entity.ID, a.Properties.BillingAccountID)
}

func (a Asset) ModifyContractQuery(query *firestore.Query) firestore.Query {
	return query.Where("type", "==", a.AssetType)
}

func (a Asset) ContractPredicate(contract *common.Contract) (bool, bool) {
	if len(contract.Assets) > 0 {
		docID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloud, a.Properties.BillingAccountID)
		for _, ref := range contract.Assets {
			if ref != nil && ref.ID == docID {
				return true, false
			}
		}

		return false, false
	}

	return true, true
}

func BillingAccountsPageHandler(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	var assets AssetsUpdateRequest

	if err := ctx.ShouldBindJSON(&assets); err != nil {
		l.Errorf("should bind json failed with error: %s", err)
		ctx.AbortWithError(http.StatusBadRequest, err)

		return
	}

	if assets.Type != common.Assets.GoogleCloud && assets.Type != common.Assets.GoogleCloudStandalone {
		err := fmt.Errorf("invalid asset type %s", assets.Type)
		l.Error(err)
		ctx.AbortWithError(http.StatusBadRequest, err)

		return
	}

	if err := updateBillingAccounts(ctx, assets.Type, assets.Assets...); err != nil {
		l.Errorf("update billing accounts failed with error: %s", err)
		ctx.AbortWithError(http.StatusInternalServerError, err)

		return
	}
}

func updateBillingAccounts(ctx *gin.Context, assetType string, assets ...AssetUpdateRequest) error {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	var cb *cloudbilling.APIService

	if assetType == common.Assets.GoogleCloud {
		secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretCloudBilling)
		if err != nil {
			return err
		}

		creds := option.WithCredentialsJSON(secret)

		cb, err = cloudbilling.NewService(ctx, creds)
		if err != nil {
			return err
		}
	}

	var err error

	assetsNum := 0

	jobsChan := make(chan *job)

	for _, asset := range assets {
		if assetType == common.Assets.GoogleCloudStandalone {
			cb, err = getCloudBillingServiceForStandaloneAsset(ctx, fs, &asset)
			if err != nil {
				l.Errorf("failed to get cloud billing service for standalone asset with error: %s", err)
				continue
			}
		}

		assetsNum++

		go handleBillingAccount(ctx, l, cb, fs, jobsChan, asset)
	}

	bw := fs.BulkWriter(ctx)

	for i := 0; i < assetsNum; i++ {
		job := <-jobsChan

		if job == nil {
			continue
		}

		l.Infof("handling billing account %s job", job.BillingAccount.ID)

		docID := fmt.Sprintf("%s-%s", assetType, job.BillingAccount.ID)
		assetRef := fs.Collection("assets").Doc(docID)
		assetSettingsRef := fs.Collection("assetSettings").Doc(docID)
		customerRef := job.CustomerRef

		admins := make([]string, 0)

		for _, binding := range job.Policy.Bindings {
			if binding.Role == RoleBillingAdmin && binding.Members != nil {
				for _, member := range binding.Members {
					if strings.HasPrefix(member, "user:") || strings.HasPrefix(member, "group:") {
						admins = append(admins, member)
					}
				}
			}
		}

		allProjects := make([]string, 0)
		customerProjects := make([]string, 0)

		for _, project := range job.Projects {
			if strings.HasPrefix(project.ProjectId, DoiTCommitmentProjectNamePrefix) {
				allProjects = append(allProjects, project.ProjectId)
			} else {
				customerProjects = append(customerProjects, project.ProjectId)
			}
		}

		numCustomerProjects := int64(len(customerProjects))
		allProjects = append(allProjects, customerProjects...)

		// Firestore doc has a limited size of 1MB, we will not show the customer projects list
		// on the billing account asset if there are more than 25k projects.
		if numCustomerProjects > 25000 {
			customerProjects = []string{}
		}

		// Restricted security mode should hide all project assets and the customer project list
		// on the billing account.
		if job.Customer.SecurityMode != nil && *job.Customer.SecurityMode == common.CustomerSecurityModeRestricted {
			allProjects = []string{}
			customerProjects = []string{}
		}

		props := &AssetProperties{
			BillingAccountID: job.BillingAccount.ID,
			DisplayName:      job.BillingAccount.DisplayName,
			Admins:           admins,
			Projects:         customerProjects,
			NumProjects:      numCustomerProjects,
			Etag:             job.Policy.Etag,
		}

		asset := Asset{
			BaseAsset: common.BaseAsset{
				AssetType: assetType,
				Customer:  customerRef,
			},
			Properties: props,
		}

		paths := []firestore.FieldPath{[]string{"type"}, []string{"properties"}, []string{"customer"}}

		if customerRef.ID != fb.Orphan.ID {
			docSnap, err := assetSettingsRef.Get(ctx)
			if err != nil {
				if status.Code(err) != codes.NotFound {
					l.Errorf("failed to get asset settings %s with error: %s", assetSettingsRef.ID, err)
					continue
				}
			}

			var as common.AssetSettings

			if docSnap.Exists() {
				if err := docSnap.DataTo(&as); err != nil {
					l.Errorf("failed to populate asset settings %s with error: %s", assetSettingsRef.ID, err)
					continue
				}
			} else {
				as = common.AssetSettings{
					BaseAsset: common.BaseAsset{
						AssetType: assetType,
					},
				}
			}

			asset.Tags = as.Tags
			asset.Bucket = as.Bucket

			paths = append(paths, []string{"bucket"}, []string{"tags"})

			if entityRef, update := common.GetAssetEntity(ctx, fs, as.Customer, as.Entity, customerRef, nil); update {
				assetSettingsUpdate := map[string]interface{}{
					"type":     assetType,
					"customer": customerRef,
					"entity":   entityRef,
				}
				asset.Entity = entityRef

				paths = append(paths, []string{"entity"})

				if contractRef, update := common.GetAssetContract(ctx, fs, asset, customerRef, entityRef, nil); update {
					assetSettingsUpdate["contract"] = contractRef
					asset.Contract = contractRef

					paths = append(paths, []string{"contract"})
				}

				if _, err := bw.Set(assetSettingsRef, assetSettingsUpdate, firestore.MergeAll); err != nil {
					l.Errorf("failed to set asset settings %s (path 1) with error: %s", assetSettingsRef.ID, err)
				}
			} else if as.Customer != nil && as.Entity != nil {
				asset.Entity = as.Entity

				paths = append(paths, []string{"entity"})

				if contractRef, update := common.GetAssetContract(ctx, fs, asset, as.Customer, as.Entity, nil); update {
					assetSettingsUpdate := map[string]interface{}{
						"type":     assetType,
						"contract": contractRef,
					}

					paths = append(paths, []string{"contract"})
					asset.Contract = contractRef

					if _, err := bw.Set(assetSettingsRef, assetSettingsUpdate, firestore.MergeAll); err != nil {
						l.Errorf("failed to set asset settings %s (path 2) with error: %s", assetSettingsRef.ID, err)
					}
				}
			}
		} else {
			// Could not find customer, update settings to reference orphan customer and reset entity, contract
			if _, err := bw.Set(assetSettingsRef, map[string]interface{}{
				"customer": customerRef,
				"entity":   nil,
				"contract": nil,
				"bucket":   nil,
				"tags":     firestore.Delete,
				"type":     assetType,
			}, firestore.MergeAll); err != nil {
				l.Errorf("failed to set asset settings %s (orphan) with error: %s", assetSettingsRef.ID, err)
			}

			asset.Entity = nil
			asset.Contract = nil
			asset.Bucket = nil

			paths = append(paths, []string{"entity"}, []string{"contract"}, []string{"bucket"})
		}

		if _, err := bw.Set(assetRef, asset, firestore.Merge(paths...)); err != nil {
			l.Errorf("failed to set asset %s with error: %s", assetRef.ID, err)
		}

		if _, err := bw.Set(assetRef.Collection("assetMetadata").Doc("metadata"), map[string]interface{}{
			"lastUpdated": firestore.ServerTimestamp,
			"type":        asset.AssetType,
		}); err != nil {
			l.Errorf("failed to set asset metadata %s with error: %s", assetRef.ID, err)
		}

		handleProjectAssets(ctx, l, fs, bw, &asset, allProjects)

		bw.Flush()

		l.Infof("finished handling billing account %s job successfully", job.BillingAccount.ID)
	}

	bw.End()

	return nil
}

func handleProjectAssets(ctx context.Context, l logger.ILogger, fs *firestore.Client, bw *firestore.BulkWriter, asset *Asset, projects []string) {
	if asset.Customer == nil || len(projects) == 0 {
		return
	}

	assetType := common.Assets.GoogleCloudProject
	if asset.AssetType == common.Assets.GoogleCloudStandalone {
		assetType = common.Assets.GoogleCloudProjectStandalone
	}

	for _, projectID := range projects {
		props := &ProjectAssetProperties{
			BillingAccountID: asset.Properties.BillingAccountID,
			ProjectID:        projectID,
		}

		projectAsset := ProjectAsset{
			AssetType:  assetType,
			Customer:   asset.Customer,
			Properties: props,
		}

		docID := fmt.Sprintf("%s-%s", assetType, projectID)
		projectAssetRef := fs.Collection("assets").Doc(docID)
		projectAssetSettingsRef := fs.Collection("assetSettings").Doc(docID)

		docSnap, err := projectAssetSettingsRef.Get(ctx)
		if err != nil && status.Code(err) != codes.NotFound {
			l.Errorf("failed to get project asset settings %s with error: %s", projectAssetSettingsRef.ID, err)
			return
		}

		if docSnap.Exists() {
			var as common.AssetSettings

			if err := docSnap.DataTo(&as); err != nil {
				l.Errorf("failed to populate project asset settings %s with error: %s", projectAssetSettingsRef.ID, err)
				return
			}

			if as.Customer != nil && as.Customer.Path == asset.Customer.Path {
				projectAsset.Tags = as.Tags
				// If the project asset is not assigned to an entity then copy the assignment from the billing account asset.
				// Otherwise, copy the entity assignment from settings to the asset.
				if as.Entity == nil && asset.Entity != nil {
					projectAsset.Entity = asset.Entity
					projectAsset.Bucket = asset.Bucket
					projectAsset.Contract = asset.Contract
					if _, err := bw.Set(projectAssetSettingsRef, map[string]interface{}{
						"entity":   asset.Entity,
						"bucket":   asset.Bucket,
						"contract": asset.Contract,
					}, firestore.MergeAll); err != nil {
						l.Errorf("failed to set project asset settings %s (path 1) with error: %s", projectAssetSettingsRef.ID, err)
					}
				} else {
					projectAsset.Entity = as.Entity
					projectAsset.Bucket = as.Bucket
					projectAsset.Contract = as.Contract
				}
			} else {
				as.Customer = asset.Customer
				as.Entity = asset.Entity
				as.Contract = asset.Contract
				as.Bucket = asset.Bucket
				projectAsset.Customer = asset.Customer
				projectAsset.Entity = asset.Entity
				projectAsset.Bucket = asset.Bucket
				projectAsset.Contract = asset.Contract

				if _, err := bw.Set(projectAssetSettingsRef, as); err != nil {
					l.Errorf("failed to set project asset settings %s (path 2) with error: %s", projectAssetSettingsRef.ID, err)
				}
			}
		} else {
			projectAsset.Customer = asset.Customer
			projectAsset.Entity = asset.Entity
			projectAsset.Contract = asset.Contract

			var bucketRef *firestore.DocumentRef

			if asset.Entity != nil {
				entitySnap, err := asset.Entity.Get(ctx)
				if err != nil {
					l.Errorf("failed to get asset entity %s with error: %s", asset.Entity.ID, err)
					return
				}

				var assetEntity common.Entity

				if err := entitySnap.DataTo(&assetEntity); err != nil {
					l.Errorf("failed to populate asset entity %s with error: %s", asset.Entity.ID, err)
					return
				}

				// legacy behavior when autoAssignGCP is not defined, is equal to autoAssignGCP = true
				if assetEntity.Invoicing.AutoAssignGCP == nil || *assetEntity.Invoicing.AutoAssignGCP {
					bucketRef = asset.Bucket
				}
			} else {
				bucketRef = asset.Bucket
			}

			projectAsset.Bucket = bucketRef

			if _, err := bw.Set(projectAssetSettingsRef, map[string]interface{}{
				"type":     assetType,
				"customer": asset.Customer,
				"entity":   asset.Entity,
				"contract": asset.Contract,
				"bucket":   bucketRef,
			}, firestore.MergeAll); err != nil {
				l.Errorf("failed to set project asset settings %s (path 3) with error: %s", projectAssetSettingsRef.ID, err)
			}
		}

		if _, err := bw.Set(projectAssetRef, projectAsset); err != nil {
			l.Errorf("failed to set project asset %s with error: %s", projectAssetRef.ID, err)
		}

		if _, err := bw.Set(projectAssetRef.Collection("assetMetadata").Doc("metadata"), map[string]interface{}{
			"lastUpdated": firestore.ServerTimestamp,
			"type":        projectAsset.AssetType,
		}); err != nil {
			l.Errorf("failed to set project asset metadata %s with error: %s", projectAssetRef.ID, err)
		}
	}
}

func handleBillingAccount(ctx context.Context, l logger.ILogger, cb *cloudbilling.APIService, fs *firestore.Client, jobsChan chan<- *job, asset AssetUpdateRequest) {
	l.Infof("creating billing account %s job", asset.BillingAccount.ID)

	customerRef := fs.Collection("customers").Doc(asset.CustomerID)

	customer, err := common.GetCustomer(ctx, customerRef)
	if err != nil {
		l.Errorf("failed to get customer %s with error: %s", customerRef.ID, err)
		jobsChan <- nil

		return
	}

	policy, err := cb.BillingAccounts.GetIamPolicy(asset.BillingAccount.Name).Do()
	if err != nil {
		l.Errorf("failed to get policy %s (%s) with error: %s", asset.BillingAccount.DisplayName, customer.PrimaryDomain, err)
		jobsChan <- nil

		return
	}

	projects, err := getBillingAccountProjects(ctx, cb, asset.BillingAccount)
	if err != nil {
		l.Errorf("failed to get projects %s (%s) with error: %s", asset.BillingAccount.DisplayName, customer.PrimaryDomain, err)
		jobsChan <- nil

		return
	}

	jobsChan <- &job{
		CustomerRef:    customerRef,
		Customer:       customer,
		BillingAccount: asset.BillingAccount,
		Policy:         policy,
		Projects:       projects,
	}
}

func getBillingAccountProjects(ctx context.Context, cb *cloudbilling.APIService, billingAccount BillingAccount) ([]*cloudbilling.ProjectBillingInfo, error) {
	var projects []*cloudbilling.ProjectBillingInfo

	call := cb.BillingAccounts.Projects.List(billingAccount.Name).Fields("projectBillingInfo(projectId)")
	if err := call.Pages(ctx, func(page *cloudbilling.ListProjectBillingInfoResponse) error {
		projects = append(projects, page.ProjectBillingInfo...)
		return nil
	}); err != nil {
		return nil, err
	}

	return projects, nil
}
