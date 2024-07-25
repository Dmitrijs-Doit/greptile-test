package authorization

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/http"
	"github.com/doitintl/http/mocks"
)

func TestGetSalesforceJWToken(t *testing.T) {
	t.Skip()

	ctx := context.Background()

	var mockHTTP = &mocks.IClient{}

	service, _ := setupAuthorizationService(ctx, t, mockHTTP)

	mockHTTP.
		On("Post",
			mock.MatchedBy(func(_ context.Context) bool { return true }),
			mock.MatchedBy(func(req *http.Request) bool {

				assertion := req.QueryParams["assertion"][0]
				gType := req.QueryParams["grant_type"][0]
				assert.Equal(t, "urn:ietf:params:oauth:grant-type:jwt-bearer", gType)
				validateToken(t, assertion, service.(*authService).sfSecret)
				return req.URL == "/services/oauth2/token"
			})).
		Return(nil, nil).Once()

	_, err := service.Authenticate(ctx)

	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateHTTPClient(t *testing.T) {
	t.Skip()

	ctx := context.Background()

	service, _ := setupAuthorizationService(ctx, t, nil)

	assert.NotNil(t, service.(*authService).httpClient)
}

func setupAuthorizationService(ctx context.Context, t *testing.T, mockHTTP *mocks.IClient) (AuthorizationService, error) {
	log, err := logger.NewLogging(ctx)

	if err != nil {
		t.Error(err)
	}

	return NewAuthService(log, mockHTTP)
}

func validateToken(t *testing.T, tokenString string, sfSecret *Secret) {
	t.Skip()

	parsedPublicKey, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(sfSecret.PublicKey))
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			t.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return parsedPublicKey, nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		assert.True(t, claims.VerifyAudience("https://test.salesforce.com", true))
		assert.True(t, claims.VerifyIssuer(sfSecret.ConsumerKey, true))
		assert.Equal(t, sfSecret.Username, claims["sub"])
		assert.Nil(t, claims.Valid())
	}
}
