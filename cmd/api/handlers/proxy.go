package handlers

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"github.com/doitintl/errors"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/mid/permissions/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

const ProxyPath = "path"

type proxyService struct {
	baseURL     string
	tokenSource oauth2.TokenSource
}

func NewProxyService(baseURL string, tokenSource oauth2.TokenSource) *proxyService {
	return &proxyService{
		baseURL,
		tokenSource,
	}
}

func (h *proxyService) Proxy(c *gin.Context) error {
	remote, err := url.Parse(h.baseURL)
	if err != nil {
		return err
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Director = func(req *http.Request) {
		req.Header = c.Request.Header
		req.Host = remote.Host
		req.URL.Scheme = remote.Scheme
		req.URL.Host = remote.Host
		req.URL.Path = c.Param(ProxyPath)

		token, err := h.tokenSource.Token()
		if err != nil {
			panic(err)
		}

		if token.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		}
	}

	proxy.ServeHTTP(c.Writer, c.Request)

	return nil
}

func MethodBasedPermissionProxyHandler(flexapiProxy *proxyService, conn *connection.Connection) web.Handler {
	return func(c *gin.Context) error {
		// flexapi GET methods do not require flexsave-admin role
		if c.Request.Method == http.MethodGet {
			return flexapiProxy.Proxy(c)
		}

		s := doitemployees.NewService(conn)
		email := c.GetString("email")

		admin, err := s.CheckDoiTEmployeeRole(c, string(domain.DoitRoleFlexsaveAdmin), email)
		if err != nil {
			return errors.Wrapf(err, "CheckDoiTEmployeeRole() failed for user '%s'", email)
		}

		if !admin {
			return errors.Errorf("no flexsave admin permission for user '%s'", email)
		}

		return flexapiProxy.Proxy(c)
	}
}
