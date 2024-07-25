package googleclouddirect

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type Table struct {
	Dataset string `firestore:"dataset" binding:"required"`
	Project string `firestore:"project" binding:"required"`
	Table   string `firestore:"table" binding:"required"`
}

type Properties struct {
	BillingAccountID string `firestore:"billingAccountId"`
}

type GoogleCloudBillingAsset struct {
	common.BaseAsset
	Tables          []Table          `firestore:"tables"`
	CopyJobMetadata *CopyJobMetadata `firestore:"copyJobMetadata" binding:"required"`
	Properties      *Properties      `firestore:"properties" binding:"required"`
}

func newCopyJobMetadata() *CopyJobMetadata {
	return &CopyJobMetadata{
		Progress: 0,
		Reason:   "",
		Status:   "idle",
	}
}

type CreateAssetParams struct {
	BillingAccountID string `json:"billingAccountId" binding:"required"`
	Dataset          string `json:"dataset" binding:"required"`
	Project          string `json:"project" binding:"required"`
	CustomerID       string `json:"customerId" binding:"required"`
}

func (s *AssetService) CreateGoogleCloudDirectAssetService(ctx context.Context, p *CreateAssetParams) error {
	fs := s.conn.Firestore(ctx)
	l := s.loggerProvider(ctx)

	docSnaps, err := fs.Collection("assets").
		Where("properties.billingAccountId", "==", p.BillingAccountID).
		Select().Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	if len(docSnaps) > 0 {
		return errors.New("asset for this billing account was already created")
	}

	customerRef := fs.Collection("customers").Doc(p.CustomerID)

	customer, err := common.GetCustomer(ctx, customerRef)
	if err != nil || customer == nil {
		return errors.New("customer not found")
	}

	if ok, _ := MatchBillingAccount(p.BillingAccountID); !ok {
		return errors.New("billing account id format is invalid")
	}

	if ok, _ := MatchDataset(p.Dataset); !ok {
		return errors.New("dataset id format is invalid")
	}

	if ok, _ := MatchProject(p.Project); !ok {
		return errors.New("project id format is invalid")
	}

	tables := []Table{{Table: fmt.Sprintf("gcp_billing_export_v1_%s", strings.ReplaceAll(p.BillingAccountID, "-", "_")), Dataset: p.Dataset, Project: p.Project}}
	asset := GoogleCloudBillingAsset{
		BaseAsset: common.BaseAsset{
			AssetType: common.Assets.GoogleCloudDirect,
			Customer:  customerRef,
		},
		CopyJobMetadata: newCopyJobMetadata(),
		Tables:          tables,
		Properties: &Properties{
			BillingAccountID: p.BillingAccountID,
		},
	}

	docID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloudDirect, p.BillingAccountID)
	if _, err := fs.Collection("assets").Doc(docID).Set(ctx, asset); err != nil {
		l.Error(err)
		return errors.New("unable to create the asset")
	}

	return nil
}
