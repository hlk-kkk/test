// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/kuaishou_e2e_test.go
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
	"io"
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

func TestCLIKuaishouSearchJSONWithMockServers(t *testing.T) {
	result := runCLIKuaishouSearchWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "kuaishou", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read kuaishou result file: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal kuaishou result json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one saved kuaishou row, got %d: %s", len(rows), body)
	}
	row := rows[0]
	if row["video_id"] != "ks_video_e2e" {
		t.Fatalf("unexpected video_id: %#v", row["video_id"])
	}
	if row["video_type"] != "photo" {
		t.Fatalf("unexpected video_type: %#v", row["video_type"])
	}
	if row["title"] != "Kuaishou E2E mock video" || row["desc"] != "Kuaishou E2E mock video" {
		t.Fatalf("unexpected title/desc: %#v", row)
	}
	if row["create_time"] != "1710000700" {
		t.Fatalf("unexpected create_time: %#v", row["create_time"])
	}
	if row["liked_count"] != "66" || row["viewd_count"] != "777" {
		t.Fatalf("unexpected count fields: %#v", row)
	}
	if row["video_url"] != "https://www.kuaishou.com/short-video/ks_video_e2e" {
		t.Fatalf("unexpected video_url: %#v", row["video_url"])
	}
	if row["video_cover_url"] != "https://example.test/ks-cover.png" {
		t.Fatalf("unexpected video_cover_url: %#v", row["video_cover_url"])
	}
	if row["video_play_url"] != "https://example.test/ks-play.mp4" {
		t.Fatalf("unexpected video_play_url: %#v", row["video_play_url"])
	}
	if row["source_keyword"] != "mock-ks" {
		t.Fatalf("unexpected source_keyword: %#v", row["source_keyword"])
	}
	if row["user_id"] != "ks-author" || row["nickname"] != "mock kuaishou author" || row["avatar"] != "https://example.test/ks-avatar.png" {
		t.Fatalf("unexpected author fields: %#v", row)
	}
	if _, ok := row["add_ts"]; ok {
		t.Fatalf("kuaishou json output should match Python and omit add_ts: %#v", row["add_ts"])
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected last_modify_ts in kuaishou json row: %#v", row)
	}
}

func TestCLIKuaishouSearchCSVWithMockServers(t *testing.T) {
	result := runCLIKuaishouSearchWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "kuaishou", "1_search_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open kuaishou csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read kuaishou csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected csv header and one kuaishou row, got %d: %#v", len(records), records)
	}
	if csvCell(records, "video_id") != "ks_video_e2e" {
		t.Fatalf("unexpected csv video_id: %#v", csvCell(records, "video_id"))
	}
	if csvCell(records, "video_type") != "photo" {
		t.Fatalf("unexpected csv video_type: %#v", csvCell(records, "video_type"))
	}
	if csvCell(records, "title") != "Kuaishou E2E mock video" || csvCell(records, "desc") != "Kuaishou E2E mock video" {
		t.Fatalf("unexpected csv title/desc: %#v", records)
	}
	if csvCell(records, "create_time") != "1710000700" {
		t.Fatalf("unexpected csv create_time: %#v", csvCell(records, "create_time"))
	}
	if csvCell(records, "liked_count") != "66" || csvCell(records, "viewd_count") != "777" {
		t.Fatalf("unexpected csv count fields: %#v", records)
	}
	if csvCell(records, "video_url") != "https://www.kuaishou.com/short-video/ks_video_e2e" {
		t.Fatalf("unexpected csv video_url: %#v", csvCell(records, "video_url"))
	}
	if csvCell(records, "video_cover_url") != "https://example.test/ks-cover.png" || csvCell(records, "video_play_url") != "https://example.test/ks-play.mp4" {
		t.Fatalf("unexpected csv media fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "mock-ks" {
		t.Fatalf("unexpected csv source_keyword: %#v", csvCell(records, "source_keyword"))
	}
	if csvCell(records, "user_id") != "ks-author" || csvCell(records, "nickname") != "mock kuaishou author" || csvCell(records, "avatar") != "https://example.test/ks-avatar.png" {
		t.Fatalf("unexpected csv author fields: %#v", records)
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected non-empty csv last_modify_ts")
	}
}

func TestCLIKuaishouSearchSQLiteWithMockServers(t *testing.T) {
	result := runCLIKuaishouSearchWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open kuaishou sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []kuaishouSQLiteVideo
	if err := db.Table("kuaishou_video").Find(&rows).Error; err != nil {
		t.Fatalf("query kuaishou sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one kuaishou sqlite row, got %d: %#v", len(rows), rows)
	}
	row := rows[0]
	if row.VideoID != "ks_video_e2e" || row.VideoType != "photo" {
		t.Fatalf("unexpected sqlite identity fields: %#v", row)
	}
	if row.Title != "Kuaishou E2E mock video" || row.Desc != "Kuaishou E2E mock video" {
		t.Fatalf("unexpected sqlite title/desc: %#v", row)
	}
	if row.CreateTime != "1710000700" || row.LikedCount != "66" || row.ViewdCount != "777" {
		t.Fatalf("unexpected sqlite count/time fields: %#v", row)
	}
	if row.VideoURL != "https://www.kuaishou.com/short-video/ks_video_e2e" {
		t.Fatalf("unexpected sqlite video_url: %#v", row.VideoURL)
	}
	if row.VideoCoverURL != "https://example.test/ks-cover.png" || row.VideoPlayURL != "https://example.test/ks-play.mp4" {
		t.Fatalf("unexpected sqlite media fields: %#v", row)
	}
	if row.SourceKeyword != "mock-ks" {
		t.Fatalf("unexpected sqlite source_keyword: %#v", row.SourceKeyword)
	}
	if row.UserID != "ks-author" || row.Nickname != "mock kuaishou author" || row.Avatar != "https://example.test/ks-avatar.png" {
		t.Fatalf("unexpected sqlite author fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected sqlite timestamps: %#v", row)
	}
}

func TestCLIKuaishouSearchCheckpointResumeSkipsCompletedPages(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	searchCallsByCursor := make(map[string]int)
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
							"client_id": "ks-search-resume-client",
							"platforms": map[string]any{
								"ks": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"ks": "mock-kuaishou"},
						},
					},
				},
			})
		case "/api/cookies/ks":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "kpf=PC_WEB; clientid=mock-client; did=mock-did"},
			})
		case "/graphql":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read graphql body: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode graphql payload: %v\n%s", err, body)
			}
			switch payload["operationName"] {
			case "visionProfileUserList":
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfileUserList": map[string]any{"result": 1},
					},
				})
			case "visionSearchPhoto":
				variables, _ := payload["variables"].(map[string]any)
				pcursor, _ := variables["pcursor"].(string)
				searchCallsByCursor[pcursor]++
				switch pcursor {
				case "1":
					writeJSON(t, w, map[string]any{
						"data": map[string]any{
							"visionSearchPhoto": map[string]any{
								"result": 1,
								"feeds": []map[string]any{
									kuaishouMockSearchResumeFeed("ks_search_resume_video_1", "Kuaishou search resume video 1", "ks-search-resume-author-1"),
								},
								"searchSessionId": "ks-search-resume-session-1",
								"pcursor":         "2",
							},
						},
					})
				case "2":
					if got, _ := variables["searchSessionId"].(string); got != "ks-search-resume-session-1" {
						t.Fatalf("resume page 2 should keep searchSessionId, got %q", got)
					}
					if !allowSecondPage {
						http.Error(w, "transient second search page failure", http.StatusInternalServerError)
						return
					}
					writeJSON(t, w, map[string]any{
						"data": map[string]any{
							"visionSearchPhoto": map[string]any{
								"result": 1,
								"feeds": []map[string]any{
									kuaishouMockSearchResumeFeed("ks_search_resume_video_2", "Kuaishou search resume video 2", "ks-search-resume-author-2"),
								},
								"searchSessionId": "ks-search-resume-session-2",
								"pcursor":         "3",
							},
						},
					})
				default:
					writeJSON(t, w, map[string]any{
						"data": map[string]any{
							"visionSearchPhoto": map[string]any{
								"result":          1,
								"feeds":           []map[string]any{},
								"searchSessionId": "ks-search-resume-empty",
								"pcursor":         "no_more",
							},
						},
					})
				}
			default:
				t.Fatalf("unexpected graphql operation: %#v", payload["operationName"])
			}
		default:
			t.Fatalf("unexpected kuaishou search resume request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "ks"
keywords:
  - "resume-ks"
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
perNoteMaxComments: 2
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
platformEndpoints:
  kuaishouApiUrl: %q
  kuaishouRestApiUrl: %q
`, mockServer.URL, dataDir, mockServer.URL+"/graphql", mockServer.URL)
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
			"MCPRO_KUAISHOU_API_URL="+mockServer.URL+"/graphql",
			"MCPRO_KUAISHOU_REST_API_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("kuaishou search resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("kuaishou search resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if searchCallsByCursor["1"] != 1 || searchCallsByCursor["2"] != 1 {
		t.Fatalf("first kuaishou search run should fetch cursor 1 and fail cursor 2 once, calls=%#v\n%s", searchCallsByCursor, firstOutput)
	}

	allowSecondPage = true
	secondOutput := runCrawler()
	if searchCallsByCursor["1"] != 1 {
		t.Fatalf("resume should skip completed first search page, calls=%#v\n%s", searchCallsByCursor, secondOutput)
	}
	if searchCallsByCursor["2"] != 2 {
		t.Fatalf("resume should retry second search page exactly once, calls=%#v\n%s", searchCallsByCursor, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "kuaishou", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Kuaishou search rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["video_id"])] = true
	}
	if !seen["ks_search_resume_video_1"] || !seen["ks_search_resume_video_2"] {
		t.Fatalf("kuaishou search resume output missing expected video IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIKuaishouDetailCheckpointResumeSkipsCompletedIDs(t *testing.T) {
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
							"client_id": "ks-detail-resume-client",
							"platforms": map[string]any{
								"ks": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"ks": "mock-kuaishou"},
						},
					},
				},
			})
		case "/api/cookies/ks":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "kpf=PC_WEB; clientid=detail-resume-client; did=detail-resume-did"},
			})
		case "/graphql":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read graphql body: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode graphql payload: %v\n%s", err, body)
			}
			switch payload["operationName"] {
			case "visionProfileUserList":
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfileUserList": map[string]any{"result": 1},
					},
				})
			case "visionVideoDetail":
				variables, _ := payload["variables"].(map[string]any)
				videoID := fmt.Sprint(variables["photoId"])
				detailCallsByID[videoID]++
				switch videoID {
				case "ks_detail_resume_video_1":
					writeJSON(t, w, map[string]any{
						"data": map[string]any{
							"visionVideoDetail": kuaishouMockHotSearchResumeFeed(videoID, "Kuaishou detail resume video 1", "ks-detail-resume-author-1"),
						},
					})
				case "ks_detail_resume_video_2":
					if !allowSecondDetail {
						http.Error(w, "transient second detail failure", http.StatusInternalServerError)
						return
					}
					writeJSON(t, w, map[string]any{
						"data": map[string]any{
							"visionVideoDetail": kuaishouMockHotSearchResumeFeed(videoID, "Kuaishou detail resume video 2", "ks-detail-resume-author-2"),
						},
					})
				default:
					t.Fatalf("unexpected kuaishou detail resume video id: %s", videoID)
				}
			default:
				t.Fatalf("unexpected graphql operation: %#v", payload["operationName"])
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "ks"
keywords:
  - "mock-ks"
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
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
specifiedNoteIds:
  - "ks_detail_resume_video_1"
  - "ks_detail_resume_video_2"
proxyConfig:
  enableIpProxy: false
platformEndpoints:
  kuaishouApiUrl: %q
  kuaishouRestApiUrl: %q
`, mockServer.URL, dataDir, mockServer.URL+"/graphql", mockServer.URL)
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
			"MCPRO_KUAISHOU_API_URL="+mockServer.URL+"/graphql",
			"MCPRO_KUAISHOU_REST_API_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("kuaishou detail resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("kuaishou detail resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if detailCallsByID["ks_detail_resume_video_1"] != 1 || detailCallsByID["ks_detail_resume_video_2"] != 1 {
		t.Fatalf("first kuaishou detail run should fetch both IDs and fail second once, calls=%#v\n%s", detailCallsByID, firstOutput)
	}

	allowSecondDetail = true
	secondOutput := runCrawler()
	if detailCallsByID["ks_detail_resume_video_1"] != 1 {
		t.Fatalf("resume should skip completed first kuaishou detail ID, calls=%#v\n%s", detailCallsByID, secondOutput)
	}
	if detailCallsByID["ks_detail_resume_video_2"] != 2 {
		t.Fatalf("resume should retry second kuaishou detail ID exactly once, calls=%#v\n%s", detailCallsByID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "kuaishou", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Kuaishou detail rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["video_id"])] = true
	}
	if !seen["ks_detail_resume_video_1"] || !seen["ks_detail_resume_video_2"] {
		t.Fatalf("kuaishou detail resume output missing expected video IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIKuaishouDetailJSONWithMockServers(t *testing.T) {
	result := runCLIKuaishouDetailWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "kuaishou", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read kuaishou detail json result: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal kuaishou detail json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one kuaishou detail row, got %d: %s", len(rows), body)
	}
	row := rows[0]
	if row["video_id"] != "ks_detail_e2e" || row["video_type"] != "photo" {
		t.Fatalf("unexpected detail identity fields: %#v", row)
	}
	if row["title"] != "Kuaishou detail E2E mock video" || row["desc"] != "Kuaishou detail E2E mock video" {
		t.Fatalf("unexpected detail title/desc: %#v", row)
	}
	if row["create_time"] != "1710000800" || row["liked_count"] != "166" || row["viewd_count"] != "1777" {
		t.Fatalf("unexpected detail count/time fields: %#v", row)
	}
	if row["video_url"] != "https://www.kuaishou.com/short-video/ks_detail_e2e" {
		t.Fatalf("unexpected detail video_url: %#v", row["video_url"])
	}
	if row["video_cover_url"] != "https://example.test/ks-detail-cover.png" || row["video_play_url"] != "https://example.test/ks-detail-play.mp4" {
		t.Fatalf("unexpected detail media fields: %#v", row)
	}
	if row["source_keyword"] != "" {
		t.Fatalf("unexpected detail source_keyword: %#v", row["source_keyword"])
	}
	if row["user_id"] != "ks-detail-author" || row["nickname"] != "mock kuaishou detail author" || row["avatar"] != "https://example.test/ks-detail-avatar.png" {
		t.Fatalf("unexpected detail author fields: %#v", row)
	}
	if _, ok := row["add_ts"]; ok {
		t.Fatalf("kuaishou detail json should match Python and omit add_ts: %#v", row["add_ts"])
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected detail last_modify_ts: %#v", row)
	}
}

func TestCLIKuaishouDetailCSVWithMockServers(t *testing.T) {
	result := runCLIKuaishouDetailWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "kuaishou", "1_detail_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open kuaishou detail csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read kuaishou detail csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected csv header and one detail row, got %d: %#v", len(records), records)
	}
	if csvCell(records, "video_id") != "ks_detail_e2e" || csvCell(records, "title") != "Kuaishou detail E2E mock video" {
		t.Fatalf("unexpected detail csv identity fields: %#v", records)
	}
	if csvCell(records, "create_time") != "1710000800" || csvCell(records, "liked_count") != "166" || csvCell(records, "viewd_count") != "1777" {
		t.Fatalf("unexpected detail csv count/time fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "" {
		t.Fatalf("unexpected detail csv source_keyword: %#v", csvCell(records, "source_keyword"))
	}
	if csvCell(records, "user_id") != "ks-detail-author" || csvCell(records, "nickname") != "mock kuaishou detail author" {
		t.Fatalf("unexpected detail csv author fields: %#v", records)
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected non-empty detail csv last_modify_ts")
	}
}

func TestCLIKuaishouDetailSQLiteWithMockServers(t *testing.T) {
	result := runCLIKuaishouDetailWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open kuaishou detail sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []kuaishouSQLiteVideo
	if err := db.Table("kuaishou_video").Find(&rows).Error; err != nil {
		t.Fatalf("query kuaishou detail sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one detail sqlite row, got %d: %#v", len(rows), rows)
	}
	row := rows[0]
	if row.VideoID != "ks_detail_e2e" || row.Title != "Kuaishou detail E2E mock video" {
		t.Fatalf("unexpected detail sqlite identity fields: %#v", row)
	}
	if row.CreateTime != "1710000800" || row.LikedCount != "166" || row.ViewdCount != "1777" {
		t.Fatalf("unexpected detail sqlite count/time fields: %#v", row)
	}
	if row.SourceKeyword != "" {
		t.Fatalf("unexpected detail sqlite source_keyword: %#v", row.SourceKeyword)
	}
	if row.UserID != "ks-detail-author" || row.Nickname != "mock kuaishou detail author" {
		t.Fatalf("unexpected detail sqlite author fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected detail sqlite timestamps: %#v", row)
	}
}

func TestCLIKuaishouDetailJSONCommentsWithMockServers(t *testing.T) {
	result := runCLIKuaishouDetailWithMockServers(t, "json", true)

	resultPath := filepath.Join(result.dataDir, "kuaishou", "json", "detail_comments_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read kuaishou detail comments json: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal kuaishou comments json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected root and sub kuaishou comments, got %d: %s", len(rows), body)
	}
	root := rows[0]
	if root["comment_id"] != "9001" || root["parent_comment_id"] != "" || root["video_id"] != "ks_detail_e2e" {
		t.Fatalf("unexpected root kuaishou comment identity fields: %#v", root)
	}
	if root["content"] != "root kuaishou comment" || root["create_time"] != "1710000900" {
		t.Fatalf("unexpected root kuaishou comment content/time: %#v", root)
	}
	if root["sub_comment_count"] != "1" || root["like_count"] != "12" {
		t.Fatalf("unexpected root kuaishou comment counts: %#v", root)
	}
	if root["user_id"] != "ks-commenter" || root["nickname"] != "mock ks commenter" || root["avatar"] != "https://example.test/ks-comment-avatar.png" {
		t.Fatalf("unexpected root kuaishou comment author: %#v", root)
	}
	if _, ok := root["add_ts"]; ok {
		t.Fatalf("kuaishou comments json should match Python and omit add_ts: %#v", root["add_ts"])
	}
	if _, ok := root["last_modify_ts"]; !ok {
		t.Fatalf("expected root kuaishou comment last_modify_ts: %#v", root)
	}

	sub := rows[1]
	if sub["comment_id"] != "9002" || sub["parent_comment_id"] != "9001" || sub["video_id"] != "ks_detail_e2e" {
		t.Fatalf("unexpected sub kuaishou comment identity fields: %#v", sub)
	}
	if sub["content"] != "sub kuaishou comment" || sub["create_time"] != "1710000910" {
		t.Fatalf("unexpected sub kuaishou comment content/time: %#v", sub)
	}
	if sub["sub_comment_count"] != "0" || sub["like_count"] != "3" {
		t.Fatalf("unexpected sub kuaishou comment counts: %#v", sub)
	}
	if sub["user_id"] != "ks-sub-commenter" || sub["nickname"] != "mock ks sub commenter" || sub["avatar"] != "https://example.test/ks-sub-comment-avatar.png" {
		t.Fatalf("unexpected sub kuaishou comment author: %#v", sub)
	}
}

func TestCLIKuaishouDetailCSVCommentsWithMockServers(t *testing.T) {
	result := runCLIKuaishouDetailWithMockServers(t, "csv", true)

	resultPath := filepath.Join(result.dataDir, "kuaishou", "1_detail_comments_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open kuaishou detail comments csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read kuaishou detail comments csv: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 3 {
		t.Fatalf("expected csv header plus root/sub comments, got %d: %#v", len(records), records)
	}
	if csvCellAt(records, 1, "comment_id") != "9001" || csvCellAt(records, 1, "parent_comment_id") != "" || csvCellAt(records, 1, "content") != "root kuaishou comment" {
		t.Fatalf("unexpected root kuaishou csv comment: %#v", records)
	}
	if csvCellAt(records, 1, "sub_comment_count") != "1" || csvCellAt(records, 1, "like_count") != "12" || csvCellAt(records, 1, "user_id") != "ks-commenter" {
		t.Fatalf("unexpected root kuaishou csv count/author fields: %#v", records)
	}
	if csvCellAt(records, 2, "comment_id") != "9002" || csvCellAt(records, 2, "parent_comment_id") != "9001" || csvCellAt(records, 2, "content") != "sub kuaishou comment" {
		t.Fatalf("unexpected sub kuaishou csv comment: %#v", records)
	}
	if csvCellAt(records, 2, "sub_comment_count") != "0" || csvCellAt(records, 2, "like_count") != "3" || csvCellAt(records, 2, "user_id") != "ks-sub-commenter" {
		t.Fatalf("unexpected sub kuaishou csv count/author fields: %#v", records)
	}
}

func TestCLIKuaishouDetailSQLiteCommentsWithMockServers(t *testing.T) {
	result := runCLIKuaishouDetailWithMockServers(t, "sqlite", true)

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open kuaishou comments sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []kuaishouSQLiteComment
	if err := db.Table("kuaishou_video_comment").Order("comment_id").Find(&rows).Error; err != nil {
		t.Fatalf("query kuaishou comments sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two sqlite comment rows, got %d: %#v", len(rows), rows)
	}
	root := rows[0]
	if root.CommentID != "9001" || root.ParentCommentID != "" || root.Content != "root kuaishou comment" {
		t.Fatalf("unexpected root sqlite comment: %#v", root)
	}
	if root.CreateTime != "1710000900" || root.SubCommentCount != "1" || root.LikeCount != "12" {
		t.Fatalf("unexpected root sqlite count/time fields: %#v", root)
	}
	if root.UserID != "ks-commenter" || root.Nickname != "mock ks commenter" || root.Avatar != "https://example.test/ks-comment-avatar.png" {
		t.Fatalf("unexpected root sqlite author fields: %#v", root)
	}
	sub := rows[1]
	if sub.CommentID != "9002" || sub.ParentCommentID != "9001" || sub.Content != "sub kuaishou comment" {
		t.Fatalf("unexpected sub sqlite comment: %#v", sub)
	}
	if sub.CreateTime != "1710000910" || sub.SubCommentCount != "0" || sub.LikeCount != "3" {
		t.Fatalf("unexpected sub sqlite count/time fields: %#v", sub)
	}
	if sub.UserID != "ks-sub-commenter" || sub.Nickname != "mock ks sub commenter" || sub.Avatar != "https://example.test/ks-sub-comment-avatar.png" {
		t.Fatalf("unexpected sub sqlite author fields: %#v", sub)
	}
	if root.AddTs == 0 || root.LastModifyTs == 0 || sub.AddTs == 0 || sub.LastModifyTs == 0 {
		t.Fatalf("expected sqlite comment timestamps: %#v %#v", root, sub)
	}
}

func TestCLIKuaishouCreatorJSONWithMockServers(t *testing.T) {
	result := runCLIKuaishouCreatorWithMockServers(t, "json")

	creatorPath := filepath.Join(result.dataDir, "kuaishou", "json", "creator_creator_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(creatorPath)
	if err != nil {
		t.Fatalf("read kuaishou creator json: %v\ncommand output:\n%s", err, result.output)
	}
	var creators []map[string]any
	if err := json.Unmarshal(body, &creators); err != nil {
		t.Fatalf("unmarshal kuaishou creator json: %v\n%s", err, body)
	}
	if len(creators) != 1 {
		t.Fatalf("expected one creator row, got %d: %s", len(creators), body)
	}
	creator := creators[0]
	if creator["user_id"] != "ks-creator-user" || creator["nickname"] != "mock kuaishou creator" {
		t.Fatalf("unexpected creator identity fields: %#v", creator)
	}
	if creator["avatar"] != "https://example.test/ks-creator-avatar.png" || creator["gender"] != "女" || creator["desc"] != "kuaishou creator signature" {
		t.Fatalf("unexpected creator profile fields: %#v", creator)
	}
	if creator["follows"] != "12" || creator["fans"] != "345" || creator["videos_count"] != "1" {
		t.Fatalf("unexpected creator count fields: %#v", creator)
	}

	contentPath := filepath.Join(result.dataDir, "kuaishou", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	contentBody, err := os.ReadFile(contentPath)
	if err != nil {
		t.Fatalf("read kuaishou creator content json: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(contentBody, &rows); err != nil {
		t.Fatalf("unmarshal kuaishou creator content json: %v\n%s", err, contentBody)
	}
	if len(rows) != 1 || rows[0]["video_id"] != "ks_creator_video_e2e" || rows[0]["title"] != "Kuaishou creator E2E mock video" {
		t.Fatalf("unexpected creator content rows: %s", contentBody)
	}
	if rows[0]["source_keyword"] != "" || rows[0]["user_id"] != "ks-creator-author" {
		t.Fatalf("unexpected creator content keyword/author: %#v", rows[0])
	}
}

func TestCLIKuaishouCreatorCSVWithMockServers(t *testing.T) {
	result := runCLIKuaishouCreatorWithMockServers(t, "csv")

	contentPath := filepath.Join(result.dataDir, "kuaishou", "1_creator_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(contentPath)
	if err != nil {
		t.Fatalf("open kuaishou creator csv content: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read kuaishou creator csv content: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 || csvCell(records, "video_id") != "ks_creator_video_e2e" || csvCell(records, "title") != "Kuaishou creator E2E mock video" {
		t.Fatalf("unexpected creator csv content: %#v", records)
	}
	creatorPath := filepath.Join(result.dataDir, "kuaishou", "1_creator_creator_"+time.Now().Format("2006-01-02")+".csv")
	if _, err := os.Stat(creatorPath); !os.IsNotExist(err) {
		t.Fatalf("kuaishou CSV should match Python and not write creator file; stat err=%v", err)
	}
}

func TestCLIKuaishouCreatorSQLiteWithMockServers(t *testing.T) {
	result := runCLIKuaishouCreatorWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open kuaishou creator sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var creators []kuaishouSQLiteCreator
	if err := db.Table("kuaishou_creator").Find(&creators).Error; err != nil {
		t.Fatalf("query kuaishou creator rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(creators) != 1 {
		t.Fatalf("expected one sqlite creator, got %d: %#v", len(creators), creators)
	}
	creator := creators[0]
	if creator.UserID != "ks-creator-user" || creator.Nickname != "mock kuaishou creator" || creator.Gender != "女" {
		t.Fatalf("unexpected sqlite creator identity fields: %#v", creator)
	}
	if creator.Follows != "12" || creator.Fans != "345" || creator.VideosCount != "1" || creator.Desc != "kuaishou creator signature" {
		t.Fatalf("unexpected sqlite creator profile/count fields: %#v", creator)
	}

	var videos []kuaishouSQLiteVideo
	if err := db.Table("kuaishou_video").Find(&videos).Error; err != nil {
		t.Fatalf("query kuaishou creator video rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(videos) != 1 || videos[0].VideoID != "ks_creator_video_e2e" || videos[0].UserID != "ks-creator-author" {
		t.Fatalf("unexpected sqlite creator videos: %#v", videos)
	}
}

func TestCLIKuaishouCreatorCheckpointResumeByCreatorAndCursor(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	allowSecondCursor := false
	profileCalls := 0
	photoListCallsByCursor := make(map[string]int)
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
							"client_id": "ks-creator-resume-client",
							"platforms": map[string]any{
								"ks": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"ks": "mock-kuaishou"},
						},
					},
				},
			})
		case "/api/cookies/ks":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "kpf=PC_WEB; clientid=creator-resume-client; did=creator-resume-did"},
			})
		case "/graphql":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read graphql body: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode graphql payload: %v\n%s", err, body)
			}
			switch payload["operationName"] {
			case "visionProfileUserList":
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfileUserList": map[string]any{"result": 1},
					},
				})
			case "visionProfile":
				profileCalls++
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfile": map[string]any{
							"userProfile": kuaishouMockCreatorProfile(),
						},
					},
				})
			case "visionProfilePhotoList":
				variables, _ := payload["variables"].(map[string]any)
				if got := fmt.Sprint(variables["userId"]); got != "ks-creator-resume" {
					t.Fatalf("unexpected kuaishou creator resume userId: %s", got)
				}
				cursor := fmt.Sprint(variables["pcursor"])
				if cursor == "<nil>" {
					cursor = ""
				}
				photoListCallsByCursor[cursor]++
				switch cursor {
				case "":
					writeJSON(t, w, kuaishouCreatorVideosPayload("cursor-2", kuaishouMockCreatorFeedWithID("ks_creator_resume_1", "Kuaishou creator resume video 1")))
				case "cursor-2":
					if !allowSecondCursor {
						writeJSON(t, w, kuaishouCreatorVideosPayload("cursor-2"))
						return
					}
					writeJSON(t, w, kuaishouCreatorVideosPayload("no_more", kuaishouMockCreatorFeedWithID("ks_creator_resume_2", "Kuaishou creator resume video 2")))
				default:
					t.Fatalf("unexpected kuaishou creator resume cursor: %s", cursor)
				}
			default:
				t.Fatalf("unexpected graphql operation: %#v", payload["operationName"])
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "ks"
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
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
specifiedCreatorIds:
  - "ks-creator-resume"
proxyConfig:
  enableIpProxy: false
platformEndpoints:
  kuaishouApiUrl: %q
  kuaishouRestApiUrl: %q
`, mockServer.URL, dataDir, mockServer.URL+"/graphql", mockServer.URL)
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
			"MCPRO_KUAISHOU_API_URL="+mockServer.URL+"/graphql",
			"MCPRO_KUAISHOU_REST_API_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("kuaishou creator resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("kuaishou creator resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if profileCalls != 1 || photoListCallsByCursor[""] != 1 || photoListCallsByCursor["cursor-2"] != 1 {
		t.Fatalf("first creator run should fetch profile, empty cursor, then stop at cursor-2, profile=%d calls=%#v\n%s", profileCalls, photoListCallsByCursor, firstOutput)
	}

	allowSecondCursor = true
	secondOutput := runCrawler()
	if profileCalls != 2 {
		t.Fatalf("Kuaishou creator resume should match Python and refresh creator profile, profile=%d\n%s", profileCalls, secondOutput)
	}
	if photoListCallsByCursor[""] != 1 {
		t.Fatalf("resume should not re-fetch completed Kuaishou creator empty cursor, calls=%#v\n%s", photoListCallsByCursor, secondOutput)
	}
	if photoListCallsByCursor["cursor-2"] != 2 {
		t.Fatalf("resume should retry Kuaishou creator cursor-2 once, calls=%#v\n%s", photoListCallsByCursor, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "kuaishou", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Kuaishou creator rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["video_id"])] = true
	}
	if !seen["ks_creator_resume_1"] || !seen["ks_creator_resume_2"] {
		t.Fatalf("creator resume output missing expected Kuaishou video IDs, seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIKuaishouHomefeedJSONWithMockServers(t *testing.T) {
	result := runCLIKuaishouHomefeedWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "kuaishou", "json", "homefeed_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read kuaishou homefeed json: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal kuaishou homefeed json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one homefeed row, got %d: %s", len(rows), body)
	}
	row := rows[0]
	assertKuaishouHomefeedMapRow(t, row)
	if _, ok := row["add_ts"]; ok {
		t.Fatalf("kuaishou homefeed json should match Python and omit add_ts: %#v", row["add_ts"])
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected homefeed last_modify_ts: %#v", row)
	}
}

func TestCLIKuaishouHomefeedCSVWithMockServers(t *testing.T) {
	result := runCLIKuaishouHomefeedWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "kuaishou", "1_homefeed_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open kuaishou homefeed csv: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read kuaishou homefeed csv: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected csv header and one homefeed row, got %d: %#v", len(records), records)
	}
	if csvCell(records, "video_id") != "ks_homefeed_video_e2e" || csvCell(records, "video_type") != "photo" {
		t.Fatalf("unexpected homefeed csv identity fields: %#v", records)
	}
	if csvCell(records, "title") != "Kuaishou homefeed E2E mock video" || csvCell(records, "desc") != "Kuaishou homefeed E2E mock video" {
		t.Fatalf("unexpected homefeed csv title/desc: %#v", records)
	}
	if csvCell(records, "create_time") != "1710001100" || csvCell(records, "liked_count") != "366" || csvCell(records, "viewd_count") != "3777" {
		t.Fatalf("unexpected homefeed csv count/time fields: %#v", records)
	}
	if csvCell(records, "video_url") != "https://www.kuaishou.com/short-video/ks_homefeed_video_e2e" {
		t.Fatalf("unexpected homefeed csv video_url: %#v", csvCell(records, "video_url"))
	}
	if csvCell(records, "video_cover_url") != "https://example.test/ks-homefeed-cover.png" || csvCell(records, "video_play_url") != "https://example.test/ks-homefeed-play.mp4" {
		t.Fatalf("unexpected homefeed csv media fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "" {
		t.Fatalf("unexpected homefeed csv source_keyword: %#v", csvCell(records, "source_keyword"))
	}
	if csvCell(records, "user_id") != "ks-homefeed-author" || csvCell(records, "nickname") != "mock kuaishou homefeed author" || csvCell(records, "avatar") != "https://example.test/ks-homefeed-avatar.png" {
		t.Fatalf("unexpected homefeed csv author fields: %#v", records)
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected homefeed csv last_modify_ts")
	}
}

func TestCLIKuaishouHomefeedSQLiteWithMockServers(t *testing.T) {
	result := runCLIKuaishouHomefeedWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open kuaishou homefeed sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []kuaishouSQLiteVideo
	if err := db.Table("kuaishou_video").Find(&rows).Error; err != nil {
		t.Fatalf("query kuaishou homefeed sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one homefeed sqlite row, got %d: %#v", len(rows), rows)
	}
	row := rows[0]
	if row.VideoID != "ks_homefeed_video_e2e" || row.VideoType != "photo" {
		t.Fatalf("unexpected homefeed sqlite identity fields: %#v", row)
	}
	if row.Title != "Kuaishou homefeed E2E mock video" || row.Desc != "Kuaishou homefeed E2E mock video" {
		t.Fatalf("unexpected homefeed sqlite title/desc: %#v", row)
	}
	if row.CreateTime != "1710001100" || row.LikedCount != "366" || row.ViewdCount != "3777" {
		t.Fatalf("unexpected homefeed sqlite count/time fields: %#v", row)
	}
	if row.VideoURL != "https://www.kuaishou.com/short-video/ks_homefeed_video_e2e" {
		t.Fatalf("unexpected homefeed sqlite video_url: %#v", row.VideoURL)
	}
	if row.VideoCoverURL != "https://example.test/ks-homefeed-cover.png" || row.VideoPlayURL != "https://example.test/ks-homefeed-play.mp4" {
		t.Fatalf("unexpected homefeed sqlite media fields: %#v", row)
	}
	if row.SourceKeyword != "" {
		t.Fatalf("unexpected homefeed sqlite source_keyword: %#v", row.SourceKeyword)
	}
	if row.UserID != "ks-homefeed-author" || row.Nickname != "mock kuaishou homefeed author" || row.Avatar != "https://example.test/ks-homefeed-avatar.png" {
		t.Fatalf("unexpected homefeed sqlite author fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected homefeed sqlite timestamps: %#v", row)
	}
}

func TestCLIKuaishouHomefeedCheckpointResumeByCursor(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	allowSecondCursor := false
	homefeedCallsByCursor := make(map[string]int)
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
							"client_id": "ks-homefeed-resume-client",
							"platforms": map[string]any{
								"ks": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"ks": "mock-kuaishou"},
						},
					},
				},
			})
		case "/api/cookies/ks":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "kpf=PC_WEB; clientid=resume-client; did=resume-did"},
			})
		case "/graphql":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read graphql body: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode graphql payload: %v\n%s", err, body)
			}
			switch payload["operationName"] {
			case "visionProfileUserList":
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfileUserList": map[string]any{"result": 1},
					},
				})
			case "brilliantTypeDataQuery":
				variables, _ := payload["variables"].(map[string]any)
				cursor := fmt.Sprint(variables["pcursor"])
				if cursor == "<nil>" {
					cursor = ""
				}
				homefeedCallsByCursor[cursor]++
				switch cursor {
				case "":
					writeJSON(t, w, kuaishouHomefeedPayload("cursor-2", kuaishouMockHomefeedFeedWithID("ks_homefeed_resume_1", "Kuaishou homefeed resume video 1")))
				case "cursor-2":
					if !allowSecondCursor {
						writeJSON(t, w, kuaishouHomefeedPayload("cursor-2"))
						return
					}
					writeJSON(t, w, kuaishouHomefeedPayload("no_more", kuaishouMockHomefeedFeedWithID("ks_homefeed_resume_2", "Kuaishou homefeed resume video 2")))
				default:
					t.Fatalf("unexpected kuaishou homefeed resume cursor: %s", cursor)
				}
			default:
				t.Fatalf("unexpected graphql operation: %#v", payload["operationName"])
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "ks"
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
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
platformEndpoints:
  kuaishouApiUrl: %q
  kuaishouRestApiUrl: %q
`, mockServer.URL, dataDir, mockServer.URL+"/graphql", mockServer.URL)
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
			"MCPRO_KUAISHOU_API_URL="+mockServer.URL+"/graphql",
			"MCPRO_KUAISHOU_REST_API_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("kuaishou homefeed resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("kuaishou homefeed resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if homefeedCallsByCursor[""] != 1 || homefeedCallsByCursor["cursor-2"] != 1 {
		t.Fatalf("first homefeed run should crawl empty cursor and stop at cursor-2, calls=%#v\n%s", homefeedCallsByCursor, firstOutput)
	}

	allowSecondCursor = true
	secondOutput := runCrawler()
	if homefeedCallsByCursor[""] != 1 {
		t.Fatalf("resume should not re-fetch completed Kuaishou homefeed empty cursor, calls=%#v\n%s", homefeedCallsByCursor, secondOutput)
	}
	if homefeedCallsByCursor["cursor-2"] != 2 {
		t.Fatalf("resume should retry Kuaishou homefeed cursor-2 once, calls=%#v\n%s", homefeedCallsByCursor, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "kuaishou", "json", "homefeed_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Kuaishou homefeed rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["video_id"])] = true
	}
	if !seen["ks_homefeed_resume_1"] || !seen["ks_homefeed_resume_2"] {
		t.Fatalf("homefeed resume output missing expected Kuaishou video IDs, seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIKuaishouHotSearchJSONWithMockServers(t *testing.T) {
	result := runCLIKuaishouHotSearchWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "kuaishou", "json", "hot_search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read kuaishou hot_search json: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal kuaishou hot_search json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one hot_search row, got %d: %s", len(rows), body)
	}
	row := rows[0]
	assertKuaishouHotSearchMapRow(t, row)
	if _, ok := row["add_ts"]; ok {
		t.Fatalf("kuaishou hot_search json should match Python and omit add_ts: %#v", row["add_ts"])
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected hot_search last_modify_ts: %#v", row)
	}
}

func TestCLIKuaishouHotSearchCSVWithMockServers(t *testing.T) {
	result := runCLIKuaishouHotSearchWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "kuaishou", "1_hot_search_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open kuaishou hot_search csv: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read kuaishou hot_search csv: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected csv header and one hot_search row, got %d: %#v", len(records), records)
	}
	if csvCell(records, "video_id") != "ks_hot_video_e2e" || csvCell(records, "video_type") != "photo" {
		t.Fatalf("unexpected hot_search csv identity fields: %#v", records)
	}
	if csvCell(records, "title") != "Kuaishou hot search E2E mock video" || csvCell(records, "desc") != "Kuaishou hot search E2E mock video" {
		t.Fatalf("unexpected hot_search csv title/desc: %#v", records)
	}
	if csvCell(records, "create_time") != "1710001200" || csvCell(records, "liked_count") != "466" || csvCell(records, "viewd_count") != "4777" {
		t.Fatalf("unexpected hot_search csv count/time fields: %#v", records)
	}
	if csvCell(records, "video_url") != "https://www.kuaishou.com/short-video/ks_hot_video_e2e" {
		t.Fatalf("unexpected hot_search csv video_url: %#v", csvCell(records, "video_url"))
	}
	if csvCell(records, "video_cover_url") != "https://example.test/ks-hot-cover.png" || csvCell(records, "video_play_url") != "https://example.test/ks-hot-play.mp4" {
		t.Fatalf("unexpected hot_search csv media fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "kuaishou_hot_search" {
		t.Fatalf("unexpected hot_search csv source_keyword: %#v", csvCell(records, "source_keyword"))
	}
	if csvCell(records, "user_id") != "ks-hot-author" || csvCell(records, "nickname") != "mock kuaishou hot author" || csvCell(records, "avatar") != "https://example.test/ks-hot-avatar.png" {
		t.Fatalf("unexpected hot_search csv author fields: %#v", records)
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected hot_search csv last_modify_ts")
	}
}

func TestCLIKuaishouHotSearchSQLiteWithMockServers(t *testing.T) {
	result := runCLIKuaishouHotSearchWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open kuaishou hot_search sqlite db: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []kuaishouSQLiteVideo
	if err := db.Table("kuaishou_video").Find(&rows).Error; err != nil {
		t.Fatalf("query kuaishou hot_search sqlite rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one hot_search sqlite row, got %d: %#v", len(rows), rows)
	}
	row := rows[0]
	if row.VideoID != "ks_hot_video_e2e" || row.VideoType != "photo" {
		t.Fatalf("unexpected hot_search sqlite identity fields: %#v", row)
	}
	if row.Title != "Kuaishou hot search E2E mock video" || row.Desc != "Kuaishou hot search E2E mock video" {
		t.Fatalf("unexpected hot_search sqlite title/desc: %#v", row)
	}
	if row.CreateTime != "1710001200" || row.LikedCount != "466" || row.ViewdCount != "4777" {
		t.Fatalf("unexpected hot_search sqlite count/time fields: %#v", row)
	}
	if row.SourceKeyword != "kuaishou_hot_search" {
		t.Fatalf("unexpected hot_search sqlite source_keyword: %#v", row.SourceKeyword)
	}
	if row.UserID != "ks-hot-author" || row.Nickname != "mock kuaishou hot author" || row.Avatar != "https://example.test/ks-hot-avatar.png" {
		t.Fatalf("unexpected hot_search sqlite author fields: %#v", row)
	}

	var items []kuaishouSQLiteHotSearchItem
	if err := db.Table("platform_hot_search_item").Find(&items).Error; err != nil {
		t.Fatalf("query kuaishou hot_search item rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(items) != 1 {
		t.Fatalf("expected one hot_search item row, got %d: %#v", len(items), items)
	}
	item := items[0]
	if item.Platform != "ks" || item.BoardName != "hot_feed" || item.BoardType != "video" || item.RankNo != 1 {
		t.Fatalf("unexpected hot_search item identity fields: %#v", item)
	}
	if item.Title != "Kuaishou hot search E2E mock video" || item.Keyword != "kuaishou_hot_search" || item.HotValue != "466" {
		t.Fatalf("unexpected hot_search item title/value fields: %#v", item)
	}
	if item.TopicID != "ks_hot_video_e2e" || item.TopicURL != "https://www.kuaishou.com/short-video/ks_hot_video_e2e" {
		t.Fatalf("unexpected hot_search item topic fields: %#v", item)
	}

	var relations []kuaishouSQLiteHotSearchRelation
	if err := db.Table("platform_hot_search_content_rel").Find(&relations).Error; err != nil {
		t.Fatalf("query kuaishou hot_search relation rows: %v\ncommand output:\n%s", err, result.output)
	}
	if len(relations) != 1 {
		t.Fatalf("expected one hot_search relation row, got %d: %#v", len(relations), relations)
	}
	rel := relations[0]
	if rel.Platform != "ks" || rel.ContentID != "ks_hot_video_e2e" || rel.ContentType != "video" || rel.ContentTable != "kuaishou_video" {
		t.Fatalf("unexpected hot_search relation identity fields: %#v", rel)
	}
	if rel.RankNo != 1 || rel.SourceKeyword != "kuaishou_hot_search" {
		t.Fatalf("unexpected hot_search relation rank/source fields: %#v", rel)
	}
}

func TestCLIKuaishouHotSearchCheckpointResumeUsesSnapshotAndSkipsCompletedFeed(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	hotFeedCalls := 0
	detailCallsByPhotoID := make(map[string]int)
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
							"client_id": "ks-hot-resume-client",
							"platforms": map[string]any{
								"ks": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"ks": "mock-kuaishou"},
						},
					},
				},
			})
		case "/api/cookies/ks":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "kpf=PC_WEB; clientid=mock-client; did=mock-did"},
			})
		case "/rest/v/feed/hot":
			hotFeedCalls++
			writeJSON(t, w, map[string]any{
				"result":  1,
				"pcursor": "no_more",
				"feeds": []map[string]any{
					kuaishouMockHotSearchResumeFeed("ks_hot_resume_video_1", "Kuaishou hot resume video 1", "ks-hot-resume-author-1"),
					kuaishouMockHotSearchResumeFeed("ks_hot_resume_video_2", "Kuaishou hot resume video 2", "ks-hot-resume-author-2"),
				},
			})
		case "/graphql":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read graphql body: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode graphql payload: %v\n%s", err, body)
			}
			switch payload["operationName"] {
			case "visionProfileUserList":
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfileUserList": map[string]any{"result": 1},
					},
				})
			case "visionVideoDetail":
				variables, _ := payload["variables"].(map[string]any)
				photoID, _ := variables["photoId"].(string)
				detailCallsByPhotoID[photoID]++
				if photoID == "ks_hot_resume_video_2" && !allowSecondVideo {
					http.Error(w, "transient second hot feed failure", http.StatusInternalServerError)
					return
				}
				switch photoID {
				case "ks_hot_resume_video_1":
					writeJSON(t, w, map[string]any{
						"data": map[string]any{
							"visionVideoDetail": kuaishouMockHotSearchResumeFeed(photoID, "Kuaishou hot resume video 1", "ks-hot-resume-author-1"),
						},
					})
				case "ks_hot_resume_video_2":
					writeJSON(t, w, map[string]any{
						"data": map[string]any{
							"visionVideoDetail": kuaishouMockHotSearchResumeFeed(photoID, "Kuaishou hot resume video 2", "ks-hot-resume-author-2"),
						},
					})
				default:
					t.Fatalf("unexpected hot resume detail photoId: %s", photoID)
				}
			default:
				t.Fatalf("unexpected graphql operation: %#v", payload["operationName"])
			}
		default:
			t.Fatalf("unexpected kuaishou hot resume request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "ks"
keywords:
  - "mock-ks"
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
platformEndpoints:
  kuaishouApiUrl: %q
  kuaishouRestApiUrl: %q
`, mockServer.URL, dataDir, mockServer.URL+"/graphql", mockServer.URL)
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
			"MCPRO_KUAISHOU_API_URL="+mockServer.URL+"/graphql",
			"MCPRO_KUAISHOU_REST_API_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("kuaishou hot resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("kuaishou hot resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if hotFeedCalls != 1 || detailCallsByPhotoID["ks_hot_resume_video_1"] != 1 || detailCallsByPhotoID["ks_hot_resume_video_2"] != 1 {
		t.Fatalf("first kuaishou hot_search run should fetch rank and fail second detail once; hot=%d detail=%#v\n%s", hotFeedCalls, detailCallsByPhotoID, firstOutput)
	}

	allowSecondVideo = true
	secondOutput := runCrawler()
	if hotFeedCalls != 1 {
		t.Fatalf("resume should reuse kuaishou hot_search snapshot and not refetch rank, got %d\n%s", hotFeedCalls, secondOutput)
	}
	if detailCallsByPhotoID["ks_hot_resume_video_1"] != 1 {
		t.Fatalf("resume should skip completed first hot feed, detail=%#v\n%s", detailCallsByPhotoID, secondOutput)
	}
	if detailCallsByPhotoID["ks_hot_resume_video_2"] != 2 {
		t.Fatalf("resume should retry second hot feed exactly once, detail=%#v\n%s", detailCallsByPhotoID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "kuaishou", "json", "hot_search_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Kuaishou hot_search rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["video_id"])] = true
	}
	if !seen["ks_hot_resume_video_1"] || !seen["ks_hot_resume_video_2"] {
		t.Fatalf("kuaishou hot_search resume output missing expected video IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func assertKuaishouHomefeedMapRow(t *testing.T, row map[string]any) {
	t.Helper()
	if row["video_id"] != "ks_homefeed_video_e2e" || row["video_type"] != "photo" {
		t.Fatalf("unexpected homefeed identity fields: %#v", row)
	}
	if row["title"] != "Kuaishou homefeed E2E mock video" || row["desc"] != "Kuaishou homefeed E2E mock video" {
		t.Fatalf("unexpected homefeed title/desc: %#v", row)
	}
	if row["create_time"] != "1710001100" || row["liked_count"] != "366" || row["viewd_count"] != "3777" {
		t.Fatalf("unexpected homefeed count/time fields: %#v", row)
	}
	if row["video_url"] != "https://www.kuaishou.com/short-video/ks_homefeed_video_e2e" {
		t.Fatalf("unexpected homefeed video_url: %#v", row["video_url"])
	}
	if row["video_cover_url"] != "https://example.test/ks-homefeed-cover.png" || row["video_play_url"] != "https://example.test/ks-homefeed-play.mp4" {
		t.Fatalf("unexpected homefeed media fields: %#v", row)
	}
	if row["source_keyword"] != "" {
		t.Fatalf("unexpected homefeed source_keyword: %#v", row["source_keyword"])
	}
	if row["user_id"] != "ks-homefeed-author" || row["nickname"] != "mock kuaishou homefeed author" || row["avatar"] != "https://example.test/ks-homefeed-avatar.png" {
		t.Fatalf("unexpected homefeed author fields: %#v", row)
	}
}

func assertKuaishouHotSearchMapRow(t *testing.T, row map[string]any) {
	t.Helper()
	if row["video_id"] != "ks_hot_video_e2e" || row["video_type"] != "photo" {
		t.Fatalf("unexpected hot_search identity fields: %#v", row)
	}
	if row["title"] != "Kuaishou hot search E2E mock video" || row["desc"] != "Kuaishou hot search E2E mock video" {
		t.Fatalf("unexpected hot_search title/desc: %#v", row)
	}
	if row["create_time"] != "1710001200" || row["liked_count"] != "466" || row["viewd_count"] != "4777" {
		t.Fatalf("unexpected hot_search count/time fields: %#v", row)
	}
	if row["video_url"] != "https://www.kuaishou.com/short-video/ks_hot_video_e2e" {
		t.Fatalf("unexpected hot_search video_url: %#v", row["video_url"])
	}
	if row["video_cover_url"] != "https://example.test/ks-hot-cover.png" || row["video_play_url"] != "https://example.test/ks-hot-play.mp4" {
		t.Fatalf("unexpected hot_search media fields: %#v", row)
	}
	if row["source_keyword"] != "kuaishou_hot_search" {
		t.Fatalf("unexpected hot_search source_keyword: %#v", row["source_keyword"])
	}
	if row["user_id"] != "ks-hot-author" || row["nickname"] != "mock kuaishou hot author" || row["avatar"] != "https://example.test/ks-hot-avatar.png" {
		t.Fatalf("unexpected hot_search author fields: %#v", row)
	}
}

type kuaishouSQLiteVideo struct {
	UserID        string
	Nickname      string
	Avatar        string
	AddTs         int64
	LastModifyTs  int64
	VideoID       string
	VideoType     string
	Title         string
	Desc          string
	CreateTime    string
	LikedCount    string
	ViewdCount    string
	VideoURL      string
	VideoCoverURL string
	VideoPlayURL  string
	SourceKeyword string
}

type kuaishouSQLiteComment struct {
	UserID          string
	Nickname        string
	Avatar          string
	AddTs           int64
	LastModifyTs    int64
	CommentID       string
	ParentCommentID string
	VideoID         string
	Content         string
	CreateTime      string
	SubCommentCount string
	LikeCount       string
}

type kuaishouSQLiteCreator struct {
	UserID       string
	Nickname     string
	Avatar       string
	IPLocation   string
	AddTs        int64
	LastModifyTs int64
	Desc         string
	Gender       string
	Follows      string
	Fans         string
	VideosCount  string
}

type kuaishouSQLiteHotSearchItem struct {
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

type kuaishouSQLiteHotSearchRelation struct {
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

type cliKuaishouResult struct {
	dataDir    string
	sqlitePath string
	output     []byte
}

func runCLIKuaishouSearchWithMockServers(t *testing.T, saveDataOption string) cliKuaishouResult {
	t.Helper()
	repoRoot := e2e.RepoRoot(t)

	var searchCalls int
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
							"client_id": "ks-mock-client",
							"platforms": map[string]any{
								"ks": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"ks": "mock-kuaishou"},
						},
					},
				},
			})
		case "/api/cookies/ks":
			if got := r.URL.Query().Get("client_id"); got != "ks-mock-client" {
				t.Fatalf("unexpected client_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "kpf=PC_WEB; clientid=mock-client; did=mock-did"},
			})
		case "/graphql":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read graphql body: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode graphql payload: %v\n%s", err, body)
			}
			switch payload["operationName"] {
			case "visionProfileUserList":
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfileUserList": map[string]any{"result": 1},
					},
				})
			case "visionSearchPhoto":
				searchCalls++
				variables, _ := payload["variables"].(map[string]any)
				pcursor, _ := variables["pcursor"].(string)
				if pcursor != "1" {
					writeJSON(t, w, map[string]any{
						"data": map[string]any{
							"visionSearchPhoto": map[string]any{
								"result":          1,
								"feeds":           []map[string]any{},
								"searchSessionId": "ks-session-empty",
								"pcursor":         "no_more",
							},
						},
					})
					return
				}
				feed := kuaishouMockFeed()
				feeds := make([]map[string]any, 0, 21)
				for i := 0; i < 21; i++ {
					feeds = append(feeds, feed)
				}
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionSearchPhoto": map[string]any{
							"result":          1,
							"feeds":           feeds,
							"searchSessionId": "ks-session-1",
							"pcursor":         "2",
						},
					},
				})
			default:
				t.Fatalf("unexpected graphql operation: %#v", payload["operationName"])
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	sqlitePath := filepath.Join(tmpDir, "media_crawler.db")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "ks"
keywords:
  - "mock-ks"
crawlerType: "search"
saveDataOption: %q
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
sqliteDbPath: %q
startPage: 1
crawlerMaxNotesCount: 20
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
platformEndpoints:
  kuaishouApiUrl: %q
  kuaishouRestApiUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, mockServer.URL+"/graphql", mockServer.URL)
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
		"MCPRO_KUAISHOU_API_URL="+mockServer.URL+"/graphql",
		"MCPRO_KUAISHOU_REST_API_URL="+mockServer.URL,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if searchCalls != 1 {
		t.Fatalf("expected one kuaishou search request like Python duplicate-feed limit flow, got %d\n%s", searchCalls, output)
	}
	return cliKuaishouResult{
		dataDir:    dataDir,
		sqlitePath: sqlitePath,
		output:     output,
	}
}

func runCLIKuaishouDetailWithMockServers(t *testing.T, saveDataOption string, commentsEnabled ...bool) cliKuaishouResult {
	t.Helper()
	repoRoot := e2e.RepoRoot(t)

	enableComments := len(commentsEnabled) > 0 && commentsEnabled[0]
	var detailCalls int
	var rootCommentCalls int
	var subCommentCalls int
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
							"client_id": "ks-mock-client",
							"platforms": map[string]any{
								"ks": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"ks": "mock-kuaishou"},
						},
					},
				},
			})
		case "/api/cookies/ks":
			if got := r.URL.Query().Get("client_id"); got != "ks-mock-client" {
				t.Fatalf("unexpected client_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "kpf=PC_WEB; clientid=mock-client; did=mock-did"},
			})
		case "/graphql":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read graphql body: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode graphql payload: %v\n%s", err, body)
			}
			switch payload["operationName"] {
			case "visionProfileUserList":
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfileUserList": map[string]any{"result": 1},
					},
				})
			case "visionVideoDetail":
				detailCalls++
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionVideoDetail": kuaishouMockDetailFeed(),
					},
				})
			default:
				t.Fatalf("unexpected graphql operation: %#v", payload["operationName"])
			}
		case "/rest/v/photo/comment/list":
			rootCommentCalls++
			writeJSON(t, w, map[string]any{
				"result":    1,
				"pcursorV2": "no_more",
				"rootCommentsV2": []map[string]any{
					kuaishouMockRootComment(),
				},
			})
		case "/rest/v/photo/comment/sublist":
			subCommentCalls++
			writeJSON(t, w, map[string]any{
				"result":    1,
				"pcursorV2": "no_more",
				"subCommentsV2": []map[string]any{
					kuaishouMockSubComment(),
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	sqlitePath := filepath.Join(tmpDir, "media_crawler.db")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "ks"
keywords:
  - "mock-ks"
crawlerType: "detail"
saveDataOption: %q
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
sqliteDbPath: %q
startPage: 1
crawlerMaxNotesCount: 20
maxConcurrencyNum: 1
requestTimeOut: 2
enableGetComments: %t
enableGetSubComments: %t
enableCheckpoint: false
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
specifiedNoteIds:
  - "ks_detail_e2e"
proxyConfig:
  enableIpProxy: false
platformEndpoints:
  kuaishouApiUrl: %q
  kuaishouRestApiUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, enableComments, enableComments, mockServer.URL+"/graphql", mockServer.URL)
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
		"MCPRO_KUAISHOU_API_URL="+mockServer.URL+"/graphql",
		"MCPRO_KUAISHOU_REST_API_URL="+mockServer.URL,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if detailCalls != 1 {
		t.Fatalf("expected one kuaishou detail request, got %d\n%s", detailCalls, output)
	}
	if enableComments && rootCommentCalls != 1 {
		t.Fatalf("expected one kuaishou root comment request, got %d\n%s", rootCommentCalls, output)
	}
	if enableComments && subCommentCalls != 1 {
		t.Fatalf("expected one kuaishou sub comment request, got %d\n%s", subCommentCalls, output)
	}
	return cliKuaishouResult{
		dataDir:    dataDir,
		sqlitePath: sqlitePath,
		output:     output,
	}
}

func runCLIKuaishouCreatorWithMockServers(t *testing.T, saveDataOption string) cliKuaishouResult {
	t.Helper()
	repoRoot := e2e.RepoRoot(t)

	var profileCalls int
	var photoListCalls int
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
							"client_id": "ks-mock-client",
							"platforms": map[string]any{
								"ks": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"ks": "mock-kuaishou"},
						},
					},
				},
			})
		case "/api/cookies/ks":
			if got := r.URL.Query().Get("client_id"); got != "ks-mock-client" {
				t.Fatalf("unexpected client_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "kpf=PC_WEB; clientid=mock-client; did=mock-did"},
			})
		case "/graphql":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read graphql body: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode graphql payload: %v\n%s", err, body)
			}
			switch payload["operationName"] {
			case "visionProfileUserList":
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfileUserList": map[string]any{"result": 1},
					},
				})
			case "visionProfile":
				profileCalls++
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfile": map[string]any{
							"userProfile": kuaishouMockCreatorProfile(),
						},
					},
				})
			case "visionProfilePhotoList":
				photoListCalls++
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfilePhotoList": map[string]any{
							"pcursor": "no_more",
							"feeds": []map[string]any{
								kuaishouMockCreatorFeed(),
							},
						},
					},
				})
			default:
				t.Fatalf("unexpected graphql operation: %#v", payload["operationName"])
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	sqlitePath := filepath.Join(tmpDir, "media_crawler.db")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "ks"
keywords:
  - "mock-ks"
crawlerType: "creator"
saveDataOption: %q
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
sqliteDbPath: %q
startPage: 1
crawlerMaxNotesCount: 20
maxConcurrencyNum: 1
requestTimeOut: 2
enableGetComments: false
enableGetSubComments: false
enableCheckpoint: false
perNoteMaxComments: 10
crawlerTimeSleep: 0
debugRequest: false
specifiedCreatorIds:
  - "ks-creator-user"
proxyConfig:
  enableIpProxy: false
platformEndpoints:
  kuaishouApiUrl: %q
  kuaishouRestApiUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, mockServer.URL+"/graphql", mockServer.URL)
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
		"MCPRO_KUAISHOU_API_URL="+mockServer.URL+"/graphql",
		"MCPRO_KUAISHOU_REST_API_URL="+mockServer.URL,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if profileCalls != 1 {
		t.Fatalf("expected one kuaishou creator profile request, got %d\n%s", profileCalls, output)
	}
	if photoListCalls != 1 {
		t.Fatalf("expected one kuaishou creator photo list request, got %d\n%s", photoListCalls, output)
	}
	return cliKuaishouResult{
		dataDir:    dataDir,
		sqlitePath: sqlitePath,
		output:     output,
	}
}

func runCLIKuaishouHomefeedWithMockServers(t *testing.T, saveDataOption string) cliKuaishouResult {
	t.Helper()
	repoRoot := e2e.RepoRoot(t)

	var homefeedCalls int
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
							"client_id": "ks-mock-client",
							"platforms": map[string]any{
								"ks": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"ks": "mock-kuaishou"},
						},
					},
				},
			})
		case "/api/cookies/ks":
			if got := r.URL.Query().Get("client_id"); got != "ks-mock-client" {
				t.Fatalf("unexpected client_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "kpf=PC_WEB; clientid=mock-client; did=mock-did"},
			})
		case "/graphql":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read graphql body: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode graphql payload: %v\n%s", err, body)
			}
			switch payload["operationName"] {
			case "visionProfileUserList":
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfileUserList": map[string]any{"result": 1},
					},
				})
			case "brilliantTypeDataQuery":
				homefeedCalls++
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"brilliantTypeData": map[string]any{
							"pcursor": "no_more",
							"feeds": []map[string]any{
								kuaishouMockHomefeedFeed(),
							},
						},
					},
				})
			default:
				t.Fatalf("unexpected graphql operation: %#v", payload["operationName"])
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	sqlitePath := filepath.Join(tmpDir, "media_crawler.db")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "ks"
keywords:
  - "mock-ks"
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
platformEndpoints:
  kuaishouApiUrl: %q
  kuaishouRestApiUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, mockServer.URL+"/graphql", mockServer.URL)
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
		"MCPRO_KUAISHOU_API_URL="+mockServer.URL+"/graphql",
		"MCPRO_KUAISHOU_REST_API_URL="+mockServer.URL,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if homefeedCalls != 1 {
		t.Fatalf("expected one kuaishou homefeed request, got %d\n%s", homefeedCalls, output)
	}
	return cliKuaishouResult{
		dataDir:    dataDir,
		sqlitePath: sqlitePath,
		output:     output,
	}
}

func runCLIKuaishouHotSearchWithMockServers(t *testing.T, saveDataOption string) cliKuaishouResult {
	t.Helper()
	repoRoot := e2e.RepoRoot(t)

	var hotFeedCalls int
	var detailCalls int
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
							"client_id": "ks-mock-client",
							"platforms": map[string]any{
								"ks": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"ks": "mock-kuaishou"},
						},
					},
				},
			})
		case "/api/cookies/ks":
			if got := r.URL.Query().Get("client_id"); got != "ks-mock-client" {
				t.Fatalf("unexpected client_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "kpf=PC_WEB; clientid=mock-client; did=mock-did"},
			})
		case "/rest/v/feed/hot":
			hotFeedCalls++
			writeJSON(t, w, map[string]any{
				"result":  1,
				"pcursor": "no_more",
				"feeds": []map[string]any{
					kuaishouMockHotSearchFeed(),
				},
			})
		case "/graphql":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read graphql body: %v", err)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode graphql payload: %v\n%s", err, body)
			}
			switch payload["operationName"] {
			case "visionProfileUserList":
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfileUserList": map[string]any{"result": 1},
					},
				})
			case "visionVideoDetail":
				detailCalls++
				variables, _ := payload["variables"].(map[string]any)
				if got, _ := variables["photoId"].(string); got != "ks_hot_video_e2e" {
					t.Fatalf("unexpected hot_search detail photoId: %s", got)
				}
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionVideoDetail": kuaishouMockHotSearchFeed(),
					},
				})
			default:
				t.Fatalf("unexpected graphql operation: %#v", payload["operationName"])
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer mockServer.Close()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	sqlitePath := filepath.Join(tmpDir, "media_crawler.db")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configYAML := fmt.Sprintf(`
platform: "ks"
keywords:
  - "mock-ks"
crawlerType: "hot_search"
saveDataOption: %q
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: %q
dataDir: %q
sqliteDbPath: %q
startPage: 1
crawlerMaxNotesCount: 20
hotSearchMaxItemsCount: 1
hotSearchPerItemContentCount: 1
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
platformEndpoints:
  kuaishouApiUrl: %q
  kuaishouRestApiUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, mockServer.URL+"/graphql", mockServer.URL)
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
		"MCPRO_KUAISHOU_API_URL="+mockServer.URL+"/graphql",
		"MCPRO_KUAISHOU_REST_API_URL="+mockServer.URL,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if hotFeedCalls != 1 || detailCalls != 1 {
		t.Fatalf("unexpected hot_search calls: hot_feed=%d detail=%d\n%s", hotFeedCalls, detailCalls, output)
	}
	return cliKuaishouResult{
		dataDir:    dataDir,
		sqlitePath: sqlitePath,
		output:     output,
	}
}

func kuaishouMockRootComment() map[string]any {
	return map[string]any{
		"comment_id":     9001,
		"content":        "root kuaishou comment",
		"timestamp":      1710000900,
		"commentCount":   1,
		"likeCount":      12,
		"author_id":      "ks-commenter",
		"author_name":    "mock ks commenter",
		"headurl":        "https://example.test/ks-comment-avatar.png",
		"hasSubComments": true,
	}
}

func kuaishouMockCreatorProfile() map[string]any {
	return map[string]any{
		"ownerCount": map[string]any{
			"follow":       12,
			"fan":          345,
			"photo_public": 1,
		},
		"profile": map[string]any{
			"user_name": "mock kuaishou creator",
			"gender":    "F",
			"headurl":   "https://example.test/ks-creator-avatar.png",
			"user_text": "kuaishou creator signature",
		},
	}
}

func kuaishouMockCreatorFeed() map[string]any {
	return kuaishouMockCreatorFeedWithID("ks_creator_video_e2e", "Kuaishou creator E2E mock video")
}

func kuaishouMockCreatorFeedWithID(videoID, title string) map[string]any {
	return map[string]any{
		"type": "photo",
		"author": map[string]any{
			"id":        "ks-creator-author",
			"name":      "mock kuaishou creator author",
			"headerUrl": "https://example.test/ks-creator-author-avatar.png",
		},
		"photo": map[string]any{
			"id":            videoID,
			"caption":       title,
			"timestamp":     1710001000,
			"realLikeCount": 266,
			"viewCount":     2777,
			"coverUrl":      "https://example.test/ks-creator-cover.png",
			"photoUrl":      "https://example.test/ks-creator-play.mp4",
		},
	}
}

func kuaishouCreatorVideosPayload(pcursor string, feeds ...map[string]any) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"visionProfilePhotoList": map[string]any{
				"pcursor": pcursor,
				"feeds":   feeds,
			},
		},
	}
}

func kuaishouMockHomefeedFeed() map[string]any {
	return kuaishouMockHomefeedFeedWithID("ks_homefeed_video_e2e", "Kuaishou homefeed E2E mock video")
}

func kuaishouMockHomefeedFeedWithID(videoID, title string) map[string]any {
	return map[string]any{
		"type": "photo",
		"author": map[string]any{
			"id":        "ks-homefeed-author",
			"name":      "mock kuaishou homefeed author",
			"headerUrl": "https://example.test/ks-homefeed-avatar.png",
		},
		"photo": map[string]any{
			"id":            videoID,
			"caption":       title,
			"timestamp":     1710001100,
			"realLikeCount": 366,
			"viewCount":     3777,
			"coverUrl":      "https://example.test/ks-homefeed-cover.png",
			"photoUrl":      "https://example.test/ks-homefeed-play.mp4",
		},
	}
}

func kuaishouHomefeedPayload(pcursor string, feeds ...map[string]any) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"brilliantTypeData": map[string]any{
				"pcursor": pcursor,
				"feeds":   feeds,
			},
		},
	}
}

func kuaishouMockHotSearchFeed() map[string]any {
	return map[string]any{
		"type": "photo",
		"author": map[string]any{
			"id":        "ks-hot-author",
			"name":      "mock kuaishou hot author",
			"headerUrl": "https://example.test/ks-hot-avatar.png",
		},
		"photo": map[string]any{
			"id":            "ks_hot_video_e2e",
			"caption":       "Kuaishou hot search E2E mock video",
			"timestamp":     1710001200,
			"realLikeCount": 466,
			"viewCount":     4777,
			"coverUrl":      "https://example.test/ks-hot-cover.png",
			"photoUrl":      "https://example.test/ks-hot-play.mp4",
		},
	}
}

func kuaishouMockHotSearchResumeFeed(videoID, caption, authorID string) map[string]any {
	return map[string]any{
		"type": "photo",
		"author": map[string]any{
			"id":        authorID,
			"name":      "mock " + authorID,
			"headerUrl": "https://example.test/" + authorID + ".png",
		},
		"photo": map[string]any{
			"id":            videoID,
			"caption":       caption,
			"timestamp":     1710001300,
			"realLikeCount": 566,
			"viewCount":     5777,
			"coverUrl":      "https://example.test/" + videoID + "-cover.png",
			"photoUrl":      "https://example.test/" + videoID + "-play.mp4",
		},
	}
}

func kuaishouMockSubComment() map[string]any {
	return map[string]any{
		"comment_id":   9002,
		"content":      "sub kuaishou comment",
		"timestamp":    1710000910,
		"commentCount": 0,
		"likeCount":    3,
		"author_id":    "ks-sub-commenter",
		"author_name":  "mock ks sub commenter",
		"headurl":      "https://example.test/ks-sub-comment-avatar.png",
	}
}

func kuaishouMockFeed() map[string]any {
	return map[string]any{
		"type": "photo",
		"author": map[string]any{
			"id":        "ks-author",
			"name":      "mock kuaishou author",
			"headerUrl": "https://example.test/ks-avatar.png",
		},
		"photo": map[string]any{
			"id":            "ks_video_e2e",
			"caption":       "Kuaishou E2E mock video",
			"timestamp":     1710000700,
			"realLikeCount": 66,
			"viewCount":     777,
			"coverUrl":      "https://example.test/ks-cover.png",
			"photoUrl":      "https://example.test/ks-play.mp4",
		},
	}
}

func kuaishouMockSearchResumeFeed(videoID, caption, authorID string) map[string]any {
	return map[string]any{
		"type": "photo",
		"author": map[string]any{
			"id":        authorID,
			"name":      "mock " + authorID,
			"headerUrl": "https://example.test/" + authorID + ".png",
		},
		"photo": map[string]any{
			"id":            videoID,
			"caption":       caption,
			"timestamp":     1710000750,
			"realLikeCount": 76,
			"viewCount":     877,
			"coverUrl":      "https://example.test/" + videoID + "-cover.png",
			"photoUrl":      "https://example.test/" + videoID + "-play.mp4",
		},
	}
}

func kuaishouMockDetailFeed() map[string]any {
	return map[string]any{
		"type": "photo",
		"author": map[string]any{
			"id":        "ks-detail-author",
			"name":      "mock kuaishou detail author",
			"headerUrl": "https://example.test/ks-detail-avatar.png",
		},
		"photo": map[string]any{
			"id":            "ks_detail_e2e",
			"caption":       "Kuaishou detail E2E mock video",
			"timestamp":     1710000800,
			"realLikeCount": 166,
			"viewCount":     1777,
			"coverUrl":      "https://example.test/ks-detail-cover.png",
			"photoUrl":      "https://example.test/ks-detail-play.mp4",
		},
	}
}
