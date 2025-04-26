package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DailyRotatingWriter is a writer that automatically rotates log files daily
type DailyRotatingWriter struct {
	file           *os.File
	currentDate    string
	logDir         string
	filenameFormat string
	mu             sync.Mutex
}

// NewDailyRotatingWriter creates a new daily rotating writer
func NewDailyRotatingWriter(logDir string, filenameFormat string) (*DailyRotatingWriter, error) {
	writer := &DailyRotatingWriter{
		logDir:         logDir,
		filenameFormat: filenameFormat,
	}

	// Initialize with the current date and file
	if err := writer.rotateIfNeeded(); err != nil {
		return nil, err
	}

	return writer, nil
}

// rotateIfNeeded checks if the log file needs to be rotated and does so if necessary
func (w *DailyRotatingWriter) rotateIfNeeded() error {
	today := time.Now().Format("2006-01-02")

	// If the date hasn't changed and we already have a file, no need to rotate
	if today == w.currentDate && w.file != nil {
		return nil
	}

	// Close the existing file if it's open
	if w.file != nil {
		w.file.Close()
	}

	// Create the new log file with today's date
	filename := fmt.Sprintf(w.filenameFormat, today)
	logFilePath := filepath.Join(w.logDir, filename)

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	w.file = file
	w.currentDate = today

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

	return w.file.Write(p)
}

// Close closes the underlying file
func (w *DailyRotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}

	return nil
}
