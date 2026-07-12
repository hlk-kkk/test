// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/zhihu_e2e_test.go
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

func TestCLIZhihuSearchJSONWithMockServers(t *testing.T) {
	result := runCLIZhihuSearchWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "zhihu", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, result.output)
	if len(rows) != 1 {
		t.Fatalf("expected one zhihu json row, got %d: %#v", len(rows), rows)
	}
	assertZhihuSearchMapRow(t, rows[0], result.zhihuURL)
	if _, ok := rows[0]["add_ts"]; ok {
		t.Fatalf("zhihu json should match Python and omit add_ts: %#v", rows[0])
	}
	if _, ok := rows[0]["last_modify_ts"]; !ok {
		t.Fatalf("expected zhihu json last_modify_ts: %#v", rows[0])
	}
}

func TestCLIZhihuSearchCSVWithMockServers(t *testing.T) {
	result := runCLIZhihuSearchWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "zhihu", "1_search_contents_"+time.Now().Format("2006-01-02")+".csv")
	records := readCSVRecords(t, resultPath, result.output)
	if len(records) != 2 {
		t.Fatalf("expected zhihu csv header and one row, got %d: %#v", len(records), records)
	}
	assertZhihuSearchCSVRow(t, records, result.zhihuURL)
	if csvCell(records, "add_ts") != "" {
		t.Fatalf("zhihu csv should not include add_ts value: %#v", csvCell(records, "add_ts"))
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected zhihu csv last_modify_ts")
	}
}

func TestCLIZhihuSearchSQLiteWithMockServers(t *testing.T) {
	result := runCLIZhihuSearchWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open zhihu sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []zhihuSQLiteContent
	if err := db.Table("zhihu_content").Find(&rows).Error; err != nil {
		t.Fatalf("query zhihu sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one zhihu sqlite row, got %d: %#v", len(rows), rows)
	}
	assertZhihuSQLiteContent(t, rows[0], result.zhihuURL)

	var commentCount int64
	if err := db.Table("zhihu_comment").Count(&commentCount).Error; err != nil {
		t.Fatalf("expected zhihu_comment table to match Python sqlite schema: %v\ncommand output:\n%s", err, result.output)
	}
	if commentCount != 0 {
		t.Fatalf("expected no zhihu sqlite comments when comments disabled, got %d", commentCount)
	}
	var creatorCount int64
	if err := db.Table("zhihu_creator").Count(&creatorCount).Error; err != nil {
		t.Fatalf("expected zhihu_creator table to match Python sqlite schema: %v\ncommand output:\n%s", err, result.output)
	}
	if creatorCount != 0 {
		t.Fatalf("expected no zhihu sqlite creators during search, got %d", creatorCount)
	}
}

func TestCLIZhihuSearchCheckpointResumeSkipsCompletedPages(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	searchCallsByOffset := make(map[string]int)
	allowPage2 := false
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
							"client_id": "zhihu-resume-client",
							"platforms": map[string]any{
								"zhihu": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"zhihu": "mock-zhihu"},
						},
					},
				},
			})
		case "/api/cookies/zhihu":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "z_c0=resume-zc0; q_c1=resume-qc1"},
			})
		case "/api/v4/me":
			writeJSON(t, w, map[string]any{"uid": "zh-resume-user", "name": "mock zhihu resume account"})
		case "/signsrv/v1/zhihu/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_zst_81": "resume-x-zst-81",
					"x_zse_96": "resume-x-zse-96",
				},
			})
		case "/api/v4/search_v3":
			if r.URL.Query().Get("q") != "resume-zhihu" {
				t.Fatalf("unexpected zhihu resume keyword: %s", r.URL.RawQuery)
			}
			offset := r.URL.Query().Get("offset")
			searchCallsByOffset[offset]++
			switch offset {
			case "0":
				writeJSON(t, w, zhihuResumeSearchPayload("zh_resume_answer_1", "zh_resume_question_1", "Zhihu resume question 1", 1710100000))
			case "20":
				if !allowPage2 {
					http.Error(w, "transient page 2 failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, zhihuResumeSearchPayload("zh_resume_answer_2", "zh_resume_question_2", "Zhihu resume question 2", 1710100100))
			case "40":
				writeJSON(t, w, map[string]any{"data": []map[string]any{}})
			default:
				t.Fatalf("unexpected zhihu resume search offset: %s (%s)", offset, r.URL.RawQuery)
			}
		default:
			t.Fatalf("unexpected zhihu resume request: %s %s", r.Method, r.URL.String())
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
platform: "zhihu"
keywords:
  - "resume-zhihu"
crawlerType: "search"
saveDataOption: "json"
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
startPage: 1
crawlerMaxNotesCount: 21
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
  zhihuUrl: %q
  zhihuZhuanlanUrl: %q
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
			"MCPRO_ZHIHU_URL="+mockServer.URL,
			"MCPRO_ZHIHU_ZHUANLAN_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("zhihu resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("zhihu resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if searchCallsByOffset["0"] != 1 || searchCallsByOffset["20"] != 1 {
		t.Fatalf("first run should crawl page 1 and fail on page 2 once, calls=%#v\n%s", searchCallsByOffset, firstOutput)
	}

	allowPage2 = true
	secondOutput := runCrawler()
	if searchCallsByOffset["0"] != 1 {
		t.Fatalf("resume should not re-fetch completed page 1, calls=%#v\n%s", searchCallsByOffset, secondOutput)
	}
	if searchCallsByOffset["20"] != 2 {
		t.Fatalf("resume should retry page 2 exactly once after transient failure, calls=%#v\n%s", searchCallsByOffset, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "zhihu", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read zhihu resume json: %v\n%s", err, secondOutput)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal zhihu resume json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Zhihu rows after resume, got %d: %s", len(rows), body)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["content_id"])] = true
	}
	if !seen["zh_resume_answer_1"] || !seen["zh_resume_answer_2"] {
		t.Fatalf("resume output missing expected Zhihu content IDs: seen=%#v rows=%s", seen, body)
	}
}

func TestCLIZhihuDetailCheckpointResumeSkipsCompletedIDs(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	detailCallsByAnswerID := make(map[string]int)
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
							"client_id": "zhihu-detail-resume-client",
							"platforms": map[string]any{
								"zhihu": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"zhihu": "mock-zhihu"},
						},
					},
				},
			})
		case "/api/cookies/zhihu":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "z_c0=mock-zhihu-detail-resume-zc0; q_c1=mock-zhihu-detail-resume-qc1"},
			})
		case "/api/v4/me":
			writeJSON(t, w, map[string]any{"uid": "zh-current-user", "name": "mock zhihu account"})
		case "/signsrv/v1/zhihu/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_zst_81": "detail-resume-x-zst-81",
					"x_zse_96": "detail-resume-x-zse-96",
				},
			})
		case "/question/zh_detail_resume_question_1/answer/zh_detail_resume_answer_1":
			detailCallsByAnswerID["zh_detail_resume_answer_1"]++
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(zhihuDetailHTMLFor("zh_detail_resume_question_1", "zh_detail_resume_answer_1", "Zhihu detail resume answer 1", "zh-detail-resume-author-1")))
		case "/question/zh_detail_resume_question_2/answer/zh_detail_resume_answer_2":
			detailCallsByAnswerID["zh_detail_resume_answer_2"]++
			if !allowSecondDetail {
				http.Error(w, "transient second detail failure", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(zhihuDetailHTMLFor("zh_detail_resume_question_2", "zh_detail_resume_answer_2", "Zhihu detail resume answer 2", "zh-detail-resume-author-2")))
		default:
			t.Fatalf("unexpected zhihu detail resume request: %s %s", r.Method, r.URL.String())
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
	firstURL := mockServer.URL + "/question/zh_detail_resume_question_1/answer/zh_detail_resume_answer_1"
	secondURL := mockServer.URL + "/question/zh_detail_resume_question_2/answer/zh_detail_resume_answer_2"
	configYAML := fmt.Sprintf(`
platform: "zhihu"
keywords:
  - "unused"
crawlerType: "detail"
saveDataOption: "json"
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
specifiedNoteIds: [%q, %q]
specifiedNoteURLs: [%q, %q]
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
  zhihuUrl: %q
  zhihuZhuanlanUrl: %q
`, mockServer.URL, dataDir, firstURL, secondURL, firstURL, secondURL, host, port, mockServer.URL, mockServer.URL)
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
			"MCPRO_ZHIHU_URL="+mockServer.URL,
			"MCPRO_ZHIHU_ZHUANLAN_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("zhihu detail resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("zhihu detail resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if detailCallsByAnswerID["zh_detail_resume_answer_1"] != 1 || detailCallsByAnswerID["zh_detail_resume_answer_2"] != 1 {
		t.Fatalf("first zhihu detail run should fetch both IDs and fail second once, calls=%#v\n%s", detailCallsByAnswerID, firstOutput)
	}

	allowSecondDetail = true
	secondOutput := runCrawler()
	if detailCallsByAnswerID["zh_detail_resume_answer_1"] != 1 {
		t.Fatalf("resume should skip completed first zhihu detail ID, calls=%#v\n%s", detailCallsByAnswerID, secondOutput)
	}
	if detailCallsByAnswerID["zh_detail_resume_answer_2"] != 2 {
		t.Fatalf("resume should retry second zhihu detail ID exactly once, calls=%#v\n%s", detailCallsByAnswerID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "zhihu", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Zhihu detail rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["content_id"])] = true
	}
	if !seen["zh_detail_resume_answer_1"] || !seen["zh_detail_resume_answer_2"] {
		t.Fatalf("zhihu detail resume output missing expected content IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIZhihuDetailJSONWithMockServers(t *testing.T) {
	result := runCLIZhihuDetailWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "zhihu", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, result.output)
	if len(rows) != 1 {
		t.Fatalf("expected one zhihu detail json row, got %d: %#v", len(rows), rows)
	}
	assertZhihuDetailMapRow(t, rows[0], result.zhihuURL)
	if _, ok := rows[0]["add_ts"]; ok {
		t.Fatalf("zhihu detail json should match Python and omit add_ts: %#v", rows[0])
	}
	if _, ok := rows[0]["last_modify_ts"]; !ok {
		t.Fatalf("expected zhihu detail json last_modify_ts: %#v", rows[0])
	}
}

func TestCLIZhihuDetailCSVWithMockServers(t *testing.T) {
	result := runCLIZhihuDetailWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "zhihu", "1_detail_contents_"+time.Now().Format("2006-01-02")+".csv")
	records := readCSVRecords(t, resultPath, result.output)
	if len(records) != 2 {
		t.Fatalf("expected zhihu detail csv header and one row, got %d: %#v", len(records), records)
	}
	assertZhihuDetailCSVRow(t, records, result.zhihuURL)
	if csvCell(records, "add_ts") != "" {
		t.Fatalf("zhihu detail csv should not include add_ts value: %#v", csvCell(records, "add_ts"))
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected zhihu detail csv last_modify_ts")
	}
}

func TestCLIZhihuDetailSQLiteWithMockServers(t *testing.T) {
	result := runCLIZhihuDetailWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open zhihu detail sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []zhihuSQLiteContent
	if err := db.Table("zhihu_content").Find(&rows).Error; err != nil {
		t.Fatalf("query zhihu detail sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one zhihu detail sqlite row, got %d: %#v", len(rows), rows)
	}
	assertZhihuSQLiteDetailContent(t, rows[0], result.zhihuURL)

	var commentCount int64
	if err := db.Table("zhihu_comment").Count(&commentCount).Error; err != nil {
		t.Fatalf("expected zhihu_comment table to match Python sqlite schema: %v\ncommand output:\n%s", err, result.output)
	}
	if commentCount != 0 {
		t.Fatalf("expected no zhihu detail sqlite comments when comments disabled, got %d", commentCount)
	}
	var creatorCount int64
	if err := db.Table("zhihu_creator").Count(&creatorCount).Error; err != nil {
		t.Fatalf("expected zhihu_creator table to match Python sqlite schema: %v\ncommand output:\n%s", err, result.output)
	}
	if creatorCount != 0 {
		t.Fatalf("expected no zhihu detail sqlite creators during detail, got %d", creatorCount)
	}
}

func TestCLIZhihuDetailCommentsJSONWithMockServers(t *testing.T) {
	result := runCLIZhihuDetailCommentsWithMockServers(t, "json")

	contentsPath := filepath.Join(result.dataDir, "zhihu", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	contentRows := readJSONRows(t, contentsPath, result.output)
	if len(contentRows) != 1 {
		t.Fatalf("expected one zhihu detail content json row with comments enabled, got %d: %#v", len(contentRows), contentRows)
	}
	assertZhihuDetailMapRow(t, contentRows[0], result.zhihuURL)

	commentsPath := filepath.Join(result.dataDir, "zhihu", "json", "detail_comments_"+time.Now().Format("2006-01-02")+".json")
	commentRows := readJSONRows(t, commentsPath, result.output)
	if len(commentRows) != 2 {
		t.Fatalf("expected root and sub zhihu comments, got %d: %#v", len(commentRows), commentRows)
	}
	assertZhihuCommentMapRow(t, commentRows[0], result.zhihuURL, "zh_comment_root", "", "root zhihu comment", "1710001900", "广东", "1", "12", "2", "zh-commenter", "mock zhihu commenter", "https://example.test/zhihu-comment-avatar.png", "mock-zh-commenter")
	assertZhihuCommentMapRow(t, commentRows[1], result.zhihuURL, "zh_comment_sub", "zh_comment_root", "sub zhihu comment", "1710001910", "深圳", "0", "3", "0", "zh-sub-commenter", "mock zhihu sub commenter", "https://example.test/zhihu-sub-comment-avatar.png", "mock-zh-sub-commenter")
}

func TestCLIZhihuDetailCommentsCSVWithMockServers(t *testing.T) {
	result := runCLIZhihuDetailCommentsWithMockServers(t, "csv")

	contentsPath := filepath.Join(result.dataDir, "zhihu", "1_detail_contents_"+time.Now().Format("2006-01-02")+".csv")
	contentRecords := readCSVRecords(t, contentsPath, result.output)
	if len(contentRecords) != 2 {
		t.Fatalf("expected zhihu detail csv header and one content row with comments enabled, got %d: %#v", len(contentRecords), contentRecords)
	}
	assertZhihuDetailCSVRow(t, contentRecords, result.zhihuURL)

	commentsPath := filepath.Join(result.dataDir, "zhihu", "1_detail_comments_"+time.Now().Format("2006-01-02")+".csv")
	commentRecords := readCSVRecords(t, commentsPath, result.output)
	if len(commentRecords) != 3 {
		t.Fatalf("expected zhihu comments csv header plus root/sub comments, got %d: %#v", len(commentRecords), commentRecords)
	}
	assertZhihuCommentCSVRow(t, commentRecords, 1, result.zhihuURL, "zh_comment_root", "", "root zhihu comment", "1710001900", "广东", "1", "12", "2", "zh-commenter", "mock zhihu commenter", "https://example.test/zhihu-comment-avatar.png", "mock-zh-commenter")
	assertZhihuCommentCSVRow(t, commentRecords, 2, result.zhihuURL, "zh_comment_sub", "zh_comment_root", "sub zhihu comment", "1710001910", "深圳", "0", "3", "0", "zh-sub-commenter", "mock zhihu sub commenter", "https://example.test/zhihu-sub-comment-avatar.png", "mock-zh-sub-commenter")
	if csvCellAt(commentRecords, 1, "add_ts") != "" || csvCellAt(commentRecords, 2, "add_ts") != "" {
		t.Fatalf("zhihu comment csv should match Python and omit add_ts values: %#v", commentRecords)
	}
	if csvCellAt(commentRecords, 1, "last_modify_ts") == "" || csvCellAt(commentRecords, 2, "last_modify_ts") == "" {
		t.Fatalf("expected zhihu comment csv last_modify_ts values: %#v", commentRecords)
	}
}

func TestCLIZhihuDetailCommentsSQLiteWithMockServers(t *testing.T) {
	result := runCLIZhihuDetailCommentsWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open zhihu detail comments sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var contents []zhihuSQLiteContent
	if err := db.Table("zhihu_content").Find(&contents).Error; err != nil {
		t.Fatalf("query zhihu sqlite contents with comments: %v\ncommand output:\n%s", err, result.output)
	}
	if len(contents) != 1 {
		t.Fatalf("expected one zhihu sqlite content with comments, got %d: %#v", len(contents), contents)
	}
	assertZhihuSQLiteDetailContent(t, contents[0], result.zhihuURL)

	var comments []zhihuSQLiteComment
	if err := db.Table("zhihu_comment").Order("comment_id").Find(&comments).Error; err != nil {
		t.Fatalf("query zhihu sqlite comments: %v\ncommand output:\n%s", err, result.output)
	}
	if len(comments) != 2 {
		t.Fatalf("expected root and sub zhihu sqlite comments, got %d: %#v", len(comments), comments)
	}
	assertZhihuSQLiteComment(t, comments[0], result.zhihuURL, "zh_comment_root", "", "root zhihu comment", "1710001900", "广东", 1, 12, 2, "zh-commenter", "mock zhihu commenter", "https://example.test/zhihu-comment-avatar.png", "mock-zh-commenter")
	assertZhihuSQLiteComment(t, comments[1], result.zhihuURL, "zh_comment_sub", "zh_comment_root", "sub zhihu comment", "1710001910", "深圳", 0, 3, 0, "zh-sub-commenter", "mock zhihu sub commenter", "https://example.test/zhihu-sub-comment-avatar.png", "mock-zh-sub-commenter")
}

func TestCLIZhihuCreatorJSONWithMockServers(t *testing.T) {
	result := runCLIZhihuCreatorWithMockServers(t, "json")

	creatorPath := filepath.Join(result.dataDir, "zhihu", "json", "creator_creator_"+time.Now().Format("2006-01-02")+".json")
	creatorRows := readJSONRows(t, creatorPath, result.output)
	if len(creatorRows) != 1 {
		t.Fatalf("expected one zhihu creator json row, got %d: %#v", len(creatorRows), creatorRows)
	}
	assertZhihuCreatorMapRow(t, creatorRows[0], result.zhihuURL)

	contentsPath := filepath.Join(result.dataDir, "zhihu", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	contentRows := readJSONRows(t, contentsPath, result.output)
	if len(contentRows) != 1 {
		t.Fatalf("expected one zhihu creator content json row, got %d: %#v", len(contentRows), contentRows)
	}
	assertZhihuCreatorContentMapRow(t, contentRows[0], result.zhihuURL)
}

func TestCLIZhihuCreatorCSVWithMockServers(t *testing.T) {
	result := runCLIZhihuCreatorWithMockServers(t, "csv")

	creatorPath := filepath.Join(result.dataDir, "zhihu", "1_creator_creator_"+time.Now().Format("2006-01-02")+".csv")
	creatorRecords := readCSVRecords(t, creatorPath, result.output)
	if len(creatorRecords) != 2 {
		t.Fatalf("expected zhihu creator csv header and one row, got %d: %#v", len(creatorRecords), creatorRecords)
	}
	assertZhihuCreatorCSVRow(t, creatorRecords, result.zhihuURL)

	contentsPath := filepath.Join(result.dataDir, "zhihu", "1_creator_contents_"+time.Now().Format("2006-01-02")+".csv")
	contentRecords := readCSVRecords(t, contentsPath, result.output)
	if len(contentRecords) != 2 {
		t.Fatalf("expected zhihu creator content csv header and one row, got %d: %#v", len(contentRecords), contentRecords)
	}
	assertZhihuCreatorContentCSVRow(t, contentRecords, result.zhihuURL)
}

func TestCLIZhihuCreatorSQLiteWithMockServers(t *testing.T) {
	result := runCLIZhihuCreatorWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open zhihu creator sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var creators []zhihuSQLiteCreator
	if err := db.Table("zhihu_creator").Find(&creators).Error; err != nil {
		t.Fatalf("query zhihu sqlite creators: %v\ncommand output:\n%s", err, result.output)
	}
	if len(creators) != 1 {
		t.Fatalf("expected one zhihu sqlite creator, got %d: %#v", len(creators), creators)
	}
	assertZhihuSQLiteCreator(t, creators[0], result.zhihuURL)

	var contents []zhihuSQLiteContent
	if err := db.Table("zhihu_content").Find(&contents).Error; err != nil {
		t.Fatalf("query zhihu sqlite creator contents: %v\ncommand output:\n%s", err, result.output)
	}
	if len(contents) != 1 {
		t.Fatalf("expected one zhihu sqlite creator content, got %d: %#v", len(contents), contents)
	}
	assertZhihuSQLiteCreatorContent(t, contents[0], result.zhihuURL)
}

func TestCLIZhihuCreatorCheckpointResumeByCreatorAndOffset(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	profileCalls := 0
	answerCallsByOffset := make(map[string]int)
	allowSecondOffset := false
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
							"client_id": "zhihu-creator-resume-client",
							"platforms": map[string]any{
								"zhihu": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"zhihu": "mock-zhihu"},
						},
					},
				},
			})
		case "/api/cookies/zhihu":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "z_c0=mock-zhihu-creator-resume-zc0; q_c1=mock-zhihu-creator-resume-qc1"},
			})
		case "/api/v4/me":
			writeJSON(t, w, map[string]any{"uid": "zh-current-user", "name": "mock zhihu account"})
		case "/people/mock-zh-creator-resume":
			profileCalls++
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(zhihuCreatorResumeHTML()))
		case "/api/v4/members/mock-zh-creator-resume/answers":
			offset := r.URL.Query().Get("offset")
			answerCallsByOffset[offset]++
			switch offset {
			case "0":
				writeJSON(t, w, zhihuCreatorAnswersResumePayload("zh_creator_resume_answer_1", "Zhihu creator resume answer 1", false))
			case "20":
				if !allowSecondOffset {
					http.Error(w, "transient creator offset failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, zhihuCreatorAnswersResumePayload("zh_creator_resume_answer_2", "Zhihu creator resume answer 2", true))
			default:
				t.Fatalf("unexpected zhihu creator offset: %s (%s)", offset, r.URL.RawQuery)
			}
		case "/signsrv/v1/zhihu/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_zst_81": "mock-x-zst-81",
					"x_zse_96": "mock-x-zse-96",
				},
			})
		default:
			t.Fatalf("unexpected zhihu creator resume request: %s %s", r.Method, r.URL.String())
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
platform: "zhihu"
keywords:
  - "unused"
crawlerType: "creator"
saveDataOption: "json"
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
specifiedCreatorIds:
  - "mock-zh-creator-resume"
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
  zhihuUrl: %q
  zhihuZhuanlanUrl: %q
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
			"MCPRO_ZHIHU_URL="+mockServer.URL,
			"MCPRO_ZHIHU_ZHUANLAN_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("zhihu creator resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("zhihu creator resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if profileCalls != 1 || answerCallsByOffset["0"] != 1 || answerCallsByOffset["20"] != 1 {
		t.Fatalf("first zhihu creator run should fetch profile, offset 0, and fail offset 20 once, profileCalls=%d answerCalls=%#v\n%s", profileCalls, answerCallsByOffset, firstOutput)
	}

	allowSecondOffset = true
	secondOutput := runCrawler()
	if profileCalls != 2 {
		t.Fatalf("resume should refresh creator profile to match Python behavior, profileCalls=%d\n%s", profileCalls, secondOutput)
	}
	if answerCallsByOffset["0"] != 1 {
		t.Fatalf("resume should skip completed creator offset 0, calls=%#v\n%s", answerCallsByOffset, secondOutput)
	}
	if answerCallsByOffset["20"] != 2 {
		t.Fatalf("resume should retry creator offset 20 exactly once, calls=%#v\n%s", answerCallsByOffset, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "zhihu", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Zhihu creator rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["content_id"])] = true
	}
	if !seen["zh_creator_resume_answer_1"] || !seen["zh_creator_resume_answer_2"] {
		t.Fatalf("zhihu creator resume output missing expected content IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIZhihuHomefeedJSONWithMockServers(t *testing.T) {
	result := runCLIZhihuHomefeedWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "zhihu", "json", "homefeed_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, result.output)
	if len(rows) != 1 {
		t.Fatalf("expected one zhihu homefeed json row, got %d: %#v", len(rows), rows)
	}
	assertZhihuHomefeedMapRow(t, rows[0], result.zhihuURL)
	if _, ok := rows[0]["add_ts"]; ok {
		t.Fatalf("zhihu homefeed json should match Python and omit add_ts: %#v", rows[0])
	}
	if _, ok := rows[0]["last_modify_ts"]; !ok {
		t.Fatalf("expected zhihu homefeed json last_modify_ts: %#v", rows[0])
	}
}

func TestCLIZhihuHomefeedCSVWithMockServers(t *testing.T) {
	result := runCLIZhihuHomefeedWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "zhihu", "1_homefeed_contents_"+time.Now().Format("2006-01-02")+".csv")
	records := readCSVRecords(t, resultPath, result.output)
	if len(records) != 2 {
		t.Fatalf("expected zhihu homefeed csv header and one row, got %d: %#v", len(records), records)
	}
	assertZhihuHomefeedCSVRow(t, records, result.zhihuURL)
	if csvCell(records, "add_ts") != "" {
		t.Fatalf("zhihu homefeed csv should not include add_ts value: %#v", csvCell(records, "add_ts"))
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected zhihu homefeed csv last_modify_ts")
	}
}

func TestCLIZhihuHomefeedSQLiteWithMockServers(t *testing.T) {
	result := runCLIZhihuHomefeedWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open zhihu homefeed sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []zhihuSQLiteContent
	if err := db.Table("zhihu_content").Find(&rows).Error; err != nil {
		t.Fatalf("query zhihu homefeed sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one zhihu homefeed sqlite row, got %d: %#v", len(rows), rows)
	}
	assertZhihuSQLiteHomefeedContent(t, rows[0], result.zhihuURL)

	var commentCount int64
	if err := db.Table("zhihu_comment").Count(&commentCount).Error; err != nil {
		t.Fatalf("expected zhihu_comment table to match Python sqlite schema: %v\ncommand output:\n%s", err, result.output)
	}
	if commentCount != 0 {
		t.Fatalf("expected no zhihu homefeed sqlite comments when comments disabled, got %d", commentCount)
	}
	var creatorCount int64
	if err := db.Table("zhihu_creator").Count(&creatorCount).Error; err != nil {
		t.Fatalf("expected zhihu_creator table to match Python sqlite schema: %v\ncommand output:\n%s", err, result.output)
	}
	if creatorCount != 0 {
		t.Fatalf("expected no zhihu homefeed sqlite creators during homefeed, got %d", creatorCount)
	}
}

func TestCLIZhihuHomefeedCheckpointResumeByCursorAndPage(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	homefeedCallsByPage := make(map[string]int)
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
							"client_id": "zhihu-homefeed-resume-client",
							"platforms": map[string]any{
								"zhihu": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"zhihu": "mock-zhihu"},
						},
					},
				},
			})
		case "/api/cookies/zhihu":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "z_c0=mock-zhihu-homefeed-resume-zc0; q_c1=mock-zhihu-homefeed-resume-qc1"},
			})
		case "/api/v4/me":
			writeJSON(t, w, map[string]any{"uid": "zh-current-user", "name": "mock zhihu account"})
		case "/api/v3/feed/topstory/recommend":
			pageNumber := r.URL.Query().Get("page_number")
			homefeedCallsByPage[pageNumber]++
			switch pageNumber {
			case "1":
				if r.URL.Query().Get("after_id") != "0" || r.URL.Query().Get("end_offset") != "0" || r.URL.Query().Get("session_token") != "" {
					t.Fatalf("unexpected first zhihu homefeed cursor: %s", r.URL.RawQuery)
				}
				nextURL := "http://" + r.Host + "/api/v3/feed/topstory/recommend?after_id=1&end_offset=20&session_token=mock-homefeed-resume-session&page_number=2"
				writeJSON(t, w, zhihuHomefeedResumePayload("zh_homefeed_resume_answer_1", "Zhihu homefeed resume answer 1", nextURL))
			case "2":
				if r.URL.Query().Get("after_id") != "1" || r.URL.Query().Get("end_offset") != "20" || r.URL.Query().Get("session_token") != "mock-homefeed-resume-session" {
					t.Fatalf("unexpected second zhihu homefeed cursor: %s", r.URL.RawQuery)
				}
				if !allowSecondPage {
					http.Error(w, "transient second homefeed page failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, zhihuHomefeedResumePayload("zh_homefeed_resume_answer_2", "Zhihu homefeed resume answer 2", ""))
			default:
				t.Fatalf("unexpected zhihu homefeed page_number: %s (%s)", pageNumber, r.URL.RawQuery)
			}
		case "/signsrv/v1/zhihu/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_zst_81": "mock-x-zst-81",
					"x_zse_96": "mock-x-zse-96",
				},
			})
		default:
			t.Fatalf("unexpected zhihu homefeed resume request: %s %s", r.Method, r.URL.String())
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
platform: "zhihu"
keywords:
  - "unused"
crawlerType: "homefeed"
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
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  zhihuUrl: %q
  zhihuZhuanlanUrl: %q
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
			"MCPRO_ZHIHU_URL="+mockServer.URL,
			"MCPRO_ZHIHU_ZHUANLAN_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("zhihu homefeed resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("zhihu homefeed resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if homefeedCallsByPage["1"] != 1 || homefeedCallsByPage["2"] != 1 {
		t.Fatalf("first zhihu homefeed run should fetch page 1 and fail page 2 once, calls=%#v\n%s", homefeedCallsByPage, firstOutput)
	}

	allowSecondPage = true
	secondOutput := runCrawler()
	if homefeedCallsByPage["1"] != 1 {
		t.Fatalf("resume should skip completed first homefeed page, calls=%#v\n%s", homefeedCallsByPage, secondOutput)
	}
	if homefeedCallsByPage["2"] != 2 {
		t.Fatalf("resume should retry second homefeed page exactly once, calls=%#v\n%s", homefeedCallsByPage, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "zhihu", "json", "homefeed_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Zhihu homefeed rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["content_id"])] = true
	}
	if !seen["zh_homefeed_resume_answer_1"] || !seen["zh_homefeed_resume_answer_2"] {
		t.Fatalf("zhihu homefeed resume output missing expected content IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIZhihuHotSearchJSONWithMockServers(t *testing.T) {
	result := runCLIZhihuHotSearchWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "zhihu", "json", "hot_search_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, result.output)
	if len(rows) != 1 {
		t.Fatalf("expected one zhihu hot_search json row, got %d: %#v", len(rows), rows)
	}
	assertZhihuHotSearchMapRow(t, rows[0], result.zhihuURL)
	if _, ok := rows[0]["add_ts"]; ok {
		t.Fatalf("zhihu hot_search json should match Python and omit add_ts: %#v", rows[0])
	}
	if _, ok := rows[0]["last_modify_ts"]; !ok {
		t.Fatalf("expected zhihu hot_search json last_modify_ts: %#v", rows[0])
	}
}

func TestCLIZhihuHotSearchCSVWithMockServers(t *testing.T) {
	result := runCLIZhihuHotSearchWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "zhihu", "1_hot_search_contents_"+time.Now().Format("2006-01-02")+".csv")
	records := readCSVRecords(t, resultPath, result.output)
	if len(records) != 2 {
		t.Fatalf("expected zhihu hot_search csv header and one row, got %d: %#v", len(records), records)
	}
	assertZhihuHotSearchCSVRow(t, records, result.zhihuURL)
	if csvCell(records, "add_ts") != "" {
		t.Fatalf("zhihu hot_search csv should not include add_ts value: %#v", csvCell(records, "add_ts"))
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected zhihu hot_search csv last_modify_ts")
	}
}

func TestCLIZhihuHotSearchSQLiteWithMockServers(t *testing.T) {
	result := runCLIZhihuHotSearchWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open zhihu hot_search sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var contents []zhihuSQLiteContent
	if err := db.Table("zhihu_content").Find(&contents).Error; err != nil {
		t.Fatalf("query zhihu hot_search sqlite contents: %v\ncommand output:\n%s", err, result.output)
	}
	if len(contents) != 1 {
		t.Fatalf("expected one zhihu hot_search sqlite content, got %d: %#v", len(contents), contents)
	}
	assertZhihuSQLiteHotSearchContent(t, contents[0], result.zhihuURL)

	var items []zhihuSQLiteHotSearchItem
	if err := db.Table("platform_hot_search_item").Find(&items).Error; err != nil {
		t.Fatalf("query zhihu hot_search sqlite items: %v\ncommand output:\n%s", err, result.output)
	}
	if len(items) != 1 {
		t.Fatalf("expected one zhihu hot_search sqlite item, got %d: %#v", len(items), items)
	}
	assertZhihuSQLiteHotSearchItem(t, items[0])

	var relations []zhihuSQLiteHotSearchRelation
	if err := db.Table("platform_hot_search_content_rel").Find(&relations).Error; err != nil {
		t.Fatalf("query zhihu hot_search sqlite relations: %v\ncommand output:\n%s", err, result.output)
	}
	if len(relations) != 1 {
		t.Fatalf("expected one zhihu hot_search sqlite relation, got %d: %#v", len(relations), relations)
	}
	assertZhihuSQLiteHotSearchRelation(t, relations[0], items[0].HotItemID)
}

func TestCLIZhihuHotSearchCheckpointResumeUsesSnapshotAndSkipsCompletedQuestion(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	hotRankCalls := 0
	feedsCallsByQuestion := make(map[string]int)
	allowSecondQuestion := false
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
							"client_id": "zhihu-hot-resume-client",
							"platforms": map[string]any{
								"zhihu": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"zhihu": "mock-zhihu"},
						},
					},
				},
			})
		case "/api/cookies/zhihu":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "z_c0=hot-resume-zc0; q_c1=hot-resume-qc1"},
			})
		case "/api/v4/me":
			writeJSON(t, w, map[string]any{"uid": "zh-hot-resume-user", "name": "mock zhihu hot resume account"})
		case "/signsrv/v1/zhihu/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_zst_81": "hot-resume-x-zst-81",
					"x_zse_96": "hot-resume-x-zse-96",
				},
			})
		case "/api/v4/creators/rank/hot":
			hotRankCalls++
			writeJSON(t, w, zhihuHotSearchResumePayload())
		case "/api/v4/questions/zh_hot_resume_question_1/feeds":
			feedsCallsByQuestion["zh_hot_resume_question_1"]++
			writeJSON(t, w, zhihuHotQuestionResumeFeedsPayload("zh_hot_resume_answer_1", "zh_hot_resume_question_1", "Zhihu hot resume question 1", 1710200000))
		case "/api/v4/questions/zh_hot_resume_question_2/feeds":
			feedsCallsByQuestion["zh_hot_resume_question_2"]++
			if !allowSecondQuestion {
				http.Error(w, "transient hot question failure", http.StatusInternalServerError)
				return
			}
			writeJSON(t, w, zhihuHotQuestionResumeFeedsPayload("zh_hot_resume_answer_2", "zh_hot_resume_question_2", "Zhihu hot resume question 2", 1710200100))
		default:
			t.Fatalf("unexpected zhihu hot resume request: %s %s", r.Method, r.URL.String())
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
platform: "zhihu"
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
enableGetComments: false
enableGetSubComments: false
enableCheckpoint: true
hotSearchMaxItemsCount: 2
hotSearchPerItemContentCount: 1
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  zhihuUrl: %q
  zhihuZhuanlanUrl: %q
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
			"MCPRO_ZHIHU_URL="+mockServer.URL,
			"MCPRO_ZHIHU_ZHUANLAN_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("zhihu hot resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("zhihu hot resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if hotRankCalls != 1 || feedsCallsByQuestion["zh_hot_resume_question_1"] != 1 || feedsCallsByQuestion["zh_hot_resume_question_2"] != 1 {
		t.Fatalf("first hot_search run should fetch rank, question 1, and fail question 2 once; rank=%d feeds=%#v\n%s", hotRankCalls, feedsCallsByQuestion, firstOutput)
	}

	allowSecondQuestion = true
	secondOutput := runCrawler()
	if hotRankCalls != 1 {
		t.Fatalf("resume should reuse hot_search snapshot and not refetch rank, got %d\n%s", hotRankCalls, secondOutput)
	}
	if feedsCallsByQuestion["zh_hot_resume_question_1"] != 1 {
		t.Fatalf("resume should skip completed hot question 1, feeds=%#v\n%s", feedsCallsByQuestion, secondOutput)
	}
	if feedsCallsByQuestion["zh_hot_resume_question_2"] != 2 {
		t.Fatalf("resume should retry hot question 2 exactly once, feeds=%#v\n%s", feedsCallsByQuestion, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "zhihu", "json", "hot_search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read zhihu hot resume json: %v\n%s", err, secondOutput)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal zhihu hot resume json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Zhihu hot_search rows after resume, got %d: %s", len(rows), body)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["content_id"])] = true
	}
	if !seen["zh_hot_resume_answer_1"] || !seen["zh_hot_resume_answer_2"] {
		t.Fatalf("hot_search resume output missing expected content IDs: seen=%#v rows=%s", seen, body)
	}
}

func TestCLIZhihuHotSearchCheckpointStaleSnapshotStartsFresh(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	hotRankCalls := 0
	staleFeedsCalls := 0
	freshFeedsCalls := 0
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
							"client_id": "zhihu-hot-stale-client",
							"platforms": map[string]any{
								"zhihu": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"zhihu": "mock-zhihu"},
						},
					},
				},
			})
		case "/api/cookies/zhihu":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "z_c0=hot-stale-zc0; q_c1=hot-stale-qc1"},
			})
		case "/api/v4/me":
			writeJSON(t, w, map[string]any{"uid": "zh-hot-stale-user", "name": "mock zhihu hot stale account"})
		case "/signsrv/v1/zhihu/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_zst_81": "hot-stale-x-zst-81",
					"x_zse_96": "hot-stale-x-zse-96",
				},
			})
		case "/api/v4/creators/rank/hot":
			hotRankCalls++
			writeJSON(t, w, zhihuHotSearchPayload())
		case "/api/v4/questions/zh_hot_question_e2e/feeds":
			freshFeedsCalls++
			writeJSON(t, w, zhihuHotQuestionFeedsPayload())
		case "/api/v4/questions/zh_stale_question/feeds":
			staleFeedsCalls++
			http.Error(w, "stale checkpoint should not be used", http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected zhihu hot stale request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	host, port := serverHostPort(t, mockServer.URL)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	checkpointDir := filepath.Join(tmpDir, "data", "checkpoints")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(checkpointDir, 0755); err != nil {
		t.Fatal(err)
	}
	staleItems, err := json.Marshal([]map[string]any{
		{
			"question": map[string]any{"id": "zh_stale_question", "title": "Zhihu stale question"},
			"score":    1,
			"label":    "旧",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	staleCheckpoint := map[string]any{
		"id":            "hot_search_latest",
		"platform":      "zhihu",
		"crawler_type":  "hot_search",
		"current_page":  0,
		"total_crawled": 0,
		"extra": map[string]string{
			"hot_search_snapshot_items": string(staleItems),
			"hot_search_snapshot_ts":    "1",
		},
	}
	checkpointBody, err := json.MarshalIndent(staleCheckpoint, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(checkpointDir, "hot_search_latest.json"), checkpointBody, 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	configYAML := fmt.Sprintf(`
platform: "zhihu"
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
enableCheckpoint: true
hotSearchMaxItemsCount: 1
hotSearchPerItemContentCount: 1
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  zhihuUrl: %q
  zhihuZhuanlanUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL)
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
		"MCPRO_ZHIHU_URL="+mockServer.URL,
		"MCPRO_ZHIHU_ZHUANLAN_URL="+mockServer.URL,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("zhihu hot stale crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("zhihu hot stale crawler command failed: %v\n%s", err, output)
	}
	if hotRankCalls != 1 || staleFeedsCalls != 0 || freshFeedsCalls != 1 {
		t.Fatalf("stale checkpoint should fetch fresh rank and fresh feeds only; rank=%d stale=%d fresh=%d\n%s", hotRankCalls, staleFeedsCalls, freshFeedsCalls, output)
	}

	resultPath := filepath.Join(dataDir, "zhihu", "json", "hot_search_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, output)
	if len(rows) != 1 || rows[0]["content_id"] != "zh_hot_answer_e2e" {
		t.Fatalf("expected fresh hot_search row after stale checkpoint, got %#v", rows)
	}
}

func TestCLIZhihuHotSearchCheckpointResumeSkipsStoredContentIDsWithinQuestion(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	hotRankCalls := 0
	feedsCallsByOffset := make(map[string]int)
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
							"client_id": "zhihu-hot-content-resume-client",
							"platforms": map[string]any{
								"zhihu": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"zhihu": "mock-zhihu"},
						},
					},
				},
			})
		case "/api/cookies/zhihu":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "z_c0=hot-content-resume-zc0; q_c1=hot-content-resume-qc1"},
			})
		case "/api/v4/me":
			writeJSON(t, w, map[string]any{"uid": "zh-hot-content-resume-user", "name": "mock zhihu hot content resume account"})
		case "/signsrv/v1/zhihu/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_zst_81": "hot-content-resume-x-zst-81",
					"x_zse_96": "hot-content-resume-x-zse-96",
				},
			})
		case "/api/v4/creators/rank/hot":
			hotRankCalls++
			writeJSON(t, w, zhihuHotSearchPayload())
		case "/api/v4/questions/zh_hot_question_e2e/feeds":
			offset := r.URL.Query().Get("offset")
			feedsCallsByOffset[offset]++
			switch offset {
			case "0":
				writeJSON(t, w, zhihuHotQuestionPagedFeedsPayload("zh_hot_answer_page_1", "Zhihu hot paged answer 1", false, "http://zhihu.test/api/v4/questions/zh_hot_question_e2e/feeds?cursor=next-hot&limit=5&offset=1&order=default&session_id=hot-session"))
			case "1":
				if !allowSecondPage {
					http.Error(w, "transient hot answer page failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, zhihuHotQuestionPagedFeedsPayload("zh_hot_answer_page_2", "Zhihu hot paged answer 2", true, ""))
			default:
				t.Fatalf("unexpected zhihu hot paged offset: %s (%s)", offset, r.URL.RawQuery)
			}
		default:
			t.Fatalf("unexpected zhihu hot content resume request: %s %s", r.Method, r.URL.String())
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
platform: "zhihu"
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
enableGetComments: false
enableGetSubComments: false
enableCheckpoint: true
hotSearchMaxItemsCount: 1
hotSearchPerItemContentCount: 2
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  zhihuUrl: %q
  zhihuZhuanlanUrl: %q
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
			"MCPRO_ZHIHU_URL="+mockServer.URL,
			"MCPRO_ZHIHU_ZHUANLAN_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("zhihu hot content resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("zhihu hot content resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if hotRankCalls != 1 || feedsCallsByOffset["0"] != 1 || feedsCallsByOffset["1"] != 1 {
		t.Fatalf("first hot_search content resume run should fetch rank and fail page 2 once; rank=%d feeds=%#v\n%s", hotRankCalls, feedsCallsByOffset, firstOutput)
	}

	allowSecondPage = true
	secondOutput := runCrawler()
	if hotRankCalls != 1 || feedsCallsByOffset["0"] != 2 || feedsCallsByOffset["1"] != 2 {
		t.Fatalf("resume should reuse snapshot and retry paged feeds; rank=%d feeds=%#v\n%s", hotRankCalls, feedsCallsByOffset, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "zhihu", "json", "hot_search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read zhihu hot content resume json: %v\n%s", err, secondOutput)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal zhihu hot content resume json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Zhihu hot_search rows after content-id resume, got %d: %s", len(rows), body)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["content_id"])] = true
	}
	if !seen["zh_hot_answer_page_1"] || !seen["zh_hot_answer_page_2"] {
		t.Fatalf("hot_search content-id resume missing expected content IDs: seen=%#v rows=%s", seen, body)
	}
}

type cliZhihuResult struct {
	output      []byte
	dataDir     string
	sqlitePath  string
	zhihuURL    string
	searchCalls int
}

type zhihuSQLiteContent struct {
	ID            int64
	ContentID     string
	ContentType   string
	ContentText   string
	ContentURL    string
	QuestionID    string
	Title         string
	Desc          string `gorm:"column:desc"`
	CreatedTime   string
	UpdatedTime   string
	VoteupCount   int
	CommentCount  int
	SourceKeyword string
	UserID        string
	UserLink      string
	UserNickname  string
	UserAvatar    string
	UserURLToken  string `gorm:"column:user_url_token"`
	AddTs         int64
	LastModifyTs  int64
}

type zhihuSQLiteComment struct {
	ID              int64
	CommentID       string
	ParentCommentID string
	Content         string
	PublishTime     string
	IPLocation      string `gorm:"column:ip_location"`
	SubCommentCount int
	LikeCount       int
	DislikeCount    int
	ContentID       string
	ContentType     string
	UserID          string
	UserLink        string
	UserNickname    string
	UserAvatar      string
	AddTs           int64
	LastModifyTs    int64
}

type zhihuSQLiteCreator struct {
	ID             int64
	UserID         string
	UserLink       string
	UserNickname   string
	UserAvatar     string
	URLToken       string `gorm:"column:url_token"`
	Gender         string
	IPLocation     string `gorm:"column:ip_location"`
	Follows        int
	Fans           int
	AnwserCount    int
	VideoCount     int
	QuestionCount  int
	ArticleCount   int
	ColumnCount    int
	GetVoteupCount int
	AddTs          int64
	LastModifyTs   int64
}

type zhihuSQLiteHotSearchItem struct {
	ID           int64
	HotItemID    string `gorm:"column:hot_item_id"`
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

type zhihuSQLiteHotSearchRelation struct {
	ID            int64
	RelID         string `gorm:"column:rel_id"`
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

func assertZhihuSearchMapRow(t *testing.T, row map[string]any, baseURL string) {
	t.Helper()
	if row["content_id"] != "zh_answer_e2e" || row["content_type"] != "answer" {
		t.Fatalf("unexpected zhihu identity fields: %#v", row)
	}
	if row["content_text"] != "Zhihu answer E2E content" || row["title"] != "Zhihu E2E question" || row["desc"] != "Zhihu answer description" {
		t.Fatalf("unexpected zhihu text/title fields: %#v", row)
	}
	if row["content_url"] != baseURL+"/question/zh_question_e2e/answer/zh_answer_e2e" || row["question_id"] != "zh_question_e2e" {
		t.Fatalf("unexpected zhihu url/question fields: %#v", row)
	}
	if zhihuJSONValue(row["created_time"]) != "1710001500" || zhihuJSONValue(row["updated_time"]) != "1710001600" {
		t.Fatalf("unexpected zhihu time fields: %#v", row)
	}
	if zhihuJSONValue(row["voteup_count"]) != "88" || zhihuJSONValue(row["comment_count"]) != "5" {
		t.Fatalf("unexpected zhihu count fields: %#v", row)
	}
	if row["source_keyword"] != "mock-zhihu" {
		t.Fatalf("unexpected zhihu source keyword: %#v", row)
	}
	if row["user_id"] != "zh-author" || row["user_link"] != baseURL+"/people/mock-zh-author" || row["user_nickname"] != "mock zhihu author" {
		t.Fatalf("unexpected zhihu user fields: %#v", row)
	}
	if row["user_avatar"] != "https://example.test/zhihu-avatar.png" || row["user_url_token"] != "mock-zh-author" {
		t.Fatalf("unexpected zhihu avatar/token fields: %#v", row)
	}
}

func assertZhihuDetailMapRow(t *testing.T, row map[string]any, baseURL string) {
	t.Helper()
	if row["content_id"] != "zh_answer_detail_e2e" || row["content_type"] != "answer" {
		t.Fatalf("unexpected zhihu detail identity fields: %#v", row)
	}
	if row["content_text"] != "Zhihu detail E2E content" || row["title"] != "Zhihu detail question" || row["desc"] != "Zhihu detail description" {
		t.Fatalf("unexpected zhihu detail text/title fields: %#v", row)
	}
	if row["content_url"] != baseURL+"/question/zh_question_detail_e2e/answer/zh_answer_detail_e2e" || row["question_id"] != "zh_question_detail_e2e" {
		t.Fatalf("unexpected zhihu detail url/question fields: %#v", row)
	}
	if zhihuJSONValue(row["created_time"]) != "1710001700" || zhihuJSONValue(row["updated_time"]) != "1710001800" {
		t.Fatalf("unexpected zhihu detail time fields: %#v", row)
	}
	if zhihuJSONValue(row["voteup_count"]) != "188" || zhihuJSONValue(row["comment_count"]) != "15" {
		t.Fatalf("unexpected zhihu detail count fields: %#v", row)
	}
	if row["source_keyword"] != "" {
		t.Fatalf("unexpected zhihu detail source keyword: %#v", row)
	}
	if row["user_id"] != "zh-detail-author" || row["user_link"] != baseURL+"/people/mock-zh-detail-author" || row["user_nickname"] != "mock zhihu detail author" {
		t.Fatalf("unexpected zhihu detail user fields: %#v", row)
	}
	if row["user_avatar"] != "https://example.test/zhihu-detail-avatar.png" || row["user_url_token"] != "mock-zh-detail-author" {
		t.Fatalf("unexpected zhihu detail avatar/token fields: %#v", row)
	}
}

func assertZhihuSearchCSVRow(t *testing.T, records [][]string, baseURL string) {
	t.Helper()
	if csvCell(records, "content_id") != "zh_answer_e2e" || csvCell(records, "content_type") != "answer" {
		t.Fatalf("unexpected zhihu csv identity fields: %#v", records)
	}
	if csvCell(records, "content_text") != "Zhihu answer E2E content" || csvCell(records, "title") != "Zhihu E2E question" || csvCell(records, "desc") != "Zhihu answer description" {
		t.Fatalf("unexpected zhihu csv text/title fields: %#v", records)
	}
	if csvCell(records, "content_url") != baseURL+"/question/zh_question_e2e/answer/zh_answer_e2e" || csvCell(records, "question_id") != "zh_question_e2e" {
		t.Fatalf("unexpected zhihu csv url/question fields: %#v", records)
	}
	if csvCell(records, "created_time") != "1710001500" || csvCell(records, "updated_time") != "1710001600" {
		t.Fatalf("unexpected zhihu csv time fields: %#v", records)
	}
	if csvCell(records, "voteup_count") != "88" || csvCell(records, "comment_count") != "5" {
		t.Fatalf("unexpected zhihu csv count fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "mock-zhihu" {
		t.Fatalf("unexpected zhihu csv source keyword: %#v", records)
	}
	if csvCell(records, "user_id") != "zh-author" || csvCell(records, "user_link") != baseURL+"/people/mock-zh-author" || csvCell(records, "user_nickname") != "mock zhihu author" {
		t.Fatalf("unexpected zhihu csv user fields: %#v", records)
	}
	if csvCell(records, "user_avatar") != "https://example.test/zhihu-avatar.png" || csvCell(records, "user_url_token") != "mock-zh-author" {
		t.Fatalf("unexpected zhihu csv avatar/token fields: %#v", records)
	}
}

func assertZhihuDetailCSVRow(t *testing.T, records [][]string, baseURL string) {
	t.Helper()
	if csvCell(records, "content_id") != "zh_answer_detail_e2e" || csvCell(records, "content_type") != "answer" {
		t.Fatalf("unexpected zhihu detail csv identity fields: %#v", records)
	}
	if csvCell(records, "content_text") != "Zhihu detail E2E content" || csvCell(records, "title") != "Zhihu detail question" || csvCell(records, "desc") != "Zhihu detail description" {
		t.Fatalf("unexpected zhihu detail csv text/title fields: %#v", records)
	}
	if csvCell(records, "content_url") != baseURL+"/question/zh_question_detail_e2e/answer/zh_answer_detail_e2e" || csvCell(records, "question_id") != "zh_question_detail_e2e" {
		t.Fatalf("unexpected zhihu detail csv url/question fields: %#v", records)
	}
	if csvCell(records, "created_time") != "1710001700" || csvCell(records, "updated_time") != "1710001800" {
		t.Fatalf("unexpected zhihu detail csv time fields: %#v", records)
	}
	if csvCell(records, "voteup_count") != "188" || csvCell(records, "comment_count") != "15" {
		t.Fatalf("unexpected zhihu detail csv count fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "" {
		t.Fatalf("unexpected zhihu detail csv source keyword: %#v", records)
	}
	if csvCell(records, "user_id") != "zh-detail-author" || csvCell(records, "user_link") != baseURL+"/people/mock-zh-detail-author" || csvCell(records, "user_nickname") != "mock zhihu detail author" {
		t.Fatalf("unexpected zhihu detail csv user fields: %#v", records)
	}
	if csvCell(records, "user_avatar") != "https://example.test/zhihu-detail-avatar.png" || csvCell(records, "user_url_token") != "mock-zh-detail-author" {
		t.Fatalf("unexpected zhihu detail csv avatar/token fields: %#v", records)
	}
}

func assertZhihuSQLiteContent(t *testing.T, row zhihuSQLiteContent, baseURL string) {
	t.Helper()
	if row.ContentID != "zh_answer_e2e" || row.ContentType != "answer" {
		t.Fatalf("unexpected zhihu sqlite identity fields: %#v", row)
	}
	if row.ContentText != "Zhihu answer E2E content" || row.Title != "Zhihu E2E question" || row.Desc != "Zhihu answer description" {
		t.Fatalf("unexpected zhihu sqlite text/title fields: %#v", row)
	}
	if row.ContentURL != baseURL+"/question/zh_question_e2e/answer/zh_answer_e2e" || row.QuestionID != "zh_question_e2e" {
		t.Fatalf("unexpected zhihu sqlite url/question fields: %#v", row)
	}
	if row.CreatedTime != "1710001500" || row.UpdatedTime != "1710001600" {
		t.Fatalf("unexpected zhihu sqlite time fields: %#v", row)
	}
	if row.VoteupCount != 88 || row.CommentCount != 5 {
		t.Fatalf("unexpected zhihu sqlite count fields: %#v", row)
	}
	if row.SourceKeyword != "mock-zhihu" {
		t.Fatalf("unexpected zhihu sqlite source keyword: %#v", row)
	}
	if row.UserID != "zh-author" || row.UserLink != baseURL+"/people/mock-zh-author" || row.UserNickname != "mock zhihu author" {
		t.Fatalf("unexpected zhihu sqlite user fields: %#v", row)
	}
	if row.UserAvatar != "https://example.test/zhihu-avatar.png" || row.UserURLToken != "mock-zh-author" {
		t.Fatalf("unexpected zhihu sqlite avatar/token fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected zhihu sqlite timestamps: %#v", row)
	}
}

func assertZhihuSQLiteDetailContent(t *testing.T, row zhihuSQLiteContent, baseURL string) {
	t.Helper()
	if row.ContentID != "zh_answer_detail_e2e" || row.ContentType != "answer" {
		t.Fatalf("unexpected zhihu detail sqlite identity fields: %#v", row)
	}
	if row.ContentText != "Zhihu detail E2E content" || row.Title != "Zhihu detail question" || row.Desc != "Zhihu detail description" {
		t.Fatalf("unexpected zhihu detail sqlite text/title fields: %#v", row)
	}
	if row.ContentURL != baseURL+"/question/zh_question_detail_e2e/answer/zh_answer_detail_e2e" || row.QuestionID != "zh_question_detail_e2e" {
		t.Fatalf("unexpected zhihu detail sqlite url/question fields: %#v", row)
	}
	if row.CreatedTime != "1710001700" || row.UpdatedTime != "1710001800" {
		t.Fatalf("unexpected zhihu detail sqlite time fields: %#v", row)
	}
	if row.VoteupCount != 188 || row.CommentCount != 15 {
		t.Fatalf("unexpected zhihu detail sqlite count fields: %#v", row)
	}
	if row.SourceKeyword != "" {
		t.Fatalf("unexpected zhihu detail sqlite source keyword: %#v", row)
	}
	if row.UserID != "zh-detail-author" || row.UserLink != baseURL+"/people/mock-zh-detail-author" || row.UserNickname != "mock zhihu detail author" {
		t.Fatalf("unexpected zhihu detail sqlite user fields: %#v", row)
	}
	if row.UserAvatar != "https://example.test/zhihu-detail-avatar.png" || row.UserURLToken != "mock-zh-detail-author" {
		t.Fatalf("unexpected zhihu detail sqlite avatar/token fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected zhihu detail sqlite timestamps: %#v", row)
	}
}

func assertZhihuCommentMapRow(t *testing.T, row map[string]any, baseURL, commentID, parentID, content, publishTime, ipLocation, subCount, likeCount, dislikeCount, userID, nickname, avatar, urlToken string) {
	t.Helper()
	if row["comment_id"] != commentID || row["parent_comment_id"] != parentID || row["content"] != content {
		t.Fatalf("unexpected zhihu comment identity/content fields: %#v", row)
	}
	if zhihuJSONValue(row["publish_time"]) != publishTime || row["ip_location"] != ipLocation {
		t.Fatalf("unexpected zhihu comment time/location fields: %#v", row)
	}
	if zhihuJSONValue(row["sub_comment_count"]) != subCount || zhihuJSONValue(row["like_count"]) != likeCount || zhihuJSONValue(row["dislike_count"]) != dislikeCount {
		t.Fatalf("unexpected zhihu comment count fields: %#v", row)
	}
	if row["content_id"] != "zh_answer_detail_e2e" || row["content_type"] != "answer" {
		t.Fatalf("unexpected zhihu comment content refs: %#v", row)
	}
	if row["user_id"] != userID || row["user_link"] != baseURL+"/people/"+urlToken || row["user_nickname"] != nickname || row["user_avatar"] != avatar {
		t.Fatalf("unexpected zhihu comment user fields: %#v", row)
	}
	if _, ok := row["add_ts"]; ok {
		t.Fatalf("zhihu comment json should match Python and omit add_ts: %#v", row)
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected zhihu comment json last_modify_ts: %#v", row)
	}
}

func assertZhihuCommentCSVRow(t *testing.T, records [][]string, row int, baseURL, commentID, parentID, content, publishTime, ipLocation, subCount, likeCount, dislikeCount, userID, nickname, avatar, urlToken string) {
	t.Helper()
	if csvCellAt(records, row, "comment_id") != commentID || csvCellAt(records, row, "parent_comment_id") != parentID || csvCellAt(records, row, "content") != content {
		t.Fatalf("unexpected zhihu comment csv identity/content fields: %#v", records[row])
	}
	if csvCellAt(records, row, "publish_time") != publishTime || csvCellAt(records, row, "ip_location") != ipLocation {
		t.Fatalf("unexpected zhihu comment csv time/location fields: %#v", records[row])
	}
	if csvCellAt(records, row, "sub_comment_count") != subCount || csvCellAt(records, row, "like_count") != likeCount || csvCellAt(records, row, "dislike_count") != dislikeCount {
		t.Fatalf("unexpected zhihu comment csv count fields: %#v", records[row])
	}
	if csvCellAt(records, row, "content_id") != "zh_answer_detail_e2e" || csvCellAt(records, row, "content_type") != "answer" {
		t.Fatalf("unexpected zhihu comment csv content refs: %#v", records[row])
	}
	if csvCellAt(records, row, "user_id") != userID || csvCellAt(records, row, "user_link") != baseURL+"/people/"+urlToken || csvCellAt(records, row, "user_nickname") != nickname || csvCellAt(records, row, "user_avatar") != avatar {
		t.Fatalf("unexpected zhihu comment csv user fields: %#v", records[row])
	}
}

func assertZhihuSQLiteComment(t *testing.T, row zhihuSQLiteComment, baseURL, commentID, parentID, content, publishTime, ipLocation string, subCount, likeCount, dislikeCount int, userID, nickname, avatar, urlToken string) {
	t.Helper()
	if row.CommentID != commentID || row.ParentCommentID != parentID || row.Content != content {
		t.Fatalf("unexpected zhihu sqlite comment identity/content fields: %#v", row)
	}
	if row.PublishTime != publishTime || row.IPLocation != ipLocation {
		t.Fatalf("unexpected zhihu sqlite comment time/location fields: %#v", row)
	}
	if row.SubCommentCount != subCount || row.LikeCount != likeCount || row.DislikeCount != dislikeCount {
		t.Fatalf("unexpected zhihu sqlite comment count fields: %#v", row)
	}
	if row.ContentID != "zh_answer_detail_e2e" || row.ContentType != "answer" {
		t.Fatalf("unexpected zhihu sqlite comment content refs: %#v", row)
	}
	if row.UserID != userID || row.UserLink != baseURL+"/people/"+urlToken || row.UserNickname != nickname || row.UserAvatar != avatar {
		t.Fatalf("unexpected zhihu sqlite comment user fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected zhihu sqlite comment timestamps: %#v", row)
	}
}

func assertZhihuCreatorMapRow(t *testing.T, row map[string]any, baseURL string) {
	t.Helper()
	if row["user_id"] != "zh-creator-user" || row["user_link"] != baseURL+"/people/mock-zh-creator" || row["user_nickname"] != "mock zhihu creator" {
		t.Fatalf("unexpected zhihu creator identity fields: %#v", row)
	}
	if row["user_avatar"] != "https://example.test/zhihu-creator-avatar.png" || row["url_token"] != "mock-zh-creator" || row["gender"] != "男" || row["ip_location"] != "浙江" {
		t.Fatalf("unexpected zhihu creator profile fields: %#v", row)
	}
	if zhihuJSONValue(row["follows"]) != "12" || zhihuJSONValue(row["fans"]) != "345" || zhihuJSONValue(row["anwser_count"]) != "7" || zhihuJSONValue(row["video_count"]) != "2" || zhihuJSONValue(row["question_count"]) != "3" || zhihuJSONValue(row["article_count"]) != "4" || zhihuJSONValue(row["column_count"]) != "1" || zhihuJSONValue(row["get_voteup_count"]) != "9876" {
		t.Fatalf("unexpected zhihu creator count fields: %#v", row)
	}
	if _, ok := row["add_ts"]; ok {
		t.Fatalf("zhihu creator json should match Python and omit add_ts: %#v", row)
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected zhihu creator json last_modify_ts: %#v", row)
	}
}

func assertZhihuCreatorCSVRow(t *testing.T, records [][]string, baseURL string) {
	t.Helper()
	if csvCell(records, "user_id") != "zh-creator-user" || csvCell(records, "user_link") != baseURL+"/people/mock-zh-creator" || csvCell(records, "user_nickname") != "mock zhihu creator" {
		t.Fatalf("unexpected zhihu creator csv identity fields: %#v", records)
	}
	if csvCell(records, "user_avatar") != "https://example.test/zhihu-creator-avatar.png" || csvCell(records, "url_token") != "mock-zh-creator" || csvCell(records, "gender") != "男" || csvCell(records, "ip_location") != "浙江" {
		t.Fatalf("unexpected zhihu creator csv profile fields: %#v", records)
	}
	if csvCell(records, "follows") != "12" || csvCell(records, "fans") != "345" || csvCell(records, "anwser_count") != "7" || csvCell(records, "video_count") != "2" || csvCell(records, "question_count") != "3" || csvCell(records, "article_count") != "4" || csvCell(records, "column_count") != "1" || csvCell(records, "get_voteup_count") != "9876" {
		t.Fatalf("unexpected zhihu creator csv count fields: %#v", records)
	}
	if csvCell(records, "add_ts") != "" {
		t.Fatalf("zhihu creator csv should not include add_ts value: %#v", csvCell(records, "add_ts"))
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected zhihu creator csv last_modify_ts")
	}
}

func assertZhihuSQLiteCreator(t *testing.T, row zhihuSQLiteCreator, baseURL string) {
	t.Helper()
	if row.UserID != "zh-creator-user" || row.UserLink != baseURL+"/people/mock-zh-creator" || row.UserNickname != "mock zhihu creator" {
		t.Fatalf("unexpected zhihu sqlite creator identity fields: %#v", row)
	}
	if row.UserAvatar != "https://example.test/zhihu-creator-avatar.png" || row.URLToken != "mock-zh-creator" || row.Gender != "男" || row.IPLocation != "浙江" {
		t.Fatalf("unexpected zhihu sqlite creator profile fields: %#v", row)
	}
	if row.Follows != 12 || row.Fans != 345 || row.AnwserCount != 7 || row.VideoCount != 2 || row.QuestionCount != 3 || row.ArticleCount != 4 || row.ColumnCount != 1 || row.GetVoteupCount != 9876 {
		t.Fatalf("unexpected zhihu sqlite creator count fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected zhihu sqlite creator timestamps: %#v", row)
	}
}

func assertZhihuCreatorContentMapRow(t *testing.T, row map[string]any, baseURL string) {
	t.Helper()
	if row["content_id"] != "zh_creator_answer_e2e" || row["content_type"] != "answer" {
		t.Fatalf("unexpected zhihu creator content identity fields: %#v", row)
	}
	if row["content_text"] != "Zhihu creator answer E2E content" || row["title"] != "Zhihu creator question" || row["desc"] != "Zhihu creator answer description" {
		t.Fatalf("unexpected zhihu creator content text/title fields: %#v", row)
	}
	if row["content_url"] != baseURL+"/question/zh_creator_question_e2e/answer/zh_creator_answer_e2e" || row["question_id"] != "zh_creator_question_e2e" {
		t.Fatalf("unexpected zhihu creator content url/question fields: %#v", row)
	}
	if zhihuJSONValue(row["created_time"]) != "1710002000" || zhihuJSONValue(row["updated_time"]) != "1710002100" || zhihuJSONValue(row["voteup_count"]) != "288" || zhihuJSONValue(row["comment_count"]) != "25" {
		t.Fatalf("unexpected zhihu creator content numeric fields: %#v", row)
	}
	if row["source_keyword"] != "" || row["user_id"] != "zh-creator-user" || row["user_link"] != baseURL+"/people/mock-zh-creator" || row["user_nickname"] != "mock zhihu creator" || row["user_avatar"] != "https://example.test/zhihu-creator-avatar.png" || row["user_url_token"] != "mock-zh-creator" {
		t.Fatalf("unexpected zhihu creator content user/source fields: %#v", row)
	}
}

func assertZhihuCreatorContentCSVRow(t *testing.T, records [][]string, baseURL string) {
	t.Helper()
	if csvCell(records, "content_id") != "zh_creator_answer_e2e" || csvCell(records, "content_type") != "answer" {
		t.Fatalf("unexpected zhihu creator content csv identity fields: %#v", records)
	}
	if csvCell(records, "content_text") != "Zhihu creator answer E2E content" || csvCell(records, "title") != "Zhihu creator question" || csvCell(records, "desc") != "Zhihu creator answer description" {
		t.Fatalf("unexpected zhihu creator content csv text/title fields: %#v", records)
	}
	if csvCell(records, "content_url") != baseURL+"/question/zh_creator_question_e2e/answer/zh_creator_answer_e2e" || csvCell(records, "question_id") != "zh_creator_question_e2e" {
		t.Fatalf("unexpected zhihu creator content csv url/question fields: %#v", records)
	}
	if csvCell(records, "created_time") != "1710002000" || csvCell(records, "updated_time") != "1710002100" || csvCell(records, "voteup_count") != "288" || csvCell(records, "comment_count") != "25" {
		t.Fatalf("unexpected zhihu creator content csv numeric fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "" || csvCell(records, "user_id") != "zh-creator-user" || csvCell(records, "user_link") != baseURL+"/people/mock-zh-creator" || csvCell(records, "user_nickname") != "mock zhihu creator" || csvCell(records, "user_avatar") != "https://example.test/zhihu-creator-avatar.png" || csvCell(records, "user_url_token") != "mock-zh-creator" {
		t.Fatalf("unexpected zhihu creator content csv user/source fields: %#v", records)
	}
}

func assertZhihuSQLiteCreatorContent(t *testing.T, row zhihuSQLiteContent, baseURL string) {
	t.Helper()
	if row.ContentID != "zh_creator_answer_e2e" || row.ContentType != "answer" || row.ContentText != "Zhihu creator answer E2E content" || row.Title != "Zhihu creator question" || row.Desc != "Zhihu creator answer description" {
		t.Fatalf("unexpected zhihu sqlite creator content identity/text fields: %#v", row)
	}
	if row.ContentURL != baseURL+"/question/zh_creator_question_e2e/answer/zh_creator_answer_e2e" || row.QuestionID != "zh_creator_question_e2e" {
		t.Fatalf("unexpected zhihu sqlite creator content url/question fields: %#v", row)
	}
	if row.CreatedTime != "1710002000" || row.UpdatedTime != "1710002100" || row.VoteupCount != 288 || row.CommentCount != 25 {
		t.Fatalf("unexpected zhihu sqlite creator content numeric fields: %#v", row)
	}
	if row.SourceKeyword != "" || row.UserID != "zh-creator-user" || row.UserLink != baseURL+"/people/mock-zh-creator" || row.UserNickname != "mock zhihu creator" || row.UserAvatar != "https://example.test/zhihu-creator-avatar.png" || row.UserURLToken != "mock-zh-creator" {
		t.Fatalf("unexpected zhihu sqlite creator content user/source fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected zhihu sqlite creator content timestamps: %#v", row)
	}
}

func assertZhihuHomefeedMapRow(t *testing.T, row map[string]any, baseURL string) {
	t.Helper()
	if row["content_id"] != "zh_homefeed_answer_e2e" || row["content_type"] != "answer" {
		t.Fatalf("unexpected zhihu homefeed identity fields: %#v", row)
	}
	if row["content_text"] != "Zhihu homefeed answer E2E content" || row["title"] != "Zhihu homefeed question" || row["desc"] != "Zhihu homefeed answer description" {
		t.Fatalf("unexpected zhihu homefeed text/title fields: %#v", row)
	}
	if row["content_url"] != baseURL+"/question/zh_homefeed_question_e2e/answer/zh_homefeed_answer_e2e" || row["question_id"] != "zh_homefeed_question_e2e" {
		t.Fatalf("unexpected zhihu homefeed url/question fields: %#v", row)
	}
	if zhihuJSONValue(row["created_time"]) != "1710002200" || zhihuJSONValue(row["updated_time"]) != "1710002300" || zhihuJSONValue(row["voteup_count"]) != "388" || zhihuJSONValue(row["comment_count"]) != "35" {
		t.Fatalf("unexpected zhihu homefeed numeric fields: %#v", row)
	}
	if row["source_keyword"] != "" || row["user_id"] != "zh-homefeed-author" || row["user_link"] != baseURL+"/people/mock-zh-homefeed-author" || row["user_nickname"] != "mock zhihu homefeed author" || row["user_avatar"] != "https://example.test/zhihu-homefeed-avatar.png" || row["user_url_token"] != "mock-zh-homefeed-author" {
		t.Fatalf("unexpected zhihu homefeed user/source fields: %#v", row)
	}
}

func assertZhihuHomefeedCSVRow(t *testing.T, records [][]string, baseURL string) {
	t.Helper()
	if csvCell(records, "content_id") != "zh_homefeed_answer_e2e" || csvCell(records, "content_type") != "answer" {
		t.Fatalf("unexpected zhihu homefeed csv identity fields: %#v", records)
	}
	if csvCell(records, "content_text") != "Zhihu homefeed answer E2E content" || csvCell(records, "title") != "Zhihu homefeed question" || csvCell(records, "desc") != "Zhihu homefeed answer description" {
		t.Fatalf("unexpected zhihu homefeed csv text/title fields: %#v", records)
	}
	if csvCell(records, "content_url") != baseURL+"/question/zh_homefeed_question_e2e/answer/zh_homefeed_answer_e2e" || csvCell(records, "question_id") != "zh_homefeed_question_e2e" {
		t.Fatalf("unexpected zhihu homefeed csv url/question fields: %#v", records)
	}
	if csvCell(records, "created_time") != "1710002200" || csvCell(records, "updated_time") != "1710002300" || csvCell(records, "voteup_count") != "388" || csvCell(records, "comment_count") != "35" {
		t.Fatalf("unexpected zhihu homefeed csv numeric fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "" || csvCell(records, "user_id") != "zh-homefeed-author" || csvCell(records, "user_link") != baseURL+"/people/mock-zh-homefeed-author" || csvCell(records, "user_nickname") != "mock zhihu homefeed author" || csvCell(records, "user_avatar") != "https://example.test/zhihu-homefeed-avatar.png" || csvCell(records, "user_url_token") != "mock-zh-homefeed-author" {
		t.Fatalf("unexpected zhihu homefeed csv user/source fields: %#v", records)
	}
}

func assertZhihuSQLiteHomefeedContent(t *testing.T, row zhihuSQLiteContent, baseURL string) {
	t.Helper()
	if row.ContentID != "zh_homefeed_answer_e2e" || row.ContentType != "answer" || row.ContentText != "Zhihu homefeed answer E2E content" || row.Title != "Zhihu homefeed question" || row.Desc != "Zhihu homefeed answer description" {
		t.Fatalf("unexpected zhihu sqlite homefeed content identity/text fields: %#v", row)
	}
	if row.ContentURL != baseURL+"/question/zh_homefeed_question_e2e/answer/zh_homefeed_answer_e2e" || row.QuestionID != "zh_homefeed_question_e2e" {
		t.Fatalf("unexpected zhihu sqlite homefeed content url/question fields: %#v", row)
	}
	if row.CreatedTime != "1710002200" || row.UpdatedTime != "1710002300" || row.VoteupCount != 388 || row.CommentCount != 35 {
		t.Fatalf("unexpected zhihu sqlite homefeed content numeric fields: %#v", row)
	}
	if row.SourceKeyword != "" || row.UserID != "zh-homefeed-author" || row.UserLink != baseURL+"/people/mock-zh-homefeed-author" || row.UserNickname != "mock zhihu homefeed author" || row.UserAvatar != "https://example.test/zhihu-homefeed-avatar.png" || row.UserURLToken != "mock-zh-homefeed-author" {
		t.Fatalf("unexpected zhihu sqlite homefeed content user/source fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected zhihu sqlite homefeed content timestamps: %#v", row)
	}
}

func assertZhihuHotSearchMapRow(t *testing.T, row map[string]any, baseURL string) {
	t.Helper()
	if row["content_id"] != "zh_hot_answer_e2e" || row["content_type"] != "answer" {
		t.Fatalf("unexpected zhihu hot_search identity fields: %#v", row)
	}
	if row["content_text"] != "Zhihu hot answer E2E content" || row["title"] != "Zhihu hot question" || row["desc"] != "Zhihu hot answer description" {
		t.Fatalf("unexpected zhihu hot_search text/title fields: %#v", row)
	}
	if row["content_url"] != baseURL+"/question/zh_hot_question_e2e/answer/zh_hot_answer_e2e" || row["question_id"] != "zh_hot_question_e2e" {
		t.Fatalf("unexpected zhihu hot_search url/question fields: %#v", row)
	}
	if zhihuJSONValue(row["created_time"]) != "1710002400" || zhihuJSONValue(row["updated_time"]) != "1710002500" || zhihuJSONValue(row["voteup_count"]) != "488" || zhihuJSONValue(row["comment_count"]) != "45" {
		t.Fatalf("unexpected zhihu hot_search numeric fields: %#v", row)
	}
	if row["source_keyword"] != "Zhihu hot question" || row["user_id"] != "zh-hot-author" || row["user_link"] != baseURL+"/people/mock-zh-hot-author" || row["user_nickname"] != "mock zhihu hot author" || row["user_avatar"] != "https://example.test/zhihu-hot-avatar.png" || row["user_url_token"] != "mock-zh-hot-author" {
		t.Fatalf("unexpected zhihu hot_search user/source fields: %#v", row)
	}
}

func assertZhihuHotSearchCSVRow(t *testing.T, records [][]string, baseURL string) {
	t.Helper()
	if csvCell(records, "content_id") != "zh_hot_answer_e2e" || csvCell(records, "content_type") != "answer" {
		t.Fatalf("unexpected zhihu hot_search csv identity fields: %#v", records)
	}
	if csvCell(records, "content_text") != "Zhihu hot answer E2E content" || csvCell(records, "title") != "Zhihu hot question" || csvCell(records, "desc") != "Zhihu hot answer description" {
		t.Fatalf("unexpected zhihu hot_search csv text/title fields: %#v", records)
	}
	if csvCell(records, "content_url") != baseURL+"/question/zh_hot_question_e2e/answer/zh_hot_answer_e2e" || csvCell(records, "question_id") != "zh_hot_question_e2e" {
		t.Fatalf("unexpected zhihu hot_search csv url/question fields: %#v", records)
	}
	if csvCell(records, "created_time") != "1710002400" || csvCell(records, "updated_time") != "1710002500" || csvCell(records, "voteup_count") != "488" || csvCell(records, "comment_count") != "45" {
		t.Fatalf("unexpected zhihu hot_search csv numeric fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "Zhihu hot question" || csvCell(records, "user_id") != "zh-hot-author" || csvCell(records, "user_link") != baseURL+"/people/mock-zh-hot-author" || csvCell(records, "user_nickname") != "mock zhihu hot author" || csvCell(records, "user_avatar") != "https://example.test/zhihu-hot-avatar.png" || csvCell(records, "user_url_token") != "mock-zh-hot-author" {
		t.Fatalf("unexpected zhihu hot_search csv user/source fields: %#v", records)
	}
}

func assertZhihuSQLiteHotSearchContent(t *testing.T, row zhihuSQLiteContent, baseURL string) {
	t.Helper()
	if row.ContentID != "zh_hot_answer_e2e" || row.ContentType != "answer" || row.ContentText != "Zhihu hot answer E2E content" || row.Title != "Zhihu hot question" || row.Desc != "Zhihu hot answer description" {
		t.Fatalf("unexpected zhihu sqlite hot_search content identity/text fields: %#v", row)
	}
	if row.ContentURL != baseURL+"/question/zh_hot_question_e2e/answer/zh_hot_answer_e2e" || row.QuestionID != "zh_hot_question_e2e" {
		t.Fatalf("unexpected zhihu sqlite hot_search content url/question fields: %#v", row)
	}
	if row.CreatedTime != "1710002400" || row.UpdatedTime != "1710002500" || row.VoteupCount != 488 || row.CommentCount != 45 {
		t.Fatalf("unexpected zhihu sqlite hot_search content numeric fields: %#v", row)
	}
	if row.SourceKeyword != "Zhihu hot question" || row.UserID != "zh-hot-author" || row.UserLink != baseURL+"/people/mock-zh-hot-author" || row.UserNickname != "mock zhihu hot author" || row.UserAvatar != "https://example.test/zhihu-hot-avatar.png" || row.UserURLToken != "mock-zh-hot-author" {
		t.Fatalf("unexpected zhihu sqlite hot_search content user/source fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected zhihu sqlite hot_search content timestamps: %#v", row)
	}
}

func assertZhihuSQLiteHotSearchItem(t *testing.T, row zhihuSQLiteHotSearchItem) {
	t.Helper()
	if row.HotItemID == "" || row.Platform != "zhihu" || row.BoardName != "hot_search" || row.BoardType != "question" {
		t.Fatalf("unexpected zhihu hot_search sqlite item identity fields: %#v", row)
	}
	if row.RankNo != 1 || row.Title != "Zhihu hot question" || row.Keyword != "Zhihu hot question" || row.HotValue != "12345" || row.Badge != "热" || row.TopicID != "zh_hot_question_e2e" || row.TopicURL != "https://www.zhihu.com/question/zh_hot_question_e2e" {
		t.Fatalf("unexpected zhihu hot_search sqlite item fields: %#v", row)
	}
	if row.RawJSON == "" || !strings.Contains(row.RawJSON, "zh_hot_question_e2e") || row.AddTs == 0 || row.LastModifyTs == 0 || row.SnapshotTs == 0 || row.SnapshotDate == "" {
		t.Fatalf("expected zhihu hot_search sqlite item metadata: %#v", row)
	}
}

func assertZhihuSQLiteHotSearchRelation(t *testing.T, row zhihuSQLiteHotSearchRelation, hotItemID string) {
	t.Helper()
	if row.RelID == "" || row.HotItemID != hotItemID || row.Platform != "zhihu" {
		t.Fatalf("unexpected zhihu hot_search sqlite relation identity fields: %#v", row)
	}
	if row.ContentID != "zh_hot_answer_e2e" || row.ContentType != "answer" || row.ContentTable != "zhihu_content" || row.RankNo != 1 || row.SourceKeyword != "Zhihu hot question" {
		t.Fatalf("unexpected zhihu hot_search sqlite relation fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected zhihu hot_search sqlite relation timestamps: %#v", row)
	}
}

func zhihuJSONValue(value any) string {
	switch typed := value.(type) {
	case float64:
		return fmt.Sprintf("%.0f", typed)
	default:
		return fmt.Sprint(typed)
	}
}

func runCLIZhihuSearchWithMockServers(t *testing.T, saveDataOption string) cliZhihuResult {
	t.Helper()

	repoRoot := e2e.RepoRoot(t)
	searchCalls := 0
	mockServer := newZhihuMockServer(t, &searchCalls)
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
	configYAML := fmt.Sprintf(`
platform: "zhihu"
keywords:
  - "mock-zhihu"
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
enableGetComments: false
enableGetSubComments: false
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
  zhihuUrl: %q
  zhihuZhuanlanUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, host, port, mockServer.URL, mockServer.URL)
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
		"MCPRO_ZHIHU_URL="+mockServer.URL,
		"MCPRO_ZHIHU_ZHUANLAN_URL="+mockServer.URL,
	)
	if saveDataOption == "sqlite" {
		runCmd.Env = append(runCmd.Env, "MCPRO_SQLITE_DB_PATH="+sqlitePath)
	}
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("zhihu crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("zhihu crawler command failed: %v\n%s", err, output)
	}
	if searchCalls != 2 {
		t.Fatalf("expected zhihu search endpoint to be called for page 1 and empty page 2, got %d\n%s", searchCalls, output)
	}
	return cliZhihuResult{output: output, dataDir: dataDir, sqlitePath: sqlitePath, zhihuURL: mockServer.URL, searchCalls: searchCalls}
}

func runCLIZhihuDetailWithMockServers(t *testing.T, saveDataOption string) cliZhihuResult {
	return runCLIZhihuDetailWithMockServersAndComments(t, saveDataOption, false, false)
}

func runCLIZhihuDetailCommentsWithMockServers(t *testing.T, saveDataOption string) cliZhihuResult {
	return runCLIZhihuDetailWithMockServersAndComments(t, saveDataOption, true, true)
}

func runCLIZhihuDetailWithMockServersAndComments(t *testing.T, saveDataOption string, enableComments, enableSubComments bool) cliZhihuResult {
	t.Helper()

	repoRoot := e2e.RepoRoot(t)
	searchCalls := 0
	mockServer := newZhihuMockServer(t, &searchCalls)
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
	detailURL := mockServer.URL + "/question/zh_question_detail_e2e/answer/zh_answer_detail_e2e"
	configYAML := fmt.Sprintf(`
platform: "zhihu"
keywords:
  - "unused"
crawlerType: "detail"
saveDataOption: %q
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
sqliteDbPath: %q
specifiedNoteIds: [%q]
specifiedNoteURLs: [%q]
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
  zhihuUrl: %q
  zhihuZhuanlanUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, detailURL, detailURL, enableComments, enableSubComments, host, port, mockServer.URL, mockServer.URL)
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
		"MCPRO_ZHIHU_URL="+mockServer.URL,
		"MCPRO_ZHIHU_ZHUANLAN_URL="+mockServer.URL,
	)
	if saveDataOption == "sqlite" {
		runCmd.Env = append(runCmd.Env, "MCPRO_SQLITE_DB_PATH="+sqlitePath)
	}
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("zhihu detail crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("zhihu detail crawler command failed: %v\n%s", err, output)
	}
	if searchCalls != 0 {
		t.Fatalf("zhihu detail should not call search endpoint, got %d\n%s", searchCalls, output)
	}
	return cliZhihuResult{output: output, dataDir: dataDir, sqlitePath: sqlitePath, zhihuURL: mockServer.URL, searchCalls: searchCalls}
}

func runCLIZhihuCreatorWithMockServers(t *testing.T, saveDataOption string) cliZhihuResult {
	t.Helper()

	repoRoot := e2e.RepoRoot(t)
	searchCalls := 0
	mockServer := newZhihuMockServer(t, &searchCalls)
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
	creatorURL := mockServer.URL + "/people/mock-zh-creator"
	configYAML := fmt.Sprintf(`
platform: "zhihu"
keywords:
  - "unused"
crawlerType: "creator"
saveDataOption: %q
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
sqliteDbPath: %q
specifiedCreatorIds: [%q]
specifiedCreatorURLs: [%q]
startPage: 1
crawlerMaxNotesCount: 1
maxConcurrencyNum: 1
requestTimeOut: 2
enableGetComments: false
enableGetSubComments: false
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
  zhihuUrl: %q
  zhihuZhuanlanUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, creatorURL, creatorURL, host, port, mockServer.URL, mockServer.URL)
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
		"MCPRO_ZHIHU_URL="+mockServer.URL,
		"MCPRO_ZHIHU_ZHUANLAN_URL="+mockServer.URL,
	)
	if saveDataOption == "sqlite" {
		runCmd.Env = append(runCmd.Env, "MCPRO_SQLITE_DB_PATH="+sqlitePath)
	}
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("zhihu creator crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("zhihu creator crawler command failed: %v\n%s", err, output)
	}
	if searchCalls != 0 {
		t.Fatalf("zhihu creator should not call search endpoint, got %d\n%s", searchCalls, output)
	}
	return cliZhihuResult{output: output, dataDir: dataDir, sqlitePath: sqlitePath, zhihuURL: mockServer.URL, searchCalls: searchCalls}
}

func runCLIZhihuHomefeedWithMockServers(t *testing.T, saveDataOption string) cliZhihuResult {
	t.Helper()

	repoRoot := e2e.RepoRoot(t)
	searchCalls := 0
	homefeedCalls := 0
	mockServer := newZhihuMockServer(t, &searchCalls, &homefeedCalls)
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
	configYAML := fmt.Sprintf(`
platform: "zhihu"
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
  zhihuUrl: %q
  zhihuZhuanlanUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, host, port, mockServer.URL, mockServer.URL)
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
		"MCPRO_ZHIHU_URL="+mockServer.URL,
		"MCPRO_ZHIHU_ZHUANLAN_URL="+mockServer.URL,
	)
	if saveDataOption == "sqlite" {
		runCmd.Env = append(runCmd.Env, "MCPRO_SQLITE_DB_PATH="+sqlitePath)
	}
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("zhihu homefeed crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("zhihu homefeed crawler command failed: %v\n%s", err, output)
	}
	if searchCalls != 0 {
		t.Fatalf("zhihu homefeed should not call search endpoint, got %d\n%s", searchCalls, output)
	}
	if homefeedCalls != 2 {
		t.Fatalf("expected zhihu homefeed endpoint to be called for page 1 and is_end page 2, got %d\n%s", homefeedCalls, output)
	}
	return cliZhihuResult{output: output, dataDir: dataDir, sqlitePath: sqlitePath, zhihuURL: mockServer.URL, searchCalls: searchCalls}
}

func runCLIZhihuHotSearchWithMockServers(t *testing.T, saveDataOption string) cliZhihuResult {
	t.Helper()

	repoRoot := e2e.RepoRoot(t)
	searchCalls := 0
	hotRankCalls := 0
	hotFeedsCalls := 0
	mockServer := newZhihuMockServer(t, &searchCalls, nil, &hotRankCalls, &hotFeedsCalls)
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
	configYAML := fmt.Sprintf(`
platform: "zhihu"
keywords:
  - "unused"
crawlerType: "hot_search"
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
hotSearchMaxItemsCount: 1
hotSearchPerItemContentCount: 1
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  zhihuUrl: %q
  zhihuZhuanlanUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, host, port, mockServer.URL, mockServer.URL)
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
		"MCPRO_ZHIHU_URL="+mockServer.URL,
		"MCPRO_ZHIHU_ZHUANLAN_URL="+mockServer.URL,
	)
	if saveDataOption == "sqlite" {
		runCmd.Env = append(runCmd.Env, "MCPRO_SQLITE_DB_PATH="+sqlitePath)
	}
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("zhihu hot_search crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("zhihu hot_search crawler command failed: %v\n%s", err, output)
	}
	if searchCalls != 0 {
		t.Fatalf("zhihu hot_search should not call search endpoint, got %d\n%s", searchCalls, output)
	}
	if hotRankCalls != 1 {
		t.Fatalf("expected zhihu hot_search rank endpoint once, got %d\n%s", hotRankCalls, output)
	}
	if hotFeedsCalls != 1 {
		t.Fatalf("expected zhihu hot_search question feeds endpoint once, got %d\n%s", hotFeedsCalls, output)
	}
	return cliZhihuResult{output: output, dataDir: dataDir, sqlitePath: sqlitePath, zhihuURL: mockServer.URL, searchCalls: searchCalls}
}

func newZhihuMockServer(t *testing.T, searchCalls *int, homefeedCalls ...*int) *httptest.Server {
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
							"client_id": "zhihu-client",
							"platforms": map[string]any{
								"zhihu": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"zhihu": "mock-zhihu"},
						},
					},
				},
			})
		case "/api/cookies/zhihu":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "z_c0=mock-zhihu-zc0; q_c1=mock-zhihu-qc1"},
			})
		case "/api/v4/me":
			writeJSON(t, w, map[string]any{"uid": "zh-current-user", "name": "mock zhihu account"})
		case "/question/zh_question_detail_e2e/answer/zh_answer_detail_e2e":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(zhihuDetailHTML()))
		case "/people/mock-zh-creator":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(zhihuCreatorHTML()))
		case "/api/v4/members/mock-zh-creator/answers":
			writeJSON(t, w, zhihuCreatorAnswersPayload())
		case "/api/v3/feed/topstory/recommend":
			if len(homefeedCalls) > 0 && homefeedCalls[0] != nil {
				(*homefeedCalls[0])++
			}
			pageNumber := r.URL.Query().Get("page_number")
			switch pageNumber {
			case "1":
				writeJSON(t, w, zhihuHomefeedPayload("http://"+r.Host))
			case "2":
				writeJSON(t, w, map[string]any{"paging": map[string]any{"is_end": true, "next": ""}, "data": []map[string]any{}})
			default:
				t.Fatalf("unexpected zhihu homefeed page_number: %s (%s)", pageNumber, r.URL.RawQuery)
			}
		case "/api/v4/creators/rank/hot":
			if len(homefeedCalls) > 1 && homefeedCalls[1] != nil {
				(*homefeedCalls[1])++
			}
			writeJSON(t, w, zhihuHotSearchPayload())
		case "/api/v4/questions/zh_hot_question_e2e/feeds":
			if len(homefeedCalls) > 2 && homefeedCalls[2] != nil {
				(*homefeedCalls[2])++
			}
			writeJSON(t, w, zhihuHotQuestionFeedsPayload())
		case "/api/v4/comment_v5/answers/zh_answer_detail_e2e/root_comment":
			writeJSON(t, w, zhihuRootCommentsPayload())
		case "/api/v4/comment_v5/comment/zh_comment_root/child_comment":
			writeJSON(t, w, zhihuChildCommentsPayload())
		case "/signsrv/v1/zhihu/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"x_zst_81": "mock-x-zst-81",
					"x_zse_96": "mock-x-zse-96",
				},
			})
		case "/api/v4/search_v3":
			*searchCalls = *searchCalls + 1
			if r.URL.Query().Get("q") != "mock-zhihu" {
				t.Fatalf("unexpected zhihu search keyword: %s", r.URL.RawQuery)
			}
			if r.URL.Query().Get("offset") != "0" {
				writeJSON(t, w, map[string]any{"data": []map[string]any{}})
				return
			}
			writeJSON(t, w, zhihuSearchPayload())
		default:
			t.Fatalf("unexpected zhihu request: %s %s", r.Method, r.URL.String())
		}
	}))
}

func zhihuCreatorHTML() string {
	payload := map[string]any{
		"initialState": map[string]any{
			"entities": map[string]any{
				"users": map[string]any{
					"mock-zh-creator": map[string]any{
						"id":             "zh-creator-user",
						"name":           "mock zhihu creator",
						"avatarUrl":      "https://example.test/zhihu-creator-avatar.png",
						"urlToken":       "mock-zh-creator",
						"gender":         1,
						"ipInfo":         "浙江",
						"followingCount": 12,
						"followerCount":  345,
						"answerCount":    7,
						"zvideoCount":    2,
						"questionCount":  3,
						"articlesCount":  4,
						"columnsCount":   1,
						"voteupCount":    9876,
					},
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return `<html><body><script id="js-initialData" type="application/json">` + string(body) + `</script></body></html>`
}

func zhihuCreatorResumeHTML() string {
	payload := map[string]any{
		"initialState": map[string]any{
			"entities": map[string]any{
				"users": map[string]any{
					"mock-zh-creator-resume": map[string]any{
						"id":             "zh-creator-resume-user",
						"name":           "mock zhihu creator resume",
						"avatarUrl":      "https://example.test/zhihu-creator-resume-avatar.png",
						"urlToken":       "mock-zh-creator-resume",
						"gender":         1,
						"ipInfo":         "浙江",
						"followingCount": 12,
						"followerCount":  345,
						"answerCount":    7,
						"zvideoCount":    2,
						"questionCount":  3,
						"articlesCount":  4,
						"columnsCount":   1,
						"voteupCount":    9876,
					},
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return `<html><body><script id="js-initialData" type="application/json">` + string(body) + `</script></body></html>`
}

func zhihuCreatorAnswersPayload() map[string]any {
	return map[string]any{
		"paging": map[string]any{"is_end": true, "next": ""},
		"data": []map[string]any{
			{
				"type":          "answer",
				"id":            "zh_creator_answer_e2e",
				"content":       "<p>Zhihu creator answer E2E content</p>",
				"question":      map[string]any{"id": "zh_creator_question_e2e"},
				"title":         "<b>Zhihu creator question</b>",
				"description":   "<p>Zhihu creator answer description</p>",
				"excerpt":       "<p>unused creator excerpt</p>",
				"created_time":  1710002000,
				"updated_time":  1710002100,
				"voteup_count":  288,
				"comment_count": 25,
				"author": map[string]any{
					"id":         "zh-creator-user",
					"name":       "mock zhihu creator",
					"avatar_url": "https://example.test/zhihu-creator-avatar.png",
					"url_token":  "mock-zh-creator",
				},
			},
		},
	}
}

func zhihuCreatorAnswersResumePayload(contentID, text string, isEnd bool) map[string]any {
	return map[string]any{
		"paging": map[string]any{"is_end": isEnd, "next": ""},
		"data": []map[string]any{
			{
				"type":          "answer",
				"id":            contentID,
				"content":       "<p>" + text + "</p>",
				"question":      map[string]any{"id": "zh_creator_resume_question"},
				"title":         "<b>Zhihu creator resume question</b>",
				"description":   "<p>Zhihu creator resume answer description</p>",
				"excerpt":       "<p>unused creator resume excerpt</p>",
				"created_time":  1710002000,
				"updated_time":  1710002100,
				"voteup_count":  288,
				"comment_count": 25,
				"author": map[string]any{
					"id":         "zh-creator-resume-user",
					"name":       "mock zhihu creator resume",
					"avatar_url": "https://example.test/zhihu-creator-resume-avatar.png",
					"url_token":  "mock-zh-creator-resume",
				},
			},
		},
	}
}

func zhihuHomefeedPayload(baseURL string) map[string]any {
	return map[string]any{
		"paging": map[string]any{
			"is_end": false,
			"next":   baseURL + "/api/v3/feed/topstory/recommend?after_id=1&end_offset=20&session_token=mock-homefeed-session&page_number=2",
		},
		"data": []map[string]any{
			{
				"type": "feed",
				"target": map[string]any{
					"type":          "answer",
					"id":            "zh_homefeed_answer_e2e",
					"content":       "<p>Zhihu homefeed answer E2E content</p>",
					"question":      map[string]any{"id": "zh_homefeed_question_e2e"},
					"title":         "<b>Zhihu homefeed question</b>",
					"description":   "<p>Zhihu homefeed answer description</p>",
					"excerpt":       "<p>unused homefeed excerpt</p>",
					"created_time":  1710002200,
					"updated_time":  1710002300,
					"voteup_count":  388,
					"comment_count": 35,
					"author": map[string]any{
						"id":         "zh-homefeed-author",
						"name":       "mock zhihu homefeed author",
						"avatar_url": "https://example.test/zhihu-homefeed-avatar.png",
						"url_token":  "mock-zh-homefeed-author",
					},
				},
			},
		},
	}
}

func zhihuHomefeedResumePayload(contentID, text, nextURL string) map[string]any {
	return map[string]any{
		"paging": map[string]any{
			"is_end": false,
			"next":   nextURL,
		},
		"data": []map[string]any{
			{
				"type": "feed",
				"target": map[string]any{
					"type":          "answer",
					"id":            contentID,
					"content":       "<p>" + text + "</p>",
					"question":      map[string]any{"id": "zh_homefeed_resume_question"},
					"title":         "<b>Zhihu homefeed resume question</b>",
					"description":   "<p>Zhihu homefeed resume answer description</p>",
					"excerpt":       "<p>unused homefeed resume excerpt</p>",
					"created_time":  1710002200,
					"updated_time":  1710002300,
					"voteup_count":  388,
					"comment_count": 35,
					"author": map[string]any{
						"id":         "zh-homefeed-resume-author",
						"name":       "mock zhihu homefeed resume author",
						"avatar_url": "https://example.test/zhihu-homefeed-resume-avatar.png",
						"url_token":  "mock-zh-homefeed-resume-author",
					},
				},
			},
		},
	}
}

func zhihuHotSearchPayload() map[string]any {
	return map[string]any{
		"data": []map[string]any{
			{
				"question": map[string]any{
					"id":    "zh_hot_question_e2e",
					"title": "Zhihu hot question",
				},
				"score": 12345,
				"label": "热",
			},
		},
	}
}

func zhihuHotQuestionFeedsPayload() map[string]any {
	return map[string]any{
		"paging": map[string]any{"is_end": true, "next": ""},
		"data": []map[string]any{
			{
				"target_type": "answer",
				"target": map[string]any{
					"type":          "answer",
					"id":            "zh_hot_answer_e2e",
					"content":       "<p>Zhihu hot answer E2E content</p>",
					"question":      map[string]any{"id": "zh_hot_question_e2e"},
					"title":         "<b>Zhihu hot question</b>",
					"description":   "<p>Zhihu hot answer description</p>",
					"excerpt":       "<p>unused hot excerpt</p>",
					"created_time":  1710002400,
					"updated_time":  1710002500,
					"voteup_count":  488,
					"comment_count": 45,
					"author": map[string]any{
						"id":         "zh-hot-author",
						"name":       "mock zhihu hot author",
						"avatar_url": "https://example.test/zhihu-hot-avatar.png",
						"url_token":  "mock-zh-hot-author",
					},
				},
			},
		},
	}
}

func zhihuHotSearchResumePayload() map[string]any {
	return map[string]any{
		"data": []map[string]any{
			{
				"question": map[string]any{"id": "zh_hot_resume_question_1", "title": "Zhihu hot resume question 1"},
				"score":    22345,
				"label":    "热",
			},
			{
				"question": map[string]any{"id": "zh_hot_resume_question_2", "title": "Zhihu hot resume question 2"},
				"score":    22344,
				"label":    "新",
			},
		},
	}
}

func zhihuHotQuestionResumeFeedsPayload(contentID, questionID, title string, createdTime int) map[string]any {
	return map[string]any{
		"paging": map[string]any{"is_end": true, "next": ""},
		"data": []map[string]any{
			{
				"target_type": "answer",
				"target": map[string]any{
					"type":          "answer",
					"id":            contentID,
					"content":       "<p>" + title + " content</p>",
					"question":      map[string]any{"id": questionID},
					"title":         "<b>" + title + "</b>",
					"description":   "<p>" + title + " description</p>",
					"excerpt":       "<p>unused hot resume excerpt</p>",
					"created_time":  createdTime,
					"updated_time":  createdTime + 100,
					"voteup_count":  48,
					"comment_count": 4,
					"author": map[string]any{
						"id":         "zh-hot-resume-author",
						"name":       "mock zhihu hot resume author",
						"avatar_url": "https://example.test/zhihu-hot-resume-avatar.png",
						"url_token":  "mock-zh-hot-resume-author",
					},
				},
			},
		},
	}
}

func zhihuHotQuestionPagedFeedsPayload(contentID, title string, isEnd bool, next string) map[string]any {
	return map[string]any{
		"paging": map[string]any{"is_end": isEnd, "next": next},
		"data": []map[string]any{
			{
				"target_type": "answer",
				"target": map[string]any{
					"type":          "answer",
					"id":            contentID,
					"content":       "<p>" + title + " content</p>",
					"question":      map[string]any{"id": "zh_hot_question_e2e"},
					"title":         "<b>Zhihu hot question</b>",
					"description":   "<p>" + title + " description</p>",
					"excerpt":       "<p>unused hot paged excerpt</p>",
					"created_time":  1710300000,
					"updated_time":  1710300100,
					"voteup_count":  148,
					"comment_count": 14,
					"author": map[string]any{
						"id":         "zh-hot-paged-author",
						"name":       "mock zhihu hot paged author",
						"avatar_url": "https://example.test/zhihu-hot-paged-avatar.png",
						"url_token":  "mock-zh-hot-paged-author",
					},
				},
			},
		},
	}
}

func zhihuRootCommentsPayload() map[string]any {
	return map[string]any{
		"paging": map[string]any{"is_end": true, "next": ""},
		"data": []map[string]any{
			{
				"type":                "comment",
				"id":                  "zh_comment_root",
				"reply_comment_id":    "",
				"content":             "<p>root zhihu comment</p>",
				"created_time":        1710001900,
				"comment_tag":         []map[string]string{{"type": "ip_info", "text": "广东"}},
				"child_comment_count": 1,
				"like_count":          12,
				"dislike_count":       2,
				"author": map[string]any{
					"id":         "zh-commenter",
					"name":       "mock zhihu commenter",
					"avatar_url": "https://example.test/zhihu-comment-avatar.png",
					"url_token":  "mock-zh-commenter",
				},
			},
		},
	}
}

func zhihuChildCommentsPayload() map[string]any {
	return map[string]any{
		"paging": map[string]any{"is_end": true, "next": ""},
		"data": []map[string]any{
			{
				"type":                "comment",
				"id":                  "zh_comment_sub",
				"reply_comment_id":    "zh_comment_root",
				"content":             "<p>sub zhihu comment</p>",
				"created_time":        1710001910,
				"comment_tag":         []map[string]string{{"type": "ip_info", "text": "深圳"}},
				"child_comment_count": 0,
				"like_count":          3,
				"dislike_count":       0,
				"author": map[string]any{
					"id":         "zh-sub-commenter",
					"name":       "mock zhihu sub commenter",
					"avatar_url": "https://example.test/zhihu-sub-comment-avatar.png",
					"url_token":  "mock-zh-sub-commenter",
				},
			},
		},
	}
}

func zhihuSearchPayload() map[string]any {
	return map[string]any{
		"data": []map[string]any{
			{
				"type": "search_result",
				"object": map[string]any{
					"type":          "answer",
					"id":            "zh_answer_e2e",
					"content":       "<p>Zhihu answer E2E content</p>",
					"title":         "<b>Zhihu E2E question</b>",
					"description":   "<p>Zhihu answer description</p>",
					"excerpt":       "<p>unused excerpt</p>",
					"created_time":  1710001500,
					"updated_time":  1710001600,
					"voteup_count":  88,
					"comment_count": 5,
					"question": map[string]any{
						"id": "zh_question_e2e",
					},
					"author": map[string]any{
						"id":         "zh-author",
						"name":       "mock zhihu author",
						"avatar_url": "https://example.test/zhihu-avatar.png",
						"url_token":  "mock-zh-author",
					},
				},
			},
		},
	}
}

func zhihuResumeSearchPayload(contentID, questionID, title string, createdTime int) map[string]any {
	return map[string]any{
		"data": []map[string]any{
			{
				"type": "search_result",
				"object": map[string]any{
					"type":          "answer",
					"id":            contentID,
					"content":       "<p>" + title + " content</p>",
					"title":         "<b>" + title + "</b>",
					"description":   "<p>" + title + " description</p>",
					"excerpt":       "<p>unused resume excerpt</p>",
					"created_time":  createdTime,
					"updated_time":  createdTime + 100,
					"voteup_count":  8,
					"comment_count": 1,
					"question": map[string]any{
						"id": questionID,
					},
					"author": map[string]any{
						"id":         "zh-resume-author",
						"name":       "mock zhihu resume author",
						"avatar_url": "https://example.test/zhihu-resume-avatar.png",
						"url_token":  "mock-zh-resume-author",
					},
				},
			},
		},
	}
}

func zhihuDetailHTML() string {
	payload := map[string]any{
		"initialState": map[string]any{
			"entities": map[string]any{
				"answers": map[string]any{
					"zh_answer_detail_e2e": map[string]any{
						"type":          "answer",
						"id":            "zh_answer_detail_e2e",
						"content":       "<p>Zhihu detail E2E content</p>",
						"question":      map[string]any{"id": "zh_question_detail_e2e"},
						"title":         "<b>Zhihu detail question</b>",
						"description":   "<p>Zhihu detail description</p>",
						"excerpt":       "<p>unused detail excerpt</p>",
						"created_time":  1710001700,
						"updated_time":  1710001800,
						"voteup_count":  188,
						"comment_count": 15,
						"author": map[string]any{
							"id":         "zh-detail-author",
							"name":       "mock zhihu detail author",
							"avatar_url": "https://example.test/zhihu-detail-avatar.png",
							"url_token":  "mock-zh-detail-author",
						},
					},
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return `<html><body><script id="js-initialData" type="application/json">` + string(body) + `</script></body></html>`
}

func zhihuDetailHTMLFor(questionID, answerID, title, authorID string) string {
	payload := map[string]any{
		"initialState": map[string]any{
			"entities": map[string]any{
				"answers": map[string]any{
					answerID: map[string]any{
						"type":          "answer",
						"id":            answerID,
						"content":       "<p>" + title + " content</p>",
						"question":      map[string]any{"id": questionID},
						"title":         "<b>" + title + "</b>",
						"description":   "<p>" + title + " description</p>",
						"excerpt":       "<p>unused detail excerpt</p>",
						"created_time":  1710001700,
						"updated_time":  1710001800,
						"voteup_count":  188,
						"comment_count": 15,
						"author": map[string]any{
							"id":         authorID,
							"name":       "mock " + authorID,
							"avatar_url": "https://example.test/" + authorID + ".png",
							"url_token":  authorID,
						},
					},
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return `<html><body><script id="js-initialData" type="application/json">` + string(body) + `</script></body></html>`
}
