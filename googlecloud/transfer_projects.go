package googlecloud

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

type CreateServiceAccountRequest struct {
	BillingAccountID string `json:"billingAccountId"`
}

type TransferProjectsRequest struct {
	BillingAccountID string   `json:"billingAccountId"`
	Projects         []string `json:"projects"`
}

type Project struct {
	ProjectID   string `json:"projectId"`
	BillingID   string `json:"billingId"`
	BillingName string `json:"billingName"`
}

type DeleteHandler struct {
	ServiceAccountEmail string `json:"serviceAccountEmail"`
	CustomerID          string `json:"customerID"`
	BillingAccountID    string `json:"billingAccountId"`
}

type ProjectsTransferd struct {
	TransferredProjects []string `json:"transferredProjects"`
	BlockedProjects     []string `json:"blockedProjects"`
}

func CreateServiceAccountForCustomer(ctx *gin.Context) {
	customerID := ctx.Param("customerID")
	email := ctx.GetString("email")

	l := logger.FromContext(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	var body CreateServiceAccountRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	billingAccountID := body.BillingAccountID
	if customerID == "" || billingAccountID == "" {
		return
	}

	fs := common.GetFirestoreClient(ctx)

	client, err := google.DefaultClient(ctx, iam.CloudPlatformScope, cloudresourcemanager.CloudPlatformScope)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	iamService, err := iam.New(client)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	saDoc, err := fs.Collection("integrations").Doc("google-cloud").Collection("serviceAccounts").Doc(customerID).Get(ctx)
	if err != nil {
		// No existing service account key
	} else {
		saKey, _ := saDoc.DataAt("saKey")
		saEmail, _ := saDoc.DataAt("serviceAccountEmail")
		kmsD, _ := common.DecryptSymmetric([]byte(saKey.([]byte)))

		if string(kmsD) != "" {
			if err := AddSAToBillingAccount(ctx, l, billingAccountID, saEmail.(string)); err != nil {
				ctx.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			getProjectsList(ctx, customerID, billingAccountID)

			return
		}
	}

	customerDoc, err := fs.Collection("customers").Doc(customerID).Get(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	primaryDomain, _ := customerDoc.DataAt("primaryDomain")
	name := "projects/" + common.ProjectID

	rand.Seed(time.Now().UnixNano())

	rb := &iam.CreateServiceAccountRequest{
		AccountId: "sa-" + strings.ToLower(customerID) + "-" + strconv.Itoa(rangeIn(1000, 9999)),
		ServiceAccount: &iam.ServiceAccount{
			Description: "cmp-transfer-projects-for-" + primaryDomain.(string),
		},
	}

	newServiceAccount, err := iamService.Projects.ServiceAccounts.Create(name, rb).Context(ctx).Do()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	serviceAccountEmail := newServiceAccount.Email
	l.Info(newServiceAccount)
	time.Sleep(3 * time.Second) // Wait for service account to be created

	respKey, err := iamService.Projects.ServiceAccounts.Keys.Create(
		newServiceAccount.Name, &iam.CreateServiceAccountKeyRequest{},
	).Context(ctx).Do()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	kmsEncrypt, err := common.EncryptSymmetric([]byte(respKey.PrivateKeyData))
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	fs.Collection("integrations").Doc("google-cloud").Collection("serviceAccounts").Doc(customerID).Set(ctx, map[string]interface{}{
		"saKey":               kmsEncrypt,
		"serviceAccountEmail": serviceAccountEmail,
	})

	if err := AddSAToBillingAccount(ctx, l, billingAccountID, serviceAccountEmail); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	CreateTask(ctx, serviceAccountEmail, customerID, billingAccountID)
	getProjectsList(ctx, customerID, billingAccountID)
}

func AddSAToBillingAccount(ctx *gin.Context, l logger.ILogger, billingAccountID string, serviceAccountEmail string) error {
	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretCloudBilling)
	if err != nil {
		return err
	}

	creds := option.WithCredentialsJSON(secret)

	cb, err := cloudbilling.NewService(ctx, creds)
	if err != nil {
		return err
	}

	resource := "billingAccounts/" + billingAccountID

	policy, err := cb.BillingAccounts.GetIamPolicy(resource).Context(ctx).Do()
	if err != nil {
		return err
	}

	l.Info(policy)

	policy.Bindings = append(policy.Bindings, &cloudbilling.Binding{
		Role:    RoleBillingAdmin,
		Members: []string{fmt.Sprintf("serviceAccount:%s", serviceAccountEmail)},
	})

	updatedPolicy, err := cb.BillingAccounts.SetIamPolicy(resource, &cloudbilling.SetIamPolicyRequest{
		Policy: &cloudbilling.Policy{
			Bindings: policy.Bindings, Etag: policy.Etag,
		},
	}).Do()
	if err != nil {
		return err
	}

	l.Info(updatedPolicy)

	return nil
}

func getProjectsList(ctx *gin.Context, customerID string, billingAccountID string) {
	fs := common.GetFirestoreClient(ctx)

	if customerID == "" {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	doc, err := fs.Collection("integrations").Doc("google-cloud").Collection("serviceAccounts").Doc(customerID).Get(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	saKey, _ := doc.DataAt("saKey")
	saEmail, _ := doc.DataAt("serviceAccountEmail")
	kmsD, _ := common.DecryptSymmetric([]byte(saKey.([]byte)))
	sDec, _ := base64.StdEncoding.DecodeString(string(kmsD))

	conf, err := google.JWTConfigFromJSON(sDec, cloudbilling.CloudPlatformScope)
	if err != nil {
		_ = err
	}

	c := conf.Client(ctx)

	client, err := cloudbilling.New(c)
	if err != nil {
		_ = err
	}

	projectsList := []Project{}

	call := client.BillingAccounts.List()
	if err := call.Pages(ctx, func(page *cloudbilling.ListBillingAccountsResponse) error {
		for _, v := range page.BillingAccounts {
			name := v.Name
			displayName := v.DisplayName
			call := client.BillingAccounts.Projects.List(name)

			if err := call.Pages(ctx, func(page *cloudbilling.ListProjectBillingInfoResponse) error {
				for _, info := range page.ProjectBillingInfo {
					if "billingAccounts/"+billingAccountID != info.BillingAccountName {
						item := Project{
							ProjectID:   info.ProjectId,
							BillingID:   info.BillingAccountName,
							BillingName: displayName,
						}
						projectsList = append(projectsList, item)
					}
				}
				return nil
			}); err != nil {
				_ = err
			}

		}
		return nil
	}); err != nil {
		_ = err
	}

	ctx.JSON(http.StatusOK, gin.H{
		"sa":          saEmail.(string),
		"projectList": projectsList,
	})
}

func TransferProjects(ctx *gin.Context) {
	customerID := ctx.Param("customerID")
	email := ctx.GetString("email")

	l := logger.FromContext(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	var body TransferProjectsRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	l.Info(ctx.GetStringMap("claims"))
	l.Info(body)
	projects := body.Projects
	billingAccountID := body.BillingAccountID

	if customerID == "" || billingAccountID == "" || len(projects) == 0 {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	fs := common.GetFirestoreClient(ctx)

	doc, err := fs.Collection("integrations").Doc("google-cloud").Collection("serviceAccounts").Doc(customerID).Get(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	saKey, err := doc.DataAt("saKey")
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	kmsD, err := common.DecryptSymmetric(saKey.([]byte))
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	sDec, err := base64.StdEncoding.DecodeString(string(kmsD))
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	conf, err := google.JWTConfigFromJSON(sDec, cloudbilling.CloudPlatformScope)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	cloudbillingService, err := cloudbilling.New(conf.Client(ctx))
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	dstBillingAccountName := "billingAccounts/" + billingAccountID

	var projectsResults ProjectsTransferd

	for _, projectID := range projects {
		name := "projects/" + projectID

		resp, err := cloudbillingService.Projects.UpdateBillingInfo(name, &cloudbilling.ProjectBillingInfo{
			BillingAccountName: dstBillingAccountName,
		}).Context(ctx).Do()
		if err != nil {
			l.Error(err)

			projectsResults.BlockedProjects = append(projectsResults.BlockedProjects, projectID)
		} else {
			projectsResults.TransferredProjects = append(projectsResults.TransferredProjects, projectID)
		}

		l.Info(resp)
	}

	// Update billing account and projects assets
	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretCloudBilling)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	creds := option.WithCredentialsJSON(secret)

	cb, err := cloudbilling.NewService(ctx, creds)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	resp, err := cb.BillingAccounts.Get(dstBillingAccountName).Context(ctx).Do()
	if err != nil {
		l.Error(err)
		return
	}

	billingAccount := BillingAccount{
		ID:          billingAccountID,
		DisplayName: resp.DisplayName,
		Name:        resp.Name,
	}
	if err := updateBillingAccounts(ctx, common.Assets.GoogleCloud, AssetUpdateRequest{
		CustomerID:     customerID,
		BillingAccount: billingAccount,
	}); err != nil {
		l.Error(err)
	}

	ctx.JSON(http.StatusOK, projectsResults)
}

func CreateTask(ctx *gin.Context, serviceAccountEmail string, customerID string, billingAccountID string) {
	d := DeleteHandler{
		ServiceAccountEmail: serviceAccountEmail,
		CustomerID:          customerID,
		BillingAccountID:    billingAccountID,
	}

	taskBody, err := json.Marshal(d)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	scheduleTime := time.Now().Add(time.Hour * time.Duration(24*29))

	config := common.CloudTaskConfig{
		Method:       cloudtaskspb.HttpMethod_POST,
		Path:         "/tasks/assets/delete-customer-sa",
		Queue:        common.TaskQueueBillingTransfer,
		Body:         taskBody,
		ScheduleTime: common.TimeToTimestamp(scheduleTime),
	}

	_, err = common.CreateCloudTask(ctx, &config)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}

func DeleteServiceAccount(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	var deleteHandlerObj DeleteHandler
	if err := ctx.ShouldBindJSON(&deleteHandlerObj); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	l.Info(deleteHandlerObj)

	fs := common.GetFirestoreClient(ctx)

	// Delete the SA
	client, err := google.DefaultClient(ctx, iam.CloudPlatformScope)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	iamService, err := iam.New(client)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	name := "projects/" + common.ProjectID + "/serviceAccounts/" + deleteHandlerObj.ServiceAccountEmail
	if _, err := iamService.Projects.ServiceAccounts.Delete(name).Context(ctx).Do(); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Delete document from firestore
	docSnap, err := fs.Collection("integrations").Doc("google-cloud").
		Collection("serviceAccounts").Doc(deleteHandlerObj.CustomerID).
		Delete(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	l.Info(docSnap.UpdateTime)
}

func CheckServiceAccountPermissions(ctx *gin.Context) {
	customerID := ctx.Param("customerID")

	var body CreateServiceAccountRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	fs := common.GetFirestoreClient(ctx)

	doc, err := fs.Collection("integrations").Doc("google-cloud").Collection("serviceAccounts").Doc(customerID).Get(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	saKey, _ := doc.DataAt("saKey")
	kmsD, _ := common.DecryptSymmetric([]byte(saKey.([]byte)))
	sDec, _ := base64.StdEncoding.DecodeString(string(kmsD))

	conf, err := google.JWTConfigFromJSON(sDec, cloudbilling.CloudPlatformScope)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	client, err := cloudbilling.New(conf.Client(ctx))
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	isValid := false

	call := client.BillingAccounts.List()
	if err := call.Pages(ctx, func(page *cloudbilling.ListBillingAccountsResponse) error {
		for _, v := range page.BillingAccounts {
			if v.Name != "billingAccounts/"+body.BillingAccountID && v.MasterBillingAccount != "billingAccounts/"+masterBillingAccount {
				isValid = true
			}
		}
		return nil
	}); err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"isValid": false,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"isValid": isValid,
	})
}
func rangeIn(low, hi int) int {
	return low + rand.Intn(hi-low)
}
