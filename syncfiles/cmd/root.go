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
// Package cmd contains all the cli commands for the toolbox
package cmd

import (
	"context"

	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/logger"
	"github.com/spf13/cobra"
)

// NewRootCommand create the basic command endpoint for the syncfiles tool
func NewRootCommand(ctx context.Context, name string) *cobra.Command {
	if name == "" {
		name = "syncfiles"
	}
	// ogLevel used in logger
	logLevel := &logger.LogLeveler{}
	// initializing the logger
	log := logger.NewLoggerFromContext(ctx, logLevel)
	ctx = logger.WithLogger(ctx, log)
	cmd := &cobra.Command{
		Use:   name,
		Short: "Sync files command based on a configuration file",
		Long:  rootLongDescription,
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return log.Sync()
		},
	}
	// global flags
	cmd.PersistentFlags().StringVarP(&logLevel.Level, "log-level", "l", "info", "Set the logging level (debug, info, warn, error, panic, fatal)")

	// Add subcommands
	cmd.AddCommand(NewCopyCommand(ctx))

	return cmd
}

const (
	rootLongDescription = `Sync files cli for complex file operations`
)
