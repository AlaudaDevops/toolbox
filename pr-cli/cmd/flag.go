/*
Copyright 2025 The AlaudaDevops Authors.

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
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// EnvPrefix is the prefix for environment variables
const EnvPrefix = "PR"

var (
	outputFormat string
)

func init() {
	cobra.OnInitialize(initConfig)

	// Initialize PROption and add flags
	prOption = NewPROption()
	prOption.AddFlags(rootCmd.Flags())

	// Add global version flag
	rootCmd.Flags().BoolP("version", "v", false, "Show version information")

	// Add output format flag for version display
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format for version (text|json)")

	// Bind flags to viper for environment variable support
	viper.BindPFlags(rootCmd.Flags())
}

func initConfig() {
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}
