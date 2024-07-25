package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud"
)

const numWorkers = 5

type workerJob struct {
	Type string
	ID   string
	Orgs []*common.Organization
}

func (h *AnalyticsMetadata) UpdateOrgsMetadata(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	orgID := ctx.Param("orgID")

	// Skip E2E customer for now, causing spam of Metadata update jobs
	if customerID == common.E2ETestCustomerID {
		return nil
	}

	fs := h.conn.Firestore(ctx)
	customerRef := fs.Collection("customers").Doc(customerID)

	orgs, err := common.GetCustomerOrgs(ctx, fs, customerRef, orgID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	// Update GCP metadata:
	var wg sync.WaitGroup

	jobsArr := make([]*workerJob, 0)

	gcpDocSnaps, err := fs.Collection("assets").
		Where("customer", "==", customerRef).
		Where("type", "in", []string{common.Assets.GoogleCloud, common.Assets.GoogleCloudStandalone}).
		Documents(ctx).GetAll()
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	for _, docSnap := range gcpDocSnaps {
		var asset googlecloud.Asset
		if err := docSnap.DataTo(&asset); err != nil {
			errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
			continue
		}

		job := &workerJob{
			Type: asset.AssetType,
			ID:   asset.Properties.BillingAccountID,
			Orgs: orgs,
		}
		jobsArr = append(jobsArr, job)
	}

	jobs := make(chan *workerJob, len(gcpDocSnaps))
	for _, job := range jobsArr {
		jobs <- job

		wg.Add(1)
	}

	for w := 1; w <= numWorkers; w++ {
		go h.updateGCPMetadataWorker(ctx, jobs, &wg)
	}

	// Update AWS metadata:
	wg.Add(1)

	go func() {
		if err := h.service.UpdateAWSCustomerMetadata(ctx, customerID, orgs); err != nil {
			errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		}

		wg.Done()
	}()

	wg.Wait()
	close(jobs)

	return nil
}

func (h *AnalyticsMetadata) updateGCPMetadataWorker(ctx context.Context, jobs <-chan *workerJob, wg *sync.WaitGroup) {
	for job := range jobs {
		assetID := fmt.Sprintf("%s-%s", job.Type, job.ID)
		if err := h.service.UpdateGCPBillingAccountMetadata(ctx, assetID, job.ID, job.Orgs); err != nil {
			h.loggerProvider(ctx).Errorf("Failed to update GCP metadata for asset %s job %s job orgs: %v", assetID, job.ID, job.Orgs, err)
		}

		wg.Done()
	}
}
