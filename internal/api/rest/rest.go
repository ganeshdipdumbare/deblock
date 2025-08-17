package rest

import (
	"context"
	"deblock/internal/txmonitor"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
	nilArgErr   = "nil %v not allowed"
	emptyArgErr = "empty %v not allowed"
)

// @title Deblock transaction monitor API
// @version 1.0
// @description This is a Deblock service API for monitoring blockchain transactions
// @description
// @description Endpoints:
// @description - POST /txmonitor/start: Start monitoring blockchain transactions
// @description - POST /txmonitor/stop: Stop monitoring blockchain transactions
// @description - GET /health: Check service health
// @termsOfService http://swagger.io/terms/

// @contact.name Ganesh Dipdumbare
// @contact.url https://github.com/ganeshdipdumbare
// @contact.email ganeshdip.dumbare@gmail.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// RestApi defines methods to handle rest server
type RestApi interface {
	StartServer()
	GracefulStopServer()
}

type apiDetails struct {
	logger     *slog.Logger
	server     *http.Server
	service    txmonitor.TxMonitorService
	serverPort string
}

// NewApi creates new api instance, otherwise returns error
func NewApi(logger *slog.Logger, port string, service txmonitor.TxMonitorService) (RestApi, error) {
	if logger == nil {
		return nil, fmt.Errorf(nilArgErr, "logger")
	}

	if port == "" {
		return nil, fmt.Errorf(emptyArgErr, "port")
	}

	if service == nil {
		return nil, fmt.Errorf(nilArgErr, "transaction monitor service")
	}

	api := &apiDetails{
		logger:     logger,
		service:    service,
		serverPort: port,
	}

	router := api.setupRouter()
	api.server = &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%v", port),
		Handler: router,
	}

	// Add Swagger documentation route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return api, nil
}

// StartServer starts the rest server
// it listens for a kill signal to stop the server gracefully
func (api *apiDetails) StartServer() {
	// Ensure correct server address format
	serverAddr := api.serverPort
	if !strings.Contains(serverAddr, ":") {
		serverAddr = ":" + serverAddr
	}

	// Create server
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: api.setupRouter(),
	}

	// Create channel for server errors
	serverErrChan := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		api.logger.Info("Starting server",
			"address", serverAddr,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- fmt.Errorf("server listen error: %w", err)
		}
	}()

	// Create a channel to receive OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either server error or shutdown signal
	select {
	case err := <-serverErrChan:
		api.logger.Error("Server startup failed", "error", err)
		os.Exit(1)
	case sig := <-stop:
		api.logger.Info("Shutdown signal received",
			"signal", sig,
		)

		// Create a context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Stop transaction monitor
		if err := api.service.Stop(ctx); err != nil {
			api.logger.Error("Failed to stop service", "error", err)
		}

		// Shutdown HTTP server
		if err := srv.Shutdown(ctx); err != nil {
			api.logger.Error("Server shutdown failed", "error", err)
		}

		api.logger.Info("Server stopped")
	}
}

// GracefulStopServer stops the rest server gracefully
func (a *apiDetails) GracefulStopServer() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
	}
	a.logger.Info("Server exiting")
}
