package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neekaru/whatsappgo-bot/internal/app"
	"github.com/neekaru/whatsappgo-bot/internal/config"
	"github.com/neekaru/whatsappgo-bot/internal/server"
	"github.com/neekaru/whatsappgo-bot/pkg/logger"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Set up logging
	appLogger, err := logger.SetupLogging()
	if err != nil {
		appLogger = logger.SetupFallbackLogger()
	}

	appLogger.Println("Starting WhatsApp API service")

	// Create application configuration
	appConfig := config.NewConfig()

	// Ensure data directory exists
	if err := appConfig.EnsureDataDir(); err != nil {
		appLogger.Fatalf("Failed to create data directory: %v", err)
	}
	appLogger.Println("Ensured data directory exists")

	// Create application instance
	application := app.NewApp(appLogger)

	// Create and configure HTTP server
	srv := server.NewServer(application, appConfig)
	srv.SetupRoutes()

	// Start the server
	if err := srv.Start(); err != nil {
		appLogger.Fatalf("Failed to start server: %v", err)
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Fatalf("Server shutdown failed: %v", err)
	}

	// Close the logger to ensure all logs are flushed
	if err := logger.CloseLogger(); err != nil {
		fmt.Printf("Error closing logger: %v\n", err)
	}
}
