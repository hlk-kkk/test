// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/zhihu/client_test.go
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

package zhihu

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/rpc/signsrv"
)

func TestGetNotesByKeywordSignsPathQueryAndUsesPythonCompatibleHeaders(t *testing.T) {
	t.Setenv("MCPRO_TEST_BLOCK_EXTERNAL_HTTP", "1")

	var signRequests []signsrv.ZhihuSignRequest
	var apiHeaders http.Header
	var mu sync.Mutex

	signServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/signsrv/v1/zhihu/sign":
			var req signsrv.ZhihuSignRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode zhihu sign request: %v", err)
			}
			mu.Lock()
			signRequests = append(signRequests, req)
			mu.Unlock()
			writeZhihuTestJSON(w, map[string]any{
				"biz_code": 0,
				"msg":      "success",
				"isok":     true,
				"data": map[string]string{
					"x_zst_81": "mock-x-zst-81",
					"x_zse_96": "mock-x-zse-96",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer signServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/search_v3" {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		apiHeaders = r.Header.Clone()
		mu.Unlock()
		writeZhihuTestJSON(w, map[string]any{
			"data": []map[string]any{
				{
					"type": "search_result",
					"object": map[string]any{
						"type":  "answer",
						"id":    "answer_test",
						"title": "test answer",
						"author": map[string]any{
							"id":   "author_test",
							"name": "author",
						},
					},
				},
			},
		})
	}))
	defer apiServer.Close()

	cfg := &config.Config{
		RequestTimeOut: 3,
		PlatformEndpoints: &config.PlatformEndpointsConfig{
			ZhihuURL: apiServer.URL,
		},
		SignSrvConfig: testZhihuSignSrvConfig(t, signServer.URL),
	}
	client := &Client{
		accountInfo: &models.AccountWithIpPoolModel{
			Account: &models.AccountInfoModel{Cookies: "z_c0=test-cookie"},
		},
		config:     cfg,
		signClient: signsrv.NewSignClient(cfg),
		httpClient: &http.Client{Timeout: 3 * time.Second},
		userAgent:  defaultUserAgent,
	}

	contents, err := client.GetNotesByKeyword("deepseek", 1, 20)
	if err != nil {
		t.Fatalf("get notes by keyword: %v", err)
	}
	if len(contents) != 1 {
		t.Fatalf("expected one mock content row, got %d", len(contents))
	}

	mu.Lock()
	defer mu.Unlock()
	if len(signRequests) != 1 {
		t.Fatalf("expected one zhihu sign request, got %d", len(signRequests))
	}
	signReq := signRequests[0]
	if strings.HasPrefix(signReq.Uri, "http://") || strings.HasPrefix(signReq.Uri, "https://") {
		t.Fatalf("zhihu sign uri should be path+query like Python, got absolute uri %q", signReq.Uri)
	}
	if !strings.HasPrefix(signReq.Uri, "/api/v4/search_v3?") {
		t.Fatalf("unexpected zhihu sign uri: %q", signReq.Uri)
	}
	parsedSignURI, err := url.Parse(signReq.Uri)
	if err != nil {
		t.Fatalf("parse zhihu sign uri: %v", err)
	}
	query := parsedSignURI.Query()
	for key, want := range map[string]string{
		"q":             "deepseek",
		"offset":        "0",
		"limit":         "20",
		"search_source": "Filter",
		"time_interval": "",
		"sort":          "",
		"vertical":      "",
	} {
		if got := query.Get(key); got != want {
			t.Fatalf("sign query %s = %q, want %q; uri=%s", key, got, want, signReq.Uri)
		}
	}
	if !strings.Contains(signReq.Cookies, "z_c0=test-cookie") {
		t.Fatalf("expected cookies in zhihu sign request")
	}

	for key, want := range map[string]string{
		"x-zst-81":         "mock-x-zst-81",
		"x-zse-96":         "mock-x-zse-96",
		"x-zse-93":         "101_3_3.0",
		"x-api-version":    "3.0.91",
		"x-app-za":         "OS=Web",
		"x-requested-with": "fetch",
	} {
		if got := apiHeaders.Get(key); got != want {
			t.Fatalf("header %s = %q, want %q", key, got, want)
		}
	}
}

func testZhihuSignSrvConfig(t *testing.T, serverURL string) *config.SignSrvConfig {
	t.Helper()
	parsed, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse sign server url: %v", err)
	}
	host, portText, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("split sign server host port: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parse sign server port: %v", err)
	}
	return &config.SignSrvConfig{SignSrvHost: host, SignSrvPort: port}
}

func writeZhihuTestJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
