// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/other_platforms_live_e2e_test.go
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

package e2e_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/e2e"
)

type liveSearchPlatformCase struct {
	Platform     string
	DataPlatform string
	Keyword      string
	IDField      string
	UserField    string
	SQLiteTable  string
	Required     []string
	ExpectedRows int
}

type liveSearchRun struct {
	WorkDir    string
	DataDir    string
	SQLitePath string
	Output     []byte
}

type liveSearchConfig struct {
	SaveDataOption   string
	EnableCheckpoint bool
}

func TestCLIOtherPlatformsLiveSearchWithRealCookie(t *testing.T) {
	cookieBridgeURL := os.Getenv("MCPRO_OTHER_LIVE_COOKIE_BRIDGE_URL")
	if cookieBridgeURL == "" {
		t.Skip("set MCPRO_OTHER_LIVE_COOKIE_BRIDGE_URL to run live xhs/douyin/zhihu business tests")
	}
	if _, err := url.ParseRequestURI(cookieBridgeURL); err != nil {
		t.Fatalf("invalid MCPRO_OTHER_LIVE_COOKIE_BRIDGE_URL: %v", err)
	}

	binaryPath := e2e.BuildCLI(t, e2e.RepoRoot(t))
	baseEnv := []string{
		"COOKIE_BRIDGE_URL=" + cookieBridgeURL,
		"MCPRO_TEST_BLOCK_EXTERNAL_HTTP=0",
		"NO_PROXY=localhost,127.0.0.1",
		"no_proxy=localhost,127.0.0.1",
	}

	for _, tc := range []liveSearchPlatformCase{
		{
			Platform:     "xhs",
			DataPlatform: "xhs",
			Keyword:      envOrDefault("MCPRO_XHS_LIVE_KEYWORD", "deepseek"),
			IDField:      "note_id",
			UserField:    "user_id",
			SQLiteTable:  "xhs_note",
			Required:     []string{"note_id", "title", "source_keyword", "user_id", "nickname"},
			ExpectedRows: 1,
		},
		{
			Platform:     "dy",
			DataPlatform: "douyin",
			Keyword:      envOrDefault("MCPRO_DOUYIN_LIVE_KEYWORD", "deepseek"),
			IDField:      "aweme_id",
			UserField:    "user_id",
			SQLiteTable:  "douyin_aweme",
			Required:     []string{"aweme_id", "title", "source_keyword", "user_id", "nickname"},
		},
		{
			Platform:     "zhihu",
			DataPlatform: "zhihu",
			Keyword:      envOrDefault("MCPRO_ZHIHU_LIVE_KEYWORD", "deepseek"),
			IDField:      "content_id",
			UserField:    "user_id",
			SQLiteTable:  "zhihu_content",
			Required:     []string{"content_id", "title", "source_keyword", "user_id", "user_nickname"},
		},
	} {
		t.Run(tc.Platform, func(t *testing.T) {
			jsonRun := runLiveSearchCLI(t, binaryPath, baseEnv, tc, liveSearchConfig{SaveDataOption: "json"})
			jsonRows := readLiveSearchRows(t, jsonRun, tc)
			requireLiveSearchRows(t, tc, jsonRows, jsonRun.Output)

			csvRun := runLiveSearchCLI(t, binaryPath, baseEnv, tc, liveSearchConfig{SaveDataOption: "csv"})
			csvRecords := readLiveCSV(t, filepath.Join(csvRun.DataDir, tc.DataPlatform, "1_"+datedFile("search_contents", "csv")), csvRun.Output)
			if len(csvRecords) < 2 {
				t.Fatalf("live %s csv should contain header and at least one row, got %d\n%s", tc.Platform, len(csvRecords), csvRun.Output)
			}
			if got := e2e.CSVCellAt(csvRecords, 1, "source_keyword"); got != tc.Keyword {
				t.Fatalf("live %s csv source_keyword = %q, want %q\n%s", tc.Platform, got, tc.Keyword, csvRun.Output)
			}
			if e2e.CSVCellAt(csvRecords, 1, tc.IDField) == "" || e2e.CSVCellAt(csvRecords, 1, tc.UserField) == "" {
				t.Fatalf("live %s csv missing required id/user fields: %#v\n%s", tc.Platform, csvRecords[1], csvRun.Output)
			}

			sqliteRun := runLiveSearchCLI(t, binaryPath, baseEnv, tc, liveSearchConfig{SaveDataOption: "sqlite"})
			if count := countSQLiteRows(t, sqliteRun.SQLitePath, tc.SQLiteTable, sqliteRun.Output); count == 0 {
				t.Fatalf("live %s sqlite should save rows in %s\n%s", tc.Platform, tc.SQLiteTable, sqliteRun.Output)
			}

			assertLiveSearchCheckpointResume(t, binaryPath, baseEnv, tc)
		})
	}
}

func runLiveSearchCLI(t *testing.T, binaryPath string, env []string, tc liveSearchPlatformCase, opts liveSearchConfig) liveSearchRun {
	t.Helper()
	workDir, err := os.MkdirTemp("/tmp", "mcpro-"+tc.Platform+"-live-e2e-*")
	if err != nil {
		t.Fatalf("create live %s work dir: %v", tc.Platform, err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(workDir) })
	dataDir := filepath.Join(workDir, "data")
	sqlitePath := filepath.Join(workDir, "media_crawler.db")
	return runLiveSearchCLIInWorkDir(t, binaryPath, env, tc, opts, workDir, dataDir, sqlitePath)
}

func runLiveSearchCLIInWorkDir(t *testing.T, binaryPath string, env []string, tc liveSearchPlatformCase, opts liveSearchConfig, workDir, dataDir, sqlitePath string) liveSearchRun {
	t.Helper()
	configDir := filepath.Join(workDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("create live %s config dir: %v", tc.Platform, err)
	}
	configYAML := liveSearchConfigYAML(tc, opts, dataDir, sqlitePath)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(configYAML), 0644); err != nil {
		t.Fatalf("write live %s config: %v", tc.Platform, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+dataDir,
		"MCPRO_SQLITE_DB_PATH="+sqlitePath,
	)
	cmd.Env = append(cmd.Env, env...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("live %s command timed out\n%s", tc.Platform, output)
	}
	if err != nil {
		t.Fatalf("live %s command failed: %v\n%s", tc.Platform, err, output)
	}
	if strings.Contains(string(output), " ERROR ") {
		t.Fatalf("live %s command logged an error\n%s", tc.Platform, output)
	}
	return liveSearchRun{WorkDir: workDir, DataDir: dataDir, SQLitePath: sqlitePath, Output: output}
}

func liveSearchConfigYAML(tc liveSearchPlatformCase, opts liveSearchConfig, dataDir, sqlitePath string) string {
	saveDataOption := opts.SaveDataOption
	if saveDataOption == "" {
		saveDataOption = "json"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("platform: %q\n", tc.Platform))
	writeLiveList(&builder, "keywords", []string{tc.Keyword})
	builder.WriteString("crawlerType: \"search\"\n")
	builder.WriteString(fmt.Sprintf("saveDataOption: %q\n", saveDataOption))
	builder.WriteString("accountPoolSaveType: \"cookie_bridge\"\n")
	builder.WriteString(fmt.Sprintf("cookieBridgeURL: %q\n", envOrDefault("MCPRO_OTHER_LIVE_COOKIE_BRIDGE_URL", "http://127.0.0.1:8274")))
	builder.WriteString(fmt.Sprintf("dataDir: %q\n", dataDir))
	builder.WriteString(fmt.Sprintf("sqliteDbPath: %q\n", sqlitePath))
	builder.WriteString("startPage: 1\n")
	builder.WriteString("crawlerMaxNotesCount: 1\n")
	builder.WriteString("maxConcurrencyNum: 1\n")
	builder.WriteString("requestTimeOut: 20\n")
	builder.WriteString("hotSearchMaxItemsCount: 1\n")
	builder.WriteString("hotSearchPerItemContentCount: 1\n")
	builder.WriteString("enableGetComments: false\n")
	builder.WriteString("enableGetSubComments: false\n")
	builder.WriteString(fmt.Sprintf("enableCheckpoint: %s\n", strconv.FormatBool(opts.EnableCheckpoint)))
	builder.WriteString("perNoteMaxComments: 1\n")
	builder.WriteString("crawlerTimeSleep: 0\n")
	builder.WriteString("hotSearchCheckpointMaxAgeSeconds: 21600\n")
	builder.WriteString("debugRequest: false\n")
	writeLiveList(&builder, "specifiedNoteIds", nil)
	writeLiveList(&builder, "specifiedNoteURLs", nil)
	writeLiveList(&builder, "specifiedCreatorIds", nil)
	writeLiveList(&builder, "specifiedCreatorURLs", nil)
	builder.WriteString("proxyConfig:\n  enableIpProxy: false\n")
	builder.WriteString("signSrvConfig:\n")
	builder.WriteString(fmt.Sprintf("  signSrvHost: %q\n", envOrDefault("MCPRO_OTHER_LIVE_SIGN_HOST", "127.0.0.1")))
	port, _ := strconv.Atoi(envOrDefault("MCPRO_OTHER_LIVE_SIGN_PORT", "8989"))
	builder.WriteString(fmt.Sprintf("  signSrvPort: %d\n", port))
	builder.WriteString("platformEndpoints: {}\n")
	builder.WriteString("dbConfig:\n  relationDbHost: \"127.0.0.1\"\n  relationDbPort: 3306\n  relationDbUser: \"root\"\n  relationDbPwd: \"\"\n  relationDbName: \"media_crawler_pro\"\n")
	return builder.String()
}

func readLiveSearchRows(t *testing.T, run liveSearchRun, tc liveSearchPlatformCase) []map[string]any {
	t.Helper()
	return readLiveJSONRows(t, filepath.Join(run.DataDir, tc.DataPlatform, "json", datedFile("search_contents", "json")), run.Output)
}

func requireLiveSearchRows(t *testing.T, tc liveSearchPlatformCase, rows []map[string]any, output []byte) {
	t.Helper()
	if len(rows) == 0 {
		t.Fatalf("live %s search should save at least one row\n%s", tc.Platform, output)
	}
	if tc.ExpectedRows > 0 && len(rows) != tc.ExpectedRows {
		t.Fatalf("live %s search should save %d rows, got %d\n%s", tc.Platform, tc.ExpectedRows, len(rows), output)
	}
	seen := map[string]struct{}{}
	for _, row := range rows {
		for _, field := range tc.Required {
			if liveString(row, field) == "" {
				t.Fatalf("live %s row missing %s: %#v\n%s", tc.Platform, field, row, output)
			}
		}
		if got := liveString(row, "source_keyword"); got != tc.Keyword {
			t.Fatalf("live %s row source_keyword = %q, want %q: %#v\n%s", tc.Platform, got, tc.Keyword, row, output)
		}
		id := liveString(row, tc.IDField)
		if _, ok := seen[id]; ok {
			t.Fatalf("live %s rows contain duplicate %s=%s: %#v\n%s", tc.Platform, tc.IDField, id, rows, output)
		}
		seen[id] = struct{}{}
	}
}

func assertLiveSearchCheckpointResume(t *testing.T, binaryPath string, env []string, tc liveSearchPlatformCase) {
	t.Helper()
	workDir, err := os.MkdirTemp("/tmp", "mcpro-"+tc.Platform+"-live-checkpoint-*")
	if err != nil {
		t.Fatalf("create live %s checkpoint work dir: %v", tc.Platform, err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(workDir) })
	dataDir := filepath.Join(workDir, "data")
	sqlitePath := filepath.Join(workDir, "media_crawler.db")
	opts := liveSearchConfig{SaveDataOption: "json", EnableCheckpoint: true}

	first := runLiveSearchCLIInWorkDir(t, binaryPath, env, tc, opts, workDir, dataDir, sqlitePath)
	rowsAfterFirst := readLiveSearchRows(t, first, tc)
	requireLiveSearchRows(t, tc, rowsAfterFirst, first.Output)

	second := runLiveSearchCLIInWorkDir(t, binaryPath, env, tc, opts, workDir, dataDir, sqlitePath)
	rowsAfterSecond := readLiveSearchRows(t, second, tc)
	if len(rowsAfterSecond) != len(rowsAfterFirst) {
		t.Fatalf("live %s checkpoint second run should not append duplicate rows: first=%d second=%d\n%s", tc.Platform, len(rowsAfterFirst), len(rowsAfterSecond), second.Output)
	}
}
