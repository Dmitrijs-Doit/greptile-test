package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/presentations/domain"
	"github.com/doitintl/hello/scheduled-tasks/presentations/log"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
)

const (
	costAnomaliesPerCloudLimit = 3
)

type CostAnomaly struct {
	Attribution *firestore.DocumentRef `firestore:"attribution"`
	ChartData   interface{}            `firestore:"chart_data"`
	Customer    *firestore.DocumentRef `firestore:"customer"`
	MetaData    AnomalyMetaData        `firestore:"metadata"`
	Status      string                 `firestore:"status"`
	Timestamp   *time.Time             `firestore:"timestamp"`
	Updates     interface{}            `firestore:"updates"`

	// skipping these fields for now, as they need to be anonymized somehow
	// Explainer    interface{}        `firestore:"explainer"`
	// ExplainerHTML    string        `firestore:"explainer_html"`
}

// AnomalyMetaData
type AnomalyMetaData struct {
	AlertID          string     `firestore:"alert_id"`
	BillingAccountID string     `firestore:"billing_account_id"`
	Platform         string     `firestore:"platform"`
	ServiceName      string     `firestore:"service_name"`
	ProjectID        string     `firestore:"project_id"`
	Severity         int        `firestore:"severity"`
	SkuName          string     `firestore:"sku_name"`
	Timestamp        *time.Time `firestore:"timestamp"`
	Type             string     `firestore:"type"`
	Unit             string     `firestore:"unit"`
	Frequency        string     `firestore:"frequency"`
	Value            float64    `firestore:"value"`
	Excess           float64    `firestore:"excess"`

	// deprecated anomaly fields
	Context         string                 `firestore:"context"`
	UsageStartTime  string                 `firestore:"usage_start_time"`
	ExploratedLevel AnomalyExploratedLevel `firestore:"explorated_level"`
}

type AnomalyExploratedLevel struct {
	RulesModel string `firestore:"rules_model"`
}

type CostAnomaliesGenerationStrategy struct {
	Platform                       string
	SourceCustomerIDs              []string
	ProjectIDAnonymizerFnGenerator func(customerID string) string
	ProjectIDAnonymizerFnName      string
	AnonymizeCostAnomaly           func(ctx context.Context, customerID string, costAnomaly *CostAnomaly) error
}

func (p *PresentationService) CopyCostAnomaliesToCustomers(ctx *gin.Context) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "cost-anomalies")

	presentationCustomers, err := p.customersDAL.GetPresentationCustomers(ctx)
	if err != nil {
		return fmt.Errorf(FetchCustomerErr, err)
	}

	for _, presentationCustomer := range presentationCustomers {
		if err = p.doCopyCostAnomaliesToCustomer(ctx, presentationCustomer); err != nil {
			return err
		}
	}

	return nil
}

func (p *PresentationService) CopyCostAnomaliesToCustomer(ctx *gin.Context, customerID string) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "cost-anomalies")

	customer, err := p.getDemoCustomerFromID(ctx, customerID)
	if err != nil {
		return err
	}

	return p.doCopyCostAnomaliesToCustomer(ctx, customer)
}

func (p *PresentationService) doCopyCostAnomaliesToCustomer(ctx *gin.Context, customer *common.Customer) error {
	generationStrategies := []CostAnomaliesGenerationStrategy{
		{
			Platform:                       common.Assets.AmazonWebServices,
			SourceCustomerIDs:              []string{moonActiveCustomerID, connatixCustomerID, quinxCustomerID},
			ProjectIDAnonymizerFnGenerator: createAwsProjectIDAnonymizer,
			ProjectIDAnonymizerFnName:      "AWSProjectIdAnonymizer",
			AnonymizeCostAnomaly:           AnonymizeAWSCostAnomaly,
		},
		{
			Platform:                       common.Assets.MicrosoftAzure,
			SourceCustomerIDs:              []string{taboolaCustomerID, connatixCustomerID, aigoAiCustomerID},
			ProjectIDAnonymizerFnGenerator: createAzureProjectIDAnonymizer,
			ProjectIDAnonymizerFnName:      "AzureProjectIdAnonymizer",
			AnonymizeCostAnomaly:           p.AnonymizeAzureCostAnomaly,
		},
		{
			Platform:                       common.Assets.GoogleCloud,
			SourceCustomerIDs:              []string{moonActiveCustomerID, takeoffCustomerID},
			ProjectIDAnonymizerFnGenerator: func(_ string) string { return createLabelGenerator() + projectIdGenerator },
			ProjectIDAnonymizerFnName:      "ProjectIdGenerator",
			AnonymizeCostAnomaly:           AnonymizeGCPCostAnomaly,
		},
	}

	for _, strategy := range generationStrategies {
		if slices.Contains(customer.Assets, strategy.Platform) {
			if err := p.copyCostAnomaliesByPlatform(ctx, customer, &strategy); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *PresentationService) copyCostAnomaliesByPlatform(
	ctx context.Context,
	customer *common.Customer,
	strategy *CostAnomaliesGenerationStrategy,
) error {
	costAnomalies, err := p.getCostAnomalies(ctx, strategy)
	if err != nil {
		return err
	}

	mappedProjectIDs, err := p.getAnonymizedProjectIDs(ctx, customer.ID, costAnomalies, strategy)
	if err != nil {
		return err
	}

	if err := p.anonymizeCostAnomalyDetails(ctx, customer, costAnomalies, mappedProjectIDs, strategy); err != nil {
		return err
	}

	if err := p.persistCostAnomalies(ctx, costAnomalies); err != nil {
		return err
	}

	return nil
}

func (p *PresentationService) getCostAnomalies(ctx context.Context, strategy *CostAnomaliesGenerationStrategy) ([]*CostAnomaly, error) {
	fs := p.conn.Firestore(ctx)

	sourceCustomerRefs := p.generateCustomerRefs(ctx, strategy.SourceCustomerIDs)

	costAnomalySnaps, err := fs.CollectionGroup("billingAnomalies").
		Where("customer", "in", sourceCustomerRefs).
		Where("metadata.platform", "==", strategy.Platform).
		Limit(costAnomaliesPerCloudLimit).
		OrderBy("metadata.timestamp", firestore.Desc).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	costAnomalies := make([]*CostAnomaly, 0)

	for _, costAnomalySnap := range costAnomalySnaps {
		var costAnomaly CostAnomaly
		if err := costAnomalySnap.DataTo(&costAnomaly); err != nil {
			return nil, err
		}

		costAnomalies = append(costAnomalies, &costAnomaly)
	}

	return costAnomalies, nil
}

func (p *PresentationService) persistCostAnomalies(ctx context.Context, costAnomalies []*CostAnomaly) error {
	fs := p.conn.Firestore(ctx)

	for _, costAnomaly := range costAnomalies {
		if _, err := fs.
			Collection("assets").
			Doc(strings.Join([]string{costAnomaly.MetaData.Platform, costAnomaly.MetaData.ProjectID}, "-")).
			Collection("billingAnomalies").
			Doc(costAnomaly.MetaData.AlertID).
			Set(ctx, costAnomaly); err != nil {
			return err
		}
	}

	return nil
}

func (p *PresentationService) generateCustomerRefs(ctx context.Context, customerIDs []string) []*firestore.DocumentRef {
	fs := p.conn.Firestore(ctx)

	var customersRefs []*firestore.DocumentRef

	for _, customerID := range customerIDs {
		newCustomerRef := fs.Collection("customers").Doc(customerID)
		customersRefs = append(customersRefs, newCustomerRef)
	}

	return customersRefs
}

func (p *PresentationService) anonymizeCostAnomalyDetails(
	ctx context.Context,
	customer *common.Customer,
	costAnomalies []*CostAnomaly,
	mappedProjectIDs map[string]string,
	strategy *CostAnomaliesGenerationStrategy,
) error {
	for _, costAnomaly := range costAnomalies {
		costAnomaly.Customer = customer.Snapshot.Ref

		costAnomaly.MetaData.ProjectID = mappedProjectIDs[costAnomaly.MetaData.ProjectID]

		if err := strategy.AnonymizeCostAnomaly(ctx, customer.ID, costAnomaly); err != nil {
			return err
		}
	}

	return nil
}

func (p *PresentationService) getAnonymizedProjectIDs(
	ctx context.Context,
	customerID string,
	costAnomalies []*CostAnomaly,
	strategy *CostAnomaliesGenerationStrategy,
) (map[string]string, error) {

	projectIDs := make([]string, 0)

	for _, costAnomaly := range costAnomalies {
		projectIDs = append(projectIDs, costAnomaly.MetaData.ProjectID)
	}

	mappedProjectIDs, err := p.generateAnonymizedIDsUsingBQ(
		ctx,
		customerID,
		projectIDs,
		strategy.ProjectIDAnonymizerFnGenerator(customerID),
		strategy.ProjectIDAnonymizerFnName)
	if err != nil {
		return nil, err
	}

	return mappedProjectIDs, nil
}

func (p *PresentationService) generateAnonymizedIDsUsingBQ(
	ctx context.Context,
	customerID string,
	ids []string,
	funcDefinition string,
	funcName string,

) (map[string]string, error) {
	logger := p.Logger(ctx)
	bq := p.conn.Bigquery(ctx)

	projectIDsAsString, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}

	query := getQueryWithLabels(ctx, bq, fmt.Sprintf(`
		%s
		SELECT id, %s(id) as anonymizedId
		FROM UNNEST(%s) as id
	`,
		funcDefinition,
		funcName,
		projectIDsAsString,
	), customerID)

	logger.Info(query.QueryConfig.Q)

	iter, err := query.Read(ctx)
	if err != nil {
		logger.Errorln(err)
		return nil, err
	}

	mappedIDs := map[string]string{}

	for {
		var resultSet struct {
			ID           string `bigquery:"id"`
			AnonymizedID string `bigquery:"anonymizedId"`
		}

		err := iter.Next(&resultSet)
		if err == iterator.Done {
			break
		}

		if err != nil {
			logger.Errorln(err)
			return nil, err
		}

		mappedIDs[resultSet.ID] = resultSet.AnonymizedID
	}

	return mappedIDs, nil
}

func AnonymizeAWSCostAnomaly(_ context.Context, _ string, costAnomaly *CostAnomaly) error {
	costAnomaly.MetaData.BillingAccountID = awsDemoBillingAccountID

	return nil
}

func AnonymizeGCPCostAnomaly(_ context.Context, customerID string, costAnomaly *CostAnomaly) error {
	costAnomaly.MetaData.BillingAccountID = domain.HashCustomerIdIntoABillingAccountId(customerID)

	return nil
}

func (p *PresentationService) AnonymizeAzureCostAnomaly(ctx context.Context, customerID string, costAnomaly *CostAnomaly) error {
	anonymizedBillingIDs, err := p.generateAnonymizedIDsUsingBQ(
		ctx,
		customerID,
		[]string{costAnomaly.MetaData.BillingAccountID},
		createAzureUUIDv4Anonymizer(customerID),
		"AzureUuidV4Anonymizer",
	)
	if err != nil {
		return nil
	}

	costAnomaly.MetaData.BillingAccountID = anonymizedBillingIDs[costAnomaly.MetaData.BillingAccountID]

	return nil
}
