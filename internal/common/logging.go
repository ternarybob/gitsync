package common

import (
	"fmt"
	"github.com/ternarybob/arbor"
)

var logger arbor.ILogger

func InitLogger(config *LoggingConfig) error {
	logger = arbor.NewLogger()
	return nil
}

func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:       "debug",
		Format:      "text",
		Output:      "stdout",
		MaxFileSize: 100,
		MaxBackups:  3,
		MaxAge:      3,
	}
}

// LogInfo logs an info message with optional key-value pairs
func LogInfo(logger arbor.ILogger, message string, fields ...interface{}) {
	msg := logger.Info()
	if len(fields) > 0 {
		msg = msg.Str("additional_fields", FormatFields(fields...))
	}
	msg.Msg(message)
}

// LogError logs an error message with optional key-value pairs
func LogError(logger arbor.ILogger, message string, fields ...interface{}) {
	msg := logger.Error()
	if len(fields) > 0 {
		msg = msg.Str("additional_fields", FormatFields(fields...))
	}
	msg.Msg(message)
}

// LogDebug logs a debug message with optional key-value pairs
func LogDebug(logger arbor.ILogger, message string, fields ...interface{}) {
	msg := logger.Debug()
	if len(fields) > 0 {
		msg = msg.Str("additional_fields", FormatFields(fields...))
	}
	msg.Msg(message)
}

// LogWarn logs a warning message with optional key-value pairs
func LogWarn(logger arbor.ILogger, message string, fields ...interface{}) {
	msg := logger.Warn()
	if len(fields) > 0 {
		msg = msg.Str("additional_fields", FormatFields(fields...))
	}
	msg.Msg(message)
}

// FormatFields formats key-value pairs for logging
func FormatFields(fields ...interface{}) string {
	if len(fields) == 0 {
		return ""
	}

	result := ""
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			result += fmt.Sprintf(" %v=%v", fields[i], fields[i+1])
		}
	}
	return result
}

// Legacy compatibility functions
func Info(msg string) {
	if logger != nil {
		logger.Info().Msg(msg)
	}
}

func Infof(format string, args ...interface{}) {
	if logger != nil {
		logger.Info().Msg(fmt.Sprintf(format, args...))
	}
}

func Debug(msg string) {
	if logger != nil {
		logger.Debug().Msg(msg)
	}
}

func Error(msg string) {
	if logger != nil {
		logger.Error().Msg(msg)
	}
}

func Errorf(format string, args ...interface{}) {
	if logger != nil {
		logger.Error().Msg(fmt.Sprintf(format, args...))
	}
}

func Warn(msg string) {
	if logger != nil {
		logger.Warn().Msg(msg)
	}
}

func Warnf(format string, args ...interface{}) {
	if logger != nil {
		logger.Warn().Msg(fmt.Sprintf(format, args...))
	}
}

func Fatal(msg string) {
	if logger != nil {
		logger.Error().Msg(msg)
	}
	// Note: arbor doesn't have Fatal level, using Error and manual exit
	panic(msg)
}

func Fatalf(format string, args ...interface{}) {
	if logger != nil {
		logger.Error().Msg(fmt.Sprintf(format, args...))
	}
	// Note: arbor doesn't have Fatal level, using Error and manual exit
	panic(fmt.Sprintf(format, args...))
}

// Enhanced field logging for map-based fields
type LogEvent struct {
	fields map[string]interface{}
	level  string
}

func (le *LogEvent) Info(msg string) {
	if logger != nil && le.level == "info" {
		LogInfo(logger, msg, le.fieldsToSlice()...)
	}
}

func (le *LogEvent) Error(msg string) {
	if logger != nil && le.level == "error" {
		LogError(logger, msg, le.fieldsToSlice()...)
	}
}

func (le *LogEvent) Debug(msg string) {
	if logger != nil && le.level == "debug" {
		LogDebug(logger, msg, le.fieldsToSlice()...)
	}
}

func (le *LogEvent) Warn(msg string) {
	if logger != nil && le.level == "warn" {
		LogWarn(logger, msg, le.fieldsToSlice()...)
	}
}

func (le *LogEvent) fieldsToSlice() []interface{} {
	var result []interface{}
	for k, v := range le.fields {
		result = append(result, k, v)
	}
	return result
}

func WithField(key string, value interface{}) *LogEvent {
	return &LogEvent{
		fields: map[string]interface{}{key: value},
		level:  "info",
	}
}

func WithFields(fields map[string]interface{}) *LogEvent {
	return &LogEvent{
		fields: fields,
		level:  "info",
	}
}

func GetLogger() arbor.ILogger {
	if logger == nil {
		logger = arbor.NewLogger()
	}
	return logger
}