package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/salesforce/authorization/mocks"
	"github.com/doitintl/http"
	httpMocks "github.com/doitintl/http/mocks"
)

func TestCompositeRequest(t *testing.T) {
	ctx := context.Background()

	var mockHTTP = &httpMocks.IClient{}

	var mockAuth = mocks.NewAuthorizationService(t)

	var mockAuthorization = mocks.Authorization{}

	mockAuthorization.On("GetToken").Return("token")
	mockAuthorization.On("GetInstanceURL").Return("https://test.salesforce.com")

	mockAuth.On("Authenticate", mock.MatchedBy(func(_ context.Context) bool { return true })).Return(&mockAuthorization, nil)

	mockHTTP.
		On("Post",
			mock.MatchedBy(func(_ context.Context) bool { return true }),
			mock.MatchedBy(func(req *http.Request) bool {
				return req.URL == "/services/data/v55.0/composite"
			})).
		Return(nil, http.WebError{Err: errors.New("Session expired or invalid"), Code: 401}).Once().
		On("Post",
			mock.MatchedBy(func(_ context.Context) bool { return true }),
			mock.MatchedBy(func(req *http.Request) bool {
				return req.URL == "/services/data/v55.0/composite"
			})).
		Return(nil, nil).Once()

	service, _ := setupCompositeService(ctx, t, mockHTTP, mockAuth)
	_, err := service.CompositeRequest(ctx, CompositeRequest{
		AllOrNone:        false,
		CompositeRequest: nil,
	})

	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(mockHTTP.Calls), 2)

	for _, call := range mockHTTP.Calls {
		assert.Equal(t, "Post", call.Method)
		assert.Equal(t, compositeReqURL, call.Arguments[1].(*http.Request).URL)
	}

	assert.Equal(t, len(mockAuth.Calls), 2)
}

func setupCompositeService(ctx context.Context, t *testing.T, mockHTTP *httpMocks.IClient, authService *mocks.AuthorizationService) (*CompositeService, error) {
	log, err := logger.NewLogging(ctx)

	if err != nil {
		t.Error(err)
	}

	return NewCompositeService(log, authService, mockHTTP)
}
