/*
Copyright 2024 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// globalLogger is the singleton logger instance
	globalLogger *zap.Logger
	// once ensures the logger is initialized only once
	once sync.Once
)

// Config represents logger configuration
type Config struct {
	Level       string `mapstructure:"level"`       // debug, info, warn, error
	Development bool   `mapstructure:"development"` // enables development mode
	Encoding    string `mapstructure:"encoding"`    // json or console
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:       "info",
		Development: false,
		Encoding:    "json",
	}
}

// Initialize initializes the global logger with the given configuration
func Initialize(cfg *Config) error {
	var err error
	once.Do(func() {
		if cfg == nil {
			cfg = DefaultConfig()
		}

		// Parse log level
		level, parseErr := zapcore.ParseLevel(cfg.Level)
		if parseErr != nil {
			level = zapcore.InfoLevel
		}

		// Create encoder config
		var encoderConfig zapcore.EncoderConfig
		if cfg.Development {
			encoderConfig = zap.NewDevelopmentEncoderConfig()
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		} else {
			encoderConfig = zap.NewProductionEncoderConfig()
			encoderConfig.TimeKey = "timestamp"
			encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		}

		// Create encoder
		var encoder zapcore.Encoder
		if cfg.Encoding == "console" {
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		} else {
			encoder = zapcore.NewJSONEncoder(encoderConfig)
		}

		// Create core
		core := zapcore.NewCore(
			encoder,
			zapcore.AddSync(os.Stdout),
			level,
		)

		// Create logger
		if cfg.Development {
			globalLogger = zap.New(core, zap.Development(), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
		} else {
			globalLogger = zap.New(core, zap.AddCaller())
		}

		err = parseErr
	})

	return err
}

// Get returns the global logger instance
// If the logger hasn't been initialized, it initializes with default config
func Get() *zap.Logger {
	if globalLogger == nil {
		_ = Initialize(DefaultConfig())
	}
	return globalLogger
}

// GetSugar returns a sugared logger for easier use
func GetSugar() *zap.SugaredLogger {
	return Get().Sugar()
}

// Sync flushes any buffered log entries
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// WithFields creates a new logger with the given fields
func WithFields(fields ...zap.Field) *zap.Logger {
	return Get().With(fields...)
}

// WithComponent creates a new logger with a component field
func WithComponent(component string) *zap.Logger {
	return Get().With(zap.String("component", component))
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// Panic logs a panic message and panics
func Panic(msg string, fields ...zap.Field) {
	Get().Panic(msg, fields...)
}

// Debugf logs a debug message with formatting (sugar)
func Debugf(template string, args ...interface{}) {
	GetSugar().Debugf(template, args...)
}

// Infof logs an info message with formatting (sugar)
func Infof(template string, args ...interface{}) {
	GetSugar().Infof(template, args...)
}

// Warnf logs a warning message with formatting (sugar)
func Warnf(template string, args ...interface{}) {
	GetSugar().Warnf(template, args...)
}

// Errorf logs an error message with formatting (sugar)
func Errorf(template string, args ...interface{}) {
	GetSugar().Errorf(template, args...)
}

// Fatalf logs a fatal message with formatting and exits (sugar)
func Fatalf(template string, args ...interface{}) {
	GetSugar().Fatalf(template, args...)
}

// Panicf logs a panic message with formatting and panics (sugar)
func Panicf(template string, args ...interface{}) {
	GetSugar().Panicf(template, args...)
}
