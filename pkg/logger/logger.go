package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Global variable to track the rotating writer for proper cleanup
var activeRotatingWriter *DailyRotatingWriter

// SetupLogging configures the application logging
func SetupLogging() (*log.Logger, error) {
	// Ensure logs directory exists
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %v", err)
	}

	// Create a daily rotating writer
	fileWriter, err := NewDailyRotatingWriter(logDir, "whatsapp-api-%s.log")
	if err != nil {
		return nil, fmt.Errorf("failed to create log writer: %v", err)
	}

	// Store the writer for later cleanup
	activeRotatingWriter = fileWriter

	// Create multi-writer to log to both file and console
	multiWriter := io.MultiWriter(os.Stdout, fileWriter)
	logger := log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)

	// Get today's date for the initial log message
	today := time.Now().Format("2006-01-02")
	logFilePath := filepath.Join(logDir, fmt.Sprintf("whatsapp-api-%s.log", today))

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

// CloseLogger properly closes the log file
func CloseLogger() error {
	if activeRotatingWriter != nil {
		return activeRotatingWriter.Close()
	}
	return nil
}
