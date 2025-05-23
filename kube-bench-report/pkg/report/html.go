package report

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"github.com/alaudadevops/toolbox/kube-bench-report/pkg/parser"
)

// GenerateHTML generates an HTML report from the benchmark data
func GenerateHTML(data *parser.BenchmarkData) (string, error) {
	// Create a template
	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML template: %v", err)
	}

	// Prepare template data
	templateData := struct {
		Data      *parser.BenchmarkData
		Timestamp string
	}{
		Data:      data,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}

	return buf.String(), nil
}

// htmlTemplate is the HTML template for the report
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Kube-Bench Security Report</title>
    <style>
        :root {
            --pass-color: #4caf50;
            --fail-color: #f44336;
            --warn-color: #ff9800;
            --info-color: #aaaaaa;
            --dark-bg: #2d3748;
            --light-bg: #f8f9fa;
            --text-color: #333;
            --light-text: #f8f9fa;
            --border-color: #ddd;
        }

        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: var(--text-color);
            margin: 0;
            padding: 0;
            background-color: var(--light-bg);
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }

        header {
            background-color: var(--dark-bg);
            color: var(--light-text);
            padding: 20px;
            text-align: center;
            border-radius: 5px 5px 0 0;
        }

        h1, h2, h3, h4 {
            margin-top: 0;
        }

        .summary {
            display: flex;
            justify-content: space-between;
            margin: 20px 0;
            flex-wrap: wrap;
        }

        .summary-card {
            flex: 1;
            min-width: 200px;
            margin: 10px;
            padding: 20px;
            border-radius: 5px;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
            text-align: center;
            color: white;
        }

        .pass {
            background-color: var(--pass-color);
        }

        .fail {
            background-color: var(--fail-color);
        }

        .warn {
            background-color: var(--warn-color);
        }

        .info {
            background-color: var(--info-color);
        }

        .controls {
            margin-top: 30px;
        }

        .control {
            margin-bottom: 30px;
            border: 1px solid var(--border-color);
            border-radius: 5px;
            overflow: hidden;
        }

        .control-header {
            background-color: var(--dark-bg);
            color: var(--light-text);
            padding: 15px;
            cursor: pointer;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .control-content {
            padding: 0 15px;
        }

        .group {
            margin: 20px 0;
            border: 1px solid var(--border-color);
            border-radius: 5px;
            overflow: hidden;
        }

        .group-header {
            background-color: #f1f1f1;
            padding: 10px 15px;
            cursor: pointer;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .group-content {
            padding: 0 15px;
        }

        .check {
            margin: 15px 0;
            padding: 15px;
            border-radius: 5px;
            border-left: 5px solid;
        }

        .check.pass {
            border-left-color: var(--pass-color);
            background-color: rgba(76, 175, 80, 0.1);
        }

        .check.fail {
            border-left-color: var(--fail-color);
            background-color: rgba(244, 67, 54, 0.1);
        }

        .check.warn {
            border-left-color: var(--warn-color);
            background-color: rgba(255, 152, 0, 0.1);
        }

        .check.info {
            border-left-color: var(--info-color);
            background-color: rgba(33, 150, 243, 0.1);
        }

        .check-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
        }

        .check-id {
            font-weight: bold;
            margin-right: 10px;
        }

        .check-status {
            padding: 5px 10px;
            border-radius: 3px;
            color: white;
            font-weight: bold;
            text-transform: uppercase;
            font-size: 0.8em;
        }

        .check-text {
            margin-bottom: 10px;
        }

        .remediation {
            background-color: #f8f9fa;
            padding: 10px;
            border-radius: 3px;
            margin-top: 10px;
            white-space: pre-wrap;
            font-family: monospace;
        }

        .timestamp {
            text-align: right;
            margin-top: 20px;
            color: #666;
            font-size: 0.9em;
        }

        .toggle-icon {
            transition: transform 0.3s ease;
        }

        .collapsed .toggle-icon {
            transform: rotate(-90deg);
        }

        footer {
            text-align: center;
            margin-top: 30px;
            padding: 20px;
            background-color: var(--dark-bg);
            color: var(--light-text);
            border-radius: 0 0 5px 5px;
        }

        @media (max-width: 768px) {
            .summary-card {
                min-width: 100%;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>Kubernetes Security Benchmark Report</h1>
            <p>Based on CIS Kubernetes Benchmark</p>
        </header>

        <div class="summary">
            <div class="summary-card pass">
                <h2>{{ .Data.Totals.Pass }}</h2>
                <p>PASSED</p>
            </div>
            <div class="summary-card fail">
                <h2>{{ .Data.Totals.Fail }}</h2>
                <p>FAILED</p>
            </div>
            <div class="summary-card warn">
                <h2>{{ .Data.Totals.Warn }}</h2>
                <p>SKIPPED</p>
            </div>
            <div class="summary-card info">
                <h2>{{ .Data.Totals.Info }}</h2>
                <p>INFO</p>
            </div>
        </div>

        <div class="controls">
            {{ range .Data.Controls }}
            <div class="control">
                <div class="control-header" onclick="toggleSection(this.parentElement)">
                    <h2>{{ .ID }}. {{ .Text }}</h2>
                    <span class="toggle-icon">▼</span>
                </div>
                <div class="control-content">
                    {{ range .Groups }}
                    <div class="group">
                        <div class="group-header" onclick="toggleSection(this.parentElement)">
                            <h3>{{ .ID }} {{ .Text }}</h3>
                            <span class="toggle-icon">▼</span>
                        </div>
                        <div class="group-content">
                            {{ range .Checks }}
                            <div class="check {{ .State }}">
                                <div class="check-header">
                                    <div>
                                        <span class="check-id">{{ .ID }}</span>
                                    </div>
                                    <span class="check-status {{ .State }}">{{ .State }}</span>
                                </div>
                                <div class="check-text">{{ .Text }}</div>
                                {{ if .Remediation }}
                                <div class="remediation">
                                    <strong>Remediation:</strong>
                                    {{ .Remediation }}
                                </div>
                                {{ end }}
                            </div>
                            {{ end }}
                        </div>
                    </div>
                    {{ end }}
                </div>
            </div>
            {{ end }}
        </div>

        <div class="timestamp">
            Report generated: {{ .Timestamp }}
        </div>

        <footer>
            <p>Generated by kube-bench-report</p>
        </footer>
    </div>

    <script>
        function toggleSection(element) {
            const content = element.querySelector('.control-content, .group-content');
            const icon = element.querySelector('.toggle-icon');

            if (content.style.display === 'none') {
                content.style.display = 'block';
                element.classList.remove('collapsed');
            } else {
                content.style.display = 'none';
                element.classList.add('collapsed');
            }
        }

        // Initialize all sections as expanded
        document.addEventListener('DOMContentLoaded', function() {
            const sections = document.querySelectorAll('.control, .group');
            sections.forEach(section => {
                const content = section.querySelector('.control-content, .group-content');
                content.style.display = 'block';
            });
        });
    </script>
</body>
</html>`
