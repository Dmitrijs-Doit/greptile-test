package logger

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"github.com/gin-gonic/gin"
	"google.golang.org/genproto/googleapis/api/monitoredres"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	// CtxLoggerKey is how request values or stored/retrieved.
	CtxLoggerKey = "app-logger"

	// CtxDetailedLoggerKey is how request values or stored/retrieved.
	CtxDetailedLoggerKey = "app-_detailed-logger"

	// parentLogID is the name of the log file for parent logging.
	parentLogID = "parent_logger"

	// childLogID is the name of the log file for child logging.
	childLogID = "child_logger"

	// labels keys for monitored resource definition
	moduleIDField  = "module_id"
	projectIDField = "project_id"
	versionIDField = "version_id"

	// labels from env vars for monitored resource definition
	appEngineService = "GAE_SERVICE"
	appEngineVersion = "GAE_VERSION"
	appEngineType    = "gae_app"

	gcpLogging = "GCP_LOGGING"
)

var (
	parentLogger *logging.Logger
	childLogger  *logging.Logger
	resource     *monitoredres.MonitoredResource
	cloudLogging bool
)

type Provider func(ctx context.Context) ILogger

type Logging struct {
}

// log.SetFlags(log.LstdFlags | log.Lshortfile)
// NewLogging initializes parent & child google cloud engine logging clients.
func NewLogging(ctx context.Context) (*Logging, error) {
	client, err := logging.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	parentLogger = client.Logger(parentLogID)
	childLogger = client.Logger(childLogID)

	moduleID := common.GetEnv(appEngineService, "scheduled-tasks")
	versionID := common.GetEnv(appEngineVersion, "localhost")

	cloudLogging = true
	// disable cloud logging when running in localhost
	if common.IsLocalhost {
		cloudLogging = false
	}

	cloudLogging, err = strconv.ParseBool(common.GetEnv(gcpLogging, strconv.FormatBool(cloudLogging)))
	if err != nil {
		return nil, err
	}

	resource = &monitoredres.MonitoredResource{
		Labels: map[string]string{
			moduleIDField:  moduleID,
			projectIDField: common.ProjectID,
			versionIDField: versionID,
		},
		Type: appEngineType,
	}

	return &Logging{}, nil
}

// Logger returns the logger that was stored inside the context.
func (l *Logging) Logger(ctx context.Context) ILogger {
	return FromContext(ctx)
}

// NewLogger sets gin.Context with a new logger, with the related google trace id.
func NewLogger(ctx *gin.Context) (*Logger, error) {
	l := newDefaultLogger()
	d := newDetailedLogger()

	var h string
	if ctx.Request != nil {
		h = ctx.Request.Header.Get("X-Cloud-Trace-Context")
	}

	if h != "" {
		if i := strings.IndexByte(h, '/'); i > 0 {
			if t := h[:i]; strings.Count(t, "0") != len(t) {
				l.trace = getTrace(l.started, t)
			}
		}
	}

	ctx.Set(CtxLoggerKey, l)
	ctx.Set(CtxDetailedLoggerKey, d)

	return l, nil
}

// FromContext returns the logger that was stored in context.
// If there isn't logger stored, returns a new logger.
func FromContext(ctx context.Context) ILogger {
	if l, ok := ctx.Value(CtxLoggerKey).(*Logger); ok {
		return l
	}

	return newDefaultLogger()
}

func getTrace(started time.Time, id string) string {
	return fmt.Sprintf("projects/%s/traces/%d%s", common.ProjectID, started.UnixNano(), id)
}
