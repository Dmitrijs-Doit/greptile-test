package cloudconnect

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slice"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type RequestServiceAccount struct {
	ServiceAccount string `form:"service_account"`
	Role           string `form:"role"`
	Location       string `form:"location"`
}

type ServiceAccountKeyFile struct {
	ClientEmail string `json:"client_email"`
	ClientID    string `json:"client_id"`
}

type ServiceAccountDescriptorFile struct {
	ClientEmail string `json:"email"`
	ClientID    string `json:"uniqueId"`
	ProjectID   string `json:"projectId"`
	// there are more fields in the actual file but we only need these two. for details see: https://cloud.google.com/sdk/gcloud/reference/iam/service-accounts/describe
}

// AddGcpServiceAccount Add service account
func (s *CloudConnectService) AddGcpServiceAccount(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := s.Connection.Firestore(ctx)

	customerID := ctx.Param("customerID")
	customerRef := fs.Collection("customers").Doc(customerID)
	categoriesStatus := make(map[string]common.CloudConnectStatusType)
	countHealtyStatus := 0

	var form RequestServiceAccount
	if err := ctx.ShouldBind(&form); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if form.Location == "" {
		form.Location = "US"
	}

	mpForm, _ := ctx.MultipartForm()
	fhs := mpForm.File["service_key"]

	for _, file := range fhs {
		fileContent, _ := file.Open()
		byteContainer, _ := io.ReadAll(fileContent)

		var jsonDescriptionFile ServiceAccountDescriptorFile

		if err := json.Unmarshal(byteContainer, &jsonDescriptionFile); err != nil {
			l.Errorf("failed to unmarshal json description file: %s", err)
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

			return
		}

		// getting the client options
		initialCred := common.GoogleCloudCredential{
			Customer:                         customerRef,
			ClientID:                         jsonDescriptionFile.ClientID,
			ClientEmail:                      jsonDescriptionFile.ClientEmail,
			CloudPlatform:                    common.Assets.GoogleCloud,
			Scope:                            common.GCPScopeOrganization,
			CategoriesStatus:                 categoriesStatus,
			WorkloadIdentityFederationStatus: common.CloudConnectStatusTypeHealthy,
		}
		clientOptions, err := common.NewGcpCustomerAuthService(&initialCred).WithContext(ctx).WithScopes(compute.CloudPlatformScope).GetClientOption()

		if err != nil {
			l.Errorf("failed to get client option: %s", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})

			return
		}

		cloudresourcemanagerService, err := cloudresourcemanager.NewService(ctx, clientOptions)

		if err != nil {
			l.Errorf("failed to create new cloudresourcemanager service: %s", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})

			return
		}

		rb := &cloudresourcemanager.SearchOrganizationsRequest{}

		req := cloudresourcemanagerService.Organizations.Search(rb)

		if err := req.Pages(ctx, func(page *cloudresourcemanager.SearchOrganizationsResponse) error {
			missingPermissionsArr := []string{}
			allOrganizations := []*common.GCPConnectOrganization{}
			roleID := ""
			if len(page.Organizations) == 0 {
				l.Error("no organizations found")
				permissionsError(ctx, missingPermissionsArr, "resourcemanager.organizations.get")
				return nil
			}

			for _, organization := range page.Organizations {
				l.Infof("%+v", organization)

				resource := organization.Name
				org := &common.GCPConnectOrganization{
					Name:        resource,
					DisplayName: organization.DisplayName,
				}

				gcpServiceAccountsd, err := fs.Collection("customers").Doc(customerID).Collection("cloudConnect").Where("cloudPlatform", "==", common.Assets.GoogleCloud).Documents(ctx).GetAll()
				if err != nil {
					ctx.AbortWithError(http.StatusInternalServerError, err)
					return nil
				}

				for _, docSnap := range gcpServiceAccountsd {
					var cred common.GoogleCloudCredential

					if err := docSnap.DataTo(&cred); err != nil {
						l.Errorf("failed to populate cred data from docSnap: %s", err)
						continue
					}

					if cred.Organizations[0].DisplayName == organization.DisplayName {
						ctx.JSON(http.StatusOK, gin.H{
							"error": "organizationExist",
						})
						return nil
					}
				}

				allOrganizations = append(allOrganizations, org)
				var categoriesID []string

				permissionsCategory, err := getRequiredPermissions(ctx, fs)
				for _, category := range permissionsCategory.Categories {
					categoriesID = append(categoriesID, category.ID)
					categoryPermission, _ := getGoogleCloudPermissions(ctx, fs, []string{category.ID}, common.GCPScopeOrganization)

					rb := &cloudresourcemanager.TestIamPermissionsRequest{
						Permissions: categoryPermission,
					}
					resp, err := cloudresourcemanagerService.Organizations.TestIamPermissions(resource, rb).Context(ctx).Do()
					if err != nil {
						ctx.JSON(http.StatusInternalServerError, gin.H{
							"error": err.Error(),
						})
						return nil
					}
					categoriesStatus[category.ID] = common.CloudConnectStatusTypeHealthy
					for _, permission := range categoryPermission {
						if !slice.Contains(resp.Permissions, permission) {
							categoriesStatus[category.ID] = common.CloudConnectStatusTypeNotConfigured
							if category.ID == "core" {
								missingPermissionsArr = append(missingPermissionsArr, permission)
							}
						}
					}

					if categoriesStatus[category.ID] == common.CloudConnectStatusTypeHealthy {
						countHealtyStatus++
					}
				}

				if len(missingPermissionsArr) > 0 {
					ctx.JSON(http.StatusOK, gin.H{
						"error":              "MissingPermissions",
						"missingPermissions": missingPermissionsArr,
					})
					return nil
				}

				inputIamPolicy := &cloudresourcemanager.GetIamPolicyRequest{}
				iamPolicy, err := cloudresourcemanagerService.Organizations.GetIamPolicy(resource, inputIamPolicy).Context(ctx).Do()
				if err != nil {
					l.Errorf("failed to get iam policy: %s", err)
					return nil
				}

				for _, bind := range iamPolicy.Bindings {
					for _, member := range bind.Members {
						if "serviceAccount:"+jsonDescriptionFile.ClientEmail == member {
							roleIndex := strings.LastIndex(bind.Role, "/") + 1
							roleID = bind.Role[roleIndex:]
						}
					}

				}
			}

			docID := common.CloudConnectDocID(common.Assets.GoogleCloud, jsonDescriptionFile.ClientID)
			status := common.CloudConnectStatusTypeHealthy

			fullPermissions, _ := getRequiredPermissions(ctx, fs)
			if countHealtyStatus < len(fullPermissions.Categories) {
				status = common.CloudConnectStatusTypePartial
			}

			connectionStatus, err := common.NewWorkloadIdentityFederationStrategy(jsonDescriptionFile.ClientEmail).IsConnectionEstablished(ctx)
			if err != nil {
				return err
			}
			workloadIdentityFederationStatus := common.CloudConnectStatusTypeCritical
			if connectionStatus.IsConnectionEstablished {
				workloadIdentityFederationStatus = common.CloudConnectStatusTypeHealthy
			}

			cloudConnectCred := common.GoogleCloudCredential{
				Customer:                         customerRef,
				Organizations:                    allOrganizations,
				ClientID:                         jsonDescriptionFile.ClientID,
				ClientEmail:                      jsonDescriptionFile.ClientEmail,
				CloudPlatform:                    common.Assets.GoogleCloud,
				Status:                           status,
				RoleID:                           roleID,
				Scope:                            common.GCPScopeOrganization,
				CategoriesStatus:                 categoriesStatus,
				WorkloadIdentityFederationStatus: workloadIdentityFederationStatus,
				ProjectID:                        jsonDescriptionFile.ProjectID,
			}

			if _, err := customerRef.Collection("cloudConnect").Doc(docID).Set(ctx, cloudConnectCred); err != nil {
				return err
			}

			if categoriesStatus["bigquery-finops"] == common.CloudConnectStatusTypeHealthy {
				if err := s.CreateSinkForCustomer(ctx, customerRef.ID, form, docID); err != nil {
					l.Errorf("create sink failed for customer %s, %s", customerRef.ID, err)
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"error":    err.Error(),
						"isValid":  true,
						"clientId": jsonDescriptionFile.ClientID,
					})
					return nil
				}
			}

			ctx.JSON(http.StatusOK, gin.H{
				"isValid":  true,
				"clientId": jsonDescriptionFile.ClientID,
			})
			return nil
		}); err != nil {
			if gapiErr, ok := err.(*googleapi.Error); ok {
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"error": gapiErr.Message,
				})
			} else {
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			}

			return
		}
	}
}

func GetMissingPermissions(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	customerID := ctx.Param("customerID")

	googleDocSnaps, err := fs.Collection("customers").Doc(customerID).Collection("cloudConnect").Where("cloudPlatform", "==", common.Assets.GoogleCloud).Documents(ctx).GetAll()
	if err != nil {
		l.Errorf("failed to get google cloud credentials: %s", err)
		return
	}

	missingPermissionsArr := []string{}

	for _, docSnap := range googleDocSnaps {
		var cred common.GoogleCloudCredential

		if err := docSnap.DataTo(&cred); err != nil {
			l.Errorf("failed to populate cred data from docSnap: %s", err)

			if _, err := docSnap.Ref.Update(ctx, []firestore.Update{
				{FieldPath: []string{"status"}, Value: common.CloudConnectStatusTypeCritical},
			}); err != nil {
				l.Errorf("failed to update status: %s", err)
			}

			continue
		}

		clientOptions, err := common.NewGcpCustomerAuthService(&cred).WithContext(ctx).WithScopes(compute.CloudPlatformScope).GetClientOption()

		if err != nil {
			l.Errorf("failed to get client option: %s", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})

			return
		}

		permissions, err := getGoogleCloudPermissions(ctx, fs, []string{}, cred.Scope)
		if err != nil {
			l.Errorf("failed to get google cloud permissions: %s", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})

			return
		}

		cloudresourcemanagerService, err := cloudresourcemanager.NewService(ctx, clientOptions)

		if err != nil {
			l.Errorf("failed to create new cloudresourcemanager service: %s", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})

			return
		}

		rb := &cloudresourcemanager.TestIamPermissionsRequest{
			Permissions: permissions,
		}

		resp, err := cloudresourcemanagerService.Organizations.TestIamPermissions(cred.Organizations[0].Name, rb).Context(ctx).Do()
		if err != nil {
			l.Errorf("failed to test iam permissions: %s", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})

			return
		}

		for i := range rb.Permissions {
			if !slice.Contains(resp.Permissions, rb.Permissions[i]) {
				missingPermissionsArr = append(missingPermissionsArr, rb.Permissions[i])
			}
		}

		if len(missingPermissionsArr) > 0 {
			ctx.JSON(http.StatusOK, gin.H{
				"error":              "MissingPermissions",
				"missingPermissions": missingPermissionsArr,
			})

			return
		}
	}
}

func RemoveGcpServiceAccount(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	customerID := ctx.Param("customerID")

	var sa ServiceAccountKeyFile
	if err := ctx.ShouldBind(&sa); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	_, creds, credJSON, _, err := GetCustomerClients(ctx, customerID, bigquery.Scope)
	if err != nil {
		return
	}

	for index := 0; index < len(credJSON); index++ {
		if sa.ClientID == creds[index].ClientID {
			projectID := credJSON[index].ProjectID

			bigqueryService, err := bigquery.NewClient(ctx, projectID, option.WithCredentials(credJSON[index]))
			if err != nil {
				l.Error(err)
				ctx.AbortWithError(http.StatusInternalServerError, err)
			}

			if common.Production {
				if err := bigqueryService.Dataset("doitintl_cmp_bq").DeleteWithContents(ctx); err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{
						"error": err.Error(),
					})
				}
			}
		}
	}
	ctx.Status(http.StatusOK)

	return
}

func (s *CloudConnectService) AddPartialGcpServiceAccount(ctx *gin.Context, form RequestServiceAccount) error {
	fs := s.Firestore(ctx)
	customerID := ctx.Param("customerID")
	customerRef := fs.Collection("customers").Doc(customerID)

	categoriesStatus := make(map[string]common.CloudConnectStatusType)
	countHealtyStatus := 0

	if form.Location == "" {
		form.Location = "US"
	}

	mpForm, _ := ctx.MultipartForm()

	fhs := mpForm.File["service_key"]
	for _, file := range fhs {
		fileContent, err := file.Open()
		if err != nil {
			return err
		}

		byteContainer, err := io.ReadAll(fileContent)
		if err != nil {
			return err
		}

		var jsonDescriptionFile ServiceAccountDescriptorFile
		if err := json.Unmarshal(byteContainer, &jsonDescriptionFile); err != nil {
			return err
		}

		accessKeyCredentials, err := common.NewAccessKeyStrategy(byteContainer).GetClientOption(ctx, cloudresourcemanager.CloudPlatformScope)
		if err != nil {
			return err
		}

		cloudresourcemanagerService, err := cloudresourcemanager.NewService(ctx, accessKeyCredentials)
		if err != nil {
			return err
		}

		googleCloudprojectID := strings.Split(strings.Split(jsonDescriptionFile.ClientEmail, "@")[1], ".")[0]

		permissionsCategory, err := getProjectScopedRequiredPermissions(ctx, fs)
		if err != nil {
			return err
		}

		for _, category := range permissionsCategory.Categories {
			categoryPermission, err := getGoogleCloudPermissions(ctx, fs, []string{category.ID}, common.GCPScopeProject)
			if err != nil {
				return err
			}

			if len(categoryPermission) == 0 {
				continue
			}

			rb := &cloudresourcemanager.TestIamPermissionsRequest{
				Permissions: categoryPermission,
			}

			resp, err := cloudresourcemanagerService.Projects.TestIamPermissions(googleCloudprojectID, rb).Context(ctx).Do()
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})

				continue
			}

			categoriesStatus[category.ID] = common.CloudConnectStatusTypeHealthy
			missingPermissionsArr := []string{}

			for _, permission := range categoryPermission {
				if !slice.Contains(resp.Permissions, permission) {
					categoriesStatus[category.ID] = common.CloudConnectStatusTypeNotConfigured

					if category.ID == "core" {
						missingPermissionsArr = append(missingPermissionsArr, permission)
					}
				}
			}

			if categoriesStatus[category.ID] == common.CloudConnectStatusTypeHealthy {
				countHealtyStatus++
			}

			if len(missingPermissionsArr) > 0 {
				ctx.JSON(http.StatusOK, gin.H{
					"error":              "MissingPermissions",
					"missingPermissions": missingPermissionsArr,
				})

				continue
			}

			if category.ID == "core" {
				categoriesStatus[category.ID] = common.CloudConnectStatusTypePartial
			}
		}

		docID := common.CloudConnectDocID(common.Assets.GoogleCloud, jsonDescriptionFile.ClientID)
		status := common.CloudConnectStatusTypeHealthy

		fullPermissions, _ := getRequiredPermissions(ctx, fs)
		if countHealtyStatus < len(fullPermissions.Categories) {
			status = common.CloudConnectStatusTypePartial
		}

		kmsEncrypt, err := common.EncryptSymmetric(byteContainer)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		}

		connectionStatus, err := common.NewWorkloadIdentityFederationStrategy(jsonDescriptionFile.ClientEmail).IsConnectionEstablished(ctx)
		if err != nil {
			return err
		}

		workloadIdentityFederationStatus := common.CloudConnectStatusTypeCritical
		if connectionStatus.IsConnectionEstablished {
			workloadIdentityFederationStatus = common.CloudConnectStatusTypeHealthy
		}

		cloudConnectCred := common.GoogleCloudCredential{
			Customer:                         customerRef,
			Key:                              kmsEncrypt,
			ClientID:                         jsonDescriptionFile.ClientID,
			ClientEmail:                      jsonDescriptionFile.ClientEmail,
			CloudPlatform:                    common.Assets.GoogleCloud,
			Status:                           status,
			CategoriesStatus:                 categoriesStatus,
			Scope:                            common.GCPScopeProject,
			ProjectID:                        googleCloudprojectID,
			WorkloadIdentityFederationStatus: workloadIdentityFederationStatus,
		}

		if _, err := customerRef.Collection("cloudConnect").Doc(docID).Set(ctx, cloudConnectCred); err != nil {
			return err
		}

		ctx.JSON(http.StatusOK, gin.H{
			"isValid":  true,
			"clientId": jsonDescriptionFile.ClientID,
		})
	}

	return nil
}

func (s *CloudConnectService) GetCredentials(ctx context.Context, customerID string) ([]*common.GoogleCloudCredential, error) {
	return s.cloudconnectDal.GetCredentials(ctx, customerID)
}

func (s *CloudConnectService) GetBQLensCustomers(ctx context.Context) ([]string, error) {
	docSnaps, err := s.cloudconnectDal.GetBQLensCustomersDocs(ctx)
	if err != nil {
		return nil, err
	}

	var bqLensCustomers []string

	for _, docSnap := range docSnaps {
		var cred common.GoogleCloudCredential
		if err := docSnap.DataTo(&cred); err != nil {
			return nil, err
		}

		customerID := cred.Customer.ID
		bqLensCustomers = append(bqLensCustomers, customerID)
	}

	return bqLensCustomers, nil
}
