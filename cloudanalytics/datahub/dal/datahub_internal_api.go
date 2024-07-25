package dal

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
	"github.com/doitintl/http"
)

const ingestEventsUrl = "/datahub/v1/internal/events"

type DatahubInternalAPIDAL struct {
	Client http.IClient
}

func NewDatahubInternalAPIDAL(
	client http.IClient,
) (*DatahubInternalAPIDAL, error) {
	return &DatahubInternalAPIDAL{
		client,
	}, nil
}

func (s *DatahubInternalAPIDAL) IngestEvents(
	ctx context.Context,
	req domain.IngestEventsInternalReq,
) (*http.Response, error) {
	resp, err := s.Client.Post(ctx, &http.Request{
		URL:     ingestEventsUrl,
		Payload: req,
	})

	return resp, err
}
