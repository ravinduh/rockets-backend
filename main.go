package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"rockets-backend/service"
	"rockets-backend/transport"
	"rockets-backend/transport/http_transport"
	"syscall"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func main() {
	httpAddr := ":9114"
	logLevel := "debug"

	logger := getLogger(logLevel)

	svc := service.NewService(logger)
	endpoints := transport.MakeEndpoints(svc)

	h := http_transport.NewHttpService(endpoints)
	server := &http.Server{
		Addr:    httpAddr,
		Handler: h,
	}

	startServer(server, logger)
	gracefulShutdown(server, logger)
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

func gracefulShutdown(server *http.Server, logger log.Logger) {
	// Wait for interrupt signal to gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	_ = level.Info(logger).Log("Message", "shutting down server gracefully")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		_ = level.Error(logger).Log("Error", "server forced to shutdown", "err", err)
	} else {
		_ = level.Info(logger).Log("Message", "server exited gracefully")
	}
}
