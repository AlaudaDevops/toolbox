package report

import (
	"strings"
	"testing"
	htmltemplate "html/template"

	"github.com/alaudadevops/toolbox/kube-bench-report/pkg/parser"
)

func TestGenerateHTML(t *testing.T) {
	t.Run("ValidBenchmarkData", func(t *testing.T) {
		data := &parser.BenchmarkData{
			Controls: []parser.Control{
				{
					ID:   "4",
					Text: "Worker Node Security Configuration",
					Type: "node",
					Groups: []parser.Group{
						{
							ID:   "4.1",
							Text: "Worker Node Configuration Files",
							Checks: []parser.Check{
								{
									ID:          "4.1.1",
									Text:        "Ensure that the kubelet service file permissions are set to 600 or more restrictive",
									State:       "fail",
									Remediation: "Run the below command to fix this issue",
								},
								{
									ID:    "4.1.2",
									Text:  "Ensure that the kubelet service file ownership is set to root:root",
									State: "pass",
								},
								{
									ID:    "4.1.3",
									Text:  "If proxy kubeconfig file exists ensure permissions are set to 600 or more restrictive",
									State: "warn",
								},
								{
									ID:    "4.1.4",
									Text:  "Some info check",
									State: "info",
								},
							},
						},
					},
				},
			},
			Totals: parser.Totals{
				Pass: 1,
				Fail: 1,
				Warn: 1,
				Info: 1,
			},
		}

		html, err := GenerateHTML(data)
		if err != nil {
			t.Fatalf("Expected successful HTML generation, got error: %v", err)
		}

		if html == "" {
			t.Error("Expected non-empty HTML output")
		}

		// Check for essential HTML structure
		if !strings.Contains(html, "<!DOCTYPE html>") {
			t.Error("Expected HTML to contain DOCTYPE declaration")
		}
		if !strings.Contains(html, "<html") {
			t.Error("Expected HTML to contain html tag")
		}
		if !strings.Contains(html, "</html>") {
			t.Error("Expected HTML to contain closing html tag")
		}

		// Check for title
		if !strings.Contains(html, "Kube-Bench Security Report") {
			t.Error("Expected HTML to contain report title")
		}

		// Check for control data
		if !strings.Contains(html, "Worker Node Security Configuration") {
			t.Error("Expected HTML to contain control text")
		}
		if !strings.Contains(html, "4.1.1") {
			t.Error("Expected HTML to contain check ID")
		}

		// Check for totals
		if !strings.Contains(html, "1") { // Should contain the count "1" somewhere
			t.Error("Expected HTML to contain totals")
		}

		// Check for remediation
		if !strings.Contains(html, "Run the below command to fix this issue") {
			t.Error("Expected HTML to contain remediation text")
		}

		// Check for different states
		if !strings.Contains(html, "fail") {
			t.Error("Expected HTML to contain fail state")
		}
		if !strings.Contains(html, "pass") {
			t.Error("Expected HTML to contain pass state")
		}
		if !strings.Contains(html, "warn") {
			t.Error("Expected HTML to contain warn state")
		}

		// Check for CSS styling
		if !strings.Contains(html, "<style>") {
			t.Error("Expected HTML to contain CSS styles")
		}

		// Check for JavaScript
		if !strings.Contains(html, "<script>") {
			t.Error("Expected HTML to contain JavaScript")
		}

		// Check for timestamp placeholder (should contain date format)
		if !strings.Contains(html, "-") || !strings.Contains(html, ":") {
			t.Error("Expected HTML to contain timestamp")
		}
	})

	t.Run("EmptyBenchmarkData", func(t *testing.T) {
		data := &parser.BenchmarkData{
			Controls: []parser.Control{},
			Totals: parser.Totals{
				Pass: 0,
				Fail: 0,
				Warn: 0,
				Info: 0,
			},
		}

		html, err := GenerateHTML(data)
		if err != nil {
			t.Fatalf("Expected successful HTML generation for empty data, got error: %v", err)
		}

		if html == "" {
			t.Error("Expected non-empty HTML output even for empty data")
		}

		// Should still have basic HTML structure
		if !strings.Contains(html, "<!DOCTYPE html>") {
			t.Error("Expected HTML to contain DOCTYPE declaration")
		}

		// Should show zero totals
		if !strings.Contains(html, "0") {
			t.Error("Expected HTML to contain zero totals")
		}
	})

	t.Run("NilBenchmarkData", func(t *testing.T) {
		// Test what happens with nil data
		_, err := GenerateHTML(nil)
		if err == nil {
			t.Error("Expected error for nil benchmark data")
		}
	})

	t.Run("MultipleControls", func(t *testing.T) {
		data := &parser.BenchmarkData{
			Controls: []parser.Control{
				{
					ID:   "1",
					Text: "Master Node Security Configuration",
					Type: "master",
					Groups: []parser.Group{
						{
							ID:   "1.1",
							Text: "Master Node Configuration Files",
							Checks: []parser.Check{
								{
									ID:    "1.1.1",
									Text:  "Master check",
									State: "pass",
								},
							},
						},
					},
				},
				{
					ID:   "4",
					Text: "Worker Node Security Configuration",
					Type: "node",
					Groups: []parser.Group{
						{
							ID:   "4.1",
							Text: "Worker Node Configuration Files",
							Checks: []parser.Check{
								{
									ID:    "4.1.1",
									Text:  "Worker check",
									State: "fail",
								},
							},
						},
					},
				},
			},
			Totals: parser.Totals{
				Pass: 1,
				Fail: 1,
				Warn: 0,
				Info: 0,
			},
		}

		html, err := GenerateHTML(data)
		if err != nil {
			t.Fatalf("Expected successful HTML generation, got error: %v", err)
		}

		// Should contain both controls
		if !strings.Contains(html, "Master Node Security Configuration") {
			t.Error("Expected HTML to contain first control")
		}
		if !strings.Contains(html, "Worker Node Security Configuration") {
			t.Error("Expected HTML to contain second control")
		}

		// Should contain both check IDs
		if !strings.Contains(html, "1.1.1") {
			t.Error("Expected HTML to contain first check ID")
		}
		if !strings.Contains(html, "4.1.1") {
			t.Error("Expected HTML to contain second check ID")
		}
	})

	t.Run("SpecialCharactersInData", func(t *testing.T) {
		data := &parser.BenchmarkData{
			Controls: []parser.Control{
				{
					ID:   "1",
					Text: "Control with <script> and & special chars",
					Groups: []parser.Group{
						{
							ID:   "1.1",
							Text: "Group with \"quotes\" and 'apostrophes'",
							Checks: []parser.Check{
								{
									ID:          "1.1.1",
									Text:        "Check with <tags> & entities",
									State:       "fail",
									Remediation: "Fix this: use `command` and avoid <script>alert('xss')</script>",
								},
							},
						},
					},
				},
			},
			Totals: parser.Totals{
				Pass: 0,
				Fail: 1,
				Warn: 0,
				Info: 0,
			},
		}

		html, err := GenerateHTML(data)
		if err != nil {
			t.Fatalf("Expected successful HTML generation, got error: %v", err)
		}

		// HTML should be properly escaped
		if strings.Contains(html, "<script>alert('xss')</script>") {
			t.Error("Expected script tags to be escaped in HTML")
		}

		// Should contain escaped content
		if !strings.Contains(html, "&lt;") || !strings.Contains(html, "&gt;") {
			t.Error("Expected special characters to be HTML escaped")
		}
	})

	t.Run("LongContentData", func(t *testing.T) {
		longText := strings.Repeat("This is a very long text that should be handled properly by the HTML template. ", 100)
		
		data := &parser.BenchmarkData{
			Controls: []parser.Control{
				{
					ID:   "1",
					Text: longText,
					Groups: []parser.Group{
						{
							ID:   "1.1",
							Text: longText,
							Checks: []parser.Check{
								{
									ID:          "1.1.1",
									Text:        longText,
									State:       "fail",
									Remediation: longText,
								},
							},
						},
					},
				},
			},
			Totals: parser.Totals{
				Pass: 0,
				Fail: 1,
				Warn: 0,
				Info: 0,
			},
		}

		html, err := GenerateHTML(data)
		if err != nil {
			t.Fatalf("Expected successful HTML generation with long content, got error: %v", err)
		}

		if html == "" {
			t.Error("Expected non-empty HTML output")
		}

		// Should contain the long text
		if !strings.Contains(html, "This is a very long text") {
			t.Error("Expected HTML to contain long text content")
		}
	})

	t.Run("AllStatesRepresented", func(t *testing.T) {
		data := &parser.BenchmarkData{
			Controls: []parser.Control{
				{
					ID:   "1",
					Text: "Test Control",
					Groups: []parser.Group{
						{
							ID:   "1.1",
							Text: "Test Group",
							Checks: []parser.Check{
								{ID: "1.1.1", Text: "Pass check", State: "pass"},
								{ID: "1.1.2", Text: "Fail check", State: "fail"},
								{ID: "1.1.3", Text: "Warn check", State: "warn"},
								{ID: "1.1.4", Text: "Info check", State: "info"},
							},
						},
					},
				},
			},
			Totals: parser.Totals{
				Pass: 1,
				Fail: 1,
				Warn: 1,
				Info: 1,
			},
		}

		html, err := GenerateHTML(data)
		if err != nil {
			t.Fatalf("Expected successful HTML generation, got error: %v", err)
		}

		// Check that all states are properly represented
		stateClasses := []string{"pass", "fail", "warn", "info"}
		for _, state := range stateClasses {
			if !strings.Contains(html, state) {
				t.Errorf("Expected HTML to contain state class '%s'", state)
			}
		}
	})
}

func TestHTMLTemplate(t *testing.T) {
	t.Run("TemplateCompilation", func(t *testing.T) {
		// Test that the template can be parsed without data
		_, err := htmltemplate.New("test").Parse(htmlTemplate)
		if err != nil {
			t.Fatalf("HTML template should be valid, got parse error: %v", err)
		}
	})

	t.Run("TemplateContainsRequiredElements", func(t *testing.T) {
		// Check that template contains essential elements
		requiredElements := []string{
			"<!DOCTYPE html>",
			"<html",
			"<head>",
			"<title>",
			"<style>",
			"<body>",
			"<script>",
			"{{ .Data.Totals.Pass }}",
			"{{ .Data.Totals.Fail }}",
			"{{ .Data.Totals.Warn }}",
			"{{ .Data.Totals.Info }}",
			"{{ .Timestamp }}",
			"{{ range .Data.Controls }}",
		}

		for _, element := range requiredElements {
			if !strings.Contains(htmlTemplate, element) {
				t.Errorf("HTML template should contain '%s'", element)
			}
		}
	})
}