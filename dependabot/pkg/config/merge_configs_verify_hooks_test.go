package config

import (
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/testings"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConfigReader MergeConfigs verify hooks", func() {
	type hookSpec struct {
		Enabled         bool   `yaml:"enabled"`
		Script          string `yaml:"script"`
		Timeout         string `yaml:"timeout"`
		ContinueOnError bool   `yaml:"continueOnError"`
	}

	type configSpec struct {
		PreVerifyScan  hookSpec `yaml:"preVerifyScan"`
		PostVerifyScan hookSpec `yaml:"postVerifyScan"`
	}

	type sourceSpec struct {
		Description string     `yaml:"description"`
		Repo        configSpec `yaml:"repo"`
		CLI         configSpec `yaml:"cli"`
	}

	type expectedHookSpec struct {
		Enabled         bool   `yaml:"enabled"`
		Script          string `yaml:"script"`
		Timeout         string `yaml:"timeout"`
		ContinueOnError bool   `yaml:"continueOnError"`
	}

	type goldenSpec struct {
		Description    string           `yaml:"description"`
		PreVerifyScan  expectedHookSpec `yaml:"preVerifyScan"`
		PostVerifyScan expectedHookSpec `yaml:"postVerifyScan"`
	}

	type testCase struct {
		description string
		sourceFile  string
		goldenFile  string
	}

	buildHook := func(spec hookSpec) *HookConfig {
		if !spec.Enabled {
			return nil
		}

		return &HookConfig{
			Script:          spec.Script,
			Timeout:         spec.Timeout,
			ContinueOnError: spec.ContinueOnError,
		}
	}

	assertHook := func(hook *HookConfig, expected expectedHookSpec) {
		if !expected.Enabled {
			Expect(hook).To(BeNil())
			return
		}

		Expect(hook).NotTo(BeNil())
		Expect(hook.Script).To(Equal(expected.Script))
		Expect(hook.Timeout).To(Equal(expected.Timeout))
		Expect(hook.ContinueOnError).To(Equal(expected.ContinueOnError))
	}

	DescribeTable("merge verify hooks with cli priority", func(tc testCase) {
		var source sourceSpec
		testings.MustLoadYaml(tc.sourceFile, &source)

		var golden goldenSpec
		testings.MustLoadYaml(tc.goldenFile, &golden)

		repoConfig := &DependaBotConfig{
			Hooks: HooksConfig{
				PreVerifyScan:  buildHook(source.Repo.PreVerifyScan),
				PostVerifyScan: buildHook(source.Repo.PostVerifyScan),
			},
		}

		cliConfig := &DependaBotConfig{
			Hooks: HooksConfig{
				PreVerifyScan:  buildHook(source.CLI.PreVerifyScan),
				PostVerifyScan: buildHook(source.CLI.PostVerifyScan),
			},
		}

		reader := NewConfigReader()
		merged := reader.MergeConfigs(repoConfig, cliConfig)

		assertHook(merged.Hooks.PreVerifyScan, golden.PreVerifyScan)
		assertHook(merged.Hooks.PostVerifyScan, golden.PostVerifyScan)
	},
		Entry("repo config only", testCase{
			description: "should keep verify hooks from repo config when cli does not define them",
			sourceFile:  "./testdata/merge_configs_verify_hooks/repo_only/source.yaml",
			goldenFile:  "./testdata/merge_configs_verify_hooks/repo_only/golden.yaml",
		}),
		Entry("cli overrides pre verify", testCase{
			description: "should override preVerifyScan from cli while keeping repo postVerifyScan",
			sourceFile:  "./testdata/merge_configs_verify_hooks/cli_overrides_pre_verify/source.yaml",
			goldenFile:  "./testdata/merge_configs_verify_hooks/cli_overrides_pre_verify/golden.yaml",
		}),
		Entry("cli only", testCase{
			description: "should use verify hooks from cli config when repo does not define them",
			sourceFile:  "./testdata/merge_configs_verify_hooks/cli_only/source.yaml",
			goldenFile:  "./testdata/merge_configs_verify_hooks/cli_only/golden.yaml",
		}),
		Entry("both disabled", testCase{
			description: "should keep verify hooks empty when both repo and cli do not define them",
			sourceFile:  "./testdata/merge_configs_verify_hooks/both_disabled/source.yaml",
			goldenFile:  "./testdata/merge_configs_verify_hooks/both_disabled/golden.yaml",
		}),
	)
})
