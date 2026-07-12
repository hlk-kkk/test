// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/tieba_e2e_test.go
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

func TestCLITiebaSearchJSONWithMockServers(t *testing.T) {
	result := runCLITiebaSearchWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "tieba", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, result.output)
	if len(rows) != 1 {
		t.Fatalf("expected one tieba json row, got %d: %#v", len(rows), rows)
	}
	assertTiebaSearchMapRow(t, rows[0], result.tiebaURL)
	if _, ok := rows[0]["add_ts"]; ok {
		t.Fatalf("tieba json should match Python and omit add_ts: %#v", rows[0])
	}
	if _, ok := rows[0]["last_modify_ts"]; !ok {
		t.Fatalf("expected tieba json last_modify_ts: %#v", rows[0])
	}
}

func TestCLITiebaSearchCSVWithMockServers(t *testing.T) {
	result := runCLITiebaSearchWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "tieba", "1_search_contents_"+time.Now().Format("2006-01-02")+".csv")
	records := readCSVRecords(t, resultPath, result.output)
	if len(records) != 2 {
		t.Fatalf("expected tieba csv header and one row, got %d: %#v", len(records), records)
	}
	assertTiebaSearchCSVRow(t, records, result.tiebaURL)
	if csvCell(records, "add_ts") != "" {
		t.Fatalf("tieba csv should not include add_ts value: %#v", csvCell(records, "add_ts"))
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected tieba csv last_modify_ts")
	}
}

func TestCLITiebaSearchSQLiteWithMockServers(t *testing.T) {
	result := runCLITiebaSearchWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tieba sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []tiebaSQLiteNote
	if err := db.Table("tieba_note").Find(&rows).Error; err != nil {
		t.Fatalf("query tieba sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one tieba sqlite row, got %d: %#v", len(rows), rows)
	}
	assertTiebaSQLiteNote(t, rows[0], result.tiebaURL)
}

func TestCLITiebaSearchCheckpointResumeSkipsCompletedPages(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	searchCallsByPage := make(map[string]int)
	detailCallsByNoteID := make(map[string]int)
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
							"client_id": "tieba-search-resume-client",
							"platforms": map[string]any{
								"tieba": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"tieba": "mock-tieba"},
						},
					},
				},
			})
		case "/api/cookies/tieba":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "BDUSS=mock-tieba-resume-bduss; STOKEN=mock-tieba-resume-stoken"},
			})
		case "/c/s/pc/sync":
			writeJSON(t, w, map[string]any{
				"error_code": 0,
				"data": map[string]any{
					"anti": map[string]string{"tbs": "mock-tieba-resume-tbs"},
					"user": map[string]bool{"is_login": true},
				},
			})
		case "/mo/q/search/multsearch":
			page := r.URL.Query().Get("pn")
			searchCallsByPage[page]++
			switch page {
			case "1":
				writeJSON(t, w, tiebaSearchResumePayload("tb_search_resume_note_1"))
			case "2":
				if !allowSecondPage {
					http.Error(w, "transient second search page failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, tiebaSearchResumePayload("tb_search_resume_note_2"))
			default:
				writeJSON(t, w, map[string]any{"error_code": 0, "data": map[string]any{"card_list": []map[string]any{}}})
			}
		case "/c/f/pb/page_pc":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse tieba detail form: %v", err)
			}
			noteID := r.Form.Get("kz")
			detailCallsByNoteID[noteID]++
			switch noteID {
			case "tb_search_resume_note_1":
				writeJSON(t, w, tiebaDetailResumePayload(noteID, "Tieba search resume note 1", "Tieba search resume content 1"))
			case "tb_search_resume_note_2":
				writeJSON(t, w, tiebaDetailResumePayload(noteID, "Tieba search resume note 2", "Tieba search resume content 2"))
			default:
				t.Fatalf("unexpected tieba resume detail note id: %s", noteID)
			}
		default:
			t.Fatalf("unexpected tieba search resume request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	host, port := serverHostPort(t, mockServer.URL)
	configYAML := fmt.Sprintf(`
platform: "tieba"
keywords:
  - "resume-tieba"
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
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  tiebaUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL)
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
			"MCPRO_TIEBA_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("tieba search resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("tieba search resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if searchCallsByPage["1"] != 1 || searchCallsByPage["2"] != 1 {
		t.Fatalf("first tieba search run should fetch page 1 and fail page 2 once, calls=%#v\n%s", searchCallsByPage, firstOutput)
	}
	if detailCallsByNoteID["tb_search_resume_note_1"] != 1 || detailCallsByNoteID["tb_search_resume_note_2"] != 0 {
		t.Fatalf("first tieba search run should only detail the first page note, detail=%#v\n%s", detailCallsByNoteID, firstOutput)
	}

	allowSecondPage = true
	secondOutput := runCrawler()
	if searchCallsByPage["1"] != 1 {
		t.Fatalf("resume should skip completed first tieba search page, calls=%#v\n%s", searchCallsByPage, secondOutput)
	}
	if searchCallsByPage["2"] != 2 {
		t.Fatalf("resume should retry second tieba search page exactly once, calls=%#v\n%s", searchCallsByPage, secondOutput)
	}
	if detailCallsByNoteID["tb_search_resume_note_2"] != 1 {
		t.Fatalf("resume should detail the second page note after retry, detail=%#v\n%s", detailCallsByNoteID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "tieba", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Tieba search rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["note_id"])] = true
	}
	if !seen["tb_search_resume_note_1"] || !seen["tb_search_resume_note_2"] {
		t.Fatalf("tieba search resume output missing expected note IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLITiebaDetailCheckpointResumeSkipsCompletedIDs(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	detailCallsByNoteID := make(map[string]int)
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
							"client_id": "tieba-detail-resume-client",
							"platforms": map[string]any{
								"tieba": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"tieba": "mock-tieba"},
						},
					},
				},
			})
		case "/api/cookies/tieba":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "BDUSS=mock-tieba-detail-resume-bduss; STOKEN=mock-tieba-detail-resume-stoken"},
			})
		case "/c/s/pc/sync":
			writeJSON(t, w, map[string]any{
				"error_code": 0,
				"data": map[string]any{
					"anti": map[string]string{"tbs": "mock-tieba-detail-resume-tbs"},
					"user": map[string]bool{"is_login": true},
				},
			})
		case "/c/f/pb/page_pc":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse tieba detail form: %v", err)
			}
			noteID := r.Form.Get("kz")
			detailCallsByNoteID[noteID]++
			switch noteID {
			case "tb_detail_resume_note_1":
				writeJSON(t, w, tiebaDetailResumePayload(noteID, "Tieba detail resume note 1", "Tieba detail resume content 1"))
			case "tb_detail_resume_note_2":
				if !allowSecondDetail {
					http.Error(w, "transient second detail failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, tiebaDetailResumePayload(noteID, "Tieba detail resume note 2", "Tieba detail resume content 2"))
			default:
				t.Fatalf("unexpected tieba detail resume note id: %s", noteID)
			}
		default:
			t.Fatalf("unexpected tieba detail resume request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	host, port := serverHostPort(t, mockServer.URL)
	configYAML := fmt.Sprintf(`
platform: "tieba"
keywords:
  - "mock-tieba"
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
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  tiebaUrl: %q
specifiedNoteIds:
  - "tb_detail_resume_note_1"
  - "tb_detail_resume_note_2"
`, mockServer.URL, dataDir, host, port, mockServer.URL)
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
			"MCPRO_TIEBA_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("tieba detail resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("tieba detail resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if detailCallsByNoteID["tb_detail_resume_note_1"] != 1 || detailCallsByNoteID["tb_detail_resume_note_2"] != 1 {
		t.Fatalf("first tieba detail run should fetch both IDs and fail second once, calls=%#v\n%s", detailCallsByNoteID, firstOutput)
	}

	allowSecondDetail = true
	secondOutput := runCrawler()
	if detailCallsByNoteID["tb_detail_resume_note_1"] != 1 {
		t.Fatalf("resume should skip completed first tieba detail ID, calls=%#v\n%s", detailCallsByNoteID, secondOutput)
	}
	if detailCallsByNoteID["tb_detail_resume_note_2"] != 2 {
		t.Fatalf("resume should retry second tieba detail ID exactly once, calls=%#v\n%s", detailCallsByNoteID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "tieba", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Tieba detail rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["note_id"])] = true
	}
	if !seen["tb_detail_resume_note_1"] || !seen["tb_detail_resume_note_2"] {
		t.Fatalf("tieba detail resume output missing expected note IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLITiebaDetailJSONWithMockServers(t *testing.T) {
	result := runCLITiebaWithMockServers(t, "detail", "json", []string{"tb_note_e2e"}, false, false)

	resultPath := filepath.Join(result.dataDir, "tieba", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, result.output)
	if len(rows) != 1 {
		t.Fatalf("expected one tieba detail json row, got %d: %#v", len(rows), rows)
	}
	assertTiebaDetailMapRow(t, rows[0], result.tiebaURL)
	if _, ok := rows[0]["add_ts"]; ok {
		t.Fatalf("tieba detail json should match Python and omit add_ts: %#v", rows[0])
	}
	if _, ok := rows[0]["last_modify_ts"]; !ok {
		t.Fatalf("expected tieba detail json last_modify_ts: %#v", rows[0])
	}
}

func TestCLITiebaDetailCSVWithMockServers(t *testing.T) {
	result := runCLITiebaWithMockServers(t, "detail", "csv", []string{"tb_note_e2e"}, false, false)

	resultPath := filepath.Join(result.dataDir, "tieba", "1_detail_contents_"+time.Now().Format("2006-01-02")+".csv")
	records := readCSVRecords(t, resultPath, result.output)
	if len(records) != 2 {
		t.Fatalf("expected tieba detail csv header and one row, got %d: %#v", len(records), records)
	}
	assertTiebaDetailCSVRow(t, records, result.tiebaURL)
	if csvCell(records, "add_ts") != "" {
		t.Fatalf("tieba detail csv should not include add_ts value: %#v", csvCell(records, "add_ts"))
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected tieba detail csv last_modify_ts")
	}
}

func TestCLITiebaDetailSQLiteWithMockServers(t *testing.T) {
	result := runCLITiebaWithMockServers(t, "detail", "sqlite", []string{"tb_note_e2e"}, false, false)

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tieba detail sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []tiebaSQLiteNote
	if err := db.Table("tieba_note").Find(&rows).Error; err != nil {
		t.Fatalf("query tieba detail sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one tieba detail sqlite row, got %d: %#v", len(rows), rows)
	}
	assertTiebaSQLiteDetailNote(t, rows[0], result.tiebaURL)

	var commentCount int64
	if err := db.Table("tieba_comment").Count(&commentCount).Error; err != nil {
		t.Fatalf("expected tieba_comment table to match Python sqlite schema: %v\ncommand output:\n%s", err, result.output)
	}
	if commentCount != 0 {
		t.Fatalf("expected no tieba sqlite comments when comments disabled, got %d", commentCount)
	}
}

func TestCLITiebaDetailCommentsJSONWithMockServers(t *testing.T) {
	result := runCLITiebaWithMockServers(t, "detail", "json", []string{"tb_note_e2e"}, true, true)

	contentsPath := filepath.Join(result.dataDir, "tieba", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	contentRows := readJSONRows(t, contentsPath, result.output)
	if len(contentRows) != 1 {
		t.Fatalf("expected one tieba detail json row with comments enabled, got %d: %#v", len(contentRows), contentRows)
	}
	assertTiebaDetailMapRow(t, contentRows[0], result.tiebaURL)

	commentsPath := filepath.Join(result.dataDir, "tieba", "json", "detail_comments_"+time.Now().Format("2006-01-02")+".json")
	commentRows := readJSONRows(t, commentsPath, result.output)
	if len(commentRows) != 2 {
		t.Fatalf("expected root and sub tieba comments, got %d: %#v", len(commentRows), commentRows)
	}
	assertTiebaCommentMapRow(t, commentRows[0], result.tiebaURL, "tb_comment_root", "", "root tieba comment", result.tiebaURL+"/home/main?id=tb-commenter-portrait", "mock tieba commenter", "https://example.test/tieba-comment-avatar.png", "2024-03-10 00:23:20", "上海", "1")
	assertTiebaCommentMapRow(t, commentRows[1], result.tiebaURL, "tb_comment_sub", "tb_comment_root", "sub tieba comment", "/home/main?id=tb-sub-commenter-portrait", "mock tieba sub commenter", "https://example.test/tieba-sub-comment-avatar.png", "2024-03-10 00:24", "", "0")
}

func TestCLITiebaDetailCommentsCSVWithMockServers(t *testing.T) {
	result := runCLITiebaWithMockServers(t, "detail", "csv", []string{"tb_note_e2e"}, true, true)

	contentsPath := filepath.Join(result.dataDir, "tieba", "1_detail_contents_"+time.Now().Format("2006-01-02")+".csv")
	contentRecords := readCSVRecords(t, contentsPath, result.output)
	if len(contentRecords) != 2 {
		t.Fatalf("expected tieba detail csv header and one row with comments enabled, got %d: %#v", len(contentRecords), contentRecords)
	}
	assertTiebaDetailCSVRow(t, contentRecords, result.tiebaURL)

	commentsPath := filepath.Join(result.dataDir, "tieba", "1_detail_comments_"+time.Now().Format("2006-01-02")+".csv")
	commentRecords := readCSVRecords(t, commentsPath, result.output)
	if len(commentRecords) != 3 {
		t.Fatalf("expected tieba comments csv header plus root/sub comments, got %d: %#v", len(commentRecords), commentRecords)
	}
	assertTiebaCommentCSVRow(t, commentRecords, 1, result.tiebaURL, "tb_comment_root", "", "root tieba comment", result.tiebaURL+"/home/main?id=tb-commenter-portrait", "mock tieba commenter", "https://example.test/tieba-comment-avatar.png", "2024-03-10 00:23:20", "上海", "1")
	assertTiebaCommentCSVRow(t, commentRecords, 2, result.tiebaURL, "tb_comment_sub", "tb_comment_root", "sub tieba comment", "/home/main?id=tb-sub-commenter-portrait", "mock tieba sub commenter", "https://example.test/tieba-sub-comment-avatar.png", "2024-03-10 00:24", "", "0")
	if csvCellAt(commentRecords, 1, "add_ts") != "" || csvCellAt(commentRecords, 2, "add_ts") != "" {
		t.Fatalf("tieba comment csv should match Python and omit add_ts values: %#v", commentRecords)
	}
	if csvCellAt(commentRecords, 1, "last_modify_ts") == "" || csvCellAt(commentRecords, 2, "last_modify_ts") == "" {
		t.Fatalf("expected tieba comment csv last_modify_ts values: %#v", commentRecords)
	}
}

func TestCLITiebaDetailCommentsSQLiteWithMockServers(t *testing.T) {
	result := runCLITiebaWithMockServers(t, "detail", "sqlite", []string{"tb_note_e2e"}, true, true)

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tieba comments sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var notes []tiebaSQLiteNote
	if err := db.Table("tieba_note").Find(&notes).Error; err != nil {
		t.Fatalf("query tieba sqlite notes with comments: %v\ncommand output:\n%s", err, result.output)
	}
	if len(notes) != 1 {
		t.Fatalf("expected one tieba sqlite note with comments, got %d: %#v", len(notes), notes)
	}
	assertTiebaSQLiteDetailNote(t, notes[0], result.tiebaURL)

	var comments []tiebaSQLiteComment
	if err := db.Table("tieba_comment").Order("comment_id").Find(&comments).Error; err != nil {
		t.Fatalf("query tieba sqlite comments: %v\ncommand output:\n%s", err, result.output)
	}
	if len(comments) != 2 {
		t.Fatalf("expected root and sub tieba sqlite comments, got %d: %#v", len(comments), comments)
	}
	assertTiebaSQLiteComment(t, comments[0], result.tiebaURL, "tb_comment_root", "", "root tieba comment", result.tiebaURL+"/home/main?id=tb-commenter-portrait", "mock tieba commenter", "https://example.test/tieba-comment-avatar.png", "2024-03-10 00:23:20", "上海", 1)
	assertTiebaSQLiteComment(t, comments[1], result.tiebaURL, "tb_comment_sub", "tb_comment_root", "sub tieba comment", "/home/main?id=tb-sub-commenter-portrait", "mock tieba sub commenter", "https://example.test/tieba-sub-comment-avatar.png", "2024-03-10 00:24", "", 0)
}

func TestCLITiebaCreatorJSONWithMockServers(t *testing.T) {
	result := runCLITiebaWithMockServers(t, "creator", "json", []string{"tb-creator-portrait"}, false, false)

	creatorPath := filepath.Join(result.dataDir, "tieba", "json", "creator_creator_"+time.Now().Format("2006-01-02")+".json")
	creatorRows := readJSONRows(t, creatorPath, result.output)
	if len(creatorRows) != 1 {
		t.Fatalf("expected one tieba creator json row, got %d: %#v", len(creatorRows), creatorRows)
	}
	assertTiebaCreatorMapRow(t, creatorRows[0])

	contentsPath := filepath.Join(result.dataDir, "tieba", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	contentRows := readJSONRows(t, contentsPath, result.output)
	if len(contentRows) != 1 {
		t.Fatalf("expected one tieba creator content json row, got %d: %#v", len(contentRows), contentRows)
	}
	assertTiebaDetailMapRow(t, contentRows[0], result.tiebaURL)
}

func TestCLITiebaCreatorCSVWithMockServers(t *testing.T) {
	result := runCLITiebaWithMockServers(t, "creator", "csv", []string{"tb-creator-portrait"}, false, false)

	creatorPath := filepath.Join(result.dataDir, "tieba", "1_creator_creator_"+time.Now().Format("2006-01-02")+".csv")
	creatorRecords := readCSVRecords(t, creatorPath, result.output)
	if len(creatorRecords) != 2 {
		t.Fatalf("expected tieba creator csv header and one row, got %d: %#v", len(creatorRecords), creatorRecords)
	}
	assertTiebaCreatorCSVRow(t, creatorRecords)

	contentsPath := filepath.Join(result.dataDir, "tieba", "1_creator_contents_"+time.Now().Format("2006-01-02")+".csv")
	contentRecords := readCSVRecords(t, contentsPath, result.output)
	if len(contentRecords) != 2 {
		t.Fatalf("expected tieba creator content csv header and one row, got %d: %#v", len(contentRecords), contentRecords)
	}
	assertTiebaDetailCSVRow(t, contentRecords, result.tiebaURL)
}

func TestCLITiebaCreatorSQLiteWithMockServers(t *testing.T) {
	result := runCLITiebaWithMockServers(t, "creator", "sqlite", []string{"tb-creator-portrait"}, false, false)

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open tieba creator sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var creators []tiebaSQLiteCreator
	if err := db.Table("tieba_creator").Find(&creators).Error; err != nil {
		t.Fatalf("query tieba sqlite creators: %v\ncommand output:\n%s", err, result.output)
	}
	if len(creators) != 1 {
		t.Fatalf("expected one tieba sqlite creator, got %d: %#v", len(creators), creators)
	}
	assertTiebaSQLiteCreator(t, creators[0])

	var notes []tiebaSQLiteNote
	if err := db.Table("tieba_note").Find(&notes).Error; err != nil {
		t.Fatalf("query tieba sqlite creator notes: %v\ncommand output:\n%s", err, result.output)
	}
	if len(notes) != 1 {
		t.Fatalf("expected one tieba sqlite creator note, got %d: %#v", len(notes), notes)
	}
	assertTiebaSQLiteDetailNote(t, notes[0], result.tiebaURL)
}

func TestCLITiebaCreatorCheckpointResumeByCreatorAndPage(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	infoCalls := 0
	threadCallsByPage := make(map[string]int)
	detailCallsByNoteID := make(map[string]int)
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
							"client_id": "tieba-creator-resume-client",
							"platforms": map[string]any{
								"tieba": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"tieba": "mock-tieba"},
						},
					},
				},
			})
		case "/api/cookies/tieba":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "BDUSS=mock-tieba-creator-resume-bduss; STOKEN=mock-tieba-creator-resume-stoken"},
			})
		case "/c/s/pc/sync":
			writeJSON(t, w, map[string]any{
				"error_code": 0,
				"data": map[string]any{
					"anti": map[string]string{"tbs": "mock-tieba-creator-resume-tbs"},
					"user": map[string]bool{"is_login": true},
				},
			})
		case "/c/u/pc/homeSidebarRight":
			if r.URL.Query().Get("portrait") != "tb-creator-resume-portrait" {
				t.Fatalf("unexpected tieba creator portrait: %s", r.URL.Query().Get("portrait"))
			}
			infoCalls++
			writeJSON(t, w, tiebaCreatorResumePayload())
		case "/c/u/feed/myThread":
			if r.URL.Query().Get("portrait") != "tb-creator-resume-portrait" {
				t.Fatalf("unexpected tieba creator thread portrait: %s", r.URL.Query().Get("portrait"))
			}
			page := r.URL.Query().Get("pn")
			threadCallsByPage[page]++
			switch page {
			case "1":
				writeJSON(t, w, tiebaCreatorThreadsResumePayload("tb_creator_resume_note_1", true))
			case "2":
				if !allowSecondPage {
					http.Error(w, "transient second creator page failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, tiebaCreatorThreadsResumePayload("tb_creator_resume_note_2", false))
			default:
				writeJSON(t, w, map[string]any{"error_code": 0, "data": map[string]any{"has_more": 0, "list": []map[string]any{}}})
			}
		case "/c/f/pb/page_pc":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse tieba creator detail form: %v", err)
			}
			noteID := r.Form.Get("kz")
			detailCallsByNoteID[noteID]++
			switch noteID {
			case "tb_creator_resume_note_1":
				writeJSON(t, w, tiebaDetailResumePayload(noteID, "Tieba creator resume note 1", "Tieba creator resume content 1"))
			case "tb_creator_resume_note_2":
				writeJSON(t, w, tiebaDetailResumePayload(noteID, "Tieba creator resume note 2", "Tieba creator resume content 2"))
			default:
				t.Fatalf("unexpected tieba creator resume detail note id: %s", noteID)
			}
		default:
			t.Fatalf("unexpected tieba creator resume request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	host, port := serverHostPort(t, mockServer.URL)
	configYAML := fmt.Sprintf(`
platform: "tieba"
keywords:
  - "mock-tieba"
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
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  tiebaUrl: %q
specifiedCreatorIds:
  - "tb-creator-resume-portrait"
`, mockServer.URL, dataDir, host, port, mockServer.URL)
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
			"MCPRO_TIEBA_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("tieba creator resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("tieba creator resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if infoCalls != 1 || threadCallsByPage["1"] != 1 || threadCallsByPage["2"] != 1 {
		t.Fatalf("first tieba creator run should fetch profile, page 1, and fail page 2 once, infoCalls=%d threadCalls=%#v\n%s", infoCalls, threadCallsByPage, firstOutput)
	}
	if detailCallsByNoteID["tb_creator_resume_note_1"] != 1 || detailCallsByNoteID["tb_creator_resume_note_2"] != 0 {
		t.Fatalf("first tieba creator run should only detail page 1 note, detailCalls=%#v\n%s", detailCallsByNoteID, firstOutput)
	}

	allowSecondPage = true
	secondOutput := runCrawler()
	if infoCalls != 2 {
		t.Fatalf("resume should refresh creator profile to match Python behavior, infoCalls=%d\n%s", infoCalls, secondOutput)
	}
	if threadCallsByPage["1"] != 1 {
		t.Fatalf("resume should skip completed first creator page, calls=%#v\n%s", threadCallsByPage, secondOutput)
	}
	if threadCallsByPage["2"] != 2 {
		t.Fatalf("resume should retry second creator page exactly once, calls=%#v\n%s", threadCallsByPage, secondOutput)
	}
	if detailCallsByNoteID["tb_creator_resume_note_2"] != 1 {
		t.Fatalf("resume should detail page 2 note after retry, detailCalls=%#v\n%s", detailCallsByNoteID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "tieba", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Tieba creator rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["note_id"])] = true
	}
	if !seen["tb_creator_resume_note_1"] || !seen["tb_creator_resume_note_2"] {
		t.Fatalf("tieba creator resume output missing expected note IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLITiebaHotSearchPlaceholderWithMockServers(t *testing.T) {
	result := runCLITiebaWithMockServers(t, "hot_search", "json", nil, false, false)

	if strings.Contains(string(result.output), "Unknown crawler type") {
		t.Fatalf("tieba hot_search should be accepted as a Python-compatible placeholder\n%s", result.output)
	}
	if _, err := os.Stat(filepath.Join(result.dataDir, "tieba")); err == nil {
		t.Fatalf("tieba hot_search placeholder should not write data files")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat tieba hot_search data dir: %v", err)
	}
}

type cliTiebaResult struct {
	output     []byte
	dataDir    string
	sqlitePath string
	tiebaURL   string
}

type tiebaSQLiteNote struct {
	ID              int64
	NoteID          string
	Title           string
	Desc            string `gorm:"column:desc"`
	NoteURL         string `gorm:"column:note_url"`
	PublishTime     string
	UserLink        string
	UserNickname    string
	UserAvatar      string
	TiebaID         string
	TiebaName       string
	TiebaLink       string
	TotalReplayNum  int
	TotalReplayPage int
	IPLocation      string `gorm:"column:ip_location"`
	AddTs           int64
	LastModifyTs    int64
	SourceKeyword   string
}

type tiebaSQLiteComment struct {
	ID              int64
	CommentID       string
	ParentCommentID string
	Content         string
	UserLink        string
	UserNickname    string
	UserAvatar      string
	TiebaID         string
	TiebaName       string
	TiebaLink       string
	PublishTime     string
	IPLocation      string `gorm:"column:ip_location"`
	SubCommentCount int
	NoteID          string
	NoteURL         string `gorm:"column:note_url"`
	AddTs           int64
	LastModifyTs    int64
}

type tiebaSQLiteCreator struct {
	ID                   int64
	UserID               string
	UserName             string
	Nickname             string
	Avatar               string
	IPLocation           string `gorm:"column:ip_location"`
	AddTs                int64
	LastModifyTs         int64
	Gender               string
	Follows              string
	Fans                 string
	RegistrationDuration string
}

func assertTiebaSearchMapRow(t *testing.T, row map[string]any, baseURL string) {
	t.Helper()
	if row["note_id"] != "tb_note_e2e" || row["title"] != "Tieba E2E mock note" || row["desc"] != "Tieba E2E mock content" {
		t.Fatalf("unexpected tieba identity/title fields: %#v", row)
	}
	if row["note_url"] != baseURL+"/p/tb_note_e2e" || row["publish_time"] != "2024-03-10 00:21:40" {
		t.Fatalf("unexpected tieba url/time fields: %#v", row)
	}
	if row["user_link"] != baseURL+"/home/main?id=tb-author-portrait" || row["user_nickname"] != "mock tieba author" || row["user_avatar"] != "https://example.test/tieba-avatar.png" {
		t.Fatalf("unexpected tieba user fields: %#v", row)
	}
	if row["tieba_name"] != "mock tieba吧" || row["tieba_link"] != baseURL+"/f?kw=mock%20tieba" {
		t.Fatalf("unexpected tieba forum fields: %#v", row)
	}
	if fmt.Sprint(row["total_replay_num"]) != "12" || fmt.Sprint(row["total_replay_page"]) != "2" {
		t.Fatalf("unexpected tieba replay fields: %#v", row)
	}
	if row["ip_location"] != "广东" || row["source_keyword"] != "mock-tieba" {
		t.Fatalf("unexpected tieba location/source fields: %#v", row)
	}
}

func assertTiebaDetailMapRow(t *testing.T, row map[string]any, baseURL string) {
	t.Helper()
	if row["note_id"] != "tb_note_e2e" || row["title"] != "Tieba E2E mock note" || row["desc"] != "Tieba E2E mock content" {
		t.Fatalf("unexpected tieba detail identity/title fields: %#v", row)
	}
	if row["note_url"] != baseURL+"/p/tb_note_e2e" || row["publish_time"] != "2024-03-10 00:21:40" {
		t.Fatalf("unexpected tieba detail url/time fields: %#v", row)
	}
	if row["user_link"] != baseURL+"/home/main?id=tb-author-portrait" || row["user_nickname"] != "mock tieba author" || row["user_avatar"] != "https://example.test/tieba-avatar.png" {
		t.Fatalf("unexpected tieba detail user fields: %#v", row)
	}
	if row["tieba_name"] != "mock tieba吧" || row["tieba_link"] != baseURL+"/f?kw=mock%20tieba" {
		t.Fatalf("unexpected tieba detail forum fields: %#v", row)
	}
	if fmt.Sprint(row["total_replay_num"]) != "12" || fmt.Sprint(row["total_replay_page"]) != "1" {
		t.Fatalf("unexpected tieba detail replay fields: %#v", row)
	}
	if row["ip_location"] != "广东" || row["source_keyword"] != "" {
		t.Fatalf("unexpected tieba detail location/source fields: %#v", row)
	}
}

func assertTiebaSearchCSVRow(t *testing.T, records [][]string, baseURL string) {
	t.Helper()
	if csvCell(records, "note_id") != "tb_note_e2e" || csvCell(records, "title") != "Tieba E2E mock note" || csvCell(records, "desc") != "Tieba E2E mock content" {
		t.Fatalf("unexpected tieba csv identity/title fields: %#v", records)
	}
	if csvCell(records, "note_url") != baseURL+"/p/tb_note_e2e" || csvCell(records, "publish_time") != "2024-03-10 00:21:40" {
		t.Fatalf("unexpected tieba csv url/time fields: %#v", records)
	}
	if csvCell(records, "user_link") != baseURL+"/home/main?id=tb-author-portrait" || csvCell(records, "user_nickname") != "mock tieba author" || csvCell(records, "user_avatar") != "https://example.test/tieba-avatar.png" {
		t.Fatalf("unexpected tieba csv user fields: %#v", records)
	}
	if csvCell(records, "tieba_name") != "mock tieba吧" || csvCell(records, "tieba_link") != baseURL+"/f?kw=mock%20tieba" {
		t.Fatalf("unexpected tieba csv forum fields: %#v", records)
	}
	if csvCell(records, "total_replay_num") != "12" || csvCell(records, "total_replay_page") != "2" {
		t.Fatalf("unexpected tieba csv replay fields: %#v", records)
	}
	if csvCell(records, "ip_location") != "广东" || csvCell(records, "source_keyword") != "mock-tieba" {
		t.Fatalf("unexpected tieba csv location/source fields: %#v", records)
	}
}

func assertTiebaDetailCSVRow(t *testing.T, records [][]string, baseURL string) {
	t.Helper()
	if csvCell(records, "note_id") != "tb_note_e2e" || csvCell(records, "title") != "Tieba E2E mock note" || csvCell(records, "desc") != "Tieba E2E mock content" {
		t.Fatalf("unexpected tieba detail csv identity/title fields: %#v", records)
	}
	if csvCell(records, "note_url") != baseURL+"/p/tb_note_e2e" || csvCell(records, "publish_time") != "2024-03-10 00:21:40" {
		t.Fatalf("unexpected tieba detail csv url/time fields: %#v", records)
	}
	if csvCell(records, "user_link") != baseURL+"/home/main?id=tb-author-portrait" || csvCell(records, "user_nickname") != "mock tieba author" || csvCell(records, "user_avatar") != "https://example.test/tieba-avatar.png" {
		t.Fatalf("unexpected tieba detail csv user fields: %#v", records)
	}
	if csvCell(records, "tieba_name") != "mock tieba吧" || csvCell(records, "tieba_link") != baseURL+"/f?kw=mock%20tieba" {
		t.Fatalf("unexpected tieba detail csv forum fields: %#v", records)
	}
	if csvCell(records, "total_replay_num") != "12" || csvCell(records, "total_replay_page") != "1" {
		t.Fatalf("unexpected tieba detail csv replay fields: %#v", records)
	}
	if csvCell(records, "ip_location") != "广东" || csvCell(records, "source_keyword") != "" {
		t.Fatalf("unexpected tieba detail csv location/source fields: %#v", records)
	}
}

func assertTiebaSQLiteNote(t *testing.T, row tiebaSQLiteNote, baseURL string) {
	t.Helper()
	if row.NoteID != "tb_note_e2e" || row.Title != "Tieba E2E mock note" || row.Desc != "Tieba E2E mock content" {
		t.Fatalf("unexpected tieba sqlite identity/title fields: %#v", row)
	}
	if row.NoteURL != baseURL+"/p/tb_note_e2e" || row.PublishTime != "2024-03-10 00:21:40" {
		t.Fatalf("unexpected tieba sqlite url/time fields: %#v", row)
	}
	if row.UserLink != baseURL+"/home/main?id=tb-author-portrait" || row.UserNickname != "mock tieba author" || row.UserAvatar != "https://example.test/tieba-avatar.png" {
		t.Fatalf("unexpected tieba sqlite user fields: %#v", row)
	}
	if row.TiebaID != "" || row.TiebaName != "mock tieba吧" || row.TiebaLink != baseURL+"/f?kw=mock%20tieba" {
		t.Fatalf("unexpected tieba sqlite forum fields: %#v", row)
	}
	if row.TotalReplayNum != 12 || row.TotalReplayPage != 2 {
		t.Fatalf("unexpected tieba sqlite replay fields: %#v", row)
	}
	if row.IPLocation != "广东" || row.SourceKeyword != "mock-tieba" || row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("unexpected tieba sqlite location/source/timestamps: %#v", row)
	}
}

func assertTiebaSQLiteDetailNote(t *testing.T, row tiebaSQLiteNote, baseURL string) {
	t.Helper()
	if row.NoteID != "tb_note_e2e" || row.Title != "Tieba E2E mock note" || row.Desc != "Tieba E2E mock content" {
		t.Fatalf("unexpected tieba sqlite detail identity/title fields: %#v", row)
	}
	if row.NoteURL != baseURL+"/p/tb_note_e2e" || row.PublishTime != "2024-03-10 00:21:40" {
		t.Fatalf("unexpected tieba sqlite detail url/time fields: %#v", row)
	}
	if row.UserLink != baseURL+"/home/main?id=tb-author-portrait" || row.UserNickname != "mock tieba author" || row.UserAvatar != "https://example.test/tieba-avatar.png" {
		t.Fatalf("unexpected tieba sqlite detail user fields: %#v", row)
	}
	if row.TiebaID != "" || row.TiebaName != "mock tieba吧" || row.TiebaLink != baseURL+"/f?kw=mock%20tieba" {
		t.Fatalf("unexpected tieba sqlite detail forum fields: %#v", row)
	}
	if row.TotalReplayNum != 12 || row.TotalReplayPage != 1 {
		t.Fatalf("unexpected tieba sqlite detail replay fields: %#v", row)
	}
	if row.IPLocation != "广东" || row.SourceKeyword != "" || row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("unexpected tieba sqlite detail location/source/timestamps: %#v", row)
	}
}

func assertTiebaCommentMapRow(t *testing.T, row map[string]any, baseURL, commentID, parentID, content, userLink, nickname, avatar, publishTime, ipLocation, subCount string) {
	t.Helper()
	if row["comment_id"] != commentID || row["parent_comment_id"] != parentID || row["content"] != content {
		t.Fatalf("unexpected tieba comment identity/content fields: %#v", row)
	}
	if row["user_link"] != userLink || row["user_nickname"] != nickname || row["user_avatar"] != avatar {
		t.Fatalf("unexpected tieba comment user fields: %#v", row)
	}
	if row["publish_time"] != publishTime || row["ip_location"] != ipLocation || fmt.Sprint(row["sub_comment_count"]) != subCount {
		t.Fatalf("unexpected tieba comment time/location/count fields: %#v", row)
	}
	if row["note_id"] != "tb_note_e2e" || row["note_url"] != baseURL+"/p/tb_note_e2e" {
		t.Fatalf("unexpected tieba comment note fields: %#v", row)
	}
	if row["tieba_id"] != "tb-forum-id" || row["tieba_name"] != "mock tieba吧" || row["tieba_link"] != baseURL+"/f?kw=mock%20tieba" {
		t.Fatalf("unexpected tieba comment forum fields: %#v", row)
	}
	if _, ok := row["add_ts"]; ok {
		t.Fatalf("tieba comment json should match Python and omit add_ts: %#v", row)
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected tieba comment json last_modify_ts: %#v", row)
	}
}

func assertTiebaCommentCSVRow(t *testing.T, records [][]string, row int, baseURL, commentID, parentID, content, userLink, nickname, avatar, publishTime, ipLocation, subCount string) {
	t.Helper()
	if csvCellAt(records, row, "comment_id") != commentID || csvCellAt(records, row, "parent_comment_id") != parentID || csvCellAt(records, row, "content") != content {
		t.Fatalf("unexpected tieba comment csv identity/content fields: %#v", records[row])
	}
	if csvCellAt(records, row, "user_link") != userLink || csvCellAt(records, row, "user_nickname") != nickname || csvCellAt(records, row, "user_avatar") != avatar {
		t.Fatalf("unexpected tieba comment csv user fields: %#v", records[row])
	}
	if csvCellAt(records, row, "publish_time") != publishTime || csvCellAt(records, row, "ip_location") != ipLocation || csvCellAt(records, row, "sub_comment_count") != subCount {
		t.Fatalf("unexpected tieba comment csv time/location/count fields: %#v", records[row])
	}
	if csvCellAt(records, row, "note_id") != "tb_note_e2e" || csvCellAt(records, row, "note_url") != baseURL+"/p/tb_note_e2e" {
		t.Fatalf("unexpected tieba comment csv note fields: %#v", records[row])
	}
	if csvCellAt(records, row, "tieba_id") != "tb-forum-id" || csvCellAt(records, row, "tieba_name") != "mock tieba吧" || csvCellAt(records, row, "tieba_link") != baseURL+"/f?kw=mock%20tieba" {
		t.Fatalf("unexpected tieba comment csv forum fields: %#v", records[row])
	}
}

func assertTiebaSQLiteComment(t *testing.T, row tiebaSQLiteComment, baseURL, commentID, parentID, content, userLink, nickname, avatar, publishTime, ipLocation string, subCount int) {
	t.Helper()
	if row.CommentID != commentID || row.ParentCommentID != parentID || row.Content != content {
		t.Fatalf("unexpected tieba sqlite comment identity/content fields: %#v", row)
	}
	if row.UserLink != userLink || row.UserNickname != nickname || row.UserAvatar != avatar {
		t.Fatalf("unexpected tieba sqlite comment user fields: %#v", row)
	}
	if row.PublishTime != publishTime || row.IPLocation != ipLocation || row.SubCommentCount != subCount {
		t.Fatalf("unexpected tieba sqlite comment time/location/count fields: %#v", row)
	}
	if row.NoteID != "tb_note_e2e" || row.NoteURL != baseURL+"/p/tb_note_e2e" {
		t.Fatalf("unexpected tieba sqlite comment note fields: %#v", row)
	}
	if row.TiebaID != "tb-forum-id" || row.TiebaName != "mock tieba吧" || row.TiebaLink != baseURL+"/f?kw=mock%20tieba" {
		t.Fatalf("unexpected tieba sqlite comment forum fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected tieba sqlite comment timestamps: %#v", row)
	}
}

func assertTiebaCreatorMapRow(t *testing.T, row map[string]any) {
	t.Helper()
	if row["user_id"] != "tb-creator-user" || row["user_name"] != "mock_tieba_creator" || row["nickname"] != "mock tieba creator" {
		t.Fatalf("unexpected tieba creator identity fields: %#v", row)
	}
	if row["gender"] != "女" || row["avatar"] != "https://example.test/tieba-creator-avatar.png" || row["ip_location"] != "浙江" {
		t.Fatalf("unexpected tieba creator profile fields: %#v", row)
	}
	if fmt.Sprint(row["follows"]) != "12" || fmt.Sprint(row["fans"]) != "345" || row["registration_duration"] != "5.6年" {
		t.Fatalf("unexpected tieba creator count/duration fields: %#v", row)
	}
	if _, ok := row["add_ts"]; ok {
		t.Fatalf("tieba creator json should match Python and omit add_ts: %#v", row)
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected tieba creator json last_modify_ts: %#v", row)
	}
}

func assertTiebaCreatorCSVRow(t *testing.T, records [][]string) {
	t.Helper()
	if csvCell(records, "user_id") != "tb-creator-user" || csvCell(records, "user_name") != "mock_tieba_creator" || csvCell(records, "nickname") != "mock tieba creator" {
		t.Fatalf("unexpected tieba creator csv identity fields: %#v", records)
	}
	if csvCell(records, "gender") != "女" || csvCell(records, "avatar") != "https://example.test/tieba-creator-avatar.png" || csvCell(records, "ip_location") != "浙江" {
		t.Fatalf("unexpected tieba creator csv profile fields: %#v", records)
	}
	if csvCell(records, "follows") != "12" || csvCell(records, "fans") != "345" || csvCell(records, "registration_duration") != "5.6年" {
		t.Fatalf("unexpected tieba creator csv count/duration fields: %#v", records)
	}
	if csvCell(records, "add_ts") != "" {
		t.Fatalf("tieba creator csv should not include add_ts value: %#v", csvCell(records, "add_ts"))
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected tieba creator csv last_modify_ts")
	}
}

func assertTiebaSQLiteCreator(t *testing.T, row tiebaSQLiteCreator) {
	t.Helper()
	if row.UserID != "tb-creator-user" || row.UserName != "mock_tieba_creator" || row.Nickname != "mock tieba creator" {
		t.Fatalf("unexpected tieba sqlite creator identity fields: %#v", row)
	}
	if row.Gender != "女" || row.Avatar != "https://example.test/tieba-creator-avatar.png" || row.IPLocation != "浙江" {
		t.Fatalf("unexpected tieba sqlite creator profile fields: %#v", row)
	}
	if row.Follows != "12" || row.Fans != "345" || row.RegistrationDuration != "5.6年" {
		t.Fatalf("unexpected tieba sqlite creator count/duration fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected tieba sqlite creator timestamps: %#v", row)
	}
}

func runCLITiebaSearchWithMockServers(t *testing.T, saveDataOption string) cliTiebaResult {
	return runCLITiebaWithMockServers(t, "search", saveDataOption, nil, false, false)
}

func runCLITiebaWithMockServers(t *testing.T, crawlerType, saveDataOption string, noteIDs []string, enableComments, enableSubComments bool) cliTiebaResult {
	t.Helper()

	repoRoot := e2e.RepoRoot(t)
	mockServer := newTiebaMockServer(t, crawlerType)
	defer mockServer.Close()

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

	host, port := serverHostPort(t, mockServer.URL)
	specifiedIDsYAML := ""
	if len(noteIDs) > 0 {
		quotedIDs := make([]string, 0, len(noteIDs))
		for _, noteID := range noteIDs {
			quotedIDs = append(quotedIDs, fmt.Sprintf("%q", noteID))
		}
		if crawlerType == "creator" {
			specifiedIDsYAML = fmt.Sprintf("specifiedCreatorIds: [%s]\nspecifiedCreatorURLs: [%s]\n", strings.Join(quotedIDs, ", "), strings.Join(quotedIDs, ", "))
		} else {
			specifiedIDsYAML = fmt.Sprintf("specifiedNoteIds: [%s]\nspecifiedNoteURLs: [%s]\n", strings.Join(quotedIDs, ", "), strings.Join(quotedIDs, ", "))
		}
	}
	configYAML := fmt.Sprintf(`
platform: "tieba"
keywords:
  - "mock-tieba"
crawlerType: %q
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
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  tiebaUrl: %q
%s`, crawlerType, saveDataOption, mockServer.URL, dataDir, sqlitePath, enableComments, enableSubComments, host, port, mockServer.URL, specifiedIDsYAML)
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
		"MCPRO_TIEBA_URL="+mockServer.URL,
	)
	if saveDataOption == "sqlite" {
		runCmd.Env = append(runCmd.Env, "MCPRO_SQLITE_DB_PATH="+sqlitePath)
	}
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("tieba crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("tieba crawler command failed: %v\n%s", err, output)
	}
	return cliTiebaResult{output: output, dataDir: dataDir, sqlitePath: sqlitePath, tiebaURL: mockServer.URL}
}

func newTiebaMockServer(t *testing.T, crawlerType string) *httptest.Server {
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
							"client_id": "tieba-client",
							"platforms": map[string]any{
								"tieba": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"tieba": "mock-tieba"},
						},
					},
				},
			})
		case "/api/cookies/tieba":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "BDUSS=mock-tieba-bduss; STOKEN=mock-tieba-stoken"},
			})
		case "/c/s/pc/sync":
			writeJSON(t, w, map[string]any{
				"error_code": 0,
				"data": map[string]any{
					"anti": map[string]string{"tbs": "mock-tieba-tbs"},
					"user": map[string]bool{"is_login": true},
				},
			})
		case "/mo/q/search/multsearch":
			if r.URL.Query().Get("pn") != "1" {
				writeJSON(t, w, map[string]any{"error_code": 0, "data": map[string]any{"card_list": []map[string]any{}}})
				return
			}
			writeJSON(t, w, tiebaSearchPayload())
		case "/c/u/pc/homeSidebarRight":
			writeJSON(t, w, tiebaCreatorPayload())
		case "/c/u/feed/myThread":
			if r.URL.Query().Get("pn") != "1" {
				writeJSON(t, w, map[string]any{"error_code": 0, "data": map[string]any{"has_more": 0, "list": []map[string]any{}}})
				return
			}
			writeJSON(t, w, tiebaCreatorThreadsPayload())
		case "/c/f/pb/page_pc":
			if crawlerType == "detail" || crawlerType == "creator" {
				writeJSON(t, w, tiebaDetailPayload(1, true))
				return
			}
			writeJSON(t, w, tiebaDetailPayload(2, false))
		case "/p/comment":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(tiebaSubCommentsHTML()))
		default:
			t.Fatalf("unexpected tieba request: %s %s", r.Method, r.URL.String())
		}
	}))
}

func tiebaSearchPayload() map[string]any {
	return map[string]any{
		"error_code": 0,
		"data": map[string]any{
			"card_list": []map[string]any{
				{
					"cardInfo":  "thread",
					"cardStyle": "thread",
					"data": map[string]any{
						"tid":        "tb_note_e2e",
						"title":      "Tieba search list title",
						"content":    "search list content should not be final",
						"time":       1710001200,
						"forum_name": "mock tieba",
						"post_num":   9,
						"user": map[string]any{
							"show_nickname": "mock tieba search author",
							"portrait":      "tb-search-portrait",
						},
					},
				},
			},
		},
	}
}

func tiebaSearchResumePayload(noteID string) map[string]any {
	return map[string]any{
		"error_code": 0,
		"data": map[string]any{
			"card_list": []map[string]any{
				{
					"cardInfo":  "thread",
					"cardStyle": "thread",
					"data": map[string]any{
						"tid":        noteID,
						"title":      "Tieba search resume list title",
						"content":    "resume search list content should not be final",
						"time":       1710001250,
						"forum_name": "mock tieba",
						"post_num":   19,
						"user": map[string]any{
							"show_nickname": "mock tieba resume search author",
							"portrait":      "tb-search-resume-portrait",
						},
					},
				},
			},
		},
	}
}

func tiebaDetailResumePayload(noteID, title, content string) map[string]any {
	return map[string]any{
		"error_code": 0,
		"thread": map[string]any{
			"id":          noteID,
			"tid":         noteID,
			"title":       title,
			"reply_num":   22,
			"create_time": 1710001350,
		},
		"first_floor": map[string]any{
			"tid":       noteID,
			"title":     title,
			"author_id": "tb-resume-author-id",
			"content": []map[string]string{
				{"text": content},
			},
			"time": 1710001350,
		},
		"forum": map[string]any{
			"id":   "tb-resume-forum-id",
			"name": "mock tieba",
		},
		"page": map[string]any{"total_page": 1},
		"user_list": []map[string]any{
			{
				"id":         "tb-resume-author-id",
				"name":       "mock_tieba_resume_author",
				"name_show":  "mock tieba resume author",
				"portrait":   "tb-resume-author-portrait",
				"ip_address": "广东",
				"user_show_info": map[string]any{
					"feed_head": map[string]any{
						"image_data": map[string]string{
							"img_url": "https://example.test/tieba-resume-avatar.png",
						},
					},
				},
			},
		},
		"post_list": []map[string]any{},
	}
}

func tiebaCreatorPayload() map[string]any {
	return map[string]any{
		"error_code": 0,
		"data": map[string]any{
			"user": map[string]any{
				"id":          "tb-creator-user",
				"name":        "mock_tieba_creator",
				"name_show":   "mock tieba creator",
				"portrait":    "tb-creator-portrait",
				"sex":         2,
				"ip_address":  "浙江",
				"concern_num": 12,
				"fans_num":    345,
				"tb_age":      "5.6年",
				"user_show_info": map[string]any{
					"feed_head": map[string]any{
						"image_data": map[string]string{
							"img_url": "https://example.test/tieba-creator-avatar.png",
						},
					},
				},
			},
		},
	}
}

func tiebaCreatorResumePayload() map[string]any {
	return map[string]any{
		"error_code": 0,
		"data": map[string]any{
			"user": map[string]any{
				"id":          "tb-creator-resume-user",
				"name":        "mock_tieba_creator_resume",
				"name_show":   "mock tieba creator resume",
				"portrait":    "tb-creator-resume-portrait",
				"sex":         2,
				"ip_address":  "浙江",
				"concern_num": 12,
				"fans_num":    345,
				"tb_age":      "5.6年",
				"user_show_info": map[string]any{
					"feed_head": map[string]any{
						"image_data": map[string]string{
							"img_url": "https://example.test/tieba-creator-resume-avatar.png",
						},
					},
				},
			},
		},
	}
}

func tiebaCreatorThreadsPayload() map[string]any {
	return map[string]any{
		"error_code": 0,
		"data": map[string]any{
			"has_more": 0,
			"list": []map[string]any{
				{"thread_info": map[string]any{"tid": "tb_note_e2e"}},
			},
		},
	}
}

func tiebaCreatorThreadsResumePayload(noteID string, hasMore bool) map[string]any {
	hasMoreValue := 0
	if hasMore {
		hasMoreValue = 1
	}
	return map[string]any{
		"error_code": 0,
		"data": map[string]any{
			"has_more": hasMoreValue,
			"list": []map[string]any{
				{"thread_info": map[string]any{"tid": noteID}},
			},
		},
	}
}

func tiebaDetailPayload(totalPage int, includeComments bool) map[string]any {
	postList := []map[string]any{}
	userList := []map[string]any{
		{
			"id":         "tb-author-id",
			"name":       "mock_tieba_author",
			"name_show":  "mock tieba author",
			"portrait":   "tb-author-portrait",
			"ip_address": "广东",
			"user_show_info": map[string]any{
				"feed_head": map[string]any{
					"image_data": map[string]string{
						"img_url": "https://example.test/tieba-avatar.png",
					},
				},
			},
		},
	}
	if includeComments {
		postList = append(postList, map[string]any{
			"id":              "tb_comment_root",
			"author_id":       "tb-commenter-id",
			"sub_post_number": 1,
			"content": []map[string]string{
				{"text": "root tieba comment"},
			},
			"time": 1710001400,
		})
		userList = append(userList, map[string]any{
			"id":         "tb-commenter-id",
			"name":       "mock_tieba_commenter",
			"name_show":  "mock tieba commenter",
			"portrait":   "tb-commenter-portrait",
			"ip_address": "上海",
			"user_show_info": map[string]any{
				"feed_head": map[string]any{
					"image_data": map[string]string{
						"img_url": "https://example.test/tieba-comment-avatar.png",
					},
				},
			},
		})
	}
	return map[string]any{
		"error_code": 0,
		"thread": map[string]any{
			"id":          "tb_note_e2e",
			"tid":         "tb_note_e2e",
			"title":       "Tieba E2E mock note",
			"reply_num":   12,
			"create_time": 1710001300,
		},
		"first_floor": map[string]any{
			"tid":       "tb_note_e2e",
			"title":     "Tieba E2E mock note",
			"author_id": "tb-author-id",
			"content": []map[string]string{
				{"text": "Tieba E2E mock content"},
			},
			"time": 1710001300,
		},
		"forum": map[string]any{
			"id":   "tb-forum-id",
			"name": "mock tieba",
		},
		"page":      map[string]any{"total_page": totalPage},
		"user_list": userList,
		"post_list": postList,
	}
}

func tiebaSubCommentsHTML() string {
	return `<html><body><ul>
<li data-field='{"spid":"tb_comment_sub","showname":"mock tieba sub commenter"}'>
  <a href="/home/main?id=tb-sub-commenter-portrait"><img src="https://example.test/tieba-sub-comment-avatar.png"/></a>
  <span class="lzl_content_main">sub tieba comment</span>
  <span class="lzl_time">2024-03-10 00:24</span>
</li>
</ul></body></html>`
}
