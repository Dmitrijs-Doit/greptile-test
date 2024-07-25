package common

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/doitintl/cloudtasks/iface"
)

type TaskQueue string

const (
	// Cloud Analytics
	TaskQueueCloudAnalyticsTablesAzure        TaskQueue = "cloud-analytics-tables-azure"
	TaskQueueCloudAnalyticsTablesAWS          TaskQueue = "cloud-analytics-tables-aws"
	TaskQueueCloudAnalyticsTablesGCP          TaskQueue = "cloud-analytics-tables-gcp"
	TaskQueueCloudAnalyticsAlerts             TaskQueue = "cloud-analytics-alerts"
	TaskQueueCloudAnalyticsBudgets            TaskQueue = "cloud-analytics-budgets"
	TaskQueueCloudAnalyticsOnDemandTasks      TaskQueue = "cloud-analytics-ondemand-tasks"
	TaskQueueCloudAnalyticsMetadataAzure      TaskQueue = "cloud-analytics-metadata-azure"
	TaskQueueCloudAnalyticsMetadataAWS        TaskQueue = "cloud-analytics-metadata-aws"
	TaskQueueCloudAnalyticsMetadataGCP        TaskQueue = "cloud-analytics-metadata-gcp"
	TaskQueueCloudAnalyticsMetadataDataHub    TaskQueue = "cloud-analytics-metadata-datahub"
	TaskQueueCloudAnalyticsDigest             TaskQueue = "cloud-analytics-digest"
	TaskQueueCloudAnalyticsCSP                TaskQueue = "cloud-analytics-csp"
	TaskQueueCloudAnalyticsCustomerDashboards TaskQueue = "cloud-analytics-customer-dashboards"
	TaskQueueCloudAnalyticsWidgets            TaskQueue = "cloud-analytics-widgets"
	TaskQueueCloudAnalyticsWidgetsPrioritized TaskQueue = "cloud-analytics-widgets-prioritized"
	TaskQueueGKECostAllocationTasks           TaskQueue = "gke-cost-allocation-tasks"
	TaskQueuePresentationGCP                  TaskQueue = "presentation-gcp"
	TaskQueuePresentationAWS                  TaskQueue = "presentation-aws"
	TaskQueuePresentationAzure                TaskQueue = "presentation-azure"
	TaskQueueCourierNotificationsExportTasks  TaskQueue = "courier-notifications-export-tasks"

	// Flexsave
	TaskQueueFlexsaveAWSAutopilot         TaskQueue = "flexsave-autopilot-aws"
	TaskQueueFlexsaveAWSPotential         TaskQueue = "flexsave-potential-aws"
	TaskQueueFlexsaveAWSCache             TaskQueue = "flexsave-cache"
	TaskQueueFlexsaveAWSSavingsPlansCache TaskQueue = "flexsave-savings-plans-cache"
	TaskQueueFlexsaveInvoiceAdjustment    TaskQueue = "flexsave-invoice-adjustment"

	// Flexsave Standalone
	TaskQueueFlexSaveStandaloneSpendSummary            TaskQueue = "flexsave-standalone-spend-summary"
	TaskQueueFlexsaveStandaloneGCPNewServiceAccounts   TaskQueue = "flexsave-standalone-create-new-sa"
	TaskQueueFlexSaveStandaloneInternalTasks           TaskQueue = "flexsave-standalone-internal-tasks"
	TaskQueueFlexSaveStandaloneOnboarding              TaskQueue = "flexsave-standalone-onboarding"
	TaskQueueFlexSaveStandaloneOffboarding             TaskQueue = "flexsave-standalone-offboarding"
	TaskQueueFlexSaveStandaloneAutomationTasks         TaskQueue = "flexsave-standalone-automation-tasks"
	TaskQueueFlexSaveStandaloneExternalToBucketTasks   TaskQueue = "flexsave-standalone-external-to-bucket-tasks"
	TaskQueueFlexSaveStandaloneExternalFromBucketTasks TaskQueue = "flexsave-standalone-external-from-bucket-tasks"
	TaskQueueFlexSaveStandaloneRowsValidatorTasks      TaskQueue = "flexsave-standalone-rows-validator-tasks"
	TaskQueueFlexSaveStandaloneMonitorTasks            TaskQueue = "flexsave-standalone-monitor-tasks"

	// Invoices & Billing
	TaskQueueInvoicesSync       TaskQueue = "customers-invoices-sync"
	TaskQueueInvoicing          TaskQueue = "invoicing"
	TaskQueueInvoicingAnalytics TaskQueue = "invoicing-analytics"

	// Billing Explainer
	TaskQueueBillingExplainer TaskQueue = "billing-explainer"

	// Assets
	TaskQueueAssetsAWS           TaskQueue = "assets-aws"
	TaskQueueAssetsAWSStandAlone TaskQueue = "assets-aws-standalone"
	TaskQueueAssetsAWSSaaS       TaskQueue = "assets-aws-saas"
	TaskQueueAssetsGCP           TaskQueue = "assets-gcp"
	TaskQueueAssetsMicrosoft     TaskQueue = "assets-microsoft"
	TaskQueueAssetsMicrosoftSaas TaskQueue = "assets-microsoft-standalone"

	// Misc
	TaskQueueDefault                   TaskQueue = "default"
	TaskQueueBillingTransfer           TaskQueue = "billing-transfer"
	TaskQueueGCPRecommenderRightsizing TaskQueue = "gcp-recommender-rightsizing"
	TaskQueueHubspot                   TaskQueue = "hubspot"
	TaskQueueSalesforce                TaskQueue = "salesforce-sync"
	TaskQueueSendgrid                  TaskQueue = "sendgrid-mail-tasks"
	TaskQueueMPAGoogleGroup            TaskQueue = "master-payer-accounts-google-group-creation-tasks"
	TaskQueueUpdateRampPlan            TaskQueue = "update-ramp-plan"
	TaskQueueUpdateCustomersSegment    TaskQueue = "update-customers-segment"

	// Entities
	TaskQueueEntityInvoiceAttributionsSync TaskQueue = "entity-invoice-attributions-sync-daily"
	TaskQueueStripePayments                TaskQueue = "stripe-payments"

	// SaaS Billing
	TaskQueueBillingSaaSGCPNewServiceAccounts TaskQueue = "gcp-saas-billing-create-new-service-accounts"
	TaskQueueAWSSaaSValidateConnection        TaskQueue = "aws-saas-validate-connection"

	// BigQuery Lens
	TaskQueueBQLensTablesDiscovery TaskQueue = "bq-lens-tables-discovery"
	TaskQueueBQLensOptimizer       TaskQueue = "bq-lens-optimizer-migrated"

	// Contracts
	TaskQueueContractsRefresh              TaskQueue = "contracts-refresh"
	TaskQueueContractsAggregateInvoiceData TaskQueue = "contracts-aggregated-invoice-data"

	// Ava
	TaskQueueAvaCustomersEmbeddings TaskQueue = "ava-customers-embeddings"

	// Recalculation
	TaskQueueRecalculation TaskQueue = "cmp-aws-cur-recalculate"

	// Datahub
	TaskQueueDatahubDeleteCustomerData TaskQueue = "datahub-delete-customer-data"
)

var (
	queueResourceNameFormat   = "projects/%s/locations/%s/queues/%s"
	serviceAccountEmailFormat = "gcp-jobs@%s.iam.gserviceaccount.com"
)

type CloudTaskConfig struct {
	Method           cloudtaskspb.HttpMethod
	Path             string
	Queue            TaskQueue
	Body             []byte
	ScheduleTime     *timestamppb.Timestamp
	DispatchDeadline *durationpb.Duration
	URL              string
	Audience         string
}

type CloudTaskConfigAppEngine struct {
	Queue        TaskQueue
	Method       cloudtaskspb.HttpMethod
	RelativeURI  string
	Service      string
	Body         []byte
	ScheduleTime *timestamppb.Timestamp
}

func (ctc *CloudTaskConfig) Config(payload interface{}) *iface.Config {
	email := fmt.Sprintf(serviceAccountEmailFormat, ProjectID)

	// default old audience we use
	audience := GAEService
	if ctc.Audience != "" {
		audience = ctc.Audience
	}

	// default url for current project
	url := CreateCloudTaskURL(ctc.Path)
	if ctc.URL != "" {
		url = ctc.URL
	}

	return &iface.Config{
		Project:             ProjectID,
		Location:            location,
		QueueID:             string(ctc.Queue),
		URL:                 url,
		Audience:            audience,
		ServiceAccountEmail: email,
		Payload:             payload,
		HTTPMethod:          ctc.Method,
		ScheduleTime:        ctc.ScheduleTime,
		DispatchDeadline:    ctc.DispatchDeadline,
	}
}

func (ctc *CloudTaskConfig) AppEngineConfig(payload interface{}) *iface.AppEngineConfig {
	return &iface.AppEngineConfig{
		Project:          ProjectID,
		Location:         location,
		QueueID:          string(ctc.Queue),
		RelativeURI:      ctc.Path,
		Payload:          payload,
		HTTPMethod:       ctc.Method,
		ScheduleTime:     ctc.ScheduleTime,
		DispatchDeadline: ctc.DispatchDeadline,
	}
}

// TimeToTimestamp creates timestamp.Timestamp from go time.Time
func TimeToTimestamp(t time.Time) *timestamppb.Timestamp {
	seconds := t.Unix()

	return &timestamppb.Timestamp{
		Seconds: seconds,
		Nanos:   int32(t.UnixNano() - seconds),
	}
}

func GetQueueResourceName(queue TaskQueue) string {
	return fmt.Sprintf(queueResourceNameFormat, ProjectID, location, queue)
}

func CreateAppEngineAudienceWithValues(service, project string) string {
	return fmt.Sprintf(appEngineURLFormat, service, project)
}

func CreateAppEngineAudience() string {
	return CreateAppEngineAudienceWithValues(GAEService, ProjectID)
}

// CreateCloudTaskURLWithValues is the full app engine with project and service URL
func CreateCloudTaskURLWithValues(service, project, path string) string {
	return CreateAppEngineAudienceWithValues(service, project) + path
}

// CreateCloudTaskURL is the full app engine URL
func CreateCloudTaskURL(path string) string {
	return CreateCloudTaskURLWithValues(GAEService, ProjectID, path)
}

// Deprecated: Use connection.CloudTaskClient.CreateTask or connection.CloudTaskClient.CreateAppEngineCloudTask instead.
// CreateCloudTask constructs a task with a authorization token
// and HTTP target then adds it to a Queue.
func CreateCloudTask(ctx context.Context, config *CloudTaskConfig) (*cloudtaskspb.Task, error) {
	email := fmt.Sprintf(serviceAccountEmailFormat, ProjectID)

	var audience string
	if config.Audience != "" {
		audience = config.Audience
	} else {
		// default old audience we use
		audience = GAEService
	}

	var url string
	if config.URL != "" {
		url = config.URL
	} else {
		// default url for current project
		url = CreateCloudTaskURL(config.Path)
	}

	createTaskRequest := &cloudtaskspb.CreateTaskRequest{
		Parent: GetQueueResourceName(config.Queue),
		Task: &cloudtaskspb.Task{
			MessageType: &cloudtaskspb.Task_HttpRequest{
				HttpRequest: &cloudtaskspb.HttpRequest{
					HttpMethod: config.Method,
					Url:        url,
					AuthorizationHeader: &cloudtaskspb.HttpRequest_OidcToken{
						OidcToken: &cloudtaskspb.OidcToken{
							ServiceAccountEmail: email,
							Audience:            audience,
						},
					},
				},
			},
		},
	}

	if config.DispatchDeadline != nil {
		createTaskRequest.Task.DispatchDeadline = config.DispatchDeadline
	} else {
		// set the default dispatch deadline to 30 minutes (maximum duration)
		createTaskRequest.Task.DispatchDeadline = durationpb.New(time.Minute * 30)
	}

	if config.ScheduleTime != nil {
		createTaskRequest.Task.ScheduleTime = config.ScheduleTime
	}

	if config.Body != nil {
		createTaskRequest.Task.GetHttpRequest().Body = config.Body
	}

	createdTask, err := ct.CreateTask(ctx, createTaskRequest)
	if err != nil {
		return nil, fmt.Errorf("error creating cloud task: %s", err.Error())
	}

	return createdTask, nil
}

func CreateAppEngineCloudTask(ctx context.Context, config *CloudTaskConfigAppEngine) (*cloudtaskspb.Task, error) {
	createTaskRequest := &cloudtaskspb.CreateTaskRequest{
		Parent: GetQueueResourceName(config.Queue),
		Task: &cloudtaskspb.Task{
			MessageType: &cloudtaskspb.Task_AppEngineHttpRequest{
				AppEngineHttpRequest: &cloudtaskspb.AppEngineHttpRequest{
					HttpMethod:  config.Method,
					RelativeUri: config.RelativeURI,
					AppEngineRouting: &cloudtaskspb.AppEngineRouting{
						Service: config.Service,
					},
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: config.Body,
				},
			},
			ScheduleTime: config.ScheduleTime,
		},
	}

	// Create the task
	createdTask, err := ct.CreateTask(ctx, createTaskRequest)
	if err != nil {
		return nil, fmt.Errorf("client.CreateTask: %v", err)
	}
	return createdTask, nil
}
