// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/httpclient/config_test.go
// GitHub: https://github.com/NanmiCoder
// Licensed under NON-COMMERCIAL LEARNING LICENSE 1.1
//
// 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
// 1. 不得用于任何商业用途。
// 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
// 3. 不得进行大规模爬取或对平台造成运营干扰。
// 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
// 5. 不得用于任何非法或不当的用途。
//
// 详细许可条款请参阅项目根目录下的LICENSE文件。
// 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。

package httpclient

import (
	"testing"
	"time"
)

func TestHeaderIsSetCorrectly(t *testing.T) {
	config := &ClientConfig{Headers: make(map[string]string)}
	config.SetHeader("Content-Type", "application/json")
	if config.GetHeader("Content-Type") != "application/json" {
		t.Errorf("expected %s, got %s", "application/json", config.GetHeader("Content-Type"))
	}
}

func TestHeaderIsUpdatedCorrectly(t *testing.T) {
	config := &ClientConfig{Headers: make(map[string]string)}
	config.SetHeader("Content-Type", "application/json")
	config.SetHeader("Content-Type", "text/plain")
	if config.GetHeader("Content-Type") != "text/plain" {
		t.Errorf("expected %s, got %s", "text/plain", config.GetHeader("Content-Type"))
	}
}

func TestHeaderIsRetrievedCorrectly(t *testing.T) {
	config := &ClientConfig{Headers: make(map[string]string)}
	config.SetHeader("Authorization", "Bearer token")
	if config.GetHeader("Authorization") != "Bearer token" {
		t.Errorf("expected %s, got %s", "Bearer token", config.GetHeader("Authorization"))
	}
}

func TestCookieHeaderIsRetrievedCorrectly(t *testing.T) {
	config := &ClientConfig{Headers: make(map[string]string)}
	config.SetHeader("Cookie", "sessionId=abc123")
	if config.GetCookie() != "sessionId=abc123" {
		t.Errorf("expected %s, got %s", "sessionId=abc123", config.GetCookie())
	}
}

func TestTimeoutIsSetCorrectly(t *testing.T) {
	config := &ClientConfig{TimeOut: 30 * time.Second}
	if config.TimeOut != 30*time.Second {
		t.Errorf("expected %v, got %v", 30*time.Second, config.TimeOut)
	}
}

func TestClientConfig_GetHeaders(t *testing.T) {
	config := &ClientConfig{Headers: make(map[string]string)}
	config.SetHeader("Content-Type", "application/json")
	config.SetHeader("Authorization", "Bearer token")

	headers := config.GetHeaders()
	if len(headers) != 2 {
		t.Errorf("expected %d, got %d", 2, len(headers))
	}

	if headers["Content-Type"] != "application/json" {
		t.Errorf("expected %s, got %s", "application/json", headers["Content-Type"])
	}

	if headers["Authorization"] != "Bearer token" {
		t.Errorf("expected %s, got %s", "Bearer token", headers["Authorization"])
	}
}
