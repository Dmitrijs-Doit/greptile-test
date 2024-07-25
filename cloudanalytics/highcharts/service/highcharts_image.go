package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	domainHighCharts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	httpClient "github.com/doitintl/http"
)

// budgetUtilization - struct holding budget's 2 utilization types -> Current & Forecasted
type budgetUtilization struct {
	Type  string
	URL   string
	Error error
}

func (s *Highcharts) GetBudgetImages(ctx context.Context, budgetID, customerID string, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) (string, string, error) {
	b, err := s.budgetService.GetBudget(ctx, budgetID)
	if err != nil {
		return "", "", err
	}

	images := make(chan *budgetUtilization)

	var wg sync.WaitGroup

	imageURLs := map[string]string{
		"current":    "",
		"forecasted": "",
	}

	for utilizationType := range imageURLs {
		wg.Add(1)

		go func(utilizationType string, images chan *budgetUtilization) {
			imageURL, err := s.getBudgetImage(ctx, b, budgetID, customerID, highchartsFontSettings, utilizationType)
			images <- &budgetUtilization{Type: utilizationType, URL: imageURL, Error: err}

			wg.Done()
		}(utilizationType, images)
	}

	go func() {
		wg.Wait()
		close(images)
	}()

	for image := range images {
		if image.Error != nil {
			return "", "", image.Error
		}

		imageURLs[image.Type] = image.URL
	}

	return imageURLs["current"], imageURLs["forecasted"], err
}

func (s *Highcharts) getBudgetImage(ctx context.Context, b *budget.Budget, budgetID, customerID string, highchartsFontSettings *domainHighCharts.HighchartsFontSettings, utilizationType string) (string, error) {
	hcr := s.GetHighchartsRequestBudget(utilizationType, b, highchartsFontSettings)

	chartImageData, err := s.GetChartImage(ctx, hcr)
	if err != nil {
		return "", err
	}

	if len(chartImageData) <= 32 {
		return "", errors.New("invalid image data")
	}

	imageURL, err := s.SaveImageToGCS(ctx, chartImageData, budgetID, customerID, "budgets")
	if err != nil {
		return "", err
	}

	return imageURL, err
}

func (s *Highcharts) GetReportImageData(ctx context.Context, reportID string, customerID string, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) ([]byte, error) {
	l := s.loggerProvider(ctx)

	email, _ := ctx.Value("email").(string)

	isTreemapExact, err := s.isExactTreemapCheck(ctx, customerID)
	if err != nil {
		return nil, err
	}

	qr, report, err := s.cloudAnalytics.GetQueryRequest(ctx, customerID, reportID)
	if err != nil {
		return nil, err
	}

	result, err := s.cloudAnalytics.GetQueryResult(ctx, qr, customerID, email)
	if err != nil {
		return nil, err
	}

	hcr, err := s.GetHighchartsRequestReport(ctx, qr, &result, report, isTreemapExact, highchartsFontSettings)
	if err != nil {
		return nil, err
	}

	l.Info(hcr.Callback)

	chartImageData, err := s.GetChartImage(ctx, hcr)
	if err != nil {
		return nil, err
	}

	if len(chartImageData) <= 32 {
		return nil, errors.New("invalid image data")
	}

	return chartImageData, err
}

func (s *Highcharts) GetReportImage(ctx context.Context, reportID, customerID string, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) (string, error) {
	chartImageData, err := s.GetReportImageData(ctx, reportID, customerID, highchartsFontSettings)
	if err != nil {
		return "", err
	}

	imageURL, err := s.SaveImageToGCS(ctx, chartImageData, reportID, customerID, "reports")

	return imageURL, err
}

func (s *Highcharts) GetChartImage(ctx context.Context, hcr *domainHighCharts.HighchartsRequest) ([]byte, error) {
	l := s.loggerProvider(ctx)

	if hcr == nil {
		return nil, errors.New("invalid nil highcharts request")
	}

	if common.IsLocalhost {
		jsonBytesIndent, err := json.MarshalIndent(hcr, "", "    ")
		if err != nil {
			log.Fatalf("Error marshalling to pretty JSON: %v", err)
		}

		l.Infoln("dump request to highcharts export server")
		l.Infoln(string(jsonBytesIndent))
	}

	r := httpClient.Request{
		Payload:       hcr,
		SkipUnmarshal: true,
	}

	res, err := s.highchartsExportClient.Post(ctx, &r)
	if err != nil {
		return nil, fmt.Errorf("highcharts request failed with error: %s", err)
	}

	if res.StatusCode == http.StatusOK {
		return res.Body, nil
	}

	return nil, fmt.Errorf("failed to get chart image with status code %d", res.StatusCode)
}

func (s *Highcharts) SaveImageToGCS(ctx context.Context, imageData []byte, chartID, customerID, chartType string) (string, error) {
	gcs := s.conn.CloudStorage(ctx)

	fileName, err := generateFileName(chartID)
	if err != nil {
		return "", err
	}

	bucketName := common.GetStaticEphemeralBucket()
	bkt := gcs.Bucket(bucketName)
	gcsPath := "cloud-analytics/" + chartType + "/" + customerID + "/" + chartID + "/" + fileName
	obj := bkt.Object(gcsPath)
	wc := obj.NewWriter(ctx)
	wc.ContentType = "image/png"

	if _, err := wc.Write(imageData); err != nil {
		return "", err
	}

	if err := wc.Close(); err != nil {
		return "", err
	}

	acl := obj.ACL()
	if err := acl.Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return "", err
	}

	imagePath := "https://storage.googleapis.com/" + bucketName + "/" + gcsPath

	return imagePath, nil
}

func generateFileName(chartID string) (string, error) {
	tempSecret := "this is secret"
	currentDateTime := time.Now().Format(time.RFC3339)

	randomID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	msg := chartID + currentDateTime + randomID.String()

	fileName, err := common.Sha256HMAC(msg, []byte(tempSecret))
	if err != nil {
		return "", err
	}

	return fileName, nil
}

// Check for flag set on customer that signals their treemap reports should use exact proportions
func (s *Highcharts) isExactTreemapCheck(ctx context.Context, customerID string) (bool, error) {
	fs := s.conn.Firestore(ctx)

	customerDocSnap, err := fs.Collection("customers").Doc(customerID).Get(ctx)
	if err != nil {
		return false, err
	}

	var customer struct {
		TreemapRenderConfig string `json:"treemapRenderConfig"`
	}

	// Don't return error if marshalling fails
	if err := customerDocSnap.DataTo(&customer); err != nil {
		return false, nil
	}

	return customer.TreemapRenderConfig == "exact", nil
}
