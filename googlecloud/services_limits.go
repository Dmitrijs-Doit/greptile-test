package googlecloud

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/quotas"
	notificationcenterDomain "github.com/doitintl/notificationcenter/domain"
	notificationcenterClient "github.com/doitintl/notificationcenter/pkg"
	notificationcenter "github.com/doitintl/notificationcenter/service"
)

type CustomerQuota struct {
	Services map[string][]ServiceQuota `firestore:"services"`
}

type ServiceQuota struct {
	ProjectID   string  `firestore:"projectId"`
	ProjectName string  `firestore:"projectName"`
	ServiceName string  `firestore:"service"`
	Region      string  `firestore:"region"`
	Limit       float64 `firestore:"limit"`
	Usage       float64 `firestore:"usage"`
	Percent     int     `firestore:"percent"`
	IsEmailSent bool    `firestore:"isEmailSent"`
	StatusColor string  `firestore:"status"`
}

const (
	googleDocumentationURL = "https://cloud.google.com/compute/quotas"
	googleAddQuotaURL      = "https://console.cloud.google.com/iam-admin/quotas"
	platform               = "Google Cloud"
	quotaBase              = 50
	quotaLimitWarning      = 80
)

func (s *GoogleCloudService) GetCustomerServicesLimits(ctx context.Context) error {
	fs := s.Firestore(ctx)
	l := s.loggerProvider(ctx)

	docSnaps, err := fs.CollectionGroup("cloudConnect").
		Where("cloudPlatform", "==", common.Assets.GoogleCloud).
		Where("categoriesStatus.core", "==", common.CloudConnectStatusTypeHealthy).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	updateCustomersQuotas(ctx, fs, l, docSnaps)

	return nil
}

func updateCustomersQuotas(ctx context.Context, fs *firestore.Client, l logger.ILogger, docSnaps []*firestore.DocumentSnapshot) {
	common.RunConcurrentJobsOnCollection(ctx, docSnaps, 5, func(ctx context.Context, docSnap *firestore.DocumentSnapshot) {
		var cred common.GoogleCloudCredential
		if err := docSnap.DataTo(&cred); err != nil {
			return
		}

		if cred.Scope == common.GCPScopeProject {
			l.Debugf("skipping non organization scoped cloudconnect: %s", cred.Customer.ID)
			return
		}

		if err := getCustomerQuotas(ctx, fs, cred); err != nil {
			l.Debugf("quota error %s", err.Error())
			return
		}
	})
}

func (s *GoogleCloudService) UpdateCustomerLimit(ctx context.Context, customerID string) error {
	fs := s.Firestore(ctx)
	l := s.loggerProvider(ctx)

	docSnaps, err := fs.Collection("customers").Doc(customerID).Collection("cloudConnect").Where("cloudPlatform", "==", common.Assets.GoogleCloud).Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	updateCustomersQuotas(ctx, fs, l, docSnaps)

	return nil
}

func getCustomerQuotas(ctx context.Context, fs *firestore.Client, cred common.GoogleCloudCredential) error {
	var sq []ServiceQuota

	customerCredentials := common.NewGcpCustomerAuthService(&cred)

	clientOptions, err := customerCredentials.GetClientOption()
	if err != nil {
		return err
	}

	computeService, err := compute.NewService(ctx, clientOptions)
	if err != nil {
		return err
	}

	cloudResourceManagerService, err := cloudresourcemanager.NewService(ctx, clientOptions)
	if err != nil {
		return err
	}

	var allProjects []*cloudresourcemanager.Project

	allCustomerProjects, err := cloudResourceManagerService.Projects.List().Do()
	if err != nil {
		return err
	}

	allProjects = append(allProjects, allCustomerProjects.Projects...)

	for allCustomerProjects.NextPageToken != "" {
		req := cloudResourceManagerService.Projects.List().PageToken(allCustomerProjects.NextPageToken)

		allCustomerProjects, err = req.Do()
		if err != nil {
			break
		}

		allProjects = append(allProjects, allCustomerProjects.Projects...)
	}

	var allLimitsForEmail []quotas.EmailLimit

	var customerFS CustomerQuota

	currentQuotas, err := fs.Collection("integrations").Doc("google-cloud").Collection("service-limits").Doc(cred.Customer.ID).Get(ctx)
	if err == nil {
		if err := currentQuotas.DataTo(&customerFS); err != nil {
			log.Println(err)
		}
	}

	organizationID := cred.Organizations[0].Name[14:]

	runConcurrentJobsOnProjects(ctx, allProjects, func(ctx context.Context, project *cloudresourcemanager.Project) {
		resp2, err := computeService.Projects.Get(project.ProjectId).Context(ctx).Do()
		if err != nil {
			return
		} else {
			for _, quota := range resp2.Quotas {
				percent := quota.Usage * 100 / quota.Limit
				serviceQuotaObj := ServiceQuota{
					ProjectID:   project.ProjectId,
					ProjectName: project.Name,
					Region:      project.Name,
					ServiceName: quota.Metric,
					Limit:       quota.Limit,
					Usage:       quota.Usage,
					Percent:     int(percent),
					IsEmailSent: false,
					StatusColor: "green",
				}

				if percent >= quotaBase {
					limitObj, sqObj := getQuotaObject(&serviceQuotaObj, customerFS, organizationID)

					if limitObj.Service != "" {
						allLimitsForEmail = append(allLimitsForEmail, limitObj)
					}

					if sqObj.ProjectID != "" {
						sq = append(sq, serviceQuotaObj)
					}
				}
			}
		}

		req := computeService.Regions.List(project.ProjectId)
		if err := req.Pages(ctx, func(page *compute.RegionList) error {
			for _, region := range page.Items {
				for _, quota := range region.Quotas {
					if quota.Usage > 0 {
						percent := quota.Usage * 100 / quota.Limit
						serviceQuotaObj := ServiceQuota{
							ProjectID:   project.ProjectId,
							ProjectName: project.Name,
							Region:      region.Name,
							ServiceName: quota.Metric,
							Limit:       quota.Limit,
							Usage:       quota.Usage,
							Percent:     int(percent),
							IsEmailSent: false,
							StatusColor: "green",
						}
						if percent >= quotaBase {
							limitObjCompute, sqObjCompute := getQuotaObject(&serviceQuotaObj, customerFS, organizationID)

							if limitObjCompute.Service != "" {
								allLimitsForEmail = append(allLimitsForEmail, limitObjCompute)
							}
							if sqObjCompute.ProjectID != "" {
								sq = append(sq, serviceQuotaObj)
							}

						}

					}
				}
			}
			return nil
		}); err != nil {
			//return err
		}
	})
	sort.Slice(sq[:], func(i, j int) bool {
		return sq[i].Percent > sq[j].Percent
	})

	if len(allLimitsForEmail) > 0 {
		customerData, _ := fs.Collection("customers").Doc(cred.Customer.ID).Get(ctx)

		notificationService := notificationcenter.NewRecipientsService(fs)

		notificationClient, err := notificationcenterClient.NewClient(ctx, common.ProjectID)
		if err != nil {
			log.Printf("Failed to create new notification client: %v\n", err)
			return err
		}

		recipients, err := notificationService.GetNotificationRecipientsForCustomer(ctx, cred.Customer, notificationcenterDomain.NotificationCloudQuotaUtilization)
		if err != nil {
			log.Printf("Failed to get notification recipients for customer: %v\n", err)
			return err
		}

		emailTo, emailCc, slackChannels := quotas.GetQuotaNotificationTargets(ctx, recipients, cred.Customer.ID, common.AccountManagerCompanyGcp, fs)

		primaryDomain, ok := customerData.Data()["primaryDomain"].(string)
		if !ok {
			log.Printf("primaryDomain of customer %s is not a string\n", cred.Customer.ID)

			primaryDomain = ""
		}

		notificationToSend := quotas.CreateQuotaNotification(quotas.QuotaNotificationData{
			EmailTo:        emailTo,
			EmailCc:        emailCc,
			PrimaryDomain:  primaryDomain,
			Limits:         allLimitsForEmail,
			Platform:       platform,
			Link:           googleAddQuotaURL,
			Documentation:  googleDocumentationURL,
			SlackChannells: slackChannels,
		})

		task, err := notificationClient.CreateSendTask(ctx, notificationToSend)
		if err != nil {
			log.Printf("Failed to send AWS cloud quota utilization notification for customer: %s, with task name:%s and error: %v\n", cred.Customer.ID, task.GetName(), err)
			return err
		} else {
			log.Printf("Sent AWS cloud quota utilization notification for customer: %s, with requestID: %s\n", cred.Customer.ID, task.GetName())
		}
	}

	if len(sq) > 50 {
		sq = sq[:50]
	}

	if _, err := fs.Collection("integrations").Doc("google-cloud").Collection("service-limits").Doc(cred.Customer.ID).Set(ctx, map[string]interface{}{
		"services": map[string]interface{}{
			organizationID: sq,
		},
		"customer": cred.Customer,
	}, firestore.MergeAll); err != nil {
		return err
	}

	return nil
}

func runConcurrentJobsOnProjects(ctx context.Context, collection []*cloudresourcemanager.Project, job func(ctx context.Context, doc *cloudresourcemanager.Project)) {
	maxNbConcurrentGoroutines := 10
	concurrentGoroutines := make(chan struct{}, maxNbConcurrentGoroutines)

	var wg sync.WaitGroup

	for _, doc := range collection {
		wg.Add(1)

		go func(ctx context.Context, doc *cloudresourcemanager.Project) {
			defer wg.Done()

			concurrentGoroutines <- struct{}{}

			job(ctx, doc)

			<-concurrentGoroutines
		}(ctx, doc)
	}

	wg.Wait()
	close(concurrentGoroutines)
}

func getQuotaObject(serviceQuotaObj *ServiceQuota, customerFS CustomerQuota, organizationID string) (quotas.EmailLimit, *ServiceQuota) {
	percent := serviceQuotaObj.Percent

	for _, q := range customerFS.Services[organizationID] {
		if q.ServiceName == serviceQuotaObj.ServiceName && q.IsEmailSent && serviceQuotaObj.Percent >= quotaLimitWarning && serviceQuotaObj.Region == q.Region && q.ProjectName == serviceQuotaObj.ProjectName {
			serviceQuotaObj.IsEmailSent = true
			serviceQuotaObj.StatusColor = "Yellow"

			if serviceQuotaObj.Percent >= 98 {
				serviceQuotaObj.StatusColor = "red"
			}
		}
	}

	statusName := "Warning"
	limitObj := quotas.EmailLimit{}

	if percent >= quotaLimitWarning && !serviceQuotaObj.IsEmailSent {
		//warning
		serviceQuotaObj.IsEmailSent = true
		serviceQuotaObj.StatusColor = "Yellow"

		if percent > 98 {
			//limit reached.
			statusName = "Limit Reached"
			serviceQuotaObj.StatusColor = "red"
		}

		limitObj = quotas.EmailLimit{
			Service:   serviceQuotaObj.ServiceName,
			Region:    serviceQuotaObj.Region,
			Status:    statusName,
			Limit:     fmt.Sprintf("%d", serviceQuotaObj.Percent) + "%",
			AccountID: serviceQuotaObj.ProjectID,
		}
	}

	return limitObj, serviceQuotaObj
}
