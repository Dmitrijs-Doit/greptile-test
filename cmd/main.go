package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"cloud.google.com/go/profiler"

	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cmd/api"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	defaultAddr = "0.0.0.0:8082"
)

func main() {
	if err := run(); err != nil {
		log.Println("error: ", err)
		os.Exit(1)
	}
}

func run() error {
	// Profiler initialization, best done as early as possible.
	if common.Production {
		if err := profiler.Start(profiler.Config{
			// Service and ServiceVersion can be automatically inferred when running
			// on App Engine.
			// ProjectID must be set if not running on GCP.
		}); err != nil {
			log.Printf("main: could not start profiler: %v", err)
		}
	}

	// Initialize basic context
	ctx := context.Background()

	// Initialize app engine logging clients
	logging, err := logger.NewLogging(ctx)
	if err != nil {
		log.Printf("main: could not initialize logging. error %s", err)
		return err
	}

	// Initialize db connections clients
	var bqProjects []string

	// Cloud analytics per-project bq clients.
	bqProjects = append(bqProjects, domainOrigin.CloudAnalyticsBQProjects()...)

	conn, err := connection.NewConnection(ctx, logging, bqProjects...)
	if err != nil {
		log.Printf("main: could not initialize db connections. error %s", err)
		return err
	}

	// =================
	// Start API Service
	log.Print("started: initializing api support")

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Inject needed functionality into the api.
	a := api.NewAPI(shutdown, logging, conn)

	addr, err := getAddr()
	if err != nil {
		log.Println(err)
		return err
	}

	server := http.Server{
		Addr:    addr,
		Handler: a.Build(),
	}

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		log.Printf("listening on %s", addr)
		serverErrors <- server.ListenAndServe()
	}()

	// =================
	// Shutdown

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return fmt.Errorf("%s : starting server", err)

	case sig := <-shutdown:
		log.Printf("%v : start shutdown", sig)

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Asking listener to shutdown and load shed.
		err := server.Shutdown(ctx)
		if err != nil {
			log.Printf("main : graceful shutdown did not complete")

			err = server.Close()
		}

		// Log the status of this shutdown.
		switch {
		case sig == syscall.SIGSTOP:
			return errors.New("integrity issue caused shutdown")
		case err != nil:
			return fmt.Errorf("could not stop server gracefully: %s", err)
		}
	}

	return nil
}

func getAddr() (string, error) {
	port := os.Getenv("PORT")
	if port == "" {
		return defaultAddr, nil
	}

	return fmt.Sprintf(":%s", port), nil
}
