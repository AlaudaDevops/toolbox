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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AlaudaDevops/toolbox/dependabot/pkg/git"
	"github.com/AlaudaDevops/toolbox/dependabot/pkg/types"
	"github.com/sirupsen/logrus"
)

// WeComNotifier implements Notifier interface for WeChat Work (Enterprise WeChat)
type WeComNotifier struct {
	// webhookURL is the WeChat Work webhook URL
	webhookURL string
	// httpClient is the HTTP client for making requests
	httpClient *http.Client
}

// WeComMessageCard represents a WeChat Work message card
type WeComMessageCard struct {
	MsgType  string        `json:"msgtype"`
	Markdown WeComMarkdown `json:"markdown"`
}

// WeComMarkdown represents the markdown content for WeChat Work
type WeComMarkdown struct {
	Content string `json:"content"`
}

// NewWeComNotifier creates a new WeChat Work notifier
func NewWeComNotifier(webhookURL string) *WeComNotifier {
	return &WeComNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Notify sends a notification to WeChat Work
func (w *WeComNotifier) Notify(repoURL string, vulnFixResults types.VulnFixResults, pr types.PRInfo) error {
	logrus.Debugf("Sending WeChat Work notification for PR: %s", pr.Title)

	gitRepo, _ := git.ParseRepoURL(repoURL)
	message := w.buildMessage(gitRepo.String(), vulnFixResults, pr)
	messageCard := WeComMessageCard{
		MsgType: "markdown",
		Markdown: WeComMarkdown{
			Content: message,
		},
	}

	jsonData, err := json.Marshal(messageCard)
	if err != nil {
		return fmt.Errorf("failed to marshal WeChat Work message: %w", err)
	}

	req, err := http.NewRequest("POST", w.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send WeChat Work notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WeChat Work notification failed with status: %d", resp.StatusCode)
	}

	logrus.Infof("WeChat Work notification sent successfully for PR: %s", pr.Title)
	return nil
}

// buildMessage builds the markdown message for WeChat Work
func (w *WeComNotifier) buildMessage(componentName string, vulnFixResults types.VulnFixResults, pr types.PRInfo) string {
	var msg bytes.Buffer

	// Header
	msg.WriteString(fmt.Sprintf("# 🔐 %s 安全更新通知\n\n", componentName))

	// Summary
	totalCount := vulnFixResults.TotalVulnCount()
	fixedCount := vulnFixResults.FixedVulnCount()
	failedCount := vulnFixResults.FixFailedVulnCount()

	msg.WriteString("\n**📊 更新摘要：**\n")
	msg.WriteString(fmt.Sprintf("总漏洞数：（%d） 成功修复：（%d） 修复失败：（%d）\n", totalCount, fixedCount, failedCount))
	msg.WriteString("\n")

	// PR Information
	if pr.URL != "" {
		msg.WriteString("\n**🔗 Pull Request：**\n")
		msg.WriteString(fmt.Sprintf(" - 标题：%s\n", pr.Title))
		msg.WriteString(fmt.Sprintf(" - 链接：[%s](%s)\n\n", pr.URL, pr.URL))
	}

	// Fixed vulnerabilities
	if fixedCount > 0 {
		msg.WriteString("\n**✅ 已修复的漏洞：**\n")
		fixedVulns := vulnFixResults.FixedVulns()
		for i, vuln := range fixedVulns {
			if i >= 10 { // 限制显示数量
				msg.WriteString(fmt.Sprintf(" - 还有 %d 个漏洞已修复...\n", len(fixedVulns)-i))
				break
			}
			msg.WriteString(fmt.Sprintf(" - %s: %s → %s\n", vuln.PackageName, vuln.CurrentVersion, vuln.FixedVersion))
		}
		msg.WriteString("\n")
	}

	// Failed vulnerabilities
	if failedCount > 0 {
		msg.WriteString("**❌ 修复失败的漏洞：**\n")
		failedVulns := vulnFixResults.FixFailedVulns()
		for i, vuln := range failedVulns {
			if i >= 5 { // 限制显示数量
				msg.WriteString(fmt.Sprintf(" - 还有 %d 个漏洞修复失败...\n", len(failedVulns)-i))
				break
			}
			msg.WriteString(fmt.Sprintf("- %s: %s (需要手动处理)\n", vuln.PackageName, vuln.CurrentVersion))
		}
		msg.WriteString("\n")
	}
	return msg.String()
}
