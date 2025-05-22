# Kube-Bench Report Generator

A CLI tool that generates user-readable, self-contained HTML reports from kube-bench output files.

## Overview

Kube-Bench Report Generator is a Go CLI application that takes kube-bench output (in text, JSON, or JUnit XML format) and generates a comprehensive, interactive HTML report. This makes it easier to understand and share the results of Kubernetes security benchmark tests.

## Features

- Parses text, JSON, and JUnit XML output formats from kube-bench
- Generates a self-contained HTML report with all styling included
- Interactive UI with expandable/collapsible sections
- Color-coded results for easy identification of issues
- Summary statistics at the top of the report
- Detailed remediation instructions for failed checks

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/alaudadevops/toolbox.git
cd toolbox/kube-bench-report

# Build the application
go build -o kube-bench-report

# Move the binary to a location in your PATH (optional)
sudo mv kube-bench-report /usr/local/bin/
```

## Usage

```bash
# Basic usage
kube-bench-report --input <kube-bench-output-file> --output <html-report-file>

# Example with text output
kube-bench-report --input kube-bench.txt --output report.html

# Example with JSON output
kube-bench-report --input kube-bench.json --output report.html

# Example with JUnit XML output
kube-bench-report --input kube-bench-junit.xml --output report.html
```

### Command Line Options

- `--input`, `-i`: Input file containing kube-bench output (required)
- `--output`, `-o`: Output HTML report file (default: "kube-bench-report.html")
- `--format`, `-f`: Input format (auto, text, json, junit) (default: "auto")
- `--help`, `-h`: Show help information

## Example Workflow

1. Run kube-bench on your Kubernetes cluster:
   ```bash
   # Run kube-bench and save the output
   kube-bench > kube-bench.txt

   # Or for JSON output
   kube-bench --json > kube-bench.json

   # Or for JUnit XML output
   kube-bench --junit > kube-bench-junit.xml
   ```

2. Generate an HTML report:
   ```bash
   kube-bench-report --input kube-bench.txt --output report.html
   ```

3. Open the HTML report in your browser:
   ```bash
   open report.html
   ```

## Report Structure

The generated HTML report includes:

- Summary statistics (pass, fail, warn, info counts)
- Controls organized by section (e.g., Control Plane, Worker Nodes)
- Groups of checks within each control
- Individual check results with:
  - ID and description
  - Status (PASS, FAIL, WARN, INFO)
  - Remediation instructions for failed checks

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Apache 2.0 License - see the LICENSE file for details.
