package common

import (
	"context"
	"encoding/json"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

type AccessKey struct {
	key []byte
}

type IAccessStrategy interface {
	GetClientOption(context context.Context, scopes ...string) (option.ClientOption, error)
}

func NewAccessKeyStrategy(key []byte) *AccessKey {
	return &AccessKey{
		key: key,
	}
}
func (ak *AccessKey) GetClientOption(context context.Context, scopes ...string) (option.ClientOption, error) {
	byteContainer, err := DecryptSymmetric(ak.key)
	if err != nil {
		return nil, err
	}

	conf, err := google.JWTConfigFromJSON(byteContainer, scopes...)
	if err != nil {
		return nil, err
	}

	tokenSource := conf.TokenSource(context)

	return option.WithTokenSource(tokenSource), nil
}

type ICustomerCredentials interface {
	WithContext(ctx context.Context) *GcpCustomerAuth
	WithScopes(scopes ...string) *GcpCustomerAuth
	UseWorkloadIdentityFederationStrategy() *GcpCustomerAuth
	UseAccessKeyStrategy() *GcpCustomerAuth
	GetClientOption() (option.ClientOption, error)
}

type GcpCustomerAuth struct {
	scopes                 []string
	context                *context.Context
	strategy               IAccessStrategy
	googleCloudCredentials *GoogleCloudCredential
}

func NewGcpCustomerAuthService(googleCloudCredentials *GoogleCloudCredential) ICustomerCredentials {
	var strategy IAccessStrategy
	if googleCloudCredentials.WorkloadIdentityFederationStatus == CloudConnectStatusTypeHealthy {
		strategy = NewWorkloadIdentityFederationStrategy(googleCloudCredentials.ClientEmail)
	} else {
		strategy = NewAccessKeyStrategy(googleCloudCredentials.Key)
	}

	context := context.Background()

	return &GcpCustomerAuth{
		googleCloudCredentials: googleCloudCredentials,
		strategy:               strategy,
		context:                &context,
		scopes:                 []string{compute.CloudPlatformScope},
	}
}

func (cc *GcpCustomerAuth) WithContext(ctx context.Context) *GcpCustomerAuth {
	cc.context = &ctx
	return cc
}

func (cc *GcpCustomerAuth) WithScopes(scopes ...string) *GcpCustomerAuth {
	cc.scopes = scopes
	return cc
}

func (cc *GcpCustomerAuth) UseAccessKeyStrategy() *GcpCustomerAuth {
	cc.strategy = NewAccessKeyStrategy(cc.googleCloudCredentials.Key)
	return cc
}

func (cc *GcpCustomerAuth) UseWorkloadIdentityFederationStrategy() *GcpCustomerAuth {
	cc.strategy = NewWorkloadIdentityFederationStrategy(cc.googleCloudCredentials.ClientEmail)
	return cc
}

func (cc *GcpCustomerAuth) GetClientOption() (option.ClientOption, error) {
	return cc.strategy.GetClientOption(*cc.context, cc.scopes...)
}

type WorkloadIdentityFederation struct {
	targetPrincipal string
	delegates       []string
}

func NewWorkloadIdentityFederationStrategy(targetPrincipal string) *WorkloadIdentityFederation {
	var delegates []string
	if Production {
		delegates = []string{"doit-connect@me-doit-intl-com.iam.gserviceaccount.com"}
	} else {
		delegates = []string{"doit-connect@doitintl-cmp-dev.iam.gserviceaccount.com"}
	}

	return &WorkloadIdentityFederation{
		targetPrincipal: targetPrincipal,
		delegates:       delegates,
	}
}

func (slt *WorkloadIdentityFederation) GetClientOption(context context.Context, scopes ...string) (option.ClientOption, error) {
	ts, err := impersonate.CredentialsTokenSource(context, impersonate.CredentialsConfig{
		TargetPrincipal: slt.targetPrincipal,
		Scopes:          scopes,
		Delegates:       slt.delegates,
	})
	if err != nil {
		return nil, err
	}

	return option.WithTokenSource(ts), nil
}

type ErrorDetails struct {
	Error ErrorMessage `json:"error"`
}

type ErrorMessage struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

type WorkloadIdentityFederationConnectionStatus struct {
	IsConnectionEstablished bool
	ConnectionDetails       *ErrorDetails
}

func (slt *WorkloadIdentityFederation) IsConnectionEstablished(ctx context.Context) (*WorkloadIdentityFederationConnectionStatus, error) {
	connectionNotEstablished := &WorkloadIdentityFederationConnectionStatus{
		IsConnectionEstablished: false,
		ConnectionDetails:       nil,
	}
	ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
		TargetPrincipal: slt.targetPrincipal,
		Scopes:          []string{compute.CloudPlatformScope},
		Delegates:       slt.delegates,
	})

	if err != nil {
		return connectionNotEstablished, err
	}

	_, tokenRetrieveError := ts.Token()
	tokenRetrieveErrorDetails := new(ErrorDetails)

	if tokenRetrieveError != nil {
		i := strings.Index(tokenRetrieveError.Error(), "{")

		err = json.Unmarshal([]byte(tokenRetrieveError.Error()[i:]), tokenRetrieveErrorDetails)
		if err != nil {
			return connectionNotEstablished, err
		}
	}

	return &WorkloadIdentityFederationConnectionStatus{
		IsConnectionEstablished: tokenRetrieveError == nil,
		ConnectionDetails:       tokenRetrieveErrorDetails,
	}, nil
}
