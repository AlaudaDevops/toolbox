package parser

// BenchmarkData represents the parsed kube-bench data
type BenchmarkData struct {
	Controls []Control `json:"controls"`
	Totals   Totals    `json:"totals"`
}

// Control represents a top-level control section (e.g., "Master Node Security Configuration")
type Control struct {
	ID     string  `json:"id"`
	Text   string  `json:"text"`
	Type   string  `json:"type"`
	Groups []Group `json:"groups"`
}

// Group represents a group of checks within a control
type Group struct {
	ID     string  `json:"id"`
	Text   string  `json:"text"`
	Checks []Check `json:"checks"`
}

// Check represents an individual check item
type Check struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Audit       string `json:"audit,omitempty"`
	Remediation string `json:"remediation,omitempty"`
	State       string `json:"state"` // "pass", "fail", "warn", "info", "skipped"
}

// Totals represents the summary counts
type Totals struct {
	Pass int `json:"pass"`
	Fail int `json:"fail"`
	Warn int `json:"warn"`
	Info int `json:"info"`
}
