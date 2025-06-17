/*
Copyright 2025 The AlaudaDevops Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package scanner provides interfaces and types for security scanning
package scanner

import (
	"fmt"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/types"
)

// Scanner is the interface that all security scanners must implement
type Scanner interface {
	// Scan performs the security scan and returns vulnerabilities
	Scan() ([]types.Vulnerability, error)
	// Cleanup cleans up any temporary resources created during scanning
	Cleanup() error
	// GetName returns the name of the scanner
	GetName() string
}

// ScanResult represents the result of a security scan
type ScanResult struct {
	// Vulnerabilities is the list of vulnerabilities found
	Vulnerabilities []types.Vulnerability `json:"vulnerabilities"`
	// ScannerName is the name of the scanner that produced this result
	ScannerName string `json:"scanner_name"`
	// ScanPath is the path that was scanned
	ScanPath string `json:"scan_path"`
}

// NewScanner creates a new scanner instance based on the provided configuration
func NewScanner(scanPath string, config config.ScannerConfig) (Scanner, error) {
	switch config.Type {
	case "trivy":
		return NewTrivyScanner(scanPath, config), nil
	case "govulncheck":
		// Future implementation
		return nil, fmt.Errorf("govulncheck scanner not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported scanner type: %s", config.Type)
	}
}
