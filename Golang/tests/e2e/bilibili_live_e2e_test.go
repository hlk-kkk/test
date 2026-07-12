// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/bilibili_live_e2e_test.go
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
	"database/sql"
	"encoding/csv"
	"encoding/json"
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
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type liveBilibiliRun struct {
	WorkDir    string
	DataDir    string
	SQLitePath string
	Output     []byte
}

type liveBilibiliConfig struct {
	CrawlerType       string
	SaveDataOption    string
	Keywords          []string
	SpecifiedNoteIDs  []string
	SpecifiedCreator  []string
	EnableComments    bool
	EnableCheckpoint  bool
	CrawlerMaxCount   int
	HotSearchMaxItems int
}

func TestCLIBilibiliLiveBusinessWithRealCookie(t *testing.T) {
	cookieBridgeURL := os.Getenv("MCPRO_BILIBILI_LIVE_COOKIE_BRIDGE_URL")
	if cookieBridgeURL == "" {
		t.Skip("set MCPRO_BILIBILI_LIVE_COOKIE_BRIDGE_URL to run live Bilibili business tests")
	}
	if _, err := url.ParseRequestURI(cookieBridgeURL); err != nil {
		t.Fatalf("invalid MCPRO_BILIBILI_LIVE_COOKIE_BRIDGE_URL: %v", err)
	}

	binaryPath := e2e.BuildCLI(t, e2e.RepoRoot(t))
	baseEnv := []string{
		"COOKIE_BRIDGE_URL=" + cookieBridgeURL,
		"MCPRO_TEST_BLOCK_EXTERNAL_HTTP=0",
	}
	keyword := envOrDefault("MCPRO_BILIBILI_LIVE_KEYWORD", "deepseek")

	search := runLiveBilibiliCLI(t, binaryPath, baseEnv, liveBilibiliConfig{
		CrawlerType:      "search",
		SaveDataOption:   "json",
		Keywords:         []string{keyword},
		CrawlerMaxCount:  21,
		EnableCheckpoint: false,
	})
	searchRows := readLiveJSONRows(t, filepath.Join(search.DataDir, "bilibili", "json", datedFile("search_contents", "json")), search.Output)
	if len(searchRows) != 21 {
		t.Fatalf("live search pagination should save 21 rows across page 1 and 2, got %d\n%s", len(searchRows), search.Output)
	}
	if !strings.Contains(string(search.Output), "Searching page: 2") {
		t.Fatalf("live search pagination did not reach page 2\n%s", search.Output)
	}
	requireLiveVideoRows(t, searchRows, keyword, search.Output)
	bvid := fmt.Sprint(searchRows[0]["bvid"])
	creatorID := fmt.Sprint(searchRows[0]["user_id"])
	if bvid == "" || bvid == "<nil>" || creatorID == "" || creatorID == "<nil>" {
		t.Fatalf("live search did not return usable bvid/user_id: %#v", searchRows[0])
	}
	commentBvid := pickLiveCommentableBVID(t, searchRows)

	csvRun := runLiveBilibiliCLI(t, binaryPath, baseEnv, liveBilibiliConfig{
		CrawlerType:     "search",
		SaveDataOption:  "csv",
		Keywords:        []string{keyword},
		CrawlerMaxCount: 1,
	})
	csvRecords := readLiveCSV(t, filepath.Join(csvRun.DataDir, "bilibili", "1_"+datedFile("search_contents", "csv")), csvRun.Output)
	if len(csvRecords) != 2 {
		t.Fatalf("live search csv should contain header plus one row, got %d\n%s", len(csvRecords), csvRun.Output)
	}
	if e2e.CSVCellAt(csvRecords, 1, "source_keyword") != keyword || e2e.CSVCellAt(csvRecords, 1, "bvid") == "" || e2e.CSVCellAt(csvRecords, 1, "title") == "" {
		t.Fatalf("live search csv row has unexpected data: %#v\n%s", csvRecords, csvRun.Output)
	}

	sqliteRun := runLiveBilibiliCLI(t, binaryPath, baseEnv, liveBilibiliConfig{
		CrawlerType:     "search",
		SaveDataOption:  "sqlite",
		Keywords:        []string{keyword},
		CrawlerMaxCount: 1,
	})
	if countSQLiteRows(t, sqliteRun.SQLitePath, "bilibili_video", sqliteRun.Output) != 1 {
		t.Fatalf("live search sqlite should contain one bilibili_video row\n%s", sqliteRun.Output)
	}
	sqliteVideo := readSQLiteVideo(t, sqliteRun.SQLitePath, sqliteRun.Output)
	if sqliteVideo.Bvid == "" || sqliteVideo.Title == "" || sqliteVideo.SourceKeyword != keyword {
		t.Fatalf("live sqlite video data mismatch: %#v\n%s", sqliteVideo, sqliteRun.Output)
	}

	detail := runLiveBilibiliCLI(t, binaryPath, baseEnv, liveBilibiliConfig{
		CrawlerType:      "detail",
		SaveDataOption:   "json",
		SpecifiedNoteIDs: []string{bvid},
		CrawlerMaxCount:  1,
	})
	detailRows := readLiveJSONRows(t, filepath.Join(detail.DataDir, "bilibili", "json", datedFile("detail_contents", "json")), detail.Output)
	if len(detailRows) != 1 || fmt.Sprint(detailRows[0]["bvid"]) != bvid {
		t.Fatalf("live detail should save requested bvid %s, rows=%#v\n%s", bvid, detailRows, detail.Output)
	}
	requireLiveVideoRows(t, detailRows, "", detail.Output)

	commentRun := runLiveBilibiliCLI(t, binaryPath, baseEnv, liveBilibiliConfig{
		CrawlerType:      "detail",
		SaveDataOption:   "sqlite",
		SpecifiedNoteIDs: []string{commentBvid},
		EnableComments:   true,
		CrawlerMaxCount:  1,
	})
	if countSQLiteRows(t, commentRun.SQLitePath, "bilibili_video", commentRun.Output) != 1 {
		t.Fatalf("live detail comments sqlite should save the video row\n%s", commentRun.Output)
	}
	comment := readSQLiteComment(t, commentRun.SQLitePath, commentRun.Output)
	if comment.CommentID == "" || comment.VideoID == "" || comment.Content == "" || comment.Nickname == "" {
		t.Fatalf("live detail comments should save real comment data, got %#v\n%s", comment, commentRun.Output)
	}

	creator := runLiveBilibiliCLI(t, binaryPath, baseEnv, liveBilibiliConfig{
		CrawlerType:      "creator",
		SaveDataOption:   "json",
		SpecifiedCreator: []string{creatorID},
		CrawlerMaxCount:  1,
	})
	creatorRows := readLiveJSONRows(t, filepath.Join(creator.DataDir, "bilibili", "json", datedFile("creator_creators", "json")), creator.Output)
	if len(creatorRows) != 1 || fmt.Sprint(creatorRows[0]["user_id"]) != creatorID {
		t.Fatalf("live creator should save requested creator %s, rows=%#v\n%s", creatorID, creatorRows, creator.Output)
	}
	if fmt.Sprint(creatorRows[0]["nickname"]) == "" || fmt.Sprint(creatorRows[0]["avatar"]) == "" {
		t.Fatalf("live creator should save profile fields, row=%#v\n%s", creatorRows[0], creator.Output)
	}
	creatorContentRows := readLiveJSONRows(t, filepath.Join(creator.DataDir, "bilibili", "json", datedFile("creator_contents", "json")), creator.Output)
	if len(creatorContentRows) != 1 {
		t.Fatalf("live creator should save one creator content row, got %d\n%s", len(creatorContentRows), creator.Output)
	}
	requireLiveVideoRows(t, creatorContentRows, "", creator.Output)

	homefeed := runLiveBilibiliCLI(t, binaryPath, baseEnv, liveBilibiliConfig{
		CrawlerType:     "homefeed",
		SaveDataOption:  "json",
		CrawlerMaxCount: 1,
	})
	homefeedRows := readLiveJSONRows(t, filepath.Join(homefeed.DataDir, "bilibili", "json", datedFile("homefeed_contents", "json")), homefeed.Output)
	if len(homefeedRows) != 1 {
		t.Fatalf("live homefeed should save one row, got %d\n%s", len(homefeedRows), homefeed.Output)
	}
	requireLiveVideoRows(t, homefeedRows, "", homefeed.Output)

	hotSearch := runLiveBilibiliCLI(t, binaryPath, baseEnv, liveBilibiliConfig{
		CrawlerType:       "hot_search",
		SaveDataOption:    "json",
		CrawlerMaxCount:   1,
		HotSearchMaxItems: 1,
	})
	hotRows := readLiveJSONRows(t, filepath.Join(hotSearch.DataDir, "bilibili", "json", datedFile("hot_search_contents", "json")), hotSearch.Output)
	if len(hotRows) != 1 {
		t.Fatalf("live hot_search should save one row, got %d\n%s", len(hotRows), hotSearch.Output)
	}
	requireLiveVideoRows(t, hotRows, "bilibili_hot_search", hotSearch.Output)

	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	dbRun := runLiveBilibiliCLI(t, binaryPath, append(baseEnv, mysqlServer.Env()...), liveBilibiliConfig{
		CrawlerType:     "search",
		SaveDataOption:  "db",
		Keywords:        []string{keyword},
		CrawlerMaxCount: 1,
	})
	db, err := sql.Open("mysql", mysqlServer.DSN())
	if err != nil {
		t.Fatalf("open live mysql db: %v", err)
	}
	defer db.Close()
	var mysqlCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM bilibili_video").Scan(&mysqlCount); err != nil {
		t.Fatalf("query live mysql bilibili_video: %v\n%s", err, dbRun.Output)
	}
	if mysqlCount != 1 {
		t.Fatalf("live mysql search should save one row, got %d\n%s", mysqlCount, dbRun.Output)
	}
	var mysqlBvid, mysqlTitle, mysqlSourceKeyword string
	if err := db.QueryRow("SELECT bvid, title, source_keyword FROM bilibili_video LIMIT 1").Scan(&mysqlBvid, &mysqlTitle, &mysqlSourceKeyword); err != nil {
		t.Fatalf("query live mysql bilibili_video fields: %v\n%s", err, dbRun.Output)
	}
	if mysqlBvid == "" || mysqlTitle == "" || mysqlSourceKeyword != keyword {
		t.Fatalf("live mysql video data mismatch: bvid=%q title=%q source_keyword=%q\n%s", mysqlBvid, mysqlTitle, mysqlSourceKeyword, dbRun.Output)
	}

	assertLiveCheckpointResume(t, binaryPath, baseEnv, liveBilibiliConfig{
		CrawlerType:      "search",
		SaveDataOption:   "json",
		Keywords:         []string{keyword},
		CrawlerMaxCount:  1,
		EnableCheckpoint: true,
	}, filepath.Join("bilibili", "json", datedFile("search_contents", "json")), "Resume from checkpoint")
	assertLiveCheckpointResume(t, binaryPath, baseEnv, liveBilibiliConfig{
		CrawlerType:      "detail",
		SaveDataOption:   "json",
		SpecifiedNoteIDs: []string{bvid},
		CrawlerMaxCount:  1,
		EnableCheckpoint: true,
	}, filepath.Join("bilibili", "json", datedFile("detail_contents", "json")), "Resume from checkpoint")
	assertLiveCheckpointResume(t, binaryPath, baseEnv, liveBilibiliConfig{
		CrawlerType:      "homefeed",
		SaveDataOption:   "json",
		CrawlerMaxCount:  1,
		EnableCheckpoint: true,
	}, filepath.Join("bilibili", "json", datedFile("homefeed_contents", "json")), "Resume from checkpoint")
	assertLiveCreatorCheckpointResume(t, binaryPath, baseEnv, creatorID)
}

func runLiveBilibiliCLI(t *testing.T, binaryPath string, env []string, opts liveBilibiliConfig) liveBilibiliRun {
	t.Helper()
	workDir, dataDir, sqlitePath := newLiveBilibiliWorkDir(t)
	return runLiveBilibiliCLIInWorkDir(t, binaryPath, env, workDir, dataDir, sqlitePath, opts)
}

func newLiveBilibiliWorkDir(t *testing.T) (string, string, string) {
	t.Helper()
	workDir, err := os.MkdirTemp("/tmp", "mcpro-bili-live-e2e-*")
	if err != nil {
		t.Fatalf("create live work dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(workDir) })
	dataDir := filepath.Join(workDir, "data")
	sqlitePath := filepath.Join(workDir, "media_crawler.db")
	return workDir, dataDir, sqlitePath
}

func runLiveBilibiliCLIInWorkDir(t *testing.T, binaryPath string, env []string, workDir, dataDir, sqlitePath string, opts liveBilibiliConfig) liveBilibiliRun {
	t.Helper()
	configDir := filepath.Join(workDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("create live config dir: %v", err)
	}
	configYAML := liveBilibiliConfigYAML(dataDir, sqlitePath, opts)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(configYAML), 0644); err != nil {
		t.Fatalf("write live config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
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
		t.Fatalf("live bilibili command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("live bilibili command failed: %v\n%s", err, output)
	}
	if strings.Contains(string(output), " ERROR ") {
		t.Fatalf("live bilibili command logged an error\n%s", output)
	}
	return liveBilibiliRun{WorkDir: workDir, DataDir: dataDir, SQLitePath: sqlitePath, Output: output}
}

func liveBilibiliConfigYAML(dataDir, sqlitePath string, opts liveBilibiliConfig) string {
	if opts.CrawlerType == "" {
		opts.CrawlerType = "search"
	}
	if opts.SaveDataOption == "" {
		opts.SaveDataOption = "json"
	}
	if len(opts.Keywords) == 0 {
		opts.Keywords = []string{"deepseek"}
	}
	if opts.CrawlerMaxCount == 0 {
		opts.CrawlerMaxCount = 1
	}
	if opts.HotSearchMaxItems == 0 {
		opts.HotSearchMaxItems = 1
	}
	var builder strings.Builder
	builder.WriteString("platform: \"bili\"\n")
	writeLiveList(&builder, "keywords", opts.Keywords)
	builder.WriteString(fmt.Sprintf("crawlerType: %q\n", opts.CrawlerType))
	builder.WriteString(fmt.Sprintf("saveDataOption: %q\n", opts.SaveDataOption))
	builder.WriteString("accountPoolSaveType: \"cookie_bridge\"\n")
	builder.WriteString("cookieBridgeURL: \"http://127.0.0.1:8274\"\n")
	builder.WriteString(fmt.Sprintf("dataDir: %q\n", dataDir))
	builder.WriteString(fmt.Sprintf("sqliteDbPath: %q\n", sqlitePath))
	builder.WriteString("startPage: 1\n")
	builder.WriteString(fmt.Sprintf("crawlerMaxNotesCount: %d\n", opts.CrawlerMaxCount))
	builder.WriteString("maxConcurrencyNum: 1\n")
	builder.WriteString("requestTimeOut: 20\n")
	builder.WriteString(fmt.Sprintf("hotSearchMaxItemsCount: %d\n", opts.HotSearchMaxItems))
	builder.WriteString("hotSearchPerItemContentCount: 1\n")
	builder.WriteString(fmt.Sprintf("enableGetComments: %s\n", strconv.FormatBool(opts.EnableComments)))
	builder.WriteString("enableGetSubComments: false\n")
	builder.WriteString(fmt.Sprintf("enableCheckpoint: %s\n", strconv.FormatBool(opts.EnableCheckpoint)))
	builder.WriteString("perNoteMaxComments: 1\n")
	builder.WriteString("crawlerTimeSleep: 0\n")
	builder.WriteString("hotSearchCheckpointMaxAgeSeconds: 21600\n")
	builder.WriteString("debugRequest: false\n")
	writeLiveList(&builder, "specifiedNoteIds", opts.SpecifiedNoteIDs)
	writeLiveList(&builder, "specifiedNoteURLs", opts.SpecifiedNoteIDs)
	writeLiveList(&builder, "specifiedCreatorIds", opts.SpecifiedCreator)
	writeLiveList(&builder, "specifiedCreatorURLs", opts.SpecifiedCreator)
	builder.WriteString("proxyConfig:\n  enableIpProxy: false\n")
	builder.WriteString("signSrvConfig:\n")
	builder.WriteString(fmt.Sprintf("  signSrvHost: %q\n", envOrDefault("MCPRO_BILIBILI_LIVE_SIGN_HOST", "127.0.0.1")))
	port, _ := strconv.Atoi(envOrDefault("MCPRO_BILIBILI_LIVE_SIGN_PORT", "8989"))
	builder.WriteString(fmt.Sprintf("  signSrvPort: %d\n", port))
	builder.WriteString("platformEndpoints: {}\n")
	builder.WriteString("dbConfig:\n  relationDbHost: \"127.0.0.1\"\n  relationDbPort: 3306\n  relationDbUser: \"root\"\n  relationDbPwd: \"\"\n  relationDbName: \"media_crawler_pro\"\n")
	return builder.String()
}

func writeLiveList(builder *strings.Builder, key string, values []string) {
	builder.WriteString(key + ":\n")
	for _, value := range values {
		builder.WriteString(fmt.Sprintf("  - %q\n", value))
	}
}

func readLiveJSONRows(t *testing.T, path string, output []byte) []map[string]any {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read live json %s: %v\n%s", path, err, output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal live json %s: %v\n%s", path, err, body)
	}
	return rows
}

func readLiveCSV(t *testing.T, path string, output []byte) [][]string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open live csv %s: %v\n%s", path, err, output)
	}
	defer file.Close()
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read live csv %s: %v\n%s", path, err, output)
	}
	return records
}

func countSQLiteRows(t *testing.T, dbPath, table string, output []byte) int64 {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open live sqlite db: %v\n%s", err, output)
	}
	var count int64
	if err := db.Table(table).Count(&count).Error; err != nil {
		t.Fatalf("count live sqlite table %s: %v\n%s", table, err, output)
	}
	return count
}

type liveSQLiteVideo struct {
	VideoID       string `gorm:"column:video_id"`
	Bvid          string `gorm:"column:bvid"`
	Title         string `gorm:"column:title"`
	SourceKeyword string `gorm:"column:source_keyword"`
}

type liveSQLiteComment struct {
	CommentID string `gorm:"column:comment_id"`
	VideoID   string `gorm:"column:video_id"`
	Content   string `gorm:"column:content"`
	Nickname  string `gorm:"column:nickname"`
}

func readSQLiteVideo(t *testing.T, dbPath string, output []byte) liveSQLiteVideo {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open live sqlite db: %v\n%s", err, output)
	}
	var row liveSQLiteVideo
	if err := db.Table("bilibili_video").Limit(1).First(&row).Error; err != nil {
		t.Fatalf("read live sqlite video: %v\n%s", err, output)
	}
	return row
}

func readSQLiteComment(t *testing.T, dbPath string, output []byte) liveSQLiteComment {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open live sqlite db: %v\n%s", err, output)
	}
	var row liveSQLiteComment
	if err := db.Table("bilibili_video_comment").Limit(1).First(&row).Error; err != nil {
		t.Fatalf("read live sqlite comment: %v\n%s", err, output)
	}
	return row
}

func requireLiveVideoRows(t *testing.T, rows []map[string]any, expectedSourceKeyword string, output []byte) {
	t.Helper()
	seen := map[string]struct{}{}
	for _, row := range rows {
		bvid := liveString(row, "bvid")
		videoID := liveString(row, "video_id")
		title := liveString(row, "title")
		userID := liveString(row, "user_id")
		if bvid == "" || videoID == "" || title == "" || userID == "" {
			t.Fatalf("live video row missing required fields: %#v\n%s", row, output)
		}
		if expectedSourceKeyword != "" && liveString(row, "source_keyword") != expectedSourceKeyword {
			t.Fatalf("live video row source_keyword mismatch, want %q row=%#v\n%s", expectedSourceKeyword, row, output)
		}
		if _, ok := seen[bvid]; ok {
			t.Fatalf("live video rows contain duplicate bvid %s: %#v\n%s", bvid, rows, output)
		}
		seen[bvid] = struct{}{}
	}
}

func pickLiveCommentableBVID(t *testing.T, rows []map[string]any) string {
	t.Helper()
	for _, row := range rows {
		commentCount, _ := strconv.Atoi(liveString(row, "video_comment"))
		if commentCount > 0 {
			return liveString(row, "bvid")
		}
	}
	t.Fatalf("live search did not return a video with comments: %#v", rows)
	return ""
}

func assertLiveCheckpointResume(t *testing.T, binaryPath string, env []string, opts liveBilibiliConfig, resultRelativePath string, resumeLog string) {
	t.Helper()
	workDir, dataDir, sqlitePath := newLiveBilibiliWorkDir(t)
	first := runLiveBilibiliCLIInWorkDir(t, binaryPath, env, workDir, dataDir, sqlitePath, opts)
	rowsAfterFirst := readLiveJSONRows(t, filepath.Join(dataDir, resultRelativePath), first.Output)
	if len(rowsAfterFirst) != 1 {
		t.Fatalf("checkpoint first run should save one row, got %d\n%s", len(rowsAfterFirst), first.Output)
	}
	second := runLiveBilibiliCLIInWorkDir(t, binaryPath, env, workDir, dataDir, sqlitePath, opts)
	if !strings.Contains(string(second.Output), resumeLog) {
		t.Fatalf("checkpoint second run should log %q\n%s", resumeLog, second.Output)
	}
	rowsAfterSecond := readLiveJSONRows(t, filepath.Join(dataDir, resultRelativePath), second.Output)
	if len(rowsAfterSecond) != 1 {
		t.Fatalf("checkpoint second run should not append duplicate rows, got %d\n%s", len(rowsAfterSecond), second.Output)
	}
}

func assertLiveCreatorCheckpointResume(t *testing.T, binaryPath string, env []string, creatorID string) {
	t.Helper()
	workDir, dataDir, sqlitePath := newLiveBilibiliWorkDir(t)
	opts := liveBilibiliConfig{
		CrawlerType:      "creator",
		SaveDataOption:   "json",
		SpecifiedCreator: []string{creatorID},
		CrawlerMaxCount:  1,
		EnableCheckpoint: true,
	}
	first := runLiveBilibiliCLIInWorkDir(t, binaryPath, env, workDir, dataDir, sqlitePath, opts)
	contentPath := filepath.Join(dataDir, "bilibili", "json", datedFile("creator_contents", "json"))
	rowsAfterFirst := readLiveJSONRows(t, contentPath, first.Output)
	if len(rowsAfterFirst) != 1 {
		t.Fatalf("creator checkpoint first run should save one content row, got %d\n%s", len(rowsAfterFirst), first.Output)
	}
	second := runLiveBilibiliCLIInWorkDir(t, binaryPath, env, workDir, dataDir, sqlitePath, opts)
	if !strings.Contains(string(second.Output), "Resume from checkpoint") {
		t.Fatalf("creator checkpoint second run should resume\n%s", second.Output)
	}
	rowsAfterSecond := readLiveJSONRows(t, contentPath, second.Output)
	if len(rowsAfterSecond) != 1 {
		t.Fatalf("creator checkpoint second run should not append duplicate content rows, got %d\n%s", len(rowsAfterSecond), second.Output)
	}
}

func liveString(row map[string]any, key string) string {
	value := row[key]
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func datedFile(prefix, ext string) string {
	return prefix + "_" + time.Now().Format("2006-01-02") + "." + ext
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
