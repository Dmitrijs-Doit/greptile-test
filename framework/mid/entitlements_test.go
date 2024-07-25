package mid

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/auth"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/tiers/service"
	"github.com/doitintl/tiers/service/mocks"
)

func GetContext() (*gin.Context, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(http.MethodPost, "http://example.com/foo", nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	ctx.Set("email", "test@email.com")
	ctx.Set("doitEmployee", false)
	ctx.Set("userId", "user123")
	ctx.Set(common.CtxKeys.CustomerID, "customer123")

	return ctx, recorder
}

func TestEntitlementMiddleware_InvalidEntitlement(t *testing.T) {
	testHandler := func(ctx *gin.Context) error { return nil }
	ctx, _ := GetContext()

	tierSvc := mocks.NewTierServiceIface(t)
	tierSvc.On("CustomerCanAccessFeature", mock.Anything, mock.Anything, mock.Anything).Return(false, errors.New("invalid entitlement"))
	hasEntitlement := HasEntitlementFunc(tierSvc)

	mw := hasEntitlement("")

	err := mw(testHandler)(ctx)

	if assert.Error(t, err) {
		var webErr *web.Error
		if assert.ErrorAs(t, err, &webErr) {
			assert.ErrorIs(t, webErr.Err, service.ErrFeatureNotAccessible)
			assert.Equal(t, http.StatusForbidden, webErr.Status)
		}
	}
}

func TestEntitlementMiddleware_NoEntitlmentFound(t *testing.T) {
	testHandler := func(ctx *gin.Context) error { return nil }
	ctx, _ := GetContext()

	tierSvc := mocks.NewTierServiceIface(t)
	tierSvc.On("CustomerCanAccessFeature", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
	hasEntitlement := HasEntitlementFunc(tierSvc)

	mw := hasEntitlement(pkg.TiersFeatureKeyAlerts)

	err := mw(testHandler)(ctx)

	if assert.Error(t, err) {
		var webErr *web.Error
		if assert.ErrorAs(t, err, &webErr) {
			assert.ErrorIs(t, webErr.Err, service.ErrFeatureNotAccessible)
			assert.Equal(t, http.StatusForbidden, webErr.Status)
		}
	}
}

func TestEntitlementMiddleware_FeatureFound(t *testing.T) {
	testHandler := func(ctx *gin.Context) error { ctx.String(http.StatusOK, "%s", "success"); return nil }
	ctx, recorder := GetContext()

	tierSvc := mocks.NewTierServiceIface(t)
	tierSvc.On("CustomerCanAccessFeature", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
	hasEntitlement := HasEntitlementFunc(tierSvc)

	mw := hasEntitlement(pkg.TiersFeatureKeyAlerts)

	err := mw(testHandler)(ctx)

	assert.Nil(t, err, "Err should be nil")
	assert.Equal(t, http.StatusOK, ctx.Writer.Status())
	assert.Equal(t, "success", recorder.Body.String())
}

func TestEntitlementMiddleware_DoerShouldNotBeChecked(t *testing.T) {
	testHandler := func(ctx *gin.Context) error { ctx.String(http.StatusOK, "%s", "success"); return nil }
	ctx, recorder := GetContext()
	ctx.Set("doitEmployee", true)

	tierSvc := mocks.NewTierServiceIface(t)
	hasEntitlement := HasEntitlementFunc(tierSvc)

	mw := hasEntitlement(pkg.TiersFeatureKeyAlerts)

	err := mw(testHandler)(ctx)

	assert.Nil(t, err, "Err should be nil")
	assert.Equal(t, http.StatusOK, ctx.Writer.Status())
	assert.Equal(t, "success", recorder.Body.String())
	tierSvc.AssertNotCalled(t, "CustomerCanAccessFeature")
}

func TestEntitlementMiddleware_WorksWithExternalAPI(t *testing.T) {
	testHandler := func(ctx *gin.Context) error { ctx.String(http.StatusOK, "%s", "success"); return nil }
	ctx, recorder := GetContext()
	ctx.Set("doitEmployee", false)
	ctx.Set(auth.CtxKeyVerifiedCustomerID, "customer-1234")
	ctx.Set(auth.CtxKeyExternalAPI, true)

	customerID := "customer-1234"

	tierSvc := mocks.NewTierServiceIface(t)
	tierSvc.On("CustomerCanAccessFeature", mock.Anything, customerID, mock.Anything).Return(true, nil)
	hasEntitlement := HasEntitlementFunc(tierSvc)

	mw := hasEntitlement(pkg.TiersFeatureKeyAlerts)

	err := mw(testHandler)(ctx)

	assert.Nil(t, err, "Err should be nil")
	assert.Equal(t, http.StatusOK, ctx.Writer.Status())
	assert.Equal(t, "success", recorder.Body.String())
}

func TestEntitlementMiddleware_ErrorWhenNoCustomerIDExists(t *testing.T) {
	testHandler := func(ctx *gin.Context) error { ctx.String(http.StatusOK, "%s", "success"); return nil }
	ctx, _ := GetContext()
	ctx.Set("doitEmployee", false)
	ctx.Set(common.CtxKeys.CustomerID, "")

	tierSvc := mocks.NewTierServiceIface(t)
	hasEntitlement := HasEntitlementFunc(tierSvc)

	mw := hasEntitlement(pkg.TiersFeatureKeyAlerts)

	err := mw(testHandler)(ctx)

	if assert.Error(t, err) {
		var webErr *web.Error
		if assert.ErrorAs(t, err, &webErr) {
			assert.Equal(t, http.StatusBadRequest, webErr.Status)
		}
	}
}
