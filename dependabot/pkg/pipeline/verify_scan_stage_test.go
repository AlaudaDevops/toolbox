package pipeline

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/testings"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pipeline verify scan stage", func() {
	type hookSpec struct {
		Enabled         bool   `yaml:"enabled"`
		ContinueOnError bool   `yaml:"continueOnError"`
		ScriptTemplate  string `yaml:"scriptTemplate"`
	}

	type scanSpec struct {
		VulnerabilityCount int    `yaml:"vulnerabilityCount"`
		Error              string `yaml:"error"`
	}

	type sourceSpec struct {
		Description string   `yaml:"description"`
		PreVerify   hookSpec `yaml:"preVerify"`
		PostVerify  hookSpec `yaml:"postVerify"`
		Scan        scanSpec `yaml:"scan"`
	}

	type goldenSpec struct {
		Description       string `yaml:"description"`
		ShouldSkipUpdate  bool   `yaml:"shouldSkipUpdate"`
		WantError         bool   `yaml:"wantError"`
		ExpectPreMarker   bool   `yaml:"expectPreMarker"`
		ExpectPostMarker  bool   `yaml:"expectPostMarker"`
		ExpectScanInvoked bool   `yaml:"expectScanInvoked"`
	}

	type testCase struct {
		description string
		sourceFile  string
		goldenFile  string
	}

	renderHook := func(spec hookSpec, preMarker, postMarker string) *config.HookConfig {
		if !spec.Enabled {
			return nil
		}

		replacer := strings.NewReplacer(
			"${PRE_MARKER}", preMarker,
			"${POST_MARKER}", postMarker,
		)

		return &config.HookConfig{
			Script:          replacer.Replace(spec.ScriptTemplate),
			ContinueOnError: spec.ContinueOnError,
		}
	}

	fileExists := func(path string) bool {
		_, err := os.Stat(path)
		return err == nil
	}

	buildScanFn := func(spec scanSpec) (func() ([]types.Vulnerability, error), *bool) {
		scanInvoked := false

		scanFn := func() ([]types.Vulnerability, error) {
			scanInvoked = true
			if spec.Error != "" {
				return nil, errors.New(spec.Error)
			}

			vulnerabilities := make([]types.Vulnerability, 0, spec.VulnerabilityCount)
			for i := 0; i < spec.VulnerabilityCount; i++ {
				vulnerabilities = append(vulnerabilities, types.Vulnerability{
					PackageName: fmt.Sprintf("example.com/pkg-%d", i),
					Language:    "go",
				})
			}
			return vulnerabilities, nil
		}

		return scanFn, &scanInvoked
	}

	DescribeTable("execute verify scan stage", func(tc testCase) {
		var source sourceSpec
		testings.MustLoadYaml(tc.sourceFile, &source)

		var golden goldenSpec
		testings.MustLoadYaml(tc.goldenFile, &golden)

		projectPath := GinkgoT().TempDir()
		preMarkerPath := filepath.Join(projectPath, "pre.marker")
		postMarkerPath := filepath.Join(projectPath, "post.marker")

		pipeline := NewPipeline(&Config{
			ProjectPath: projectPath,
			DependaBotConfig: config.DependaBotConfig{
				Hooks: config.HooksConfig{
					PreVerifyScan:  renderHook(source.PreVerify, preMarkerPath, postMarkerPath),
					PostVerifyScan: renderHook(source.PostVerify, preMarkerPath, postMarkerPath),
				},
			},
		})

		scriptExecutor := NewScriptExecutor(projectPath)
		scanFn, scanInvoked := buildScanFn(source.Scan)

		shouldSkipUpdate, err := pipeline.runVerifyScanStage(scriptExecutor, scanFn)

		if golden.WantError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}

		Expect(shouldSkipUpdate).To(Equal(golden.ShouldSkipUpdate))
		Expect(fileExists(preMarkerPath)).To(Equal(golden.ExpectPreMarker))
		Expect(fileExists(postMarkerPath)).To(Equal(golden.ExpectPostMarker))
		Expect(*scanInvoked).To(Equal(golden.ExpectScanInvoked))
	},
		Entry("no verify hooks configured", testCase{
			description: "should skip verify stage when verify hooks are not configured",
			sourceFile:  "./testdata/verify_scan_stage/no_verify_hooks/source.yaml",
			goldenFile:  "./testdata/verify_scan_stage/no_verify_hooks/golden.yaml",
		}),
		Entry("verification scan passes", testCase{
			description: "should skip update flow when verification scan finds no vulnerabilities",
			sourceFile:  "./testdata/verify_scan_stage/verification_scan_passes/source.yaml",
			goldenFile:  "./testdata/verify_scan_stage/verification_scan_passes/golden.yaml",
		}),
		Entry("verification scan still has vulnerabilities", testCase{
			description: "should continue update flow when verification scan still finds vulnerabilities",
			sourceFile:  "./testdata/verify_scan_stage/verification_scan_has_vulnerabilities/source.yaml",
			goldenFile:  "./testdata/verify_scan_stage/verification_scan_has_vulnerabilities/golden.yaml",
		}),
		Entry("pre verify hook fails", testCase{
			description: "should fall back to regular flow when pre verify hook fails",
			sourceFile:  "./testdata/verify_scan_stage/pre_verify_hook_fails/source.yaml",
			goldenFile:  "./testdata/verify_scan_stage/pre_verify_hook_fails/golden.yaml",
		}),
		Entry("post verify cleanup fails", testCase{
			description: "should stop pipeline when post verify cleanup hook fails",
			sourceFile:  "./testdata/verify_scan_stage/post_verify_cleanup_fails/source.yaml",
			goldenFile:  "./testdata/verify_scan_stage/post_verify_cleanup_fails/golden.yaml",
		}),
	)
})
