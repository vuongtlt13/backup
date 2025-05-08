package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggerSingleton(t *testing.T) {
	// Get logger instance twice
	logger1 := Get()
	logger2 := Get()

	// Verify both instances are the same
	assert.Same(t, logger1, logger2, "Logger instances should be the same")
}

func TestLoggerConfiguration(t *testing.T) {
	logger := Get()

	// Verify logger is not nil
	assert.NotNil(t, logger, "Logger should not be nil")

	// Verify logger is of correct type
	_, ok := interface{}(logger).(*Logger)
	assert.True(t, ok, "Logger should be of type *Logger")
}

func TestLoggerFields(t *testing.T) {
	logger := Get()

	// Test logging with fields
	logger.Info("test", "Test message with %s", "fields")
	logger.Error("test", "Test error with %s", "fields")
	logger.Warn("Test warning with %s", "fields")

	// No assertions needed as we're just testing that logging doesn't panic
}

func TestLoggerSync(t *testing.T) {
	logger := Get()

	// Sync should not return an error
	err := logger.Sync()
	assert.NoError(t, err, "Logger sync should not return an error")
}
