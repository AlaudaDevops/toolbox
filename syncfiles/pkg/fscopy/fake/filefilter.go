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
package fake

import (
	"context"

	"io/fs"

	ifs "github.com/AlaudaDevops/toolbox/syncfiles/pkg/fs"
)

type FakeFileFilter struct {
	Allowed bool
	Err     error

	FunWalkDir func(ctx context.Context, path string, d fs.DirEntry, err error) error
}

func (f *FakeFileFilter) IsFileAllowed(ctx context.Context, info ifs.FileInfo) (bool, error) {
	return f.Allowed, f.Err
}

func (f *FakeFileFilter) WalkDirFunc(ctx context.Context, path string, d fs.DirEntry, err error) error {
	if f.FunWalkDir != nil {
		return f.FunWalkDir(ctx, path, d, err)
	}
	return f.Err
}
