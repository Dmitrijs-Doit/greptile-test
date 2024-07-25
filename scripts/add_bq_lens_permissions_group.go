package scripts

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type CloudConnectCategory struct {
	ID                      string   `firestore:"id" json:"id"`
	Name                    string   `firestore:"name" json:"name"`
	Permissions             []string `firestore:"permissions" json:"permissions"`
	OrgLevelOnlyPermissions []string `firestore:"orgLevelOnlyPermissions" json:"orgLevelOnlyPermissions"`
	Description             string   `firestore:"description" json:"description"`
}

type CloudConnectPermissions struct {
	Categories []CloudConnectCategory `firestore:"categories" json:"categories"`
}

type GCPCloudConnect struct {
	CategoriesStatus map[string]pkg.GCPCloudConnectStatusType `firestore:"categoriesStatus,omitempty"`
}

const (
	bigqueryLensBasic      = "bigquery-finops"
	bigqueryLensAdvancedID = "bigquery-finops-advanced"
	bigqueryLensEditionsID = "bigquery-finops-editions"
)

func AddBigqueryEditionsPermissionsGroup(ctx *gin.Context) []error {
	var params UpdateCloudConnectPermissionsInput

	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	if params.ProjectID == "" {
		err := errors.New("missing project id")
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, params.ProjectID)
	if err != nil {
		return []error{err}
	}

	// update customers cloud connect docs to include bigquery-lens-editions state
	err = updateCloudConnect(ctx, fs)
	if err != nil {
		return []error{err}
	}

	return nil
}

func updateCloudConnect(ctx context.Context, fs *firestore.Client) error {
	l := logger.FromContext(ctx)

	docSnaps, err := fs.CollectionGroup("cloudConnect").
		Where("cloudPlatform", "==", "google-cloud").
		Documents(ctx).
		GetAll()
	if err != nil {
		return err
	}

	bw := fs.BulkWriter(ctx)

	for _, snap := range docSnaps {
		var storedData GCPCloudConnect

		if err := snap.DataTo(&storedData); err != nil {
			l.Errorf("failed to parse cloud connect data. path: %s, error: %v", snap.Ref.Path, err)
			continue
		}

		categoriesStatus := storedData.CategoriesStatus
		advancedStatus := categoriesStatus[bigqueryLensAdvancedID]
		basicStatus := categoriesStatus[bigqueryLensBasic]

		hasAllPermissions := advancedStatus == pkg.CloudConnectStatusTypeHealthy && basicStatus == pkg.CloudConnectStatusTypeHealthy
		editionsStatus := pkg.CloudConnectStatusTypeHealthy

		if !hasAllPermissions {
			editionsStatus = pkg.CloudConnectStatusTypeNotConfigured
		}

		updates := []firestore.Update{
			{Path: "categoriesStatus." + bigqueryLensEditionsID, Value: editionsStatus},
		}

		_, err = bw.Update(snap.Ref, updates)
		if err != nil {
			l.Errorf("failed to update cloud connect. path: %s, error: %v", snap.Ref.Path, err)
			continue
		}
	}

	bw.End()

	return nil
}
