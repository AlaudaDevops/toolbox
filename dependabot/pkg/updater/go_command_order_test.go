package updater

import (
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/testings"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GoUpdater go get ordering", func() {
	type commandOrderTestCase struct {
		description   string
		updatesFile   string
		argsGolden    string
		commandGolden string
	}

	type updatesSpec struct {
		Description string `yaml:"description"`
		Packages    []struct {
			PackageName     string `yaml:"packageName"`
			FixedVersion    string `yaml:"fixedVersion"`
			UseLatest       bool   `yaml:"useLatest"`
			ResolvedVersion string `yaml:"resolvedVersion"`
		} `yaml:"packages"`
	}

	type argsGoldenSpec struct {
		Description string   `yaml:"description"`
		Args        []string `yaml:"args"`
	}

	type commandGoldenSpec struct {
		Description string `yaml:"description"`
		Command     string `yaml:"command"`
	}

	mustLoadUpdates := func(path string) []packageUpdate {
		var spec updatesSpec
		testings.MustLoadYaml(path, &spec)

		updates := make([]packageUpdate, 0, len(spec.Packages))
		for _, pkg := range spec.Packages {
			updates = append(updates, packageUpdate{
				PackageName:     pkg.PackageName,
				FixedVersion:    pkg.FixedVersion,
				UseLatest:       pkg.UseLatest,
				ResolvedVersion: pkg.ResolvedVersion,
			})
		}

		return updates
	}

	DescribeTable("build commands deterministically", func(tc commandOrderTestCase) {
		updates := mustLoadUpdates(tc.updatesFile)

		var argsGolden argsGoldenSpec
		testings.MustLoadYaml(tc.argsGolden, &argsGolden)

		var cmdGolden commandGoldenSpec
		testings.MustLoadYaml(tc.commandGolden, &cmdGolden)

		updater := NewGoUpdater("/tmp/test", nil)

		Expect(updater.buildPackageArgs(updates)).To(Equal(argsGolden.Args))
		Expect(updater.buildBatchCommand(updates)).To(Equal(cmdGolden.Command))
	},
		Entry("golang extensions are sorted", commandOrderTestCase{
			description:   "should order golang.org/x packages alphabetically",
			updatesFile:   "./testdata/command_order/golang_extensions/source.yaml",
			argsGolden:    "./testdata/command_order/golang_extensions/args_golden.yaml",
			commandGolden: "./testdata/command_order/golang_extensions/command_golden.yaml",
		}),
		Entry("standard dependencies are sorted", commandOrderTestCase{
			description:   "should order normal packages alphabetically",
			updatesFile:   "./testdata/command_order/standard_dependencies/source.yaml",
			argsGolden:    "./testdata/command_order/standard_dependencies/args_golden.yaml",
			commandGolden: "./testdata/command_order/standard_dependencies/command_golden.yaml",
		}),
		Entry("mixed dependencies remain deterministic", commandOrderTestCase{
			description:   "should maintain alphabetical order for mixed packages",
			updatesFile:   "./testdata/command_order/mixed_dependencies/source.yaml",
			argsGolden:    "./testdata/command_order/mixed_dependencies/args_golden.yaml",
			commandGolden: "./testdata/command_order/mixed_dependencies/command_golden.yaml",
		}),
	)
})
