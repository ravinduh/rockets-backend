package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"rockets-backend/service"
	"rockets-backend/transport"
	"rockets-backend/transport/http_transport"
	"syscall"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func main() {
	httpAddr := ":9114"
	logLevel := "debug"

	logger := getLogger(logLevel)

	svc := service.NewService(logger)
	endpoints := transport.MakeEndpoints(svc)

	// create a error channel, which can be used to stop the application in proper manner
	// otherwise port will not get free in local
	errChan := make(chan error)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
		_ = level.Info(logger).Log("Message", "stopping the rockets backend", "ErrChan", <-errChan)
	}()

	// Start the server listener
	go func() {
		h := http_transport.NewHttpService(endpoints)
		_ = level.Info(logger).Log("Transport", "HTTP", "Addr", httpAddr)
		server := &http.Server{
			Addr:    httpAddr,
			Handler: h,
		}

		errChan <- server.ListenAndServe()
	}()

	_ = level.Error(logger).Log("Error", <-errChan)
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
