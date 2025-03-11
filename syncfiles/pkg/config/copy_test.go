/*
	Copyright 2025 AlaudaDevops authors

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
package config_test

import (
	"context"
	"testing"

	"github.com/AlaudaDevops/toolbox/syncfiles/pkg/config"
	ifs "github.com/AlaudaDevops/toolbox/syncfiles/pkg/fs"
	"github.com/google/go-cmp/cmp"
)

func TestCopyConfig_Parse(t *testing.T) {
	configFile, err := config.Load(context.Background(), "testdata/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	expected := []ifs.LinkRequest{
		{Source: "public/source-1", Destination: "public/source-1"},
		{Source: "shared/crds", Destination: "shared/crds/source-1"},
		{Source: "zh/apis/kubernetes_apis", Destination: "zh/apis/kubernetes_apis/source-1"},
		{Source: "zh", Destination: "zh/source-1"},
	}

	result := configFile.Target.Parse(configFile.Sources[0])

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("Link requests mismatch (-want +got):\n%v", diff)
	}
}
