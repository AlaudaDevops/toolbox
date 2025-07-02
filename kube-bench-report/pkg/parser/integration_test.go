package parser

import (
	"path/filepath"
	"testing"
)

// Integration tests using real example files
func TestIntegrationWithExampleFiles(t *testing.T) {
	// Get the path to the example directory
	exampleDir := filepath.Join("..", "..", "example")

	t.Run("ParseExampleTextFile", func(t *testing.T) {
		textFile := filepath.Join(exampleDir, "kube-bench.txt")
		
		data, err := ParseFile(textFile)
		if err != nil {
			t.Fatalf("Expected successful parsing of example text file, got error: %v", err)
		}

		if data == nil {
			t.Fatal("Expected non-nil data")
		}

		if len(data.Controls) == 0 {
			t.Error("Expected at least one control in example file")
		}

		// Verify some expected content from the example file
		found4 := false
		found5 := false
		for _, control := range data.Controls {
			if control.ID == "4" {
				found4 = true
				if control.Text != "Worker Node Security Configuration" {
					t.Errorf("Expected control 4 text 'Worker Node Security Configuration', got '%s'", control.Text)
				}
			}
			if control.ID == "5" {
				found5 = true
				if control.Text != "Kubernetes Policies" {
					t.Errorf("Expected control 5 text 'Kubernetes Policies', got '%s'", control.Text)
				}
			}
		}

		if !found4 {
			t.Error("Expected to find control ID '4' in example file")
		}
		if !found5 {
			t.Error("Expected to find control ID '5' in example file")
		}

		// Verify that totals are reasonable (not zero for all)
		totalChecks := data.Totals.Pass + data.Totals.Fail + data.Totals.Warn + data.Totals.Info
		if totalChecks == 0 {
			t.Error("Expected non-zero total checks from example file")
		}
	})

	t.Run("ParseExampleJUnitFile", func(t *testing.T) {
		junitFile := filepath.Join(exampleDir, "kube-bench-junit.xml")
		
		data, err := ParseJUnitXML(junitFile)
		if err != nil {
			t.Fatalf("Expected successful parsing of example JUnit file, got error: %v", err)
		}

		if data == nil {
			t.Fatal("Expected non-nil data")
		}

		if len(data.Controls) == 0 {
			t.Error("Expected at least one control in example JUnit file")
		}

		// JUnit format should have remediations (unlike text format)
		foundRemediation := false
		for _, control := range data.Controls {
			for _, group := range control.Groups {
				for _, check := range group.Checks {
					if check.Remediation != "" {
						foundRemediation = true
						break
					}
				}
				if foundRemediation {
					break
				}
			}
			if foundRemediation {
				break
			}
		}

		if !foundRemediation {
			t.Error("Expected to find at least one remediation in JUnit example file")
		}

		// Verify different states are present
		foundStates := make(map[string]bool)
		for _, control := range data.Controls {
			for _, group := range control.Groups {
				for _, check := range group.Checks {
					foundStates[check.State] = true
				}
			}
		}

		expectedStates := []string{"pass", "fail", "warn"}
		for _, state := range expectedStates {
			if !foundStates[state] {
				t.Errorf("Expected to find state '%s' in JUnit example file", state)
			}
		}

		// Verify totals match actual counts
		totalExpected := data.Totals.Pass + data.Totals.Fail + data.Totals.Warn + data.Totals.Info
		if totalExpected == 0 {
			t.Error("Expected non-zero totals from JUnit example file")
		}
	})

	t.Run("CompareTextVsJUnitParsing", func(t *testing.T) {
		textFile := filepath.Join(exampleDir, "kube-bench.txt")
		junitFile := filepath.Join(exampleDir, "kube-bench-junit.xml")
		
		textData, err := ParseFile(textFile)
		if err != nil {
			t.Fatalf("Failed to parse text file: %v", err)
		}

		junitData, err := ParseJUnitXML(junitFile)
		if err != nil {
			t.Fatalf("Failed to parse JUnit file: %v", err)
		}

		// Both should have some controls
		if len(textData.Controls) == 0 {
			t.Error("Text parsing should find controls")
		}
		if len(junitData.Controls) == 0 {
			t.Error("JUnit parsing should find controls")
		}

		// JUnit should have more detailed information including remediations
		textRemediations := 0
		junitRemediations := 0

		for _, control := range textData.Controls {
			for _, group := range control.Groups {
				for _, check := range group.Checks {
					if check.Remediation != "" {
						textRemediations++
					}
				}
			}
		}

		for _, control := range junitData.Controls {
			for _, group := range control.Groups {
				for _, check := range group.Checks {
					if check.Remediation != "" {
						junitRemediations++
					}
				}
			}
		}

		if junitRemediations <= textRemediations {
			t.Errorf("Expected JUnit to have more remediations than text format, got JUnit:%d vs Text:%d", 
				junitRemediations, textRemediations)
		}
	})

	t.Run("ParseExampleFilesRoundTrip", func(t *testing.T) {
		junitFile := filepath.Join(exampleDir, "kube-bench-junit.xml")
		
		// Parse JUnit file
		data, err := ParseJUnitXML(junitFile)
		if err != nil {
			t.Fatalf("Failed to parse JUnit file: %v", err)
		}

		// Verify the data structure is complete and valid
		for i, control := range data.Controls {
			// Note: Control ID may be empty if suite name doesn't start with number
			// This is current implementation behavior with example files
			if control.Text == "" {
				t.Errorf("Control %d has empty text", i)
			}

			for j, group := range control.Groups {
				if group.ID == "" {
					t.Errorf("Control %d, Group %d has empty ID", i, j)
				}
				if group.Text == "" {
					t.Errorf("Control %d, Group %d has empty text", i, j)
				}

				for k, check := range group.Checks {
					if check.ID == "" {
						t.Errorf("Control %d, Group %d, Check %d has empty ID", i, j, k)
					}
					if check.Text == "" {
						t.Errorf("Control %d, Group %d, Check %d has empty text", i, j, k)
					}
					if check.State == "" {
						t.Errorf("Control %d, Group %d, Check %d has empty state", i, j, k)
					}

					// Verify state is valid
					validStates := map[string]bool{"pass": true, "fail": true, "warn": true, "info": true}
					if !validStates[check.State] {
						t.Errorf("Control %d, Group %d, Check %d has invalid state '%s'", i, j, k, check.State)
					}
				}
			}
		}

		// Verify totals are consistent with actual check counts
		actualPass := 0
		actualFail := 0
		actualWarn := 0
		actualInfo := 0

		for _, control := range data.Controls {
			for _, group := range control.Groups {
				for _, check := range group.Checks {
					switch check.State {
					case "pass":
						actualPass++
					case "fail":
						actualFail++
					case "warn":
						actualWarn++
					case "info":
						actualInfo++
					}
				}
			}
		}

		if data.Totals.Pass != actualPass {
			t.Errorf("Totals.Pass mismatch: expected %d, got %d", actualPass, data.Totals.Pass)
		}
		if data.Totals.Fail != actualFail {
			t.Errorf("Totals.Fail mismatch: expected %d, got %d", actualFail, data.Totals.Fail)
		}
		if data.Totals.Warn != actualWarn {
			t.Errorf("Totals.Warn mismatch: expected %d, got %d", actualWarn, data.Totals.Warn)
		}
		if data.Totals.Info != actualInfo {
			t.Errorf("Totals.Info mismatch: expected %d, got %d", actualInfo, data.Totals.Info)
		}
	})

	t.Run("EdgeCaseChecks", func(t *testing.T) {
		junitFile := filepath.Join(exampleDir, "kube-bench-junit.xml")
		
		data, err := ParseJUnitXML(junitFile)
		if err != nil {
			t.Fatalf("Failed to parse JUnit file: %v", err)
		}

		// Look for edge cases in the real data
		for _, control := range data.Controls {
			for _, group := range control.Groups {
				for _, check := range group.Checks {
					// Check for reasonable ID format
					if len(check.ID) < 3 {
						t.Errorf("Check ID '%s' seems too short", check.ID)
					}

					// Check for reasonable text length
					if len(check.Text) < 10 {
						t.Errorf("Check text '%s' seems too short", check.Text)
					}

					// If remediation exists, it should be meaningful
					if check.Remediation != "" && len(check.Remediation) < 5 {
						t.Errorf("Check remediation '%s' seems too short", check.Remediation)
					}
				}
			}
		}
	})
}