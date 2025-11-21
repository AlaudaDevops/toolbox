package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Go module cache cleaning", func() {
	type testCase struct {
		description  string
		scriptBody   func(logPath string) string
		expectErr    bool
		expectedArgs string
	}

	DescribeTable("cleanGoModuleCache execution",
		func(tc testCase) {
			tempDir, err := os.MkdirTemp("", "go-clean-test-")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			logPath := filepath.Join(tempDir, "go-command.log")
			scriptPath := filepath.Join(tempDir, "go")

			Expect(os.WriteFile(scriptPath, []byte(tc.scriptBody(logPath)), 0755)).To(Succeed())

			originalPath := os.Getenv("PATH")
			Expect(os.Setenv("PATH", fmt.Sprintf("%s:%s", tempDir, originalPath))).To(Succeed())
			defer os.Setenv("PATH", originalPath)

			updater := NewGoUpdater(tempDir, nil)
			err = updater.cleanGoModuleCache()

			if tc.expectErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			content, readErr := os.ReadFile(logPath)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(strings.TrimSpace(string(content))).To(Equal(tc.expectedArgs))
		},
		Entry("successful cache cleanup", testCase{
			description: "should run go clean -modcache and succeed",
			scriptBody: func(logPath string) string {
				return fmt.Sprintf(`#!/bin/sh
echo "$@" > %q
exit 0
`, logPath)
			},
			expectErr:    false,
			expectedArgs: "clean -modcache",
		}),
		Entry("failing cache cleanup", testCase{
			description: "should surface errors when go clean fails",
			scriptBody: func(logPath string) string {
				return fmt.Sprintf(`#!/bin/sh
echo "$@" > %q
echo "boom" >&2
exit 2
`, logPath)
			},
			expectErr:    true,
			expectedArgs: "clean -modcache",
		}),
	)
})
