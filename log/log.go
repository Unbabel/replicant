// Package log implements a common structured logger.
package log

import (
	"os"
	"sync"

	"github.com/brunotm/log"
)

var (
	logger *log.Logger
)

// Init logger
func Init(level string) (err error) {
	l, err := log.ParseLevel(level)
	if err != nil {
		return err
	}

	var once sync.Once

	once.Do(func() {
		config := log.DefaultConfig
		config.Level = l
		config.EnableSampling = false
		config.CallerSkip = 1
		logger = log.New(os.Stdout, config)
	})
	return nil
}

// Debug creates a new log entry with the given message using the package level logger.
func Debug(message string) (e log.Entry) {
	if logger == nil {
		return log.Entry{}
	}
	return logger.Debug(message)
}

// Info creates a new log entry with the given message using the package level logger.
func Info(message string) (e log.Entry) {
	if logger == nil {
		return log.Entry{}
	}
	return logger.Info(message)
}

// Warn creates a new log entry with the given message using the package level logger.
func Warn(message string) (e log.Entry) {
	if logger == nil {
		return log.Entry{}
	}
	return logger.Warn(message)
}

// Error creates a new log entry with the given message using the package level logger.
func Error(message string) (e log.Entry) {
	if logger == nil {
		return log.Entry{}
	}
	return logger.Error(message)
}
