package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseJUnitXML(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "junit_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("ValidJUnitXML", func(t *testing.T) {
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
	<testsuite name="4 Worker Node Security Configuration" tests="2" failures="1" errors="0" time="0">
		<testcase name="4.1.1 Ensure that the kubelet service file permissions are set to 600 or more restrictive (Automated)" classname="4.1 Worker Node Configuration Files" time="0">
			<failure type="">Run the below command</failure>
			<system-out>{"test_number":"4.1.1","test_desc":"Test description","audit":"audit command","remediation":"Fix the issue","test_info":["info"],"status":"FAIL","actual_value":"","scored":true,"IsMultiple":false,"expected_result":"expected"}</system-out>
		</testcase>
		<testcase name="4.1.2 Ensure that the kubelet service file ownership is set to root:root (Automated)" classname="4.1 Worker Node Configuration Files" time="0">
			<system-out>{"test_number":"4.1.2","test_desc":"Test description 2","status":"PASS","scored":true}</system-out>
		</testcase>
	</testsuite>
</testsuites>`

		xmlFile := filepath.Join(tmpDir, "valid.xml")
		if err := os.WriteFile(xmlFile, []byte(xmlContent), 0644); err != nil {
			t.Fatalf("Failed to write XML file: %v", err)
		}

		data, err := ParseJUnitXML(xmlFile)
		if err != nil {
			t.Fatalf("Expected successful parsing, got error: %v", err)
		}

		if len(data.Controls) != 1 {
			t.Errorf("Expected 1 control, got %d", len(data.Controls))
		}

		control := data.Controls[0]
		if control.ID != "4" {
			t.Errorf("Expected control ID '4', got '%s'", control.ID)
		}
		if control.Text != "4 Worker Node Security Configuration" {
			t.Errorf("Expected control text '4 Worker Node Security Configuration', got '%s'", control.Text)
		}

		if len(control.Groups) != 1 {
			t.Errorf("Expected 1 group, got %d", len(control.Groups))
		}

		group := control.Groups[0]
		if group.ID != "4.1" {
			t.Errorf("Expected group ID '4.1', got '%s'", group.ID)
		}

		if len(group.Checks) != 2 {
			t.Errorf("Expected 2 checks, got %d", len(group.Checks))
		}

		// Check first check with failure
		check1 := group.Checks[0]
		if check1.ID != "4.1.1" {
			t.Errorf("Expected check ID '4.1.1', got '%s'", check1.ID)
		}
		if check1.State != "fail" {
			t.Errorf("Expected check state 'fail', got '%s'", check1.State)
		}
		if check1.Remediation != "Fix the issue" {
			t.Errorf("Expected remediation 'Fix the issue', got '%s'", check1.Remediation)
		}

		// Check second check with pass
		check2 := group.Checks[1]
		if check2.ID != "4.1.2" {
			t.Errorf("Expected check ID '4.1.2', got '%s'", check2.ID)
		}
		if check2.State != "pass" {
			t.Errorf("Expected check state 'pass', got '%s'", check2.State)
		}
	})

	t.Run("InvalidXMLFormat", func(t *testing.T) {
		invalidXML := `<testsuites><testsuite><unclosed>`

		xmlFile := filepath.Join(tmpDir, "invalid.xml")
		if err := os.WriteFile(xmlFile, []byte(invalidXML), 0644); err != nil {
			t.Fatalf("Failed to write XML file: %v", err)
		}

		_, err := ParseJUnitXML(xmlFile)
		if err == nil {
			t.Error("Expected error for invalid XML format")
		}
	})

	t.Run("EmptyXMLFile", func(t *testing.T) {
		emptyFile := filepath.Join(tmpDir, "empty.xml")
		if err := os.WriteFile(emptyFile, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to write empty XML file: %v", err)
		}

		_, err := ParseJUnitXML(emptyFile)
		if err == nil {
			t.Error("Expected error for empty XML file")
		}
	})

	t.Run("MinimalValidXML", func(t *testing.T) {
		minimalXML := `<testsuites></testsuites>`

		xmlFile := filepath.Join(tmpDir, "minimal.xml")
		if err := os.WriteFile(xmlFile, []byte(minimalXML), 0644); err != nil {
			t.Fatalf("Failed to write XML file: %v", err)
		}

		data, err := ParseJUnitXML(xmlFile)
		if err != nil {
			t.Fatalf("Expected successful parsing, got error: %v", err)
		}

		if len(data.Controls) != 0 {
			t.Errorf("Expected 0 controls for minimal XML, got %d", len(data.Controls))
		}
	})

	t.Run("SkippedTestCase", func(t *testing.T) {
		skippedXML := `<testsuites>
	<testsuite name="4 Test Control" tests="1" failures="0" errors="0" time="0">
		<testcase name="4.1.1 Skipped test" classname="4.1 Test Group" time="0">
			<skipped></skipped>
			<system-out>{"test_number":"4.1.1","status":"WARN","scored":false}</system-out>
		</testcase>
	</testsuite>
</testsuites>`

		xmlFile := filepath.Join(tmpDir, "skipped.xml")
		if err := os.WriteFile(xmlFile, []byte(skippedXML), 0644); err != nil {
			t.Fatalf("Failed to write XML file: %v", err)
		}

		data, err := ParseJUnitXML(xmlFile)
		if err != nil {
			t.Fatalf("Expected successful parsing, got error: %v", err)
		}

		check := data.Controls[0].Groups[0].Checks[0]
		if check.State != "warn" {
			t.Errorf("Expected skipped test to have state 'warn', got '%s'", check.State)
		}
	})

	t.Run("MultipleTestSuites", func(t *testing.T) {
		multiSuiteXML := `<testsuites>
	<testsuite name="4 First Control" tests="1" failures="0" errors="0" time="0">
		<testcase name="4.1.1 First test" classname="4.1 First Group" time="0">
			<system-out>{"test_number":"4.1.1","status":"PASS"}</system-out>
		</testcase>
	</testsuite>
	<testsuite name="5 Second Control" tests="1" failures="1" errors="0" time="0">
		<testcase name="5.1.1 Second test" classname="5.1 Second Group" time="0">
			<failure type="">Failure message</failure>
			<system-out>{"test_number":"5.1.1","status":"FAIL"}</system-out>
		</testcase>
	</testsuite>
</testsuites>`

		xmlFile := filepath.Join(tmpDir, "multi_suite.xml")
		if err := os.WriteFile(xmlFile, []byte(multiSuiteXML), 0644); err != nil {
			t.Fatalf("Failed to write XML file: %v", err)
		}

		data, err := ParseJUnitXML(xmlFile)
		if err != nil {
			t.Fatalf("Expected successful parsing, got error: %v", err)
		}

		if len(data.Controls) != 2 {
			t.Errorf("Expected 2 controls, got %d", len(data.Controls))
		}

		if data.Controls[0].ID != "4" || data.Controls[1].ID != "5" {
			t.Errorf("Expected control IDs '4' and '5', got '%s' and '%s'",
				data.Controls[0].ID, data.Controls[1].ID)
		}
	})

	t.Run("FileNotFound", func(t *testing.T) {
		_, err := ParseJUnitXML("/nonexistent/file.xml")
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
		if !strings.Contains(err.Error(), "failed to open file") {
			t.Errorf("Expected file open error, got: %v", err)
		}
	})
}

func TestConvertJUnitToBenchmarkData(t *testing.T) {
	t.Run("ValidConversion", func(t *testing.T) {
		testSuites := JUnitTestSuites{
			TestSuites: []JUnitTestSuite{
				{
					Name:     "4 Test Control",
					Tests:    2,
					Failures: 1,
					Errors:   0,
					TestCases: []JUnitTestCase{
						{
							Name:      "4.1.1 First test",
							ClassName: "4.1 Test Group",
							Failure:   &JUnitFailure{Type: "failure", Content: "Test failed"},
							SystemOut: `{"test_number":"4.1.1","status":"FAIL","remediation":"Fix this"}`,
						},
						{
							Name:      "4.1.2 Second test", 
							ClassName: "4.1 Test Group",
							SystemOut: `{"test_number":"4.1.2","status":"PASS"}`,
						},
					},
				},
			},
		}

		data, err := convertJUnitToBenchmarkData(testSuites)
		if err != nil {
			t.Fatalf("Expected successful conversion, got error: %v", err)
		}

		if len(data.Controls) != 1 {
			t.Errorf("Expected 1 control, got %d", len(data.Controls))
		}

		control := data.Controls[0]
		if control.ID != "4" {
			t.Errorf("Expected control ID '4', got '%s'", control.ID)
		}

		if len(control.Groups) != 1 {
			t.Errorf("Expected 1 group, got %d", len(control.Groups))
		}

		if len(control.Groups[0].Checks) != 2 {
			t.Errorf("Expected 2 checks, got %d", len(control.Groups[0].Checks))
		}

		// Verify totals are calculated
		if data.Totals.Pass != 1 || data.Totals.Fail != 1 {
			t.Errorf("Expected totals 1 pass, 1 fail, got %d pass, %d fail",
				data.Totals.Pass, data.Totals.Fail)
		}
	})

	t.Run("EmptyTestSuites", func(t *testing.T) {
		testSuites := JUnitTestSuites{}

		data, err := convertJUnitToBenchmarkData(testSuites)
		if err != nil {
			t.Fatalf("Expected successful conversion, got error: %v", err)
		}

		if len(data.Controls) != 0 {
			t.Errorf("Expected 0 controls, got %d", len(data.Controls))
		}
	})
}

func TestConvertTestCaseToCheck(t *testing.T) {
	t.Run("ValidTestCaseWithSystemOut", func(t *testing.T) {
		testCase := JUnitTestCase{
			Name:      "4.1.1 Test check",
			ClassName: "4.1 Test Group",
			SystemOut: `{"test_number":"4.1.1","test_desc":"Test description","remediation":"Fix this issue","status":"FAIL","scored":true}`,
		}

		check, err := convertTestCaseToCheck(testCase)
		if err != nil {
			t.Fatalf("Expected successful conversion, got error: %v", err)
		}

		if check.ID != "4.1.1" {
			t.Errorf("Expected check ID '4.1.1', got '%s'", check.ID)
		}
		if check.Text != "Test check" {
			t.Errorf("Expected check text 'Test check', got '%s'", check.Text)
		}
		if check.Remediation != "Fix this issue" {
			t.Errorf("Expected remediation 'Fix this issue', got '%s'", check.Remediation)
		}
		if check.State != "fail" {
			t.Errorf("Expected state 'fail', got '%s'", check.State)
		}
	})

	t.Run("TestCaseWithFailure", func(t *testing.T) {
		testCase := JUnitTestCase{
			Name:      "4.1.1 Test check",
			ClassName: "4.1 Test Group",
			Failure:   &JUnitFailure{Type: "failure", Content: "Test failed"},
			SystemOut: `{"test_number":"4.1.1","status":"FAIL"}`,
		}

		check, err := convertTestCaseToCheck(testCase)
		if err != nil {
			t.Fatalf("Expected successful conversion, got error: %v", err)
		}

		if check.State != "fail" {
			t.Errorf("Expected state 'fail' for test with failure, got '%s'", check.State)
		}
	})

	t.Run("TestCaseWithSkipped", func(t *testing.T) {
		testCase := JUnitTestCase{
			Name:      "4.1.1 Test check",
			ClassName: "4.1 Test Group",
			Skipped:   &JUnitSkipped{},
			SystemOut: `{"test_number":"4.1.1","status":"WARN"}`,
		}

		check, err := convertTestCaseToCheck(testCase)
		if err != nil {
			t.Fatalf("Expected successful conversion, got error: %v", err)
		}

		if check.State != "warn" {
			t.Errorf("Expected state 'warn' for skipped test, got '%s'", check.State)
		}
	})

	t.Run("TestCaseWithInvalidSystemOut", func(t *testing.T) {
		testCase := JUnitTestCase{
			Name:      "4.1.1 Test check",
			ClassName: "4.1 Test Group",
			SystemOut: `invalid json`,
		}

		check, err := convertTestCaseToCheck(testCase)
		if err != nil {
			t.Fatalf("Expected successful conversion even with invalid system-out, got error: %v", err)
		}
		
		// Basic info should still be available from test case name
		if check.ID != "4.1.1" {
			t.Errorf("Expected check ID '4.1.1', got '%s'", check.ID)
		}
	})

	t.Run("TestCaseWithEmptySystemOut", func(t *testing.T) {
		testCase := JUnitTestCase{
			Name:      "4.1.1 Test check",
			ClassName: "4.1 Test Group",
			SystemOut: "",
		}

		check, err := convertTestCaseToCheck(testCase)
		if err != nil {
			t.Fatalf("Expected successful conversion even with empty system-out, got error: %v", err)
		}
		
		// Basic info should still be available from test case name
		if check.ID != "4.1.1" {
			t.Errorf("Expected check ID '4.1.1', got '%s'", check.ID)
		}
	})

	t.Run("TestCaseWithMalformedSystemOut", func(t *testing.T) {
		testCase := JUnitTestCase{
			Name:      "4.1.1 Test check",
			ClassName: "4.1 Test Group",
			SystemOut: `Some text {"test_number":"4.1.1","status":"PASS"} more text`,
		}

		check, err := convertTestCaseToCheck(testCase)
		if err != nil {
			t.Fatalf("Expected successful conversion even with malformed system-out, got error: %v", err)
		}

		if check.ID != "4.1.1" {
			t.Errorf("Expected check ID '4.1.1', got '%s'", check.ID)
		}
		if check.State != "pass" {
			t.Errorf("Expected state 'pass', got '%s'", check.State)
		}
	})
}

func TestExtractTestDetails(t *testing.T) {
	t.Run("ValidJSON", func(t *testing.T) {
		systemOut := `{"test_number":"4.1.1","test_desc":"Test description","remediation":"Fix issue","status":"FAIL","scored":true}`

		details, err := extractTestDetails(systemOut)
		if err != nil {
			t.Fatalf("Expected successful extraction, got error: %v", err)
		}

		if details.TestNumber != "4.1.1" {
			t.Errorf("Expected test number '4.1.1', got '%s'", details.TestNumber)
		}
		if details.TestDesc != "Test description" {
			t.Errorf("Expected test desc 'Test description', got '%s'", details.TestDesc)
		}
		if details.Remediation != "Fix issue" {
			t.Errorf("Expected remediation 'Fix issue', got '%s'", details.Remediation)
		}
		if details.Status != "FAIL" {
			t.Errorf("Expected status 'FAIL', got '%s'", details.Status)
		}
	})

	t.Run("JSONInMiddleOfText", func(t *testing.T) {
		systemOut := `Some prefix text {"test_number":"4.1.1","status":"PASS"} some suffix text`

		details, err := extractTestDetails(systemOut)
		if err != nil {
			t.Fatalf("Expected successful extraction, got error: %v", err)
		}

		if details.TestNumber != "4.1.1" {
			t.Errorf("Expected test number '4.1.1', got '%s'", details.TestNumber)
		}
		if details.Status != "PASS" {
			t.Errorf("Expected status 'PASS', got '%s'", details.Status)
		}
	})

	t.Run("NoJSONFound", func(t *testing.T) {
		systemOut := `No JSON here at all`

		_, err := extractTestDetails(systemOut)
		if err == nil {
			t.Error("Expected error when no JSON found")
		}
		if !strings.Contains(err.Error(), "no valid JSON found") {
			t.Errorf("Expected 'no valid JSON found' error, got: %v", err)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		systemOut := `{"test_number":"4.1.1","status":invalid}`

		_, err := extractTestDetails(systemOut)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("EmptyString", func(t *testing.T) {
		_, err := extractTestDetails("")
		if err == nil {
			t.Error("Expected error for empty string")
		}
	})
}

func TestGetControlIDFromSuiteName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"StandardFormat", "4 Worker Node Security Configuration", "4"},
		{"MultiDigit", "12 Some Control", "12"},
		{"NoNumber", "Worker Node Configuration", ""},
		{"NumberInMiddle", "Section 4 Configuration", ""},
		{"EmptyString", "", ""},
		{"OnlyNumber", "123", ""},
		{"NumberWithSpace", "4 ", "4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getControlIDFromSuiteName(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetControlTypeFromSuiteName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"MasterNode", "1 Master Node Security Configuration", "master"},
		{"WorkerNode", "4 Worker Node Security Configuration", "node"},
		{"ControlPlane", "1 Control Plane Security Configuration", "master"},
		{"EtcdNode", "2 Etcd Node Configuration", "node"},
		{"KubernetesPolicies", "5 Kubernetes Policies", "policies"},
		{"NoMatch", "Some Other Configuration", ""},
		{"EmptyString", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getControlTypeFromSuiteName(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetGroupIDFromTestCases(t *testing.T) {
	tests := []struct {
		name      string
		testCases []JUnitTestCase
		expected  string
	}{
		{
			name: "ValidTestCases",
			testCases: []JUnitTestCase{
				{Name: "4.1.1 Some test", ClassName: "4.1 Worker Node Configuration Files"},
				{Name: "4.1.2 Another test", ClassName: "4.1 Worker Node Configuration Files"},
			},
			expected: "4.1",
		},
		{
			name: "DifferentFormats",
			testCases: []JUnitTestCase{
				{Name: "1.2.3 Test", ClassName: "1.2 API Server"},
			},
			expected: "1.2",
		},
		{
			name:      "EmptyTestCases",
			testCases: []JUnitTestCase{},
			expected:  "",
		},
		{
			name: "NoValidClassName",
			testCases: []JUnitTestCase{
				{ClassName: "Invalid Format"},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getGroupIDFromTestCases(tt.testCases)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetCheckIDFromTestCase(t *testing.T) {
	tests := []struct {
		name     string
		testCase JUnitTestCase
		expected string
	}{
		{
			name:     "StandardFormat",
			testCase: JUnitTestCase{Name: "4.1.1 Ensure that the kubelet service file permissions are set"},
			expected: "4.1.1",
		},
		{
			name:     "DifferentID",
			testCase: JUnitTestCase{Name: "1.2.3 Some other check"},
			expected: "1.2.3",
		},
		{
			name:     "NoValidID",
			testCase: JUnitTestCase{Name: "Invalid format"},
			expected: "Invalid",
		},
		{
			name:     "EmptyName",
			testCase: JUnitTestCase{Name: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCheckIDFromTestCase(tt.testCase)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetCheckTextFromTestCase(t *testing.T) {
	tests := []struct {
		name     string
		testCase JUnitTestCase
		expected string
	}{
		{
			name:     "StandardFormat", 
			testCase: JUnitTestCase{Name: "4.1.1 Ensure that the kubelet service file permissions are set"},
			expected: "Ensure that the kubelet service file permissions are set",
		},
		{
			name:     "NoSpace",
			testCase: JUnitTestCase{Name: "4.1.1"},
			expected: "4.1.1",
		},
		{
			name:     "MultipleSpaces",
			testCase: JUnitTestCase{Name: "4.1.1   Multiple   spaces"},
			expected: "  Multiple   spaces",
		},
		{
			name:     "EmptyName",
			testCase: JUnitTestCase{Name: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCheckTextFromTestCase(tt.testCase)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}