package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ParseFile parses a kube-bench output file and returns the structured data
func ParseFile(filePath string) (*BenchmarkData, error) {
	// Determine file format based on extension
	ext := strings.ToLower(filepath.Ext(filePath))

	// Read a small portion of the file to detect format
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Read the first 1024 bytes to detect format
	header := make([]byte, 1024)
	_, err = file.Read(header)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file header: %v", err)
	}

	// Reset file pointer
	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to reset file pointer: %v", err)
	}

	// Detect format based on content and extension
	headerStr := string(header)

	// Check for XML/JUnit format
	if ext == ".xml" || strings.HasPrefix(strings.TrimSpace(headerStr), "<?xml") || strings.HasPrefix(strings.TrimSpace(headerStr), "<testsuites>") {
		return ParseJUnitXML(filePath)
	}

	// Check for JSON format
	if ext == ".json" || strings.HasPrefix(strings.TrimSpace(headerStr), "{") {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %v", err)
		}
		return parseJSON(content)
	}

	// Default to text format
	return parseText(filePath)
}

// parseJSON parses kube-bench JSON output
func parseJSON(content []byte) (*BenchmarkData, error) {
	var data BenchmarkData
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	return &data, nil
}

// parseText parses kube-bench text output
func parseText(filePath string) (*BenchmarkData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	data := &BenchmarkData{
		Controls: []Control{},
		Totals:   Totals{},
	}

	var currentControl *Control
	var currentGroup *Group

	// Regular expressions for parsing
	controlRegex := regexp.MustCompile(`^\[INFO\] (\d+) (.+)$`)
	groupRegex := regexp.MustCompile(`^\[INFO\] (\d+\.\d+) (.+)$`)
	checkRegex := regexp.MustCompile(`^\[(PASS|FAIL|WARN|INFO)\] (\d+\.\d+\.\d+) (.+)$`)
	totalsRegex := regexp.MustCompile(`^== Summary ==$`)

	scanner := bufio.NewScanner(file)
	inRemediations := false
	inSummary := false
	remediationMap := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check if we're in the remediation section
		if line == "== Remediations ==" {
			inRemediations = true
			continue
		}

		// Check if we're in the summary section
		if totalsRegex.MatchString(line) {
			inRemediations = false
			inSummary = true
			continue
		}

		// Parse remediations
		if inRemediations {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				checkID := parts[0]
				if _, exists := remediationMap[checkID]; !exists {
					remediationMap[checkID] = parts[1]
				} else {
					remediationMap[checkID] += "\n" + parts[1]
				}
			}
			continue
		}

		// Parse summary
		if inSummary {
			parts := strings.Split(line, " checks ")
			if len(parts) == 2 {
				count, err := strconv.Atoi(parts[0])
				if err != nil {
					continue
				}

				switch strings.ToUpper(parts[1]) {
				case "PASS":
					data.Totals.Pass = count
				case "FAIL":
					data.Totals.Fail = count
				case "WARN":
					data.Totals.Warn = count
				case "INFO":
					data.Totals.Info = count
				}
			}
			continue
		}

		// Parse control
		if matches := controlRegex.FindStringSubmatch(line); matches != nil && len(matches) == 3 {
			control := Control{
				ID:     matches[1],
				Text:   matches[2],
				Groups: []Group{},
			}
			data.Controls = append(data.Controls, control)
			currentControl = &data.Controls[len(data.Controls)-1]
			continue
		}

		// Parse group
		if matches := groupRegex.FindStringSubmatch(line); matches != nil && len(matches) == 3 {
			if currentControl == nil {
				continue
			}

			group := Group{
				ID:     matches[1],
				Text:   matches[2],
				Checks: []Check{},
			}
			currentControl.Groups = append(currentControl.Groups, group)
			currentGroup = &currentControl.Groups[len(currentControl.Groups)-1]
			continue
		}

		// Parse check
		if matches := checkRegex.FindStringSubmatch(line); matches != nil && len(matches) == 4 {
			if currentGroup == nil {
				continue
			}

			state := strings.ToLower(matches[1])
			id := matches[2]

			check := Check{
				ID:          id,
				Text:        matches[3],
				State:       state,
				Remediation: remediationMap[id],
			}

			currentGroup.Checks = append(currentGroup.Checks, check)

			// Update totals
			switch state {
			case "pass":
				data.Totals.Pass++
			case "fail":
				data.Totals.Fail++
			case "warn":
				data.Totals.Warn++
			case "info":
				data.Totals.Info++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file: %v", err)
	}

	return data, nil
}
