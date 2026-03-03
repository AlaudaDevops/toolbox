package scanner

import (
	"os"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/settings"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/testings"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TrivyScanner cleanup", func() {
	type sourceSpec struct {
		Description     string `yaml:"description"`
		CleanupTempDirs bool   `yaml:"cleanupTempDirs"`
		WithTempDir     bool   `yaml:"withTempDir"`
	}

	type goldenSpec struct {
		Description        string `yaml:"description"`
		WantError          bool   `yaml:"wantError"`
		ExpectDirExists    bool   `yaml:"expectDirExists"`
		ExpectTempDirEmpty bool   `yaml:"expectTempDirEmpty"`
	}

	type testCase struct {
		description string
		sourceFile  string
		goldenFile  string
	}

	DescribeTable("cleanup temp directory based on global switch", func(tc testCase) {
		var source sourceSpec
		testings.MustLoadYaml(tc.sourceFile, &source)

		var golden goldenSpec
		testings.MustLoadYaml(tc.goldenFile, &golden)

		settings.SetCleanupTempDirsEnabled(source.CleanupTempDirs)
		DeferCleanup(func() {
			settings.SetCleanupTempDirsEnabled(true)
		})

		var tempDir string
		if source.WithTempDir {
			dir, err := os.MkdirTemp("", "trivy-cleanup-test-")
			Expect(err).NotTo(HaveOccurred())
			tempDir = dir
			DeferCleanup(func() {
				_ = os.RemoveAll(dir)
			})
		}

		scanner := &TrivyScanner{tempDir: tempDir}

		err := scanner.Cleanup()
		if golden.WantError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}

		dirExists := false
		if tempDir != "" {
			_, statErr := os.Stat(tempDir)
			dirExists = statErr == nil
		}
		Expect(dirExists).To(Equal(golden.ExpectDirExists))

		if golden.ExpectTempDirEmpty {
			Expect(scanner.tempDir).To(BeEmpty())
		} else {
			Expect(scanner.tempDir).To(Equal(tempDir))
		}
	},
		Entry("cleanup enabled", testCase{
			description: "should cleanup temp directory when global switch is enabled",
			sourceFile:  "./testdata/trivy_cleanup/cleanup_enabled/source.yaml",
			goldenFile:  "./testdata/trivy_cleanup/cleanup_enabled/golden.yaml",
		}),
		Entry("cleanup disabled", testCase{
			description: "should keep temp directory when global switch is disabled",
			sourceFile:  "./testdata/trivy_cleanup/cleanup_disabled/source.yaml",
			goldenFile:  "./testdata/trivy_cleanup/cleanup_disabled/golden.yaml",
		}),
		Entry("empty temp directory", testCase{
			description: "should return nil for empty temp directory",
			sourceFile:  "./testdata/trivy_cleanup/empty_temp_dir/source.yaml",
			goldenFile:  "./testdata/trivy_cleanup/empty_temp_dir/golden.yaml",
		}),
	)
})
