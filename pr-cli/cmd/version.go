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

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/AlaudaDevops/toolbox/pr-cli/internal/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long: `Display detailed version information including version number,
git commit, build date, Go version, and platform information.

Examples:
  pr-cli version              # Display detailed version info
  pr-cli version --output json # Display version info in JSON format`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return runVersion()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text|json)")
}

func runVersion() error {
	versionInfo := version.Get()

	switch outputFormat {
	case "json":
		return printVersionJSON(versionInfo)
	case "text":
		return printVersionText(versionInfo)
	default:
		return fmt.Errorf("unsupported output format: %s (supported: text, json)", outputFormat)
	}
}

func printVersionText(info version.Info) error {
	fmt.Printf("PR CLI Version Information:\n")
	fmt.Printf("  Version:     %s\n", info.Version)
	if info.GitCommit != "" {
		fmt.Printf("  Git Commit:  %s\n", info.GitCommit)
	}
	if info.BuildDate != "" {
		fmt.Printf("  Build Date:  %s\n", info.BuildDate)
	}
	fmt.Printf("  Go Version:  %s\n", info.GoVersion)
	fmt.Printf("  Compiler:    %s\n", info.Compiler)
	fmt.Printf("  Platform:    %s\n", info.Platform)
	return nil
}

func printVersionJSON(info version.Info) error {
	output, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version info to JSON: %w", err)
	}
	fmt.Println(string(output))
	return nil
}
