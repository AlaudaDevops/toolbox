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
	"github.com/google/go-cmp/cmp"
)

func TestLoad(t *testing.T) {
	result, err := config.Load(context.Background(), "testdata/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	expected := &config.CopyConfig{
		Sources: []config.CopySource{
			{Name: "source-1", Dir: &config.Directory{Path: "../source-folder"}},
		},
		Target: &config.CopyTarget{
			CopyTo: "imported-docs",
			LinkTo: "docs",
			Links: []config.CopyLinks{
				{From: "public/<name>", Target: "public/<name>"},
				{From: "shared/crds", Target: "shared/crds/<name>"},
				{From: "zh/apis/kubernetes_apis", Target: "zh/apis/kubernetes_apis/<name>"},
				{From: "zh", Target: "zh/<name>"},
			},
		},
	}

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("Config mismatch (-want +got):\n%v", diff)
	}

}
