// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/e2e/runner.go
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

package e2e

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/cookiebridge"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/signsrv"
)

type MockPlatformOptions struct {
	Platform        string
	Accounts        []cookiebridge.Account
	PlatformHandler http.Handler
}

type MockPlatformRunner struct {
	BaseURL   string
	WorkDir   string
	DataDir   string
	ConfigDir string

	server *httptest.Server
}

func NewMockPlatformRunner(t testing.TB, opts MockPlatformOptions) *MockPlatformRunner {
	t.Helper()

	workDir, err := os.MkdirTemp("/tmp", "mediacrawlerpro-go-e2e-*")
	if err != nil {
		t.Fatalf("create e2e work dir: %v", err)
	}
	runner := &MockPlatformRunner{
		WorkDir:   workDir,
		DataDir:   filepath.Join(workDir, "data"),
		ConfigDir: filepath.Join(workDir, "config"),
	}
	for _, dir := range []string{runner.DataDir, runner.ConfigDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			_ = os.RemoveAll(workDir)
			t.Fatalf("create e2e dir %s: %v", dir, err)
		}
	}

	runner.server = httptest.NewServer(NewMockPlatformHandler(opts))
	runner.BaseURL = runner.server.URL
	t.Cleanup(runner.Close)
	return runner
}

func NewMockPlatformHandler(opts MockPlatformOptions) http.Handler {
	cookieBridgeHandler := cookiebridge.NewHandler(opts.Accounts...)
	signHandler := signsrv.NewHandler()
	platformHandler := opts.PlatformHandler
	if platformHandler == nil {
		platformHandler = http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isCookieBridgePath(r.URL.Path) {
			cookieBridgeHandler.ServeHTTP(w, r)
			return
		}
		if isSignServicePath(r.URL.Path) {
			signHandler.ServeHTTP(w, r)
			return
		}
		platformHandler.ServeHTTP(w, r)
	})
}

func (r *MockPlatformRunner) Close() {
	if r == nil {
		return
	}
	if r.server != nil {
		r.server.Close()
		r.server = nil
	}
	if r.WorkDir != "" {
		_ = os.RemoveAll(r.WorkDir)
		r.WorkDir = ""
	}
}

func isCookieBridgePath(path string) bool {
	switch path {
	case "/", "/ping", "/api/status", "/api/accounts", "/api/cookies/all":
		return true
	}
	return hasPathPrefix(path, "/api/accounts/") || hasPathPrefix(path, "/api/cookies/")
}

func isSignServicePath(path string) bool {
	return path == "/signsrv/pong" || hasPathPrefix(path, "/signsrv/v1/")
}

func hasPathPrefix(path, prefix string) bool {
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}
