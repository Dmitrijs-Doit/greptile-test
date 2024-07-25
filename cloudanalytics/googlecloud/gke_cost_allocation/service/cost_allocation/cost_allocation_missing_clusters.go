package service

import (
	"context"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/domain/cost_allocation"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func (c *CostAllocationService) UpdateMissingClusters(ctx context.Context) error {
	l := c.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		serviceField: serviceName,
	})

	snapDocs, err := c.dal.GetAllEnabledCostAllocation(ctx)
	if err != nil {
		l.Error(err)
		return err
	}

	interval, err := c.getInterval(ctx)
	if err != nil {
		l.Error(err)
		return err
	}

	baClusters, err := c.getClustersByBillingAccountID(ctx, interval)
	if err != nil {
		l.Error(err)
		return err
	}

	for _, snapDoc := range snapDocs {
		var ca domain.CostAllocation

		err := snapDoc.DataTo(&ca)
		if err != nil {
			l.Errorf("unable to convert document %v; %v", snapDoc.Ref.ID, err)
			continue
		}

		err = c.updateCustomerMissingClusters(ctx, &ca, baClusters)
		if err != nil {
			l.Errorf("unable to update missing clusters for customer %s; %v", ca.Customer.ID, err)
		}
	}

	return nil
}

func (c *CostAllocationService) updateCustomerMissingClusters(
	ctx context.Context,
	costAllocation *domain.CostAllocation,
	baClusters domain.BillingAccountsClusters,
) error {
	l := c.loggerProvider(ctx)
	customerID := costAllocation.Customer.ID

	assets, err := c.assetsDal.GetCustomerGCPAssets(ctx, customerID)
	if err != nil {
		return err
	}

	if !hasAssetConfigDoc(assets) {
		l.Warningf("customer %s has no asset config doc, account is possibly terminated", customerID)
		return nil
	}

	var unenabledClusters []string

	for _, ba := range costAllocation.BillingAccountIds {
		clusters, ok := baClusters[ba]
		if !ok {
			continue
		}

		// Remove GKE Cost Allocation clusters from the full list.
		for _, caCluster := range costAllocation.Labels[domain.ClustersLabel] {
			delete(clusters, caCluster)
		}

		// Whatever remains is not using cost allocation.
		for cluster := range clusters {
			unenabledClusters = append(unenabledClusters, cluster)
		}
	}

	costAllocation.UnenabledClusters = unenabledClusters

	return c.dal.UpdateCostAllocation(ctx, customerID, costAllocation)
}

func hasAssetConfigDoc(assets []*pkg.GCPAsset) bool {
	for _, asset := range assets {
		if asset.Properties != nil {
			return true
		}
	}

	return false
}

func (c *CostAllocationService) getClustersByBillingAccountID(ctx context.Context, interval *domain.Interval) (domain.BillingAccountsClusters, error) {
	l := c.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		serviceField: serviceName,
	})

	workloads := make(domain.BillingAccountsClusters)

	query := c.getQuery(common.AssetTypeResold, interval, domain.AllClustersByBillingAccountIDTmpl)
	if err := c.getClustersFromBQ(ctx, query, &workloads); err != nil {
		return nil, err
	}

	query = c.getQuery(common.AssetTypeStandalone, interval, domain.AllClustersByBillingAccountIDTmpl)
	if err := c.getClustersFromBQ(ctx, query, &workloads); err != nil {
		return nil, err
	}

	return workloads, nil
}

func (c *CostAllocationService) getClustersFromBQ(ctx context.Context, query string, workloads *domain.BillingAccountsClusters) error {
	bq := c.conn.Bigquery(ctx)
	l := c.loggerProvider(ctx)

	l.Info(query)

	queryJob := bq.Query(query)

	it, err := queryJob.Read(ctx)
	if err != nil {
		return err
	}

	type row struct {
		BillingAccountID bigquery.NullString `bigquery:"billing_account_id"`
		Clusters         []string            `bigquery:"clusters"`
	}

	for {
		value := row{}

		err := it.Next(&value)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return err
		}

		if value.BillingAccountID.Valid {
			clusters := make(domain.BillingAccountClusters)
			for _, cluster := range value.Clusters {
				clusters[cluster] = struct{}{}
			}

			(*workloads)[value.BillingAccountID.String()] = clusters
		}
	}

	return nil
}
