package cmd

import (
	"fmt"
	"os"

	"github.com/alaudadevops/toolbox/kube-bench-report/pkg/parser"
	"github.com/alaudadevops/toolbox/kube-bench-report/pkg/report"
	"github.com/spf13/cobra"
)

var (
	inputFile  string
	outputFile string
	format     string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kube-bench-report",
	Short: "Generate HTML reports from kube-bench output",
	Long: `kube-bench-report is a CLI tool that generates user-readable, self-contained HTML reports
from kube-bench output files. It can process text, JSON, and JUnit XML output formats from kube-bench.

Example usage:
  kube-bench-report --input kube-bench.txt --output report.html
  kube-bench-report --input kube-bench.json --output report.html
  kube-bench-report --input kube-bench-junit.xml --output report.html`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate input file exists
		if _, err := os.Stat(inputFile); os.IsNotExist(err) {
			return fmt.Errorf("input file does not exist: %s", inputFile)
		}

		// Parse the input file
		benchData, err := parser.ParseFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to parse input file: %v", err)
		}

		// Generate the HTML report
		htmlContent, err := report.GenerateHTML(benchData)
		if err != nil {
			return fmt.Errorf("failed to generate HTML report: %v", err)
		}

		// Write the HTML report to the output file
		if err := os.WriteFile(outputFile, []byte(htmlContent), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %v", err)
		}

		fmt.Printf("Report successfully generated: %s\n", outputFile)
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input file containing kube-bench output (required)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "kube-bench-report.html", "Output HTML report file")
	rootCmd.Flags().StringVarP(&format, "format", "f", "auto", "Input format (auto, text, json, junit)")

	rootCmd.MarkFlagRequired("input")
}
