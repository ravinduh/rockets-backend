package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"rockets-backend/database"
	"rockets-backend/pkg"
	"rockets-backend/repository"
	"rockets-backend/service"
	"rockets-backend/transport"
	"rockets-backend/transport/http_transport"
	"rockets-backend/worker"
	"syscall"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func main() {
	httpAddr := pkg.GetEnv("HTTP_PORT", ":8088")
	logLevel := pkg.GetEnv("LOG_LEVEL", "debug")

	logger := getLogger(logLevel)

	// Initialize database connection
	db, err := database.NewConnection()
	if err != nil {
		_ = level.Error(logger).Log("error", "failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize repository and service
	rocketRepository := repository.NewPostgresRocketRepository(db)
	svc := service.NewService(logger, rocketRepository)
	endpoints := transport.MakeEndpoints(svc)
	h := http_transport.NewHttpService(endpoints)
	server := &http.Server{
		Addr:    httpAddr,
		Handler: h,
	}

	_ = level.Info(logger).Log("msg", "rockets backend starting", "addr", httpAddr)
	// Initialize background workers and server
	eventProcessor := initializeWorkers(svc, rocketRepository, logger)
	startServer(server, logger)

	gracefulShutdown(server, logger, eventProcessor)
}

func getLogger(logLevel string) log.Logger {
	// creating a new structured logger
	logger := log.NewLogfmtLogger(os.Stdout)
	// making it safe for concurrent use
	logger = log.NewSyncLogger(logger)
	switch logLevel {
	case "error":
		logger = level.NewFilter(logger, level.AllowError())
	case "info":
		logger = level.NewFilter(logger, level.AllowInfo())
	default:
		logger = level.NewFilter(logger, level.AllowDebug())
	}
	logger = log.With(logger,
		"Service", "rockets-backend",
		"TimeStamp", log.DefaultTimestampUTC,
		"Caller", log.DefaultCaller,
	)
	return logger
}

func startServer(server *http.Server, logger log.Logger) {
	go func() {
		_ = level.Info(logger).Log("Transport", "HTTP", "Addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			_ = level.Error(logger).Log("Error", "server failed to start", "err", err)
		}
	}()
}

func initializeWorkers(svc service.Service, repo repository.RocketRepository, logger log.Logger) *worker.EventProcessor {
	workerConfig := worker.DefaultConfig()
	eventProcessor := worker.NewEventProcessor(svc, repo, logger, workerConfig)

	// Start background worker
	ctx := context.Background()
	err := eventProcessor.Start(ctx)
	if err != nil {
		_ = level.Error(logger).Log("error", "failed to start event processor", "err", err)
		os.Exit(1)
	}

	return eventProcessor
}

func gracefulShutdown(server *http.Server, logger log.Logger, eventProcessor *worker.EventProcessor) {
	// Wait for interrupt signal to gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	_ = level.Info(logger).Log("Message", "shutting down server gracefully")

	// Stopping the background worker first
	if err := eventProcessor.Stop(); err != nil {
		_ = level.Error(logger).Log("Error", "failed to stop event processor", "err", err)
	} else {
		_ = level.Info(logger).Log("Message", "event processor stopped gracefully")
	}

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// graceful shutdown of HTTP server
	if err := server.Shutdown(ctx); err != nil {
		_ = level.Error(logger).Log("Error", "server forced to shutdown", "err", err)
	} else {
		_ = level.Info(logger).Log("Message", "server exited gracefully")
	}
}
