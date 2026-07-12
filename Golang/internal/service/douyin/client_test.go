// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/douyin/client_test.go
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

package douyin

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

func TestSearchInfoByKeywordUsesPythonCompatibleSignedParams(t *testing.T) {
	t.Setenv("MCPRO_TEST_BLOCK_EXTERNAL_HTTP", "1")

	var signRequests []signsrv.DouyinSignRequest
	var apiQuery url.Values
	var mu sync.Mutex

	signServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/signsrv/v1/douyin/sign":
			var req signsrv.DouyinSignRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode douyin sign request: %v", err)
			}
			mu.Lock()
			signRequests = append(signRequests, req)
			mu.Unlock()
			writeDouyinTestJSON(w, map[string]any{
				"biz_code": 0,
				"msg":      "success",
				"isok":     true,
				"data": map[string]string{
					"a_bogus": "mock-a-bogus",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer signServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/aweme/v1/web/general/search/single/" {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		apiQuery = r.URL.Query()
		mu.Unlock()
		writeDouyinTestJSON(w, map[string]any{
			"status_code": 0,
			"data": []map[string]any{
				{
					"aweme_info": map[string]any{
						"aweme_id": "aweme_test",
						"desc":     "test aweme",
						"author":   map[string]any{"uid": "user_test"},
					},
				},
			},
			"extra": map[string]string{"logid": "log-test"},
		})
	}))
	defer apiServer.Close()

	cfg := &config.Config{
		RequestTimeOut: 3,
		PlatformEndpoints: &config.PlatformEndpointsConfig{
			DouyinApiURL: apiServer.URL,
		},
		SignSrvConfig: testSignSrvConfig(t, signServer.URL),
	}
	client := &Client{
		accountInfo: &models.AccountWithIpPoolModel{
			Account: &models.AccountInfoModel{Cookies: "sessionid=test-session"},
		},
		config:        cfg,
		signSrvClient: signsrv.NewSignClient(cfg),
		httpClient:    &http.Client{Timeout: 3 * time.Second},
		userAgent:     defaultUserAgent,
	}

	resp, err := client.SearchInfoByKeyword("deepseek", 0, "")
	if err != nil {
		t.Fatalf("search info by keyword: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected one mock search row, got %d", len(resp.Data))
	}

	mu.Lock()
	defer mu.Unlock()
	if len(signRequests) != 1 {
		t.Fatalf("expected search request to be preprocessed by SignSrv once, got %d", len(signRequests))
	}
	signReq := signRequests[0]
	if signReq.Uri != "/aweme/v1/web/general/search/single/" {
		t.Fatalf("unexpected sign uri: %q", signReq.Uri)
	}
	if signReq.UserAgent != defaultUserAgent {
		t.Fatalf("unexpected sign user agent: %q", signReq.UserAgent)
	}
	if !strings.Contains(signReq.Cookies, "sessionid=test-session") {
		t.Fatalf("expected cookies in sign request")
	}
	signedParams, err := url.ParseQuery(signReq.QueryParams)
	if err != nil {
		t.Fatalf("parse sign query params: %v", err)
	}
	requireDouyinSearchParams(t, signedParams)
	requireDouyinSearchParams(t, apiQuery)
	if apiQuery.Get("a_bogus") != "" {
		t.Fatalf("douyin search should match Python and omit a_bogus on general search, got %q", apiQuery.Get("a_bogus"))
	}
}

func requireDouyinSearchParams(t *testing.T, values url.Values) {
	t.Helper()
	expected := map[string]string{
		"search_channel":              "aweme_general",
		"keyword":                     "deepseek",
		"search_source":               "tab_search",
		"device_platform":             "webapp",
		"aid":                         "6383",
		"channel":                     "channel_pc_web",
		"pc_client_type":              "1",
		"version_code":                "170400",
		"version_name":                "17.4.0",
		"browser_name":                "Chrome",
		"browser_platform":            "MacIntel",
		"browser_version":             "135.0.0.0",
		"browser_online":              "true",
		"engine_name":                 "Blink",
		"engine_version":              "135.0.0.0",
		"os_name":                     "Mac+OS",
		"os_version":                  "10.15.7",
		"platform":                    "PC",
		"cookie_enabled":              "true",
		"publish_video_strategy_type": "2",
	}
	for key, want := range expected {
		if got := values.Get(key); got != want {
			t.Fatalf("query %s = %q, want %q; full query: %s", key, got, want, values.Encode())
		}
	}
	for _, key := range []string{"webid", "msToken"} {
		if values.Get(key) == "" {
			t.Fatalf("expected non-empty %s in query: %s", key, values.Encode())
		}
	}
}

func testSignSrvConfig(t *testing.T, serverURL string) *config.SignSrvConfig {
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

func writeDouyinTestJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
