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
	"path/filepath"

	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/config"
	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/fscopy"
	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/logger"
	"github.com/spf13/cobra"
)

func NewCopyCommand(ctx context.Context) *cobra.Command {
	var (
		configFile string
		// clean      bool
	)
	cmd := &cobra.Command{
		Use:   "copy",
		Short: "Copy files from multiple sources to a target based on a configuration file",
		Long:  copyLongDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			if configFile == "" {
				return fmt.Errorf("--config is required")
			}
			return RunCopy(ctx, cmd, args, configFile)
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to the configuration file")
	// cmd.Flags().BoolVarP(&clean, "clean", "x", true, "Clean the target before copying. Defaults to true")
	return cmd
}

// RunCopy runs copy command logic to copy files from multiple sources to a target based on a configuration file
func RunCopy(ctx context.Context, cmd *cobra.Command, args []string, configFile string) error {
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
	// filesToCopy := map[string][]ifs.FileInfo{}
	// filesToLink := map[string][]ifs.LinkRequest{}
	for _, source := range config.Sources {
		log.Info("Listing files in source: ", source.Name)
		files, err := selector.ListFiles(ctx, source.Dir.Path)
		if err != nil {
			log.Error("error listing files in source ", source.Name, " error: ", err)
			return err
		}
		log.Debug("files in source ", source.Name, " are: ", files)
		// filesToCopy[source.Name] = files
		links := config.Target.Parse(source)
		log.Debug("links in source ", source.Name, " are: ", links)
		// filesToLink[source.Name] = links

		// TODO: skip real logic in dry-run mode later
		targetFolderForSource := filepath.Join(config.Target.CopyTo, source.Name)
		sourceFolder := source.Dir.Path

		log.Info("Copying files from source: ", sourceFolder, " to ", targetFolderForSource)
		if err := copier.Copy(ctx, sourceFolder, targetFolderForSource, files...); err != nil {
			log.Error("error copying files from source ", sourceFolder, " error: ", err)
			return err
		}
		log.Info("Linking files from source: ", sourceFolder, " to ", config.Target.LinkTo)
		if err := copier.Link(ctx, targetFolderForSource, config.Target.LinkTo, links...); err != nil {
			log.Error("error linking files from source ", sourceFolder, " error: ", err)
			return err
		}
	}
	return nil
}

const (
	copyLongDescription = `Copy files from multiple sources to a target based on a configuration file. The configuration file is a YAML file that defines the files to sync and the destination.
Example usage:

$ syncfiles copy --config config.yaml
Example configuration file: 
---
sources:
- name: <name> # 自定义的名称
  dir: # 目录信息 跟repository二选一
    path: ../tektoncd-pipeline

target:
  copyTo: imported-docs
  base: docs
  links:
  - from: public/<name> # 从下游视角，在docs目录下路径，这里使用多种文件的链接方式
    target: public/<name> # 在上游的docs目录下往哪里连接
  - from: shared/crds # 使用目录链接
    target: shared/crds/<name> # 使用 <name> placeholder, 来自 sources[].name
  - from: en/apis/kubernetes_apis
    target: en/apis/kubernetes_apis/<name>
  - from: en
    target: en/<name>
		`
)
