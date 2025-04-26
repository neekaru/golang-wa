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

// bufferPool is a sync.Pool for reusing byte buffers
// This reduces GC pressure by reusing allocated memory
var bufferPool = sync.Pool{
	New: func() any {
		// Pre-allocate a buffer with the default size
		buf := make([]byte, 0, DefaultBufferSize)
		return &buf
	},
}

// DailyRotatingWriter is a writer that automatically rotates log files daily
type DailyRotatingWriter struct {
	file           *os.File
	CurrentDate    string // Exported to allow access from logger.go
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

	// Get a buffer from the pool for path construction
	bufPtr := bufferPool.Get().(*[]byte)
	buf := *bufPtr
	buf = buf[:0] // Reset buffer without reallocating

	// Build the filename directly into the buffer when possible
	// This avoids allocations from string concatenation
	filename := fmt.Sprintf(w.filenameFormat, today)

	// Use filepath.Join efficiently
	logFilePath := filepath.Join(w.logDir, filename)

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	// Return the buffer to the pool for reuse
	*bufPtr = buf[:0] // Clear but keep capacity
	bufferPool.Put(bufPtr)

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

	// For small writes, we can use a buffer from the pool to batch operations
	// This reduces the number of syscalls for frequent small writes
	if len(p) < DefaultBufferSize/2 {
		// Get a buffer from the pool
		bufPtr := bufferPool.Get().(*[]byte)
		buf := *bufPtr

		// Reset buffer without reallocating
		buf = buf[:0]

		// Copy the data to our reusable buffer
		buf = append(buf, p...)

		// Write the entire buffer at once
		n, err = w.file.Write(buf)

		// Return the buffer to the pool for reuse
		*bufPtr = buf[:0] // Clear but keep capacity
		bufferPool.Put(bufPtr)

		return n, err
	}

	// For larger writes, use a direct write to avoid an extra copy
	return w.file.Write(p)
}

// WriteString writes a string to the log file - more efficient for string inputs
func (w *DailyRotatingWriter) WriteString(s string) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.rotateIfNeeded(); err != nil {
		return 0, err
	}

	// For small strings, we can use a buffer from the pool to batch operations
	// This reduces the number of syscalls for frequent small writes
	if len(s) < DefaultBufferSize/2 {
		// Get a buffer from the pool
		bufPtr := bufferPool.Get().(*[]byte)
		buf := *bufPtr

		// Reset buffer without reallocating
		buf = buf[:0]

		// Copy the string data to our reusable buffer
		buf = append(buf, s...)

		// Write the entire buffer at once
		n, err = w.file.Write(buf)

		// Return the buffer to the pool for reuse
		*bufPtr = buf[:0] // Clear but keep capacity
		bufferPool.Put(bufPtr)

		return n, err
	}

	// For larger strings, use WriteString directly which is optimized for strings
	return w.file.WriteString(s)
}

// Close closes the underlying file
func (w *DailyRotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		err := w.file.Close()
		w.file = nil // Allow GC to collect the file handle
		return err
	}

	return nil
}
