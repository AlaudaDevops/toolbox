/*
Copyright 2025 The AlaudaDevops Authors.

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

package executor

import (
	"fmt"
	"time"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/git"
	"github.com/sirupsen/logrus"
)

// MockPRHandler is a mock implementation of PRHandler for testing
type MockPRHandler struct {
	ExecuteCommandFunc        func(command string, args []string) error
	PostCommentFunc           func(body string) error
	CheckPRStatusFunc         func(expectedStatus string) error
	GetCommentsWithCacheFunc  func() ([]git.Comment, error)
}

// ExecuteCommand mocks the ExecuteCommand method
func (m *MockPRHandler) ExecuteCommand(command string, args []string) error {
	if m.ExecuteCommandFunc != nil {
		return m.ExecuteCommandFunc(command, args)
	}
	return nil
}

// PostComment mocks the PostComment method
func (m *MockPRHandler) PostComment(body string) error {
	if m.PostCommentFunc != nil {
		return m.PostCommentFunc(body)
	}
	return nil
}

// CheckPRStatus mocks the CheckPRStatus method
func (m *MockPRHandler) CheckPRStatus(expectedStatus string) error {
	if m.CheckPRStatusFunc != nil {
		return m.CheckPRStatusFunc(expectedStatus)
	}
	return nil
}

// GetCommentsWithCache mocks the GetCommentsWithCache method
func (m *MockPRHandler) GetCommentsWithCache() ([]git.Comment, error) {
	if m.GetCommentsWithCacheFunc != nil {
		return m.GetCommentsWithCacheFunc()
	}
	return []git.Comment{}, nil
}

// MockMetricsRecorder is a mock implementation of MetricsRecorder for testing
type MockMetricsRecorder struct {
	CommandExecutionCalls   []CommandExecutionCall
	ProcessingDurationCalls []ProcessingDurationCall
}

// CommandExecutionCall represents a call to RecordCommandExecution
type CommandExecutionCall struct {
	Platform string
	Command  string
	Status   string
}

// ProcessingDurationCall represents a call to RecordProcessingDuration
type ProcessingDurationCall struct {
	Platform string
	Command  string
	Duration time.Duration
}

// RecordCommandExecution records a command execution call
func (m *MockMetricsRecorder) RecordCommandExecution(platform, command, status string) {
	m.CommandExecutionCalls = append(m.CommandExecutionCalls, CommandExecutionCall{
		Platform: platform,
		Command:  command,
		Status:   status,
	})
}

// RecordProcessingDuration records a processing duration call
func (m *MockMetricsRecorder) RecordProcessingDuration(platform, command string, duration time.Duration) {
	m.ProcessingDurationCalls = append(m.ProcessingDurationCalls, ProcessingDurationCall{
		Platform: platform,
		Command:  command,
		Duration: duration,
	})
}

// GetCommandExecutionCount returns the number of times RecordCommandExecution was called
func (m *MockMetricsRecorder) GetCommandExecutionCount() int {
	return len(m.CommandExecutionCalls)
}

// GetProcessingDurationCount returns the number of times RecordProcessingDuration was called
func (m *MockMetricsRecorder) GetProcessingDurationCount() int {
	return len(m.ProcessingDurationCalls)
}

// GetLastCommandExecution returns the last command execution call
func (m *MockMetricsRecorder) GetLastCommandExecution() *CommandExecutionCall {
	if len(m.CommandExecutionCalls) == 0 {
		return nil
	}
	return &m.CommandExecutionCalls[len(m.CommandExecutionCalls)-1]
}

// MockLogger is a mock implementation of logrus.FieldLogger for testing
type MockLogger struct {
	InfoMessages  []string
	ErrorMessages []string
	DebugMessages []string
}

// Infof logs an info message
func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.InfoMessages = append(m.InfoMessages, fmt.Sprintf(format, args...))
}

// Errorf logs an error message
func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.ErrorMessages = append(m.ErrorMessages, fmt.Sprintf(format, args...))
}

// Debugf logs a debug message
func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.DebugMessages = append(m.DebugMessages, fmt.Sprintf(format, args...))
}

// Warnf logs a warning message
func (m *MockLogger) Warnf(format string, args ...interface{}) {
	// Not used in current implementation
}

// Printf logs a message
func (m *MockLogger) Printf(format string, args ...interface{}) {
	// Not used in current implementation
}

// Print logs a message
func (m *MockLogger) Print(args ...interface{}) {
	// Not used in current implementation
}

// Debug logs a debug message
func (m *MockLogger) Debug(args ...interface{}) {
	// Not used in current implementation
}

// Info logs an info message
func (m *MockLogger) Info(args ...interface{}) {
	// Not used in current implementation
}

// Warn logs a warning message
func (m *MockLogger) Warn(args ...interface{}) {
	// Not used in current implementation
}

// Error logs an error message
func (m *MockLogger) Error(args ...interface{}) {
	// Not used in current implementation
}

// Fatal logs a fatal message
func (m *MockLogger) Fatal(args ...interface{}) {
	// Not used in current implementation
}

// Panic logs a panic message
func (m *MockLogger) Panic(args ...interface{}) {
	// Not used in current implementation
}

// WithField creates a new logger with a field
func (m *MockLogger) WithField(key string, value interface{}) *logrus.Entry {
	// Return a new entry for interface compliance
	return logrus.NewEntry(logrus.StandardLogger())
}

// WithFields creates a new logger with fields
func (m *MockLogger) WithFields(fields logrus.Fields) *logrus.Entry {
	// Return a new entry for interface compliance
	return logrus.NewEntry(logrus.StandardLogger())
}

// WithError creates a new logger with an error
func (m *MockLogger) WithError(err error) *logrus.Entry {
	// Return a new entry for interface compliance
	return logrus.NewEntry(logrus.StandardLogger())
}

// Tracef logs a trace message
func (m *MockLogger) Tracef(format string, args ...interface{}) {}

// Trace logs a trace message
func (m *MockLogger) Trace(args ...interface{}) {}

// Debugln logs a debug message with newline
func (m *MockLogger) Debugln(args ...interface{}) {}

// Infoln logs an info message with newline
func (m *MockLogger) Infoln(args ...interface{}) {}

// Println logs a message with newline
func (m *MockLogger) Println(args ...interface{}) {}

// Warnln logs a warning message with newline
func (m *MockLogger) Warnln(args ...interface{}) {}

// Warningln logs a warning message with newline
func (m *MockLogger) Warningln(args ...interface{}) {}

// Errorln logs an error message with newline
func (m *MockLogger) Errorln(args ...interface{}) {}

// Fatalln logs a fatal message with newline
func (m *MockLogger) Fatalln(args ...interface{}) {}

// Panicln logs a panic message with newline
func (m *MockLogger) Panicln(args ...interface{}) {}

// Traceln logs a trace message with newline
func (m *MockLogger) Traceln(args ...interface{}) {}

// Warningf logs a warning message
func (m *MockLogger) Warningf(format string, args ...interface{}) {}

// Warning logs a warning message
func (m *MockLogger) Warning(args ...interface{}) {}

// Fatalf logs a fatal message
func (m *MockLogger) Fatalf(format string, args ...interface{}) {}

// Panicf logs a panic message
func (m *MockLogger) Panicf(format string, args ...interface{}) {}
