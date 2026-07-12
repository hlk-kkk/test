// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/bilibili_e2e_test.go
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
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/cookiebridge"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCLIBilibiliSearchJSONWithMockServers(t *testing.T) {
	result := runCLIBilibiliSearchWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "bilibili", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
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
	if rows[0]["video_id"] != "12345" {
		t.Fatalf("unexpected video_id: %#v", rows[0]["video_id"])
	}
	if rows[0]["bvid"] != "BV1mockbili" {
		t.Fatalf("unexpected bvid: %#v", rows[0]["bvid"])
	}
	if rows[0]["title"] != "Bilibili E2E mock video" {
		t.Fatalf("unexpected title: %#v", rows[0]["title"])
	}
	if rows[0]["source_keyword"] != "mock-keyword" {
		t.Fatalf("unexpected source_keyword: %#v", rows[0]["source_keyword"])
	}
	if rows[0]["create_time"] != "1710000000" {
		t.Fatalf("unexpected create_time: %#v", rows[0]["create_time"])
	}
	if rows[0]["duration"] != "66" {
		t.Fatalf("unexpected duration: %#v", rows[0]["duration"])
	}
	if rows[0]["video_cover_url"] != "https://example.test/bili-cover.png" {
		t.Fatalf("unexpected video_cover_url: %#v", rows[0]["video_cover_url"])
	}
	if rows[0]["video_url"] != "https://www.bilibili.com/video/BV1mockbili" {
		t.Fatalf("unexpected video_url: %#v", rows[0]["video_url"])
	}
	if value, ok := rows[0]["add_ts"]; !ok || value != nil {
		t.Fatalf("expected Python-compatible add_ts:null, got present=%t value=%#v", ok, value)
	}
}

func TestCLIBilibiliSearchRespectsMaxNotesWithinPage(t *testing.T) {
	var viewCalls int
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "bili",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "bili-search-limit-client",
				Connected: true,
				Nicknames: map[string]string{"bili": "mock-bili"},
				Cookies:   map[string]string{"bili": "SESSDATA=search-limit; buvid3=search-limit"},
			},
		},
		PlatformHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/x/web-interface/wbi/search/type":
				writeJSON(t, w, map[string]any{
					"code":    0,
					"message": "0",
					"data": map[string]any{
						"numResults": 2,
						"page":       1,
						"pagesize":   20,
						"result": []map[string]any{
							{
								"type":  "video",
								"aid":   int64(12345),
								"bvid":  "BV1mockbili",
								"title": "mock search title 1",
							},
							{
								"type":  "video",
								"aid":   int64(67890),
								"bvid":  "BV1mockbili2",
								"title": "mock search title 2",
							},
						},
					},
				})
			case "/x/web-interface/view/detail":
				viewCalls++
				switch r.URL.Query().Get("bvid") {
				case "BV1mockbili":
					writeJSON(t, w, bilibiliResumeDetailPayload(12345, "BV1mockbili", "Bilibili max limit video 1"))
				case "BV1mockbili2":
					writeJSON(t, w, bilibiliResumeDetailPayload(67890, "BV1mockbili2", "Bilibili max limit video 2"))
				default:
					t.Fatalf("unexpected detail bvid: %s", r.URL.Query().Get("bvid"))
				}
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
			}
		}),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "bili",
		Keywords:             []string{"mock-keyword"},
		CrawlerType:          "search",
		SaveDataOption:       "json",
		CrawlerMaxNotesCount: 1,
		EnableGetComments:    false,
		EnableCheckpoint:     false,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Timeout: 20 * time.Second,
	})

	resultPath := filepath.Join(runner.DataDir, "bilibili", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := e2e.ReadJSONRows(t, resultPath, output)
	if len(rows) != 1 {
		t.Fatalf("crawlerMaxNotesCount=1 should save one row from a multi-item search page, got %d rows: %#v\n%s", len(rows), rows, output)
	}
	if viewCalls != 1 {
		t.Fatalf("crawlerMaxNotesCount=1 should request one detail, got %d\n%s", viewCalls, output)
	}
	if rows[0]["bvid"] != "BV1mockbili" {
		t.Fatalf("unexpected saved bvid: %#v", rows[0]["bvid"])
	}
}

func TestCLIBilibiliHotSearchJSONWithMockServers(t *testing.T) {
	result := runCLIBilibiliHotSearchWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "bilibili", "json", "hot_search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read hot_search result file: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal hot_search result json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one saved hot_search row, got %d: %s", len(rows), body)
	}
	row := rows[0]
	if row["video_id"] != "12345" {
		t.Fatalf("unexpected hot_search video_id: %#v", row["video_id"])
	}
	if row["bvid"] != "BV1mockbili" {
		t.Fatalf("unexpected hot_search bvid: %#v", row["bvid"])
	}
	if row["title"] != "Bilibili E2E mock video" {
		t.Fatalf("unexpected hot_search title: %#v", row["title"])
	}
	if row["source_keyword"] != "bilibili_hot_search" {
		t.Fatalf("unexpected hot_search source_keyword: %#v", row["source_keyword"])
	}
	if row["create_time"] != "1710000000" {
		t.Fatalf("unexpected hot_search create_time: %#v", row["create_time"])
	}
	if row["duration"] != "66" {
		t.Fatalf("unexpected hot_search duration: %#v", row["duration"])
	}
	if row["video_url"] != "https://www.bilibili.com/video/BV1mockbili" {
		t.Fatalf("unexpected hot_search video_url: %#v", row["video_url"])
	}
	if row["video_cover_url"] != "https://example.test/bili-cover.png" {
		t.Fatalf("unexpected hot_search video_cover_url: %#v", row["video_cover_url"])
	}
	if value, ok := row["add_ts"]; !ok || value != nil {
		t.Fatalf("expected Python-compatible hot_search add_ts:null, got present=%t value=%#v", ok, value)
	}
}

func TestCLIBilibiliHotSearchCSVWithMockServers(t *testing.T) {
	result := runCLIBilibiliHotSearchWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "bilibili", "1_hot_search_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open hot_search csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read hot_search csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected csv header and one hot_search row, got %d: %#v", len(records), records)
	}
	if csvCell(records, "video_id") != "12345" {
		t.Fatalf("unexpected hot_search csv video_id: %#v", csvCell(records, "video_id"))
	}
	if csvCell(records, "bvid") != "BV1mockbili" {
		t.Fatalf("unexpected hot_search csv bvid: %#v", csvCell(records, "bvid"))
	}
	if csvCell(records, "source_keyword") != "bilibili_hot_search" {
		t.Fatalf("unexpected hot_search csv source_keyword: %#v", csvCell(records, "source_keyword"))
	}
	if csvCell(records, "create_time") != "1710000000" || csvCell(records, "duration") != "66" {
		t.Fatalf("unexpected hot_search csv time fields: %#v", records)
	}
	if csvCell(records, "video_cover_url") != "https://example.test/bili-cover.png" {
		t.Fatalf("unexpected hot_search csv video_cover_url: %#v", csvCell(records, "video_cover_url"))
	}
	if csvCell(records, "add_ts") != "" {
		t.Fatalf("expected Python-compatible blank hot_search csv add_ts, got %#v", csvCell(records, "add_ts"))
	}
}

func TestCLIBilibiliHotSearchSQLiteWithMockServers(t *testing.T) {
	result := runCLIBilibiliHotSearchWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open hot_search sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	var row struct {
		VideoID       string `gorm:"column:video_id"`
		Bvid          string `gorm:"column:bvid"`
		Title         string `gorm:"column:title"`
		CreateTime    int64  `gorm:"column:create_time"`
		Duration      string `gorm:"column:duration"`
		VideoCoverURL string `gorm:"column:video_cover_url"`
		SourceKeyword string `gorm:"column:source_keyword"`
		AddTs         int64  `gorm:"column:add_ts"`
		LastModifyTs  int64  `gorm:"column:last_modify_ts"`
	}
	if err := db.Table("bilibili_video").Where("video_id = ?", "12345").First(&row).Error; err != nil {
		t.Fatalf("query hot_search sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	if row.Bvid != "BV1mockbili" || row.Title != "Bilibili E2E mock video" {
		t.Fatalf("unexpected hot_search sqlite identity fields: %#v", row)
	}
	if row.CreateTime != 1710000000 || row.Duration != "66" {
		t.Fatalf("unexpected hot_search sqlite time fields: %#v", row)
	}
	if row.VideoCoverURL != "https://example.test/bili-cover.png" {
		t.Fatalf("unexpected hot_search sqlite video_cover_url: %#v", row.VideoCoverURL)
	}
	if row.SourceKeyword != "bilibili_hot_search" {
		t.Fatalf("unexpected hot_search sqlite source_keyword: %#v", row.SourceKeyword)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected non-zero hot_search sqlite timestamps: %#v", row)
	}
}

func TestCLIBilibiliHotSearchJSONCommentsWithMockServers(t *testing.T) {
	result := runCLIBilibiliHotSearchWithMockServers(t, "json", true)
	if result.commentCalls != 1 {
		t.Fatalf("expected one hot_search root comment request, got %d\n%s", result.commentCalls, result.output)
	}
	if result.subCommentCalls != 1 {
		t.Fatalf("expected one hot_search sub-comment request, got %d\n%s", result.subCommentCalls, result.output)
	}

	resultPath := filepath.Join(result.dataDir, "bilibili", "json", "hot_search_comments_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read hot_search comments file: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal hot_search comments json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected root and sub hot_search comment rows, got %d: %s", len(rows), body)
	}

	comments := make(map[string]map[string]any, len(rows))
	for _, row := range rows {
		comments[fmt.Sprint(row["comment_id"])] = row
	}
	root := comments["9001"]
	if root["video_id"] != "12345" || root["content"] != "root bili comment" {
		t.Fatalf("unexpected hot_search root comment: %#v", root)
	}
	if root["create_time"] != "1710000100" || root["sub_comment_count"] != "1" || root["like_count"] != "4" {
		t.Fatalf("unexpected hot_search root count fields: %#v", root)
	}
	if root["parent_comment_id"] != "" {
		t.Fatalf("unexpected hot_search root parent_comment_id: %#v", root["parent_comment_id"])
	}
	if root["user_id"] != "8801" || root["nickname"] != "bili root commenter" || root["avatar"] != "https://example.test/bili-root.png" {
		t.Fatalf("unexpected hot_search root user fields: %#v", root)
	}
	if value, ok := root["add_ts"]; !ok || value != nil {
		t.Fatalf("expected Python-compatible hot_search root add_ts:null, got present=%t value=%#v", ok, value)
	}

	sub := comments["9002"]
	if sub["video_id"] != "12345" || sub["content"] != "sub bili comment" {
		t.Fatalf("unexpected hot_search sub comment: %#v", sub)
	}
	if sub["create_time"] != "1710000200" || sub["sub_comment_count"] != "0" || sub["like_count"] != "2" {
		t.Fatalf("unexpected hot_search sub count fields: %#v", sub)
	}
	if sub["parent_comment_id"] != "9001" {
		t.Fatalf("unexpected hot_search sub parent_comment_id: %#v", sub["parent_comment_id"])
	}
	if sub["user_id"] != "8802" || sub["nickname"] != "bili sub commenter" || sub["avatar"] != "https://example.test/bili-sub.png" {
		t.Fatalf("unexpected hot_search sub user fields: %#v", sub)
	}
	if value, ok := sub["add_ts"]; !ok || value != nil {
		t.Fatalf("expected Python-compatible hot_search sub add_ts:null, got present=%t value=%#v", ok, value)
	}
}

func TestCLIBilibiliSearchDoesNotWriteCheckpointWhenDisabled(t *testing.T) {
	result := runCLIBilibiliSearchWithMockServers(t, "json")

	checkpointDir := filepath.Join(result.dataDir, "checkpoints")
	entries, err := os.ReadDir(checkpointDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read checkpoint dir: %v\ncommand output:\n%s", err, result.output)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no checkpoint files when enableCheckpoint=false, got %d entries in %s", len(entries), checkpointDir)
	}
}

func TestCLIBilibiliSearchCheckpointResumeSkipsCompletedPages(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	searchCallsByPage := make(map[string]int)
	viewCallsByBvid := make(map[string]int)
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
							"client_id": "bili-resume-client",
							"platforms": map[string]any{
								"bili": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"bili": "mock-bili"},
						},
					},
				},
			})
		case "/api/cookies/bili":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "SESSDATA=resume-session; buvid3=resume-buvid"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/bilibili/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"wts": "1710000000", "w_rid": "resume-wrid"},
			})
		case "/x/web-interface/wbi/search/type":
			page := r.URL.Query().Get("page")
			searchCallsByPage[page]++
			switch page {
			case "1":
				writeJSON(t, w, bilibiliResumeSearchPayload(20001, "BVresume1"))
			case "2":
				writeJSON(t, w, bilibiliResumeSearchPayload(20002, "BVresume2"))
			default:
				writeJSON(t, w, map[string]any{
					"code":    0,
					"message": "0",
					"data": map[string]any{
						"numResults": 0,
						"page":       page,
						"pagesize":   20,
						"result":     []map[string]any{},
					},
				})
			}
		case "/x/web-interface/view/detail":
			bvid := r.URL.Query().Get("bvid")
			viewCallsByBvid[bvid]++
			switch bvid {
			case "BVresume1":
				writeJSON(t, w, bilibiliResumeDetailPayload(20001, "BVresume1", "Bilibili resume video 1"))
			case "BVresume2":
				writeJSON(t, w, bilibiliResumeDetailPayload(20002, "BVresume2", "Bilibili resume video 2"))
			default:
				t.Fatalf("unexpected detail bvid: %s", bvid)
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
platform: "bili"
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
  bilibiliApiUrl: %q
  bilibiliIndexUrl: %q
  bilibiliSpaceUrl: %q
`, mockServer.URL, dataDir, maxNotes, host, port, mockServer.URL, mockServer.URL, mockServer.URL)
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
	if searchCallsByPage["1"] != 1 || viewCallsByBvid["BVresume1"] != 1 {
		t.Fatalf("first run did not crawl page 1 once, search=%#v view=%#v\n%s", searchCallsByPage, viewCallsByBvid, firstOutput)
	}

	writeResumeConfig(2)
	secondOutput := runCrawler()
	if searchCallsByPage["1"] != 1 {
		t.Fatalf("resume should not re-fetch completed page 1, search calls=%#v\n%s", searchCallsByPage, secondOutput)
	}
	if searchCallsByPage["2"] != 1 || viewCallsByBvid["BVresume2"] != 1 {
		t.Fatalf("resume did not crawl page 2 once, search=%#v view=%#v\n%s", searchCallsByPage, viewCallsByBvid, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "bilibili", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read resume result json: %v\n%s", err, secondOutput)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal resume result json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique rows after resume, got %d: %s", len(rows), body)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["video_id"])] = true
	}
	if !seen["20001"] || !seen["20002"] {
		t.Fatalf("resume output missing expected video IDs: seen=%#v rows=%s", seen, body)
	}
}

func TestCLIBilibiliHomefeedCheckpointResumeSkipsCompletedFreshIdx(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	feedCallsByFreshIdx := make(map[string]int)
	viewCallsByBvid := make(map[string]int)
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
							"client_id": "bili-homefeed-resume-client",
							"platforms": map[string]any{
								"bili": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"bili": "mock-bili"},
						},
					},
				},
			})
		case "/api/cookies/bili":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "SESSDATA=homefeed-resume-session; buvid3=homefeed-resume-buvid"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/bilibili/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"wts": "1710000000", "w_rid": "homefeed-resume-wrid"},
			})
		case "/x/web-interface/wbi/index/top/feed/rcmd":
			freshIdx := r.URL.Query().Get("fresh_idx")
			feedCallsByFreshIdx[freshIdx]++
			switch freshIdx {
			case "1":
				writeJSON(t, w, bilibiliHomefeedPagePayload(23001, "BVhomeResume1"))
			case "2":
				writeJSON(t, w, bilibiliHomefeedPagePayload(23002, "BVhomeResume2"))
			default:
				writeJSON(t, w, map[string]any{
					"code":    0,
					"message": "0",
					"data":    map[string]any{"item": []map[string]any{}},
				})
			}
		case "/x/web-interface/view/detail":
			bvid := r.URL.Query().Get("bvid")
			viewCallsByBvid[bvid]++
			switch bvid {
			case "BVhomeResume1":
				writeJSON(t, w, bilibiliResumeDetailPayload(23001, bvid, "Bilibili homefeed resume video 1"))
			case "BVhomeResume2":
				writeJSON(t, w, bilibiliResumeDetailPayload(23002, bvid, "Bilibili homefeed resume video 2"))
			default:
				t.Fatalf("unexpected homefeed resume detail bvid: %s", bvid)
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
platform: "bili"
keywords:
  - "unused"
crawlerType: "homefeed"
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
  bilibiliApiUrl: %q
  bilibiliIndexUrl: %q
  bilibiliSpaceUrl: %q
`, mockServer.URL, dataDir, maxNotes, host, port, mockServer.URL, mockServer.URL, mockServer.URL)
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
			t.Fatalf("bilibili homefeed resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("bilibili homefeed resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	writeResumeConfig(1)
	firstOutput := runCrawler()
	if feedCallsByFreshIdx["1"] != 1 || viewCallsByBvid["BVhomeResume1"] != 1 {
		t.Fatalf("first homefeed run should crawl fresh_idx=1 once, feed=%#v view=%#v\n%s", feedCallsByFreshIdx, viewCallsByBvid, firstOutput)
	}

	writeResumeConfig(2)
	secondOutput := runCrawler()
	if feedCallsByFreshIdx["1"] != 1 {
		t.Fatalf("resume should not re-fetch completed homefeed fresh_idx=1, feed calls=%#v\n%s", feedCallsByFreshIdx, secondOutput)
	}
	if feedCallsByFreshIdx["2"] != 1 || viewCallsByBvid["BVhomeResume2"] != 1 {
		t.Fatalf("resume did not crawl homefeed fresh_idx=2 once, feed=%#v view=%#v\n%s", feedCallsByFreshIdx, viewCallsByBvid, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "bilibili", "json", "homefeed_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Bilibili homefeed rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["video_id"])] = true
	}
	if !seen["23001"] || !seen["23002"] {
		t.Fatalf("resume output missing expected homefeed video IDs, seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIBilibiliHomefeedCheckpointResumeRetriesOnlyFailedItem(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	allowSecondDetail := false
	feedCallsByFreshIdx := make(map[string]int)
	viewCallsByBvid := make(map[string]int)
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
							"client_id": "bili-homefeed-partial-resume-client",
							"platforms": map[string]any{
								"bili": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"bili": "mock-bili"},
						},
					},
				},
			})
		case "/api/cookies/bili":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "SESSDATA=homefeed-partial-resume-session; buvid3=homefeed-partial-resume-buvid"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/bilibili/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"wts": "1710000000", "w_rid": "homefeed-partial-resume-wrid"},
			})
		case "/x/web-interface/wbi/index/top/feed/rcmd":
			freshIdx := r.URL.Query().Get("fresh_idx")
			feedCallsByFreshIdx[freshIdx]++
			if freshIdx != "1" {
				writeJSON(t, w, map[string]any{
					"code":    0,
					"message": "0",
					"data":    map[string]any{"item": []map[string]any{}},
				})
				return
			}
			writeJSON(t, w, bilibiliHomefeedItemsPayload(
				bilibiliHomefeedItem(23101, "BVhomePartialResume1"),
				bilibiliHomefeedItem(23102, "BVhomePartialResume2"),
			))
		case "/x/web-interface/view/detail":
			bvid := r.URL.Query().Get("bvid")
			viewCallsByBvid[bvid]++
			switch bvid {
			case "BVhomePartialResume1":
				writeJSON(t, w, bilibiliResumeDetailPayload(23101, bvid, "Bilibili homefeed partial resume video 1"))
			case "BVhomePartialResume2":
				if !allowSecondDetail {
					http.Error(w, "transient second homefeed detail failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, bilibiliResumeDetailPayload(23102, bvid, "Bilibili homefeed partial resume video 2"))
			default:
				t.Fatalf("unexpected homefeed partial resume detail bvid: %s", bvid)
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

	configYAML := fmt.Sprintf(`
platform: "bili"
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
  bilibiliApiUrl: %q
  bilibiliIndexUrl: %q
  bilibiliSpaceUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL, mockServer.URL)
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
			t.Fatalf("bilibili homefeed partial resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("bilibili homefeed partial resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if feedCallsByFreshIdx["1"] != 1 || viewCallsByBvid["BVhomePartialResume1"] != 1 || viewCallsByBvid["BVhomePartialResume2"] != 1 {
		t.Fatalf("first partial run should fetch page 1 and attempt both details once, feed=%#v view=%#v\n%s", feedCallsByFreshIdx, viewCallsByBvid, firstOutput)
	}

	allowSecondDetail = true
	secondOutput := runCrawler()
	if feedCallsByFreshIdx["1"] != 2 {
		t.Fatalf("partial resume should retry the same homefeed page once, feed calls=%#v\n%s", feedCallsByFreshIdx, secondOutput)
	}
	if viewCallsByBvid["BVhomePartialResume1"] != 1 {
		t.Fatalf("partial resume should skip completed first homefeed BVID, calls=%#v\n%s", viewCallsByBvid, secondOutput)
	}
	if viewCallsByBvid["BVhomePartialResume2"] != 2 {
		t.Fatalf("partial resume should retry failed second homefeed BVID exactly once, calls=%#v\n%s", viewCallsByBvid, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "bilibili", "json", "homefeed_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Bilibili homefeed rows after partial resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["video_id"])] = true
	}
	if !seen["23101"] || !seen["23102"] {
		t.Fatalf("partial resume output missing expected homefeed video IDs, seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIBilibiliCreatorCheckpointResumeByCreatorAndPage(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	allowSecondPage := false
	creatorInfoCalls := 0
	creatorVideoCallsByPage := make(map[string]int)
	viewCallsByBvid := make(map[string]int)
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
							"client_id": "bili-creator-resume-client",
							"platforms": map[string]any{
								"bili": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"bili": "mock-bili"},
						},
					},
				},
			})
		case "/api/cookies/bili":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "SESSDATA=creator-resume-session; buvid3=creator-resume-buvid"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/bilibili/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"wts": "1710000000", "w_rid": "creator-resume-wrid"},
			})
		case "/x/space/wbi/acc/info":
			if got := r.URL.Query().Get("mid"); got != "777" {
				t.Fatalf("unexpected creator info mid: %s", got)
			}
			creatorInfoCalls++
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data": map[string]any{
					"mid":  int64(777),
					"name": "resume bili author",
					"face": "https://example.test/bili-avatar.png",
					"sign": "creator resume desc",
				},
			})
		case "/x/relation/stat":
			if got := r.URL.Query().Get("vmid"); got != "777" {
				t.Fatalf("unexpected relation stat vmid: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data":    map[string]any{"follower": int64(1234), "following": int64(56)},
			})
		case "/x/space/navnum":
			if got := r.URL.Query().Get("mid"); got != "777" {
				t.Fatalf("unexpected navnum mid: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data":    map[string]any{"video": int64(2)},
			})
		case "/x/space/wbi/arc/search":
			if got := r.URL.Query().Get("mid"); got != "777" {
				t.Fatalf("unexpected creator video mid: %s", got)
			}
			page := r.URL.Query().Get("pn")
			creatorVideoCallsByPage[page]++
			switch page {
			case "1":
				writeJSON(t, w, bilibiliCreatorVideosPagePayload(1, 2, 23201, "BVcreatorResume1"))
			case "2":
				if !allowSecondPage {
					http.Error(w, "transient second creator page failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, bilibiliCreatorVideosPagePayload(2, 2, 23202, "BVcreatorResume2"))
			default:
				writeJSON(t, w, map[string]any{
					"code":    0,
					"message": "0",
					"data": map[string]any{
						"list": map[string]any{"vlist": []map[string]any{}},
						"page": map[string]any{"pn": page, "ps": 30, "count": 2},
					},
				})
			}
		case "/x/web-interface/view/detail":
			bvid := r.URL.Query().Get("bvid")
			viewCallsByBvid[bvid]++
			switch bvid {
			case "BVcreatorResume1":
				writeJSON(t, w, bilibiliResumeDetailPayload(23201, bvid, "Bilibili creator resume video 1"))
			case "BVcreatorResume2":
				writeJSON(t, w, bilibiliResumeDetailPayload(23202, bvid, "Bilibili creator resume video 2"))
			default:
				t.Fatalf("unexpected creator resume detail bvid: %s", bvid)
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
platform: "bili"
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
specifiedCreatorIds:
  - "777"
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  bilibiliApiUrl: %q
  bilibiliIndexUrl: %q
  bilibiliSpaceUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL, mockServer.URL)
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
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("bilibili creator resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("bilibili creator resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if creatorInfoCalls != 1 || creatorVideoCallsByPage["1"] != 1 || viewCallsByBvid["BVcreatorResume1"] != 1 {
		t.Fatalf("first creator run should fetch profile and page 1 once, profile=%d pages=%#v view=%#v\n%s", creatorInfoCalls, creatorVideoCallsByPage, viewCallsByBvid, firstOutput)
	}
	if creatorVideoCallsByPage["2"] != 1 {
		t.Fatalf("first creator run should stop after one failed page 2 attempt, pages=%#v\n%s", creatorVideoCallsByPage, firstOutput)
	}

	allowSecondPage = true
	secondOutput := runCrawler()
	if creatorInfoCalls != 2 {
		t.Fatalf("Bilibili creator resume should match Python and refresh creator info, profile=%d\n%s", creatorInfoCalls, secondOutput)
	}
	if creatorVideoCallsByPage["1"] != 1 {
		t.Fatalf("resume should not re-fetch completed creator page 1, pages=%#v\n%s", creatorVideoCallsByPage, secondOutput)
	}
	if creatorVideoCallsByPage["2"] != 2 || viewCallsByBvid["BVcreatorResume2"] != 1 {
		t.Fatalf("resume did not continue from creator page 2 exactly once, pages=%#v view=%#v\n%s", creatorVideoCallsByPage, viewCallsByBvid, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "bilibili", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Bilibili creator content rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["video_id"])] = true
	}
	if !seen["23201"] || !seen["23202"] {
		t.Fatalf("creator resume output missing expected Bilibili video IDs, seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIBilibiliDetailCheckpointResumeSkipsCompletedIDs(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	viewCallsByBvid := make(map[string]int)
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
							"client_id": "bili-detail-resume-client",
							"platforms": map[string]any{
								"bili": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"bili": "mock-bili"},
						},
					},
				},
			})
		case "/api/cookies/bili":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "SESSDATA=detail-resume-session; buvid3=detail-resume-buvid"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/bilibili/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"wts": "1710000000", "w_rid": "detail-resume-wrid"},
			})
		case "/x/web-interface/view/detail":
			bvid := r.URL.Query().Get("bvid")
			viewCallsByBvid[bvid]++
			switch bvid {
			case "BVdetailResume1":
				writeJSON(t, w, bilibiliResumeDetailPayload(22001, bvid, "Bilibili detail resume video 1"))
			case "BVdetailResume2":
				if !allowSecondDetail {
					http.Error(w, "transient second detail failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, bilibiliResumeDetailPayload(22002, bvid, "Bilibili detail resume video 2"))
			default:
				t.Fatalf("unexpected detail resume bvid: %s", bvid)
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
platform: "bili"
keywords:
  - "mock-keyword"
crawlerType: "detail"
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
specifiedNoteIds:
  - "BVdetailResume1"
  - "BVdetailResume2"
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  bilibiliApiUrl: %q
  bilibiliIndexUrl: %q
  bilibiliSpaceUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL, mockServer.URL)
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
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("bilibili detail resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("bilibili detail resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if viewCallsByBvid["BVdetailResume1"] != 1 || viewCallsByBvid["BVdetailResume2"] != 1 {
		t.Fatalf("first bilibili detail run should fetch both IDs and fail second once, calls=%#v\n%s", viewCallsByBvid, firstOutput)
	}

	allowSecondDetail = true
	secondOutput := runCrawler()
	if viewCallsByBvid["BVdetailResume1"] != 1 {
		t.Fatalf("resume should skip completed first bilibili detail ID, calls=%#v\n%s", viewCallsByBvid, secondOutput)
	}
	if viewCallsByBvid["BVdetailResume2"] != 2 {
		t.Fatalf("resume should retry second bilibili detail ID exactly once, calls=%#v\n%s", viewCallsByBvid, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "bilibili", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Bilibili detail rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["video_id"])] = true
	}
	if !seen["22001"] || !seen["22002"] {
		t.Fatalf("bilibili detail resume output missing expected video IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIBilibiliHotSearchCheckpointResumeUsesSnapshotAndSkipsCompletedVideo(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	popularCalls := 0
	viewCallsByBvid := make(map[string]int)
	allowSecondVideo := false
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
							"client_id": "bili-hot-resume-client",
							"platforms": map[string]any{
								"bili": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"bili": "mock-bili"},
						},
					},
				},
			})
		case "/api/cookies/bili":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "SESSDATA=resume-session; buvid3=resume-buvid"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/bilibili/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"wts": "1710000000", "w_rid": "resume-wrid"},
			})
		case "/x/web-interface/popular":
			popularCalls++
			if r.URL.Query().Get("pn") != "1" || r.URL.Query().Get("ps") != "2" {
				t.Fatalf("unexpected popular query: %s", r.URL.RawQuery)
			}
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data": map[string]any{
					"list": []map[string]any{
						{"aid": int64(21001), "bvid": "BVhotresume1", "title": "hot resume title 1"},
						{"aid": int64(21002), "bvid": "BVhotresume2", "title": "hot resume title 2"},
					},
				},
			})
		case "/x/web-interface/view/detail":
			bvid := r.URL.Query().Get("bvid")
			viewCallsByBvid[bvid]++
			if bvid == "BVhotresume2" && !allowSecondVideo {
				http.Error(w, "transient second hot video failure", http.StatusInternalServerError)
				return
			}
			switch bvid {
			case "BVhotresume1":
				writeJSON(t, w, bilibiliResumeDetailPayload(21001, bvid, "Bilibili hot resume video 1"))
			case "BVhotresume2":
				writeJSON(t, w, bilibiliResumeDetailPayload(21002, bvid, "Bilibili hot resume video 2"))
			default:
				t.Fatalf("unexpected hot resume detail bvid: %s", bvid)
			}
		default:
			t.Fatalf("unexpected bilibili hot resume request: %s %s", r.Method, r.URL.String())
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
platform: "bili"
keywords:
  - "unused"
crawlerType: "hot_search"
saveDataOption: "json"
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
startPage: 1
crawlerMaxNotesCount: 2
hotSearchMaxItemsCount: 2
hotSearchPerItemContentCount: 1
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
  bilibiliApiUrl: %q
  bilibiliIndexUrl: %q
  bilibiliSpaceUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL, mockServer.URL)
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
			t.Fatalf("bilibili hot resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("bilibili hot resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if popularCalls != 1 || viewCallsByBvid["BVhotresume1"] != 1 || viewCallsByBvid["BVhotresume2"] != 1 {
		t.Fatalf("first bilibili hot_search run should fetch rank and fail second detail once; popular=%d detail=%#v\n%s", popularCalls, viewCallsByBvid, firstOutput)
	}

	allowSecondVideo = true
	secondOutput := runCrawler()
	if popularCalls != 1 {
		t.Fatalf("resume should reuse bilibili hot_search snapshot and not refetch popular list, got %d\n%s", popularCalls, secondOutput)
	}
	if viewCallsByBvid["BVhotresume1"] != 1 {
		t.Fatalf("resume should skip completed first hot video, detail=%#v\n%s", viewCallsByBvid, secondOutput)
	}
	if viewCallsByBvid["BVhotresume2"] != 2 {
		t.Fatalf("resume should retry second hot video exactly once, detail=%#v\n%s", viewCallsByBvid, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "bilibili", "json", "hot_search_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Bilibili hot_search rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["bvid"])] = true
	}
	if !seen["BVhotresume1"] || !seen["BVhotresume2"] {
		t.Fatalf("bilibili hot_search resume output missing expected bvids: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIBilibiliSearchJSONCommentsWithMockServers(t *testing.T) {
	result := runCLIBilibiliSearchWithMockServers(t, "json", true)
	if result.commentCalls != 1 {
		t.Fatalf("expected one root comment request, got %d\n%s", result.commentCalls, result.output)
	}
	if result.subCommentCalls != 1 {
		t.Fatalf("expected one sub-comment request, got %d\n%s", result.subCommentCalls, result.output)
	}

	resultPath := filepath.Join(result.dataDir, "bilibili", "json", "search_comments_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read comments file: %v\ncommand output:\n%s", err, result.output)
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
	root := comments["9001"]
	if root["video_id"] != "12345" {
		t.Fatalf("unexpected root video_id: %#v", root["video_id"])
	}
	if root["content"] != "root bili comment" {
		t.Fatalf("unexpected root content: %#v", root["content"])
	}
	if root["create_time"] != "1710000100" {
		t.Fatalf("unexpected root create_time: %#v", root["create_time"])
	}
	if root["sub_comment_count"] != "1" {
		t.Fatalf("unexpected root sub_comment_count: %#v", root["sub_comment_count"])
	}
	if root["like_count"] != "4" {
		t.Fatalf("unexpected root like_count: %#v", root["like_count"])
	}
	if root["parent_comment_id"] != "" {
		t.Fatalf("unexpected root parent_comment_id: %#v", root["parent_comment_id"])
	}
	if root["user_id"] != "8801" || root["nickname"] != "bili root commenter" || root["avatar"] != "https://example.test/bili-root.png" {
		t.Fatalf("unexpected root user fields: %#v", root)
	}
	if value, ok := root["add_ts"]; !ok || value != nil {
		t.Fatalf("expected Python-compatible root add_ts:null, got present=%t value=%#v", ok, value)
	}

	sub := comments["9002"]
	if sub["video_id"] != "12345" {
		t.Fatalf("unexpected sub video_id: %#v", sub["video_id"])
	}
	if sub["content"] != "sub bili comment" {
		t.Fatalf("unexpected sub content: %#v", sub["content"])
	}
	if sub["create_time"] != "1710000200" {
		t.Fatalf("unexpected sub create_time: %#v", sub["create_time"])
	}
	if sub["sub_comment_count"] != "0" {
		t.Fatalf("unexpected sub sub_comment_count: %#v", sub["sub_comment_count"])
	}
	if sub["like_count"] != "2" {
		t.Fatalf("unexpected sub like_count: %#v", sub["like_count"])
	}
	if sub["parent_comment_id"] != "9001" {
		t.Fatalf("unexpected sub parent_comment_id: %#v", sub["parent_comment_id"])
	}
	if sub["user_id"] != "8802" || sub["nickname"] != "bili sub commenter" || sub["avatar"] != "https://example.test/bili-sub.png" {
		t.Fatalf("unexpected sub user fields: %#v", sub)
	}
	if value, ok := sub["add_ts"]; !ok || value != nil {
		t.Fatalf("expected Python-compatible sub add_ts:null, got present=%t value=%#v", ok, value)
	}
}

func TestCLIBilibiliSearchCSVCommentsWithMockServers(t *testing.T) {
	result := runCLIBilibiliSearchWithMockServers(t, "csv", true)
	if result.commentCalls != 1 {
		t.Fatalf("expected one root comment request, got %d\n%s", result.commentCalls, result.output)
	}
	if result.subCommentCalls != 1 {
		t.Fatalf("expected one sub-comment request, got %d\n%s", result.subCommentCalls, result.output)
	}

	resultPath := filepath.Join(result.dataDir, "bilibili", "1_search_comments_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open comments csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read comments csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 3 {
		t.Fatalf("expected csv header and two comment rows, got %d: %#v", len(records), records)
	}
	comments := make(map[string]map[string]string, len(records)-1)
	for _, record := range records[1:] {
		row := make(map[string]string, len(records[0]))
		for idx, header := range records[0] {
			row[strings.TrimPrefix(header, "\ufeff")] = record[idx]
		}
		comments[row["comment_id"]] = row
	}

	root := comments["9001"]
	if root["video_id"] != "12345" || root["content"] != "root bili comment" {
		t.Fatalf("unexpected root csv row: %#v", root)
	}
	if root["create_time"] != "1710000100" || root["sub_comment_count"] != "1" || root["like_count"] != "4" {
		t.Fatalf("unexpected root csv count fields: %#v", root)
	}
	if root["parent_comment_id"] != "" || root["add_ts"] != "" {
		t.Fatalf("unexpected root csv parent/add_ts fields: %#v", root)
	}
	if root["user_id"] != "8801" || root["nickname"] != "bili root commenter" || root["avatar"] != "https://example.test/bili-root.png" {
		t.Fatalf("unexpected root csv user fields: %#v", root)
	}

	sub := comments["9002"]
	if sub["video_id"] != "12345" || sub["content"] != "sub bili comment" {
		t.Fatalf("unexpected sub csv row: %#v", sub)
	}
	if sub["create_time"] != "1710000200" || sub["sub_comment_count"] != "0" || sub["like_count"] != "2" {
		t.Fatalf("unexpected sub csv count fields: %#v", sub)
	}
	if sub["parent_comment_id"] != "9001" || sub["add_ts"] != "" {
		t.Fatalf("unexpected sub csv parent/add_ts fields: %#v", sub)
	}
	if sub["user_id"] != "8802" || sub["nickname"] != "bili sub commenter" || sub["avatar"] != "https://example.test/bili-sub.png" {
		t.Fatalf("unexpected sub csv user fields: %#v", sub)
	}
}

func TestCLIBilibiliSearchSQLiteCommentsWithMockServers(t *testing.T) {
	result := runCLIBilibiliSearchWithMockServers(t, "sqlite", true)
	if result.commentCalls != 1 {
		t.Fatalf("expected one root comment request, got %d\n%s", result.commentCalls, result.output)
	}
	if result.subCommentCalls != 1 {
		t.Fatalf("expected one sub-comment request, got %d\n%s", result.subCommentCalls, result.output)
	}

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []struct {
		CommentID       string `gorm:"column:comment_id"`
		VideoID         string `gorm:"column:video_id"`
		Content         string `gorm:"column:content"`
		CreateTime      int64  `gorm:"column:create_time"`
		SubCommentCount string `gorm:"column:sub_comment_count"`
		LikeCount       string `gorm:"column:like_count"`
		ParentCommentID string `gorm:"column:parent_comment_id"`
		UserID          string `gorm:"column:user_id"`
		Nickname        string `gorm:"column:nickname"`
		Avatar          string `gorm:"column:avatar"`
		AddTs           int64  `gorm:"column:add_ts"`
		LastModifyTs    int64  `gorm:"column:last_modify_ts"`
	}
	if err := db.Table("bilibili_video_comment").Order("comment_id").Find(&rows).Error; err != nil {
		t.Fatalf("query sqlite comments result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two sqlite comment rows, got %d: %#v", len(rows), rows)
	}

	root := rows[0]
	if root.CommentID != "9001" || root.VideoID != "12345" || root.Content != "root bili comment" {
		t.Fatalf("unexpected root sqlite row: %#v", root)
	}
	if root.CreateTime != 1710000100 || root.SubCommentCount != "1" || root.LikeCount != "4" {
		t.Fatalf("unexpected root sqlite count fields: %#v", root)
	}
	if root.ParentCommentID != "" || root.UserID != "8801" || root.Nickname != "bili root commenter" || root.Avatar != "https://example.test/bili-root.png" {
		t.Fatalf("unexpected root sqlite relation/user fields: %#v", root)
	}
	if root.AddTs == 0 || root.LastModifyTs == 0 {
		t.Fatalf("expected non-zero root sqlite timestamps: %#v", root)
	}

	sub := rows[1]
	if sub.CommentID != "9002" || sub.VideoID != "12345" || sub.Content != "sub bili comment" {
		t.Fatalf("unexpected sub sqlite row: %#v", sub)
	}
	if sub.CreateTime != 1710000200 || sub.SubCommentCount != "0" || sub.LikeCount != "2" {
		t.Fatalf("unexpected sub sqlite count fields: %#v", sub)
	}
	if sub.ParentCommentID != "9001" || sub.UserID != "8802" || sub.Nickname != "bili sub commenter" || sub.Avatar != "https://example.test/bili-sub.png" {
		t.Fatalf("unexpected sub sqlite relation/user fields: %#v", sub)
	}
	if sub.AddTs == 0 || sub.LastModifyTs == 0 {
		t.Fatalf("expected non-zero sub sqlite timestamps: %#v", sub)
	}
}

func TestCLIBilibiliSearchCSVWithMockServers(t *testing.T) {
	result := runCLIBilibiliSearchWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "bilibili", "1_search_contents_"+time.Now().Format("2006-01-02")+".csv")
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
	if csvCell(records, "video_id") != "12345" {
		t.Fatalf("unexpected csv video_id: %#v", csvCell(records, "video_id"))
	}
	if csvCell(records, "bvid") != "BV1mockbili" {
		t.Fatalf("unexpected csv bvid: %#v", csvCell(records, "bvid"))
	}
	if csvCell(records, "create_time") != "1710000000" {
		t.Fatalf("unexpected csv create_time: %#v", csvCell(records, "create_time"))
	}
	if csvCell(records, "duration") != "66" {
		t.Fatalf("unexpected csv duration: %#v", csvCell(records, "duration"))
	}
	if csvCell(records, "video_cover_url") != "https://example.test/bili-cover.png" {
		t.Fatalf("unexpected csv video_cover_url: %#v", csvCell(records, "video_cover_url"))
	}
	if csvCell(records, "source_keyword") != "mock-keyword" {
		t.Fatalf("unexpected csv source_keyword: %#v", csvCell(records, "source_keyword"))
	}
	if csvCell(records, "add_ts") != "" {
		t.Fatalf("expected Python-compatible blank csv add_ts, got %#v", csvCell(records, "add_ts"))
	}
}

func TestCLIBilibiliSearchSQLiteWithMockServers(t *testing.T) {
	result := runCLIBilibiliSearchWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	var row struct {
		VideoID        string `gorm:"column:video_id"`
		Bvid           string `gorm:"column:bvid"`
		VideoType      string `gorm:"column:video_type"`
		Title          string `gorm:"column:title"`
		Desc           string `gorm:"column:desc"`
		CreateTime     int64  `gorm:"column:create_time"`
		Duration       string `gorm:"column:duration"`
		LikedCount     string `gorm:"column:liked_count"`
		VideoPlayCount string `gorm:"column:video_play_count"`
		VideoDanmaku   string `gorm:"column:video_danmaku"`
		VideoComment   string `gorm:"column:video_comment"`
		VideoURL       string `gorm:"column:video_url"`
		VideoCoverURL  string `gorm:"column:video_cover_url"`
		UserID         string `gorm:"column:user_id"`
		Nickname       string `gorm:"column:nickname"`
		Avatar         string `gorm:"column:avatar"`
		SourceKeyword  string `gorm:"column:source_keyword"`
		AddTs          int64  `gorm:"column:add_ts"`
		LastModifyTs   int64  `gorm:"column:last_modify_ts"`
	}
	if err := db.Table("bilibili_video").Where("video_id = ?", "12345").First(&row).Error; err != nil {
		t.Fatalf("query sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	if row.Bvid != "BV1mockbili" {
		t.Fatalf("unexpected sqlite bvid: %#v", row.Bvid)
	}
	if row.VideoType != "科技" {
		t.Fatalf("unexpected sqlite video_type: %#v", row.VideoType)
	}
	if row.Title != "Bilibili E2E mock video" {
		t.Fatalf("unexpected sqlite title: %#v", row.Title)
	}
	if row.Desc != "mock bilibili desc" {
		t.Fatalf("unexpected sqlite desc: %#v", row.Desc)
	}
	if row.CreateTime != 1710000000 {
		t.Fatalf("unexpected sqlite create_time: %#v", row.CreateTime)
	}
	if row.Duration != "66" {
		t.Fatalf("unexpected sqlite duration: %#v", row.Duration)
	}
	if row.LikedCount != "88" {
		t.Fatalf("unexpected sqlite liked_count: %#v", row.LikedCount)
	}
	if row.VideoPlayCount != "1000" {
		t.Fatalf("unexpected sqlite video_play_count: %#v", row.VideoPlayCount)
	}
	if row.VideoDanmaku != "12" {
		t.Fatalf("unexpected sqlite video_danmaku: %#v", row.VideoDanmaku)
	}
	if row.VideoComment != "5" {
		t.Fatalf("unexpected sqlite video_comment: %#v", row.VideoComment)
	}
	if row.VideoURL != "https://www.bilibili.com/video/BV1mockbili" {
		t.Fatalf("unexpected sqlite video_url: %#v", row.VideoURL)
	}
	if row.VideoCoverURL != "https://example.test/bili-cover.png" {
		t.Fatalf("unexpected sqlite video_cover_url: %#v", row.VideoCoverURL)
	}
	if row.UserID != "777" {
		t.Fatalf("unexpected sqlite user_id: %#v", row.UserID)
	}
	if row.Nickname != "mock bili author" {
		t.Fatalf("unexpected sqlite nickname: %#v", row.Nickname)
	}
	if row.Avatar != "https://example.test/bili-avatar.png" {
		t.Fatalf("unexpected sqlite avatar: %#v", row.Avatar)
	}
	if row.SourceKeyword != "mock-keyword" {
		t.Fatalf("unexpected sqlite source_keyword: %#v", row.SourceKeyword)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected non-zero sqlite timestamps, got add_ts=%d last_modify_ts=%d", row.AddTs, row.LastModifyTs)
	}
}

func bilibiliResumeSearchPayload(aid int64, bvid string) map[string]any {
	return map[string]any{
		"code":    0,
		"message": "0",
		"data": map[string]any{
			"numResults": 1,
			"page":       1,
			"pagesize":   20,
			"result": []map[string]any{
				{
					"type":        "video",
					"aid":         aid,
					"bvid":        bvid,
					"title":       "resume search title",
					"description": "resume search desc",
					"author":      "resume author",
					"mid":         int64(777),
					"play":        int64(100),
					"like":        int64(9),
					"pubdate":     int64(1710000000),
				},
			},
		},
	}
}

func bilibiliHomefeedPagePayload(aid int64, bvid string) map[string]any {
	return bilibiliHomefeedItemsPayload(bilibiliHomefeedItem(aid, bvid))
}

func bilibiliHomefeedItemsPayload(items ...map[string]any) map[string]any {
	return map[string]any{
		"code":    0,
		"message": "0",
		"data": map[string]any{
			"item": items,
		},
	}
}

func bilibiliHomefeedItem(aid int64, bvid string) map[string]any {
	return map[string]any{
		"id":        aid,
		"bvid":      bvid,
		"goto":      "av",
		"show_info": 1,
		"title":     "homefeed resume title",
	}
}

func bilibiliCreatorVideosPagePayload(page int, totalCount int, aid int64, bvid string) map[string]any {
	return map[string]any{
		"code":    0,
		"message": "0",
		"data": map[string]any{
			"list": map[string]any{
				"vlist": []map[string]any{
					{
						"aid":         aid,
						"bvid":        bvid,
						"title":       "creator resume title",
						"description": "creator resume desc",
						"play":        int64(1000),
						"comment":     int64(5),
						"created":     int64(1710000000),
						"pic":         "https://example.test/" + bvid + ".png",
						"length":      "01:06",
						"author":      "resume bili author",
						"mid":         int64(777),
					},
				},
			},
			"page": map[string]any{"pn": page, "ps": 30, "count": totalCount},
		},
	}
}

func bilibiliResumeDetailPayload(aid int64, bvid string, title string) map[string]any {
	return map[string]any{
		"code":    0,
		"message": "0",
		"data": map[string]any{
			"View": map[string]any{
				"bvid":     bvid,
				"aid":      aid,
				"tname":    "科技",
				"pic":      "https://example.test/" + bvid + ".png",
				"title":    title,
				"pubdate":  int64(1710000000),
				"ctime":    int64(1710000000),
				"desc":     "resume bilibili desc",
				"duration": int64(66),
				"owner": map[string]any{
					"mid":  int64(777),
					"name": "resume bili author",
					"face": "https://example.test/bili-avatar.png",
				},
				"stat": map[string]any{
					"aid":     aid,
					"view":    int64(1000),
					"danmaku": int64(12),
					"reply":   int64(5),
					"like":    int64(88),
				},
			},
		},
	}
}

type cliBilibiliSearchResult struct {
	dataDir         string
	sqlitePath      string
	output          []byte
	commentCalls    int
	subCommentCalls int
}

func runCLIBilibiliSearchWithMockServers(t *testing.T, saveDataOption string, enableComments ...bool) cliBilibiliSearchResult {
	t.Helper()
	commentsEnabled := len(enableComments) > 0 && enableComments[0]
	repoRoot := e2e.RepoRoot(t)

	var searchCalls, viewCalls, commentCalls, subCommentCalls int
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
							"client_id": "bili-mock-client",
							"platforms": map[string]any{
								"bili": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"bili": "mock-bili"},
						},
					},
				},
			})
		case "/api/cookies/bili":
			if got := r.URL.Query().Get("client_id"); got != "bili-mock-client" {
				t.Fatalf("unexpected client_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "SESSDATA=mock-session; buvid3=mock-buvid"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/bilibili/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"wts":   "1710000000",
					"w_rid": "mock-wrid",
				},
			})
		case "/x/web-interface/wbi/search/type":
			searchCalls++
			if r.URL.Query().Get("page") != "1" {
				writeJSON(t, w, map[string]any{
					"code":    0,
					"message": "0",
					"data": map[string]any{
						"numResults": 0,
						"page":       2,
						"pagesize":   20,
						"result":     []map[string]any{},
					},
				})
				return
			}
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data": map[string]any{
					"numResults": 1,
					"page":       1,
					"pagesize":   20,
					"result": []map[string]any{
						{
							"type":        "video",
							"aid":         int64(12345),
							"bvid":        "BV1mockbili",
							"title":       "mock search title",
							"description": "mock search desc",
							"author":      "mock author",
							"mid":         int64(777),
							"play":        int64(100),
							"like":        int64(9),
							"pubdate":     int64(1710000000),
						},
					},
				},
			})
		case "/x/web-interface/view/detail":
			viewCalls++
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data": map[string]any{
					"View": map[string]any{
						"bvid":     "BV1mockbili",
						"aid":      int64(12345),
						"tname":    "科技",
						"pic":      "https://example.test/bili-cover.png",
						"title":    "Bilibili E2E mock video",
						"pubdate":  int64(1710000000),
						"ctime":    int64(1710000000),
						"desc":     "mock bilibili desc",
						"duration": int64(66),
						"owner": map[string]any{
							"mid":  int64(777),
							"name": "mock bili author",
							"face": "https://example.test/bili-avatar.png",
						},
						"stat": map[string]any{
							"aid":      int64(12345),
							"view":     int64(1000),
							"danmaku":  int64(12),
							"reply":    int64(5),
							"favorite": int64(20),
							"coin":     int64(7),
							"share":    int64(3),
							"like":     int64(88),
						},
					},
				},
			})
		case "/x/v2/reply/wbi/main":
			commentCalls++
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data": map[string]any{
					"cursor": map[string]any{
						"all_count": 1,
						"is_begin":  true,
						"is_end":    true,
						"mode":      3,
						"next":      0,
					},
					"replies": []map[string]any{
						{
							"rpid":   int64(9001),
							"oid":    int64(12345),
							"mid":    int64(8801),
							"root":   int64(0),
							"parent": int64(0),
							"count":  1,
							"rcount": 1,
							"like":   4,
							"ctime":  int64(1710000100),
							"content": map[string]string{
								"message": "root bili comment",
							},
							"member": map[string]string{
								"mid":    "8801",
								"uname":  "bili root commenter",
								"avatar": "https://example.test/bili-root.png",
							},
							"replies": []map[string]any{},
						},
					},
				},
			})
		case "/x/v2/reply/reply":
			subCommentCalls++
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data": map[string]any{
					"page": map[string]any{"count": 1},
					"replies": []map[string]any{
						{
							"rpid":   int64(9002),
							"oid":    int64(12345),
							"mid":    int64(8802),
							"root":   int64(9001),
							"parent": int64(9001),
							"count":  0,
							"rcount": 0,
							"like":   2,
							"ctime":  int64(1710000200),
							"content": map[string]string{
								"message": "sub bili comment",
							},
							"member": map[string]string{
								"mid":    "8802",
								"uname":  "bili sub commenter",
								"avatar": "https://example.test/bili-sub.png",
							},
							"replies": []map[string]any{},
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
platform: "bili"
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
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  bilibiliApiUrl: %q
  bilibiliIndexUrl: %q
  bilibiliSpaceUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, commentsEnabled, commentsEnabled, host, port, mockServer.URL, mockServer.URL, mockServer.URL)
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
	if viewCalls != 1 {
		t.Fatalf("expected one view request, got %d\n%s", viewCalls, output)
	}
	return cliBilibiliSearchResult{
		dataDir:         dataDir,
		sqlitePath:      sqlitePath,
		output:          output,
		commentCalls:    commentCalls,
		subCommentCalls: subCommentCalls,
	}
}

func runCLIBilibiliHotSearchWithMockServers(t *testing.T, saveDataOption string, enableComments ...bool) cliBilibiliSearchResult {
	t.Helper()
	commentsEnabled := len(enableComments) > 0 && enableComments[0]
	repoRoot := e2e.RepoRoot(t)

	var popularCalls, viewCalls, commentCalls, subCommentCalls int
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
							"client_id": "bili-hot-client",
							"platforms": map[string]any{
								"bili": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"bili": "mock-bili"},
						},
					},
				},
			})
		case "/api/cookies/bili":
			if got := r.URL.Query().Get("client_id"); got != "bili-hot-client" {
				t.Fatalf("unexpected client_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "SESSDATA=mock-session; buvid3=mock-buvid"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/bilibili/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"wts":   "1710000000",
					"w_rid": "mock-wrid",
				},
			})
		case "/x/web-interface/popular":
			popularCalls++
			if r.URL.Query().Get("pn") != "1" || r.URL.Query().Get("ps") != "1" {
				t.Fatalf("unexpected popular query: %s", r.URL.RawQuery)
			}
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data": map[string]any{
					"list": []map[string]any{
						{
							"aid":   int64(12345),
							"bvid":  "BV1mockbili",
							"title": "mock popular title",
							"owner": map[string]any{
								"mid":  int64(777),
								"name": "mock bili author",
								"face": "https://example.test/bili-avatar.png",
							},
							"stat": map[string]any{
								"view": int64(1000),
							},
						},
						{
							"aid":   int64(67890),
							"bvid":  "BV1mockbili2",
							"title": "mock popular title 2",
							"stat": map[string]any{
								"view": int64(2000),
							},
						},
					},
				},
			})
		case "/x/web-interface/view/detail":
			viewCalls++
			switch r.URL.Query().Get("bvid") {
			case "BV1mockbili":
				writeJSON(t, w, map[string]any{
					"code":    0,
					"message": "0",
					"data": map[string]any{
						"View": map[string]any{
							"bvid":     "BV1mockbili",
							"aid":      int64(12345),
							"tname":    "科技",
							"pic":      "https://example.test/bili-cover.png",
							"title":    "Bilibili E2E mock video",
							"pubdate":  int64(1710000000),
							"ctime":    int64(1710000000),
							"desc":     "mock bilibili desc",
							"duration": int64(66),
							"owner": map[string]any{
								"mid":  int64(777),
								"name": "mock bili author",
								"face": "https://example.test/bili-avatar.png",
							},
							"stat": map[string]any{
								"aid":      int64(12345),
								"view":     int64(1000),
								"danmaku":  int64(12),
								"reply":    int64(5),
								"favorite": int64(20),
								"coin":     int64(7),
								"share":    int64(3),
								"like":     int64(88),
							},
						},
					},
				})
			case "BV1mockbili2":
				writeJSON(t, w, bilibiliResumeDetailPayload(67890, "BV1mockbili2", "Bilibili E2E mock video 2"))
			default:
				t.Fatalf("unexpected detail bvid: %s", r.URL.Query().Get("bvid"))
			}
		case "/x/v2/reply/wbi/main":
			commentCalls++
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data": map[string]any{
					"cursor": map[string]any{
						"all_count": 1,
						"is_begin":  true,
						"is_end":    true,
						"mode":      3,
						"next":      0,
					},
					"replies": []map[string]any{
						{
							"rpid":   int64(9001),
							"oid":    int64(12345),
							"mid":    int64(8801),
							"root":   int64(0),
							"parent": int64(0),
							"count":  1,
							"rcount": 1,
							"like":   4,
							"ctime":  int64(1710000100),
							"content": map[string]string{
								"message": "root bili comment",
							},
							"member": map[string]string{
								"mid":    "8801",
								"uname":  "bili root commenter",
								"avatar": "https://example.test/bili-root.png",
							},
							"replies": []map[string]any{},
						},
					},
				},
			})
		case "/x/v2/reply/reply":
			subCommentCalls++
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data": map[string]any{
					"page": map[string]any{"count": 1},
					"replies": []map[string]any{
						{
							"rpid":   int64(9002),
							"oid":    int64(12345),
							"mid":    int64(8802),
							"root":   int64(9001),
							"parent": int64(9001),
							"count":  0,
							"rcount": 0,
							"like":   2,
							"ctime":  int64(1710000200),
							"content": map[string]string{
								"message": "sub bili comment",
							},
							"member": map[string]string{
								"mid":    "8802",
								"uname":  "bili sub commenter",
								"avatar": "https://example.test/bili-sub.png",
							},
							"replies": []map[string]any{},
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
platform: "bili"
keywords:
  - "unused"
crawlerType: "hot_search"
saveDataOption: %q
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
sqliteDbPath: %q
startPage: 1
crawlerMaxNotesCount: 20
hotSearchMaxItemsCount: 1
hotSearchPerItemContentCount: 4
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
  bilibiliApiUrl: %q
  bilibiliIndexUrl: %q
  bilibiliSpaceUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, commentsEnabled, commentsEnabled, host, port, mockServer.URL, mockServer.URL, mockServer.URL)
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
	if popularCalls != 1 {
		t.Fatalf("expected one popular request, got %d\n%s", popularCalls, output)
	}
	if viewCalls != 1 {
		t.Fatalf("expected one detail request after hot_search max item truncation, got %d\n%s", viewCalls, output)
	}
	return cliBilibiliSearchResult{
		dataDir:         dataDir,
		sqlitePath:      sqlitePath,
		output:          output,
		commentCalls:    commentCalls,
		subCommentCalls: subCommentCalls,
	}
}
