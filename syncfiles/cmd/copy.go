/*    Copyright 2025 AlaudaDevops authors

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
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/config"
	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/fs"
	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/fscopy"
	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/logger"
	"github.com/spf13/cobra"
)

func NewCopyCommand(ctx context.Context) *cobra.Command {
	var (
		configFile   string
		cleanTarget  bool
		copyDirectly bool
		skipLink     bool
	)
	cmd := &cobra.Command{
		Use:   "copy",
		Short: "Copy files from multiple sources to a target based on a configuration file",
		Long:  copyLongDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			if configFile == "" {
				return fmt.Errorf("--config is required")
			}
			if copyDirectly {
				return fmt.Errorf("--copy-directly is not supported yet")
			}
			return RunCopy(ctx, cmd, args, configFile, cleanTarget, skipLink)
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to the configuration file")
	cmd.Flags().BoolVarP(&cleanTarget, "clean", "x", true, "Clean the target before copying. Defaults to true")
	cmd.Flags().BoolVarP(&copyDirectly, "copy-directly", "d", false, "Copy files directly from source to target. Defaults to false")
	cmd.Flags().BoolVarP(&skipLink, "skip-link", "s", false, "Skip creating symbolic links. Defaults to false")
	return cmd
}

// RunCopy runs copy command logic to copy files from multiple sources to a target based on a configuration file
func RunCopy(ctx context.Context, cmd *cobra.Command, args []string, configFile string, cleanTarget bool, skipLink bool) error {
	log := logger.GetLogger(ctx)

	config, err := config.Load(ctx, configFile)
	if err != nil {
		log.Error("error loading config: ", err)
		return err
	}
	if config.Target == nil {
		config.Target = config.Target.Default()
	}

	if err := config.Validate(ctx); err != nil {
		log.Error("error validating config: ", err)
		return err
	}

	copier := fscopy.FileSystemCopier{}
	selector := fscopy.FileSystemSelector{}
	for _, source := range config.Sources {
		log.Info("Listing files in source: ", source.Name)
		files, err := selector.ListFiles(ctx, source.Dir.Path)
		if err != nil {
			log.Error("error listing files in source ", source.Name, " error: ", err)
			return err
		}
		log.Debug("files in source ", source.Name, " are: ", files)
		links := config.Target.Parse(source)
		log.Debug("links in source ", source.Name, " are: ", links)

		// TODO: skip real logic in dry-run mode later
		targetFolderForSource := filepath.Join(config.Target.CopyTo, source.Name)
		sourceFolder := source.Dir.Path

		if cleanTarget {
			log.Info("Cleaning target folder: ", targetFolderForSource)
			if err := os.RemoveAll(targetFolderForSource); err != nil {
				log.Error("error cleaning target folder ", targetFolderForSource, " error: ", err)
			}
		}
		log.Info("Copying files from source: ", sourceFolder, " to ", targetFolderForSource)
		if err := copier.Copy(ctx, sourceFolder, targetFolderForSource, files...); err != nil {
			log.Error("error copying files from source ", sourceFolder, " error: ", err)
			return err
		}
		if !skipLink {
			log.Info("Linking files from source: ", sourceFolder, " to ", config.Target.LinkTo)
			if err := copier.Link(ctx, targetFolderForSource, config.Target.LinkTo, links...); err != nil {
				log.Error("error linking files from source ", sourceFolder, " error: ", err)
				return err
			}
		}
	}
	return nil
}

func RunCopyDirectly(ctx context.Context, cmd *cobra.Command, args []string, configFile string) error {
	log := logger.GetLogger(ctx)
	config, err := config.Load(ctx, configFile)
	if err != nil {
		log.Error("error loading config: ", err)
		return err
	}
	if config.Target == nil {
		config.Target = config.Target.Default()
	}

	// if err := config.Validate(ctx); err != nil {
	// 	log.Error("error validating config: ", err)
	// 	return err
	// }
	// copier := fscopy.FileSystemCopier{}
	selector := fscopy.FileSystemSelector{}
	for _, source := range config.Sources {
		log.Info("Listing files in source: ", source.Name)
		files, err := selector.ListFiles(ctx, source.Dir.Path)
		if err != nil {
			log.Error("error listing files in source ", source.Name, " error: ", err)
			return err
		}
		log.Debug("files in source ", source.Name, " are: ", files)
		targetFolders := config.Target.Parse(source)
		log.Debug("target folders in source ", source.Name, " are: ", targetFolders)

		// TODO: skip real logic in dry-run mode later
		// targetFolderForSource := filepath.Join(config.Target.CopyTo, source.Name)
		// sourceFolder := source.Dir.Path
		for _, folders := range targetFolders {
			sourceFolder := filepath.Join(source.Dir.Path, folders.Source)
			targetFolder := filepath.Join(config.Target.CopyTo, folders.Destination)
			filesToCopy := fs.FilterBySource(sourceFolder, files...)
			log.Info("Copying files from source: ", sourceFolder, " to ", targetFolder, " files: ", filesToCopy)
			// if err := copier.Copy(ctx, sourceFolder, targetFolder, files...); err != nil {
			// 	log.Error("error copying files from source ", sourceFolder, " error: ", err)
			// 	return err
			// }
		}

	}
	return nil
}

const (
	copyLongDescription = `Copy files from multiple sources to a target based on a configuration file. The configuration file is a YAML file that defines the files to sync and the destination.
Example usage:

$ syncfiles copy --config syncfiles.yaml
Example configuration file: 
---
sources:
- name: <name> # provided name for the source
  dir: # directory information, either dir or repository is required
    path: ../tektoncd-pipeline

target:
  copyTo: imported-docs # destination directory for copied files
  linkTo: docs # base directory for symbolic links
  links:
  - from: public/<name> # from downstream perspective, path in docs directory, here use multiple file link
    target: public/<name> # in upstream docs directory, where to connect
  - from: shared/crds # use directory link
    target: shared/crds/<name> # use <name> placeholder, comes from sources[].name
  - from: en/apis/kubernetes_apis
    target: en/apis/kubernetes_apis/<name>
  - from: en
    target: en/<name>
  - from: zh/apis/kubernetes_apis
    target: zh/apis/kubernetes_apis/<name>
  - from: zh
    target: zh/<name>
---
`
)
