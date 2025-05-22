package parser

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// JUnitTestSuites represents the root element of a JUnit XML file
type JUnitTestSuites struct {
	XMLName    xml.Name        `xml:"testsuites"`
	TestSuites []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a test suite in JUnit XML
type JUnitTestSuite struct {
	XMLName   xml.Name       `xml:"testsuite"`
	Name      string         `xml:"name,attr"`
	Tests     int            `xml:"tests,attr"`
	Failures  int            `xml:"failures,attr"`
	Errors    int            `xml:"errors,attr"`
	Time      string         `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a test case in JUnit XML
type JUnitTestCase struct {
	XMLName     xml.Name          `xml:"testcase"`
	Name        string            `xml:"name,attr"`
	ClassName   string            `xml:"classname,attr"`
	Time        string            `xml:"time,attr"`
	Failure     *JUnitFailure     `xml:"failure,omitempty"`
	Skipped     *JUnitSkipped     `xml:"skipped,omitempty"`
	SystemOut   string            `xml:"system-out,omitempty"`
}

// JUnitFailure represents a failure in a test case
type JUnitFailure struct {
	XMLName xml.Name `xml:"failure"`
	Type    string   `xml:"type,attr"`
	Content string   `xml:",chardata"`
}

// JUnitSkipped represents a skipped test case
type JUnitSkipped struct {
	XMLName xml.Name `xml:"skipped"`
}

// TestDetails represents the detailed test information from system-out
type TestDetails struct {
	TestNumber     string   `json:"test_number"`
	TestDesc       string   `json:"test_desc"`
	Audit          string   `json:"audit"`
	AuditEnv       string   `json:"AuditEnv"`
	AuditConfig    string   `json:"AuditConfig"`
	Type           string   `json:"type"`
	Remediation    string   `json:"remediation"`
	TestInfo       []string `json:"test_info"`
	Status         string   `json:"status"`
	ActualValue    string   `json:"actual_value"`
	Scored         bool     `json:"scored"`
	IsMultiple     bool     `json:"IsMultiple"`
	ExpectedResult string   `json:"expected_result"`
	Reason         string   `json:"reason,omitempty"`
}

// ParseJUnitXML parses a JUnit XML file and converts it to BenchmarkData
func ParseJUnitXML(filePath string) (*BenchmarkData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var testSuites JUnitTestSuites
	if err := xml.Unmarshal(data, &testSuites); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %v", err)
	}

	return convertJUnitToBenchmarkData(testSuites)
}

// convertJUnitToBenchmarkData converts JUnit test data to BenchmarkData
func convertJUnitToBenchmarkData(testSuites JUnitTestSuites) (*BenchmarkData, error) {
	benchmarkData := &BenchmarkData{
		Controls: []Control{},
		Totals: Totals{
			Pass: 0,
			Fail: 0,
			Warn: 0,
			Info: 0,
		},
	}

	// Process each test suite as a control
	for _, suite := range testSuites.TestSuites {
		control := Control{
			ID:     getControlIDFromSuiteName(suite.Name),
			Text:   suite.Name,
			Type:   getControlTypeFromSuiteName(suite.Name),
			Groups: []Group{},
		}

		// Group test cases by class name
		groupMap := make(map[string][]JUnitTestCase)
		for _, testCase := range suite.TestCases {
			groupMap[testCase.ClassName] = append(groupMap[testCase.ClassName], testCase)
		}

		// Create groups from the map
		for groupName, testCases := range groupMap {
			group := Group{
				ID:     getGroupIDFromTestCases(testCases),
				Text:   groupName,
				Checks: []Check{},
			}

			// Process each test case as a check
			for _, testCase := range testCases {
				check, err := convertTestCaseToCheck(testCase)
				if err != nil {
					return nil, err
				}

				group.Checks = append(group.Checks, check)

				// Update totals
				switch check.State {
				case "pass":
					benchmarkData.Totals.Pass++
				case "fail":
					benchmarkData.Totals.Fail++
				case "warn":
					benchmarkData.Totals.Warn++
				case "info":
					benchmarkData.Totals.Info++
				}
			}

			control.Groups = append(control.Groups, group)
		}

		benchmarkData.Controls = append(benchmarkData.Controls, control)
	}

	return benchmarkData, nil
}

// convertTestCaseToCheck converts a JUnit test case to a Check
func convertTestCaseToCheck(testCase JUnitTestCase) (Check, error) {
	check := Check{
		ID:   getCheckIDFromTestCase(testCase),
		Text: getCheckTextFromTestCase(testCase),
	}

	// Determine state based on test case attributes
	if testCase.Failure != nil {
		check.State = "fail"
		check.Remediation = testCase.Failure.Content
	} else if testCase.Skipped != nil {
		check.State = "warn"
	} else {
		check.State = "pass"
	}

	// Extract additional details from system-out if available
	if testCase.SystemOut != "" {
		details, err := extractTestDetails(testCase.SystemOut)
		if err != nil {
			return check, nil // Continue with basic info if we can't parse details
		}

		// Override state with the one from details if available
		if details.Status != "" {
			check.State = strings.ToLower(details.Status)
		}

		// Use remediation from details if available
		if details.Remediation != "" {
			check.Remediation = details.Remediation
		}
	}

	return check, nil
}

// extractTestDetails extracts detailed test information from system-out
func extractTestDetails(systemOut string) (*TestDetails, error) {
	// Find the JSON part in system-out
	start := strings.Index(systemOut, "{")
	end := strings.LastIndex(systemOut, "}")

	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no valid JSON found in system-out")
	}

	jsonStr := systemOut[start:end+1]

	var details TestDetails
	if err := json.Unmarshal([]byte(jsonStr), &details); err != nil {
		return nil, err
	}

	return &details, nil
}

// getControlIDFromSuiteName extracts the control ID from the test suite name
func getControlIDFromSuiteName(suiteName string) string {
	// Extract numeric prefix if it exists
	for i, c := range suiteName {
		if !('0' <= c && c <= '9') {
			if i > 0 {
				return suiteName[:i]
			}
			break
		}
	}
	return ""
}

// getControlTypeFromSuiteName determines the control type based on the suite name
func getControlTypeFromSuiteName(suiteName string) string {
	suiteName = strings.ToLower(suiteName)
	if strings.Contains(suiteName, "master") || strings.Contains(suiteName, "control plane") {
		return "master"
	} else if strings.Contains(suiteName, "worker") || strings.Contains(suiteName, "node") {
		return "node"
	} else if strings.Contains(suiteName, "etcd") {
		return "etcd"
	} else if strings.Contains(suiteName, "policies") {
		return "policies"
	}
	return ""
}

// getGroupIDFromTestCases extracts the group ID from test cases
func getGroupIDFromTestCases(testCases []JUnitTestCase) string {
	if len(testCases) == 0 {
		return ""
	}

	// Try to extract group ID from the first test case ID
	id := getCheckIDFromTestCase(testCases[0])
	parts := strings.Split(id, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}

	return ""
}

// getCheckIDFromTestCase extracts the check ID from a test case
func getCheckIDFromTestCase(testCase JUnitTestCase) string {
	// Extract ID from the beginning of the test case name
	parts := strings.SplitN(testCase.Name, " ", 2)
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

// getCheckTextFromTestCase extracts the check text from a test case
func getCheckTextFromTestCase(testCase JUnitTestCase) string {
	// Extract text after the ID in the test case name
	parts := strings.SplitN(testCase.Name, " ", 2)
	if len(parts) >= 2 {
		return parts[1]
	}
	return testCase.Name
}
