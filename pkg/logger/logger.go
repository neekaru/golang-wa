package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"path/filepath"

	"github.com/rs/zerolog"
)

// Global variable to track the rotating writer for proper cleanup
var activeRotatingWriter *DailyRotatingWriter

// Logger is a compatibility wrapper around zerolog that preserves
// common stdlib-style logging methods used across the codebase.
type Logger struct {
	zlog   zerolog.Logger
	writer io.Writer
}

// New creates a new compatibility logger.
func New(writer io.Writer) *Logger {
	base := zerolog.New(writer).With().Timestamp().Caller().Logger()
	return &Logger{zlog: base, writer: writer}
}

// WithPrefix returns a child logger that includes a static component field.
func (l *Logger) WithPrefix(component string) *Logger {
	child := l.zlog.With().Str("component", component).Logger()
	return &Logger{zlog: child, writer: l.writer}
}

// Printf logs with a level inferred from message content.
func (l *Logger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.logWithInferredLevel(msg)
}

// Println logs informational messages.
func (l *Logger) Println(v ...interface{}) {
	msg := strings.TrimSpace(fmt.Sprintln(v...))
	l.zlog.Info().Msg(msg)
}

// Fatalf logs an error and exits with status code 1.
func (l *Logger) Fatalf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.zlog.Fatal().Msg(msg)
}

// Writer returns the underlying output writer.
func (l *Logger) Writer() io.Writer {
	return l.writer
}

func (l *Logger) logWithInferredLevel(msg string) {
	text := strings.ToLower(msg)
	switch {
	case strings.Contains(text, "warn"):
		l.zlog.Warn().Msg(msg)
	case strings.Contains(text, "error"), strings.Contains(text, "failed"):
		l.zlog.Error().Msg(msg)
	default:
		l.zlog.Info().Msg(msg)
	}
}

// SetupLogging configures the application logging
func SetupLogging() (*Logger, error) {
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

	// Create zerolog-compatible logger wrapper with the multi-writer
	logger := New(multiWriter)

	// Get current log file path
	logFilePath := filepath.Join(logDir, fmt.Sprintf("whatsapp-api-%s.log", fileWriter.CurrentDate))

	// Log initialization - this will go to both console and file
	logger.Printf("Logging initialized to %s", logFilePath)
	logger.Printf("Log startup timestamp: %s", time.Now().Format(time.RFC3339))

	return logger, nil
}

// SetupFallbackLogger creates a simple console logger when file logging fails
func SetupFallbackLogger() *Logger {
	fmt.Printf("Failed to set up file logging, using console logging only\n")
	return New(os.Stdout)
}

// GetWriter returns the writer for the logger
func GetWriter(logger *Logger) io.Writer {
	return logger.Writer()
}

// CloseLogger properly closes the log file
func CloseLogger() error {
	if activeRotatingWriter != nil {
		return activeRotatingWriter.Close()
	}
	return nil
}
