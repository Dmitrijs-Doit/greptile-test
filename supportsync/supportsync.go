package supportsync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/http"
)

type objectData struct {
	Platform string
	Version  int64
	Services []*common.Service
}

const (
	tags        string = "tags"
	platformCRE string = "cre"
	gcsBucket   string = "cloud_tags"
	gcsBaseURL  string = "https://storage.googleapis.com"
	schema      string = "schema"

	// errors
	errorData   string = "error getting data"
	errorParse  string = "parsing error"
	errorAppDAL string = "App DAL error"
	errorGCS    string = "Google Cloud Storage client error"
	errorHTTP   string = "HTTP client error"
)

// Sync is a job to sync CRE services to firestore
func (s *SupportSyncService) Sync(ctx context.Context) error {
	lastUpdate := time.Now()
	objectsIter := s.gcsClient.GetObjectIterator(ctx, gcsBucket)

	for {
		objectData, err := s.extractObjectData(ctx, objectsIter)
		if err != nil {
			return s.formatError(err, "extractObjectData()")
		}

		if iteratorDone := (objectData.Platform == "" && objectData.Version == 0 && objectData.Services == nil); iteratorDone {
			break
		}

		if objectData.Platform == tags || objectData.Platform == platformCRE || objectData.Platform == schema {
			continue
		}

		prevVersion, err := s.appDAL.GetServicesPlatformVersion(ctx, objectData.Platform)
		if err != nil {
			return s.formatError(err, errorAppDAL)
		}

		if prevVersion < objectData.Version {
			if err := s.updateServices(ctx, objectData.Version, lastUpdate, objectData.Services, objectData.Platform); err != nil {
				return s.formatError(err, "updateServices()")
			}
		}
	}

	return nil
}

func (s *SupportSyncService) updateServices(ctx context.Context, version int64, lastUpdate time.Time, services []*common.Service, platform string) error {
	logger := s.loggerProvider(ctx)

	for _, service := range services {
		service.Version = version
		service.LastUpdate = lastUpdate
		service.Platform = platform
	}

	if err := s.appDAL.UpdateServices(ctx, lastUpdate, services); err != nil {
		return s.formatError(err, errorAppDAL)
	}

	deletedDocs, err := s.appDAL.CleanOutdatedServices(ctx, platform, version)
	logger.Printf("updated %d services for platform %s version %d, %d services deleted", len(services), platform, version, deletedDocs)

	if err != nil {
		return s.formatError(err, errorAppDAL)
	}

	return nil
}

// extractObjectData extract object's platform, version, body (to be parsed)
func (s *SupportSyncService) extractObjectData(ctx context.Context, objectsIter *storage.ObjectIterator) (*objectData, error) {
	obj, err := objectsIter.Next()

	objectData := &objectData{}
	if err == iterator.Done {
		return objectData, nil // break outer scope loop
	}

	if err != nil {
		return objectData, s.formatError(err, errorGCS)
	}

	objectData.Version = obj.Generation

	objectData.Platform = s.getPlatform(obj.Name)
	if objectData.Platform == tags || objectData.Platform == schema {
		return objectData, nil
	}

	url := strings.Replace(obj.MediaLink, gcsBaseURL, "", -1)

	request := &http.Request{
		URL:          url,
		ResponseType: &objectData.Services,
	}
	if _, err := s.httpClient.Get(ctx, request); err != nil {
		return objectData, s.formatError(err, errorHTTP)
	}

	return objectData, nil
}

func (s *SupportSyncService) formatError(err error, prefix string) error {
	return fmt.Errorf("%s: %s", prefix, err.Error())
}

func (s *SupportSyncService) getPlatform(fileName string) string {
	abbreviated := strings.Replace(fileName, ".json", "", -1)

	switch abbreviated {
	case "aws":
		return common.Assets.AmazonWebServices
	case "azure":
		return common.Assets.MicrosoftAzure
	case "cmp":
		return common.CMP
	case "gcp", "gcp-firebase":
		return common.Assets.GoogleCloud
	case "gsuite":
		return common.Assets.GSuite
	case "ms365":
		return common.Assets.Office365
	default:
		if strings.Contains(abbreviated, platformCRE) {
			return platformCRE
		}

		return abbreviated
	}
}
