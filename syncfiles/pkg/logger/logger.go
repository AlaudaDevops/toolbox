/*
	Copyright 2025 AlaudaDevops authors

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
	"context"
	"strings"

	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerKey struct{}

// LogLevel global flag

// WithLogger set a logger instance into a context
func WithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerKey{}, logger)
}

// GetLogger get a logger instance form a context
func GetLogger(ctx context.Context) (logger *zap.SugaredLogger) {
	if ctx == nil {
		return nil
	}
	logger, ok := ctx.Value(loggerKey{}).(*zap.SugaredLogger)
	if ok {
		return logger
	}
	return nil
	// fallback logger
	// return NewLogger(zapcore.AddSync(os.Stderr), zapcore.InfoLevel)
}

// NewLoggerFromContext similar to `GetLogger`, but return a default logger if there is no
// logger instance in the context
func NewLoggerFromContext(ctx context.Context, level zapcore.LevelEnabler) (logger *zap.SugaredLogger) {
	if logger = GetLogger(ctx); logger == nil {
		logger = NewLogger(zapcore.AddSync(os.Stderr), level)
	}
	return
}

// NewLogger construct a logger
func NewLogger(writer zapcore.WriteSyncer, level zapcore.LevelEnabler, opts ...zap.Option) *zap.SugaredLogger {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "logger",
		EncodeLevel:    EmojiLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderCfg), writer, level)
	return zap.New(core, opts...).Sugar()
}

// LogLevel wrapper for zapcore.LevelEnabler
type LogLeveler struct {
	Level string
}

// Enabled implements zapcore.LevelEnabler to return true if a log level was implemented
func (l LogLeveler) Enabled(level zapcore.Level) bool {
	return GetLogLevel(l.Level).Enabled(level)
}

// GetLogLevel returns a representation of the log level from a string
func GetLogLevel(level string) zapcore.LevelEnabler {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	case "info":
		fallthrough
	default:
		return zapcore.InfoLevel
	}
}

// EmojiLevelEncoder prints an emoji instead of the log level
// ❌ for Panic, Error and Fatal levels
// 🐛 for Debug
// ❗️ for Warning
// 📢 for Info and everything else
func EmojiLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	value := "==> "
	switch l {
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.ErrorLevel, zapcore.FatalLevel:
		value += "❌"
	case zap.DebugLevel:
		value += "🐛"
	case zap.WarnLevel:
		value += "❗️"
	case zap.InfoLevel:
		fallthrough
	default:
		value += "📢"
	}
	enc.AppendString(value)
}
