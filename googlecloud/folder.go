package googlecloud

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func ListOrganizationFolders(ctx *gin.Context) {
	customerID := ctx.Param("customerID")

	var org Organization
	if err := ctx.ShouldBindQuery(&org); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	fs := common.GetFirestoreClient(ctx)

	// Get customer cloud connect key
	docSnaps, err := fs.Collection("customers").
		Doc(customerID).
		Collection("cloudConnect").
		Where("cloudPlatform", "==", common.Assets.GoogleCloud).
		Where("categoriesStatus.core", "==", common.CloudConnectStatusTypeHealthy).
		Where("organizations", "array-contains", org).
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

	cred, err := common.NewGcpCustomerAuthService(&cloudConnectCred).GetClientOption()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	crmv2, err := cloudresourcemanagerv2.NewService(ctx, cred)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	folders, err := listOrganizationFolders(ctx, crmv2, org.Name)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, folders)
}

func listOrganizationFolders(ctx context.Context, crmv2 *cloudresourcemanagerv2.Service, organizationResourceName string) ([]*cloudresourcemanagerv2.Folder, error) {
	folders := make([]*cloudresourcemanagerv2.Folder, 0)

	if err := crmv2.Folders.List().Parent(organizationResourceName).Pages(ctx, func(response *cloudresourcemanagerv2.ListFoldersResponse) error {
		folders = append(folders, response.Folders...)
		return nil
	}); err != nil {
		return nil, err
	}

	return folders, nil
}

func UpdateOrganizationIAMResources(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	var googleDocSnaps []*firestore.DocumentSnapshot

	googleDocSnaps, err := fs.CollectionGroup("cloudConnect").
		Where("cloudPlatform", "==", common.Assets.GoogleCloud).
		Where("categoriesStatus.core", "==", common.CloudConnectStatusTypeHealthy).
		Documents(ctx).GetAll()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if len(googleDocSnaps) == 0 {
		l.Errorf("no google cloud connect credentials were found")
		return
	}

	for _, docSnap := range googleDocSnaps {
		var cloudConnectCred common.GoogleCloudCredential

		if err := docSnap.DataTo(&cloudConnectCred); err != nil {
			l.Errorf("failed to get gcp cloudconnect credentials %s with error: %s", docSnap.Ref.Path, err)
			continue
		}

		cred, err := common.NewGcpCustomerAuthService(&cloudConnectCred).GetClientOption()
		if err != nil {
			l.Errorf("failed to get gcp client option for customer %s %s with error: %s", cloudConnectCred.Customer.ID, cloudConnectCred.ClientID, err)
			continue
		}

		crmv2, err := cloudresourcemanagerv2.NewService(ctx, cred)
		if err != nil {
			l.Errorf("failed to init crm service for customer %s %s with error: %s", cloudConnectCred.Customer.ID, cloudConnectCred.ClientID, err)
			continue
		}

		foldersSearchRes, err := crmv2.Folders.Search(&cloudresourcemanagerv2.SearchFoldersRequest{}).Do()
		if err != nil {
			l.Errorf("failed to get all organization folders for customer %s %s with error: %s", cloudConnectCred.Customer.ID, cloudConnectCred.ClientID, err)
			continue
		}

		if len(foldersSearchRes.Folders) > 0 {
			resourcesMap := make(map[string]string)

			for _, folder := range foldersSearchRes.Folders {
				resourcesMap[getResourceID(folder.Name)] = folder.DisplayName
			}

			for _, org := range cloudConnectCred.Organizations {
				resourcesMap[getResourceID(org.Name)] = org.DisplayName
			}

			if _, err := fs.Collection("integrations").Doc("google-cloud").Collection("googleCloudResources").Doc(cloudConnectCred.Customer.ID).Set(ctx, map[string]interface{}{
				"customer":  cloudConnectCred.Customer,
				"resources": resourcesMap,
				"timestamp": firestore.ServerTimestamp,
			}, firestore.MergeAll); err != nil {
				l.Errorf("failed to update google cloud resources for customer %s with error: %s", cloudConnectCred.Customer.ID, err)
				continue
			}
		}
	}
}

func getResourceID(path string) string {
	slashIdx := strings.Index(path, "/")
	return path[slashIdx+1:]
}
