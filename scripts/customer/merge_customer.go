package customer

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type MergeCustomersInput struct {
	SourceCustomerID string `json:"source_customer_id"`
	TargetCustomerID string `json:"target_customer_id"`
}

type txUpdateOperations struct {
	ref     *firestore.DocumentRef
	updates []firestore.Update
}

func (s *Scripts) MergeCustomers(ctx *gin.Context) []error {
	l := s.loggerProvider(ctx)

	var requestBody MergeCustomersInput

	if err := ctx.BindJSON(&requestBody); err != nil {
		return []error{err}
	}

	input := bufio.NewScanner(os.Stdin)

	l.Infof("WARNING: You are going to merge customer %s into customer %s\n", requestBody.SourceCustomerID, requestBody.TargetCustomerID)

	l.Infof("Are you sure you want to merge these customers? Enter the SOURCE customer id to confirm\n")

	input.Scan()

	if r := input.Text(); r != requestBody.SourceCustomerID {
		l.Info("Merge aborted!")
		return nil
	}

	l.Infof("Are you really really sure? This action is irreversible! Enter the TARGET customer id to confirm\n")

	input.Scan()

	if r := input.Text(); r != requestBody.TargetCustomerID {
		l.Info("Merge aborted!")
		return nil
	}

	l.Info("Starting merge...")

	if err := s.mergeCustomers(ctx, requestBody.SourceCustomerID, requestBody.TargetCustomerID); err != nil {
		l.Errorf("Merge failed with error: %s", err)
		return []error{err}
	}

	l.Info("Merge completed successfully!")

	return nil
}

// mergeCustomers merges customer data from source customer to target customer.
// It is running in a Firestore transaction to ensure data consistency.
// This scripts supports:
// 1. Customer domains
// 2. Billing profiles (invoices will migrate on their own)
// 3. Contracts and Ramp Plans
// 4. Stripe customers mapping (TODO: update stripe customer metadata using stripe api)
// 5. Google Partner Sales Console (PSC) customer mapping
// 6. AWS Master payer accounts mapping
// 7. Merging *RESOLD* assets of type Google Cloud, Google Cloud Project, GSuite, Office365, Microsoft Azure
// 8. Merging Cloud Analytics objects (reports, budgets, alerts, attributions, attribution groups, metrics)
func (s *Scripts) mergeCustomers(ctx context.Context, sourceCustomerID, targetCustomerID string) error {
	fs := s.conn.Firestore(ctx)

	sourceCustomerRef := fs.Collection("customers").Doc(sourceCustomerID)
	targetCustomerRef := fs.Collection("customers").Doc(targetCustomerID)

	if err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		customerDomainOps, err := s.mergeCustomerDomains(ctx, tx, sourceCustomerRef, targetCustomerRef)
		if err != nil {
			return err
		}

		billingProfileOps, err := s.mergeBillingProfiles(ctx, tx, sourceCustomerRef, targetCustomerRef)
		if err != nil {
			return err
		}

		mergeStripeCustomersOps, err := s.mergeStripeCustomersMapping(ctx, tx, sourceCustomerRef, targetCustomerRef)
		if err != nil {
			return err
		}

		contractOps, err := s.mergeContracts(ctx, tx, sourceCustomerRef, targetCustomerRef)
		if err != nil {
			return err
		}

		rampPlansOps, err := s.mergeRampPlans(ctx, tx, sourceCustomerRef, targetCustomerRef)
		if err != nil {
			return err
		}

		mergeGooglePscMappingOps, err := s.mergeGooglePartnerSalesConsoleMapping(ctx, tx, sourceCustomerRef, targetCustomerRef)
		if err != nil {
			return err
		}

		googleCloudAssetsOps, err := s.mergeAssetsOfType(ctx, tx, sourceCustomerRef, targetCustomerRef, common.Assets.GoogleCloud)
		if err != nil {
			return err
		}

		googleCloudProjectAssetsOps, err := s.mergeAssetsOfType(ctx, tx, sourceCustomerRef, targetCustomerRef, common.Assets.GoogleCloudProject)
		if err != nil {
			return err
		}

		gsuiteAssetsOps, err := s.mergeAssetsOfType(ctx, tx, sourceCustomerRef, targetCustomerRef, common.Assets.GSuite)
		if err != nil {
			return err
		}

		office365AssetsOps, err := s.mergeAssetsOfType(ctx, tx, sourceCustomerRef, targetCustomerRef, common.Assets.Office365)
		if err != nil {
			return err
		}

		msazureAssetsOps, err := s.mergeAssetsOfType(ctx, tx, sourceCustomerRef, targetCustomerRef, common.Assets.MicrosoftAzure)
		if err != nil {
			return err
		}

		awsAssetsOps, err := s.mergeAssetsOfType(ctx, tx, sourceCustomerRef, targetCustomerRef, common.Assets.AmazonWebServices)
		if err != nil {
			return err
		}

		mergeAwsMasterPayerAccountsOps, err := s.mergeAwsMasterPayerAccounts(ctx, tx, sourceCustomerRef, targetCustomerRef)
		if err != nil {
			return err
		}

		mergeCostAnomalies, err := s.mergeCostAnomalies(ctx, tx, sourceCustomerRef, targetCustomerRef)
		if err != nil {
			return err
		}

		mergeCloudAnalyticsOps, err := s.mergeCloudAnalytics(ctx, tx, sourceCustomerRef, targetCustomerRef)
		if err != nil {
			return err
		}

		if err := s.applyUpdateOperations(
			tx,
			customerDomainOps,
			contractOps,
			rampPlansOps,
			billingProfileOps,
			mergeStripeCustomersOps,
			mergeGooglePscMappingOps,
			awsAssetsOps,
			mergeAwsMasterPayerAccountsOps,
			googleCloudAssetsOps,
			googleCloudProjectAssetsOps,
			gsuiteAssetsOps,
			office365AssetsOps,
			msazureAssetsOps,
			mergeCostAnomalies,
			mergeCloudAnalyticsOps,
		); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// applyUpdateOperations applies update operations to the transaction
func (s *Scripts) applyUpdateOperations(tx *firestore.Transaction, ops ...[]txUpdateOperations) error {
	for _, op := range ops {
		if len(op) > 0 {
			for _, doc := range op {
				if err := tx.Update(doc.ref, doc.updates); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// mergeCustomerDomains merges customer domains from source customer to target customer
// and set the classification of the source customer to inactive.
func (s *Scripts) mergeCustomerDomains(ctx context.Context, tx *firestore.Transaction, sourceCustomerRef, targetCustomerRef *firestore.DocumentRef) ([]txUpdateOperations, error) {
	l := s.loggerProvider(ctx)

	docSnap, err := tx.Get(sourceCustomerRef)
	if err != nil {
		return nil, err
	}

	var srcCustomer common.Customer
	if err := docSnap.DataTo(&srcCustomer); err != nil {
		return nil, err
	}

	l.Infof("Merging domains: %v", srcCustomer.Domains)

	res := make([]txUpdateOperations, 0, 2)

	newPrimaryDomain := fmt.Sprintf("merged.%s", srcCustomer.PrimaryDomain)

	domains := make([]interface{}, 0, len(srcCustomer.Domains))
	for _, domain := range srcCustomer.Domains {
		domains = append(domains, domain)
	}

	res = append(res,
		txUpdateOperations{
			ref: sourceCustomerRef,
			updates: []firestore.Update{
				{Path: "primaryDomain", Value: newPrimaryDomain},
				{Path: "domains", Value: []string{newPrimaryDomain}},
				{Path: "classification", Value: common.CustomerClassificationInactive},
			},
		},
		txUpdateOperations{
			ref: targetCustomerRef,
			updates: []firestore.Update{
				{Path: "domains", Value: firestore.ArrayUnion(domains...)},
			},
		},
	)

	return res, nil
}
