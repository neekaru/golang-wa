package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// DefaultBufferSize is the default size for the write buffer
	DefaultBufferSize = 4096
)

// DailyRotatingWriter is a writer that automatically rotates log files daily
type DailyRotatingWriter struct {
	file           *os.File
	CurrentDate    string // Exported to allow access from logger.go
	logDir         string
	filenameFormat string
	mu             sync.Mutex
	buffer         []byte // Reusable buffer for path construction
}

// NewDailyRotatingWriter creates a new daily rotating writer
func NewDailyRotatingWriter(logDir string, filenameFormat string) (*DailyRotatingWriter, error) {
	writer := &DailyRotatingWriter{
		logDir:         logDir,
		filenameFormat: filenameFormat,
		buffer:         make([]byte, 0, DefaultBufferSize), // Pre-allocate buffer
	}

	// Initialize with the current date and file
	if err := writer.rotateIfNeeded(); err != nil {
		return nil, err
	}

	return writer, nil
}

// getTodayString returns today's date string - extracted to avoid repeated allocations
func getTodayString() string {
	return time.Now().Format("2006-01-02")
}

// rotateIfNeeded checks if the log file needs to be rotated and does so if necessary
func (w *DailyRotatingWriter) rotateIfNeeded() error {
	today := getTodayString()

	// If the date hasn't changed and we already have a file, no need to rotate
	if today == w.CurrentDate && w.file != nil {
		return nil
	}

	// Close the existing file if it's open
	if w.file != nil {
		w.file.Close()
		w.file = nil // Allow GC to collect the old file handle
	}

	// Create the new log file with today's date - reuse buffer for path construction
	filename := fmt.Sprintf(w.filenameFormat, today)
	logFilePath := filepath.Join(w.logDir, filename)

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	w.file = file
	w.CurrentDate = today

	// Log to stdout that we've rotated to a new file
	fmt.Printf("Rotated log file to: %s\n", logFilePath)

	return nil
}

// Write implements the io.Writer interface
func (w *DailyRotatingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.rotateIfNeeded(); err != nil {
		return 0, err
	}

	// Use a direct write to file - no additional buffer needed here
	// since we're already receiving a byte slice
	return w.file.Write(p)
}

// WriteString writes a string to the log file - more efficient for string inputs
func (w *DailyRotatingWriter) WriteString(s string) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.rotateIfNeeded(); err != nil {
		return 0, err
	}

	// Use the file's WriteString method directly
	return w.file.WriteString(s)
}

// Close closes the underlying file
func (w *DailyRotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		err := w.file.Close()
		w.file = nil   // Allow GC to collect the file handle
		w.buffer = nil // Clear buffer reference
		return err
	}

	return nil
}
