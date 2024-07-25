package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/iam/apiv1/iampb"
	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"google.golang.org/api/googleapi"

	domainBackfill "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type clients struct {
	bigqueryClient   *bigquery.Client
	customerBQClient *bigquery.Client
	storageClient    *storage.Client
}

const (
	tmpFilesPrefix         = "data_"
	regionBucketNameFormat = "%s-gcp-billing-data-%s"
)

func (s *BackfillService) BackfillCustomer(
	ctx context.Context,
	customerID string,
	taskBody *domainBackfill.TaskBodyHandlerCustomer,
) error {
	l := s.loggerProvider(ctx)
	startTime := time.Now()
	category := "HandleCustomer"
	action := "copyGCPBillingData"

	var flowInfo domainBackfill.FlowInfo

	config, err := s.backfillDAL.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed reading config; %v", err)
	}

	billingAccountID := taskBody.BillingAccountID

	asset, err := s.backfillDAL.GetCustomerAsset(ctx, customerID, billingAccountID)
	if err != nil {
		return err
	}

	if len(asset.Tables) != 1 {
		return fmt.Errorf(ErrInvalidAssetStr, customerID, billingAccountID, err)
	}

	t := asset.Tables[0]

	if t.Table == "" || t.Dataset == "" || t.Project == "" {
		return fmt.Errorf(ErrInvalidAssetStr, customerID, billingAccountID, err)
	}

	jobID, _ := uuid.NewRandom()
	flowInfo.TotalSteps = 8
	flowInfo.JobID = jobID.String()
	flowInfo.Operation = "cmp.copy.gcp"
	flowInfo.ProjectID = t.Project
	flowInfo.DatasetID = t.Dataset
	flowInfo.TableID = t.Table
	flowInfo.PartitionDate = taskBody.PartitionDate
	flowInfo.BillingAccountID = billingAccountID
	flowInfo.CustomerID = customerID
	flowInfo.Config = config

	l.Infof("GCP billing data backfill task started for customer: %s", customerID)
	err = s.initCopyCustomerBillingData(ctx, &flowInfo)
	logToCloudLogging(l, category, action, "sectionRun", "InitCopyCustomerBillingData", startTime, err, &flowInfo)

	if err != nil {
		return err
	}

	return nil
}

// InitCopyCustomerBillingData initializes a copy job for Customer Billing Data with given customerID.
// If partitionDate is given, this function will copy only a given partition. Otherwise it will copy the entire table
func (s *BackfillService) initCopyCustomerBillingData(ctx context.Context, flowInfo *domainBackfill.FlowInfo) error {
	l := s.loggerProvider(ctx)
	category := "InitCopyCustomerBillingData"
	action := "getCustomerDataFromFS"
	startTime := time.Now()

	// Initialize BQ and GCS clients
	clients, err := s.initializeClients(ctx, flowInfo)
	if err != nil {
		return err
	}

	defer clients.Close()

	// Get customer Cloud Connect (cc) doc from Firestore
	cloudConnect, err := s.backfillDAL.GetCustomerGCPDoc(ctx, flowInfo.CustomerID)
	if len(cloudConnect.Docs) == 0 {
		l.Errorf(ErrAbortTaskStr, domainBackfill.E0001, err)
		err = fmt.Errorf(domainBackfill.E0001)
	}

	if flowInfo.BillingAccountID != "" {
		s.reportProgress(ctx, category, action, "sectionRun", "getCustomerGCPDoc", startTime, err, flowInfo)
	} else {
		logToCloudLogging(l, category, action, "sectionRun", "getCustomerGCPDoc", startTime, err, flowInfo)
	}

	if err != nil {
		errorDesc := fmt.Sprintf("Aborting task. Failed while getting cloudConnect doc for customer %s.", flowInfo.CustomerID)
		l.Errorf(errorDesc)

		return err
	}

	// Get customer GCP Credentials. There should be only one SA per customer, that's why we fetch the first element from the array.
	cloudConnectDoc := cloudConnect.Docs[0]
	customerEmail := cloudConnectDoc.GCPCredentials.ClientEmail

	customerCredentials, err := common.NewGcpCustomerAuthService(&cloudConnectDoc.GCPCredentials).GetClientOption()
	if err != nil {
		l.Errorf(ErrAbortTaskStr, "Failed getting customer's direct account assets", err)
		return err
	}

	// Get customer GCP billing accounts with billing tables location
	startTime = time.Now()
	billingAccountDocs, err := s.backfillDAL.GetDirectBillingAccountsDocs(ctx, flowInfo.CustomerID)
	s.reportProgress(ctx, category, action, "sectionRun", "getDirectBillingAccountsDocs", startTime, err, flowInfo)

	if err != nil {
		l.Errorf(ErrAbortTaskStr, "Failed getting customer's direct account assets", err)
		return err
	}

	// Loop over all direct billing account a customer has
	for j := range billingAccountDocs {
		var billingTablesToCopy domainBackfill.BillingTables

		startTime = time.Now()
		err := billingAccountDocs[j].DataTo(&billingTablesToCopy)
		s.reportProgress(ctx, category, action, "sectionRun", "deserializeBillingTablesStruct", startTime, err, flowInfo)

		if err != nil {
			l.Errorf(ErrAbortTaskStr, "Failed deserializing Billing Tables struct", err)
			return err
		}

		// Get customer unique dest table ID under which it will be stored in DoiT project
		billingAccountID := fmt.Sprintf("%v", billingTablesToCopy.Properties["billingAccountId"])

		// If billing account ID specified in the request body, process only it and skip customer's remainig billing accounts
		if flowInfo.BillingAccountID != "" && flowInfo.BillingAccountID != billingAccountID {
			continue
		}

		flowInfo.BillingAccountID = billingAccountID
		billingAccount := domainBackfill.BillingAccount{
			BillingAccountID:    billingAccountID,
			CustomerEmail:       customerEmail,
			CustomerID:          flowInfo.CustomerID,
			CustomerCredentials: customerCredentials,
			CloudConnectDocID:   cloudConnectDoc.DocID,
		}

		dstTable := strings.ReplaceAll(fmt.Sprintf(flowInfo.Config.DestinationTableFormat, billingAccountID), "-", "_")
		dstDataset := strings.ReplaceAll(fmt.Sprintf(flowInfo.Config.DestinationDatasetFormat, billingAccountID), "-", "_")

		// Initialize TableCopyJob struct
		tableCopyJobDetails := domainBackfill.TableCopyJob{
			PartitionDate:       flowInfo.PartitionDate,
			DstDataset:          dstDataset,
			DstTable:            dstTable,
			DstTableNoDecorator: dstTable,
			ExportRows:          -1,
			LoadRows:            -1,
		}

		err = s.handleBillingAccount(ctx, clients, &billingAccount, &billingTablesToCopy, &tableCopyJobDetails, flowInfo)
		s.reportProgress(ctx, category, "copyBillingAccountData", "sectionRun", "handleBillingAccount", startTime, err, flowInfo)

		if err != nil {
			l.Errorf("aborting task. Failed copying billing data; %v", err)
			return err
		}
	}

	return nil
}

func (s *BackfillService) handleBillingAccount(
	ctx context.Context,
	clients *clients,
	billingAccount *domainBackfill.BillingAccount,
	billingTablesToCopy *domainBackfill.BillingTables,
	tableCopyJobDetails *domainBackfill.TableCopyJob,
	flowInfo *domainBackfill.FlowInfo,
) error {
	category := "handleBillingAccount"
	startTime := time.Now()
	l := s.loggerProvider(ctx)

	// Make sure the target dataset for storing customer's billing data exists on DoiT side - if not, create it
	// The location is obtained by joining the following strings: cfg.DestinationDatasetFormat, billingAccountId
	err := clients.createDatasetAndGrantPermissions(ctx, tableCopyJobDetails.DstDataset, billingAccount.CustomerEmail)
	if err != nil {
		l.Errorf(ErrAbortTaskStr, domainBackfill.E0007, err)
		err = fmt.Errorf(domainBackfill.E0007)

		return err
	}

	s.reportProgress(ctx, category, "createDestinationDataset", "sectionRun", "createDatasetAndGrantPermissions", startTime, err, flowInfo)

	// Make sure the target table for storing customer's billing data exists on DoiT side - if not, create it
	// The location is obtained by joining the following strings: cfg.DestinationDatasetFormat, cfg.DestinationTableFormat, billingAccountId
	startTime = time.Now()
	exists, err := clients.tableExists(ctx, tableCopyJobDetails.DstDataset, tableCopyJobDetails.DstTable)
	l.Infof("target table %s.%s exists: %t", tableCopyJobDetails.DstDataset, tableCopyJobDetails.DstTable, exists)

	if err != nil {
		l.Errorf(ErrAbortTaskStr, domainBackfill.E0008, err)
		return fmt.Errorf(domainBackfill.E0008)
	}

	s.reportProgress(ctx, category, "createDestinationTable", "sectionRun", "tableExists", startTime, err, flowInfo)

	// If target table does not exist, create it
	if !exists {
		startTime = time.Now()

		err = clients.createCustomerBillingDataTable(
			ctx,
			flowInfo.Config.TemplateBillingDataDatasetID,
			flowInfo.Config.TemplateBillingDataTableID,
			tableCopyJobDetails.DstDataset,
			tableCopyJobDetails.DstTable,
		)

		if err != nil {
			l.Errorf(ErrAbortTaskStr, domainBackfill.E0009, err)
			return fmt.Errorf(domainBackfill.E0009)
		}

		s.reportProgress(ctx, category, "createDestinationTable", "sectionRun", "createCustomerBillingDataTable", startTime, err, flowInfo)
	}

	// Copy only the first table specified by a customer
	startTime = time.Now()
	err = s.copyCustomerBillingTable(ctx, clients, billingAccount, tableCopyJobDetails, billingTablesToCopy.Tables[0], flowInfo)
	s.reportProgress(ctx, category, "copyBillingTable", "sectionRun", "copyCustomerBillingTable", startTime, err, flowInfo)

	if err != nil {
		errorDesc := fmt.Sprintf("aborting task. Failed copying customer billing data; %v", err)
		l.Errorf(errorDesc)

		return err
	}

	return nil
}

func (s *BackfillService) initializeClients(ctx context.Context, flowInfo *domainBackfill.FlowInfo) (*clients, error) {
	// Initialize DoiT BQ client
	bigqueryClient, err := bigquery.NewClient(ctx, flowInfo.Config.DestinationProject)
	if err != nil {
		return nil, err
	}

	// Initialize GCS Client
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &clients{
		bigqueryClient: bigqueryClient,
		storageClient:  storageClient,
	}, nil
}

// Close calls Close() on each client of Clients
func (clients *clients) Close() {
	clients.bigqueryClient.Close()
	clients.storageClient.Close()
}

func (s *BackfillService) copyCustomerBillingTable(
	ctx context.Context,
	clients *clients,
	billingAccountDetails *domainBackfill.BillingAccount,
	tableCopyJobDetails *domainBackfill.TableCopyJob,
	table *domainBackfill.BillingTable,
	flowInfo *domainBackfill.FlowInfo,
) error {
	l := s.loggerProvider(ctx)
	category := "handleBillingAccount"
	startTime := time.Now()
	action := "copyBillingTable"

	// If source table is not specified, assume it is the default billing data table name: gcp_billing_export_resource_v1_XXXXXX_XXXXXX_XXXXXX
	if table.Table == "" {
		table.Table = strings.ReplaceAll(fmt.Sprintf("%s_%s", flowInfo.Config.DestinationTableFormat, billingAccountDetails.BillingAccountID), "-", "_")
	}

	// Initialize customer BQ client
	customerBQClient, err := bigquery.NewClient(ctx, table.Project, billingAccountDetails.CustomerCredentials)
	logToCloudLogging(l, category, action, "initializeClient", "BigQuery", startTime, err, flowInfo)

	if err != nil {
		l.Errorf(ErrSkipTableStr, domainBackfill.E0004, err)
		return fmt.Errorf(domainBackfill.E0004)
	}

	defer customerBQClient.Close()
	clients.customerBQClient = customerBQClient

	// If dataset location containing billing table is unknown, get it
	if table.Location == "" {
		// Check dataset location
		datasetLocation, err := getDatasetLocation(ctx, clients.customerBQClient, table.Project, table.Dataset)
		logToCloudLogging(l, category, action, "sectionRun", "getDatasetLocation", startTime, err, flowInfo)

		if err != nil {
			l.Errorf(ErrSkipTableStr, domainBackfill.E0003, err)
			return fmt.Errorf(domainBackfill.E0003)
		}

		// Set the location in table struct
		table.Location = datasetLocation
	}

	// Check if bucket location in given region is already set in FS
	regionBucket := flowInfo.Config.RegionsBuckets[table.Location]
	if regionBucket == "" {
		startTime = time.Now()
		// If bucket location is not set yet, create it and update FS
		regionBucket = strings.ToLower(fmt.Sprintf(regionBucketNameFormat, flowInfo.Config.DestinationProject, table.Location))
		err := s.createBucket(ctx, clients.storageClient, flowInfo.Config.DestinationProject, table.Location, regionBucket)
		logToCloudLogging(l, category, action, "sectionRun", "createBucket", startTime, err, flowInfo)

		if err != nil {
			l.Errorf("%s for region: %s; %v", domainBackfill.E0010, table.Location, err)
			return fmt.Errorf(domainBackfill.E0010)
		}

		l.Infof("created bucket %s in %s", regionBucket, table.Location)

		// Update bucket location in FS
		flowInfo.Config.RegionsBuckets[table.Location] = regionBucket

		err = s.backfillDAL.UpdateConfigDoc(ctx, table.Location, regionBucket)
		if err != nil {
			l.Errorf("failed to update config doc; %v", err)
		}
	}

	// Update regionBucket value inside the struct
	tableCopyJobDetails.RegionBucket = regionBucket

	var dataToBeCopiedSizeQuery string

	if tableCopyJobDetails.PartitionDate == "" {
		l.Infof("Copying entire table '%s'", table.Table)
		dataToBeCopiedSizeQuery = fmt.Sprintf(tableSizeQueryFormat, table.Dataset, table.Table)
		tableCopyJobDetails.SrcTable = table.Table
	} else {
		l.Infof("Copying partition %s from table '%s'", tableCopyJobDetails.PartitionDate, table.Table)
		dataToBeCopiedSizeQuery = fmt.Sprintf(partitionSizeQueryFormat, table.Dataset, table.Table, tableCopyJobDetails.PartitionDate)
		partitionToCopy := strings.ReplaceAll(tableCopyJobDetails.PartitionDate, "-", "")
		tableCopyJobDetails.SrcTable = fmt.Sprintf("%s$%s", table.Table, partitionToCopy)
		tableCopyJobDetails.DstTable = fmt.Sprintf("%s$%s", tableCopyJobDetails.DstTable, partitionToCopy)
	}

	startTime = time.Now()
	results, err := executeQuery(ctx, clients.customerBQClient, dataToBeCopiedSizeQuery)
	s.reportProgress(ctx, category, action, "queryRun", "dataToBeCopiedSizeQuery", startTime, err, flowInfo)

	if err != nil || len(results) == 0 {
		if gapiErr, ok := err.(*googleapi.Error); ok && gapiErr.Code == http.StatusForbidden {
			// break partitions loop if there is no acces to table
			l.Errorf(ErrSkipTableStr, domainBackfill.E0005, err)
			return fmt.Errorf(domainBackfill.E0005)
		}

		l.Errorf("skipping table partition %s; %v", domainBackfill.E0006, err)

		return fmt.Errorf(domainBackfill.E0006)
	}

	tableCopyJobDetails.ExportRows = results[0].Count

	startTime = time.Now()
	err = s.initCopyingData(ctx, clients, billingAccountDetails, tableCopyJobDetails, table, flowInfo)
	logToCloudLogging(l, category, action, "sectionRun", "initCopyingData", startTime, err, flowInfo)

	if err != nil {
		l.Errorf("%s; %v", domainBackfill.E0011, err)
		return fmt.Errorf(domainBackfill.E0011)
	}

	return nil
}

func (s *BackfillService) initCopyingData(
	ctx context.Context,
	clients *clients,
	billingAccountDetails *domainBackfill.BillingAccount,
	tableCopyJobDetails *domainBackfill.TableCopyJob,
	table *domainBackfill.BillingTable,
	flowInfo *domainBackfill.FlowInfo,
) error {
	var loadRows int64

	l := s.loggerProvider(ctx)

	// Copy data via the GCS route
	numberLoadRows, err := s.copyDataViaGCSRoute(ctx, clients, billingAccountDetails, tableCopyJobDetails, table, flowInfo)
	loadRows = numberLoadRows

	if err != nil {
		l.Errorf("copying data with GCS as intermediate step failed; %s", err)
		return err
	}

	if loadRows != tableCopyJobDetails.ExportRows {
		err = fmt.Errorf(
			"mismatch between number of rows in partition to the number of rows loaded to BigQuery: %v loadRows vs %v export rows",
			loadRows, tableCopyJobDetails.ExportRows)
		l.Errorf("error initCopyingData: %s", err)

		return err
	}

	if tableCopyJobDetails.PartitionDate == "" {
		path := fmt.Sprintf("/tasks/analytics/google-cloud/accounts/%s?allPartitions=true&refreshMetadata=true&assetType=%s",
			billingAccountDetails.BillingAccountID, common.Assets.GoogleCloudDirect)

		// Trigger a cloud task to enrich copied data
		c := &common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   path,
			Queue:  common.TaskQueueCloudAnalyticsTablesGCP,
		}

		if _, err = s.conn.CloudTaskClient.CreateAppEngineTask(ctx, c.AppEngineConfig(nil)); err != nil {
			return err
		}
	}

	l.Infof("succesfully copied data for billing account %s from table %s.%s.%s \n",
		billingAccountDetails.BillingAccountID,
		table.Project, table.Dataset, table.Table)

	return nil
}

func (s *BackfillService) copyDataViaGCSRoute(
	ctx context.Context,
	clients *clients,
	billingAccountDetails *domainBackfill.BillingAccount,
	tableCopyJobDetails *domainBackfill.TableCopyJob,
	table *domainBackfill.BillingTable,
	flowInfo *domainBackfill.FlowInfo,
) (int64, error) {
	category := "copyDataViaGCSRoute"
	startTime := time.Now()
	action := "copyBillingTable"
	l := s.loggerProvider(ctx)

	// Grant write permissions to the customer's SA to export billing data to our GCS bucket
	err := s.grantCustomerSAWritePermission(
		ctx,
		clients.storageClient,
		tableCopyJobDetails.RegionBucket,
		billingAccountDetails.CustomerEmail,
		flowInfo)

	s.reportProgress(ctx, category, action, "sectionRun", "grantCustomerSAWritePermission", startTime, err, flowInfo)

	if err != nil {
		errorDesc := fmt.Sprintf("failed updating bucket with permissions and object life cycle. Bucket: %s.\n Error: %v",
			tableCopyJobDetails.RegionBucket, err)
		l.Errorf(errorDesc)

		return 0, err
	}

	// Get target location for exporting billing data
	gcsTmpObjectsFolder := fmt.Sprintf("gs://%s/%s/%s/%s/%v",
		tableCopyJobDetails.RegionBucket, billingAccountDetails.CustomerID,
		table.Project, table.Dataset, common.MakeTimestamp())

	gcsURI := fmt.Sprintf("%s/%s/%s*", gcsTmpObjectsFolder, billingAccountDetails.BillingAccountID, tmpFilesPrefix)

	l.Infof("table: %s.%s.%s partition: %s size %v \n",
		table.Project, table.Dataset, tableCopyJobDetails.SrcTable,
		tableCopyJobDetails.PartitionDate, tableCopyJobDetails.ExportRows)

	startTime = time.Now()

	err = clients.exportTableToRegionBucket(ctx, table, gcsURI, tableCopyJobDetails.SrcTable, table.Location)
	s.reportProgress(ctx, category, action, "sectionRun", "exportTableToRegionBucket", startTime, err, flowInfo)

	if err != nil {
		errorDesc := fmt.Sprintf("Skipping table. Reason: failed exporting data from customer's table.\n Error: %v", err)
		l.Errorf(errorDesc)

		return 0, err
	}

	startTime = time.Now()
	loadRows, err := clients.loadFilesToBQ(
		ctx, flowInfo.Config.DestinationProject, tableCopyJobDetails.DstDataset, tableCopyJobDetails.DstTable, gcsURI,
	)

	s.reportProgress(ctx, category, action, "sectionRun", "loadFilesToBQ", startTime, err, flowInfo)

	tableCopyJobDetails.LoadRows = loadRows

	if err != nil {
		errorDesc := fmt.Sprintf("Failed loading files to customer's table.\n Error: %v", err)
		l.Errorf(errorDesc)

		return 0, err
	}

	return loadRows, nil
}

func (s *BackfillService) grantCustomerSAWritePermission(
	ctx context.Context,
	c *storage.Client,
	bucketName, clientEmail string,
	flowInfo *domainBackfill.FlowInfo,
) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	bucket := c.Bucket(bucketName)

	policy, err := bucket.IAM().V3().Policy(ctx)
	if err != nil {
		return err
	}

	role := fmt.Sprintf(flowInfo.Config.StorageRole, flowInfo.Config.DestinationProject)

	policy.Bindings = append(policy.Bindings, &iampb.Binding{
		Role:    role,
		Members: []string{"serviceAccount:" + clientEmail},
	})
	if err := bucket.IAM().V3().SetPolicy(ctx, policy); err != nil {
		return err
	}

	return nil
}

func (s *BackfillService) createBucket(
	ctx context.Context,
	client *storage.Client,
	projectID, location, bucketName string,
) error {
	bucketAttrs := &storage.BucketAttrs{
		Location: location,
		Lifecycle: storage.Lifecycle{
			Rules: []storage.LifecycleRule{
				{
					Action: storage.LifecycleAction{Type: "Delete"},
					Condition: storage.LifecycleCondition{
						AgeInDays: 1,
					},
				},
			},
		},
	}

	bucket := client.Bucket(bucketName)

	if err := bucket.Create(ctx, projectID, bucketAttrs); err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok {
			if gapiErr.Code != http.StatusConflict {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (s *BackfillService) reportProgress(
	ctx context.Context,
	category, action, logContext, subContext string,
	startTime time.Time,
	err error,
	flowInfo *domainBackfill.FlowInfo,
) {
	l := s.loggerProvider(ctx)
	progress := float64(domainBackfill.Steps[subContext] * 100 / flowInfo.TotalSteps)
	status := s.getJobStatusFromProgress(progress)

	e := s.backfillDAL.UpdateAssetCopyJobProgress(ctx, status, progress, err, flowInfo)
	if e != nil {
		l.Errorf("error updating asset copy job progress; %v", e)
		logToCloudLogging(l, category, action, "updateAssetCopyJobProgress", subContext, startTime, e, flowInfo)
	}

	logToCloudLogging(l, category, action, logContext, subContext, startTime, err, flowInfo)
}

func (s *BackfillService) getJobStatusFromProgress(progress float64) string {
	status := "processing"

	if progress == 100.0 {
		status = "done"
	}

	return status
}
