package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCmd(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "cmd_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("ValidJUnitInput", func(t *testing.T) {
		// Create a minimal JUnit XML file
		xmlContent := `<testsuites>
	<testsuite name="4 Test Control" tests="1" failures="0" errors="0" time="0">
		<testcase name="4.1.1 Test check" classname="4.1 Test Group" time="0">
			<system-out>{"test_number":"4.1.1","status":"PASS"}</system-out>
		</testcase>
	</testsuite>
</testsuites>`

		inputFile := filepath.Join(tmpDir, "input.xml")
		outputFile := filepath.Join(tmpDir, "output.html")

		if err := os.WriteFile(inputFile, []byte(xmlContent), 0644); err != nil {
			t.Fatalf("Failed to write input file: %v", err)
		}

		// Reset command state
		rootCmd.SetArgs([]string{"--input", inputFile, "--output", outputFile})
		rootCmd.SilenceErrors = true
		rootCmd.SilenceUsage = true

		// Execute command
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("Expected successful execution, got error: %v", err)
		}

		// Verify output file was created
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Error("Expected output file to be created")
		}

		// Verify output file contains HTML
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		htmlContent := string(content)
		if !strings.Contains(htmlContent, "<!DOCTYPE html>") {
			t.Error("Expected output to contain HTML DOCTYPE")
		}
		if !strings.Contains(htmlContent, "Test check") {
			t.Error("Expected output to contain test data")
		}
	})

	t.Run("MissingInputFile", func(t *testing.T) {
		nonExistentFile := filepath.Join(tmpDir, "nonexistent.xml")
		outputFile := filepath.Join(tmpDir, "output2.html")

		// Create new command instance to avoid state contamination
		cmd := &cobra.Command{
			Use:   "kube-bench-report",
			Short: "Generate HTML reports from kube-bench output",
			RunE:  rootCmd.RunE,
		}
		cmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input file containing kube-bench output (required)")
		cmd.Flags().StringVarP(&outputFile, "output", "o", "kube-bench-report.html", "Output HTML report file")
		cmd.MarkFlagRequired("input")

		cmd.SetArgs([]string{"--input", nonExistentFile, "--output", outputFile})
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true

		// Execute command - should fail
		if err := cmd.Execute(); err == nil {
			t.Error("Expected error for nonexistent input file")
		}
	})

	t.Run("InvalidInputFile", func(t *testing.T) {
		// Create an invalid XML file
		invalidContent := `<invalid xml>`

		inputFile := filepath.Join(tmpDir, "invalid.xml")
		outputFile := filepath.Join(tmpDir, "output3.html")

		if err := os.WriteFile(inputFile, []byte(invalidContent), 0644); err != nil {
			t.Fatalf("Failed to write invalid input file: %v", err)
		}

		// Create new command instance
		cmd := &cobra.Command{
			Use:   "kube-bench-report",
			Short: "Generate HTML reports from kube-bench output",
			RunE:  rootCmd.RunE,
		}
		cmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input file containing kube-bench output (required)")
		cmd.Flags().StringVarP(&outputFile, "output", "o", "kube-bench-report.html", "Output HTML report file")
		cmd.MarkFlagRequired("input")

		cmd.SetArgs([]string{"--input", inputFile, "--output", outputFile})
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true

		// Execute command - should fail
		if err := cmd.Execute(); err == nil {
			t.Error("Expected error for invalid input file")
		}
	})
}