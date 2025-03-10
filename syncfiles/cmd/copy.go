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

	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/logger"
	"github.com/spf13/cobra"
)

func NewCopyCommand(ctx context.Context) *cobra.Command {
	var (
		configFile string
		clean      bool
	)
	cmd := &cobra.Command{
		Use:   "copy",
		Short: "Copy files from multiple sources to a target based on a configuration file",
		Long:  copyLongDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			log := logger.GetLogger(ctx)
			log.Debug("Copy command called")
			log.Info("This is an info log")
			return fmt.Errorf("implement me")
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to the configuration file")
	cmd.Flags().BoolVarP(&clean, "clean", "x", true, "Clean the target before copying. Defaults to true")
	return cmd
}

const (
	copyLongDescription = `Copy files from multiple sources to a target based on a configuration file. The configuration file is a YAML file that defines the files to sync and the destination.
Example usage:

$ syncfiles copy --config config.yaml
Example configuration file: 
---
sources:
- name: <name> # 自定义的名称
  repository: # 代码信息, 动态克隆代码进行复制
    url: https://github.com/alaudadevops/tektoncd-pipeline
    revision: main

  dir: # 目录信息 跟repository二选一
    path: ../tektoncd-pipeline

target:
  copyTo: imported-docs
  base: docs
  links:
  - from: public/*.png,public/*.svg # 从下游视角，在docs目录下路径，这里使用多种文件的链接方式
    target: public/ # 在上游的docs目录下往哪里连接
  - from: shared/crds # 使用目录链接
    target: shared/crds/<name> # 使用 <name> placeholder, 来自 sources[].name
  - from: zh/apis/kubernetes_apis
    target: zh/apis/kubernetes_apis/<name>
  - from: zh
    target: zh/<name>
		`
)
