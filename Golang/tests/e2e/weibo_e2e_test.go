// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/weibo_e2e_test.go
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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/e2e"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCLIWeiboSearchJSONWithMockServers(t *testing.T) {
	result := runCLIWeiboSearchWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "weibo", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read weibo result file: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal weibo result json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one saved weibo row, got %d: %s", len(rows), body)
	}
	row := rows[0]
	assertWeiboJSONRow(t, row)
	if _, ok := row["add_ts"]; !ok {
		t.Fatalf("weibo json should match Python and keep add_ts null: %#v", row)
	}
	if row["add_ts"] != nil {
		t.Fatalf("unexpected json add_ts: %#v", row["add_ts"])
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected json last_modify_ts: %#v", row)
	}
}

func TestCLIWeiboSearchCSVWithMockServers(t *testing.T) {
	result := runCLIWeiboSearchWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "weibo", "search_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open weibo csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read weibo csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected csv header and one weibo row, got %d: %#v", len(records), records)
	}
	assertWeiboCSVRow(t, records)
	if csvCell(records, "add_ts") != "" {
		t.Fatalf("unexpected csv add_ts: %#v", csvCell(records, "add_ts"))
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected non-empty csv last_modify_ts")
	}
}

func TestCLIWeiboSearchSQLiteWithMockServers(t *testing.T) {
	result := runCLIWeiboSearchWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open weibo sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []weiboSQLiteNote
	if err := db.Table("weibo_note").Find(&rows).Error; err != nil {
		t.Fatalf("query weibo sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one weibo sqlite row, got %d: %#v", len(rows), rows)
	}
	row := rows[0]
	if row.NoteID != "wb_note_e2e" || row.Content != "Weibo E2E mock note" {
		t.Fatalf("unexpected sqlite identity/content fields: %#v", row)
	}
	if row.CreateTime != 1710000700 || row.CreateDateTime != "2024-03-10 00:11:40+08:00" {
		t.Fatalf("unexpected sqlite create fields: %#v", row)
	}
	if row.LikedCount != "66" || row.CommentsCount != "7" || row.SharedCount != "8" {
		t.Fatalf("unexpected sqlite count fields: %#v", row)
	}
	if row.NoteURL != "https://m.weibo.cn/detail/wb_note_e2e" || row.IPLocation != "上海" {
		t.Fatalf("unexpected sqlite url/location fields: %#v", row)
	}
	if row.ImageList != "https://example.test/wb-image-1.jpg,https://example.test/wb-image-2.jpg" {
		t.Fatalf("unexpected sqlite image_list: %#v", row.ImageList)
	}
	if row.VideoURL != "https://example.test/wb-720.mp4,https://example.test/wb-hd.mp4" {
		t.Fatalf("unexpected sqlite video_url: %#v", row.VideoURL)
	}
	if row.UserID != "wb-author" || row.Nickname != "mock weibo author" || row.Gender != "m" {
		t.Fatalf("unexpected sqlite user fields: %#v", row)
	}
	if row.ProfileURL != "https://m.weibo.cn/u/wb-author" || row.Avatar != "https://example.test/wb-avatar.png" {
		t.Fatalf("unexpected sqlite profile fields: %#v", row)
	}
	if row.SourceKeyword != "mock-wb" {
		t.Fatalf("unexpected sqlite source_keyword: %#v", row.SourceKeyword)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected sqlite timestamps: %#v", row)
	}
}

func TestCLIWeiboSearchCheckpointResumeSkipsCompletedPages(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	searchCallsByPage := make(map[string]int)
	allowSecondPage := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/accounts":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]any{
					"accounts": []map[string]any{
						{
							"client_id": "weibo-search-resume-client",
							"platforms": map[string]any{
								"wb":    map[string]any{"has_cookies": true},
								"weibo": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"wb": "mock-weibo", "weibo": "mock-weibo"},
						},
					},
				},
			})
		case "/api/cookies/wb", "/api/cookies/weibo":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "SUB=mock-weibo-search-resume-sub; XSRF-TOKEN=mock-search-resume-xsrf"},
			})
		case "/api/config":
			writeJSON(t, w, map[string]any{"ok": 1, "data": map[string]bool{"login": true}})
		case "/api/container/getIndex":
			if !strings.Contains(r.URL.Query().Get("containerid"), "resume-wb") {
				t.Fatalf("unexpected weibo search containerid: %s", r.URL.Query().Get("containerid"))
			}
			page := r.URL.Query().Get("page")
			searchCallsByPage[page]++
			switch page {
			case "1":
				writeJSON(t, w, weiboSearchResumePayload("wb_search_resume_note_1", "Weibo search resume note 1", "wb-search-resume-author-1"))
			case "2":
				if !allowSecondPage {
					http.Error(w, "transient second search page failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, weiboSearchResumePayload("wb_search_resume_note_2", "Weibo search resume note 2", "wb-search-resume-author-2"))
			default:
				writeJSON(t, w, map[string]any{"ok": 1, "data": map[string]any{"cards": []map[string]any{}}})
			}
		default:
			t.Fatalf("unexpected weibo search resume request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	host, port := serverHostPort(t, mockServer.URL)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "wb"
keywords:
  - "resume-wb"
crawlerType: "search"
saveDataOption: "json"
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
startPage: 1
crawlerMaxNotesCount: 2
maxConcurrencyNum: 1
requestTimeOut: 2
enableGetComments: false
enableGetSubComments: false
enableCheckpoint: true
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  weiboApiUrl: %q
  weiboHotBandUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL+"/ajax/statuses/hot_band")
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	runCrawler := func() []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		runCmd := exec.CommandContext(ctx, binaryPath)
		runCmd.Dir = tmpDir
		runCmd.Env = append(os.Environ(),
			"RELATION_DB_HOST=127.0.0.1",
			"RELATION_DB_PORT=1",
			"MCPRO_DATA_DIR="+dataDir,
			"MCPRO_WEIBO_API_URL="+mockServer.URL,
			"MCPRO_WEIBO_HOT_BAND_URL="+mockServer.URL+"/ajax/statuses/hot_band",
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("weibo search resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("weibo search resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if searchCallsByPage["1"] != 1 || searchCallsByPage["2"] != 1 {
		t.Fatalf("first weibo search run should fetch page 1 and fail page 2 once, calls=%#v\n%s", searchCallsByPage, firstOutput)
	}

	allowSecondPage = true
	secondOutput := runCrawler()
	if searchCallsByPage["1"] != 1 {
		t.Fatalf("resume should skip completed first search page, calls=%#v\n%s", searchCallsByPage, secondOutput)
	}
	if searchCallsByPage["2"] != 2 {
		t.Fatalf("resume should retry second search page exactly once, calls=%#v\n%s", searchCallsByPage, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "weibo", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Weibo search rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["note_id"])] = true
	}
	if !seen["wb_search_resume_note_1"] || !seen["wb_search_resume_note_2"] {
		t.Fatalf("weibo search resume output missing expected note IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIWeiboDetailCheckpointResumeSkipsCompletedIDs(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	detailCallsByID := make(map[string]int)
	allowSecondDetail := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/accounts":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]any{
					"accounts": []map[string]any{
						{
							"client_id": "weibo-detail-resume-client",
							"platforms": map[string]any{
								"wb":    map[string]any{"has_cookies": true},
								"weibo": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"wb": "mock-weibo", "weibo": "mock-weibo"},
						},
					},
				},
			})
		case "/api/cookies/wb", "/api/cookies/weibo":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "SUB=mock-weibo-detail-resume-sub; XSRF-TOKEN=mock-detail-resume-xsrf"},
			})
		case "/api/config":
			writeJSON(t, w, map[string]any{"ok": 1, "data": map[string]bool{"login": true}})
		case "/detail/wb_detail_resume_note_1":
			detailCallsByID["wb_detail_resume_note_1"]++
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(weiboDetailHTMLFor("wb_detail_resume_note_1", "Weibo detail resume note 1", "wb-detail-resume-author-1")))
		case "/detail/wb_detail_resume_note_2":
			detailCallsByID["wb_detail_resume_note_2"]++
			if !allowSecondDetail {
				http.Error(w, "transient second detail failure", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(weiboDetailHTMLFor("wb_detail_resume_note_2", "Weibo detail resume note 2", "wb-detail-resume-author-2")))
		default:
			t.Fatalf("unexpected weibo detail resume request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	host, port := serverHostPort(t, mockServer.URL)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "wb"
keywords:
  - "mock-wb"
crawlerType: "detail"
saveDataOption: "json"
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
startPage: 1
crawlerMaxNotesCount: 10
maxConcurrencyNum: 1
requestTimeOut: 2
enableGetComments: false
enableGetSubComments: false
enableCheckpoint: true
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  weiboApiUrl: %q
  weiboHotBandUrl: %q
specifiedNoteIds:
  - "wb_detail_resume_note_1"
  - "wb_detail_resume_note_2"
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL+"/ajax/statuses/hot_band")
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	runCrawler := func() []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		runCmd := exec.CommandContext(ctx, binaryPath)
		runCmd.Dir = tmpDir
		runCmd.Env = append(os.Environ(),
			"RELATION_DB_HOST=127.0.0.1",
			"RELATION_DB_PORT=1",
			"MCPRO_DATA_DIR="+dataDir,
			"MCPRO_WEIBO_API_URL="+mockServer.URL,
			"MCPRO_WEIBO_HOT_BAND_URL="+mockServer.URL+"/ajax/statuses/hot_band",
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("weibo detail resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("weibo detail resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if detailCallsByID["wb_detail_resume_note_1"] != 1 || detailCallsByID["wb_detail_resume_note_2"] != 1 {
		t.Fatalf("first weibo detail run should fetch both IDs and fail second once, calls=%#v\n%s", detailCallsByID, firstOutput)
	}

	allowSecondDetail = true
	secondOutput := runCrawler()
	if detailCallsByID["wb_detail_resume_note_1"] != 1 {
		t.Fatalf("resume should skip completed first detail ID, calls=%#v\n%s", detailCallsByID, secondOutput)
	}
	if detailCallsByID["wb_detail_resume_note_2"] != 2 {
		t.Fatalf("resume should retry second detail ID exactly once, calls=%#v\n%s", detailCallsByID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "weibo", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Weibo detail rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["note_id"])] = true
	}
	if !seen["wb_detail_resume_note_1"] || !seen["wb_detail_resume_note_2"] {
		t.Fatalf("weibo detail resume output missing expected note IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIWeiboDetailJSONWithMockServers(t *testing.T) {
	result := runCLIWeiboDetailWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "weibo", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read weibo detail result file: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal weibo detail json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one saved weibo detail row, got %d: %s", len(rows), body)
	}
	row := rows[0]
	assertWeiboDetailJSONRow(t, row)
	if _, ok := row["add_ts"]; !ok {
		t.Fatalf("weibo detail json should match Python and keep add_ts null: %#v", row)
	}
	if row["add_ts"] != nil {
		t.Fatalf("unexpected detail json add_ts: %#v", row["add_ts"])
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected detail json last_modify_ts: %#v", row)
	}
}

func TestCLIWeiboDetailCSVWithMockServers(t *testing.T) {
	result := runCLIWeiboDetailWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "weibo", "detail_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open weibo detail csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read weibo detail csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected detail csv header and one row, got %d: %#v", len(records), records)
	}
	assertWeiboDetailCSVRow(t, records)
	if csvCell(records, "add_ts") != "" {
		t.Fatalf("unexpected detail csv add_ts: %#v", csvCell(records, "add_ts"))
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected non-empty detail csv last_modify_ts")
	}
}

func TestCLIWeiboDetailSQLiteWithMockServers(t *testing.T) {
	result := runCLIWeiboDetailWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open weibo detail sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []weiboSQLiteNote
	if err := db.Table("weibo_note").Find(&rows).Error; err != nil {
		t.Fatalf("query weibo detail sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one weibo detail sqlite row, got %d: %#v", len(rows), rows)
	}
	row := rows[0]
	if row.NoteID != "wb_detail_e2e" || row.Content != "Weibo detail E2E mock note" {
		t.Fatalf("unexpected detail sqlite identity/content fields: %#v", row)
	}
	if row.CreateTime != 1710000800 || row.CreateDateTime != "2024-03-10 00:13:20+08:00" {
		t.Fatalf("unexpected detail sqlite create fields: %#v", row)
	}
	if row.LikedCount != "166" || row.CommentsCount != "17" || row.SharedCount != "18" {
		t.Fatalf("unexpected detail sqlite count fields: %#v", row)
	}
	if row.NoteURL != "https://m.weibo.cn/detail/wb_detail_e2e" || row.IPLocation != "北京" {
		t.Fatalf("unexpected detail sqlite url/location fields: %#v", row)
	}
	if row.ImageList != "https://example.test/wb-detail-image-1.jpg" {
		t.Fatalf("unexpected detail sqlite image_list: %#v", row.ImageList)
	}
	if row.VideoURL != "https://example.test/wb-detail-720.mp4,https://example.test/wb-detail-ld.mp4" {
		t.Fatalf("unexpected detail sqlite video_url: %#v", row.VideoURL)
	}
	if row.UserID != "wb-detail-author" || row.Nickname != "mock weibo detail author" || row.Gender != "f" {
		t.Fatalf("unexpected detail sqlite user fields: %#v", row)
	}
	if row.ProfileURL != "https://m.weibo.cn/u/wb-detail-author" || row.Avatar != "https://example.test/wb-detail-avatar.png" {
		t.Fatalf("unexpected detail sqlite profile fields: %#v", row)
	}
	if row.SourceKeyword != "" {
		t.Fatalf("unexpected detail sqlite source_keyword: %#v", row.SourceKeyword)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected detail sqlite timestamps: %#v", row)
	}
}

func TestCLIWeiboDetailCommentsJSONWithMockServers(t *testing.T) {
	result := runCLIWeiboDetailCommentsWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "weibo", "json", "detail_comments_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read weibo detail comments json: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal weibo detail comments json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two weibo comment rows, got %d: %s", len(rows), body)
	}
	assertWeiboCommentJSONRow(t, rows[0], "wb_comment_root", "", "root weibo comment", "1710000900", "2024-03-10 00:15:00+08:00", "1", "12", "广东", "wb-commenter", "mock wb commenter", "m", "https://m.weibo.cn/u/wb-commenter", "https://example.test/wb-comment-avatar.png")
	assertWeiboCommentJSONRow(t, rows[1], "wb_comment_sub", "wb_comment_root", "sub weibo comment", "1710000910", "2024-03-10 00:15:10+08:00", "0", "3", "深圳", "wb-sub-commenter", "mock wb sub commenter", "f", "https://m.weibo.cn/u/wb-sub-commenter", "https://example.test/wb-sub-comment-avatar.png")
}

func TestCLIWeiboDetailCommentsCSVWithMockServers(t *testing.T) {
	result := runCLIWeiboDetailCommentsWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "weibo", "detail_comments_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open weibo detail comments csv: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read weibo detail comments csv: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 3 {
		t.Fatalf("expected comments csv header and two rows, got %d: %#v", len(records), records)
	}
	assertWeiboCommentCSVRow(t, records, 1, "wb_comment_root", "", "root weibo comment", "1710000900", "2024-03-10 00:15:00+08:00", "1", "12", "广东", "wb-commenter", "mock wb commenter", "m", "https://m.weibo.cn/u/wb-commenter", "https://example.test/wb-comment-avatar.png")
	assertWeiboCommentCSVRow(t, records, 2, "wb_comment_sub", "wb_comment_root", "sub weibo comment", "1710000910", "2024-03-10 00:15:10+08:00", "0", "3", "深圳", "wb-sub-commenter", "mock wb sub commenter", "f", "https://m.weibo.cn/u/wb-sub-commenter", "https://example.test/wb-sub-comment-avatar.png")
}

func TestCLIWeiboDetailCommentsSQLiteWithMockServers(t *testing.T) {
	result := runCLIWeiboDetailCommentsWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open weibo detail comments sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []weiboSQLiteComment
	if err := db.Table("weibo_note_comment").Order("comment_id").Find(&rows).Error; err != nil {
		t.Fatalf("query weibo detail comments sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two weibo comment sqlite rows, got %d: %#v", len(rows), rows)
	}
	assertWeiboSQLiteComment(t, rows[0], "wb_comment_root", "", "root weibo comment", 1710000900, "2024-03-10 00:15:00+08:00", "1", "12", "广东", "wb-commenter", "mock wb commenter", "m", "https://m.weibo.cn/u/wb-commenter", "https://example.test/wb-comment-avatar.png")
	assertWeiboSQLiteComment(t, rows[1], "wb_comment_sub", "wb_comment_root", "sub weibo comment", 1710000910, "2024-03-10 00:15:10+08:00", "0", "3", "深圳", "wb-sub-commenter", "mock wb sub commenter", "f", "https://m.weibo.cn/u/wb-sub-commenter", "https://example.test/wb-sub-comment-avatar.png")
}

func TestCLIWeiboCreatorJSONWithMockServers(t *testing.T) {
	result := runCLIWeiboCreatorWithMockServers(t, "json")

	creatorPath := filepath.Join(result.dataDir, "weibo", "json", "creator_creator_"+time.Now().Format("2006-01-02")+".json")
	contentPath := filepath.Join(result.dataDir, "weibo", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	creatorRows := readJSONRows(t, creatorPath, result.output)
	contentRows := readJSONRows(t, contentPath, result.output)
	if len(creatorRows) != 1 || len(contentRows) != 1 {
		t.Fatalf("expected one creator and one content row, got creators=%d contents=%d", len(creatorRows), len(contentRows))
	}
	assertWeiboCreatorJSONRow(t, creatorRows[0])
	assertWeiboCreatorContentJSONRow(t, contentRows[0])
}

func TestCLIWeiboCreatorCSVWithMockServers(t *testing.T) {
	result := runCLIWeiboCreatorWithMockServers(t, "csv")

	creatorPath := filepath.Join(result.dataDir, "weibo", "creator_creator_"+time.Now().Format("2006-01-02")+".csv")
	contentPath := filepath.Join(result.dataDir, "weibo", "creator_contents_"+time.Now().Format("2006-01-02")+".csv")
	creatorRecords := readCSVRecords(t, creatorPath, result.output)
	contentRecords := readCSVRecords(t, contentPath, result.output)
	if len(creatorRecords) != 2 || len(contentRecords) != 2 {
		t.Fatalf("expected creator/content csv header and one row, got creators=%d contents=%d", len(creatorRecords), len(contentRecords))
	}
	assertWeiboCreatorCSVRow(t, creatorRecords)
	assertWeiboCreatorContentCSVRow(t, contentRecords)
}

func TestCLIWeiboCreatorSQLiteWithMockServers(t *testing.T) {
	result := runCLIWeiboCreatorWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open weibo creator sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var creatorRows []weiboSQLiteCreator
	if err := db.Table("weibo_creator").Find(&creatorRows).Error; err != nil {
		t.Fatalf("query weibo creator rows: %v\ncommand output:\n%s", err, result.output)
	}
	var contentRows []weiboSQLiteNote
	if err := db.Table("weibo_note").Find(&contentRows).Error; err != nil {
		t.Fatalf("query weibo creator content rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(creatorRows) != 1 || len(contentRows) != 1 {
		t.Fatalf("expected one creator and one content sqlite row, got creators=%d contents=%d", len(creatorRows), len(contentRows))
	}
	assertWeiboSQLiteCreator(t, creatorRows[0])
	assertWeiboSQLiteCreatorContent(t, contentRows[0])
}

func TestCLIWeiboCreatorCheckpointResumeByCreatorAndSinceID(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	infoCalls := 0
	noteCallsBySinceID := make(map[string]int)
	allowSecondPage := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/accounts":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]any{
					"accounts": []map[string]any{
						{
							"client_id": "weibo-creator-resume-client",
							"platforms": map[string]any{
								"wb":    map[string]any{"has_cookies": true},
								"weibo": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"wb": "mock-weibo", "weibo": "mock-weibo"},
						},
					},
				},
			})
		case "/api/cookies/wb", "/api/cookies/weibo":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "SUB=mock-weibo-creator-resume-sub; XSRF-TOKEN=mock-creator-resume-xsrf"},
			})
		case "/api/config":
			writeJSON(t, w, map[string]any{"ok": 1, "data": map[string]bool{"login": true}})
		case "/api/container/getIndex":
			containerID := r.URL.Query().Get("containerid")
			switch containerID {
			case "100505wb-creator-resume":
				infoCalls++
				writeJSON(t, w, weiboCreatorResumeInfoPayload())
			case "107603wb-creator-resume":
				sinceID := r.URL.Query().Get("since_id")
				if sinceID == "" {
					sinceID = "0"
				}
				noteCallsBySinceID[sinceID]++
				switch sinceID {
				case "0":
					writeJSON(t, w, weiboCreatorResumeNotesPayload("wb_creator_resume_note_1", "Weibo creator resume note 1", "wb-creator-resume-author-1", "since-2", 20))
				case "since-2":
					if !allowSecondPage {
						http.Error(w, "transient second creator page failure", http.StatusInternalServerError)
						return
					}
					writeJSON(t, w, weiboCreatorResumeNotesPayload("wb_creator_resume_note_2", "Weibo creator resume note 2", "wb-creator-resume-author-2", "0", 20))
				default:
					t.Fatalf("unexpected weibo creator since_id: %s", sinceID)
				}
			default:
				t.Fatalf("unexpected weibo creator containerid: %s", containerID)
			}
		default:
			t.Fatalf("unexpected weibo creator resume request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	host, port := serverHostPort(t, mockServer.URL)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "wb"
keywords:
  - "mock-wb"
crawlerType: "creator"
saveDataOption: "json"
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
startPage: 1
crawlerMaxNotesCount: 20
maxConcurrencyNum: 1
requestTimeOut: 2
enableGetComments: false
enableGetSubComments: false
enableCheckpoint: true
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  weiboApiUrl: %q
  weiboHotBandUrl: %q
specifiedCreatorIds:
  - "wb-creator-resume"
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL+"/ajax/statuses/hot_band")
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	runCrawler := func() []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		runCmd := exec.CommandContext(ctx, binaryPath)
		runCmd.Dir = tmpDir
		runCmd.Env = append(os.Environ(),
			"RELATION_DB_HOST=127.0.0.1",
			"RELATION_DB_PORT=1",
			"MCPRO_DATA_DIR="+dataDir,
			"MCPRO_WEIBO_API_URL="+mockServer.URL,
			"MCPRO_WEIBO_HOT_BAND_URL="+mockServer.URL+"/ajax/statuses/hot_band",
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("weibo creator resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("weibo creator resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if infoCalls != 1 || noteCallsBySinceID["0"] != 1 || noteCallsBySinceID["since-2"] != 1 {
		t.Fatalf("first weibo creator run should fetch profile, first since_id, and fail second since_id once, infoCalls=%d calls=%#v\n%s", infoCalls, noteCallsBySinceID, firstOutput)
	}

	allowSecondPage = true
	secondOutput := runCrawler()
	if infoCalls != 2 {
		t.Fatalf("resume should refresh creator profile to match Python behavior, infoCalls=%d\n%s", infoCalls, secondOutput)
	}
	if noteCallsBySinceID["0"] != 1 {
		t.Fatalf("resume should skip completed first creator since_id, calls=%#v\n%s", noteCallsBySinceID, secondOutput)
	}
	if noteCallsBySinceID["since-2"] != 2 {
		t.Fatalf("resume should retry second creator since_id exactly once, calls=%#v\n%s", noteCallsBySinceID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "weibo", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Weibo creator rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["note_id"])] = true
	}
	if !seen["wb_creator_resume_note_1"] || !seen["wb_creator_resume_note_2"] {
		t.Fatalf("weibo creator resume output missing expected note IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIWeiboHotSearchJSONWithMockServers(t *testing.T) {
	result := runCLIWeiboHotSearchWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "weibo", "json", "hot_search_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, result.output)
	if len(rows) != 1 {
		t.Fatalf("expected one hot_search json row, got %d: %#v", len(rows), rows)
	}
	assertWeiboHotSearchMapRow(t, rows[0])
	if _, ok := rows[0]["add_ts"]; !ok {
		t.Fatalf("weibo hot_search json should match Python and keep add_ts null: %#v", rows[0])
	}
	if rows[0]["add_ts"] != nil {
		t.Fatalf("unexpected hot_search json add_ts: %#v", rows[0]["add_ts"])
	}
	if _, ok := rows[0]["last_modify_ts"]; !ok {
		t.Fatalf("expected hot_search json last_modify_ts: %#v", rows[0])
	}
}

func TestCLIWeiboHotSearchCSVWithMockServers(t *testing.T) {
	result := runCLIWeiboHotSearchWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "weibo", "hot_search_contents_"+time.Now().Format("2006-01-02")+".csv")
	records := readCSVRecords(t, resultPath, result.output)
	if len(records) != 2 {
		t.Fatalf("expected hot_search csv header and one row, got %d: %#v", len(records), records)
	}
	assertWeiboHotSearchCSVRow(t, records)
	if csvCell(records, "add_ts") != "" {
		t.Fatalf("unexpected hot_search csv add_ts: %#v", csvCell(records, "add_ts"))
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected non-empty hot_search csv last_modify_ts")
	}
}

func TestCLIWeiboHotSearchSQLiteWithMockServers(t *testing.T) {
	result := runCLIWeiboHotSearchWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open weibo hot_search sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []weiboSQLiteNote
	if err := db.Table("weibo_note").Find(&rows).Error; err != nil {
		t.Fatalf("query weibo hot_search sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one hot_search sqlite note row, got %d: %#v", len(rows), rows)
	}
	assertWeiboSQLiteHotSearchNote(t, rows[0])

	var items []weiboSQLiteHotSearchItem
	if err := db.Table("platform_hot_search_item").Find(&items).Error; err != nil {
		t.Fatalf("query weibo hot_search item rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(items) != 1 {
		t.Fatalf("expected one hot_search item row, got %d: %#v", len(items), items)
	}
	item := items[0]
	if item.HotItemID == "" || item.Platform != "wb" || item.BoardName != "hot_search" || item.BoardType != "topic" || item.RankNo != 1 {
		t.Fatalf("unexpected hot_search item identity fields: %#v", item)
	}
	if item.Title != "mock hot weibo" || item.Keyword != "mock hot weibo" || item.HotValue != "999" || item.Badge != "热" {
		t.Fatalf("unexpected hot_search item title/value fields: %#v", item)
	}
	if item.TopicID != "wb-hot-mid" || item.TopicURL != "https://s.weibo.com/weibo?q=mock%20hot%20weibo" {
		t.Fatalf("unexpected hot_search item topic fields: %#v", item)
	}
	if !strings.Contains(item.RawJSON, "mock hot weibo") || item.SnapshotDate == "" || item.SnapshotTs == 0 || item.AddTs == 0 || item.LastModifyTs == 0 {
		t.Fatalf("unexpected hot_search item raw/timestamp fields: %#v", item)
	}

	var relations []weiboSQLiteHotSearchRelation
	if err := db.Table("platform_hot_search_content_rel").Find(&relations).Error; err != nil {
		t.Fatalf("query weibo hot_search relation rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(relations) != 1 {
		t.Fatalf("expected one hot_search relation row, got %d: %#v", len(relations), relations)
	}
	rel := relations[0]
	if rel.RelID == "" || rel.HotItemID != item.HotItemID || rel.Platform != "wb" || rel.ContentID != "wb_hot_note_e2e" {
		t.Fatalf("unexpected hot_search relation identity fields: %#v", rel)
	}
	if rel.ContentType != "note" || rel.ContentTable != "weibo_note" || rel.RankNo != 1 || rel.SourceKeyword != "mock hot weibo" {
		t.Fatalf("unexpected hot_search relation content/rank fields: %#v", rel)
	}
	if rel.AddTs == 0 || rel.LastModifyTs == 0 {
		t.Fatalf("expected hot_search relation timestamps: %#v", rel)
	}
}

func TestCLIWeiboHotSearchCheckpointResumeUsesSnapshotAndSkipsCompletedTopic(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	hotBandCalls := 0
	searchCallsByKeyword := make(map[string]int)
	allowSecondTopic := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/accounts":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]any{
					"accounts": []map[string]any{
						{
							"client_id": "weibo-hot-resume-client",
							"platforms": map[string]any{
								"wb":    map[string]any{"has_cookies": true},
								"weibo": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"wb": "mock-weibo", "weibo": "mock-weibo"},
						},
					},
				},
			})
		case "/api/cookies/wb", "/api/cookies/weibo":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "SUB=mock-weibo-hot-resume-sub; XSRF-TOKEN=mock-hot-resume-xsrf"},
			})
		case "/api/config":
			writeJSON(t, w, map[string]any{"ok": 1, "data": map[string]bool{"login": true}})
		case "/ajax/statuses/hot_band":
			hotBandCalls++
			writeJSON(t, w, weiboHotBandResumePayload())
		case "/api/container/getIndex":
			containerID := r.URL.Query().Get("containerid")
			switch {
			case strings.Contains(containerID, "resume hot one"):
				searchCallsByKeyword["resume hot one"]++
				writeJSON(t, w, weiboHotSearchResumePayload("wb_hot_resume_note_1", "Weibo hot resume note 1", "wb-hot-resume-author-1"))
			case strings.Contains(containerID, "resume hot two"):
				searchCallsByKeyword["resume hot two"]++
				if !allowSecondTopic {
					http.Error(w, "transient second hot topic failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, weiboHotSearchResumePayload("wb_hot_resume_note_2", "Weibo hot resume note 2", "wb-hot-resume-author-2"))
			default:
				t.Fatalf("unexpected weibo hot resume search containerid: %s", containerID)
			}
		default:
			t.Fatalf("unexpected weibo hot resume request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	host, port := serverHostPort(t, mockServer.URL)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	configYAML := fmt.Sprintf(`
platform: "wb"
keywords:
  - "unused"
crawlerType: "hot_search"
saveDataOption: "json"
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
startPage: 1
crawlerMaxNotesCount: 2
maxConcurrencyNum: 1
requestTimeOut: 2
hotSearchMaxItemsCount: 2
hotSearchPerItemContentCount: 1
enableGetComments: false
enableGetSubComments: false
enableCheckpoint: true
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  weiboApiUrl: %q
  weiboHotBandUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL+"/ajax/statuses/hot_band")
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	runCrawler := func() []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		runCmd := exec.CommandContext(ctx, binaryPath)
		runCmd.Dir = tmpDir
		runCmd.Env = append(os.Environ(),
			"RELATION_DB_HOST=127.0.0.1",
			"RELATION_DB_PORT=1",
			"MCPRO_DATA_DIR="+dataDir,
			"MCPRO_WEIBO_API_URL="+mockServer.URL,
			"MCPRO_WEIBO_HOT_BAND_URL="+mockServer.URL+"/ajax/statuses/hot_band",
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("weibo hot resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("weibo hot resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if hotBandCalls != 1 || searchCallsByKeyword["resume hot one"] != 1 || searchCallsByKeyword["resume hot two"] != 1 {
		t.Fatalf("first weibo hot_search run should fetch rank and fail second topic once; rank=%d search=%#v\n%s", hotBandCalls, searchCallsByKeyword, firstOutput)
	}

	allowSecondTopic = true
	secondOutput := runCrawler()
	if hotBandCalls != 1 {
		t.Fatalf("resume should reuse weibo hot_search snapshot and not refetch rank, got %d\n%s", hotBandCalls, secondOutput)
	}
	if searchCallsByKeyword["resume hot one"] != 1 {
		t.Fatalf("resume should skip completed first hot topic, search=%#v\n%s", searchCallsByKeyword, secondOutput)
	}
	if searchCallsByKeyword["resume hot two"] != 2 {
		t.Fatalf("resume should retry second hot topic exactly once, search=%#v\n%s", searchCallsByKeyword, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "weibo", "json", "hot_search_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Weibo hot_search rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["note_id"])] = true
	}
	if !seen["wb_hot_resume_note_1"] || !seen["wb_hot_resume_note_2"] {
		t.Fatalf("weibo hot_search resume output missing expected note IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func assertWeiboJSONRow(t *testing.T, row map[string]any) {
	t.Helper()
	if row["note_id"] != "wb_note_e2e" || row["content"] != "Weibo E2E mock note" {
		t.Fatalf("unexpected weibo identity/content fields: %#v", row)
	}
	if row["create_time"] != "1710000700" || row["create_date_time"] != "2024-03-10 00:11:40+08:00" {
		t.Fatalf("unexpected create fields: %#v", row)
	}
	if row["liked_count"] != "66" || row["comments_count"] != "7" || row["shared_count"] != "8" {
		t.Fatalf("unexpected count fields: %#v", row)
	}
	if row["note_url"] != "https://m.weibo.cn/detail/wb_note_e2e" || row["ip_location"] != "上海" {
		t.Fatalf("unexpected url/location fields: %#v", row)
	}
	if row["image_list"] != "https://example.test/wb-image-1.jpg,https://example.test/wb-image-2.jpg" {
		t.Fatalf("unexpected image_list: %#v", row["image_list"])
	}
	if row["video_url"] != "https://example.test/wb-720.mp4,https://example.test/wb-hd.mp4" {
		t.Fatalf("unexpected video_url: %#v", row["video_url"])
	}
	if row["user_id"] != "wb-author" || row["nickname"] != "mock weibo author" || row["gender"] != "m" {
		t.Fatalf("unexpected user fields: %#v", row)
	}
	if row["profile_url"] != "https://m.weibo.cn/u/wb-author" || row["avatar"] != "https://example.test/wb-avatar.png" {
		t.Fatalf("unexpected profile fields: %#v", row)
	}
	if row["source_keyword"] != "mock-wb" {
		t.Fatalf("unexpected source_keyword: %#v", row["source_keyword"])
	}
}

func assertWeiboCSVRow(t *testing.T, records [][]string) {
	t.Helper()
	if csvCell(records, "note_id") != "wb_note_e2e" || csvCell(records, "content") != "Weibo E2E mock note" {
		t.Fatalf("unexpected csv identity/content fields: %#v", records)
	}
	if csvCell(records, "create_time") != "1710000700" || csvCell(records, "create_date_time") != "2024-03-10 00:11:40+08:00" {
		t.Fatalf("unexpected csv create fields: %#v", records)
	}
	if csvCell(records, "liked_count") != "66" || csvCell(records, "comments_count") != "7" || csvCell(records, "shared_count") != "8" {
		t.Fatalf("unexpected csv count fields: %#v", records)
	}
	if csvCell(records, "note_url") != "https://m.weibo.cn/detail/wb_note_e2e" || csvCell(records, "ip_location") != "上海" {
		t.Fatalf("unexpected csv url/location fields: %#v", records)
	}
	if csvCell(records, "image_list") != "https://example.test/wb-image-1.jpg,https://example.test/wb-image-2.jpg" {
		t.Fatalf("unexpected csv image_list: %#v", csvCell(records, "image_list"))
	}
	if csvCell(records, "video_url") != "https://example.test/wb-720.mp4,https://example.test/wb-hd.mp4" {
		t.Fatalf("unexpected csv video_url: %#v", csvCell(records, "video_url"))
	}
	if csvCell(records, "user_id") != "wb-author" || csvCell(records, "nickname") != "mock weibo author" || csvCell(records, "gender") != "m" {
		t.Fatalf("unexpected csv user fields: %#v", records)
	}
	if csvCell(records, "profile_url") != "https://m.weibo.cn/u/wb-author" || csvCell(records, "avatar") != "https://example.test/wb-avatar.png" {
		t.Fatalf("unexpected csv profile fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "mock-wb" {
		t.Fatalf("unexpected csv source_keyword: %#v", csvCell(records, "source_keyword"))
	}
}

func assertWeiboDetailJSONRow(t *testing.T, row map[string]any) {
	t.Helper()
	if row["note_id"] != "wb_detail_e2e" || row["content"] != "Weibo detail E2E mock note" {
		t.Fatalf("unexpected detail identity/content fields: %#v", row)
	}
	if row["create_time"] != "1710000800" || row["create_date_time"] != "2024-03-10 00:13:20+08:00" {
		t.Fatalf("unexpected detail create fields: %#v", row)
	}
	if row["liked_count"] != "166" || row["comments_count"] != "17" || row["shared_count"] != "18" {
		t.Fatalf("unexpected detail count fields: %#v", row)
	}
	if row["note_url"] != "https://m.weibo.cn/detail/wb_detail_e2e" || row["ip_location"] != "北京" {
		t.Fatalf("unexpected detail url/location fields: %#v", row)
	}
	if row["image_list"] != "https://example.test/wb-detail-image-1.jpg" {
		t.Fatalf("unexpected detail image_list: %#v", row["image_list"])
	}
	if row["video_url"] != "https://example.test/wb-detail-720.mp4,https://example.test/wb-detail-ld.mp4" {
		t.Fatalf("unexpected detail video_url: %#v", row["video_url"])
	}
	if row["user_id"] != "wb-detail-author" || row["nickname"] != "mock weibo detail author" || row["gender"] != "f" {
		t.Fatalf("unexpected detail user fields: %#v", row)
	}
	if row["profile_url"] != "https://m.weibo.cn/u/wb-detail-author" || row["avatar"] != "https://example.test/wb-detail-avatar.png" {
		t.Fatalf("unexpected detail profile fields: %#v", row)
	}
	if row["source_keyword"] != "" {
		t.Fatalf("unexpected detail source_keyword: %#v", row["source_keyword"])
	}
}

func assertWeiboDetailCSVRow(t *testing.T, records [][]string) {
	t.Helper()
	if csvCell(records, "note_id") != "wb_detail_e2e" || csvCell(records, "content") != "Weibo detail E2E mock note" {
		t.Fatalf("unexpected detail csv identity/content fields: %#v", records)
	}
	if csvCell(records, "create_time") != "1710000800" || csvCell(records, "create_date_time") != "2024-03-10 00:13:20+08:00" {
		t.Fatalf("unexpected detail csv create fields: %#v", records)
	}
	if csvCell(records, "liked_count") != "166" || csvCell(records, "comments_count") != "17" || csvCell(records, "shared_count") != "18" {
		t.Fatalf("unexpected detail csv count fields: %#v", records)
	}
	if csvCell(records, "note_url") != "https://m.weibo.cn/detail/wb_detail_e2e" || csvCell(records, "ip_location") != "北京" {
		t.Fatalf("unexpected detail csv url/location fields: %#v", records)
	}
	if csvCell(records, "image_list") != "https://example.test/wb-detail-image-1.jpg" {
		t.Fatalf("unexpected detail csv image_list: %#v", csvCell(records, "image_list"))
	}
	if csvCell(records, "video_url") != "https://example.test/wb-detail-720.mp4,https://example.test/wb-detail-ld.mp4" {
		t.Fatalf("unexpected detail csv video_url: %#v", csvCell(records, "video_url"))
	}
	if csvCell(records, "user_id") != "wb-detail-author" || csvCell(records, "nickname") != "mock weibo detail author" || csvCell(records, "gender") != "f" {
		t.Fatalf("unexpected detail csv user fields: %#v", records)
	}
	if csvCell(records, "profile_url") != "https://m.weibo.cn/u/wb-detail-author" || csvCell(records, "avatar") != "https://example.test/wb-detail-avatar.png" {
		t.Fatalf("unexpected detail csv profile fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "" {
		t.Fatalf("unexpected detail csv source_keyword: %#v", csvCell(records, "source_keyword"))
	}
}

func assertWeiboCommentJSONRow(t *testing.T, row map[string]any, commentID, parentID, content, createTime, createDateTime, subCount, likeCount, ipLocation, userID, nickname, gender, profileURL, avatar string) {
	t.Helper()
	if row["comment_id"] != commentID || row["note_id"] != "wb_detail_e2e" || row["content"] != content {
		t.Fatalf("unexpected comment identity/content fields: %#v", row)
	}
	if row["create_time"] != createTime || row["create_date_time"] != createDateTime {
		t.Fatalf("unexpected comment create fields: %#v", row)
	}
	if row["sub_comment_count"] != subCount || row["like_count"] != likeCount {
		t.Fatalf("unexpected comment count fields: %#v", row)
	}
	if row["ip_location"] != ipLocation || row["parent_comment_id"] != parentID {
		t.Fatalf("unexpected comment location/parent fields: %#v", row)
	}
	if row["user_id"] != userID || row["nickname"] != nickname || row["gender"] != gender {
		t.Fatalf("unexpected comment user fields: %#v", row)
	}
	if row["profile_url"] != profileURL || row["avatar"] != avatar {
		t.Fatalf("unexpected comment profile fields: %#v", row)
	}
	if row["add_ts"] != nil {
		t.Fatalf("unexpected comment add_ts: %#v", row["add_ts"])
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected comment last_modify_ts: %#v", row)
	}
}

func assertWeiboCommentCSVRow(t *testing.T, records [][]string, row int, commentID, parentID, content, createTime, createDateTime, subCount, likeCount, ipLocation, userID, nickname, gender, profileURL, avatar string) {
	t.Helper()
	if csvCellAt(records, row, "comment_id") != commentID || csvCellAt(records, row, "note_id") != "wb_detail_e2e" || csvCellAt(records, row, "content") != content {
		t.Fatalf("unexpected comment csv identity/content fields: %#v", records[row])
	}
	if csvCellAt(records, row, "create_time") != createTime || csvCellAt(records, row, "create_date_time") != createDateTime {
		t.Fatalf("unexpected comment csv create fields: %#v", records[row])
	}
	if csvCellAt(records, row, "sub_comment_count") != subCount || csvCellAt(records, row, "like_count") != likeCount {
		t.Fatalf("unexpected comment csv count fields: %#v", records[row])
	}
	if csvCellAt(records, row, "ip_location") != ipLocation || csvCellAt(records, row, "parent_comment_id") != parentID {
		t.Fatalf("unexpected comment csv location/parent fields: %#v", records[row])
	}
	if csvCellAt(records, row, "user_id") != userID || csvCellAt(records, row, "nickname") != nickname || csvCellAt(records, row, "gender") != gender {
		t.Fatalf("unexpected comment csv user fields: %#v", records[row])
	}
	if csvCellAt(records, row, "profile_url") != profileURL || csvCellAt(records, row, "avatar") != avatar {
		t.Fatalf("unexpected comment csv profile fields: %#v", records[row])
	}
	if csvCellAt(records, row, "add_ts") != "" || csvCellAt(records, row, "last_modify_ts") == "" {
		t.Fatalf("unexpected comment csv timestamps: %#v", records[row])
	}
}

func assertWeiboCreatorJSONRow(t *testing.T, row map[string]any) {
	t.Helper()
	if row["user_id"] != "wb-creator-user" || row["nickname"] != "mock weibo creator" {
		t.Fatalf("unexpected creator identity fields: %#v", row)
	}
	if row["gender"] != "女" || row["avatar"] != "https://example.test/wb-creator-avatar.png" {
		t.Fatalf("unexpected creator gender/avatar fields: %#v", row)
	}
	if row["desc"] != "weibo creator signature" || row["ip_location"] != "杭州" {
		t.Fatalf("unexpected creator desc/location fields: %#v", row)
	}
	if row["follows"] != "12" || row["fans"] != "3456" || row["tag_list"] != "" {
		t.Fatalf("unexpected creator count/tag fields: %#v", row)
	}
	if row["add_ts"] != nil {
		t.Fatalf("unexpected creator add_ts: %#v", row["add_ts"])
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected creator last_modify_ts: %#v", row)
	}
}

func assertWeiboCreatorContentJSONRow(t *testing.T, row map[string]any) {
	t.Helper()
	if row["note_id"] != "wb_creator_note_e2e" || row["content"] != "Weibo creator E2E mock note" {
		t.Fatalf("unexpected creator content identity/content fields: %#v", row)
	}
	if row["create_time"] != "1710001000" || row["create_date_time"] != "2024-03-10 00:16:40+08:00" {
		t.Fatalf("unexpected creator content create fields: %#v", row)
	}
	if row["liked_count"] != "266" || row["comments_count"] != "27" || row["shared_count"] != "28" {
		t.Fatalf("unexpected creator content count fields: %#v", row)
	}
	if row["note_url"] != "https://m.weibo.cn/detail/wb_creator_note_e2e" || row["ip_location"] != "成都" {
		t.Fatalf("unexpected creator content url/location fields: %#v", row)
	}
	if row["image_list"] != "https://example.test/wb-creator-image.jpg" || row["video_url"] != "https://example.test/wb-creator-720.mp4" {
		t.Fatalf("unexpected creator content media fields: %#v", row)
	}
	if row["user_id"] != "wb-creator-author" || row["nickname"] != "mock weibo creator author" || row["gender"] != "m" {
		t.Fatalf("unexpected creator content user fields: %#v", row)
	}
	if row["profile_url"] != "https://m.weibo.cn/u/wb-creator-author" || row["avatar"] != "https://example.test/wb-creator-author-avatar.png" {
		t.Fatalf("unexpected creator content profile fields: %#v", row)
	}
	if row["source_keyword"] != "" || row["add_ts"] != nil {
		t.Fatalf("unexpected creator content source/timestamps: %#v", row)
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected creator content last_modify_ts: %#v", row)
	}
}

func assertWeiboCreatorCSVRow(t *testing.T, records [][]string) {
	t.Helper()
	if csvCell(records, "user_id") != "wb-creator-user" || csvCell(records, "nickname") != "mock weibo creator" {
		t.Fatalf("unexpected creator csv identity fields: %#v", records)
	}
	if csvCell(records, "gender") != "女" || csvCell(records, "avatar") != "https://example.test/wb-creator-avatar.png" {
		t.Fatalf("unexpected creator csv gender/avatar fields: %#v", records)
	}
	if csvCell(records, "desc") != "weibo creator signature" || csvCell(records, "ip_location") != "杭州" {
		t.Fatalf("unexpected creator csv desc/location fields: %#v", records)
	}
	if csvCell(records, "follows") != "12" || csvCell(records, "fans") != "3456" || csvCell(records, "tag_list") != "" {
		t.Fatalf("unexpected creator csv count/tag fields: %#v", records)
	}
	if csvCell(records, "add_ts") != "" || csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("unexpected creator csv timestamps: %#v", records)
	}
}

func assertWeiboCreatorContentCSVRow(t *testing.T, records [][]string) {
	t.Helper()
	if csvCell(records, "note_id") != "wb_creator_note_e2e" || csvCell(records, "content") != "Weibo creator E2E mock note" {
		t.Fatalf("unexpected creator content csv identity/content fields: %#v", records)
	}
	if csvCell(records, "create_time") != "1710001000" || csvCell(records, "create_date_time") != "2024-03-10 00:16:40+08:00" {
		t.Fatalf("unexpected creator content csv create fields: %#v", records)
	}
	if csvCell(records, "liked_count") != "266" || csvCell(records, "comments_count") != "27" || csvCell(records, "shared_count") != "28" {
		t.Fatalf("unexpected creator content csv count fields: %#v", records)
	}
	if csvCell(records, "note_url") != "https://m.weibo.cn/detail/wb_creator_note_e2e" || csvCell(records, "ip_location") != "成都" {
		t.Fatalf("unexpected creator content csv url/location fields: %#v", records)
	}
	if csvCell(records, "image_list") != "https://example.test/wb-creator-image.jpg" || csvCell(records, "video_url") != "https://example.test/wb-creator-720.mp4" {
		t.Fatalf("unexpected creator content csv media fields: %#v", records)
	}
	if csvCell(records, "user_id") != "wb-creator-author" || csvCell(records, "nickname") != "mock weibo creator author" || csvCell(records, "gender") != "m" {
		t.Fatalf("unexpected creator content csv user fields: %#v", records)
	}
	if csvCell(records, "profile_url") != "https://m.weibo.cn/u/wb-creator-author" || csvCell(records, "avatar") != "https://example.test/wb-creator-author-avatar.png" {
		t.Fatalf("unexpected creator content csv profile fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "" || csvCell(records, "add_ts") != "" || csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("unexpected creator content csv source/timestamps: %#v", records)
	}
}

func assertWeiboHotSearchMapRow(t *testing.T, row map[string]any) {
	t.Helper()
	if row["note_id"] != "wb_hot_note_e2e" || row["content"] != "Weibo hot search E2E mock note" {
		t.Fatalf("unexpected hot_search identity/content fields: %#v", row)
	}
	if row["create_time"] != "1710001100" || row["create_date_time"] != "2024-03-10 00:18:20+08:00" {
		t.Fatalf("unexpected hot_search create fields: %#v", row)
	}
	if row["liked_count"] != "366" || row["comments_count"] != "37" || row["shared_count"] != "38" {
		t.Fatalf("unexpected hot_search count fields: %#v", row)
	}
	if row["note_url"] != "https://m.weibo.cn/detail/wb_hot_note_e2e" || row["ip_location"] != "深圳" {
		t.Fatalf("unexpected hot_search url/location fields: %#v", row)
	}
	if row["image_list"] != "https://example.test/wb-hot-image.jpg" || row["video_url"] != "https://example.test/wb-hot-720.mp4" {
		t.Fatalf("unexpected hot_search media fields: %#v", row)
	}
	if row["user_id"] != "wb-hot-author" || row["nickname"] != "mock weibo hot author" || row["gender"] != "f" {
		t.Fatalf("unexpected hot_search user fields: %#v", row)
	}
	if row["profile_url"] != "https://m.weibo.cn/u/wb-hot-author" || row["avatar"] != "https://example.test/wb-hot-avatar.png" {
		t.Fatalf("unexpected hot_search profile fields: %#v", row)
	}
	if row["source_keyword"] != "mock hot weibo" {
		t.Fatalf("unexpected hot_search source_keyword: %#v", row["source_keyword"])
	}
}

func assertWeiboHotSearchCSVRow(t *testing.T, records [][]string) {
	t.Helper()
	if csvCell(records, "note_id") != "wb_hot_note_e2e" || csvCell(records, "content") != "Weibo hot search E2E mock note" {
		t.Fatalf("unexpected hot_search csv identity/content fields: %#v", records)
	}
	if csvCell(records, "create_time") != "1710001100" || csvCell(records, "create_date_time") != "2024-03-10 00:18:20+08:00" {
		t.Fatalf("unexpected hot_search csv create fields: %#v", records)
	}
	if csvCell(records, "liked_count") != "366" || csvCell(records, "comments_count") != "37" || csvCell(records, "shared_count") != "38" {
		t.Fatalf("unexpected hot_search csv count fields: %#v", records)
	}
	if csvCell(records, "note_url") != "https://m.weibo.cn/detail/wb_hot_note_e2e" || csvCell(records, "ip_location") != "深圳" {
		t.Fatalf("unexpected hot_search csv url/location fields: %#v", records)
	}
	if csvCell(records, "image_list") != "https://example.test/wb-hot-image.jpg" || csvCell(records, "video_url") != "https://example.test/wb-hot-720.mp4" {
		t.Fatalf("unexpected hot_search csv media fields: %#v", records)
	}
	if csvCell(records, "user_id") != "wb-hot-author" || csvCell(records, "nickname") != "mock weibo hot author" || csvCell(records, "gender") != "f" {
		t.Fatalf("unexpected hot_search csv user fields: %#v", records)
	}
	if csvCell(records, "profile_url") != "https://m.weibo.cn/u/wb-hot-author" || csvCell(records, "avatar") != "https://example.test/wb-hot-avatar.png" {
		t.Fatalf("unexpected hot_search csv profile fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "mock hot weibo" {
		t.Fatalf("unexpected hot_search csv source_keyword: %#v", csvCell(records, "source_keyword"))
	}
}

type cliWeiboResult struct {
	output     []byte
	dataDir    string
	sqlitePath string
}

type weiboSQLiteNote struct {
	ID             int64
	UserID         string
	Nickname       string
	Avatar         string
	Gender         string
	ProfileURL     string `gorm:"column:profile_url"`
	IPLocation     string `gorm:"column:ip_location"`
	AddTs          int64
	LastModifyTs   int64
	NoteID         string
	Content        string
	CreateTime     int64
	CreateDateTime string
	LikedCount     string
	CommentsCount  string
	SharedCount    string
	NoteURL        string `gorm:"column:note_url"`
	ImageList      string
	VideoURL       string `gorm:"column:video_url"`
	SourceKeyword  string
}

type weiboSQLiteComment struct {
	ID              int64
	UserID          string
	Nickname        string
	Avatar          string
	Gender          string
	ProfileURL      string `gorm:"column:profile_url"`
	IPLocation      string `gorm:"column:ip_location"`
	AddTs           int64
	LastModifyTs    int64
	CommentID       string
	NoteID          string
	Content         string
	CreateTime      int64
	CreateDateTime  string
	SubCommentCount string
	ParentCommentID string
	LikeCount       string
}

type weiboSQLiteCreator struct {
	ID           int64
	UserID       string
	Nickname     string
	Avatar       string
	IPLocation   string `gorm:"column:ip_location"`
	AddTs        int64
	LastModifyTs int64
	Desc         string
	Gender       string
	Follows      string
	Fans         string
	TagList      string
}

type weiboSQLiteHotSearchItem struct {
	ID           int64
	HotItemID    string
	Platform     string
	BoardName    string
	BoardType    string
	SnapshotDate string
	SnapshotTs   int64
	RankNo       int
	Title        string
	Keyword      string
	HotValue     string
	Badge        string
	TopicID      string
	TopicURL     string
	RawJSON      string
	AddTs        int64
	LastModifyTs int64
}

type weiboSQLiteHotSearchRelation struct {
	ID            int64
	RelID         string
	HotItemID     string
	Platform      string
	ContentID     string
	ContentType   string
	ContentTable  string
	RankNo        int
	SourceKeyword string
	AddTs         int64
	LastModifyTs  int64
}

func assertWeiboSQLiteComment(t *testing.T, row weiboSQLiteComment, commentID, parentID, content string, createTime int64, createDateTime, subCount, likeCount, ipLocation, userID, nickname, gender, profileURL, avatar string) {
	t.Helper()
	if row.CommentID != commentID || row.NoteID != "wb_detail_e2e" || row.Content != content {
		t.Fatalf("unexpected sqlite comment identity/content fields: %#v", row)
	}
	if row.CreateTime != createTime || row.CreateDateTime != createDateTime {
		t.Fatalf("unexpected sqlite comment create fields: %#v", row)
	}
	if row.SubCommentCount != subCount || row.LikeCount != likeCount {
		t.Fatalf("unexpected sqlite comment count fields: %#v", row)
	}
	if row.IPLocation != ipLocation || row.ParentCommentID != parentID {
		t.Fatalf("unexpected sqlite comment location/parent fields: %#v", row)
	}
	if row.UserID != userID || row.Nickname != nickname || row.Gender != gender {
		t.Fatalf("unexpected sqlite comment user fields: %#v", row)
	}
	if row.ProfileURL != profileURL || row.Avatar != avatar {
		t.Fatalf("unexpected sqlite comment profile fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected sqlite comment timestamps: %#v", row)
	}
}

func assertWeiboSQLiteCreator(t *testing.T, row weiboSQLiteCreator) {
	t.Helper()
	if row.UserID != "wb-creator-user" || row.Nickname != "mock weibo creator" {
		t.Fatalf("unexpected sqlite creator identity fields: %#v", row)
	}
	if row.Gender != "女" || row.Avatar != "https://example.test/wb-creator-avatar.png" {
		t.Fatalf("unexpected sqlite creator gender/avatar fields: %#v", row)
	}
	if row.Desc != "weibo creator signature" || row.IPLocation != "杭州" {
		t.Fatalf("unexpected sqlite creator desc/location fields: %#v", row)
	}
	if row.Follows != "12" || row.Fans != "3456" || row.TagList != "" {
		t.Fatalf("unexpected sqlite creator count/tag fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected sqlite creator timestamps: %#v", row)
	}
}

func assertWeiboSQLiteCreatorContent(t *testing.T, row weiboSQLiteNote) {
	t.Helper()
	if row.NoteID != "wb_creator_note_e2e" || row.Content != "Weibo creator E2E mock note" {
		t.Fatalf("unexpected sqlite creator content identity/content fields: %#v", row)
	}
	if row.CreateTime != 1710001000 || row.CreateDateTime != "2024-03-10 00:16:40+08:00" {
		t.Fatalf("unexpected sqlite creator content create fields: %#v", row)
	}
	if row.LikedCount != "266" || row.CommentsCount != "27" || row.SharedCount != "28" {
		t.Fatalf("unexpected sqlite creator content count fields: %#v", row)
	}
	if row.NoteURL != "https://m.weibo.cn/detail/wb_creator_note_e2e" || row.IPLocation != "成都" {
		t.Fatalf("unexpected sqlite creator content url/location fields: %#v", row)
	}
	if row.ImageList != "https://example.test/wb-creator-image.jpg" || row.VideoURL != "https://example.test/wb-creator-720.mp4" {
		t.Fatalf("unexpected sqlite creator content media fields: %#v", row)
	}
	if row.UserID != "wb-creator-author" || row.Nickname != "mock weibo creator author" || row.Gender != "m" {
		t.Fatalf("unexpected sqlite creator content user fields: %#v", row)
	}
	if row.ProfileURL != "https://m.weibo.cn/u/wb-creator-author" || row.Avatar != "https://example.test/wb-creator-author-avatar.png" {
		t.Fatalf("unexpected sqlite creator content profile fields: %#v", row)
	}
	if row.SourceKeyword != "" || row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("unexpected sqlite creator content source/timestamps: %#v", row)
	}
}

func assertWeiboSQLiteHotSearchNote(t *testing.T, row weiboSQLiteNote) {
	t.Helper()
	if row.NoteID != "wb_hot_note_e2e" || row.Content != "Weibo hot search E2E mock note" {
		t.Fatalf("unexpected sqlite hot_search identity/content fields: %#v", row)
	}
	if row.CreateTime != 1710001100 || row.CreateDateTime != "2024-03-10 00:18:20+08:00" {
		t.Fatalf("unexpected sqlite hot_search create fields: %#v", row)
	}
	if row.LikedCount != "366" || row.CommentsCount != "37" || row.SharedCount != "38" {
		t.Fatalf("unexpected sqlite hot_search count fields: %#v", row)
	}
	if row.NoteURL != "https://m.weibo.cn/detail/wb_hot_note_e2e" || row.IPLocation != "深圳" {
		t.Fatalf("unexpected sqlite hot_search url/location fields: %#v", row)
	}
	if row.ImageList != "https://example.test/wb-hot-image.jpg" || row.VideoURL != "https://example.test/wb-hot-720.mp4" {
		t.Fatalf("unexpected sqlite hot_search media fields: %#v", row)
	}
	if row.UserID != "wb-hot-author" || row.Nickname != "mock weibo hot author" || row.Gender != "f" {
		t.Fatalf("unexpected sqlite hot_search user fields: %#v", row)
	}
	if row.ProfileURL != "https://m.weibo.cn/u/wb-hot-author" || row.Avatar != "https://example.test/wb-hot-avatar.png" {
		t.Fatalf("unexpected sqlite hot_search profile fields: %#v", row)
	}
	if row.SourceKeyword != "mock hot weibo" || row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("unexpected sqlite hot_search source/timestamps: %#v", row)
	}
}

func runCLIWeiboSearchWithMockServers(t *testing.T, saveDataOption string) cliWeiboResult {
	t.Helper()
	return runCLIWeiboWithMockServers(t, "search", saveDataOption, nil, false, false)
}

func runCLIWeiboDetailWithMockServers(t *testing.T, saveDataOption string) cliWeiboResult {
	t.Helper()
	return runCLIWeiboWithMockServers(t, "detail", saveDataOption, []string{"wb_detail_e2e"}, false, false)
}

func runCLIWeiboDetailCommentsWithMockServers(t *testing.T, saveDataOption string) cliWeiboResult {
	t.Helper()
	return runCLIWeiboWithMockServers(t, "detail", saveDataOption, []string{"wb_detail_e2e"}, true, true)
}

func runCLIWeiboCreatorWithMockServers(t *testing.T, saveDataOption string) cliWeiboResult {
	t.Helper()
	return runCLIWeiboWithMockServers(t, "creator", saveDataOption, []string{"wb-creator-user"}, false, false)
}

func runCLIWeiboHotSearchWithMockServers(t *testing.T, saveDataOption string) cliWeiboResult {
	t.Helper()
	return runCLIWeiboWithMockServers(t, "hot_search", saveDataOption, nil, false, false)
}

func runCLIWeiboWithMockServers(t *testing.T, crawlerType, saveDataOption string, noteIDs []string, enableComments, enableSubComments bool) cliWeiboResult {
	t.Helper()

	repoRoot := e2e.RepoRoot(t)

	mockServer := newWeiboSearchMockServer(t)
	defer mockServer.Close()

	host, port := serverHostPort(t, mockServer.URL)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	sqlitePath := filepath.Join(tmpDir, "media_crawler.db")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	specifiedIDsYAML := ""
	if len(noteIDs) > 0 {
		quotedIDs := make([]string, 0, len(noteIDs))
		for _, noteID := range noteIDs {
			quotedIDs = append(quotedIDs, fmt.Sprintf("%q", noteID))
		}
		switch crawlerType {
		case "creator":
			specifiedIDsYAML = fmt.Sprintf("specifiedCreatorIds: [%s]\nspecifiedCreatorURLs: [%s]\n", strings.Join(quotedIDs, ", "), strings.Join(quotedIDs, ", "))
		default:
			specifiedIDsYAML = fmt.Sprintf("specifiedNoteIds: [%s]\nspecifiedNoteURLs: [%s]\n", strings.Join(quotedIDs, ", "), strings.Join(quotedIDs, ", "))
		}
	}

	configYAML := fmt.Sprintf(`
platform: "wb"
keywords:
  - "mock-wb"
crawlerType: %q
saveDataOption: %q
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
sqliteDbPath: %q
startPage: 1
crawlerMaxNotesCount: 10
maxConcurrencyNum: 1
requestTimeOut: 2
hotSearchMaxItemsCount: 1
hotSearchPerItemContentCount: 1
enableGetComments: %t
enableGetSubComments: %t
enableCheckpoint: false
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  weiboApiUrl: %q
  weiboHotBandUrl: %q
%s`, crawlerType, saveDataOption, mockServer.URL, dataDir, sqlitePath, enableComments, enableSubComments, host, port, mockServer.URL, mockServer.URL+"/ajax/statuses/hot_band", specifiedIDsYAML)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	runCmd := exec.CommandContext(ctx, binaryPath)
	runCmd.Dir = tmpDir
	runCmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+dataDir,
		"MCPRO_WEIBO_API_URL="+mockServer.URL,
		"MCPRO_WEIBO_HOT_BAND_URL="+mockServer.URL+"/ajax/statuses/hot_band",
	)
	if saveDataOption == "sqlite" {
		runCmd.Env = append(runCmd.Env, "MCPRO_SQLITE_DB_PATH="+sqlitePath)
	}

	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("weibo crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("weibo crawler command failed: %v\n%s", err, output)
	}
	return cliWeiboResult{output: output, dataDir: dataDir, sqlitePath: sqlitePath}
}

func newWeiboSearchMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/accounts":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]any{
					"accounts": []map[string]any{
						{
							"client_id": "weibo-client",
							"platforms": map[string]any{
								"wb":    map[string]any{"has_cookies": true},
								"weibo": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"wb": "mock-weibo", "weibo": "mock-weibo"},
						},
					},
				},
			})
		case "/api/cookies/wb", "/api/cookies/weibo":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "SUB=mock-weibo-sub; XSRF-TOKEN=mock-xsrf"},
			})
		case "/api/config":
			writeJSON(t, w, map[string]any{"ok": 1, "data": map[string]bool{"login": true}})
		case "/api/container/getIndex":
			containerID := r.URL.Query().Get("containerid")
			if containerID == "100505wb-creator-user" {
				writeJSON(t, w, weiboCreatorInfoPayload())
				return
			}
			if containerID == "107603wb-creator-user" {
				writeJSON(t, w, weiboCreatorNotesPayload())
				return
			}
			if strings.Contains(containerID, "mock hot weibo") {
				writeJSON(t, w, weiboHotSearchPayload())
				return
			}
			if r.URL.Query().Get("page") != "1" {
				writeJSON(t, w, map[string]any{"ok": 1, "data": map[string]any{"cards": []map[string]any{}}})
				return
			}
			writeJSON(t, w, weiboSearchPayload())
		case "/detail/wb_detail_e2e":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(weiboDetailHTML()))
		case "/comments/hotflow":
			writeJSON(t, w, weiboCommentsPayload())
		case "/ajax/statuses/hot_band":
			writeJSON(t, w, weiboHotBandPayload())
		default:
			t.Fatalf("unexpected weibo request: %s %s", r.Method, r.URL.String())
		}
	}))
}

func readJSONRows(t *testing.T, path string, output []byte) []map[string]any {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read json rows from %s: %v\ncommand output:\n%s", path, err, output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal json rows from %s: %v\n%s", path, err, body)
	}
	return rows
}

func readCSVRecords(t *testing.T, path string, output []byte) [][]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open csv records from %s: %v\ncommand output:\n%s", path, err, output)
	}
	defer file.Close()
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read csv records from %s: %v\ncommand output:\n%s", path, err, output)
	}
	return records
}

func weiboCreatorInfoPayload() map[string]any {
	return map[string]any{
		"ok": 1,
		"data": map[string]any{
			"tabsInfo": map[string]any{
				"tabs": []map[string]string{
					{"tabKey": "weibo", "containerid": "107603wb-creator-user"},
				},
			},
			"userInfo": map[string]any{
				"id":              "wb-creator-user",
				"screen_name":     "mock weibo creator",
				"gender":          "f",
				"avatar_hd":       "https://example.test/wb-creator-avatar.png",
				"description":     "weibo creator signature",
				"source":          "来自杭州",
				"follow_count":    12,
				"followers_count": 3456,
			},
		},
	}
}

func weiboCreatorResumeInfoPayload() map[string]any {
	return map[string]any{
		"ok": 1,
		"data": map[string]any{
			"tabsInfo": map[string]any{
				"tabs": []map[string]string{
					{"tabKey": "weibo", "containerid": "107603wb-creator-resume"},
				},
			},
			"userInfo": map[string]any{
				"id":              "wb-creator-resume",
				"screen_name":     "mock weibo creator resume",
				"gender":          "f",
				"avatar_hd":       "https://example.test/wb-creator-resume-avatar.png",
				"description":     "weibo creator resume signature",
				"source":          "来自杭州",
				"follow_count":    12,
				"followers_count": 3456,
			},
		},
	}
}

func weiboSearchResumePayload(noteID, text, authorID string) map[string]any {
	return map[string]any{
		"ok": 1,
		"data": map[string]any{
			"cards": []map[string]any{
				{
					"card_type": 11,
					"card_group": []map[string]any{
						{
							"card_type": 9,
							"mblog": map[string]any{
								"id":              noteID,
								"text":            "<span>" + text + "</span>",
								"created_at":      "Sat Mar 09 16:11:40 +0000 2024",
								"attitudes_count": 76,
								"comments_count":  17,
								"reposts_count":   18,
								"region_name":     "发布于 上海",
								"pics": []map[string]any{
									{"url": "https://example.test/" + noteID + ".jpg"},
								},
								"user": map[string]string{
									"id":                authorID,
									"screen_name":       "mock " + authorID,
									"gender":            "m",
									"profile_url":       "https://m.weibo.cn/u/" + authorID,
									"profile_image_url": "https://example.test/" + authorID + ".png",
								},
							},
						},
					},
				},
			},
		},
	}
}

func weiboCreatorResumeNotesPayload(noteID, text, authorID, nextSinceID string, total int) map[string]any {
	return map[string]any{
		"ok": 1,
		"data": map[string]any{
			"cardlistInfo": map[string]any{"since_id": nextSinceID, "total": total},
			"cards": []map[string]any{
				{
					"card_type": 9,
					"mblog": map[string]any{
						"id":              noteID,
						"text":            "<span>" + text + "</span>",
						"created_at":      "Sat Mar 09 16:16:40 +0000 2024",
						"attitudes_count": 266,
						"comments_count":  27,
						"reposts_count":   28,
						"region_name":     "发布于 成都",
						"pics": []map[string]any{
							{"url": "https://example.test/" + noteID + ".jpg"},
						},
						"user": map[string]string{
							"id":                authorID,
							"screen_name":       "mock " + authorID,
							"gender":            "m",
							"profile_url":       "https://m.weibo.cn/u/" + authorID,
							"profile_image_url": "https://example.test/" + authorID + ".png",
						},
					},
				},
			},
		},
	}
}

func weiboCreatorNotesPayload() map[string]any {
	return map[string]any{
		"ok": 1,
		"data": map[string]any{
			"cardlistInfo": map[string]any{"since_id": "0", "total": 1},
			"cards": []map[string]any{
				{
					"card_type": 9,
					"mblog": map[string]any{
						"id":              "wb_creator_note_e2e",
						"text":            "<span>Weibo creator E2E mock note</span>",
						"created_at":      "Sat Mar 09 16:16:40 +0000 2024",
						"attitudes_count": 266,
						"comments_count":  27,
						"reposts_count":   28,
						"region_name":     "发布于 成都",
						"pics": []map[string]any{
							{"url": "https://example.test/wb-creator-image.jpg"},
						},
						"page_info": map[string]any{
							"type": "video",
							"urls": map[string]string{
								"mp4_720p_mp4": "https://example.test/wb-creator-720.mp4",
							},
						},
						"user": map[string]string{
							"id":                "wb-creator-author",
							"screen_name":       "mock weibo creator author",
							"gender":            "m",
							"profile_url":       "https://m.weibo.cn/u/wb-creator-author",
							"profile_image_url": "https://example.test/wb-creator-author-avatar.png",
						},
					},
				},
			},
		},
	}
}

func weiboCommentsPayload() map[string]any {
	return map[string]any{
		"ok": 1,
		"data": map[string]any{
			"max_id":      0,
			"max_id_type": 0,
			"data": []map[string]any{
				{
					"id":           "wb_comment_root",
					"text":         "<span>root weibo comment</span>",
					"created_at":   "Sat Mar 09 16:15:00 +0000 2024",
					"total_number": 1,
					"like_count":   12,
					"source":       "来自广东",
					"rootid":       "",
					"user": map[string]string{
						"id":                "wb-commenter",
						"screen_name":       "mock wb commenter",
						"gender":            "m",
						"profile_url":       "https://m.weibo.cn/u/wb-commenter",
						"profile_image_url": "https://example.test/wb-comment-avatar.png",
					},
					"comments": []map[string]any{
						{
							"id":           "wb_comment_sub",
							"text":         "<em>sub weibo comment</em>",
							"created_at":   "Sat Mar 09 16:15:10 +0000 2024",
							"total_number": 0,
							"like_count":   3,
							"source":       "来自深圳",
							"rootid":       "wb_comment_root",
							"user": map[string]string{
								"id":                "wb-sub-commenter",
								"screen_name":       "mock wb sub commenter",
								"gender":            "f",
								"profile_url":       "https://m.weibo.cn/u/wb-sub-commenter",
								"profile_image_url": "https://example.test/wb-sub-comment-avatar.png",
							},
						},
					},
				},
			},
		},
	}
}

func weiboHotBandPayload() map[string]any {
	return map[string]any{
		"ok": 1,
		"data": []map[string]any{
			{
				"word":        "mock hot weibo",
				"word_scheme": "mock hot weibo",
				"realpos":     1,
				"num":         999,
				"label_name":  "热",
				"mid":         "wb-hot-mid",
				"is_ad":       false,
			},
		},
	}
}

func weiboHotBandResumePayload() map[string]any {
	return map[string]any{
		"ok": 1,
		"data": []map[string]any{
			{
				"word":        "resume hot one",
				"word_scheme": "resume hot one",
				"realpos":     1,
				"num":         1999,
				"label_name":  "热",
				"mid":         "wb-hot-resume-mid-1",
				"is_ad":       false,
			},
			{
				"word":        "resume hot two",
				"word_scheme": "resume hot two",
				"realpos":     2,
				"num":         1888,
				"label_name":  "新",
				"mid":         "wb-hot-resume-mid-2",
				"is_ad":       false,
			},
		},
	}
}

func weiboHotSearchPayload() map[string]any {
	return map[string]any{
		"ok": 1,
		"data": map[string]any{
			"cards": []map[string]any{
				{
					"card_type": 9,
					"mblog": map[string]any{
						"id":              "wb_hot_note_e2e",
						"text":            "<span>Weibo hot search E2E mock note</span>",
						"created_at":      "Sat Mar 09 16:18:20 +0000 2024",
						"attitudes_count": 366,
						"comments_count":  37,
						"reposts_count":   38,
						"region_name":     "发布于 深圳",
						"pics": []map[string]any{
							{"url": "https://example.test/wb-hot-image.jpg"},
						},
						"page_info": map[string]any{
							"type": "video",
							"urls": map[string]string{
								"mp4_720p_mp4": "https://example.test/wb-hot-720.mp4",
							},
						},
						"user": map[string]string{
							"id":                "wb-hot-author",
							"screen_name":       "mock weibo hot author",
							"gender":            "f",
							"profile_url":       "https://m.weibo.cn/u/wb-hot-author",
							"profile_image_url": "https://example.test/wb-hot-avatar.png",
						},
					},
				},
			},
		},
	}
}

func weiboHotSearchResumePayload(noteID, text, authorID string) map[string]any {
	return map[string]any{
		"ok": 1,
		"data": map[string]any{
			"cards": []map[string]any{
				{
					"card_type": 9,
					"mblog": map[string]any{
						"id":              noteID,
						"text":            "<span>" + text + "</span>",
						"created_at":      "Sat Mar 09 16:18:20 +0000 2024",
						"attitudes_count": 466,
						"comments_count":  47,
						"reposts_count":   48,
						"region_name":     "发布于 深圳",
						"pics": []map[string]any{
							{"url": "https://example.test/" + noteID + ".jpg"},
						},
						"user": map[string]string{
							"id":                authorID,
							"screen_name":       "mock " + authorID,
							"gender":            "f",
							"profile_url":       "https://m.weibo.cn/u/" + authorID,
							"profile_image_url": "https://example.test/" + authorID + ".png",
						},
					},
				},
			},
		},
	}
}

func weiboDetailHTML() string {
	payload := `[{"status":{"id":"wb_detail_e2e","text":"<p>Weibo detail E2E mock note</p>","created_at":"Sat Mar 09 16:13:20 +0000 2024","attitudes_count":166,"comments_count":17,"reposts_count":18,"region_name":"发布于 北京","pics":[{"url":"https://example.test/wb-detail-image-1.jpg"}],"page_info":{"type":"video","urls":{"mp4_720p_mp4":"https://example.test/wb-detail-720.mp4","mp4_ld_mp4":"https://example.test/wb-detail-ld.mp4"}},"user":{"id":"wb-detail-author","screen_name":"mock weibo detail author","gender":"f","profile_url":"https://m.weibo.cn/u/wb-detail-author","profile_image_url":"https://example.test/wb-detail-avatar.png"}}}]`
	return "<html><body><script>var $render_data = " + payload + "[0]</script></body></html>"
}

func weiboDetailHTMLFor(noteID, text, authorID string) string {
	payload := fmt.Sprintf(`[{"status":{"id":%q,"text":%q,"created_at":"Sat Mar 09 16:13:20 +0000 2024","attitudes_count":166,"comments_count":17,"reposts_count":18,"region_name":"发布于 北京","pics":[{"url":"https://example.test/wb-detail-image-1.jpg"}],"page_info":{"type":"video","urls":{"mp4_720p_mp4":"https://example.test/wb-detail-720.mp4","mp4_ld_mp4":"https://example.test/wb-detail-ld.mp4"}},"user":{"id":%q,"screen_name":%q,"gender":"f","profile_url":%q,"profile_image_url":"https://example.test/wb-detail-avatar.png"}}}]`,
		noteID,
		"<p>"+text+"</p>",
		authorID,
		"mock "+authorID,
		"https://m.weibo.cn/u/"+authorID,
	)
	return "<html><body><script>var $render_data = " + payload + "[0]</script></body></html>"
}

func weiboSearchPayload() map[string]any {
	return map[string]any{
		"ok": 1,
		"data": map[string]any{
			"cards": []map[string]any{
				{
					"card_type": 11,
					"card_group": []map[string]any{
						{
							"card_type": 9,
							"mblog": map[string]any{
								"id":              "wb_note_e2e",
								"text":            "<span>Weibo E2E mock note</span>",
								"created_at":      "Sat Mar 09 16:11:40 +0000 2024",
								"attitudes_count": 66,
								"comments_count":  7,
								"reposts_count":   8,
								"region_name":     "发布于 上海",
								"pics": []map[string]any{
									{"url": "https://example.test/wb-image-1.jpg"},
									{"url": "https://example.test/wb-image-2.jpg"},
								},
								"page_info": map[string]any{
									"type": "video",
									"urls": map[string]string{
										"mp4_720p_mp4": "https://example.test/wb-720.mp4",
										"mp4_hd_mp4":   "https://example.test/wb-hd.mp4",
									},
								},
								"user": map[string]string{
									"id":                "wb-author",
									"screen_name":       "mock weibo author",
									"gender":            "m",
									"profile_url":       "https://m.weibo.cn/u/wb-author",
									"profile_image_url": "https://example.test/wb-avatar.png",
								},
							},
						},
					},
				},
			},
		},
	}
}
