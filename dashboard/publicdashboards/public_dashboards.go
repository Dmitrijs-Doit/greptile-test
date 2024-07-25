package publicdashboards

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/pkg"
	costAllocationDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/domain/cost_allocation"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	dashboardDomain "github.com/doitintl/hello/scheduled-tasks/dashboard"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tiers "github.com/doitintl/tiers/service"
)

const (
	AWS          string = "AWS"
	SaaSAWSLens  string = "saas-aws-lens"
	BigQueryLens string = "superquery"
	Pulse        string = "pulse"
	GcpLens      string = "gcp-lens"
	SaaSGcpLens  string = "saas-gcp-lens"
	GkeLensV2    string = "gke-lens-v2"
	AzureLens    string = "azure-lens"
	EKSLens      string = "eks-lens"
)

var DashboardsToAttach = []dashboardDomain.DashboardDetails{
	{DashboardID: "YnJfHkFLw7lsLINCkISG", DashboardType: AWS},
	{DashboardID: "CBFt28GqijUr2ozwk2vx", DashboardType: SaaSAWSLens},
	{DashboardID: "uItbeGssgbOpmpOKQ9wj", DashboardType: BigQueryLens},
	{DashboardID: "kr7df9r9FQmSH67O2wkO", DashboardType: Pulse},
	{DashboardID: "ixThO1hY3IWlo8O8Py2q", DashboardType: GcpLens},
	{DashboardID: "D1JeYOEi2xSPjh9upVCI", DashboardType: SaaSGcpLens},
	{DashboardID: "U6R46sYmgx8L55WbHvpb", DashboardType: GkeLensV2},
	{DashboardID: "pnWLfwok8rB7hsK1YKGM", DashboardType: AzureLens},
	{DashboardID: "JnC6EMX44G2d0mOv6N5a", DashboardType: EKSLens},
}

var DashboardsWithEnticements = []dashboardDomain.DashboardDetails{
	{DashboardID: "uItbeGssgbOpmpOKQ9wj", DashboardType: BigQueryLens},
	{DashboardID: "ixThO1hY3IWlo8O8Py2q", DashboardType: GcpLens},
	{DashboardID: "U6R46sYmgx8L55WbHvpb", DashboardType: GkeLensV2},
	{DashboardID: "JnC6EMX44G2d0mOv6N5a", DashboardType: EKSLens},
}

func isEnticementDashboard(dashboardID string) bool {
	for _, dashboard := range DashboardsWithEnticements {
		if dashboard.DashboardID == dashboardID {
			return true
		}
	}
	return false
}

type PublicDashboardService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	tiersService   *tiers.TiersService
}

func NewPublicDashboardService(loggerProvider logger.Provider, conn *connection.Connection) *PublicDashboardService {
	tiersService := tiers.NewTiersService(conn.Firestore)

	return &PublicDashboardService{
		loggerProvider,
		conn,
		tiersService,
	}
}

func GetPublicDashboardCloudProvider(dashboardType string) string {
	switch dashboardType {
	case BigQueryLens, GcpLens, SaaSGcpLens, GkeLensV2:
		return common.Assets.GoogleCloud
	case AWS, SaaSAWSLens, EKSLens:
		return common.Assets.AmazonWebServices
	case AzureLens:
		return common.Assets.MicrosoftAzure
	default:
		// Pulse dashboard is multicloud
		return ""
	}
}

func (s *PublicDashboardService) AttachAllDashboards(ctx context.Context, customerID string) error {
	l := logger.FromContext(ctx)

	if customerID != "" {
		go func() {
			if err := s.attachDashboardToCustomers(ctx, customerID); err != nil {
				l.Errorf("failed to attach dashboard to customer %s with error: %s", customerID, err)
			}
		}()
	} else {
		if err := s.attachDashboardToCustomers(ctx, ""); err != nil {
			return err
		}
	}

	return nil
}

func (s *PublicDashboardService) attachDashboardToCustomers(ctx context.Context, customerID string) error {
	l := logger.FromContext(ctx)
	fs := s.conn.Firestore(ctx)

	dashboardsSnaps := make(map[string]*firestore.DocumentSnapshot)

	for _, dashboard := range DashboardsToAttach {
		dashboardToCreate, err := fs.Collection("dashboards").Doc("customization").
			Collection("public-dashboards").Doc(dashboard.DashboardID).
			Get(ctx)
		if err != nil {
			l.Errorf("failed to get dashboard %s with error: %s", dashboard.DashboardID, err)
			continue
		}

		dashboardsSnaps[dashboard.DashboardID] = dashboardToCreate
	}

	var customerDocSnaps []*firestore.DocumentSnapshot

	if customerID != "" {
		docSnap, err := fs.Collection("customers").Doc(customerID).Get(ctx)
		if err != nil {
			return err
		}

		customerDocSnaps = append(customerDocSnaps, docSnap)
	} else {
		docSnaps, err := fs.Collection("customers").Documents(ctx).GetAll()
		if err != nil {
			return err
		}

		customerDocSnaps = append(customerDocSnaps, docSnaps...)
	}

	if len(customerDocSnaps) == 0 {
		l.Infof("no customers found")
		return nil
	}

	var wg sync.WaitGroup

	jobs := make(chan *firestore.DocumentSnapshot, len(customerDocSnaps))

	for _, docSnap := range customerDocSnaps {
		jobs <- docSnap

		wg.Add(1)
	}

	numWorkers := 5
	if len(customerDocSnaps) < numWorkers {
		numWorkers = len(customerDocSnaps)
	}

	for w := 1; w <= numWorkers; w++ {
		go s.attachDashboardsWorker(ctx, &wg, jobs, dashboardsSnaps)
	}

	wg.Wait()
	close(jobs)

	return nil
}

func (s *PublicDashboardService) removeCustomerDashboard(ctx context.Context, customerID string, dashboardID string) error {
	fs := s.conn.Firestore(ctx)
	_, err := fs.Collection("customers").Doc(customerID).Collection("publicDashboards").Doc(dashboardID).Delete(ctx)

	return err
}

func (s *PublicDashboardService) attachDashboard(
	ctx context.Context,
	customerDocSnap *firestore.DocumentSnapshot,
	dashboard dashboardDomain.DashboardDetails,
	dashboardToCreate *firestore.DocumentSnapshot,
) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	if !isEnticementDashboard(dashboard.DashboardID) && !s.isDashboardAllowedByCustomerTier(ctx, customerDocSnap.Ref, dashboard.DashboardType) {
		if err := s.removeCustomerDashboard(ctx, customerDocSnap.Ref.ID, dashboard.DashboardID); err != nil {
			return err
		}

		return nil
	}

	if s.isCustomerValidForDashboard(ctx, customerDocSnap, dashboard.DashboardType) {
		dataToUpdate, err := updateReportWidgetsToCustomer(customerDocSnap, dashboardToCreate)
		if err != nil {
			return err
		}

		if _, err := fs.Collection("customers").Doc(customerDocSnap.Ref.ID).
			Collection("publicDashboards").Doc(dashboard.DashboardID).
			Set(ctx, dataToUpdate); err != nil {
			return err
		}

		l.Infof("attached dashboard %s of type %s to customer %s", dashboard.DashboardID, dashboard.DashboardType, customerDocSnap.Ref.ID)

		return nil
	}

	if dashboard.DashboardType == AWS || dashboard.DashboardType == SaaSAWSLens ||
		dashboard.DashboardType == GcpLens || dashboard.DashboardType == SaaSGcpLens ||
		dashboard.DashboardType == GkeLensV2 || dashboard.DashboardType == AzureLens {
		if err := s.removeCustomerDashboard(ctx, customerDocSnap.Ref.ID, dashboard.DashboardID); err != nil {
			return err
		}
	}

	return nil
}

func (s *PublicDashboardService) attachDashboardsWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	customerDocSnaps <-chan *firestore.DocumentSnapshot,
	dashboardsSnaps map[string]*firestore.DocumentSnapshot,
) {
	l := s.loggerProvider(ctx)

	for customerDocSnap := range customerDocSnaps {
		for _, dashboard := range DashboardsToAttach {
			if dashboardsSnaps[dashboard.DashboardID] != nil {
				if err := s.attachDashboard(ctx, customerDocSnap, dashboard, dashboardsSnaps[dashboard.DashboardID]); err != nil {
					l.Errorf("failed to attach dashboard %s to customer %s with error: %s", dashboard.DashboardID, customerDocSnap.Ref.ID, err)
				}
			}
		}

		wg.Done()
	}
}

func (s *PublicDashboardService) isCustomerValidForDashboard(
	ctx context.Context,
	customerDocSnap *firestore.DocumentSnapshot,
	dashboardType string,
) bool {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)
	customerRef := customerDocSnap.Ref

	// Only Pulse dashboard is attached to the CSP customer
	if customerDocSnap.Ref.ID == domainQuery.CSPCustomerID {
		return dashboardType == Pulse
	}

	var customer common.Customer

	if err := customerDocSnap.DataTo(&customer); err != nil {
		return false
	}

	if dashboardType == AWS || dashboardType == SaaSAWSLens {
		docs, err := fs.Collection("assets").
			Where("customer", "==", customerRef).
			Where("type", "in", []string{
				common.Assets.AmazonWebServices,
				common.Assets.AmazonWebServicesStandalone,
			}).
			OrderBy("properties.organization.payerAccount.id", firestore.Asc).
			Limit(1).
			Documents(ctx).GetAll()
		if err != nil {
			l.Errorf("AWS Lens failed to get assets for customer %s with error: %s", customerRef.ID, err)
			return false
		}

		if len(docs) <= 0 {
			return false
		}

		return (dashboardType == AWS && (customer.EnabledSaaSConsole == nil || !customer.EnabledSaaSConsole.AWS)) ||
			(dashboardType == SaaSAWSLens && (customer.EnabledSaaSConsole != nil && customer.EnabledSaaSConsole.AWS))
	}

	if dashboardType == BigQueryLens {
		docs, err := fs.Collection("assets").
			Where("customer", "==", customerRef).
			Where("type", "in", []string{
				common.Assets.GoogleCloud,
				common.Assets.GoogleCloudStandalone,
			}).
			Limit(1).
			Documents(ctx).GetAll()
		if err != nil {
			l.Errorf("BigQuery Lens failed to get assets for customer %s with error: %s", customerRef.ID, err)
			return false
		}

		return len(docs) > 0
	}

	if dashboardType == GcpLens || dashboardType == SaaSGcpLens {
		docs, err := fs.Collection("assets").
			Where("customer", "==", customerRef).
			Where("type", "in", []string{
				common.Assets.GoogleCloud,
				common.Assets.GoogleCloudStandalone,
			}).
			Limit(1).
			Documents(ctx).GetAll()
		if err != nil {
			l.Errorf("GCP Lens failed to get assets for customer %s with error: %s", customerRef.ID, err)
			return false
		}

		if len(docs) <= 0 {
			return false
		}

		return (dashboardType == GcpLens && (customer.EnabledSaaSConsole == nil || !customer.EnabledSaaSConsole.GCP)) ||
			(dashboardType == SaaSGcpLens && (customer.EnabledSaaSConsole != nil && customer.EnabledSaaSConsole.GCP))
	}

	if dashboardType == Pulse {
		docs, err := fs.Collection("assets").
			Where("customer", "==", customerRef).
			Where("type", "in", []string{
				common.Assets.GoogleCloud,
				common.Assets.AmazonWebServices,
				common.Assets.AmazonWebServicesStandalone,
				common.Assets.GoogleCloudStandalone,
			}).
			Limit(1).
			Documents(ctx).GetAll()
		if err != nil {
			l.Errorf("Pulse failed to get assets for customer %s with error: %s", customerRef.ID, err)
			return false
		}

		return len(docs) > 0
	}

	if dashboardType == GkeLensV2 {
		_, err := getGkeCostAllocationDoc(ctx, fs, customerDocSnap)
		if err != nil {
			return false
		}

		if dashboardType == GkeLensV2 {
			return true
		}

		return false
	}

	if dashboardType == AzureLens {
		return isAzureEnabled(ctx, fs, customerDocSnap)
	}

	if dashboardType == EKSLens {
		return isEKSCustomer(ctx, fs, customerDocSnap) || isPredefinedPresentationCustomerWithAWS(&customer)
	}

	return false
}

func isPredefinedPresentationCustomerWithAWS(customer *common.Customer) bool {
	return customer.PresentationMode != nil && customer.PresentationMode.IsPredefined && slices.Contains(customer.Assets, common.Assets.AmazonWebServices)
}

func (s *PublicDashboardService) isDashboardAllowedByCustomerTier(ctx context.Context, customerRef *firestore.DocumentRef, dashboardType string) bool {
	var tierFeatureKey pkg.TiersFeatureKey

	switch dashboardType {
	case Pulse:
		tierFeatureKey = pkg.TiersFeatureKeyPulseDashboard
	case AWS, SaaSAWSLens:
		tierFeatureKey = pkg.TiersFeatureKeyAWSLens
	case GcpLens, SaaSGcpLens:
		tierFeatureKey = pkg.TiersFeatureKeyGCPLens
	case AzureLens:
		tierFeatureKey = pkg.TiersFeatureKeyAzureLens
	case GkeLensV2:
		tierFeatureKey = pkg.TiersFeatureKeyGKELens
	case BigQueryLens:
		tierFeatureKey = pkg.TiersFeatureKeyBigqueryLens
	case EKSLens:
		tierFeatureKey = pkg.TiersFeatureKeyEKSLens
	}

	if tierFeatureKey != "" {
		if ok, err := s.tiersService.CustomerCanAccessFeature(ctx, customerRef.ID, tierFeatureKey); err != nil || !ok {
			return false
		}
	}

	return true
}

func getGkeCostAllocationDoc(ctx context.Context, fs *firestore.Client, customerDoc *firestore.DocumentSnapshot) (*costAllocationDomain.CostAllocation, error) {
	doc, err := fs.Collection("cloudAnalytics").
		Doc("gke-cost-allocations").
		Collection("cloudAnalyticsGkeCostAllocations").
		Doc(customerDoc.Ref.ID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var ca costAllocationDomain.CostAllocation

	if err := doc.DataTo(&ca); err != nil {
		return nil, err
	}

	return &ca, nil
}

func isAzureEnabled(ctx context.Context, fs *firestore.Client, customerDoc *firestore.DocumentSnapshot) bool {
	docs, err := fs.Collection("assets").
		Where("customer", "==", customerDoc.Ref).
		Where("type", "in", []string{common.Assets.MicrosoftAzure, common.Assets.MicrosoftAzureStandalone}).
		Limit(1).
		Documents(ctx).GetAll()
	if err != nil {
		return false
	}

	return len(docs) > 0
}

func isEKSCustomer(ctx context.Context, fs *firestore.Client, customerDoc *firestore.DocumentSnapshot) bool {
	docs, err := fs.Collection("integrations").Doc("k8s-metrics").Collection("eks").
		Where("customerId", "==", customerDoc.Ref.ID).
		Limit(1).
		Documents(ctx).GetAll()
	if err != nil {
		return false
	}

	return len(docs) > 0
}

func updateReportWidgetsToCustomer(customerDoc *firestore.DocumentSnapshot, dashboardToCreate *firestore.DocumentSnapshot) (*dashboardDomain.Dashboard, error) {
	var dashboard dashboardDomain.Dashboard

	if err := dashboardToCreate.DataTo(&dashboard); err != nil {
		return nil, err
	}

	dashboard.CustomerID = customerDoc.Ref.ID
	for index := range dashboard.Widgets {
		i := strings.Index(dashboard.Widgets[index].Name, "_")
		if i != -1 {
			dashboard.Widgets[index].Name = fmt.Sprintf("cloudReports::%s_%s", customerDoc.Ref.ID, dashboard.Widgets[index].Name[i+1:])
		}
	}

	return &dashboard, nil
}
