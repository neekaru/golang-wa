package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// SetupLogging configures the application logging
func SetupLogging() (*log.Logger, error) {
	// Ensure logs directory exists
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %v", err)
	}

	// Create log file with timestamp in filename
	logFilePath := filepath.Join(logDir, fmt.Sprintf("whatsapp-api-%s.log", time.Now().Format("2006-01-02")))
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// Create multi-writer to log to both file and console
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)
	logger.Printf("Logging initialized to %s", logFilePath)

	// Print log location for easier access
	fmt.Printf("Logs are being written to: %s\n", logFilePath)

	return logger, nil
}

// SetupFallbackLogger creates a simple console logger when file logging fails
func SetupFallbackLogger() *log.Logger {
	fmt.Printf("Failed to set up file logging, using console logging only\n")
	return log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
}

// GetWriter returns the writer for the logger
func GetWriter(logger *log.Logger) io.Writer {
	return logger.Writer()
}
