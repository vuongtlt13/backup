package logger

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Logger wraps zap.Logger to provide a simpler interface
type Logger struct{}

var (
	instance *Logger
	once     sync.Once
)

// Get returns the singleton logger instance
func Get() *Logger {
	once.Do(func() {
		instance = &Logger{}
	})
	return instance
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return nil // No-op for this logger
}

// Info logs a message at info level with key-value pairs
func (l *Logger) Info(tag string, msg string, args ...interface{}) {
	logPrint("INFO", tag, msg, args...)
}

// Error logs a message at error level with key-value pairs
func (l *Logger) Error(tag string, msg string, args ...interface{}) {
	logPrint("ERROR", tag, msg, args...)
}

// Warn logs a message at warn level with key-value pairs
func (l *Logger) Warn(msg string, args ...interface{}) {
	logPrint("WARN", "", msg, args...)
}

func logPrint(level, tag, msg string, args ...interface{}) {
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	if tag != "" {
		tag = fmt.Sprintf("[%s]", tag)
	}
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	fmt.Fprintf(os.Stdout, "%s %s %s %s\n", timestamp, tag, level, msg)
}
