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

package handler

// CommentedError represents an error where a comment has already been posted to the PR
type CommentedError struct {
	Err error
}

func (e *CommentedError) Error() string {
	return e.Err.Error()
}

func (e *CommentedError) Unwrap() error {
	return e.Err
}
