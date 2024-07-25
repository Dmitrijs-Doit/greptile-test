package googlecloud

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	cloudbilling "google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/option"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type SandboxPolicyInterval string
type SandboxPolicyAction string
type SandboxStatus string

const (
	SandboxPolicyIntervalMonthly      SandboxPolicyInterval = "monthly"
	SandboxPolicyIntervalOneTime      SandboxPolicyInterval = "one_time"
	SandboxPolicyActionSendAlert      SandboxPolicyAction   = "send_alert"
	SandboxPolicyActionDisableBilling SandboxPolicyAction   = "disable_billing"
	SandboxStatusActive               SandboxStatus         = "active"
	SandboxStatusDisabled             SandboxStatus         = "disabled"
	SandboxStatusAlerted              SandboxStatus         = "alerted"
)

type SandboxPolicy struct {
	Amount         int64                 `firestore:"amount"`
	Action         SandboxPolicyAction   `firestore:"action"`
	Interval       SandboxPolicyInterval `firestore:"interval"`
	Type           string                `firestore:"type"`
	NamePrefix     string                `firestore:"namePrefix"`
	BillingAccount string                `firestore:"billingAccount"`
	Organization   Organization          `firestore:"organization"`
	Folder         *Folder               `firestore:"folder"`
	Limit          *int                  `firestore:"limit"`
	Email          string                `firestore:"email"`
	Timestamp      time.Time             `firestore:"timestamp,serverTimestamp"`
}

type Organization struct {
	DisplayName string `json:"displayName" form:"displayName" firestore:"displayName"`
	Name        string `json:"name" form:"name" firestore:"name"`
}

type Folder struct {
	DisplayName string `json:"displayName" form:"displayName" firestore:"displayName"`
	Name        string `json:"name" form:"name" firestore:"name"`
}

type SandboxAccount struct {
	Customer                   *firestore.DocumentRef `firestore:"customer"`
	Policy                     *firestore.DocumentRef `firestore:"policy"`
	ProjectID                  string                 `firestore:"projectId"`
	ProjectNumber              int64                  `firestore:"projectNumber"`
	ProjectResourceName        string                 `firestore:"projectResourceName"`
	BillingAccountResourceName string                 `firestore:"billingAccountResourceName"`
	BudgetResourceName         string                 `firestore:"budgetResourceName"`
	Status                     SandboxStatus          `firestore:"status"`
	Utilization                map[string]float64     `firestore:"utilization"`
	Email                      string                 `firestore:"email"`
	UpdatedAt                  time.Time              `firestore:"updatedAt,serverTimestamp"`
	CreatedAt                  time.Time              `firestore:"createdAt,serverTimestamp"`
}

type GoogleAPIResource interface {
	MarshalJSON() ([]byte, error)
}

func marshalJSON(res GoogleAPIResource) string {
	v, _ := res.MarshalJSON()
	return string(v)
}

func CreateSandbox(ctx *gin.Context) {
	customerID := ctx.Param("customerID")
	userID := ctx.GetString("userId")
	email := ctx.GetString("email")

	l := logger.FromContext(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelUserID:     userID,
		logger.LabelCustomerID: customerID,
	})

	creds := option.WithCredentialsJSON(appEngineCredsJSON)

	cb, err := cloudbilling.NewService(ctx, creds)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	fs := common.GetFirestoreClient(ctx)

	if ctx.GetBool("doitEmployee") {
		ctx.AbortWithStatus(http.StatusForbidden)
		return
	}

	userRef := fs.Collection("users").Doc(userID)

	user, err := common.GetUser(ctx, userRef)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if !user.HasSandboxAdminPermission(ctx) && !user.HasSandboxUserPermission(ctx) {
		ctx.AbortWithStatus(http.StatusForbidden)
		return
	}

	customerRef := fs.Collection("customers").Doc(customerID)
	sandboxesCollection := fs.Collection("integrations").Doc(common.Assets.GoogleCloud).Collection("sandboxAccounts")

	policyQuery, err := customerRef.Collection("sandboxPolicies").
		Where("active", "==", true).
		Where("type", "==", common.Assets.GoogleCloud).
		OrderBy("timestamp", firestore.Desc).
		Limit(1).Documents(ctx).GetAll()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if len(policyQuery) <= 0 {
		ctx.AbortWithError(http.StatusNotFound, err)
		return
	}

	policyDocSnap := policyQuery[0]
	policyRef := policyDocSnap.Ref

	var policy SandboxPolicy
	if err := policyDocSnap.DataTo(&policy); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if policy.Limit != nil && *policy.Limit > 0 {
		docSnaps, err := sandboxesCollection.
			Where("customer", "==", customerRef).
			Where("email", "==", email).
			Where("status", common.In, []string{
				string(SandboxStatusActive),
				string(SandboxStatusDisabled),
				string(SandboxStatusAlerted),
			}).
			Documents(ctx).GetAll()

		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		currentSandboxNum := len(docSnaps)

		t := time.Now()
		currentMonth := fmt.Sprintf("%d-%02d", t.Year(), int(t.Month()))

		for _, docSnap := range docSnaps {
			var sandboxAccount SandboxAccount
			if err := docSnap.DataTo(&sandboxAccount); err != nil {
				continue
			}
			// exclude sandboxes disabled in the previous months due to budget overrun assuming a new budget cycle started
			if sandboxAccount.Status == SandboxStatusDisabled && sandboxAccount.Utilization[currentMonth] == 0 {
				currentSandboxNum--
			}
		}

		if currentSandboxNum >= *policy.Limit {
			err := errors.New("maximum sandboxes for user reached")
			ctx.AbortWithError(http.StatusForbidden, err)

			return
		}
	}

	// Get billing account info
	billingAccount, err := cb.BillingAccounts.Get("billingAccounts/" + policy.BillingAccount).Do()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	l.Info(marshalJSON(billingAccount))

	// Get customer cloud connect key
	docSnaps, err := customerRef.Collection("cloudConnect").
		Where("cloudPlatform", "==", common.Assets.GoogleCloud).
		Where("categoriesStatus.sandboxes", "==", common.CloudConnectStatusTypeHealthy).
		Where("organizations", "array-contains", policy.Organization).
		Limit(1).
		Documents(ctx).GetAll()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if len(docSnaps) == 0 {
		err := errors.New("cloud connect credentials not found")
		ctx.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	var cloudConnectCred common.GoogleCloudCredential
	if err := docSnaps[0].DataTo(&cloudConnectCred); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	customerCredentials := common.NewGcpCustomerAuthService(&cloudConnectCred)

	clientOptions, err := customerCredentials.GetClientOption()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Create the sandbox project
	project, err := CreateProject(ctx, l, clientOptions, &policy)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	l.Info(marshalJSON(project))

	// Add our service account as projecet owner, so  we will be able to link it to billing account
	if err := AddProjectOwner(ctx, clientOptions, project.ProjectId, "serviceAccount:me-doit-intl-com@appspot.gserviceaccount.com"); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Link sandbox project to the billing account
	projectBillingInfo, err := UpdateProjectBillingInfo(ctx, billingAccount, project)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	l.Info(marshalJSON(projectBillingInfo))

	// Create a google cloud budget
	budget, err := CreateBudget(ctx, projectBillingInfo, &policy)
	if err != nil {
		DisableProjectBilling(ctx, project)
		ctx.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	l.Info(marshalJSON(budget))

	sandbox := SandboxAccount{
		Customer:                   customerRef,
		Policy:                     policyRef,
		ProjectID:                  project.ProjectId,
		ProjectNumber:              project.ProjectNumber,
		ProjectResourceName:        "projects/" + project.ProjectId,
		BillingAccountResourceName: billingAccount.Name,
		BudgetResourceName:         budget.Name,
		Status:                     SandboxStatusActive,
		Utilization:                make(map[string]float64),
		Email:                      email,
	}
	if _, err := sandboxesCollection.Doc(budget.Name[45:]).Set(ctx, sandbox); err != nil {
		DisableProjectBilling(ctx, project)
		ctx.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	if err := AddProjectOwner(ctx, nil, project.ProjectId, "user:"+email); err != nil {
		DisableProjectBilling(ctx, project)
		ctx.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	// Update billing account asset
	go updateBillingAccounts(ctx, common.Assets.GoogleCloud, AssetUpdateRequest{
		CustomerID: customerID,
		BillingAccount: BillingAccount{
			ID:          policy.BillingAccount,
			DisplayName: billingAccount.DisplayName,
			Name:        billingAccount.Name,
		},
	})

	ctx.String(http.StatusOK, project.ProjectId)
}
