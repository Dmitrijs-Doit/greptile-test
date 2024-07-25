module github.com/doitintl/hello/scheduled-tasks

go 1.21

require (
	cloud.google.com/go v0.115.0
	cloud.google.com/go/bigquery v1.61.0
	cloud.google.com/go/billing v1.18.7
	cloud.google.com/go/channel v1.17.10
	cloud.google.com/go/cloudtasks v1.12.10
	cloud.google.com/go/errorreporting v0.3.1
	cloud.google.com/go/firestore v1.15.0
	cloud.google.com/go/iam v1.1.10
	cloud.google.com/go/kms v1.18.2
	cloud.google.com/go/logging v1.10.0
	cloud.google.com/go/profiler v0.4.1
	cloud.google.com/go/pubsub v1.40.0
	cloud.google.com/go/scheduler v1.10.11
	cloud.google.com/go/secretmanager v1.13.3
	cloud.google.com/go/storage v1.43.0
	firebase.google.com/go/v4 v4.14.1
	github.com/1Password/connect-sdk-go v1.5.3
	github.com/GoogleCloudPlatform/protoc-gen-bq-schema v0.0.0-20240322180233-31a7e43419f7
	github.com/algolia/algoliasearch-client-go/v3 v3.31.2
	github.com/aws/aws-sdk-go v1.54.11
	github.com/chewxy/stl v1.3.1
	github.com/doitintl/auth v0.0.0
	github.com/doitintl/aws v0.0.0
	github.com/doitintl/azure v0.0.0
	github.com/doitintl/bigquery v0.0.0
	github.com/doitintl/bq-lens-proxy v0.0.0
	github.com/doitintl/buffer v0.0.0
	github.com/doitintl/cloudlogging v0.0.0
	github.com/doitintl/cloudresourcemanager v0.0.0
	github.com/doitintl/cloudtasks v0.0.0
	github.com/doitintl/concedefy v0.0.0
	github.com/doitintl/customerapi v0.0.0
	github.com/doitintl/errors v0.0.0
	github.com/doitintl/firestore v0.0.0
	github.com/doitintl/gcs v0.0.0
	github.com/doitintl/googleadmin v0.0.0
	github.com/doitintl/http v0.0.0
	github.com/doitintl/idtoken v0.0.0
	github.com/doitintl/insights/sdk v0.0.0
	github.com/doitintl/jira v0.0.0
	github.com/doitintl/mixpanel v0.0.0
	github.com/doitintl/notificationcenter v0.0.0
	github.com/doitintl/onepassword v0.0.0
	github.com/doitintl/pubsub v0.0.0
	github.com/doitintl/retry v0.0.0
	github.com/doitintl/rippling v0.0.0
	github.com/doitintl/secretmanager v0.0.0
	github.com/doitintl/serviceusage v0.0.0
	github.com/doitintl/slackapi v0.0.0
	github.com/doitintl/tests v0.0.0
	github.com/doitintl/tiers v0.0.0-00010101000000-000000000000
	github.com/doitintl/validator v0.0.0
	github.com/doitintl/workerpool v0.0.0
	github.com/getsentry/sentry-go v0.28.1
	github.com/gin-gonic/gin v1.10.0
	github.com/go-playground/validator v9.31.0+incompatible
	github.com/go-playground/validator/v10 v10.22.0
	github.com/go-resty/resty/v2 v2.13.1
	github.com/goccy/go-json v0.10.3
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/gomarkdown/markdown v0.0.0-20240626202925-2eda941fd024
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jpillora/go-tld v1.2.1
	github.com/linkedin/goavro/v2 v2.13.0
	github.com/qmuntal/stateless v1.7.0
	github.com/sendgrid/sendgrid-go v3.14.0+incompatible
	github.com/slack-go/slack v0.13.0
	github.com/sosodev/duration v1.3.1
	github.com/stretchr/testify v1.9.0
	github.com/stripe/stripe-go/v74 v74.30.0
	github.com/trycourier/courier-go/v3 v3.0.10
	github.com/zeebo/assert v1.3.1
	golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8
	golang.org/x/net v0.27.0
	golang.org/x/oauth2 v0.21.0
	golang.org/x/sync v0.7.0
	golang.org/x/term v0.22.0
	golang.org/x/text v0.16.0
	golang.org/x/time v0.5.0
	gonum.org/v1/gonum v0.15.0
	google.golang.org/api v0.188.0
	google.golang.org/genproto v0.0.0-20240708141625-4ad9e859172b
	google.golang.org/genproto/googleapis/api v0.0.0-20240701130421-f6361c86f094
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
	gotest.tools v2.2.0+incompatible
)

require (
	cloud.google.com/go/auth v0.7.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.2 // indirect
	cloud.google.com/go/compute/metadata v0.4.0 // indirect
	cloud.google.com/go/longrunning v0.5.9 // indirect
	cloud.google.com/go/trace v1.10.9 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.11.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.7.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.8.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage v1.5.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.3.2 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.2.2 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.22.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v1.22.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.46.0 // indirect
	github.com/MicahParks/keyfunc v1.9.0 // indirect
	github.com/apache/arrow/go/arrow v0.0.0-20211112161151-bc219186db40 // indirect
	github.com/apache/arrow/go/v15 v15.0.2 // indirect
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/chewxy/hm v1.0.0 // indirect
	github.com/chewxy/math32 v1.10.1 // indirect
	github.com/chewxy/tightywhities v1.0.0 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/doitintl/logger v0.0.0 // indirect
	github.com/doitintl/ratelimit v0.0.0 // indirect
	github.com/doitintl/reflect v0.0.0 // indirect
	github.com/doitintl/tracer v0.0.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v24.3.25+incompatible // indirect
	github.com/google/pprof v0.0.0-20240528025155-186aa0362fba // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.5 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.7 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.5 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/jwx/v2 v2.1.0 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/sendgrid/rest v2.6.9+incompatible // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/uber/jaeger-client-go v2.30.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/xtgo/set v1.0.0 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/sdk v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20231121144256-b99613f794b6 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/crypto v0.25.0 // indirect
	golang.org/x/mod v0.18.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/tools v0.22.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/appengine/v2 v2.0.5 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240708141625-4ad9e859172b // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorgonia.org/dawson v1.2.0 // indirect
	gorgonia.org/tensor v0.9.24 // indirect
	gorgonia.org/vecf32 v0.9.0 // indirect
	gorgonia.org/vecf64 v0.9.0 // indirect
)

// See https://github.com/getsentry/sentry-go/issues/376
replace github.com/labstack/echo/v4 v4.1.11 => github.com/labstack/echo/v4 v4.6.1

replace github.com/ugorji/go => github.com/ugorji/go v1.2.12

replace github.com/doitintl/http v0.0.0 => ../../shared/http

replace github.com/doitintl/logger v0.0.0 => ../../shared/logger

replace github.com/doitintl/tracer v0.0.0 => ../../shared/tracer

replace github.com/doitintl/auth v0.0.0 => ../../shared/auth

replace github.com/doitintl/errors v0.0.0 => ../../shared/errors

replace github.com/doitintl/tests v0.0.0 => ../../shared/tests

replace github.com/doitintl/firestore v0.0.0 => ../../shared/firestore

replace github.com/doitintl/cloudtasks v0.0.0 => ../../shared/cloudtasks

replace github.com/doitintl/gcs v0.0.0 => ../../shared/gcs

replace github.com/doitintl/bigquery v0.0.0 => ../../shared/bigquery

replace github.com/doitintl/pubsub v0.0.0 => ../../shared/pubsub

replace github.com/doitintl/retry v0.0.0 => ../../shared/retry

replace github.com/doitintl/serviceusage v0.0.0 => ../../shared/serviceusage

replace github.com/doitintl/cloudresourcemanager v0.0.0 => ../../shared/cloudresourcemanager

replace github.com/doitintl/cloudlogging v0.0.0 => ../../shared/cloudlogging

replace github.com/doitintl/idtoken v0.0.0 => ../../shared/idtoken

replace github.com/doitintl/workerpool v0.0.0 => ../../shared/workerpool

replace github.com/doitintl/concedefy v0.0.0 => ../../shared/concedefy

replace github.com/doitintl/notificationcenter v0.0.0 => ../../shared/notificationcenter

replace github.com/doitintl/ratelimit v0.0.0 => ../../shared/ratelimit

replace github.com/doitintl/onepassword v0.0.0 => ../../shared/onepassword

replace github.com/doitintl/googleadmin v0.0.0 => ../../shared/googleadmin

replace github.com/doitintl/secretmanager v0.0.0 => ../../shared/secretmanager

replace github.com/doitintl/validator v0.0.0 => ../../shared/validator

replace github.com/doitintl/rippling v0.0.0 => ../../shared/rippling

replace github.com/doitintl/reflect v0.0.0 => ../../shared/reflect

replace github.com/doitintl/slackapi v0.0.0 => ../../shared/slackapi

replace github.com/doitintl/mixpanel v0.0.0 => ../../shared/mixpanel

replace github.com/doitintl/customerapi v0.0.0 => ../../shared/customerapi

replace github.com/doitintl/aws => ../../shared/aws

replace github.com/doitintl/tiers => ../../shared/tiers

replace github.com/doitintl/azure v0.0.0 => ../../shared/azure

replace github.com/doitintl/insights/sdk v0.0.0 => ../../../services/insights/sdk

replace github.com/doitintl/jira v0.0.0 => ../../shared/jira

replace github.com/doitintl/buffer v0.0.0 => ../../shared/buffer

replace github.com/doitintl/bq-lens-proxy v0.0.0 => ../../../services/bq-lens-proxy
