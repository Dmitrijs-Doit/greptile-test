package invoicing

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
)

const (
	gcpInvoiceRowDescription       = "Google Cloud"
	gcpInvoiceRowCreditDescription = "Google Cloud Credit"
	projectDetailsTemplate         = "Project '%s'"
	billingAccountDetailsTemplate  = "Billing account %s"
	creditAdjustmentTemplate       = "%s (Adjustment for Discount)"
)

type ProjectsValue struct {
	Bucket *firestore.DocumentRef
	Entity *firestore.DocumentRef
	Value  float64
}

type ProjectsValuePerBucket map[string]*ProjectsValue

func (s *InvoicingService) customerGoogleCloudHandler(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows) {
	fs := s.Firestore(ctx)

	res := &domain.ProductInvoiceRows{
		Type:  common.Assets.GoogleCloud,
		Rows:  make([]*domain.InvoiceRow, 0),
		Error: nil,
	}

	defer func() {
		respChan <- res
	}()

	customer, err := common.GetCustomer(ctx, customerRef)
	if err != nil {
		res.Error = err
		return
	}

	monthlyBillingData, err := customerMonthlyBillingGoogleCloud(ctx, fs, customerRef, task.InvoiceMonth)
	if err != nil {
		res.Error = err
		return
	}

	invoiceAdjustments, err := getCustomerInvoiceAdjustments(ctx, customerRef, common.Assets.GoogleCloud, task.InvoiceMonth)
	if err != nil {
		res.Error = err
		return
	}

	for docID, data := range monthlyBillingData {
		rows, err := billingAccountRows(ctx, fs, task, customer, docID, data)
		if err != nil {
			res.Error = err
			return
		}

		res.Rows = append(res.Rows, rows...)
	}

	for _, invoiceAdjustment := range invoiceAdjustments {
		entity, prs := entities[invoiceAdjustment.Entity.ID]
		if !prs {
			err := fmt.Errorf("invalid entity for invoiceAdjustment %s", invoiceAdjustment.Snapshot.Ref.Path)
			res.Error = err

			return
		}

		var quantity int64 = 1

		ppu := invoiceAdjustment.Amount
		if ppu < 0 {
			ppu *= -1
			quantity = -1
		}

		res.Rows = append(res.Rows, &domain.InvoiceRow{
			Description: invoiceAdjustment.Description,
			Details:     invoiceAdjustment.Details,
			Quantity:    quantity,
			PPU:         ppu,
			Currency:    invoiceAdjustment.Currency,
			Total:       invoiceAdjustment.Amount,
			SKU:         GoogleCloudSKU,
			Rank:        InvoiceAdjustmentRank,
			Type:        common.Assets.GoogleCloud,
			Final:       true,
			Entity:      invoiceAdjustment.Entity,
			Bucket:      entity.Invoicing.Default,
		})
	}
}

func customerMonthlyBillingGoogleCloud(ctx context.Context, fs *firestore.Client, customerRef *firestore.DocumentRef, invoiceMonth time.Time) (map[string]*MonthlyBillingGoogleCloud, error) {
	result := make(map[string]*MonthlyBillingGoogleCloud)
	docs, err := fs.CollectionGroup("monthlyBillingData").
		Where("customer", "==", customerRef).
		Where("type", "==", common.Assets.GoogleCloud).
		Where("invoiceMonth", "==", invoiceMonth.Format("2006-01")).
		Documents(ctx).GetAll()

	if err != nil {
		return nil, err
	}

	for _, docSnap := range docs {
		var baseMonthlyBillingGoogleCloud BaseMonthlyBillingGoogleCloud
		if err := docSnap.DataTo(&baseMonthlyBillingGoogleCloud); err != nil {
			return nil, err
		}

		monthlyBillingDataItemsDocs, err := docSnap.Ref.Collection("monthlyBillingDataItems").Documents(ctx).GetAll()
		if err != nil {
			return nil, err
		}

		monthlyBillingGoogleCloudDataItems, err := NewMonthlyBillingGoogleCloudDataItemsFromSnapshots(monthlyBillingDataItemsDocs)
		if err != nil {
			return nil, err
		}

		monthlyBillingGoogleCloud := MonthlyBillingGoogleCloud{
			baseMonthlyBillingGoogleCloud,
			*monthlyBillingGoogleCloudDataItems,
		}

		result[docSnap.Ref.Parent.Parent.ID] = &monthlyBillingGoogleCloud
	}

	return result, nil
}

func billingAccountRows(ctx context.Context, fs *firestore.Client, task *domain.CustomerTaskData, customer *common.Customer, docID string, data *MonthlyBillingGoogleCloud) ([]*domain.InvoiceRow, error) {
	var billingAccountAssetSettings common.AssetSettings

	docSnap, err := fs.Collection("assetSettings").Doc(docID).Get(ctx)
	if err != nil {
		return nil, err
	}

	if err := docSnap.DataTo(&billingAccountAssetSettings); err != nil {
		return nil, err
	}

	securityModeRestricted := customer.SecurityMode != nil && *customer.SecurityMode == common.CustomerSecurityModeRestricted
	billingAccountID := docID[len(common.Assets.GoogleCloud)+1:]

	defaultEntity := billingAccountAssetSettings.Entity
	defaultBucket := billingAccountAssetSettings.Bucket

	if defaultEntity == nil {
		return nil, fmt.Errorf("billing account %s is not assigned to an entity", billingAccountID)
	}

	// TODO(dror): Temporary fix for Redis billing profile. Remove the hardcoded ID when we have
	// a proper way to set this option on billing profiles (entities) for customers that always
	// want an invoice per billing account, without the hassle of making sure projects are assigned
	// to the relevant buckets every month.
	// This will set the entity and bucket for each project to be the same as the billing account,
	// ignoring whatever is set on the project asset settings.
	useBillingAccountSettings := defaultEntity.ID == "aq8f95fin8FhCKLG1uZN"

	var discount *string

	if data.Discount != nil && *data.Discount > 0 {
		t := fmt.Sprintf(" with %.2f%% discount", *data.Discount)
		discount = &t
	}

	projectSettingsRefs := make([]*firestore.DocumentRef, 0)
	rows := make([]*domain.InvoiceRow, 0)
	final := task.TimeIndex == -2 && task.Now.Day() >= 1

	// Create invoice row for billing account if exists, and remove in from projects map
	if value, ok := data.Projects.Values[gcpProjectsAggregatedBillingAccount]; ok {
		qty, value := utils.GetQuantityAndValue(1, value)
		rows = append(rows, &domain.InvoiceRow{
			Description:   gcpInvoiceRowDescription,
			Details:       fmt.Sprintf(billingAccountDetailsTemplate, billingAccountID),
			DetailsSuffix: discount,
			Tags:          billingAccountAssetSettings.Tags,
			Quantity:      qty,
			PPU:           value,
			Currency:      string(fixer.USD),
			Total:         float64(qty) * value,
			SKU:           GoogleCloudSKU,
			Rank:          2,
			Type:          common.Assets.GoogleCloud,
			Final:         final,
			Entity:        defaultEntity,
			Bucket:        defaultBucket,
		})

		delete(data.Projects.Values, gcpProjectsAggregatedBillingAccount)
	}

	// Get all projects asset settings and set all that are found in a map
	projectsSettingsMap := make(map[string]*common.AssetSettings)

	for projectID := range data.Projects.Values {
		projectDocID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloudProject, projectID)
		projectSettingsRef := fs.Collection("assetSettings").Doc(projectDocID)
		projectSettingsRefs = append(projectSettingsRefs, projectSettingsRef)
	}

	projectSettingsDocSnaps, err := fs.GetAll(ctx, projectSettingsRefs)
	if err != nil {
		return nil, err
	}

	for _, projectSettingsDocSnap := range projectSettingsDocSnaps {
		if projectSettingsDocSnap.Exists() { // GetAll does not return error if a document does not exist
			var projectSettings common.AssetSettings
			if err := projectSettingsDocSnap.DataTo(&projectSettings); err != nil {
				return nil, err
			}

			projectsSettingsMap[projectSettingsDocSnap.Ref.ID] = &projectSettings
		}
	}

	for projectID, value := range data.Projects.Values {
		projectDocID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloudProject, projectID)
		entity := defaultEntity
		bucket := defaultBucket
		projectDetails := projectID

		var tags []string

		// If project asset settings are found, use them, otherwise use the billing account default settings
		if projectSettings, ok := projectsSettingsMap[projectDocID]; ok {
			if projectSettings.Customer != nil && projectSettings.Customer.ID == billingAccountAssetSettings.Customer.ID &&
				projectSettings.Entity != nil {
				tags = projectSettings.Tags

				if !useBillingAccountSettings {
					entity = projectSettings.Entity
					bucket = projectSettings.Bucket
				}
			}
		}

		// If customer security mode is restricted, use project number instead of project ID
		if securityModeRestricted {
			if projectNumber, ok := data.ProjectNumbers.Values[projectID]; ok {
				projectDetails = projectNumber
			}
		}

		qty, value := utils.GetQuantityAndValue(1, value)
		rows = append(rows, &domain.InvoiceRow{
			Description:   gcpInvoiceRowDescription,
			Details:       fmt.Sprintf(projectDetailsTemplate, projectDetails),
			DetailsSuffix: discount,
			Tags:          tags,
			Quantity:      qty,
			PPU:           value,
			Currency:      string(fixer.USD),
			Total:         float64(qty) * value,
			SKU:           GoogleCloudSKU,
			Rank:          1,
			Type:          common.Assets.GoogleCloud,
			Final:         final,
			Entity:        entity,
			Bucket:        bucket,
		})
	}

	for creditID, projectsCredit := range data.ProjectsCredits.Values {
		docSnap, err := fs.Collection("customers").Doc(task.CustomerID).Collection("customerCredits").Doc(creditID).Get(ctx)
		if err != nil {
			return nil, err
		}

		name, err := docSnap.DataAt("name")
		if err != nil {
			return nil, err
		}

		creditPerBucketMap := calculateValuesPerBucket(
			projectsCredit,
			projectsSettingsMap,
			billingAccountAssetSettings,
			useBillingAccountSettings,
			defaultEntity,
			defaultBucket,
		)

		for _, credit := range creditPerBucketMap {
			rows = append(rows, &domain.InvoiceRow{
				Description: gcpInvoiceRowCreditDescription,
				Details:     name.(string),
				Quantity:    -1,
				PPU:         credit.Value,
				Currency:    string(fixer.USD),
				Total:       -credit.Value,
				SKU:         GoogleCloudSKU,
				Rank:        CreditRank,
				Type:        common.Assets.GoogleCloud,
				Final:       final,
				Entity:      credit.Entity,
				Bucket:      credit.Bucket,
			})
		}

		if projectsCreditAdjValue, ok := data.ProjectsCreditsDiscountAdjustment.Values[creditID]; ok {
			creditAdjPerBucketMap := calculateValuesPerBucket(
				projectsCreditAdjValue,
				projectsSettingsMap,
				billingAccountAssetSettings,
				useBillingAccountSettings,
				defaultEntity,
				defaultBucket,
			)

			for _, creditAdj := range creditAdjPerBucketMap {
				if creditAdj.Value > 0 {
					rows = append(rows, &domain.InvoiceRow{
						Description: gcpInvoiceRowCreditDescription,
						Details:     fmt.Sprintf(creditAdjustmentTemplate, name.(string)),
						Quantity:    1,
						PPU:         creditAdj.Value,
						Currency:    string(fixer.USD),
						Total:       creditAdj.Value,
						SKU:         GoogleCloudSKU,
						Rank:        CreditRank,
						Type:        common.Assets.GoogleCloud,
						Final:       final,
						Entity:      creditAdj.Entity,
						Bucket:      creditAdj.Bucket,
					})
				}
			}
		}
	}

	// calculate total savings
	flexsaveSavingsPerBucketMap := calculateValuesPerBucket(
		data.ProjectsFlexsaveSavings.Values,
		projectsSettingsMap,
		billingAccountAssetSettings,
		useBillingAccountSettings,
		defaultEntity,
		defaultBucket,
	)

	for _, flexsaveSaving := range flexsaveSavingsPerBucketMap {
		rows = append(rows, &domain.InvoiceRow{
			Description: utils.InvoiceFlexsaveSavingsDescription,
			Details:     utils.InvoiceFlexsaveSavingsDetails,
			Tags:        []string{},
			Quantity:    -1,
			PPU:         -1 * flexsaveSaving.Value, // PPU is +ve, quantity is -ve
			Currency:    string(fixer.USD),
			Total:       flexsaveSaving.Value, // Total already -ve
			SKU:         GoogleCloudSKU,
			Rank:        3,
			Type:        common.Assets.GoogleCloud,
			Final:       final,
			Entity:      flexsaveSaving.Entity,
			Bucket:      flexsaveSaving.Bucket,
		})
	}

	return rows, nil
}

func calculateValuesPerBucket(
	projectsValues map[string]float64,
	projectsSettingsMap map[string]*common.AssetSettings,
	billingAccountAssetSettings common.AssetSettings,
	useBillingAccountSettings bool,
	defaultEntity *firestore.DocumentRef,
	defaultBucket *firestore.DocumentRef,
) ProjectsValuePerBucket {
	projectValuePerBucketMap := make(ProjectsValuePerBucket)

	for projectID, value := range projectsValues {
		projectDocID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloudProject, projectID)
		entity := defaultEntity
		bucket := defaultBucket

		// If project asset settings are found, use them, otherwise use the billing account default settings
		if !useBillingAccountSettings {
			if projectSettings, ok := projectsSettingsMap[projectDocID]; ok {
				if projectSettings.Customer != nil &&
					projectSettings.Customer.ID == billingAccountAssetSettings.Customer.ID &&
					projectSettings.Entity != nil {
					entity = projectSettings.Entity
					bucket = projectSettings.Bucket
				}
			}
		}

		key := entity.ID
		if bucket != nil {
			key = key + bucket.ID
		}

		if val, ok := projectValuePerBucketMap[key]; ok {
			val.Value += value
		} else {
			projectsValue := ProjectsValue{
				Bucket: bucket,
				Entity: entity,
				Value:  value,
			}
			projectValuePerBucketMap[key] = &projectsValue
		}
	}

	return projectValuePerBucketMap
}
