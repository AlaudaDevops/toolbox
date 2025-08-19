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

import (
	"fmt"

	"github.com/AlaudaDevops/toolbox/pr-cli/pkg/messages"
)

// HandleRebase rebases the PR branch
func (h *PRHandler) HandleRebase() error {
	h.Logger.Info("Rebasing PR")

	if err := h.client.RebasePR(); err != nil {
		message := fmt.Sprintf(messages.RebaseFailedTemplate, err)
		if postErr := h.client.PostComment(message); postErr != nil {
			h.Logger.Errorf("Failed to post rebase error comment: %v", postErr)
			return fmt.Errorf("rebase failed: %w", err)
		}
		return &CommentedError{Err: err}
	}

	return h.client.PostComment(messages.RebaseSuccessTemplate)
}
