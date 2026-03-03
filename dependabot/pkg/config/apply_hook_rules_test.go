package config

import (
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/testings"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConfigReader ApplyHookRules", func() {
	type hookSpec struct {
		Enabled         bool   `yaml:"enabled"`
		Script          string `yaml:"script"`
		Timeout         string `yaml:"timeout"`
		ContinueOnError bool   `yaml:"continueOnError"`
	}

	type hooksSpec struct {
		PreVerifyScan  hookSpec `yaml:"preVerifyScan"`
		PostVerifyScan hookSpec `yaml:"postVerifyScan"`
		PreScan        hookSpec `yaml:"preScan"`
		PostScan       hookSpec `yaml:"postScan"`
		PreCommit      hookSpec `yaml:"preCommit"`
		PostCommit     hookSpec `yaml:"postCommit"`
	}

	type ruleWhenSpec struct {
		RepoNameGlob string `yaml:"repoNameGlob"`
	}

	type ruleSpec struct {
		Name     string       `yaml:"name"`
		When     ruleWhenSpec `yaml:"when"`
		Strategy string       `yaml:"strategy"`
		Hooks    hooksSpec    `yaml:"hooks"`
	}

	type sourceSpec struct {
		Description         string     `yaml:"description"`
		RepoURL             string     `yaml:"repoURL"`
		ExistingHooks       hooksSpec  `yaml:"existingHooks"`
		Rules               []ruleSpec `yaml:"rules"`
		ExpectErrorContains string     `yaml:"expectErrorContains"`
	}

	type goldenSpec struct {
		Description string    `yaml:"description"`
		Hooks       hooksSpec `yaml:"hooks"`
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

	buildRules := func(specs []ruleSpec) []HookRule {
		rules := make([]HookRule, 0, len(specs))
		for _, spec := range specs {
			rules = append(rules, HookRule{
				Name:     spec.Name,
				When:     HookRuleWhen{RepoNameGlob: spec.When.RepoNameGlob},
				Strategy: spec.Strategy,
				Hooks: HooksConfig{
					PreVerifyScan:  buildHook(spec.Hooks.PreVerifyScan),
					PostVerifyScan: buildHook(spec.Hooks.PostVerifyScan),
					PreScan:        buildHook(spec.Hooks.PreScan),
					PostScan:       buildHook(spec.Hooks.PostScan),
					PreCommit:      buildHook(spec.Hooks.PreCommit),
					PostCommit:     buildHook(spec.Hooks.PostCommit),
				},
			})
		}
		return rules
	}

	assertHook := func(actual *HookConfig, expected hookSpec) {
		if !expected.Enabled {
			Expect(actual).To(BeNil())
			return
		}

		Expect(actual).NotTo(BeNil())
		Expect(actual.Script).To(Equal(expected.Script))
		Expect(actual.Timeout).To(Equal(expected.Timeout))
		Expect(actual.ContinueOnError).To(Equal(expected.ContinueOnError))
	}

	DescribeTable("apply conditional hook rules for repository names", func(tc testCase) {
		var source sourceSpec
		testings.MustLoadYaml(tc.sourceFile, &source)

		cfg := &DependaBotConfig{
			Repo: RepoConfig{
				URL: source.RepoURL,
			},
			Hooks: HooksConfig{
				PreVerifyScan:  buildHook(source.ExistingHooks.PreVerifyScan),
				PostVerifyScan: buildHook(source.ExistingHooks.PostVerifyScan),
				PreScan:        buildHook(source.ExistingHooks.PreScan),
				PostScan:       buildHook(source.ExistingHooks.PostScan),
				PreCommit:      buildHook(source.ExistingHooks.PreCommit),
				PostCommit:     buildHook(source.ExistingHooks.PostCommit),
				Rules:          buildRules(source.Rules),
			},
		}

		reader := NewConfigReader()
		err := reader.ApplyHookRules(cfg)

		if source.ExpectErrorContains != "" {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(source.ExpectErrorContains))
			return
		}

		Expect(err).NotTo(HaveOccurred())

		var golden goldenSpec
		testings.MustLoadYaml(tc.goldenFile, &golden)

		assertHook(cfg.Hooks.PreVerifyScan, golden.Hooks.PreVerifyScan)
		assertHook(cfg.Hooks.PostVerifyScan, golden.Hooks.PostVerifyScan)
		assertHook(cfg.Hooks.PreScan, golden.Hooks.PreScan)
		assertHook(cfg.Hooks.PostScan, golden.Hooks.PostScan)
		assertHook(cfg.Hooks.PreCommit, golden.Hooks.PreCommit)
		assertHook(cfg.Hooks.PostCommit, golden.Hooks.PostCommit)
	},
		Entry("match tekton repository and fill empty hooks", testCase{
			description: "should inject verify hooks for repositories starting with tektoncd when hooks are empty",
			sourceFile:  "./testdata/apply_hook_rules/match_fill_empty/source.yaml",
			goldenFile:  "./testdata/apply_hook_rules/match_fill_empty/golden.yaml",
		}),
		Entry("skip non matching repository", testCase{
			description: "should keep hooks unchanged for non tekton repositories",
			sourceFile:  "./testdata/apply_hook_rules/non_match/source.yaml",
			goldenFile:  "./testdata/apply_hook_rules/non_match/golden.yaml",
		}),
		Entry("fill empty keeps existing hook", testCase{
			description: "should preserve existing preVerifyScan when strategy is fillEmpty",
			sourceFile:  "./testdata/apply_hook_rules/fill_empty_keep_existing/source.yaml",
			goldenFile:  "./testdata/apply_hook_rules/fill_empty_keep_existing/golden.yaml",
		}),
		Entry("override replaces existing hook", testCase{
			description: "should override existing preVerifyScan when strategy is override",
			sourceFile:  "./testdata/apply_hook_rules/override_existing/source.yaml",
			goldenFile:  "./testdata/apply_hook_rules/override_existing/golden.yaml",
		}),
		Entry("match ssh style repository url", testCase{
			description: "should extract repository name from ssh style URL and apply matching rule",
			sourceFile:  "./testdata/apply_hook_rules/match_ssh_url/source.yaml",
			goldenFile:  "./testdata/apply_hook_rules/match_ssh_url/golden.yaml",
		}),
		Entry("invalid repository glob", testCase{
			description: "should return error when repoNameGlob pattern is invalid",
			sourceFile:  "./testdata/apply_hook_rules/invalid_glob/source.yaml",
		}),
	)
})
