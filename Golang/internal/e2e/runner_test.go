// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/e2e/runner_test.go
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
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/cookiebridge"
	"github.com/stretchr/testify/require"
)

func TestMockPlatformRunnerStartsServicesUnderTmp(t *testing.T) {
	runner := NewMockPlatformRunner(t, MockPlatformOptions{
		Platform: "xhs",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "runner-client",
				Connected: true,
				Nicknames: map[string]string{"xhs": "runner-xhs"},
				Cookies:   map[string]string{"xhs": "web_session=runner; a1=runner"},
			},
		},
		PlatformHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/mock-platform" {
				http.NotFound(w, r)
				return
			}
			writeTestJSON(t, w, map[string]any{"ok": true, "platform": "xhs"})
		}),
	})
	defer runner.Close()

	require.True(t, strings.HasPrefix(runner.WorkDir, filepath.Clean("/tmp")+string(filepath.Separator)))
	require.True(t, strings.HasPrefix(runner.DataDir, runner.WorkDir))
	require.True(t, strings.HasPrefix(runner.ConfigDir, runner.WorkDir))
	require.NotEmpty(t, runner.BaseURL)

	ping := getTestJSON(t, runner.BaseURL+"/ping")
	require.True(t, ping["isok"].(bool))
	require.Equal(t, "pong", ping["data"].(map[string]any)["message"])

	cookie := getTestJSON(t, runner.BaseURL+"/api/cookies/xhs?client_id=runner-client")
	require.True(t, cookie["isok"].(bool))
	require.Equal(t, "web_session=runner; a1=runner", cookie["data"].(map[string]any)["cookies"])

	sign := postTestJSON(t, runner.BaseURL+"/signsrv/v1/xhs/sign", map[string]any{
		"uri":     "/api/sns/web/v1/search/notes",
		"data":    map[string]any{"keyword": "runner"},
		"cookies": "web_session=runner",
	})
	require.True(t, sign["isok"].(bool))
	require.Equal(t, "mock-x-s", sign["data"].(map[string]any)["x_s"])

	platform := getTestJSON(t, runner.BaseURL+"/api/mock-platform")
	require.True(t, platform["ok"].(bool))
	require.Equal(t, "xhs", platform["platform"])
}

func TestMockPlatformRunnerRunsCompiledCLIAndReadsOutputs(t *testing.T) {
	var searchCalls, feedCalls int
	runner := NewMockPlatformRunner(t, MockPlatformOptions{
		Platform: "xhs",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "runner-cli-client",
				Connected: true,
				Nicknames: map[string]string{"xhs": "runner-cli-xhs"},
				Cookies:   map[string]string{"xhs": "web_session=runner-cli; a1=runner-cli"},
			},
		},
		PlatformHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/sns/web/v1/user/selfinfo":
				writeTestJSON(t, w, map[string]any{
					"code":    0,
					"success": true,
					"msg":     "ok",
					"data": map[string]any{
						"result": map[string]any{"success": true, "message": "ok", "code": 0},
					},
				})
			case "/api/sns/web/v1/search/notes":
				searchCalls++
				t.Fatalf("xhs hot_search placeholder should not call search endpoint")
			case "/api/sns/web/v1/feed":
				feedCalls++
				t.Fatalf("xhs hot_search placeholder should not call feed endpoint")
			default:
				t.Fatalf("unexpected platform request: %s %s", r.Method, r.URL.String())
			}
		}),
	})

	repoRoot := RepoRoot(t)
	host, port := runner.HostPort(t)
	runner.WriteConfig(t, `
platform: "xhs"
keywords:
  - "unused"
crawlerType: "hot_search"
saveDataOption: "json"
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: "`+runner.BaseURL+`"
dataDir: "`+runner.DataDir+`"
startPage: 1
crawlerMaxNotesCount: 1
maxConcurrencyNum: 1
requestTimeOut: 2
enableGetComments: false
enableGetSubComments: false
enableCheckpoint: false
perNoteMaxComments: 2
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: "`+host+`"
  signSrvPort: `+port+`
platformEndpoints:
  xhsApiUrl: "`+runner.BaseURL+`"
  xhsIndexUrl: "`+runner.BaseURL+`"
`)

	binaryPath := BuildCLI(t, repoRoot)
	output := runner.RunCLI(t, binaryPath, RunOptions{Timeout: 20 * time.Second})

	require.NotContains(t, string(output), "crawler command failed")
	require.Zero(t, searchCalls)
	require.Zero(t, feedCalls)

	xhsDir := filepath.Join(runner.DataDir, "xhs")
	entries, err := os.ReadDir(xhsDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read xhs output dir: %v\ncommand output:\n%s", err, output)
	}
	require.Empty(t, entries)

	jsonPath := filepath.Join(runner.WorkDir, "sample.json")
	require.NoError(t, os.WriteFile(jsonPath, []byte(`[{"id":"json-row"}]`), 0o644))
	require.Equal(t, "json-row", ReadJSONRows(t, jsonPath, output)[0]["id"])

	csvPath := filepath.Join(runner.WorkDir, "sample.csv")
	csvFile, err := os.Create(csvPath)
	require.NoError(t, err)
	require.NoError(t, csv.NewWriter(csvFile).WriteAll([][]string{{"\ufeffid", "value"}, {"csv-row", "ok"}}))
	require.NoError(t, csvFile.Close())
	require.Equal(t, "csv-row", CSVCellAt(ReadCSVRecords(t, csvPath, output), 1, "id"))

	sqlitePath := filepath.Join(runner.WorkDir, "sample.db")
	db := OpenSQLite(t, sqlitePath, output)
	require.NoError(t, db.WithContext(context.Background()).Exec("CREATE TABLE sample (id TEXT)").Error)
	require.NoError(t, db.WithContext(context.Background()).Exec("INSERT INTO sample(id) VALUES (?)", "sqlite-row").Error)
	var got string
	require.NoError(t, db.WithContext(context.Background()).Raw("SELECT id FROM sample").Scan(&got).Error)
	require.Equal(t, "sqlite-row", got)
}

func TestMockPlatformRunnerWritesCrawlerConfigForCLI(t *testing.T) {
	var searchCalls, feedCalls int
	runner := NewMockPlatformRunner(t, MockPlatformOptions{
		Platform: "xhs",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "config-client",
				Connected: true,
				Nicknames: map[string]string{"xhs": "config-xhs"},
				Cookies:   map[string]string{"xhs": "web_session=config; a1=config"},
			},
		},
		PlatformHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/sns/web/v1/user/selfinfo":
				writeTestJSON(t, w, map[string]any{
					"code":    0,
					"success": true,
					"msg":     "ok",
					"data": map[string]any{
						"result": map[string]any{"success": true, "message": "ok", "code": 0},
					},
				})
			case "/api/sns/web/v1/search/notes":
				searchCalls++
				t.Fatalf("xhs hot_search placeholder should not call search endpoint")
			case "/api/sns/web/v1/feed":
				feedCalls++
				t.Fatalf("xhs hot_search placeholder should not call feed endpoint")
			default:
				t.Fatalf("unexpected platform request: %s %s", r.Method, r.URL.String())
			}
		}),
	})

	configPath := runner.WriteCrawlerConfig(t, CrawlerConfigOptions{
		Platform:                  "xhs",
		Keywords:                  []string{"alpha", "beta"},
		CrawlerType:               "hot_search",
		SaveDataOption:            "json",
		CrawlerMaxNotesCount:      1,
		HotSearchMaxItemsCount:    2,
		HotSearchPerItemContent:   1,
		EnableGetComments:         false,
		EnableGetSubComments:      false,
		EnableCheckpoint:          false,
		SpecifiedNoteIDs:          []string{"note-a", "note-b"},
		SpecifiedCreatorIDs:       []string{"creator-a"},
		PlatformEndpointOverrides: map[string]string{"xhsIndexUrl": runner.BaseURL + "/index"},
		ExtraConfigYAML:           "debugRequest: false\n",
	})
	body, err := os.ReadFile(configPath)
	require.NoError(t, err)
	configYAML := string(body)
	require.Contains(t, configYAML, `platform: "xhs"`)
	require.Contains(t, configYAML, `crawlerType: "hot_search"`)
	require.Contains(t, configYAML, `saveDataOption: "json"`)
	require.Contains(t, configYAML, `accountPoolSaveType: "cookie_bridge"`)
	require.Contains(t, configYAML, `cookieBridgeURL: "`+runner.BaseURL+`"`)
	require.Contains(t, configYAML, `dataDir: "`+runner.DataDir+`"`)
	require.Contains(t, configYAML, `sqliteDbPath: "`+filepath.Join(runner.WorkDir, "media_crawler.db")+`"`)
	require.Contains(t, configYAML, `  - "alpha"`)
	require.Contains(t, configYAML, `  - "note-a"`)
	require.Contains(t, configYAML, `  - "creator-a"`)
	require.Contains(t, configYAML, `xhsApiUrl: "`+runner.BaseURL+`"`)
	require.Contains(t, configYAML, `xhsIndexUrl: "`+runner.BaseURL+`/index"`)
	require.Contains(t, configYAML, `signSrvHost:`)
	require.Contains(t, configYAML, `signSrvPort:`)

	output := runner.RunCLI(t, BuildCLI(t, RepoRoot(t)), RunOptions{Timeout: 20 * time.Second})

	require.Zero(t, searchCalls)
	require.Zero(t, feedCalls)
	xhsDir := filepath.Join(runner.DataDir, "xhs")
	entries, err := os.ReadDir(xhsDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read xhs output dir: %v\ncommand output:\n%s", err, output)
	}
	require.Empty(t, entries)
}

func TestSQLiteInspectorsReadRowsByStableKeys(t *testing.T) {
	runner := NewMockPlatformRunner(t, MockPlatformOptions{Platform: "xhs"})
	output := []byte("mock command output")
	dbPath := filepath.Join(runner.WorkDir, "inspect.db")
	db := OpenSQLite(t, dbPath, output)
	require.NoError(t, db.Exec("CREATE TABLE sample (id TEXT PRIMARY KEY, title TEXT, count INTEGER)").Error)
	require.NoError(t, db.Exec("INSERT INTO sample(id, title, count) VALUES (?, ?, ?)", "b", "second", 2).Error)
	require.NoError(t, db.Exec("INSERT INTO sample(id, title, count) VALUES (?, ?, ?)", "a", "first", 1).Error)

	rows := ReadSQLiteRows(t, dbPath, "sample", "id", output)

	require.Len(t, rows, 2)
	require.Equal(t, "a", SQLiteCell(rows, 0, "id"))
	require.Equal(t, "first", SQLiteCell(rows, 0, "title"))
	require.Equal(t, "1", SQLiteCell(rows, 0, "count"))
	require.Equal(t, "b", SQLiteCell(rows, 1, "id"))
	require.Equal(t, "second", SQLiteCell(rows, 1, "title"))
	require.Equal(t, "2", SQLiteCell(rows, 1, "count"))
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(payload))
}

func getTestJSON(t *testing.T, requestURL string) map[string]any {
	t.Helper()
	resp, err := http.Get(requestURL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var payload map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	return payload
}

func postTestJSON(t *testing.T, requestURL string, body any) map[string]any {
	t.Helper()
	bodyReader := strings.NewReader(mustMarshalTestJSON(t, body))
	resp, err := http.Post(requestURL, "application/json", bodyReader)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var payload map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	return payload
}

func mustMarshalTestJSON(t *testing.T, value any) string {
	t.Helper()
	body, err := json.Marshal(value)
	require.NoError(t, err)
	return string(body)
}
