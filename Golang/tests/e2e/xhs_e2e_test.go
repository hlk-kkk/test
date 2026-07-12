// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/xhs_e2e_test.go
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
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/e2e"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCLIXhsSearchJSONWithMockServers(t *testing.T) {
	result := runCLIXhsSearchWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "xhs", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read result file: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal result json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one saved row, got %d: %s", len(rows), body)
	}
	if rows[0]["note_id"] != "note_e2e" {
		t.Fatalf("unexpected note_id: %#v", rows[0]["note_id"])
	}
	if rows[0]["title"] != "E2E mock note" {
		t.Fatalf("unexpected title: %#v", rows[0]["title"])
	}
	if rows[0]["source_keyword"] != "mock-keyword" {
		t.Fatalf("unexpected source_keyword: %#v", rows[0]["source_keyword"])
	}
	if rows[0]["note_url"] != "https://www.xiaohongshu.com/explore/note_e2e?xsec_token=token_e2e&xsec_source=pc_search" {
		t.Fatalf("unexpected note_url: %#v", rows[0]["note_url"])
	}
	if _, ok := rows[0]["add_ts"]; ok {
		t.Fatalf("json file output should match Python and omit add_ts: %#v", rows[0]["add_ts"])
	}
	if rows[0]["time"] != "1710000000000" {
		t.Fatalf("unexpected time type/value: %#v", rows[0]["time"])
	}
	if rows[0]["last_update_time"] != "1710000001000" {
		t.Fatalf("unexpected last_update_time type/value: %#v", rows[0]["last_update_time"])
	}
}

func TestCLIXhsHotSearchPlaceholderWithMockServers(t *testing.T) {
	result := runCLIXhsHotSearchPlaceholderWithMockServers(t)
	if strings.Contains(string(result.output), "Unknown crawler type") {
		t.Fatalf("xhs hot_search should be accepted like Python placeholder, got output:\n%s", result.output)
	}

	xhsDir := filepath.Join(result.dataDir, "xhs")
	entries, err := os.ReadDir(xhsDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read xhs data dir: %v\ncommand output:\n%s", err, result.output)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no xhs hot_search output files for Python placeholder parity, got %d entries in %s", len(entries), xhsDir)
	}
}

func TestCLIXhsSearchDoesNotWriteCheckpointWhenDisabled(t *testing.T) {
	result := runCLIXhsSearchWithMockServers(t, "json")

	checkpointDir := filepath.Join(result.dataDir, "checkpoints")
	entries, err := os.ReadDir(checkpointDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read checkpoint dir: %v\ncommand output:\n%s", err, result.output)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no checkpoint files when enableCheckpoint=false, got %d entries in %s", len(entries), checkpointDir)
	}
}

func TestCLIXhsSearchCheckpointResumeSkipsCompletedPages(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	searchCallsByPage := make(map[int]int)
	feedCallsByNoteID := make(map[string]int)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var payload map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&payload)
		}

		switch r.URL.Path {
		case "/api/accounts":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]any{
					"accounts": []map[string]any{
						{
							"client_id": "xhs-resume-client",
							"platforms": map[string]any{
								"xhs": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"xhs": "mock-xhs"},
						},
					},
				},
			})
		case "/api/cookies/xhs":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "web_session=resume-session; a1=resume-a1"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/xhs/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_s":           "resume-xs",
					"x_t":           "resume-xt",
					"x_s_common":    "resume-common",
					"x_b3_trace_id": "resume-trace",
				},
			})
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{"success": true, "message": "ok", "code": 0},
				},
			})
		case "/api/sns/web/v1/search/notes":
			page := int(payloadNumber(payload, "page"))
			searchCallsByPage[page]++
			switch page {
			case 1:
				writeJSON(t, w, xhsResumeSearchPayload("note_resume_1", "token_resume_1"))
			case 2:
				writeJSON(t, w, xhsResumeSearchPayload("note_resume_2", "token_resume_2"))
			default:
				writeJSON(t, w, map[string]any{
					"code":    0,
					"success": true,
					"msg":     "ok",
					"data":    map[string]any{"has_more": false, "items": []map[string]any{}},
				})
			}
		case "/api/sns/web/v1/feed":
			noteID := fmt.Sprint(payload["source_note_id"])
			feedCallsByNoteID[noteID]++
			switch noteID {
			case "note_resume_1":
				writeJSON(t, w, xhsResumeFeedPayload("note_resume_1", "Resume mock note 1", "token_resume_1"))
			case "note_resume_2":
				writeJSON(t, w, xhsResumeFeedPayload("note_resume_2", "Resume mock note 2", "token_resume_2"))
			default:
				t.Fatalf("unexpected feed note id: %s", noteID)
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
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

	writeResumeConfig := func(maxNotes int) {
		t.Helper()
		configYAML := fmt.Sprintf(`
platform: "xhs"
keywords:
  - "resume-keyword"
crawlerType: "search"
saveDataOption: "json"
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
startPage: 1
crawlerMaxNotesCount: %d
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
  xhsApiUrl: %q
  xhsIndexUrl: %q
`, mockServer.URL, dataDir, maxNotes, host, port, mockServer.URL, mockServer.URL)
		if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
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
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	writeResumeConfig(1)
	firstOutput := runCrawler()
	if searchCallsByPage[1] != 1 || feedCallsByNoteID["note_resume_1"] != 1 {
		t.Fatalf("first run did not crawl page 1 once, search=%#v feed=%#v\n%s", searchCallsByPage, feedCallsByNoteID, firstOutput)
	}

	writeResumeConfig(2)
	secondOutput := runCrawler()
	if searchCallsByPage[1] != 1 {
		t.Fatalf("resume should not re-fetch completed page 1, search calls=%#v\n%s", searchCallsByPage, secondOutput)
	}
	if searchCallsByPage[2] != 1 || feedCallsByNoteID["note_resume_2"] != 1 {
		t.Fatalf("resume did not crawl page 2 once, search=%#v feed=%#v\n%s", searchCallsByPage, feedCallsByNoteID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "xhs", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read resume xhs json: %v\n%s", err, secondOutput)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal resume xhs json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique XHS rows after resume, got %d: %s", len(rows), body)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["note_id"])] = true
	}
	if !seen["note_resume_1"] || !seen["note_resume_2"] {
		t.Fatalf("resume output missing expected XHS note IDs: seen=%#v rows=%s", seen, body)
	}
}

func TestCLIXhsDetailCheckpointResumeSkipsCompletedIDs(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	feedCallsByNoteID := make(map[string]int)
	allowSecondDetail := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var payload map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&payload)
		}

		switch r.URL.Path {
		case "/api/accounts":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]any{
					"accounts": []map[string]any{
						{
							"client_id": "xhs-detail-resume-client",
							"platforms": map[string]any{
								"xhs": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"xhs": "mock-xhs"},
						},
					},
				},
			})
		case "/api/cookies/xhs":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "web_session=detail-resume-session; a1=detail-resume-a1"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/xhs/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_s":           "detail-resume-xs",
					"x_t":           "detail-resume-xt",
					"x_s_common":    "detail-resume-common",
					"x_b3_trace_id": "detail-resume-trace",
				},
			})
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{"success": true, "message": "ok", "code": 0},
				},
			})
		case "/api/sns/web/v1/feed":
			noteID := fmt.Sprint(payload["source_note_id"])
			feedCallsByNoteID[noteID]++
			switch noteID {
			case "xhs_detail_resume_note_1":
				writeJSON(t, w, xhsResumeFeedPayload(noteID, "XHS detail resume note 1", "token_detail_resume_1"))
			case "xhs_detail_resume_note_2":
				if !allowSecondDetail {
					w.WriteHeader(http.StatusInternalServerError)
					writeJSON(t, w, map[string]any{
						"code":    9999,
						"success": false,
						"msg":     "transient second detail failure",
						"data":    map[string]any{},
					})
					return
				}
				writeJSON(t, w, xhsResumeFeedPayload(noteID, "XHS detail resume note 2", "token_detail_resume_2"))
			default:
				t.Fatalf("unexpected detail feed note id: %s", noteID)
			}
		case "/explore/xhs_detail_resume_note_2":
			http.Error(w, "transient second detail html fallback failure", http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
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
platform: "xhs"
keywords:
  - "unused"
crawlerType: "detail"
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
specifiedNoteURLs:
  - "https://www.xiaohongshu.com/explore/xhs_detail_resume_note_1?xsec_token=token_detail_resume_1&xsec_source=pc_note_detail"
  - "https://www.xiaohongshu.com/explore/xhs_detail_resume_note_2?xsec_token=token_detail_resume_2&xsec_source=pc_note_detail"
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  xhsApiUrl: %q
  xhsIndexUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL)
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
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if feedCallsByNoteID["xhs_detail_resume_note_1"] != 1 || feedCallsByNoteID["xhs_detail_resume_note_2"] != 1 {
		t.Fatalf("first run did not attempt each detail once, feed=%#v\n%s", feedCallsByNoteID, firstOutput)
	}

	allowSecondDetail = true
	secondOutput := runCrawler()
	if feedCallsByNoteID["xhs_detail_resume_note_1"] != 1 {
		t.Fatalf("resume should not re-fetch completed note 1, feed=%#v\n%s", feedCallsByNoteID, secondOutput)
	}
	if feedCallsByNoteID["xhs_detail_resume_note_2"] != 2 {
		t.Fatalf("resume should retry failed note 2 exactly once, feed=%#v\n%s", feedCallsByNoteID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "xhs", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read detail resume xhs json: %v\n%s", err, secondOutput)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal detail resume xhs json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique XHS detail rows after resume, got %d: %s", len(rows), body)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["note_id"])] = true
	}
	if !seen["xhs_detail_resume_note_1"] || !seen["xhs_detail_resume_note_2"] {
		t.Fatalf("detail resume output missing expected XHS note IDs: seen=%#v rows=%s", seen, body)
	}
}

func TestCLIXhsDetailJSONWithMockServers(t *testing.T) {
	result := runCLIXhsDetailWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "xhs", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read detail json result: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal detail json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one detail row, got %d: %s", len(rows), body)
	}
	assertXhsDetailRow(t, rows[0])
}

func TestCLIXhsDetailCSVWithMockServers(t *testing.T) {
	result := runCLIXhsDetailWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "xhs", "1_detail_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open detail csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read detail csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected detail csv header and one row, got %d: %#v", len(records), records)
	}
	if csvCell(records, "note_id") != "xhs_detail_e2e" || csvCell(records, "title") != "XHS detail note" {
		t.Fatalf("unexpected detail csv identity fields: %#v", records)
	}
	if csvCell(records, "note_url") != "https://www.xiaohongshu.com/explore/xhs_detail_e2e?xsec_token=token_detail_e2e&xsec_source=pc_note_detail" {
		t.Fatalf("unexpected detail csv note_url: %#v", csvCell(records, "note_url"))
	}
	if csvCell(records, "time") != "1710000000000" || csvCell(records, "last_update_time") != "1710000001000" {
		t.Fatalf("unexpected detail csv time fields: %#v", records)
	}
}

func TestCLIXhsDetailSQLiteWithMockServers(t *testing.T) {
	result := runCLIXhsDetailWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open detail sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	var row struct {
		NoteID         string `gorm:"column:note_id"`
		Title          string `gorm:"column:title"`
		NoteURL        string `gorm:"column:note_url"`
		Time           int64  `gorm:"column:time"`
		LastUpdateTime int64  `gorm:"column:last_update_time"`
	}
	if err := db.Table("xhs_note").Where("note_id = ?", "xhs_detail_e2e").First(&row).Error; err != nil {
		t.Fatalf("query detail sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	if row.Title != "XHS detail note" {
		t.Fatalf("unexpected sqlite detail title: %#v", row)
	}
	if row.NoteURL != "https://www.xiaohongshu.com/explore/xhs_detail_e2e?xsec_token=token_detail_e2e&xsec_source=pc_note_detail" {
		t.Fatalf("unexpected sqlite detail note_url: %#v", row.NoteURL)
	}
	if row.Time != 1710000000000 || row.LastUpdateTime != 1710000001000 {
		t.Fatalf("unexpected sqlite detail time fields: %#v", row)
	}
}

func TestCLIXhsCreatorJSONWithMockServers(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	var creatorProfileCalls, creatorNotesCalls, feedCalls int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var payload map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&payload)
		}

		switch r.URL.Path {
		case "/api/accounts":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]any{
					"accounts": []map[string]any{
						{
							"client_id": "xhs-creator-client",
							"platforms": map[string]any{
								"xhs": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"xhs": "mock-xhs"},
						},
					},
				},
			})
		case "/api/cookies/xhs":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "web_session=creator-session; a1=creator-a1"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/xhs/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_s":           "creator-xs",
					"x_t":           "creator-xt",
					"x_s_common":    "creator-common",
					"x_b3_trace_id": "creator-trace",
				},
			})
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{"success": true, "message": "ok", "code": 0},
				},
			})
		case "/user/profile/xhs_creator_e2e":
			creatorProfileCalls++
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(xhsCreatorHTML()))
		case "/api/sns/web/v1/user_posted":
			creatorNotesCalls++
			if got := r.URL.Query().Get("user_id"); got != "xhs_creator_e2e" {
				t.Fatalf("unexpected creator user_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"cursor":   "",
					"has_more": false,
					"notes": []map[string]any{
						{
							"note_id":       "xhs_creator_note_e2e",
							"xsec_token":    "token_creator_note",
							"xsec_source":   "pc_user",
							"display_title": "creator list note",
							"type":          "normal",
							"cover": map[string]string{
								"url": "https://example.test/xhs-creator-cover.png",
							},
						},
					},
				},
			})
		case "/api/sns/web/v1/feed":
			feedCalls++
			noteID := fmt.Sprint(payload["source_note_id"])
			if noteID != "xhs_creator_note_e2e" {
				t.Fatalf("unexpected creator feed note id: %s", noteID)
			}
			writeJSON(t, w, xhsResumeFeedPayload(noteID, "XHS creator note", "token_creator_note"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
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
platform: "xhs"
keywords:
  - "unused"
crawlerType: "creator"
saveDataOption: "json"
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
startPage: 1
crawlerMaxNotesCount: 1
maxConcurrencyNum: 1
requestTimeOut: 2
enableGetComments: false
enableGetSubComments: false
enableCheckpoint: true
perNoteMaxComments: 2
crawlerTimeSleep: 0
debugRequest: false
specifiedCreatorURLs:
  - "https://www.xiaohongshu.com/user/profile/xhs_creator_e2e?xsec_token=token_creator_profile&xsec_source=pc_user"
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  xhsApiUrl: %q
  xhsIndexUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	runCmd := exec.CommandContext(ctx, binaryPath)
	runCmd.Dir = tmpDir
	runCmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+dataDir,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if creatorProfileCalls != 1 || creatorNotesCalls != 1 || feedCalls != 1 {
		t.Fatalf("unexpected creator request counts profile=%d notes=%d feed=%d\n%s", creatorProfileCalls, creatorNotesCalls, feedCalls, output)
	}

	creatorPath := filepath.Join(dataDir, "xhs", "json", "creator_creator_"+time.Now().Format("2006-01-02")+".json")
	creatorBody, err := os.ReadFile(creatorPath)
	if err != nil {
		t.Fatalf("read creator json: %v\n%s", err, output)
	}
	var creators []map[string]any
	if err := json.Unmarshal(creatorBody, &creators); err != nil {
		t.Fatalf("unmarshal creator json: %v\n%s", err, creatorBody)
	}
	if len(creators) != 1 {
		t.Fatalf("expected one creator row, got %d: %s", len(creators), creatorBody)
	}
	creator := creators[0]
	if creator["user_id"] != "xhs_creator_e2e" || creator["nickname"] != "XHS creator mock" {
		t.Fatalf("unexpected creator identity fields: %#v", creator)
	}
	if creator["avatar"] != "https://example.test/xhs-creator-images.png" {
		t.Fatalf("unexpected Python-compatible creator avatar: %#v", creator["avatar"])
	}
	if creator["gender"] != "女" || creator["follows"] != "12" || creator["fans"] != "345" || creator["interaction"] != "678" {
		t.Fatalf("unexpected creator interaction fields: %#v", creator)
	}
	if creator["tag_list"] != `{"profession": "摄影师", "location": "上海"}` {
		t.Fatalf("unexpected creator tag_list: %#v", creator["tag_list"])
	}

	contentsPath := filepath.Join(dataDir, "xhs", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	contentsBody, err := os.ReadFile(contentsPath)
	if err != nil {
		t.Fatalf("read creator contents json: %v\n%s", err, output)
	}
	var contents []map[string]any
	if err := json.Unmarshal(contentsBody, &contents); err != nil {
		t.Fatalf("unmarshal creator contents json: %v\n%s", err, contentsBody)
	}
	if len(contents) != 1 {
		t.Fatalf("expected one creator content row, got %d: %s", len(contents), contentsBody)
	}
	if contents[0]["note_id"] != "xhs_creator_note_e2e" || contents[0]["title"] != "XHS creator note" {
		t.Fatalf("unexpected creator content row: %#v", contents[0])
	}
	if contents[0]["note_url"] != "https://www.xiaohongshu.com/explore/xhs_creator_note_e2e?xsec_token=token_creator_note&xsec_source=pc_user" {
		t.Fatalf("unexpected creator content note_url: %#v", contents[0]["note_url"])
	}
}

func TestCLIXhsCreatorCSVWithMockServers(t *testing.T) {
	result := runCLIXhsCreatorWithMockServers(t, "csv")

	creatorPath := filepath.Join(result.dataDir, "xhs", "1_creator_creator_"+time.Now().Format("2006-01-02")+".csv")
	creatorFile, err := os.Open(creatorPath)
	if err != nil {
		t.Fatalf("open creator csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer creatorFile.Close()

	creatorRecords, err := csv.NewReader(creatorFile).ReadAll()
	if err != nil {
		t.Fatalf("read creator csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(creatorRecords) != 2 {
		t.Fatalf("expected creator csv header and one row, got %d: %#v", len(creatorRecords), creatorRecords)
	}
	if csvCell(creatorRecords, "user_id") != "xhs_creator_e2e" || csvCell(creatorRecords, "nickname") != "XHS creator mock" {
		t.Fatalf("unexpected creator csv identity fields: %#v", creatorRecords)
	}
	if csvCell(creatorRecords, "avatar") != "https://example.test/xhs-creator-images.png" {
		t.Fatalf("unexpected creator csv avatar: %#v", csvCell(creatorRecords, "avatar"))
	}
	if csvCell(creatorRecords, "gender") != "女" || csvCell(creatorRecords, "tag_list") != `{"profession": "摄影师", "location": "上海"}` {
		t.Fatalf("unexpected creator csv profile fields: %#v", creatorRecords)
	}

	contentsPath := filepath.Join(result.dataDir, "xhs", "1_creator_contents_"+time.Now().Format("2006-01-02")+".csv")
	contentsFile, err := os.Open(contentsPath)
	if err != nil {
		t.Fatalf("open creator contents csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer contentsFile.Close()

	contentRecords, err := csv.NewReader(contentsFile).ReadAll()
	if err != nil {
		t.Fatalf("read creator contents csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(contentRecords) != 2 {
		t.Fatalf("expected content csv header and one row, got %d: %#v", len(contentRecords), contentRecords)
	}
	if csvCell(contentRecords, "note_id") != "xhs_creator_note_e2e" || csvCell(contentRecords, "title") != "XHS creator note" {
		t.Fatalf("unexpected creator content csv fields: %#v", contentRecords)
	}
	if csvCell(contentRecords, "note_url") != "https://www.xiaohongshu.com/explore/xhs_creator_note_e2e?xsec_token=token_creator_note&xsec_source=pc_user" {
		t.Fatalf("unexpected creator content csv note_url: %#v", csvCell(contentRecords, "note_url"))
	}
}

func TestCLIXhsCreatorSQLiteWithMockServers(t *testing.T) {
	result := runCLIXhsCreatorWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open creator sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	var creator struct {
		UserID      string `gorm:"column:user_id"`
		Nickname    string `gorm:"column:nickname"`
		Avatar      string `gorm:"column:avatar"`
		Gender      string `gorm:"column:gender"`
		Follows     string `gorm:"column:follows"`
		Fans        string `gorm:"column:fans"`
		Interaction string `gorm:"column:interaction"`
		TagList     string `gorm:"column:tag_list"`
	}
	if err := db.Table("xhs_creator").Where("user_id = ?", "xhs_creator_e2e").First(&creator).Error; err != nil {
		t.Fatalf("query xhs_creator sqlite row: %v\ncommand output:\n%s", err, result.output)
	}
	if creator.Nickname != "XHS creator mock" || creator.Avatar != "https://example.test/xhs-creator-images.png" {
		t.Fatalf("unexpected sqlite creator identity/profile fields: %#v", creator)
	}
	if creator.Gender != "女" || creator.Follows != "12" || creator.Fans != "345" || creator.Interaction != "678" {
		t.Fatalf("unexpected sqlite creator interaction fields: %#v", creator)
	}
	if creator.TagList != `{"profession": "摄影师", "location": "上海"}` {
		t.Fatalf("unexpected sqlite creator tag_list: %#v", creator.TagList)
	}

	var note struct {
		NoteID  string `gorm:"column:note_id"`
		Title   string `gorm:"column:title"`
		NoteURL string `gorm:"column:note_url"`
	}
	if err := db.Table("xhs_note").Where("note_id = ?", "xhs_creator_note_e2e").First(&note).Error; err != nil {
		t.Fatalf("query xhs_note sqlite row: %v\ncommand output:\n%s", err, result.output)
	}
	if note.Title != "XHS creator note" {
		t.Fatalf("unexpected sqlite creator note title: %#v", note)
	}
	if note.NoteURL != "https://www.xiaohongshu.com/explore/xhs_creator_note_e2e?xsec_token=token_creator_note&xsec_source=pc_user" {
		t.Fatalf("unexpected sqlite creator note_url: %#v", note.NoteURL)
	}
}

func TestCLIXhsCreatorDoesNotWriteCheckpointWhenDisabled(t *testing.T) {
	result := runCLIXhsCreatorWithMockServers(t, "json", false)

	checkpointDir := filepath.Join(result.dataDir, "checkpoints")
	entries, err := os.ReadDir(checkpointDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read checkpoint dir: %v\ncommand output:\n%s", err, result.output)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no checkpoint files when enableCheckpoint=false, got %d entries in %s", len(entries), checkpointDir)
	}
}

func TestCLIXhsCreatorCheckpointResumeByCreatorAndCursor(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	allowSecondPage := false
	creatorProfileCalls := 0
	creatorNotesCallsByCursor := make(map[string]int)
	feedCallsByNoteID := make(map[string]int)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var payload map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&payload)
		}

		switch r.URL.Path {
		case "/api/accounts":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]any{
					"accounts": []map[string]any{
						{
							"client_id": "xhs-creator-resume-client",
							"platforms": map[string]any{
								"xhs": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"xhs": "mock-xhs"},
						},
					},
				},
			})
		case "/api/cookies/xhs":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "web_session=creator-resume-session; a1=creator-resume-a1"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/xhs/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_s":           "creator-resume-xs",
					"x_t":           "creator-resume-xt",
					"x_s_common":    "creator-resume-common",
					"x_b3_trace_id": "creator-resume-trace",
				},
			})
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{"success": true, "message": "ok", "code": 0},
				},
			})
		case "/user/profile/xhs_creator_resume":
			creatorProfileCalls++
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(xhsCreatorHTML()))
		case "/api/sns/web/v1/user_posted":
			if got := r.URL.Query().Get("user_id"); got != "xhs_creator_resume" {
				t.Fatalf("unexpected creator resume user_id: %s", got)
			}
			cursor := r.URL.Query().Get("cursor")
			creatorNotesCallsByCursor[cursor]++
			switch cursor {
			case "":
				writeJSON(t, w, xhsCreatorNotesPagePayload("xhs_creator_resume_note_1", "token_creator_resume_1", "creator_resume_cursor_2", true))
			case "creator_resume_cursor_2":
				if !allowSecondPage {
					w.WriteHeader(http.StatusInternalServerError)
					writeJSON(t, w, map[string]any{
						"code":    9999,
						"success": false,
						"msg":     "transient second creator page failure",
						"data":    map[string]any{},
					})
					return
				}
				writeJSON(t, w, xhsCreatorNotesPagePayload("xhs_creator_resume_note_2", "token_creator_resume_2", "creator_resume_cursor_done", false))
			default:
				t.Fatalf("unexpected creator resume cursor: %s", cursor)
			}
		case "/api/sns/web/v1/feed":
			noteID := fmt.Sprint(payload["source_note_id"])
			feedCallsByNoteID[noteID]++
			switch noteID {
			case "xhs_creator_resume_note_1":
				writeJSON(t, w, xhsResumeFeedPayload(noteID, "XHS creator resume note 1", "token_creator_resume_1"))
			case "xhs_creator_resume_note_2":
				writeJSON(t, w, xhsResumeFeedPayload(noteID, "XHS creator resume note 2", "token_creator_resume_2"))
			default:
				t.Fatalf("unexpected creator resume feed note id: %s", noteID)
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
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
platform: "xhs"
keywords:
  - "unused"
crawlerType: "creator"
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
specifiedCreatorURLs:
  - "https://www.xiaohongshu.com/user/profile/xhs_creator_resume?xsec_token=token_creator_resume_profile&xsec_source=pc_user"
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  xhsApiUrl: %q
  xhsIndexUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	runCrawler := func(expectFailureLog bool) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		runCmd := exec.CommandContext(ctx, binaryPath)
		runCmd.Dir = tmpDir
		runCmd.Env = append(os.Environ(),
			"RELATION_DB_HOST=127.0.0.1",
			"RELATION_DB_PORT=1",
			"MCPRO_DATA_DIR="+dataDir,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("crawler command timed out\n%s", output)
		}
		if expectFailureLog {
			if !strings.Contains(string(output), "transient second creator page failure") {
				t.Fatalf("expected first creator run to log transient page 2 failure, err=%v\n%s", err, output)
			}
			return output
		}
		if err != nil {
			t.Fatalf("crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler(true)
	if creatorProfileCalls != 1 || creatorNotesCallsByCursor[""] != 1 || feedCallsByNoteID["xhs_creator_resume_note_1"] != 1 {
		t.Fatalf("first creator run did not process first page once, profile=%d notes=%#v feed=%#v\n%s", creatorProfileCalls, creatorNotesCallsByCursor, feedCallsByNoteID, firstOutput)
	}
	if creatorNotesCallsByCursor["creator_resume_cursor_2"] != 1 {
		t.Fatalf("first creator run did not stop on second page once, notes=%#v\n%s", creatorNotesCallsByCursor, firstOutput)
	}

	allowSecondPage = true
	secondOutput := runCrawler(false)
	if creatorProfileCalls != 1 {
		t.Fatalf("resume should not re-fetch completed creator profile, profile=%d\n%s", creatorProfileCalls, secondOutput)
	}
	if creatorNotesCallsByCursor[""] != 1 {
		t.Fatalf("resume should not re-fetch first creator page, notes=%#v\n%s", creatorNotesCallsByCursor, secondOutput)
	}
	if creatorNotesCallsByCursor["creator_resume_cursor_2"] != 2 || feedCallsByNoteID["xhs_creator_resume_note_2"] != 1 {
		t.Fatalf("resume did not continue from saved creator cursor exactly once, notes=%#v feed=%#v\n%s", creatorNotesCallsByCursor, feedCallsByNoteID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "xhs", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read creator resume xhs json: %v\n%s", err, secondOutput)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal creator resume xhs json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique XHS creator rows after resume, got %d: %s", len(rows), body)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["note_id"])] = true
	}
	if !seen["xhs_creator_resume_note_1"] || !seen["xhs_creator_resume_note_2"] {
		t.Fatalf("creator resume output missing expected XHS note IDs: seen=%#v rows=%s", seen, body)
	}
}

func TestCLIXhsHomefeedJSONWithMockServers(t *testing.T) {
	result := runCLIXhsHomefeedWithMockServers(t, "json", false)

	resultPath := filepath.Join(result.dataDir, "xhs", "json", "homefeed_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read homefeed json result: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal homefeed json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one homefeed row, got %d: %s", len(rows), body)
	}
	assertXhsHomefeedRow(t, rows[0])
}

func TestCLIXhsHomefeedCSVWithMockServers(t *testing.T) {
	result := runCLIXhsHomefeedWithMockServers(t, "csv", false)

	resultPath := filepath.Join(result.dataDir, "xhs", "1_homefeed_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open homefeed csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read homefeed csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected homefeed csv header and one row, got %d: %#v", len(records), records)
	}
	if csvCell(records, "note_id") != "xhs_homefeed_note_e2e" || csvCell(records, "title") != "XHS homefeed note" {
		t.Fatalf("unexpected homefeed csv identity fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "homefeed" {
		t.Fatalf("unexpected homefeed csv source_keyword: %#v", csvCell(records, "source_keyword"))
	}
	if csvCell(records, "note_url") != "https://www.xiaohongshu.com/explore/xhs_homefeed_note_e2e?xsec_token=token_homefeed_note&xsec_source=pc_feed" {
		t.Fatalf("unexpected homefeed csv note_url: %#v", csvCell(records, "note_url"))
	}
}

func TestCLIXhsHomefeedSQLiteWithMockServers(t *testing.T) {
	result := runCLIXhsHomefeedWithMockServers(t, "sqlite", false)

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open homefeed sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	var row struct {
		NoteID        string `gorm:"column:note_id"`
		Title         string `gorm:"column:title"`
		NoteURL       string `gorm:"column:note_url"`
		SourceKeyword string `gorm:"column:source_keyword"`
	}
	if err := db.Table("xhs_note").Where("note_id = ?", "xhs_homefeed_note_e2e").First(&row).Error; err != nil {
		t.Fatalf("query homefeed sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	if row.Title != "XHS homefeed note" || row.SourceKeyword != "homefeed" {
		t.Fatalf("unexpected sqlite homefeed fields: %#v", row)
	}
	if row.NoteURL != "https://www.xiaohongshu.com/explore/xhs_homefeed_note_e2e?xsec_token=token_homefeed_note&xsec_source=pc_feed" {
		t.Fatalf("unexpected sqlite homefeed note_url: %#v", row.NoteURL)
	}
}

func TestCLIXhsHomefeedDoesNotWriteCheckpointWhenDisabled(t *testing.T) {
	result := runCLIXhsHomefeedWithMockServers(t, "json", false)

	checkpointDir := filepath.Join(result.dataDir, "checkpoints")
	entries, err := os.ReadDir(checkpointDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read checkpoint dir: %v\ncommand output:\n%s", err, result.output)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no checkpoint files when enableCheckpoint=false, got %d entries in %s", len(entries), checkpointDir)
	}
}

func TestCLIXhsHomefeedCheckpointResumeByCursorAndIndex(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	allowSecondPage := false
	homefeedCallsByCursorIndex := make(map[string]int)
	feedCallsByNoteID := make(map[string]int)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var payload map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&payload)
		}

		switch r.URL.Path {
		case "/api/accounts":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]any{
					"accounts": []map[string]any{
						{
							"client_id": "xhs-homefeed-resume-client",
							"platforms": map[string]any{
								"xhs": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"xhs": "mock-xhs"},
						},
					},
				},
			})
		case "/api/cookies/xhs":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "web_session=homefeed-resume-session; a1=homefeed-resume-a1"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/xhs/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_s":           "homefeed-resume-xs",
					"x_t":           "homefeed-resume-xt",
					"x_s_common":    "homefeed-resume-common",
					"x_b3_trace_id": "homefeed-resume-trace",
				},
			})
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{"success": true, "message": "ok", "code": 0},
				},
			})
		case "/api/sns/web/v1/homefeed":
			cursor := fmt.Sprint(payload["cursor_score"])
			noteIndex := int(payloadNumber(payload, "note_index"))
			key := fmt.Sprintf("%s|%d", cursor, noteIndex)
			homefeedCallsByCursorIndex[key]++
			if got := fmt.Sprint(payload["category"]); got != "homefeed_recommend" {
				t.Fatalf("unexpected homefeed category: %s payload=%#v", got, payload)
			}
			switch key {
			case "|0":
				writeJSON(t, w, xhsHomefeedPagePayload("xhs_homefeed_resume_note_1", "token_homefeed_resume_1", "homefeed_resume_cursor_2"))
			case "homefeed_resume_cursor_2|18":
				if !allowSecondPage {
					w.WriteHeader(http.StatusInternalServerError)
					writeJSON(t, w, map[string]any{
						"code":    9999,
						"success": false,
						"msg":     "transient second homefeed failure",
						"data":    map[string]any{},
					})
					return
				}
				writeJSON(t, w, xhsHomefeedPagePayload("xhs_homefeed_resume_note_2", "token_homefeed_resume_2", "homefeed_resume_cursor_done"))
			default:
				t.Fatalf("unexpected homefeed cursor/index: %s payload=%#v", key, payload)
			}
		case "/api/sns/web/v1/feed":
			noteID := fmt.Sprint(payload["source_note_id"])
			feedCallsByNoteID[noteID]++
			switch noteID {
			case "xhs_homefeed_resume_note_1":
				writeJSON(t, w, xhsResumeFeedPayload(noteID, "XHS homefeed resume note 1", "token_homefeed_resume_1"))
			case "xhs_homefeed_resume_note_2":
				writeJSON(t, w, xhsResumeFeedPayload(noteID, "XHS homefeed resume note 2", "token_homefeed_resume_2"))
			default:
				t.Fatalf("unexpected homefeed resume feed note id: %s", noteID)
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
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
platform: "xhs"
keywords:
  - "unused"
crawlerType: "homefeed"
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
  xhsApiUrl: %q
  xhsIndexUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	runCrawler := func(expectErr bool) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		runCmd := exec.CommandContext(ctx, binaryPath)
		runCmd.Dir = tmpDir
		runCmd.Env = append(os.Environ(),
			"RELATION_DB_HOST=127.0.0.1",
			"RELATION_DB_PORT=1",
			"MCPRO_DATA_DIR="+dataDir,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("crawler command timed out\n%s", output)
		}
		if expectErr {
			if !strings.Contains(string(output), "transient second homefeed failure") {
				t.Fatalf("expected first crawler command to log the transient page 2 failure, err=%v\n%s", err, output)
			}
			return output
		}
		if err != nil {
			t.Fatalf("crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler(true)
	if homefeedCallsByCursorIndex["|0"] != 1 || feedCallsByNoteID["xhs_homefeed_resume_note_1"] != 1 {
		t.Fatalf("first run did not process first homefeed page once, homefeed=%#v feed=%#v\n%s", homefeedCallsByCursorIndex, feedCallsByNoteID, firstOutput)
	}
	if homefeedCallsByCursorIndex["homefeed_resume_cursor_2|18"] != 1 {
		t.Fatalf("first run did not stop on second homefeed page once, homefeed=%#v\n%s", homefeedCallsByCursorIndex, firstOutput)
	}

	allowSecondPage = true
	secondOutput := runCrawler(false)
	if homefeedCallsByCursorIndex["|0"] != 1 {
		t.Fatalf("resume should not re-fetch first homefeed page, homefeed=%#v\n%s", homefeedCallsByCursorIndex, secondOutput)
	}
	if homefeedCallsByCursorIndex["homefeed_resume_cursor_2|18"] != 2 || feedCallsByNoteID["xhs_homefeed_resume_note_2"] != 1 {
		t.Fatalf("resume did not continue from saved cursor/index exactly once, homefeed=%#v feed=%#v\n%s", homefeedCallsByCursorIndex, feedCallsByNoteID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "xhs", "json", "homefeed_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read homefeed resume xhs json: %v\n%s", err, secondOutput)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal homefeed resume xhs json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique XHS homefeed rows after resume, got %d: %s", len(rows), body)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["note_id"])] = true
	}
	if !seen["xhs_homefeed_resume_note_1"] || !seen["xhs_homefeed_resume_note_2"] {
		t.Fatalf("homefeed resume output missing expected XHS note IDs: seen=%#v rows=%s", seen, body)
	}
}

func TestCLIXhsSearchSQLiteWithMockServers(t *testing.T) {
	result := runCLIXhsSearchWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	var row struct {
		NoteID        string `gorm:"column:note_id"`
		Title         string `gorm:"column:title"`
		SourceKeyword string `gorm:"column:source_keyword"`
	}
	if err := db.Table("xhs_note").Where("note_id = ?", "note_e2e").First(&row).Error; err != nil {
		t.Fatalf("query sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	if row.Title != "E2E mock note" {
		t.Fatalf("unexpected sqlite title: %#v", row.Title)
	}
	if row.SourceKeyword != "mock-keyword" {
		t.Fatalf("unexpected sqlite source_keyword: %#v", row.SourceKeyword)
	}
}

func TestCLIXhsSearchCSVWithMockServers(t *testing.T) {
	result := runCLIXhsSearchWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "xhs", "1_search_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected csv header and one row, got %d: %#v", len(records), records)
	}
	if csvCell(records, "note_id") != "note_e2e" {
		t.Fatalf("unexpected csv note_id: %#v", csvCell(records, "note_id"))
	}
	if csvCell(records, "title") != "E2E mock note" {
		t.Fatalf("unexpected csv title: %#v", csvCell(records, "title"))
	}
	if csvCell(records, "source_keyword") != "mock-keyword" {
		t.Fatalf("unexpected csv source_keyword: %#v", csvCell(records, "source_keyword"))
	}
}

func TestCLIXhsSearchJSONCommentsWithMockServers(t *testing.T) {
	result := runCLIXhsSearchWithMockServers(t, "json", true)
	if result.commentCalls != 1 {
		t.Fatalf("expected one comment page request, got %d\n%s", result.commentCalls, result.output)
	}
	if result.subCommentCalls != 1 {
		t.Fatalf("expected one sub-comment page request, got %d\n%s", result.subCommentCalls, result.output)
	}

	resultPath := filepath.Join(result.dataDir, "xhs", "json", "search_comments_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read comments result file: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal comments json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected root and sub comment rows, got %d: %s", len(rows), body)
	}

	comments := make(map[string]map[string]any, len(rows))
	for _, row := range rows {
		comments[fmt.Sprint(row["comment_id"])] = row
	}
	if comments["comment_root"]["content"] != "root comment from mock" {
		t.Fatalf("unexpected root comment: %#v", comments["comment_root"])
	}
	if comments["comment_root"]["pictures"] != "https://example.test/comment-root.png" {
		t.Fatalf("unexpected root pictures: %#v", comments["comment_root"]["pictures"])
	}
	if comments["comment_sub"]["parent_comment_id"] != "comment_root" {
		t.Fatalf("unexpected sub parent_comment_id: %#v", comments["comment_sub"]["parent_comment_id"])
	}
	if comments["comment_sub"]["target_comment_id"] != "comment_root" {
		t.Fatalf("unexpected sub target_comment_id: %#v", comments["comment_sub"]["target_comment_id"])
	}
}

func payloadNumber(payload map[string]any, key string) float64 {
	if payload == nil {
		return 0
	}
	switch value := payload[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	default:
		return 0
	}
}

func xhsResumeSearchPayload(noteID, token string) map[string]any {
	return map[string]any{
		"code":    0,
		"success": true,
		"msg":     "ok",
		"data": map[string]any{
			"has_more": true,
			"items": []map[string]any{
				{
					"id":         noteID,
					"xsec_token": token,
					"model_type": "note",
					"note_card": map[string]any{
						"display_title": "resume title",
						"type":          "normal",
						"user": map[string]string{
							"user_id":  "author_resume",
							"nickname": "resume author",
							"avatar":   "https://example.test/resume-avatar.png",
						},
					},
				},
			},
		},
	}
}

func xhsResumeFeedPayload(noteID, title, token string) map[string]any {
	return map[string]any{
		"code":    0,
		"success": true,
		"msg":     "ok",
		"data": map[string]any{
			"current_time": time.Now().UnixMilli(),
			"cursor_score": "",
			"items": []map[string]any{
				{
					"id":         noteID,
					"model_type": "note",
					"note_card": map[string]any{
						"xsec_token":       token,
						"xsec_source":      "pc_feed",
						"note_id":          noteID,
						"title":            title,
						"desc":             "resume desc from local server",
						"type":             "normal",
						"ip_location":      "上海",
						"time":             int64(1710000000000),
						"last_update_time": int64(1710000001000),
						"user": map[string]string{
							"user_id":  "author_resume",
							"nickname": "resume author",
							"avatar":   "https://example.test/resume-avatar.png",
						},
						"interact_info": map[string]any{
							"liked_count":     "8",
							"collected_count": "5",
							"comment_count":   "3",
							"share_count":     "2",
						},
						"image_list": []map[string]any{
							{"url": "https://example.test/" + noteID + ".png"},
						},
						"tag_list": []map[string]string{
							{"id": "tag-resume", "name": "resume", "type": "topic"},
						},
					},
				},
			},
		},
	}
}

func xhsHomefeedPagePayload(noteID, token, cursor string) map[string]any {
	return map[string]any{
		"code":    0,
		"success": true,
		"msg":     "ok",
		"data": map[string]any{
			"cursor_score": cursor,
			"items": []map[string]any{
				{
					"id":         noteID,
					"model_type": "note",
					"xsec_token": token,
					"note_card": map[string]any{
						"note_id":       noteID,
						"display_title": "homefeed resume list note",
						"type":          "normal",
						"user": map[string]string{
							"user_id":  "author_homefeed_resume",
							"nickname": "homefeed resume author",
							"avatar":   "https://example.test/homefeed-resume-avatar.png",
						},
					},
				},
			},
		},
	}
}

func xhsCreatorNotesPagePayload(noteID, token, cursor string, hasMore bool) map[string]any {
	return map[string]any{
		"code":    0,
		"success": true,
		"msg":     "ok",
		"data": map[string]any{
			"cursor":   cursor,
			"has_more": hasMore,
			"notes": []map[string]any{
				{
					"note_id":       noteID,
					"xsec_token":    token,
					"xsec_source":   "pc_user",
					"display_title": "creator resume list note",
					"type":          "normal",
					"cover": map[string]string{
						"url": "https://example.test/" + noteID + "-cover.png",
					},
				},
			},
		},
	}
}

func assertXhsDetailRow(t *testing.T, row map[string]any) {
	t.Helper()
	if row["note_id"] != "xhs_detail_e2e" || row["title"] != "XHS detail note" {
		t.Fatalf("unexpected detail row identity fields: %#v", row)
	}
	if row["note_url"] != "https://www.xiaohongshu.com/explore/xhs_detail_e2e?xsec_token=token_detail_e2e&xsec_source=pc_note_detail" {
		t.Fatalf("unexpected detail note_url: %#v", row["note_url"])
	}
	if row["time"] != "1710000000000" || row["last_update_time"] != "1710000001000" {
		t.Fatalf("unexpected detail time fields: %#v", row)
	}
	if _, ok := row["add_ts"]; ok {
		t.Fatalf("xhs detail json should omit add_ts like Python file output: %#v", row["add_ts"])
	}
}

func assertXhsHomefeedRow(t *testing.T, row map[string]any) {
	t.Helper()
	if row["note_id"] != "xhs_homefeed_note_e2e" || row["title"] != "XHS homefeed note" {
		t.Fatalf("unexpected homefeed row identity fields: %#v", row)
	}
	if row["source_keyword"] != "homefeed" {
		t.Fatalf("unexpected homefeed source_keyword: %#v", row["source_keyword"])
	}
	if row["note_url"] != "https://www.xiaohongshu.com/explore/xhs_homefeed_note_e2e?xsec_token=token_homefeed_note&xsec_source=pc_feed" {
		t.Fatalf("unexpected homefeed note_url: %#v", row["note_url"])
	}
	if _, ok := row["add_ts"]; ok {
		t.Fatalf("xhs homefeed json should omit add_ts like Python file output: %#v", row["add_ts"])
	}
}

func xhsCreatorHTML() string {
	state := map[string]any{
		"user": map[string]any{
			"userPageData": map[string]any{
				"basicInfo": map[string]any{
					"nickname":   "XHS creator mock",
					"images":     "https://example.test/xhs-creator-images.png",
					"desc":       "creator desc from mock",
					"gender":     1,
					"ipLocation": "上海",
				},
				"interactions": []map[string]any{
					{"type": "follows", "name": "关注", "count": "12"},
					{"type": "fans", "name": "粉丝", "count": "345"},
					{"type": "interaction", "name": "获赞与收藏", "count": "678"},
				},
				"tags": []map[string]string{
					{"tagType": "profession", "name": "摄影师"},
					{"tagType": "location", "name": "上海"},
				},
			},
		},
	}
	body, _ := json.Marshal(state)
	return "<html><body><script>window.__INITIAL_STATE__=" + string(body) + "</script></body></html>"
}

func runCLIXhsHomefeedWithMockServers(t *testing.T, saveDataOption string, enableCheckpoint bool) cliXhsSearchResult {
	t.Helper()
	repoRoot := e2e.RepoRoot(t)

	var homefeedCalls, feedCalls int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var payload map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&payload)
		}

		switch r.URL.Path {
		case "/api/accounts":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]any{
					"accounts": []map[string]any{
						{
							"client_id": "xhs-homefeed-client",
							"platforms": map[string]any{
								"xhs": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"xhs": "mock-xhs"},
						},
					},
				},
			})
		case "/api/cookies/xhs":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "web_session=homefeed-session; a1=homefeed-a1"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/xhs/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_s":           "homefeed-xs",
					"x_t":           "homefeed-xt",
					"x_s_common":    "homefeed-common",
					"x_b3_trace_id": "homefeed-trace",
				},
			})
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{"success": true, "message": "ok", "code": 0},
				},
			})
		case "/api/sns/web/v1/homefeed":
			homefeedCalls++
			if got := fmt.Sprint(payload["category"]); got != "homefeed_recommend" {
				t.Fatalf("unexpected homefeed category: %s payload=%#v", got, payload)
			}
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"cursor_score": "homefeed_cursor_1",
					"items": []map[string]any{
						{
							"id":         "xhs_homefeed_note_e2e",
							"model_type": "note",
							"xsec_token": "token_homefeed_note",
							"note_card": map[string]any{
								"note_id":       "xhs_homefeed_note_e2e",
								"display_title": "homefeed list note",
								"type":          "normal",
								"user": map[string]string{
									"user_id":  "author_homefeed",
									"nickname": "homefeed author",
									"avatar":   "https://example.test/homefeed-avatar.png",
								},
							},
						},
					},
				},
			})
		case "/api/sns/web/v1/feed":
			feedCalls++
			noteID := fmt.Sprint(payload["source_note_id"])
			if noteID != "xhs_homefeed_note_e2e" {
				t.Fatalf("unexpected homefeed feed note id: %s", noteID)
			}
			writeJSON(t, w, xhsResumeFeedPayload(noteID, "XHS homefeed note", "token_homefeed_note"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	host, port := serverHostPort(t, mockServer.URL)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	sqlitePath := filepath.Join(tmpDir, "media_crawler.db")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "xhs"
keywords:
  - "unused"
crawlerType: "homefeed"
saveDataOption: %q
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
sqliteDbPath: %q
startPage: 1
crawlerMaxNotesCount: 1
maxConcurrencyNum: 1
requestTimeOut: 2
enableGetComments: false
enableGetSubComments: false
enableCheckpoint: %t
perNoteMaxComments: 2
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  xhsApiUrl: %q
  xhsIndexUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, enableCheckpoint, host, port, mockServer.URL, mockServer.URL)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	runCmd := exec.CommandContext(ctx, binaryPath)
	runCmd.Dir = tmpDir
	runCmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+dataDir,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if homefeedCalls != 1 || feedCalls != 1 {
		t.Fatalf("expected one homefeed and one feed request, got homefeed=%d feed=%d\n%s", homefeedCalls, feedCalls, output)
	}

	return cliXhsSearchResult{
		dataDir:    dataDir,
		sqlitePath: sqlitePath,
		output:     output,
	}
}

func runCLIXhsDetailWithMockServers(t *testing.T, saveDataOption string) cliXhsSearchResult {
	t.Helper()
	repoRoot := e2e.RepoRoot(t)

	var feedCalls int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var payload map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&payload)
		}

		switch r.URL.Path {
		case "/api/accounts":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]any{
					"accounts": []map[string]any{
						{
							"client_id": "xhs-detail-client",
							"platforms": map[string]any{
								"xhs": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"xhs": "mock-xhs"},
						},
					},
				},
			})
		case "/api/cookies/xhs":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "web_session=detail-session; a1=detail-a1"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/xhs/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_s":           "detail-xs",
					"x_t":           "detail-xt",
					"x_s_common":    "detail-common",
					"x_b3_trace_id": "detail-trace",
				},
			})
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{"success": true, "message": "ok", "code": 0},
				},
			})
		case "/api/sns/web/v1/feed":
			feedCalls++
			noteID := fmt.Sprint(payload["source_note_id"])
			if noteID != "xhs_detail_e2e" {
				t.Fatalf("unexpected detail feed note id: %s", noteID)
			}
			writeJSON(t, w, xhsResumeFeedPayload(noteID, "XHS detail note", "token_detail_e2e"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	host, port := serverHostPort(t, mockServer.URL)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	sqlitePath := filepath.Join(tmpDir, "media_crawler.db")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "xhs"
keywords:
  - "unused"
crawlerType: "detail"
saveDataOption: %q
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
sqliteDbPath: %q
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
specifiedNoteURLs:
  - "https://www.xiaohongshu.com/explore/xhs_detail_e2e?xsec_token=token_detail_e2e&xsec_source=pc_note_detail"
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  xhsApiUrl: %q
  xhsIndexUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, host, port, mockServer.URL, mockServer.URL)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	runCmd := exec.CommandContext(ctx, binaryPath)
	runCmd.Dir = tmpDir
	runCmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+dataDir,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if feedCalls != 1 {
		t.Fatalf("expected one detail feed request, got %d\n%s", feedCalls, output)
	}

	return cliXhsSearchResult{
		dataDir:    dataDir,
		sqlitePath: sqlitePath,
		output:     output,
	}
}

func runCLIXhsCreatorWithMockServers(t *testing.T, saveDataOption string, enableCheckpointOption ...bool) cliXhsSearchResult {
	t.Helper()
	enableCheckpoint := true
	if len(enableCheckpointOption) > 0 {
		enableCheckpoint = enableCheckpointOption[0]
	}
	repoRoot := e2e.RepoRoot(t)

	var creatorProfileCalls, creatorNotesCalls, feedCalls int
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var payload map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&payload)
		}

		switch r.URL.Path {
		case "/api/accounts":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]any{
					"accounts": []map[string]any{
						{
							"client_id": "xhs-creator-client",
							"platforms": map[string]any{
								"xhs": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"xhs": "mock-xhs"},
						},
					},
				},
			})
		case "/api/cookies/xhs":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "web_session=creator-session; a1=creator-a1"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/xhs/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_s":           "creator-xs",
					"x_t":           "creator-xt",
					"x_s_common":    "creator-common",
					"x_b3_trace_id": "creator-trace",
				},
			})
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{"success": true, "message": "ok", "code": 0},
				},
			})
		case "/user/profile/xhs_creator_e2e":
			creatorProfileCalls++
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(xhsCreatorHTML()))
		case "/api/sns/web/v1/user_posted":
			creatorNotesCalls++
			if got := r.URL.Query().Get("user_id"); got != "xhs_creator_e2e" {
				t.Fatalf("unexpected creator user_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"cursor":   "",
					"has_more": false,
					"notes": []map[string]any{
						{
							"note_id":       "xhs_creator_note_e2e",
							"xsec_token":    "token_creator_note",
							"xsec_source":   "pc_user",
							"display_title": "creator list note",
							"type":          "normal",
							"cover": map[string]string{
								"url": "https://example.test/xhs-creator-cover.png",
							},
						},
					},
				},
			})
		case "/api/sns/web/v1/feed":
			feedCalls++
			noteID := fmt.Sprint(payload["source_note_id"])
			if noteID != "xhs_creator_note_e2e" {
				t.Fatalf("unexpected creator feed note id: %s", noteID)
			}
			writeJSON(t, w, xhsResumeFeedPayload(noteID, "XHS creator note", "token_creator_note"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	host, port := serverHostPort(t, mockServer.URL)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	sqlitePath := filepath.Join(tmpDir, "media_crawler.db")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "xhs"
keywords:
  - "unused"
crawlerType: "creator"
saveDataOption: %q
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
sqliteDbPath: %q
startPage: 1
crawlerMaxNotesCount: 1
maxConcurrencyNum: 1
requestTimeOut: 2
enableGetComments: false
enableGetSubComments: false
enableCheckpoint: %t
perNoteMaxComments: 2
crawlerTimeSleep: 0
debugRequest: false
specifiedCreatorURLs:
  - "https://www.xiaohongshu.com/user/profile/xhs_creator_e2e?xsec_token=token_creator_profile&xsec_source=pc_user"
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  xhsApiUrl: %q
  xhsIndexUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, enableCheckpoint, host, port, mockServer.URL, mockServer.URL)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	runCmd := exec.CommandContext(ctx, binaryPath)
	runCmd.Dir = tmpDir
	runCmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+dataDir,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if creatorProfileCalls != 1 || creatorNotesCalls != 1 || feedCalls != 1 {
		t.Fatalf("unexpected creator request counts profile=%d notes=%d feed=%d\n%s", creatorProfileCalls, creatorNotesCalls, feedCalls, output)
	}

	return cliXhsSearchResult{
		dataDir:    dataDir,
		sqlitePath: sqlitePath,
		output:     output,
	}
}

type cliXhsSearchResult struct {
	dataDir         string
	sqlitePath      string
	output          []byte
	commentCalls    int
	subCommentCalls int
}

func runCLIXhsSearchWithMockServers(t *testing.T, saveDataOption string, enableComments ...bool) cliXhsSearchResult {
	t.Helper()
	commentsEnabled := len(enableComments) > 0 && enableComments[0]
	repoRoot := e2e.RepoRoot(t)

	var searchCalls, feedCalls, commentCalls, subCommentCalls int
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
							"client_id": "xhs-mock-client",
							"platforms": map[string]any{
								"xhs": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"xhs": "mock-xhs"},
						},
					},
				},
			})
		case "/api/cookies/xhs":
			if got := r.URL.Query().Get("client_id"); got != "xhs-mock-client" {
				t.Fatalf("unexpected client_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "web_session=mock-session; a1=mock-a1"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/xhs/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_s":           "mock-xs",
					"x_t":           "mock-xt",
					"x_s_common":    "mock-common",
					"x_b3_trace_id": "mock-trace",
				},
			})
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{
						"success": true,
						"message": "ok",
						"code":    0,
					},
				},
			})
		case "/api/sns/web/v1/search/notes":
			searchCalls++
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"has_more": true,
					"items": []map[string]any{
						{
							"id":         "note_e2e",
							"xsec_token": "token_e2e",
							"model_type": "note",
							"note_card": map[string]any{
								"display_title": "mock title",
								"type":          "normal",
								"user": map[string]string{
									"user_id":  "author_e2e",
									"nickname": "mock author",
									"avatar":   "https://example.test/avatar.png",
								},
							},
						},
					},
				},
			})
		case "/api/sns/web/v1/feed":
			feedCalls++
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"current_time": time.Now().UnixMilli(),
					"cursor_score": "",
					"items": []map[string]any{
						{
							"id":         "note_e2e",
							"model_type": "note",
							"note_card": map[string]any{
								"xsec_token":       "token_e2e",
								"xsec_source":      "pc_feed",
								"note_id":          "note_e2e",
								"title":            "E2E mock note",
								"desc":             "mock desc from local server",
								"type":             "normal",
								"ip_location":      "上海",
								"time":             int64(1710000000000),
								"last_update_time": int64(1710000001000),
								"user": map[string]string{
									"user_id":  "author_e2e",
									"nickname": "mock author",
									"avatar":   "https://example.test/avatar.png",
								},
								"interact_info": map[string]any{
									"liked_count":     "8",
									"collected_count": "5",
									"comment_count":   "3",
									"share_count":     "2",
								},
								"image_list": []map[string]any{
									{"url": "https://example.test/image.png"},
								},
								"tag_list": []map[string]string{
									{"id": "tag-1", "name": "mocktag", "type": "topic"},
								},
							},
						},
					},
				},
			})
		case "/api/sns/web/v2/comment/page":
			commentCalls++
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"cursor":   "",
					"has_more": false,
					"comments": []map[string]any{
						{
							"id":                "comment_root",
							"note_id":           "note_e2e",
							"content":           "root comment from mock",
							"create_time":       int64(1710000002000),
							"like_count":        "4",
							"sub_comment_count": "1",
							"pictures": []map[string]string{
								{"url_default": "https://example.test/comment-root.png"},
							},
							"user_info": map[string]string{
								"user_id":  "commenter_root",
								"nickname": "root commenter",
								"image":    "https://example.test/commenter-root.png",
							},
							"ip_location": "北京",
						},
					},
				},
			})
		case "/api/sns/web/v2/comment/sub/page":
			subCommentCalls++
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"cursor":   "",
					"has_more": false,
					"comments": []map[string]any{
						{
							"id":                "comment_sub",
							"note_id":           "note_e2e",
							"content":           "sub comment from mock",
							"create_time":       int64(1710000003000),
							"like_count":        "2",
							"sub_comment_count": "0",
							"pictures":          []map[string]string{},
							"target_comment": map[string]string{
								"id": "comment_root",
							},
							"user_info": map[string]string{
								"user_id":  "commenter_sub",
								"nickname": "sub commenter",
								"image":    "https://example.test/commenter-sub.png",
							},
							"ip_location": "广东",
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	host, port := serverHostPort(t, mockServer.URL)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	sqlitePath := filepath.Join(tmpDir, "media_crawler.db")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "xhs"
keywords:
  - "mock-keyword"
crawlerType: "search"
saveDataOption: %q
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
sqliteDbPath: %q
startPage: 1
crawlerMaxNotesCount: 1
maxConcurrencyNum: 1
requestTimeOut: 2
enableGetComments: %t
enableGetSubComments: %t
enableCheckpoint: false
perNoteMaxComments: 2
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  xhsApiUrl: %q
  xhsIndexUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, commentsEnabled, commentsEnabled, host, port, mockServer.URL, mockServer.URL)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	runCmd := exec.CommandContext(ctx, binaryPath)
	runCmd.Dir = tmpDir
	runCmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+dataDir,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if searchCalls != 1 {
		t.Fatalf("expected one search request, got %d\n%s", searchCalls, output)
	}
	if feedCalls != 1 {
		t.Fatalf("expected one feed request, got %d\n%s", feedCalls, output)
	}

	return cliXhsSearchResult{
		dataDir:         dataDir,
		sqlitePath:      sqlitePath,
		output:          output,
		commentCalls:    commentCalls,
		subCommentCalls: subCommentCalls,
	}
}

func runCLIXhsHotSearchPlaceholderWithMockServers(t *testing.T) cliXhsSearchResult {
	t.Helper()
	repoRoot := e2e.RepoRoot(t)

	var searchCalls, feedCalls int
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
							"client_id": "xhs-hot-client",
							"platforms": map[string]any{
								"xhs": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"xhs": "mock-xhs"},
						},
					},
				},
			})
		case "/api/cookies/xhs":
			if got := r.URL.Query().Get("client_id"); got != "xhs-hot-client" {
				t.Fatalf("unexpected client_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "web_session=mock-session; a1=mock-a1"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/xhs/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_s":           "mock-xs",
					"x_t":           "mock-xt",
					"x_s_common":    "mock-common",
					"x_b3_trace_id": "mock-trace",
				},
			})
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{
						"success": true,
						"message": "ok",
						"code":    0,
					},
				},
			})
		case "/api/sns/web/v1/search/notes":
			searchCalls++
			t.Fatalf("xhs hot_search placeholder should not call search endpoint")
		case "/api/sns/web/v1/feed":
			feedCalls++
			t.Fatalf("xhs hot_search placeholder should not call feed endpoint")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
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
platform: "xhs"
keywords:
  - "unused"
crawlerType: "hot_search"
saveDataOption: "json"
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
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
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  xhsApiUrl: %q
  xhsIndexUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(strings.TrimSpace(configYAML)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	runCmd := exec.CommandContext(ctx, binaryPath)
	runCmd.Dir = tmpDir
	runCmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+dataDir,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if searchCalls != 0 || feedCalls != 0 {
		t.Fatalf("xhs hot_search placeholder should not call content endpoints, search=%d feed=%d\n%s", searchCalls, feedCalls, output)
	}
	return cliXhsSearchResult{
		dataDir: dataDir,
		output:  output,
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatal(err)
	}
}

func serverHostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	host, portStr, err := net.SplitHostPort(parsedURL.Host)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	return host, port
}

func csvCell(records [][]string, column string) string {
	for idx, header := range records[0] {
		if strings.TrimPrefix(header, "\ufeff") == column {
			return records[1][idx]
		}
	}
	return ""
}
