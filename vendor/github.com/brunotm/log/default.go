package log

import (
	"os"
	"sync"
)

var (
	defaultMtx    sync.Mutex
	defaultLogger *Logger
)

func init() {
	config := DefaultConfig
	config.Level = DEBUG
	config.EnableSampling = false
	config.CallerSkip = 1
	defaultLogger = New(os.Stdout, config)
}

// Debug creates a new log entry with the given message using the package level logger.
func Debug(message string) (e Entry) {
	return defaultLogger.Debug(message)
}

// Info creates a new log entry with the given message using the package level logger.
func Info(message string) (e Entry) {
	return defaultLogger.Info(message)
}

// Warn creates a new log entry with the given message using the package level logger.
func Warn(message string) (e Entry) {
	return defaultLogger.Warn(message)
}

// Error creates a new log entry with the given message using the package level logger.
func Error(message string) (e Entry) {
	return defaultLogger.Error(message)
}
