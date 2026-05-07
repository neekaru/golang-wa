package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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

	// Create a daily rotating writer with memory optimization
	fileWriter, err := NewDailyRotatingWriter(logDir, "whatsapp-api-%s.log")
	if err != nil {
		return nil, fmt.Errorf("failed to create log writer: %v", err)
	}

	// Store the writer for later cleanup
	activeRotatingWriter = fileWriter

	// Create multi-writer to log to both file and console
	// This avoids copying the data twice by writing to both outputs in sequence
	multiWriter := io.MultiWriter(os.Stdout, fileWriter)

	// Create a logger with the multi-writer
	// Using LstdFlags|log.Lshortfile provides useful context without excessive overhead
	logger := log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)

	// Get current log file path
	logFilePath := filepath.Join(logDir, fmt.Sprintf("whatsapp-api-%s.log", fileWriter.CurrentDate))

	// Log initialization - this will go to both console and file
	logger.Printf("Logging initialized to %s", logFilePath)

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
