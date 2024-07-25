package web

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/internal"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// A Handler is a type that handles a http request within our own mini
// framework.
type Handler func(ctx *gin.Context) error

// App is the entry-point into our application and what configures our context
// object for each of our http handlers.
type App struct {
	engine      *gin.Engine
	shutdown    chan os.Signal
	conn        *connection.Connection
	middlewares []Middleware
}

// NewApp creates an App value that handle a set of routes for the application.
func NewApp(shutdown chan os.Signal, conn *connection.Connection, mw ...Middleware) *App {
	sentryRelease := os.Getenv("GAE_VERSION")
	if sentryRelease != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              "https://d62da373815d480b9e13e363e26ed6d9@o926763.ingest.sentry.io/5890616",
			Release:          sentryRelease,
			Environment:      common.ProjectID,
			TracesSampleRate: 1.0,
			SampleRate:       1.0,
			Debug:            false,
			AttachStacktrace: true,
		}); err != nil {
			fmt.Printf("Sentry initialization failed: %v\n", err)
		} else {
			fmt.Printf("Sentry initialization, Release: %s, Environment: %s\n", sentryRelease, common.ProjectID)
		}
	} else {
		fmt.Printf("Sentry initialization skipped, no GAE_VERSION in env\n")
	}

	engine := gin.New()

	engine.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))

	app := App{
		engine:      engine,
		shutdown:    shutdown,
		conn:        conn,
		middlewares: mw,
	}

	return &app
}

// SignalShutdown is used to gracefully shutdown the app when an integrity
// issue is identified.
func (a *App) SignalShutdown() {
	a.shutdown <- syscall.SIGSTOP
}

// Handle is our mechanism for mounting Handlers for a given HTTP verb and path
// pair, this makes for really easy, convenient routing.
func (a *App) Handle(verb, path string, handler Handler, mw ...Middleware) {
	// printing mapping details for handlers
	if gin.Mode() != gin.ReleaseMode {
		gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, _ int) {
			handlerName = runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
			log.Printf("[debug] %-6s %-40s --> %s \n", strings.ToLower(httpMethod), absolutePath, handlerName)
		}
	}

	wrappedHandler := wrapMiddleware(mw, handler)
	wrappedHandler = wrapMiddleware(a.middlewares, wrappedHandler)

	h := func(ctx *gin.Context) {
		log, err := logger.NewLogger(ctx)
		if err != nil {
			a.SignalShutdown()
			return
		}

		defer log.End(ctx)

		v := internal.Data{
			TraceID: log.Trace(),
			Now:     time.Now(),
		}
		internal.ContextWithData(ctx, &v)

		if gin.Mode() != gin.TestMode {
			a.conn.FirestoreWithContext(ctx)
		}

		// Call the wrapped handler functions.
		if err := wrappedHandler(ctx); err != nil {
			log.Printf("*****> critical shutdown error: %s", err)
			a.SignalShutdown()

			return
		}
	}
	// Add this handler for the specified verb and route.
	a.engine.Handle(verb, path, h)
}

// Post executes Handle with http method POST.
func (a *App) Post(path string, handler Handler, mw ...Middleware) {
	a.Handle(http.MethodPost, path, handler, mw...)
}

// Get executes Handle with http method GET.
func (a *App) Get(path string, handler Handler, mw ...Middleware) {
	a.Handle(http.MethodGet, path, handler, mw...)
}

// Put executes Handle with http method PUT.
func (a *App) Put(path string, handler Handler, mw ...Middleware) {
	a.Handle(http.MethodPut, path, handler, mw...)
}

// Delete executes Handle with http method DELETE.
func (a *App) Delete(path string, handler Handler, mw ...Middleware) {
	a.Handle(http.MethodDelete, path, handler, mw...)
}

// Patch executes Handle with http method PATCH.
func (a *App) Patch(path string, handler Handler, mw ...Middleware) {
	a.Handle(http.MethodPatch, path, handler, mw...)
}

// ServeHTTP implements the http.Handler interface.
// It overrides the ServeHTTP of the embedded gin.Engine.
// this Handler wraps the gin.Engine handler so the routes are served.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.engine.ServeHTTP(w, r)
}

// NewTestApp creates a new gin App used for handler testing.
func NewTestApp(responseRecorder http.ResponseWriter, mw ...Middleware) *App {
	gin.SetMode(gin.TestMode)
	engine := gin.Default()

	app := App{
		engine:      engine,
		shutdown:    nil,
		conn:        nil,
		middlewares: mw,
	}

	return &app
}
