/*
Copyright 2024 The AlaudaDevops Authors.

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

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AlaudaDevops/pkg/command/io"
	"github.com/AlaudaDevops/pkg/command/root"
	"github.com/AlaudaDevops/toolbox/artifact-scanner/pkg/cmd"
)

func main() {
	ctx := context.Background()

	streams := io.MustGetIOStreams(ctx)
	ctx = io.WithIOStreams(ctx, streams)

	command := root.NewRootCommand(ctx, "artifact-scanner",
		cmd.ScanCmd,
	)

	if err := command.Execute(); err != nil {
		fmt.Printf("command failed: %v", err)
		os.Exit(1)
	}
}
