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

// Package notice provides notification functionality for DependaBot
package notice

import (
	"fmt"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/config"
	"github.com/sirupsen/logrus"
)

// NewNotifier creates a new notifier based on the configuration
func NewNotifier(noticeConfig config.NoticeConfig) (Notifier, error) {
	switch noticeConfig.Type {
	case "wecom", "wechat":
		webhookURL, ok := noticeConfig.Params["webhook_url"].(string)
		if !ok || webhookURL == "" {
			return nil, fmt.Errorf("webhook_url is required for WeChat Work notifier")
		}
		logrus.Debug("Creating WeChat Work notifier")
		return NewWeComNotifier(webhookURL), nil
	case "":
		// No notification configured
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported notification type: %s", noticeConfig.Type)
	}
}

// IsNotificationEnabled checks if notification is enabled in the configuration
func IsNotificationEnabled(noticeConfig config.NoticeConfig) bool {
	return noticeConfig.Type != ""
}
