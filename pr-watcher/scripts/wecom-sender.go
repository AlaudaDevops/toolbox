package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/template"
)

// toFloat64 converts various numeric types to float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case uint:
		return float64(val)
	case uint64:
		return float64(val)
	case uint32:
		return float64(val)
	default:
		return 0
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <json-file> <template-file>\n", os.Args[0])
		os.Exit(1)
	}

	jsonFile := os.Args[1]
	templateFile := os.Args[2]

	// Read JSON file
	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading JSON file: %v\n", err)
		os.Exit(1)
	}

	// Parse JSON into generic map
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Read template file
	tmplContent, err := os.ReadFile(templateFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading template file: %v\n", err)
		os.Exit(1)
	}

	// Create template with custom functions
	funcMap := template.FuncMap{
		"sub": func(a, b interface{}) float64 {
			aFloat := toFloat64(a)
			bFloat := toFloat64(b)
			return aFloat - bFloat
		},
		"gt": func(a, b interface{}) bool {
			return toFloat64(a) > toFloat64(b)
		},
		"lt": func(a, b interface{}) bool {
			return toFloat64(a) < toFloat64(b)
		},
	}

	tmpl, err := template.New("wecom").Funcs(funcMap).Parse(string(tmplContent))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing template: %v\n", err)
		os.Exit(1)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing template: %v\n", err)
		os.Exit(1)
	}

	// Output the result
	fmt.Print(buf.String())
}
