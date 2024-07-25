package ava

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/common"
	httpClient "github.com/doitintl/http"
	"github.com/doitintl/idtoken"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Request struct {
	CustomerID string `json:"customerId"`
}

type ProcessMetadata struct {
	LastUpdated time.Time `firestore:"lastUpdated"`
	NeedsUpdate bool      `firestore:"needsUpdate"`
	Progress    float64   `firestore:"progress"`
}

type Metadata struct {
	URL        string `json:"url"`
	CustomerID string `json:"customerId"`
	Path       string `json:"path"`
	Key        string `json:"key"`
	Label      string `json:"label"`
	Field      string `json:"field"`
	Type       string `json:"type"`
	FieldID    string `json:"field_id"`
	Name       string `json:"name"`
	Source     string `json:"source"`
}

// Document represents a document within the documents array
type Document struct {
	ID          string   `json:"id"`
	PageContent string   `json:"pageContent"`
	Metadata    Metadata `json:"metadata"`
}

// Report represents the entire JSON structure
type Report struct {
	Source       string     `json:"source"`
	Documents    []Document `json:"documents"`
	AllowDelete  bool       `json:"allowDelete"`
	FullSync     bool       `json:"fullSync"`
	ChunkSize    int        `json:"chunkSize"`
	ChunkOverlap int        `json:"chunkOverlap"`
}

type worker struct {
	customerID string
	Path       string
	Source     string
	DocSnap    *firestore.DocumentSnapshot
}

type FilterValueDoc struct {
	ID     string        `firestore:"id"`
	Values []interface{} `firestore:"values"`
	Label  string        `firestore:"label"`
	Key    string        `firestore:"key"`
	Cloud  string        `firestore:"cloud"`
	Field  string        `firestore:"field"`
	Type   string        `firestore:"type"`
}

type Service struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	httpClient     httpClient.IClient
}

const (
	LocalhostURL          = "http://localhost:8080"
	ProductionURL         = "https://api4prod.doit.com"
	DevelopmentURL        = "https://api4dev.doit.com"
	sourceName            = "report-org-metadata"
	sourceAttributionName = "attributions"
	numWorkers            = 7
)

func NewAvaService(ctx context.Context, loggerProvider logger.Provider, conn *connection.Connection) *Service {
	l := loggerProvider(ctx)
	envURL := getEnv()

	tokenSource, err := idtoken.New().GetTokenSource(ctx, getAudience())
	if err != nil {
		l.Fatalf("could not get token source. error [%s]", err)
	}

	embeddingsClient, err := httpClient.NewClient(ctx, &httpClient.Config{
		BaseURL:     envURL,
		TokenSource: tokenSource,
	})
	if err != nil {
		l.Fatalf("could not get avalara client. error [%s]", err)
	}

	return &Service{
		loggerProvider,
		conn,
		embeddingsClient,
	}
}

// load document from firestore
func (s *Service) LoadDocument(ctx context.Context, path string) (*firestore.DocumentSnapshot, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	l.Infof("called with path %v", path)

	trimPath := strings.TrimPrefix(path, "/")

	docRef := fs.Doc(trimPath)

	docSnapshot, err := docRef.Get(ctx)
	if err != nil {
		l.Errorf("failed to get document: %v", err)
		return nil, err
	}

	return docSnapshot, nil
}

func (s *Service) transformDateAndReference(docData map[string]interface{}) map[string]interface{} {
	for key, value := range docData {
		// If the field is a Firestore document reference, convert it to a string
		if ref, ok := value.(*firestore.DocumentRef); ok {
			docData[key] = ref.Path
		}

		// If the field is a time.Time, convert it to an ISO 8601 string
		if t, ok := value.(time.Time); ok {
			docData[key] = t.Format(time.RFC3339)
		}
	}

	return docData
}

func normalizeJobs(totalJobs int) float64 {
	if totalJobs == 0 {
		return 0
	}

	return 100 / float64(totalJobs)
}

// update progress bar by percentage, total count job versus total count of jobs that done
func (s *Service) updateProgressBar(ctx context.Context, totalJobs int, customerID string) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	if customerID == "" {
		return fmt.Errorf("customerID is empty")
	}

	unit := normalizeJobs(totalJobs)

	if _, err := fs.Collection("app").Doc("ava").Collection("customersMetadata").Doc(customerID).Update(ctx, []firestore.Update{
		{Path: "progress", Value: firestore.Increment(unit)},
	}); err != nil {
		l.Errorf("failed to update document: %v", err)
		return err
	}

	return nil
}

// UpsertFirestoreDocumentEmbeddings upserts the Firestore document embeddings
func (s *Service) UpsertReportMetadataEmbeddings(ctx context.Context, jobs <-chan *worker, wg *sync.WaitGroup, jobsCount int) {
	l := s.loggerProvider(ctx)

	for job := range jobs {

		docSnapshot, err := s.LoadDocument(ctx, job.Path)
		if err != nil {
			l.Errorf("failed to load document: %v", err)

			wg.Done()
			continue
		}

		trimPath := strings.TrimPrefix(job.Path, "/")
		docData := docSnapshot.Data()

		var metadataValues FilterValueDoc

		if err := docSnapshot.DataTo(&metadataValues); err != nil {
			l.Errorf("failed to convert document data: %v", err)

			wg.Done()
			continue
		}

		for key, value := range docData {
			// If the field is a Firestore document reference, convert it to a string
			if ref, ok := value.(*firestore.DocumentRef); ok {
				docData[key] = ref.Path
			}

			// If the field is a time.Time, convert it to an ISO 8601 string
			if t, ok := value.(time.Time); ok {
				docData[key] = t.Format(time.RFC3339)
			}
		}

		// create string list with break line after each value
		values := make([]string, 0)
		for _, value := range metadataValues.Values {
			values = append(values, fmt.Sprintf("%s, ",
				value,
			))
		}

		output := fmt.Sprintf(
			"ID: %s, key: %s, Field: %s, type: %s, Label: %s, cloud_provider:%s\n Values: %s",
			docSnapshot.Ref.ID,
			metadataValues.Key,
			metadataValues.Field,
			metadataValues.Type,
			metadataValues.Label,
			metadataValues.Cloud,
			values,
		)

		id := extractID(trimPath)

		document := Document{
			ID:          id,
			PageContent: output,
			Metadata: Metadata{
				URL:        "https://console.doit.com",
				CustomerID: job.customerID,
				Path:       trimPath,
				Key:        metadataValues.Key,
				Label:      metadataValues.Label,
				Field:      metadataValues.Field,
				Type:       metadataValues.Type,
				FieldID:    docSnapshot.Ref.ID,
			},
		}

		if err := s.publishEmbeddings(ctx, &Report{
			Source:       fmt.Sprintf("doit/firestore/%s", job.Source),
			Documents:    []Document{document},
			AllowDelete:  false,
			ChunkSize:    2000,
			ChunkOverlap: 200,
		}); err != nil {
			l.Errorf("failed to publish embeddings: %v", err)
		}

		wg.Done()

		if err := s.updateProgressBar(ctx, jobsCount, job.customerID); err != nil {
			l.Errorf("failed to update progress bar: %v", err)
		}

	}
}

// PopulateCustomerFilterValues embeds the customer filter values
func (s *Service) PopulateCustomerFilterValues(ctx context.Context, customerID string) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	customerRef := fs.Collection("customers").Doc(customerID)
	orgRef := fs.Collection("organizations").Doc("doit-international")

	docSnaps, err := fs.
		CollectionGroup("reportOrgMetadata").
		Where("customer", "==", customerRef).
		Where("organization", "==", orgRef).
		Documents(ctx).
		GetAll()
	if err != nil {
		l.Errorf("failed to get documents: %v", err)
		return err
	}

	l.Infof("found %d documents", len(docSnaps))

	if err := s.updateMetadataLastUpdated(ctx, customerID); err != nil {
		l.Errorf("failed to update metadata last updated: %v", err)
		return err
	}

	jobsArr := make([]*worker, 0)

	var wg sync.WaitGroup

	for _, docSnap := range docSnaps {

		var filterValues FilterValueDoc
		if err := docSnap.DataTo(&filterValues); err != nil {
			l.Errorf("failed to convert document data: %v", err)
			continue
		}

		// skip if the field is invoice - experimental field
		if strings.HasPrefix(filterValues.Field, "T.invoice.") {
			continue
		}

		if len(filterValues.Values) == 0 {
			continue
		}

		jobsArr = append(jobsArr, &worker{
			customerID: customerID,
			Path:       extractPath(docSnap.Ref.Path),
			Source:     sourceName,
		})
	}

	jobs := make(chan *worker, len(jobsArr))
	for _, job := range jobsArr {
		jobs <- job

		wg.Add(1)
	}

	for w := 1; w <= numWorkers; w++ {
		go s.UpsertReportMetadataEmbeddings(
			ctx,
			jobs,
			&wg,
			len(jobsArr),
		)
	}

	wg.Wait()
	close(jobs)

	return nil
}

func (s *Service) UpsertAttributionEmbeddings(ctx context.Context, jobs <-chan *worker, wg *sync.WaitGroup) {
	l := s.loggerProvider(ctx)

	for job := range jobs {

		var attribution attribution.Attribution

		if err := job.DocSnap.DataTo(&attribution); err != nil {
			l.Errorf("failed to convert document data: %v", err)

			wg.Done()
			continue
		}

		trimPath := strings.TrimPrefix(job.Path, "/")

		docData := s.transformDateAndReference(job.DocSnap.Data())

		jsonData, err := json.Marshal(docData)
		if err != nil {
			l.Errorf("failed to marshal document data: %v", err)
			continue
		}

		jsonString := string(jsonData)

		id := extractID(trimPath)

		document := Document{
			ID:          id,
			PageContent: jsonString,
			Metadata: Metadata{
				URL:        "https://console.doit.com",
				CustomerID: job.customerID,
				Path:       trimPath,
				Source:     sourceAttributionName,
				Name:       attribution.Name,
				Type:       "preset",
				FieldID:    job.DocSnap.Ref.ID,
			},
		}

		if err := s.publishEmbeddings(ctx, &Report{
			Source:       fmt.Sprintf("doit/firestore/%s", sourceAttributionName),
			Documents:    []Document{document},
			AllowDelete:  false,
			FullSync:     false,
			ChunkSize:    4000,
			ChunkOverlap: 300,
		}); err != nil {
			l.Errorf("failed to publish embeddings: %v", err)
		}

		wg.Done()
	}
}

func (s *Service) PopulateAttributions(ctx context.Context) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	docSnaps, err := fs.Collection("dashboards/google-cloud-reports/attributions").
		Where("type", "==", "preset").
		Documents(ctx).
		GetAll()
	if err != nil {
		l.Errorf("failed to get documents: %v", err)
		return err
	}

	l.Infof("found %d documents", len(docSnaps))

	jobsArr := make([]*worker, 0)

	var wg sync.WaitGroup

	for _, docSnap := range docSnaps {
		jobsArr = append(jobsArr, &worker{
			Path:    extractPath(docSnap.Ref.Path),
			Source:  sourceAttributionName,
			DocSnap: docSnap,
		})
	}

	jobs := make(chan *worker, len(jobsArr))
	for _, job := range jobsArr {
		jobs <- job

		wg.Add(1)
	}

	for w := 1; w <= numWorkers; w++ {
		go s.UpsertAttributionEmbeddings(
			ctx,
			jobs,
			&wg,
		)
	}

	wg.Wait()
	close(jobs)

	return nil

}

func (s *Service) IsAllowToUpdate(ctx context.Context, customerID string) bool {
	l := s.loggerProvider(ctx)

	docSnap, err := s.getCustomerLastUpdateRef(ctx, customerID).Get(ctx)
	if err != nil {
		l.Errorf("failed to get document: %v", err)
		return true
	}

	var metadata ProcessMetadata

	if err := docSnap.DataTo(&metadata); err != nil {
		l.Errorf("failed to convert document data: %v", err)
		return true
	}

	// If the document was updated less than 30 days ago, do not update it
	if time.Since(metadata.LastUpdated) > time.Duration(30*24*time.Hour) {
		l.Infof("customer %s was updated less than 30 days ago", customerID)
		return false
	}

	return true
}

// update firestore document last updated time
func (s *Service) updateMetadataLastUpdated(ctx context.Context, customerID string) error {
	l := s.loggerProvider(ctx)

	docRef := s.getCustomerLastUpdateRef(ctx, customerID)

	if _, err := docRef.Set(ctx, ProcessMetadata{
		LastUpdated: time.Now().UTC(),
		NeedsUpdate: false,
		Progress:    0,
	}); err != nil {
		l.Errorf("failed to update document: %v", err)
		return err
	}

	return nil
}

func (s *Service) CreateMetadataTask(ctx context.Context, customerID string) error {
	l := s.loggerProvider(ctx)

	body, err := json.Marshal(Request{
		CustomerID: customerID,
	})
	if err != nil {
		l.Errorf("failed to marshal request: %v", err)
		return err
	}

	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   "/tasks/ava/upsert-firestore-doc-embeddings",
		Queue:  common.TaskQueueAvaCustomersEmbeddings,
		Body:   body,
	}

	if _, err := common.CreateCloudTask(ctx, &config); err != nil {
		l.Errorf("failed to create ava customer metadata task for %s with error %s", customerID, err)
	}

	return nil
}

func (s *Service) getCustomerLastUpdateRef(ctx context.Context, customerID string) *firestore.DocumentRef {
	fs := s.conn.Firestore(ctx)
	return fs.Collection("app").Doc("ava").Collection("customersMetadata").Doc(customerID)
}

// publishEmbeddings publishes the embeddings to the embeddings service
func (s *Service) publishEmbeddings(ctx context.Context, payload interface{}) error {
	l := s.loggerProvider(ctx)

	_, err := s.httpClient.Post(ctx, &httpClient.Request{
		URL:          "/api/ava/embeddings/docs",
		Payload:      payload,
		ResponseType: nil,
	})
	if err != nil {
		l.Errorf("failed to publish embeddings: %v", err)
		return err
	}

	return nil
}

func extractPath(path string) string {
	pattern := "(default)/documents/"
	start := strings.Index(path, pattern)

	if start == -1 {
		return path
	}

	extracted := path[start:]
	cleaned := strings.Replace(extracted, pattern, "", 1)

	return cleaned
}

// extrectID generate id for the document
func extractID(path string) string {
	trimPath := strings.TrimPrefix(path, "/")
	hash := md5.Sum([]byte(trimPath))
	hashString := hex.EncodeToString(hash[:])

	return fmt.Sprintf("%v-v1-c0-0", hashString)
}

// getEnv returns the appropriate environment URL based on the application context
func getEnv() string {
	if common.IsLocalhost {
		return LocalhostURL
	}

	if common.Production {
		return ProductionURL
	}

	return DevelopmentURL
}

func getAudience() string {
	if common.Production {
		return "https://api4prod.doit.com/ava-embeddings-service"
	}

	return "https://api4dev.doit.com/ava-embeddings-service"
}
