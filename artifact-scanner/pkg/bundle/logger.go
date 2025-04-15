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

package bundle

import (
	"runtime"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// zapHook represents a hook that forwards logrus logs to zap
// Logger: The zap logger to forward logs to
type zapHook struct {
	Logger *zap.Logger
}

// newEntry creates a new logrus entry with a zap hook
// logger: The zap logger to forward logs to
func newEntry(logger *zap.Logger) *logrus.Entry {
	log := logrus.New()
	hook := &zapHook{Logger: logger}
	log.Hooks.Add(hook)

	return logrus.NewEntry(log)
}

// Fire processes a logrus entry and forwards it to zap
// entry: The logrus entry to process
// Returns any error that occurred
func (hook *zapHook) Fire(entry *logrus.Entry) error {
	fields := make([]zap.Field, 0, 10)

	for key, value := range entry.Data {
		if key == logrus.ErrorKey {
			fields = append(fields, zap.Error(value.(error)))
		} else {
			fields = append(fields, zap.Any(key, value))
		}
	}

	switch entry.Level {
	case logrus.PanicLevel:
		hook.write(zapcore.PanicLevel, entry.Message, fields, entry.Caller)
	case logrus.FatalLevel:
		hook.write(zapcore.FatalLevel, entry.Message, fields, entry.Caller)
	case logrus.ErrorLevel:
		hook.write(zapcore.ErrorLevel, entry.Message, fields, entry.Caller)
	case logrus.WarnLevel:
		hook.write(zapcore.WarnLevel, entry.Message, fields, entry.Caller)
	case logrus.InfoLevel:
		hook.write(zapcore.InfoLevel, entry.Message, fields, entry.Caller)
	case logrus.DebugLevel, logrus.TraceLevel:
		hook.write(zapcore.DebugLevel, entry.Message, fields, entry.Caller)
	}

	return nil
}

// write writes a log entry to zap
// lvl: The log level
// msg: The log message
// fields: The log fields
// caller: The caller information
func (hook *zapHook) write(lvl zapcore.Level, msg string, fields []zap.Field, caller *runtime.Frame) {
	if ce := hook.Logger.Check(lvl, msg); ce != nil {
		if caller != nil {
			ce.Caller = zapcore.NewEntryCaller(caller.PC, caller.File, caller.Line, caller.PC != 0)
		}
		ce.Write(fields...)
	}
}

// Levels returns all logrus log levels that this hook should handle
func (hook *zapHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
