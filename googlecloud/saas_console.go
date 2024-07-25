package googlecloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/gin-gonic/gin"
	cloudbilling "google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

const (
	cloudBillingScope = "https://www.googleapis.com/auth/cloud-billing"
)

func StandaloneBillingAccountsListHandler(ctx *gin.Context) {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer fs.Close()

	docSnaps, err := fs.Collection("assets").Where("type", "==", common.Assets.GoogleCloudStandalone).Documents(ctx).GetAll()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var billingAccounts []AssetUpdateRequest

	page := 1

	for i, docSnap := range docSnaps {
		var asset Asset
		if err := docSnap.DataTo(&asset); err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		billingAccounts = append(billingAccounts, AssetUpdateRequest{
			CustomerID: asset.Customer.ID,
			BillingAccount: BillingAccount{
				ID:          asset.Properties.BillingAccountID,
				DisplayName: asset.Properties.DisplayName,
				Name:        fmt.Sprintf("billingAccounts/%s", asset.Properties.BillingAccountID),
			},
		})

		if (i+1)%pageSize == 0 {
			if err := scheduleStandaloneAccountsUpdate(ctx, billingAccounts, page); err != nil {
				ctx.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			page++
			billingAccounts = []AssetUpdateRequest{}
		}
	}

	if err := scheduleStandaloneAccountsUpdate(ctx, billingAccounts, page); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ctx.Status(http.StatusOK)
}

func scheduleStandaloneAccountsUpdate(ctx *gin.Context, billingAccounts []AssetUpdateRequest, page int) error {
	taskBody, err := json.Marshal(AssetsUpdateRequest{
		Type:   common.Assets.GoogleCloudStandalone,
		Assets: billingAccounts,
	})
	if err != nil {
		return err
	}

	scheduleTime := time.Now().Add(time.Second * time.Duration(10*page))

	config := common.CloudTaskConfig{
		Method:       cloudtaskspb.HttpMethod_POST,
		Path:         "/tasks/assets/google-cloud",
		Queue:        common.TaskQueueAssetsGCP,
		Body:         taskBody,
		ScheduleTime: common.TimeToTimestamp(scheduleTime),
	}

	if _, err := common.CreateCloudTask(ctx, &config); err != nil {
		return err
	}

	return nil
}

func getCloudBillingServiceForStandaloneAsset(ctx context.Context, fs *firestore.Client, asset *AssetUpdateRequest) (*cloudbilling.APIService, error) {
	docID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloud, asset.CustomerID)

	docSnap, err := fs.Collection("integrations/billing-standalone/standaloneOnboarding").Doc(docID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var onboarding pkg.GCPStandaloneOnboarding
	if err := docSnap.DataTo(&onboarding); err != nil {
		return nil, err
	}

	cb, err := cloudBillingServiceWithServiceAccount(ctx, onboarding.ServiceAccountEmail)
	if err != nil {
		return nil, err
	}

	return cb, nil
}

func cloudBillingServiceWithServiceAccount(ctx context.Context, serviceAccountEmail string) (*cloudbilling.APIService, error) {
	ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
		TargetPrincipal: serviceAccountEmail,
		Scopes:          []string{cloudBillingScope},
	})
	if err != nil {
		return nil, err
	}

	billingService, err := cloudbilling.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, err
	}

	return billingService, nil
}
