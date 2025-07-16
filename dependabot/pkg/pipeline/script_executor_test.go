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

package pipeline

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScriptExecutor(t *testing.T) {
	projectPath := "/test/project"
	executor := NewScriptExecutor(projectPath)

	assert.NotNil(t, executor)
	assert.Equal(t, projectPath, executor.projectPath)
}

func TestExecuteScript_NilConfig(t *testing.T) {
	executor := NewScriptExecutor("/test/project")

	err := executor.ExecuteScript("test", nil)
	assert.NoError(t, err)
}

func TestExecuteScript_EmptyScript(t *testing.T) {
	executor := NewScriptExecutor("/test/project")
	scriptConfig := &config.ScriptConfig{
		Script: "",
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err)
}

func TestExecuteScript_SuccessfulExecution(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script: "echo 'Hello World'",
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err)
}

func TestExecuteScript_WithOutput(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script: "echo 'Test Output' && echo 'Second Line'",
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err)
}

func TestExecuteScript_WithTimeout(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script:  "sleep 1 && echo 'Completed'",
		Timeout: "5s",
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err)
}

func TestExecuteScript_TimeoutExceeded(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script:  "sleep 10",
		Timeout: "1s",
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test script failed")
}

func TestExecuteScript_InvalidTimeoutFormat(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script:  "echo 'Test'",
		Timeout: "invalid-timeout",
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err) // Should continue without timeout
}

func TestExecuteScript_ContinueOnError(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script:          "exit 1",
		ContinueOnError: true,
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err) // Should not return error due to ContinueOnError
}

func TestExecuteScript_ErrorWithoutContinue(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script:          "exit 1",
		ContinueOnError: false,
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test script failed")
}

func TestExecuteScript_ErrorWithOutput(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script:          "echo 'Error output' && exit 1",
		ContinueOnError: false,
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test script failed")
	assert.Contains(t, err.Error(), "Error output")
}

func TestExecuteScript_ComplexScript(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script: `
echo "Starting complex script"
for i in {1..3}; do
    echo "Iteration $i"
done
echo "Script completed"
`,
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err)
}

func TestExecuteScript_WithEnvironmentVariables(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script: `
echo "PATH: $PATH"
echo "PWD: $PWD"
echo "USER: $USER"
`,
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err)
}

func TestCreateTempScriptFile(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	stage := "test"
	scriptContent := "echo 'test script'"

	tempFile, err := executor.createTempScriptFile(stage, scriptContent)
	require.NoError(t, err)
	defer executor.cleanupTempFile(tempFile)

	// Check file exists
	_, err = os.Stat(tempFile)
	assert.NoError(t, err)

	// Check file content
	content, err := os.ReadFile(tempFile)
	assert.NoError(t, err)
	assert.Equal(t, scriptContent, string(content))

	// Check file permissions
	info, err := os.Stat(tempFile)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode()&os.ModePerm)

	// Check filename pattern
	baseName := filepath.Base(tempFile)
	assert.Contains(t, baseName, "dependabot_test_")
	assert.Contains(t, baseName, ".sh")
}

func TestCreateTempScriptFile_EmptyContent(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	stage := "test"
	scriptContent := ""

	tempFile, err := executor.createTempScriptFile(stage, scriptContent)
	require.NoError(t, err)
	defer executor.cleanupTempFile(tempFile)

	// Check file exists
	_, err = os.Stat(tempFile)
	assert.NoError(t, err)

	// Check file content
	content, err := os.ReadFile(tempFile)
	assert.NoError(t, err)
	assert.Equal(t, "", string(content))
}

func TestCreateTempScriptFile_LargeContent(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	stage := "test"

	// Create large script content
	largeContent := ""
	for i := 0; i < 1000; i++ {
		largeContent += "echo 'line " + string(rune(i%10+'0')) + "'\n"
	}

	tempFile, err := executor.createTempScriptFile(stage, largeContent)
	require.NoError(t, err)
	defer executor.cleanupTempFile(tempFile)

	// Check file content
	content, err := os.ReadFile(tempFile)
	assert.NoError(t, err)
	assert.Equal(t, largeContent, string(content))
}

func TestCleanupTempFile(t *testing.T) {
	executor := NewScriptExecutor("/tmp")

	// Create a temporary file
	tempFile, err := os.CreateTemp("/tmp", "test_cleanup_*.sh")
	require.NoError(t, err)
	tempFilePath := tempFile.Name()
	tempFile.Close()

	// Verify file exists
	_, err = os.Stat(tempFilePath)
	assert.NoError(t, err)

	// Clean up the file
	executor.cleanupTempFile(tempFilePath)

	// Verify file is removed
	_, err = os.Stat(tempFilePath)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestCleanupTempFile_NonExistentFile(t *testing.T) {
	executor := NewScriptExecutor("/tmp")

	// Try to clean up non-existent file
	executor.cleanupTempFile("/tmp/non_existent_file.sh")
	// Should not panic or return error
}

func TestExecuteScript_WorkingDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("/tmp", "script_executor_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test file in the temp directory
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	executor := NewScriptExecutor(tempDir)
	scriptConfig := &config.ScriptConfig{
		Script: "ls -la test.txt",
	}

	err = executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err)
}

func TestExecuteScript_WithStderrOutput(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script: "echo 'stdout message' >&1 && echo 'stderr message' >&2",
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err)
}

func TestExecuteScript_WithLongTimeout(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script:  "echo 'Quick execution'",
		Timeout: "1h", // Very long timeout
	}

	start := time.Now()
	err := executor.ExecuteScript("test", scriptConfig)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, duration, time.Second) // Should complete quickly despite long timeout
}

func TestExecuteScript_WithZeroTimeout(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script:  "echo 'Test'",
		Timeout: "0s",
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err) // Should continue without timeout
}

func TestExecuteScript_WithNegativeTimeout(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script:  "echo 'Test'",
		Timeout: "-1s",
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err) // Should continue without timeout
}

func TestExecuteScript_WithSpecialCharacters(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script: `echo "Special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"`,
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err)
}

func TestExecuteScript_WithUnicodeCharacters(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script: `echo "Unicode: 你好世界 🌍"`,
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err)
}

func TestExecuteScript_WithMultilineScript(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script: `#!/bin/bash
# This is a multiline script
echo "Line 1"
echo "Line 2"
echo "Line 3"
# End of script`,
	}

	err := executor.ExecuteScript("test", scriptConfig)
	assert.NoError(t, err)
}

func TestExecuteScript_StageNameInFilename(t *testing.T) {
	executor := NewScriptExecutor("/tmp")
	scriptConfig := &config.ScriptConfig{
		Script: "echo 'test'",
	}

	// Test with different stage names
	stages := []string{"PreScan", "PreCommit", "PostCommit", "CUSTOM_STAGE"}

	for _, stage := range stages {
		t.Run(stage, func(t *testing.T) {
			err := executor.ExecuteScript(stage, scriptConfig)
			assert.NoError(t, err)
		})
	}
}
