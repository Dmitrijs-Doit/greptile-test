// Script which imports an array of entitlements to the firestore 'entitlements'
package scripts

import (
	"context"
	"errors"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/tiers/dal"
)

type EntitlementImporter struct {
	fs *firestore.Client
	l  logger.ILogger

	tiers dal.TierEntitlementsIface
}

func NewEntitlementsImporter(ctx context.Context, projectID string, l logger.ILogger) (*EntitlementImporter, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &EntitlementImporter{
		fs:    fs,
		l:     l,
		tiers: dal.NewTierEntitlementsDALWithClient(fs),
	}, nil
}

func (e *EntitlementImporter) Import(ctx context.Context, entitlements []*pkg.TierEntitlement, commit bool) error {
	e.l.Infof("importing %d entitlements", len(entitlements))

	for _, entitlement := range entitlements {
		e.l.Infof("creating entitlement: %s", entitlement.Key)

		if !commit {
			continue
		}

		err := e.tiers.AddEntitlement(ctx, entitlement)
		if err != nil {
			e.l.Errorf("importing entitlement %s failed with error: %s", entitlement.Key, err.Error())
			return err
		}
	}

	return nil
}

type ImportEntitlementsRequest struct {
	Entitlements []*pkg.TierEntitlement `json:"entitlements"`
	Commit       bool                   `json:"commit"`
}

func ImportEntitlements(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	var req ImportEntitlementsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return []error{err}
	}

	importer, err := NewEntitlementsImporter(ctx, common.ProjectID, l)
	if err != nil {
		return []error{err}
	}

	l.Info("starting import of entitlements")

	if !req.Commit {
		l.Info("dry run, no entitlements will be created")
	}

	err = importer.Import(ctx, req.Entitlements, req.Commit)
	if err != nil {
		return []error{err}
	}

	l.Info("entitlements imported successfully")

	return nil
}

type CopyEntitlementsRequest struct {
	SrcProject      string   `json:"src_project"`
	DstProject      string   `json:"dst_project"`
	EntitlementKeys []string `json:"entitlementKeys"`
	Commit          bool     `json:"commit"`
}

func CopyEntitlements(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	var req CopyEntitlementsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return []error{err}
	}

	if req.SrcProject == "" || req.DstProject == "" {
		err := errors.New("invalid input parameters")
		errorreporting.AbortWithErrorReport(ctx, http.StatusBadRequest, err)

		return []error{err}
	}

	l.Infof("starting copying entitlements from %s to %s", req.SrcProject, req.DstProject)

	if !req.Commit {
		l.Info("dry run, no entitlements will be copied")
	}

	srcFs, err := firestore.NewClient(ctx, req.SrcProject)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}
	defer srcFs.Close()

	dstFs, err := firestore.NewClient(ctx, req.DstProject)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}
	defer dstFs.Close()

	docSnaps, err := srcFs.Collection("entitlements").Where("key", "in", req.EntitlementKeys).Documents(ctx).GetAll()
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	for _, docSnap := range docSnaps {
		if req.Commit {
			if _, err := dstFs.Doc("entitlements/"+docSnap.Ref.ID).Set(ctx, docSnap.Data()); err != nil {
				errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
				return []error{err}
			}
		}

		key, _ := docSnap.DataAt("key")
		l.Infof("copied entitlement: %s, key %s", docSnap.Ref.ID, key)
	}

	l.Info("entitlements imported successfully")

	return nil
}

type CopyTiersRequest struct {
	SrcProject string   `json:"src_project"`
	DstProject string   `json:"dst_project"`
	TierIDs    []string `json:"tierIds"`
	Commit     bool     `json:"commit"`
}

func CopyTiers(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	var req CopyTiersRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return []error{err}
	}

	if req.SrcProject == "" || req.DstProject == "" {
		err := errors.New("invalid input parameters")
		errorreporting.AbortWithErrorReport(ctx, http.StatusBadRequest, err)

		return []error{err}
	}

	l.Infof("starting copying tiers from %s to %s", req.SrcProject, req.DstProject)

	if !req.Commit {
		l.Info("dry run, no tiers will be copied")
	}

	srcFs, err := firestore.NewClient(ctx, req.SrcProject)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}
	defer srcFs.Close()

	dstFs, err := firestore.NewClient(ctx, req.DstProject)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}
	defer dstFs.Close()

	for _, tierID := range req.TierIDs {
		if req.Commit {
			if copyTier(ctx, srcFs, dstFs, tierID); err != nil {
				errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
				return []error{err}
			}
		}

		l.Infof("copied tier: %s", tierID)
	}

	l.Info("tiers imported successfully")

	return nil
}

func copyTier(ctx context.Context, srcFs, dstFs *firestore.Client, tierID string) error {
	docSnap, err := srcFs.Collection("tiers").Doc(tierID).Get(ctx)
	if err != nil {
		return err
	}

	var tier pkg.Tier
	if err := docSnap.DataTo(&tier); err != nil {
		return err
	}

	dstEntitlements := make([]*firestore.DocumentRef, 0, len(tier.Entitlements))

	for _, e := range tier.Entitlements {
		dstEntitlements = append(dstEntitlements, dstFs.Doc("entitlements/"+e.ID))
	}

	tier.Entitlements = dstEntitlements

	if _, err := dstFs.Doc("tiers/"+tierID).Set(ctx, tier); err != nil {
		return err
	}

	return nil
}

type AssignEntitlementsToTierRequest struct {
	EntitlementKeys []string `json:"entitlementKeys"`
	TierID          string   `json:"tierId"`
}

func AssignEntitlementsToTier(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	var req AssignEntitlementsToTierRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return []error{err}
	}

	l.Infof("starting assigning entitlements to tier %s", req.TierID)

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}
	defer fs.Close()

	importer, err := NewEntitlementsImporter(ctx, common.ProjectID, l)
	if err != nil {
		return []error{err}
	}

	tierRef := fs.Doc("tiers/" + req.TierID)

	for _, key := range req.EntitlementKeys {
		entitlementRef := fs.Doc("entitlements/" + key)
		err := importer.tiers.AddEntitlementToTier(ctx, tierRef, entitlementRef)

		if err != nil {
			errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
			return []error{err}
		}

		l.Infof("assigned entitlement %s to tier %s", key, req.TierID)
	}

	l.Info("entitlements assigned successfully")

	return nil
}
