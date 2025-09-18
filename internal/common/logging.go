package common

import (
	"fmt"
	"github.com/ternarybob/arbor"
	"github.com/ternarybob/arbor/models"
	"os"
	"path/filepath"
	"sync"
)

var (
	logger arbor.ILogger
	once   sync.Once
)

func GetLogger() arbor.ILogger {
	once.Do(func() {
		if logger == nil {
			logger = initDefaultLogger()
		}
	})
	return logger
}

func InitLogger(config *LoggingConfig) error {
	var err error
	once.Do(func() {
		logger, err = createLogger(config)
	})
	return err
}

func initDefaultLogger() arbor.ILogger {
	config := DefaultLoggingConfig()
	logger, err := createLogger(config)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize default logger: %v\n", err)
		return arbor.NewLogger()
	}
	return logger
}

func createLogger(config *LoggingConfig) (arbor.ILogger, error) {
	// Get the directory where the executable is located
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// Create logs directory in the same directory as the executable
	logsDir := filepath.Join(execDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Initialize arbor logger
	l := arbor.NewLogger()

	// Configure file logging if requested
	if config.Output == "both" || config.Output == "file" || config.Output == "" {
		logFile := filepath.Join(logsDir, "gitsync.log")
		l = l.WithFileWriter(models.WriterConfiguration{
			Type:             models.LogWriterTypeFile,
			FileName:         logFile,
			TimeFormat:       "15:04:05",
			MaxSize:          int64(config.MaxSize * 1024 * 1024), // Convert MB to bytes
			MaxBackups:       config.MaxBackups,
			TextOutput:       true,
			DisableTimestamp: false,
		})
	}

	// Configure console logging if requested
	if config.Output == "both" || config.Output == "console" || config.Output == "" {
		l = l.WithConsoleWriter(models.WriterConfiguration{
			Type:             models.LogWriterTypeConsole,
			TimeFormat:       "15:04:05",
			TextOutput:       true,
			DisableTimestamp: false,
		})
	}

	// Set log level
	l = l.WithLevelFromString(config.Level)

	// Test logging immediately to verify it's working
	l.Info().Msg("GitSync logger initialized")

	return l, nil
}

func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:      "info",
		Format:     "text",
		Output:     "both",
		MaxSize:    100,
		MaxBackups: 3,
	}
}
