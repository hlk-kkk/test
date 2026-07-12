// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/httpclient/client_test.go
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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewHttpClientWithTimeout(t *testing.T) {
	client, err := NewHttpClient(10*time.Second, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client.client.Timeout != 10*time.Second {
		t.Errorf("expected %v, got %v", 10*time.Second, client.client.Timeout)
	}
}

func TestSetHeader(t *testing.T) {
	client, _ := NewHttpClient(10*time.Second, "")
	client.SetHeader("Content-Type", "application/json")
	if client.headers["Content-Type"] != "application/json" {
		t.Errorf("expected %s, got %s", "application/json", client.headers["Content-Type"])
	}
}

func TestGetRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, _ := NewHttpClient(10*time.Second, "")
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestPostRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, _ := NewHttpClient(10*time.Second, "")
	resp, err := client.Post(server.URL, []byte(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestDoRequestWithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected %s, got %s", "application/json", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, _ := NewHttpClient(10*time.Second, "")
	client.SetHeader("Content-Type", "application/json")
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestHTTPClientBlocksExternalRequestsWhenTestGuardEnabled(t *testing.T) {
	t.Setenv("MCPRO_TEST_BLOCK_EXTERNAL_HTTP", "1")

	client, err := NewHttpClient(100*time.Millisecond, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	resp, err := client.Get("http://example.test/not-a-mock")
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected external HTTP request to be blocked")
	}
	if !strings.Contains(err.Error(), "blocked external HTTP request") {
		t.Fatalf("expected network guard error, got %v", err)
	}
}

func TestHTTPClientAllowsLoopbackRequestsWhenTestGuardEnabled(t *testing.T) {
	t.Setenv("MCPRO_TEST_BLOCK_EXTERNAL_HTTP", "1")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewHttpClient(100*time.Millisecond, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("expected loopback mock server to be allowed, got %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}
}
