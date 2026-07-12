// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/douyin_e2e_test.go
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

func TestCLIDouyinSearchJSONWithMockServers(t *testing.T) {
	result := runCLIDouyinSearchWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "douyin", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read douyin result file: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal douyin result json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one saved douyin row, got %d: %s", len(rows), body)
	}
	row := rows[0]
	if row["aweme_id"] != "aweme_e2e" {
		t.Fatalf("unexpected aweme_id: %#v", row["aweme_id"])
	}
	if row["aweme_type"] != "0" {
		t.Fatalf("unexpected aweme_type: %#v", row["aweme_type"])
	}
	if row["title"] != "Douyin E2E mock aweme" {
		t.Fatalf("unexpected title: %#v", row["title"])
	}
	if row["desc"] != "mock douyin desc" {
		t.Fatalf("unexpected desc: %#v", row["desc"])
	}
	if row["create_time"] != "1710000000" {
		t.Fatalf("unexpected create_time: %#v", row["create_time"])
	}
	if row["liked_count"] != "88" || row["comment_count"] != "5" || row["share_count"] != "3" || row["collected_count"] != "20" {
		t.Fatalf("unexpected count fields: %#v", row)
	}
	if row["aweme_url"] != "https://www.douyin.com/video/aweme_e2e" {
		t.Fatalf("unexpected aweme_url: %#v", row["aweme_url"])
	}
	if row["cover_url"] != "https://example.test/douyin-cover.png" {
		t.Fatalf("unexpected cover_url: %#v", row["cover_url"])
	}
	if row["video_download_url"] != "https://example.test/douyin-play.mp4" {
		t.Fatalf("unexpected video_download_url: %#v", row["video_download_url"])
	}
	if row["source_keyword"] != "mock-keyword" {
		t.Fatalf("unexpected source_keyword: %#v", row["source_keyword"])
	}
	if row["is_ai_generated"] != float64(1) {
		t.Fatalf("unexpected is_ai_generated: %#v", row["is_ai_generated"])
	}
	if row["user_id"] != "dy-author" || row["sec_uid"] != "sec-dy-author" || row["short_user_id"] != "123456" {
		t.Fatalf("unexpected author id fields: %#v", row)
	}
	if row["user_unique_id"] != "mock_douyin_author" || row["nickname"] != "mock douyin author" || row["avatar"] != "https://example.test/douyin-avatar.png" {
		t.Fatalf("unexpected author profile fields: %#v", row)
	}
	if row["user_signature"] != "mock signature" || row["ip_location"] != "上海" {
		t.Fatalf("unexpected author/ip fields: %#v", row)
	}
	if _, ok := row["add_ts"]; ok {
		t.Fatalf("douyin json output should match Python and omit add_ts: %#v", row["add_ts"])
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected last_modify_ts in douyin json row: %#v", row)
	}
}

func TestCLIDouyinSearchJSONCommentsWithMockServers(t *testing.T) {
	result := runCLIDouyinSearchWithMockServers(t, "json", true)

	resultPath := filepath.Join(result.dataDir, "douyin", "json", "search_comments_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read douyin comments result file: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal douyin comments json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected root and sub douyin comment rows, got %d: %s", len(rows), body)
	}

	root := rows[0]
	if root["comment_id"] != "dy_comment_root" || root["aweme_id"] != "aweme_e2e" || root["content"] != "root douyin comment" {
		t.Fatalf("unexpected root comment identity fields: %#v", root)
	}
	if root["create_time"] != "1710000100" || root["sub_comment_count"] != "1" || root["like_count"] != "7" {
		t.Fatalf("unexpected root comment count fields: %#v", root)
	}
	if root["parent_comment_id"] != "" || root["reply_to_reply_id"] != "" {
		t.Fatalf("unexpected root comment relation fields: %#v", root)
	}
	if root["pictures"] != "https://example.test/dy-comment.png" || root["ip_location"] != "北京" {
		t.Fatalf("unexpected root comment media/ip fields: %#v", root)
	}
	if root["user_id"] != "dy-commenter" || root["sec_uid"] != "sec-dy-commenter" || root["short_user_id"] != "654321" {
		t.Fatalf("unexpected root comment author id fields: %#v", root)
	}
	if root["user_unique_id"] != "mock_dy_commenter" || root["nickname"] != "mock dy commenter" || root["avatar"] != "https://example.test/dy-comment-avatar.png" || root["user_signature"] != "comment signature" {
		t.Fatalf("unexpected root comment author profile fields: %#v", root)
	}
	if _, ok := root["add_ts"]; ok {
		t.Fatalf("douyin comment json should match Python and omit add_ts: %#v", root["add_ts"])
	}
	if _, ok := root["last_modify_ts"]; !ok {
		t.Fatalf("expected root comment last_modify_ts: %#v", root)
	}

	sub := rows[1]
	if sub["comment_id"] != "dy_comment_sub" || sub["aweme_id"] != "aweme_e2e" || sub["content"] != "sub douyin comment" {
		t.Fatalf("unexpected sub comment identity fields: %#v", sub)
	}
	if sub["create_time"] != "1710000200" || sub["sub_comment_count"] != "0" || sub["like_count"] != "2" {
		t.Fatalf("unexpected sub comment count fields: %#v", sub)
	}
	if sub["parent_comment_id"] != "dy_comment_root" || sub["reply_to_reply_id"] != "dy_comment_root" {
		t.Fatalf("unexpected sub comment relation fields: %#v", sub)
	}
	if sub["pictures"] != "" || sub["ip_location"] != "广州" {
		t.Fatalf("unexpected sub comment media/ip fields: %#v", sub)
	}
	if sub["user_id"] != "dy-sub-commenter" || sub["sec_uid"] != "sec-dy-sub-commenter" || sub["short_user_id"] != "654322" {
		t.Fatalf("unexpected sub comment author id fields: %#v", sub)
	}
	if sub["user_unique_id"] != "mock_dy_sub_commenter" || sub["nickname"] != "mock dy sub commenter" || sub["avatar"] != "https://example.test/dy-sub-comment-avatar.png" || sub["user_signature"] != "sub comment signature" {
		t.Fatalf("unexpected sub comment author profile fields: %#v", sub)
	}
	if _, ok := sub["last_modify_ts"]; !ok {
		t.Fatalf("expected sub comment last_modify_ts: %#v", sub)
	}
}

func TestCLIDouyinSearchCSVWithMockServers(t *testing.T) {
	result := runCLIDouyinSearchWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "douyin", "1_search_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open douyin csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read douyin csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected csv header and one douyin row, got %d: %#v", len(records), records)
	}
	if csvCell(records, "aweme_id") != "aweme_e2e" {
		t.Fatalf("unexpected csv aweme_id: %#v", csvCell(records, "aweme_id"))
	}
	if csvCell(records, "aweme_type") != "0" {
		t.Fatalf("unexpected csv aweme_type: %#v", csvCell(records, "aweme_type"))
	}
	if csvCell(records, "title") != "Douyin E2E mock aweme" {
		t.Fatalf("unexpected csv title: %#v", csvCell(records, "title"))
	}
	if csvCell(records, "create_time") != "1710000000" {
		t.Fatalf("unexpected csv create_time: %#v", csvCell(records, "create_time"))
	}
	if csvCell(records, "liked_count") != "88" || csvCell(records, "comment_count") != "5" || csvCell(records, "share_count") != "3" || csvCell(records, "collected_count") != "20" {
		t.Fatalf("unexpected csv count fields: %#v", records)
	}
	if csvCell(records, "cover_url") != "https://example.test/douyin-cover.png" {
		t.Fatalf("unexpected csv cover_url: %#v", csvCell(records, "cover_url"))
	}
	if csvCell(records, "video_download_url") != "https://example.test/douyin-play.mp4" {
		t.Fatalf("unexpected csv video_download_url: %#v", csvCell(records, "video_download_url"))
	}
	if csvCell(records, "source_keyword") != "mock-keyword" {
		t.Fatalf("unexpected csv source_keyword: %#v", csvCell(records, "source_keyword"))
	}
	if csvCell(records, "is_ai_generated") != "1" {
		t.Fatalf("unexpected csv is_ai_generated: %#v", csvCell(records, "is_ai_generated"))
	}
	if csvCell(records, "nickname") != "mock douyin author" || csvCell(records, "ip_location") != "上海" {
		t.Fatalf("unexpected csv author/ip fields: %#v", records)
	}
	if csvCell(records, "last_modify_ts") == "" {
		t.Fatalf("expected non-empty csv last_modify_ts")
	}
}

func TestCLIDouyinSearchCSVCommentsWithMockServers(t *testing.T) {
	result := runCLIDouyinSearchWithMockServers(t, "csv", true)

	resultPath := filepath.Join(result.dataDir, "douyin", "1_search_comments_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open douyin comments csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read douyin comments csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 3 {
		t.Fatalf("expected csv header plus root/sub douyin comments, got %d: %#v", len(records), records)
	}
	if csvCellAt(records, 1, "comment_id") != "dy_comment_root" || csvCellAt(records, 1, "content") != "root douyin comment" {
		t.Fatalf("unexpected root comments csv identity fields: %#v", records[1])
	}
	if csvCellAt(records, 1, "create_time") != "1710000100" || csvCellAt(records, 1, "sub_comment_count") != "1" || csvCellAt(records, 1, "like_count") != "7" {
		t.Fatalf("unexpected root comments csv metric fields: %#v", records[1])
	}
	if csvCellAt(records, 1, "pictures") != "https://example.test/dy-comment.png" || csvCellAt(records, 1, "ip_location") != "北京" {
		t.Fatalf("unexpected root comments csv media/ip fields: %#v", records[1])
	}
	if csvCellAt(records, 1, "user_id") != "dy-commenter" || csvCellAt(records, 1, "nickname") != "mock dy commenter" {
		t.Fatalf("unexpected root comments csv author fields: %#v", records[1])
	}
	if csvCellAt(records, 1, "last_modify_ts") == "" {
		t.Fatalf("expected root comments csv last_modify_ts")
	}

	if csvCellAt(records, 2, "comment_id") != "dy_comment_sub" || csvCellAt(records, 2, "content") != "sub douyin comment" {
		t.Fatalf("unexpected sub comments csv identity fields: %#v", records[2])
	}
	if csvCellAt(records, 2, "parent_comment_id") != "dy_comment_root" || csvCellAt(records, 2, "reply_to_reply_id") != "dy_comment_root" {
		t.Fatalf("unexpected sub comments csv relation fields: %#v", records[2])
	}
	if csvCellAt(records, 2, "user_id") != "dy-sub-commenter" || csvCellAt(records, 2, "nickname") != "mock dy sub commenter" {
		t.Fatalf("unexpected sub comments csv author fields: %#v", records[2])
	}
}

func TestCLIDouyinSearchSQLiteWithMockServers(t *testing.T) {
	result := runCLIDouyinSearchWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open douyin sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	var row struct {
		AwemeID          string `gorm:"column:aweme_id"`
		AwemeType        string `gorm:"column:aweme_type"`
		Title            string `gorm:"column:title"`
		Desc             string `gorm:"column:desc"`
		CreateTime       int64  `gorm:"column:create_time"`
		LikedCount       string `gorm:"column:liked_count"`
		CommentCount     string `gorm:"column:comment_count"`
		ShareCount       string `gorm:"column:share_count"`
		CollectedCount   string `gorm:"column:collected_count"`
		AwemeURL         string `gorm:"column:aweme_url"`
		CoverURL         string `gorm:"column:cover_url"`
		VideoDownloadURL string `gorm:"column:video_download_url"`
		SourceKeyword    string `gorm:"column:source_keyword"`
		IsAIGenerated    int    `gorm:"column:is_ai_generated"`
		UserID           string `gorm:"column:user_id"`
		SecUID           string `gorm:"column:sec_uid"`
		ShortUserID      string `gorm:"column:short_user_id"`
		UserUniqueID     string `gorm:"column:user_unique_id"`
		Nickname         string `gorm:"column:nickname"`
		Avatar           string `gorm:"column:avatar"`
		UserSignature    string `gorm:"column:user_signature"`
		IPLocation       string `gorm:"column:ip_location"`
		AddTs            int64  `gorm:"column:add_ts"`
		LastModifyTs     int64  `gorm:"column:last_modify_ts"`
	}
	if err := db.Table("douyin_aweme").Where("aweme_id = ?", "aweme_e2e").First(&row).Error; err != nil {
		t.Fatalf("query douyin sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	if row.AwemeType != "0" || row.Title != "Douyin E2E mock aweme" || row.Desc != "mock douyin desc" {
		t.Fatalf("unexpected sqlite identity fields: %#v", row)
	}
	if row.CreateTime != 1710000000 || row.LikedCount != "88" || row.CommentCount != "5" || row.ShareCount != "3" || row.CollectedCount != "20" {
		t.Fatalf("unexpected sqlite metric fields: %#v", row)
	}
	if row.AwemeURL != "https://www.douyin.com/video/aweme_e2e" || row.CoverURL != "https://example.test/douyin-cover.png" || row.VideoDownloadURL != "https://example.test/douyin-play.mp4" {
		t.Fatalf("unexpected sqlite url fields: %#v", row)
	}
	if row.SourceKeyword != "mock-keyword" || row.IsAIGenerated != 1 {
		t.Fatalf("unexpected sqlite source/aigc fields: %#v", row)
	}
	if row.UserID != "dy-author" || row.SecUID != "sec-dy-author" || row.ShortUserID != "123456" || row.UserUniqueID != "mock_douyin_author" {
		t.Fatalf("unexpected sqlite author id fields: %#v", row)
	}
	if row.Nickname != "mock douyin author" || row.Avatar != "https://example.test/douyin-avatar.png" || row.UserSignature != "mock signature" || row.IPLocation != "上海" {
		t.Fatalf("unexpected sqlite author profile fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected non-zero sqlite timestamps: %#v", row)
	}
}

func TestCLIDouyinSearchSQLiteCommentsWithMockServers(t *testing.T) {
	result := runCLIDouyinSearchWithMockServers(t, "sqlite", true)

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open douyin comments sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []struct {
		CommentID       string `gorm:"column:comment_id"`
		AwemeID         string `gorm:"column:aweme_id"`
		Content         string `gorm:"column:content"`
		CreateTime      int64  `gorm:"column:create_time"`
		SubCommentCount string `gorm:"column:sub_comment_count"`
		ParentCommentID string `gorm:"column:parent_comment_id"`
		ReplyToReplyID  string `gorm:"column:reply_to_reply_id"`
		LikeCount       string `gorm:"column:like_count"`
		Pictures        string `gorm:"column:pictures"`
		IPLocation      string `gorm:"column:ip_location"`
		UserID          string `gorm:"column:user_id"`
		SecUID          string `gorm:"column:sec_uid"`
		ShortUserID     string `gorm:"column:short_user_id"`
		UserUniqueID    string `gorm:"column:user_unique_id"`
		Nickname        string `gorm:"column:nickname"`
		Avatar          string `gorm:"column:avatar"`
		UserSignature   string `gorm:"column:user_signature"`
		AddTs           int64  `gorm:"column:add_ts"`
		LastModifyTs    int64  `gorm:"column:last_modify_ts"`
	}
	if err := db.Table("douyin_aweme_comment").Order("id").Find(&rows).Error; err != nil {
		t.Fatalf("query douyin comments sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two sqlite comment rows, got %d: %#v", len(rows), rows)
	}
	root := rows[0]
	if root.CommentID != "dy_comment_root" || root.AwemeID != "aweme_e2e" || root.Content != "root douyin comment" {
		t.Fatalf("unexpected root sqlite comment identity fields: %#v", root)
	}
	if root.CreateTime != 1710000100 || root.SubCommentCount != "1" || root.LikeCount != "7" {
		t.Fatalf("unexpected root sqlite comment metric fields: %#v", root)
	}
	if root.Pictures != "https://example.test/dy-comment.png" || root.IPLocation != "北京" {
		t.Fatalf("unexpected root sqlite comment media/ip fields: %#v", root)
	}
	if root.UserID != "dy-commenter" || root.SecUID != "sec-dy-commenter" || root.ShortUserID != "654321" || root.UserUniqueID != "mock_dy_commenter" {
		t.Fatalf("unexpected root sqlite comment author id fields: %#v", root)
	}
	if root.Nickname != "mock dy commenter" || root.Avatar != "https://example.test/dy-comment-avatar.png" || root.UserSignature != "comment signature" {
		t.Fatalf("unexpected root sqlite comment author profile fields: %#v", root)
	}
	if root.AddTs == 0 || root.LastModifyTs == 0 {
		t.Fatalf("expected root sqlite comment timestamps: %#v", root)
	}

	sub := rows[1]
	if sub.CommentID != "dy_comment_sub" || sub.AwemeID != "aweme_e2e" || sub.Content != "sub douyin comment" {
		t.Fatalf("unexpected sub sqlite comment identity fields: %#v", sub)
	}
	if sub.CreateTime != 1710000200 || sub.SubCommentCount != "0" || sub.LikeCount != "2" {
		t.Fatalf("unexpected sub sqlite comment metric fields: %#v", sub)
	}
	if sub.ParentCommentID != "dy_comment_root" || sub.ReplyToReplyID != "dy_comment_root" {
		t.Fatalf("unexpected sub sqlite comment relation fields: %#v", sub)
	}
	if sub.UserID != "dy-sub-commenter" || sub.SecUID != "sec-dy-sub-commenter" || sub.ShortUserID != "654322" || sub.UserUniqueID != "mock_dy_sub_commenter" {
		t.Fatalf("unexpected sub sqlite comment author id fields: %#v", sub)
	}
	if sub.Nickname != "mock dy sub commenter" || sub.Avatar != "https://example.test/dy-sub-comment-avatar.png" || sub.UserSignature != "sub comment signature" {
		t.Fatalf("unexpected sub sqlite comment author profile fields: %#v", sub)
	}
	if sub.AddTs == 0 || sub.LastModifyTs == 0 {
		t.Fatalf("expected sub sqlite comment timestamps: %#v", sub)
	}
}

func TestCLIDouyinSearchCheckpointResumeSkipsCompletedPages(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	searchCallsByOffset := make(map[string]int)
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
							"client_id": "dy-resume-client",
							"platforms": map[string]any{
								"dy": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"dy": "mock-douyin"},
						},
					},
				},
			})
		case "/api/cookies/dy":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "sessionid=resume-session; msToken=resume-token"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/douyin/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"a_bogus": "resume-a-bogus"},
			})
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/general/search/single/":
			offset := r.URL.Query().Get("offset")
			searchCallsByOffset[offset]++
			switch offset {
			case "0":
				writeJSON(t, w, douyinResumeSearchPayload("aweme_resume_1", "Douyin resume aweme 1", "resume-logid-1"))
			case "20":
				if searchCallsByOffset[offset] == 1 {
					writeJSON(t, w, map[string]any{
						"status_code": 0,
						"data":        []map[string]any{},
						"extra":       map[string]string{"logid": "resume-logid-empty-first"},
					})
					return
				}
				writeJSON(t, w, douyinResumeSearchPayload("aweme_resume_2", "Douyin resume aweme 2", "resume-logid-2"))
			default:
				writeJSON(t, w, map[string]any{
					"status_code": 0,
					"data":        []map[string]any{},
					"extra":       map[string]string{"logid": "resume-logid-empty"},
				})
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
platform: "dy"
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
perNoteMaxComments: 2
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  douyinApiUrl: %q
  douyinIndexUrl: %q
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
			"MCPRO_DOUYIN_API_URL="+mockServer.URL,
			"MCPRO_DOUYIN_INDEX_URL="+mockServer.URL,
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

	writeResumeConfig(20)
	firstOutput := runCrawler()
	if searchCallsByOffset["0"] != 1 {
		t.Fatalf("first run did not crawl offset 0 once, calls=%#v\n%s", searchCallsByOffset, firstOutput)
	}
	if searchCallsByOffset["20"] != 1 {
		t.Fatalf("first run should probe offset 20 once and stop on empty page, calls=%#v\n%s", searchCallsByOffset, firstOutput)
	}

	writeResumeConfig(40)
	secondOutput := runCrawler()
	if searchCallsByOffset["0"] != 1 {
		t.Fatalf("resume should not re-fetch completed offset 0, calls=%#v\n%s", searchCallsByOffset, secondOutput)
	}
	if searchCallsByOffset["20"] != 2 {
		t.Fatalf("resume did not crawl offset 20 once, calls=%#v\n%s", searchCallsByOffset, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "douyin", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read douyin resume json: %v\n%s", err, secondOutput)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal douyin resume json: %v\n%s", err, body)
	}
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Douyin rows after resume, got %d: %s", len(rows), body)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["aweme_id"])] = true
	}
	if !seen["aweme_resume_1"] || !seen["aweme_resume_2"] {
		t.Fatalf("resume output missing expected aweme IDs: seen=%#v rows=%s", seen, body)
	}
}

func TestCLIDouyinDetailCheckpointResumeSkipsCompletedIDs(t *testing.T) {
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
							"client_id": "dy-detail-resume-client",
							"platforms": map[string]any{
								"dy": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"dy": "mock-douyin"},
						},
					},
				},
			})
		case "/api/cookies/dy":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "sessionid=detail-resume-session; msToken=detail-resume-token"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/douyin/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"a_bogus": "detail-resume-a-bogus"},
			})
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/aweme/detail/":
			awemeID := r.URL.Query().Get("aweme_id")
			detailCallsByID[awemeID]++
			switch awemeID {
			case "aweme_detail_resume_1":
				writeJSON(t, w, map[string]any{
					"status_code":  0,
					"aweme_detail": douyinMockHotResumeAweme(awemeID, "Douyin detail resume aweme 1", "dy-detail-resume-author-1"),
				})
			case "aweme_detail_resume_2":
				if !allowSecondDetail {
					http.Error(w, "transient second detail failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, map[string]any{
					"status_code":  0,
					"aweme_detail": douyinMockHotResumeAweme(awemeID, "Douyin detail resume aweme 2", "dy-detail-resume-author-2"),
				})
			default:
				t.Fatalf("unexpected douyin detail resume aweme id: %s", awemeID)
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
platform: "dy"
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
perNoteMaxComments: 2
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  douyinApiUrl: %q
  douyinIndexUrl: %q
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
		runCmd := exec.CommandContext(ctx, binaryPath,
			"--platform", "dy",
			"--type", "detail",
			"--urls", "aweme_detail_resume_1,aweme_detail_resume_2",
			"--save_data_option", "json",
			"--no-enable_comments",
			"--enable_checkpoint",
		)
		runCmd.Dir = tmpDir
		runCmd.Env = append(os.Environ(),
			"RELATION_DB_HOST=127.0.0.1",
			"RELATION_DB_PORT=1",
			"MCPRO_DATA_DIR="+dataDir,
			"MCPRO_DOUYIN_API_URL="+mockServer.URL,
			"MCPRO_DOUYIN_INDEX_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("douyin detail resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("douyin detail resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if detailCallsByID["aweme_detail_resume_1"] != 1 || detailCallsByID["aweme_detail_resume_2"] != 1 {
		t.Fatalf("first douyin detail run should fetch both IDs and fail second once, calls=%#v\n%s", detailCallsByID, firstOutput)
	}

	allowSecondDetail = true
	secondOutput := runCrawler()
	if detailCallsByID["aweme_detail_resume_1"] != 1 {
		t.Fatalf("resume should skip completed first douyin detail ID, calls=%#v\n%s", detailCallsByID, secondOutput)
	}
	if detailCallsByID["aweme_detail_resume_2"] != 2 {
		t.Fatalf("resume should retry second douyin detail ID exactly once, calls=%#v\n%s", detailCallsByID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "douyin", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Douyin detail rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["aweme_id"])] = true
	}
	if !seen["aweme_detail_resume_1"] || !seen["aweme_detail_resume_2"] {
		t.Fatalf("douyin detail resume output missing expected aweme IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIDouyinDetailJSONWithMockServers(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

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
							"client_id": "dy-detail-client",
							"platforms": map[string]any{
								"dy": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"dy": "mock-douyin"},
						},
					},
				},
			})
		case "/api/cookies/dy":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "sessionid=detail-session; msToken=detail-token"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/douyin/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"a_bogus": "detail-a-bogus"},
			})
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/aweme/detail/":
			detailCalls++
			if got := r.URL.Query().Get("aweme_id"); got != "aweme_detail_e2e" {
				t.Fatalf("unexpected detail aweme_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"status_code":  0,
				"aweme_detail": douyinMockDetailAweme(),
			})
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
platform: "dy"
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
  douyinApiUrl: %q
  douyinIndexUrl: %q
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
	runCmd := exec.CommandContext(ctx, binaryPath,
		"--platform", "dy",
		"--type", "detail",
		"--urls", "aweme_detail_e2e",
		"--save_data_option", "json",
		"--no-enable_comments",
		"--no-enable_checkpoint",
	)
	runCmd.Dir = tmpDir
	runCmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+dataDir,
		"MCPRO_DOUYIN_API_URL="+mockServer.URL,
		"MCPRO_DOUYIN_INDEX_URL="+mockServer.URL,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if detailCalls != 1 {
		t.Fatalf("expected one douyin detail request, got %d\n%s", detailCalls, output)
	}

	resultPath := filepath.Join(dataDir, "douyin", "json", "detail_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read douyin detail result file: %v\ncommand output:\n%s", err, output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal douyin detail json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one saved douyin detail row, got %d: %s", len(rows), body)
	}
	row := rows[0]
	if row["aweme_id"] != "aweme_detail_e2e" || row["title"] != "Douyin detail E2E mock aweme" || row["desc"] != "mock douyin detail desc" {
		t.Fatalf("unexpected detail identity fields: %#v", row)
	}
	if row["create_time"] != "1710000300" || row["liked_count"] != "188" || row["comment_count"] != "15" || row["share_count"] != "13" || row["collected_count"] != "120" {
		t.Fatalf("unexpected detail metric fields: %#v", row)
	}
	if row["aweme_url"] != "https://www.douyin.com/video/aweme_detail_e2e" || row["cover_url"] != "https://example.test/douyin-detail-cover.png" || row["video_download_url"] != "https://example.test/douyin-detail-play.mp4" {
		t.Fatalf("unexpected detail url fields: %#v", row)
	}
	if row["source_keyword"] != "" || row["is_ai_generated"] != float64(0) {
		t.Fatalf("unexpected detail source/aigc fields: %#v", row)
	}
	if row["user_id"] != "dy-detail-author" || row["sec_uid"] != "sec-dy-detail-author" || row["short_user_id"] != "223344" || row["user_unique_id"] != "mock_douyin_detail_author" {
		t.Fatalf("unexpected detail author id fields: %#v", row)
	}
	if row["nickname"] != "mock douyin detail author" || row["avatar"] != "https://example.test/douyin-detail-avatar.png" || row["user_signature"] != "detail signature" || row["ip_location"] != "杭州" {
		t.Fatalf("unexpected detail author profile fields: %#v", row)
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected detail last_modify_ts: %#v", row)
	}
}

func TestCLIDouyinDetailCSVWithMockServers(t *testing.T) {
	result := runCLIDouyinDetailWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "douyin", "1_detail_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open douyin detail csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read douyin detail csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected detail csv header and one row, got %d: %#v", len(records), records)
	}
	if csvCell(records, "aweme_id") != "aweme_detail_e2e" || csvCell(records, "title") != "Douyin detail E2E mock aweme" {
		t.Fatalf("unexpected detail csv identity fields: %#v", records)
	}
	if csvCell(records, "create_time") != "1710000300" || csvCell(records, "liked_count") != "188" || csvCell(records, "comment_count") != "15" || csvCell(records, "share_count") != "13" || csvCell(records, "collected_count") != "120" {
		t.Fatalf("unexpected detail csv metric fields: %#v", records)
	}
	if csvCell(records, "cover_url") != "https://example.test/douyin-detail-cover.png" || csvCell(records, "video_download_url") != "https://example.test/douyin-detail-play.mp4" {
		t.Fatalf("unexpected detail csv url fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "" || csvCell(records, "is_ai_generated") != "0" {
		t.Fatalf("unexpected detail csv source/aigc fields: %#v", records)
	}
	if csvCell(records, "user_id") != "dy-detail-author" || csvCell(records, "nickname") != "mock douyin detail author" || csvCell(records, "ip_location") != "杭州" {
		t.Fatalf("unexpected detail csv author fields: %#v", records)
	}
}

func TestCLIDouyinDetailSQLiteWithMockServers(t *testing.T) {
	result := runCLIDouyinDetailWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open douyin detail sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	var row struct {
		AwemeID          string `gorm:"column:aweme_id"`
		Title            string `gorm:"column:title"`
		Desc             string `gorm:"column:desc"`
		CreateTime       int64  `gorm:"column:create_time"`
		LikedCount       string `gorm:"column:liked_count"`
		CommentCount     string `gorm:"column:comment_count"`
		ShareCount       string `gorm:"column:share_count"`
		CollectedCount   string `gorm:"column:collected_count"`
		AwemeURL         string `gorm:"column:aweme_url"`
		CoverURL         string `gorm:"column:cover_url"`
		VideoDownloadURL string `gorm:"column:video_download_url"`
		SourceKeyword    string `gorm:"column:source_keyword"`
		IsAIGenerated    int    `gorm:"column:is_ai_generated"`
		UserID           string `gorm:"column:user_id"`
		SecUID           string `gorm:"column:sec_uid"`
		ShortUserID      string `gorm:"column:short_user_id"`
		UserUniqueID     string `gorm:"column:user_unique_id"`
		Nickname         string `gorm:"column:nickname"`
		Avatar           string `gorm:"column:avatar"`
		UserSignature    string `gorm:"column:user_signature"`
		IPLocation       string `gorm:"column:ip_location"`
		AddTs            int64  `gorm:"column:add_ts"`
		LastModifyTs     int64  `gorm:"column:last_modify_ts"`
	}
	if err := db.Table("douyin_aweme").Where("aweme_id = ?", "aweme_detail_e2e").First(&row).Error; err != nil {
		t.Fatalf("query douyin detail sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	if row.Title != "Douyin detail E2E mock aweme" || row.Desc != "mock douyin detail desc" {
		t.Fatalf("unexpected detail sqlite identity fields: %#v", row)
	}
	if row.CreateTime != 1710000300 || row.LikedCount != "188" || row.CommentCount != "15" || row.ShareCount != "13" || row.CollectedCount != "120" {
		t.Fatalf("unexpected detail sqlite metric fields: %#v", row)
	}
	if row.AwemeURL != "https://www.douyin.com/video/aweme_detail_e2e" || row.CoverURL != "https://example.test/douyin-detail-cover.png" || row.VideoDownloadURL != "https://example.test/douyin-detail-play.mp4" {
		t.Fatalf("unexpected detail sqlite url fields: %#v", row)
	}
	if row.SourceKeyword != "" || row.IsAIGenerated != 0 {
		t.Fatalf("unexpected detail sqlite source/aigc fields: %#v", row)
	}
	if row.UserID != "dy-detail-author" || row.SecUID != "sec-dy-detail-author" || row.ShortUserID != "223344" || row.UserUniqueID != "mock_douyin_detail_author" {
		t.Fatalf("unexpected detail sqlite author id fields: %#v", row)
	}
	if row.Nickname != "mock douyin detail author" || row.Avatar != "https://example.test/douyin-detail-avatar.png" || row.UserSignature != "detail signature" || row.IPLocation != "杭州" {
		t.Fatalf("unexpected detail sqlite author profile fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected detail sqlite timestamps: %#v", row)
	}
}

func TestCLIDouyinCreatorCheckpointResumeByCreatorAndCursor(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	allowSecondCursor := false
	profileCalls := 0
	postCallsByCursor := make(map[string]int)
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
							"client_id": "dy-creator-resume-client",
							"platforms": map[string]any{
								"dy": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"dy": "mock-douyin"},
						},
					},
				},
			})
		case "/api/cookies/dy":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "sessionid=creator-resume-session; msToken=creator-resume-token"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/douyin/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"a_bogus": "creator-resume-a-bogus"},
			})
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/user/profile/other/":
			profileCalls++
			if got := r.URL.Query().Get("sec_user_id"); got != "sec-dy-creator-resume" {
				t.Fatalf("unexpected creator resume sec_user_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"status_code": 0,
				"user":        douyinMockCreator(),
			})
		case "/aweme/v1/web/aweme/post/":
			if got := r.URL.Query().Get("sec_user_id"); got != "sec-dy-creator-resume" {
				t.Fatalf("unexpected creator resume post sec_user_id: %s", got)
			}
			cursor := r.URL.Query().Get("max_cursor")
			postCallsByCursor[cursor]++
			switch cursor {
			case "0":
				writeJSON(t, w, map[string]any{
					"status_code": 0,
					"has_more":    1,
					"max_cursor":  "18",
					"aweme_list": []map[string]any{
						douyinMockCreatorAwemeWithID("aweme_creator_resume_1", "Douyin creator resume aweme 1"),
					},
				})
			case "18":
				if !allowSecondCursor {
					writeJSON(t, w, map[string]any{
						"status_code": 0,
						"has_more":    0,
						"max_cursor":  "18",
						"aweme_list":  []map[string]any{},
					})
					return
				}
				writeJSON(t, w, map[string]any{
					"status_code": 0,
					"has_more":    0,
					"max_cursor":  "36",
					"aweme_list": []map[string]any{
						douyinMockCreatorAwemeWithID("aweme_creator_resume_2", "Douyin creator resume aweme 2"),
					},
				})
			default:
				t.Fatalf("unexpected creator resume max_cursor: %s", cursor)
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
platform: "dy"
keywords:
  - "unused"
crawlerType: "creator"
specifiedCreatorIds:
  - "sec-dy-creator-resume"
specifiedCreatorURLs:
  - "sec-dy-creator-resume"
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
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  douyinApiUrl: %q
  douyinIndexUrl: %q
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
			"MCPRO_DOUYIN_API_URL="+mockServer.URL,
			"MCPRO_DOUYIN_INDEX_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("douyin creator resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("douyin creator resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if profileCalls != 1 || postCallsByCursor["0"] != 1 || postCallsByCursor["18"] != 1 {
		t.Fatalf("first creator run should fetch profile, cursor 0, then stop at cursor 18, profile=%d posts=%#v\n%s", profileCalls, postCallsByCursor, firstOutput)
	}

	allowSecondCursor = true
	secondOutput := runCrawler()
	if profileCalls != 2 {
		t.Fatalf("Douyin creator resume should match Python and refresh creator profile, profile=%d\n%s", profileCalls, secondOutput)
	}
	if postCallsByCursor["0"] != 1 {
		t.Fatalf("resume should not re-fetch completed Douyin creator cursor 0, posts=%#v\n%s", postCallsByCursor, secondOutput)
	}
	if postCallsByCursor["18"] != 2 {
		t.Fatalf("resume should retry Douyin creator cursor 18 once, posts=%#v\n%s", postCallsByCursor, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "douyin", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Douyin creator rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["aweme_id"])] = true
	}
	if !seen["aweme_creator_resume_1"] || !seen["aweme_creator_resume_2"] {
		t.Fatalf("creator resume output missing expected Douyin aweme IDs, seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIDouyinHomefeedCheckpointResumeSkipsCompletedRefreshIndex(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	allowSecondPage := false
	feedCallsByRefreshIndex := make(map[string]int)
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
							"client_id": "dy-homefeed-resume-client",
							"platforms": map[string]any{
								"dy": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"dy": "mock-douyin"},
						},
					},
				},
			})
		case "/api/cookies/dy":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "sessionid=homefeed-resume-session; msToken=homefeed-resume-token"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/douyin/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"a_bogus": "homefeed-resume-a-bogus"},
			})
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/module/feed/":
			refreshIndex := r.URL.Query().Get("refresh_index")
			feedCallsByRefreshIndex[refreshIndex]++
			switch refreshIndex {
			case "0":
				writeJSON(t, w, douyinHomefeedPagePayload(douyinMockHomefeedAwemeWithID("aweme_homefeed_resume_1", "Douyin homefeed resume aweme 1")))
			case "20":
				if !allowSecondPage {
					writeJSON(t, w, map[string]any{
						"StatusCode": 1,
						"cards":      []map[string]any{},
					})
					return
				}
				writeJSON(t, w, douyinHomefeedPagePayload(douyinMockHomefeedAwemeWithID("aweme_homefeed_resume_2", "Douyin homefeed resume aweme 2")))
			case "40":
				writeJSON(t, w, map[string]any{
					"StatusCode": 1,
					"cards":      []map[string]any{},
				})
			default:
				t.Fatalf("unexpected homefeed resume refresh_index: %s", refreshIndex)
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
platform: "dy"
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
perNoteMaxComments: 2
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  douyinApiUrl: %q
  douyinIndexUrl: %q
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
			"MCPRO_DOUYIN_API_URL="+mockServer.URL,
			"MCPRO_DOUYIN_INDEX_URL="+mockServer.URL,
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("douyin homefeed resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("douyin homefeed resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if feedCallsByRefreshIndex["0"] != 1 || feedCallsByRefreshIndex["20"] != 1 {
		t.Fatalf("first homefeed run should crawl refresh_index 0 then stop at 20, calls=%#v\n%s", feedCallsByRefreshIndex, firstOutput)
	}

	allowSecondPage = true
	secondOutput := runCrawler()
	if feedCallsByRefreshIndex["0"] != 1 {
		t.Fatalf("resume should not re-fetch completed Douyin homefeed refresh_index 0, calls=%#v\n%s", feedCallsByRefreshIndex, secondOutput)
	}
	if feedCallsByRefreshIndex["20"] != 2 {
		t.Fatalf("resume should retry Douyin homefeed refresh_index 20 once, calls=%#v\n%s", feedCallsByRefreshIndex, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "douyin", "json", "homefeed_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Douyin homefeed rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["aweme_id"])] = true
	}
	if !seen["aweme_homefeed_resume_1"] || !seen["aweme_homefeed_resume_2"] {
		t.Fatalf("homefeed resume output missing expected Douyin aweme IDs, seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIDouyinHotSearchJSONWithMockServers(t *testing.T) {
	result := runCLIDouyinHotSearchWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "douyin", "json", "hot_search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read douyin hot_search json result: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal douyin hot_search json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one douyin hot_search row, got %d: %s", len(rows), body)
	}
	row := rows[0]
	if row["aweme_id"] != "aweme_hot_e2e" || row["title"] != "Douyin hot E2E mock aweme" || row["desc"] != "mock douyin hot desc" {
		t.Fatalf("unexpected hot_search identity fields: %#v", row)
	}
	if row["create_time"] != "1710000400" || row["liked_count"] != "288" || row["comment_count"] != "25" || row["share_count"] != "23" || row["collected_count"] != "220" {
		t.Fatalf("unexpected hot_search metric fields: %#v", row)
	}
	if row["aweme_url"] != "https://www.douyin.com/video/aweme_hot_e2e" || row["cover_url"] != "https://example.test/douyin-hot-cover.png" || row["video_download_url"] != "https://example.test/douyin-hot-play.mp4" {
		t.Fatalf("unexpected hot_search url fields: %#v", row)
	}
	if row["source_keyword"] != "mock hot douyin" || row["is_ai_generated"] != float64(0) {
		t.Fatalf("unexpected hot_search source/aigc fields: %#v", row)
	}
	if row["user_id"] != "dy-hot-author" || row["sec_uid"] != "sec-dy-hot-author" || row["short_user_id"] != "334455" || row["user_unique_id"] != "mock_douyin_hot_author" {
		t.Fatalf("unexpected hot_search author id fields: %#v", row)
	}
	if row["nickname"] != "mock douyin hot author" || row["avatar"] != "https://example.test/douyin-hot-avatar.png" || row["user_signature"] != "hot signature" || row["ip_location"] != "深圳" {
		t.Fatalf("unexpected hot_search author profile fields: %#v", row)
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected hot_search last_modify_ts: %#v", row)
	}
}

func TestCLIDouyinHotSearchCSVWithMockServers(t *testing.T) {
	result := runCLIDouyinHotSearchWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "douyin", "1_hot_search_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open douyin hot_search csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read douyin hot_search csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected hot_search csv header and one row, got %d: %#v", len(records), records)
	}
	if csvCell(records, "aweme_id") != "aweme_hot_e2e" || csvCell(records, "title") != "Douyin hot E2E mock aweme" {
		t.Fatalf("unexpected hot_search csv identity fields: %#v", records)
	}
	if csvCell(records, "create_time") != "1710000400" || csvCell(records, "liked_count") != "288" || csvCell(records, "comment_count") != "25" || csvCell(records, "share_count") != "23" || csvCell(records, "collected_count") != "220" {
		t.Fatalf("unexpected hot_search csv metric fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "mock hot douyin" || csvCell(records, "is_ai_generated") != "0" {
		t.Fatalf("unexpected hot_search csv source/aigc fields: %#v", records)
	}
	if csvCell(records, "user_id") != "dy-hot-author" || csvCell(records, "nickname") != "mock douyin hot author" || csvCell(records, "ip_location") != "深圳" {
		t.Fatalf("unexpected hot_search csv author fields: %#v", records)
	}
}

func TestCLIDouyinHotSearchSQLiteWithMockServers(t *testing.T) {
	result := runCLIDouyinHotSearchWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open douyin hot_search sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	var row struct {
		AwemeID        string `gorm:"column:aweme_id"`
		Title          string `gorm:"column:title"`
		Desc           string `gorm:"column:desc"`
		CreateTime     int64  `gorm:"column:create_time"`
		LikedCount     string `gorm:"column:liked_count"`
		CommentCount   string `gorm:"column:comment_count"`
		ShareCount     string `gorm:"column:share_count"`
		CollectedCount string `gorm:"column:collected_count"`
		SourceKeyword  string `gorm:"column:source_keyword"`
		IsAIGenerated  int    `gorm:"column:is_ai_generated"`
		UserID         string `gorm:"column:user_id"`
		Nickname       string `gorm:"column:nickname"`
		IPLocation     string `gorm:"column:ip_location"`
		AddTs          int64  `gorm:"column:add_ts"`
		LastModifyTs   int64  `gorm:"column:last_modify_ts"`
	}
	if err := db.Table("douyin_aweme").Where("aweme_id = ?", "aweme_hot_e2e").First(&row).Error; err != nil {
		t.Fatalf("query douyin hot_search sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	if row.Title != "Douyin hot E2E mock aweme" || row.Desc != "mock douyin hot desc" {
		t.Fatalf("unexpected hot_search sqlite identity fields: %#v", row)
	}
	if row.CreateTime != 1710000400 || row.LikedCount != "288" || row.CommentCount != "25" || row.ShareCount != "23" || row.CollectedCount != "220" {
		t.Fatalf("unexpected hot_search sqlite metric fields: %#v", row)
	}
	if row.SourceKeyword != "mock hot douyin" || row.IsAIGenerated != 0 {
		t.Fatalf("unexpected hot_search sqlite source/aigc fields: %#v", row)
	}
	if row.UserID != "dy-hot-author" || row.Nickname != "mock douyin hot author" || row.IPLocation != "深圳" {
		t.Fatalf("unexpected hot_search sqlite author fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected hot_search sqlite timestamps: %#v", row)
	}
}

func TestCLIDouyinHotSearchCheckpointResumeUsesSnapshotAndSkipsCompletedHotWord(t *testing.T) {
	repoRoot := e2e.RepoRoot(t)

	hotListCalls := 0
	hotVideoCallsByWord := make(map[string]int)
	detailCallsByAwemeID := make(map[string]int)
	allowSecondHotWord := false
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
							"client_id": "dy-hot-resume-client",
							"platforms": map[string]any{
								"dy": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"dy": "mock-douyin"},
						},
					},
				},
			})
		case "/api/cookies/dy":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "sessionid=hot-resume-session; msToken=hot-resume-token"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/douyin/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"a_bogus": "hot-resume-a-bogus"},
			})
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/hot/search/list/":
			hotListCalls++
			writeJSON(t, w, douyinHotSearchResumeListPayload())
		case "/aweme/v1/web/hot/search/video/list/":
			hotword := r.URL.Query().Get("hotword")
			hotVideoCallsByWord[hotword]++
			switch hotword {
			case "mock hot douyin one":
				writeJSON(t, w, map[string]any{
					"status_code": 0,
					"aweme_list": []map[string]any{
						{"aweme_id": "aweme_hot_resume_1"},
					},
				})
			case "mock hot douyin two":
				if !allowSecondHotWord {
					http.Error(w, "transient second hot word failure", http.StatusInternalServerError)
					return
				}
				writeJSON(t, w, map[string]any{
					"status_code": 0,
					"aweme_list": []map[string]any{
						{"aweme_id": "aweme_hot_resume_2"},
					},
				})
			default:
				t.Fatalf("unexpected hotword: %s", hotword)
			}
		case "/aweme/v1/web/aweme/detail/":
			awemeID := r.URL.Query().Get("aweme_id")
			detailCallsByAwemeID[awemeID]++
			switch awemeID {
			case "aweme_hot_resume_1":
				writeJSON(t, w, map[string]any{
					"status_code":  0,
					"aweme_detail": douyinMockHotResumeAweme(awemeID, "Douyin hot resume aweme 1", "dy-hot-resume-author-1"),
				})
			case "aweme_hot_resume_2":
				writeJSON(t, w, map[string]any{
					"status_code":  0,
					"aweme_detail": douyinMockHotResumeAweme(awemeID, "Douyin hot resume aweme 2", "dy-hot-resume-author-2"),
				})
			default:
				t.Fatalf("unexpected hot resume detail aweme_id: %s", awemeID)
			}
		default:
			t.Fatalf("unexpected douyin hot resume request: %s %s", r.Method, r.URL.String())
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
platform: "dy"
keywords:
  - "mock-keyword"
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
perNoteMaxComments: 2
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  douyinApiUrl: %q
  douyinIndexUrl: %q
  douyinHotSearchUrl: %q
`, mockServer.URL, dataDir, host, port, mockServer.URL, mockServer.URL, mockServer.URL+"/aweme/v1/web/hot/search/list/")
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
		runCmd := exec.CommandContext(ctx, binaryPath,
			"--platform", "dy",
			"--type", "hot_search",
			"--save_data_option", "json",
			"--hot_search_max_items", "2",
			"--hot_search_per_item_count", "1",
			"--no-enable_comments",
		)
		runCmd.Dir = tmpDir
		runCmd.Env = append(os.Environ(),
			"RELATION_DB_HOST=127.0.0.1",
			"RELATION_DB_PORT=1",
			"MCPRO_DATA_DIR="+dataDir,
			"MCPRO_DOUYIN_API_URL="+mockServer.URL,
			"MCPRO_DOUYIN_INDEX_URL="+mockServer.URL,
			"MCPRO_DOUYIN_HOT_SEARCH_URL="+mockServer.URL+"/aweme/v1/web/hot/search/list/",
		)
		output, err := runCmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("douyin hot resume crawler command timed out\n%s", output)
		}
		if err != nil {
			t.Fatalf("douyin hot resume crawler command failed: %v\n%s", err, output)
		}
		return output
	}

	firstOutput := runCrawler()
	if hotListCalls != 1 || hotVideoCallsByWord["mock hot douyin one"] != 1 || hotVideoCallsByWord["mock hot douyin two"] != 1 {
		t.Fatalf("first douyin hot_search run should fetch rank and fail second hot word once; hot_list=%d hot_video=%#v\n%s", hotListCalls, hotVideoCallsByWord, firstOutput)
	}
	if detailCallsByAwemeID["aweme_hot_resume_1"] != 1 || detailCallsByAwemeID["aweme_hot_resume_2"] != 0 {
		t.Fatalf("first douyin hot_search run should only detail the completed first hot word, detail=%#v\n%s", detailCallsByAwemeID, firstOutput)
	}

	allowSecondHotWord = true
	secondOutput := runCrawler()
	if hotListCalls != 1 {
		t.Fatalf("resume should reuse douyin hot_search snapshot and not refetch hot list, got %d\n%s", hotListCalls, secondOutput)
	}
	if hotVideoCallsByWord["mock hot douyin one"] != 1 {
		t.Fatalf("resume should skip completed first hot word, hot_video=%#v\n%s", hotVideoCallsByWord, secondOutput)
	}
	if hotVideoCallsByWord["mock hot douyin two"] != 2 {
		t.Fatalf("resume should retry second hot word exactly once, hot_video=%#v\n%s", hotVideoCallsByWord, secondOutput)
	}
	if detailCallsByAwemeID["aweme_hot_resume_2"] != 1 {
		t.Fatalf("resume should detail the second hot word after retry, detail=%#v\n%s", detailCallsByAwemeID, secondOutput)
	}

	resultPath := filepath.Join(dataDir, "douyin", "json", "hot_search_contents_"+time.Now().Format("2006-01-02")+".json")
	rows := readJSONRows(t, resultPath, secondOutput)
	if len(rows) != 2 {
		t.Fatalf("expected exactly two unique Douyin hot_search rows after resume, got %d: %#v", len(rows), rows)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[fmt.Sprint(row["aweme_id"])] = true
	}
	if !seen["aweme_hot_resume_1"] || !seen["aweme_hot_resume_2"] {
		t.Fatalf("douyin hot_search resume output missing expected aweme IDs: seen=%#v rows=%#v", seen, rows)
	}
}

func TestCLIDouyinCreatorJSONWithMockServers(t *testing.T) {
	result := runCLIDouyinCreatorWithMockServers(t, "json")

	creatorPath := filepath.Join(result.dataDir, "douyin", "json", "creator_creator_"+time.Now().Format("2006-01-02")+".json")
	creatorBody, err := os.ReadFile(creatorPath)
	if err != nil {
		t.Fatalf("read douyin creator json result: %v\ncommand output:\n%s", err, result.output)
	}
	var creators []map[string]any
	if err := json.Unmarshal(creatorBody, &creators); err != nil {
		t.Fatalf("unmarshal douyin creator json: %v\n%s", err, creatorBody)
	}
	if len(creators) != 1 {
		t.Fatalf("expected one douyin creator row, got %d: %s", len(creators), creatorBody)
	}
	creator := creators[0]
	if creator["user_id"] != "dy-creator-user" || creator["nickname"] != "mock douyin creator" {
		t.Fatalf("unexpected creator identity fields: %#v", creator)
	}
	if creator["avatar"] != "https://example.test/douyin-creator-avatar-large.png" || creator["ip_location"] != "成都" || creator["desc"] != "creator signature" {
		t.Fatalf("unexpected creator profile fields: %#v", creator)
	}
	if creator["gender"] != "男" || creator["follows"] != "12" || creator["fans"] != "3456" || creator["interaction"] != "7890" || creator["videos_count"] != "1" {
		t.Fatalf("unexpected creator stats fields: %#v", creator)
	}
	if _, ok := creator["add_ts"]; ok {
		t.Fatalf("douyin creator json should match Python and omit add_ts: %#v", creator["add_ts"])
	}
	if _, ok := creator["last_modify_ts"]; !ok {
		t.Fatalf("expected creator last_modify_ts: %#v", creator)
	}

	contentPath := filepath.Join(result.dataDir, "douyin", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	contentBody, err := os.ReadFile(contentPath)
	if err != nil {
		t.Fatalf("read douyin creator content json result: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(contentBody, &rows); err != nil {
		t.Fatalf("unmarshal douyin creator content json: %v\n%s", err, contentBody)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one douyin creator content row, got %d: %s", len(rows), contentBody)
	}
	row := rows[0]
	if row["aweme_id"] != "aweme_creator_e2e" || row["title"] != "Douyin creator E2E mock aweme" || row["desc"] != "mock douyin creator desc" {
		t.Fatalf("unexpected creator aweme identity fields: %#v", row)
	}
	if row["create_time"] != "1710000500" || row["liked_count"] != "388" || row["comment_count"] != "35" || row["share_count"] != "33" || row["collected_count"] != "320" {
		t.Fatalf("unexpected creator aweme count fields: %#v", row)
	}
	if row["cover_url"] != "https://example.test/douyin-creator-cover.png" || row["video_download_url"] != "https://example.test/douyin-creator-play.mp4" {
		t.Fatalf("unexpected creator aweme media fields: %#v", row)
	}
	if row["source_keyword"] != "" || row["is_ai_generated"] != float64(0) {
		t.Fatalf("unexpected creator aweme source/aigc fields: %#v", row)
	}
	if row["user_id"] != "dy-creator-author" || row["sec_uid"] != "sec-dy-creator" || row["short_user_id"] != "445566" {
		t.Fatalf("unexpected creator aweme author id fields: %#v", row)
	}
	if row["user_unique_id"] != "mock_douyin_creator_author" || row["nickname"] != "mock douyin creator author" || row["avatar"] != "https://example.test/douyin-creator-avatar.png" {
		t.Fatalf("unexpected creator aweme author profile fields: %#v", row)
	}
	if row["user_signature"] != "creator author signature" || row["ip_location"] != "成都" {
		t.Fatalf("unexpected creator aweme author/ip fields: %#v", row)
	}
}

func TestCLIDouyinCreatorCSVWithMockServers(t *testing.T) {
	result := runCLIDouyinCreatorWithMockServers(t, "csv")

	creatorPath := filepath.Join(result.dataDir, "douyin", "1_creator_creator_"+time.Now().Format("2006-01-02")+".csv")
	creatorFile, err := os.Open(creatorPath)
	if err != nil {
		t.Fatalf("open douyin creator csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer creatorFile.Close()
	creatorRecords, err := csv.NewReader(creatorFile).ReadAll()
	if err != nil {
		t.Fatalf("read douyin creator csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(creatorRecords) != 2 {
		t.Fatalf("expected creator csv header and one row, got %d: %#v", len(creatorRecords), creatorRecords)
	}
	if csvCell(creatorRecords, "user_id") != "dy-creator-user" || csvCell(creatorRecords, "nickname") != "mock douyin creator" {
		t.Fatalf("unexpected creator csv identity fields: %#v", creatorRecords)
	}
	if csvCell(creatorRecords, "avatar") != "https://example.test/douyin-creator-avatar-large.png" || csvCell(creatorRecords, "ip_location") != "成都" || csvCell(creatorRecords, "desc") != "creator signature" {
		t.Fatalf("unexpected creator csv profile fields: %#v", creatorRecords)
	}
	if csvCell(creatorRecords, "gender") != "男" || csvCell(creatorRecords, "follows") != "12" || csvCell(creatorRecords, "fans") != "3456" || csvCell(creatorRecords, "interaction") != "7890" || csvCell(creatorRecords, "videos_count") != "1" {
		t.Fatalf("unexpected creator csv stats fields: %#v", creatorRecords)
	}

	contentPath := filepath.Join(result.dataDir, "douyin", "1_creator_contents_"+time.Now().Format("2006-01-02")+".csv")
	contentFile, err := os.Open(contentPath)
	if err != nil {
		t.Fatalf("open douyin creator content csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer contentFile.Close()
	contentRecords, err := csv.NewReader(contentFile).ReadAll()
	if err != nil {
		t.Fatalf("read douyin creator content csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(contentRecords) != 2 {
		t.Fatalf("expected creator content csv header and one row, got %d: %#v", len(contentRecords), contentRecords)
	}
	if csvCell(contentRecords, "aweme_id") != "aweme_creator_e2e" || csvCell(contentRecords, "title") != "Douyin creator E2E mock aweme" || csvCell(contentRecords, "desc") != "mock douyin creator desc" {
		t.Fatalf("unexpected creator content csv identity fields: %#v", contentRecords)
	}
	if csvCell(contentRecords, "create_time") != "1710000500" || csvCell(contentRecords, "liked_count") != "388" || csvCell(contentRecords, "comment_count") != "35" || csvCell(contentRecords, "share_count") != "33" || csvCell(contentRecords, "collected_count") != "320" {
		t.Fatalf("unexpected creator content csv metric fields: %#v", contentRecords)
	}
	if csvCell(contentRecords, "cover_url") != "https://example.test/douyin-creator-cover.png" || csvCell(contentRecords, "video_download_url") != "https://example.test/douyin-creator-play.mp4" {
		t.Fatalf("unexpected creator content csv media fields: %#v", contentRecords)
	}
	if csvCell(contentRecords, "source_keyword") != "" || csvCell(contentRecords, "is_ai_generated") != "0" {
		t.Fatalf("unexpected creator content csv source/aigc fields: %#v", contentRecords)
	}
	if csvCell(contentRecords, "user_id") != "dy-creator-author" || csvCell(contentRecords, "nickname") != "mock douyin creator author" || csvCell(contentRecords, "ip_location") != "成都" {
		t.Fatalf("unexpected creator content csv author fields: %#v", contentRecords)
	}
}

func TestCLIDouyinCreatorSQLiteWithMockServers(t *testing.T) {
	result := runCLIDouyinCreatorWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open douyin creator sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	var creator struct {
		UserID       string `gorm:"column:user_id"`
		Nickname     string `gorm:"column:nickname"`
		Avatar       string `gorm:"column:avatar"`
		IPLocation   string `gorm:"column:ip_location"`
		Desc         string `gorm:"column:desc"`
		Gender       string `gorm:"column:gender"`
		Follows      string `gorm:"column:follows"`
		Fans         string `gorm:"column:fans"`
		Interaction  string `gorm:"column:interaction"`
		VideosCount  string `gorm:"column:videos_count"`
		AddTs        int64  `gorm:"column:add_ts"`
		LastModifyTs int64  `gorm:"column:last_modify_ts"`
	}
	if err := db.Table("dy_creator").Where("user_id = ?", "dy-creator-user").First(&creator).Error; err != nil {
		t.Fatalf("query douyin creator sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	if creator.Nickname != "mock douyin creator" || creator.Avatar != "https://example.test/douyin-creator-avatar-large.png" || creator.IPLocation != "成都" || creator.Desc != "creator signature" {
		t.Fatalf("unexpected creator sqlite profile fields: %#v", creator)
	}
	if creator.Gender != "男" || creator.Follows != "12" || creator.Fans != "3456" || creator.Interaction != "7890" || creator.VideosCount != "1" {
		t.Fatalf("unexpected creator sqlite stats fields: %#v", creator)
	}
	if creator.AddTs == 0 || creator.LastModifyTs == 0 {
		t.Fatalf("expected creator sqlite timestamps: %#v", creator)
	}

	var row struct {
		AwemeID          string `gorm:"column:aweme_id"`
		Title            string `gorm:"column:title"`
		Desc             string `gorm:"column:desc"`
		CreateTime       int64  `gorm:"column:create_time"`
		LikedCount       string `gorm:"column:liked_count"`
		CommentCount     string `gorm:"column:comment_count"`
		ShareCount       string `gorm:"column:share_count"`
		CollectedCount   string `gorm:"column:collected_count"`
		CoverURL         string `gorm:"column:cover_url"`
		VideoDownloadURL string `gorm:"column:video_download_url"`
		SourceKeyword    string `gorm:"column:source_keyword"`
		IsAIGenerated    int    `gorm:"column:is_ai_generated"`
		UserID           string `gorm:"column:user_id"`
		Nickname         string `gorm:"column:nickname"`
		IPLocation       string `gorm:"column:ip_location"`
		AddTs            int64  `gorm:"column:add_ts"`
		LastModifyTs     int64  `gorm:"column:last_modify_ts"`
	}
	if err := db.Table("douyin_aweme").Where("aweme_id = ?", "aweme_creator_e2e").First(&row).Error; err != nil {
		t.Fatalf("query douyin creator content sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	if row.Title != "Douyin creator E2E mock aweme" || row.Desc != "mock douyin creator desc" {
		t.Fatalf("unexpected creator content sqlite identity fields: %#v", row)
	}
	if row.CreateTime != 1710000500 || row.LikedCount != "388" || row.CommentCount != "35" || row.ShareCount != "33" || row.CollectedCount != "320" {
		t.Fatalf("unexpected creator content sqlite metric fields: %#v", row)
	}
	if row.CoverURL != "https://example.test/douyin-creator-cover.png" || row.VideoDownloadURL != "https://example.test/douyin-creator-play.mp4" {
		t.Fatalf("unexpected creator content sqlite media fields: %#v", row)
	}
	if row.SourceKeyword != "" || row.IsAIGenerated != 0 {
		t.Fatalf("unexpected creator content sqlite source/aigc fields: %#v", row)
	}
	if row.UserID != "dy-creator-author" || row.Nickname != "mock douyin creator author" || row.IPLocation != "成都" {
		t.Fatalf("unexpected creator content sqlite author fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected creator content sqlite timestamps: %#v", row)
	}
}

func TestCLIDouyinHomefeedJSONWithMockServers(t *testing.T) {
	result := runCLIDouyinHomefeedWithMockServers(t, "json")

	resultPath := filepath.Join(result.dataDir, "douyin", "json", "homefeed_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read douyin homefeed json result: %v\ncommand output:\n%s", err, result.output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal douyin homefeed json: %v\n%s", err, body)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one douyin homefeed row, got %d: %s", len(rows), body)
	}
	row := rows[0]
	if row["aweme_id"] != "aweme_homefeed_e2e" || row["title"] != "Douyin homefeed E2E mock aweme" || row["desc"] != "mock douyin homefeed desc" {
		t.Fatalf("unexpected homefeed identity fields: %#v", row)
	}
	if row["create_time"] != "1710000600" || row["liked_count"] != "488" || row["comment_count"] != "45" || row["share_count"] != "43" || row["collected_count"] != "420" {
		t.Fatalf("unexpected homefeed metric fields: %#v", row)
	}
	if row["cover_url"] != "https://example.test/douyin-homefeed-cover.png" || row["video_download_url"] != "https://example.test/douyin-homefeed-play.mp4" {
		t.Fatalf("unexpected homefeed media fields: %#v", row)
	}
	if row["source_keyword"] != "" || row["is_ai_generated"] != float64(0) {
		t.Fatalf("unexpected homefeed source/aigc fields: %#v", row)
	}
	if row["user_id"] != "dy-homefeed-author" || row["sec_uid"] != "sec-dy-homefeed-author" || row["short_user_id"] != "556677" {
		t.Fatalf("unexpected homefeed author id fields: %#v", row)
	}
	if row["user_unique_id"] != "mock_douyin_homefeed_author" || row["nickname"] != "mock douyin homefeed author" || row["avatar"] != "https://example.test/douyin-homefeed-avatar.png" {
		t.Fatalf("unexpected homefeed author profile fields: %#v", row)
	}
	if row["user_signature"] != "homefeed signature" || row["ip_location"] != "重庆" {
		t.Fatalf("unexpected homefeed author/ip fields: %#v", row)
	}
	if _, ok := row["last_modify_ts"]; !ok {
		t.Fatalf("expected homefeed last_modify_ts: %#v", row)
	}
}

func TestCLIDouyinHomefeedCSVWithMockServers(t *testing.T) {
	result := runCLIDouyinHomefeedWithMockServers(t, "csv")

	resultPath := filepath.Join(result.dataDir, "douyin", "1_homefeed_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(resultPath)
	if err != nil {
		t.Fatalf("open douyin homefeed csv result: %v\ncommand output:\n%s", err, result.output)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read douyin homefeed csv result: %v\ncommand output:\n%s", err, result.output)
	}
	if len(records) != 2 {
		t.Fatalf("expected homefeed csv header and one row, got %d: %#v", len(records), records)
	}
	if csvCell(records, "aweme_id") != "aweme_homefeed_e2e" || csvCell(records, "title") != "Douyin homefeed E2E mock aweme" || csvCell(records, "desc") != "mock douyin homefeed desc" {
		t.Fatalf("unexpected homefeed csv identity fields: %#v", records)
	}
	if csvCell(records, "create_time") != "1710000600" || csvCell(records, "liked_count") != "488" || csvCell(records, "comment_count") != "45" || csvCell(records, "share_count") != "43" || csvCell(records, "collected_count") != "420" {
		t.Fatalf("unexpected homefeed csv metric fields: %#v", records)
	}
	if csvCell(records, "cover_url") != "https://example.test/douyin-homefeed-cover.png" || csvCell(records, "video_download_url") != "https://example.test/douyin-homefeed-play.mp4" {
		t.Fatalf("unexpected homefeed csv media fields: %#v", records)
	}
	if csvCell(records, "source_keyword") != "" || csvCell(records, "is_ai_generated") != "0" {
		t.Fatalf("unexpected homefeed csv source/aigc fields: %#v", records)
	}
	if csvCell(records, "user_id") != "dy-homefeed-author" || csvCell(records, "nickname") != "mock douyin homefeed author" || csvCell(records, "ip_location") != "重庆" {
		t.Fatalf("unexpected homefeed csv author fields: %#v", records)
	}
}

func TestCLIDouyinHomefeedSQLiteWithMockServers(t *testing.T) {
	result := runCLIDouyinHomefeedWithMockServers(t, "sqlite")

	db, err := gorm.Open(sqlite.Open(result.sqlitePath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open douyin homefeed sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	var row struct {
		AwemeID          string `gorm:"column:aweme_id"`
		Title            string `gorm:"column:title"`
		Desc             string `gorm:"column:desc"`
		CreateTime       int64  `gorm:"column:create_time"`
		LikedCount       string `gorm:"column:liked_count"`
		CommentCount     string `gorm:"column:comment_count"`
		ShareCount       string `gorm:"column:share_count"`
		CollectedCount   string `gorm:"column:collected_count"`
		CoverURL         string `gorm:"column:cover_url"`
		VideoDownloadURL string `gorm:"column:video_download_url"`
		SourceKeyword    string `gorm:"column:source_keyword"`
		IsAIGenerated    int    `gorm:"column:is_ai_generated"`
		UserID           string `gorm:"column:user_id"`
		SecUID           string `gorm:"column:sec_uid"`
		ShortUserID      string `gorm:"column:short_user_id"`
		UserUniqueID     string `gorm:"column:user_unique_id"`
		Nickname         string `gorm:"column:nickname"`
		Avatar           string `gorm:"column:avatar"`
		UserSignature    string `gorm:"column:user_signature"`
		IPLocation       string `gorm:"column:ip_location"`
		AddTs            int64  `gorm:"column:add_ts"`
		LastModifyTs     int64  `gorm:"column:last_modify_ts"`
	}
	if err := db.Table("douyin_aweme").Where("aweme_id = ?", "aweme_homefeed_e2e").First(&row).Error; err != nil {
		t.Fatalf("query douyin homefeed sqlite result: %v\ncommand output:\n%s", err, result.output)
	}
	if row.Title != "Douyin homefeed E2E mock aweme" || row.Desc != "mock douyin homefeed desc" {
		t.Fatalf("unexpected homefeed sqlite identity fields: %#v", row)
	}
	if row.CreateTime != 1710000600 || row.LikedCount != "488" || row.CommentCount != "45" || row.ShareCount != "43" || row.CollectedCount != "420" {
		t.Fatalf("unexpected homefeed sqlite metric fields: %#v", row)
	}
	if row.CoverURL != "https://example.test/douyin-homefeed-cover.png" || row.VideoDownloadURL != "https://example.test/douyin-homefeed-play.mp4" {
		t.Fatalf("unexpected homefeed sqlite media fields: %#v", row)
	}
	if row.SourceKeyword != "" || row.IsAIGenerated != 0 {
		t.Fatalf("unexpected homefeed sqlite source/aigc fields: %#v", row)
	}
	if row.UserID != "dy-homefeed-author" || row.SecUID != "sec-dy-homefeed-author" || row.ShortUserID != "556677" || row.UserUniqueID != "mock_douyin_homefeed_author" {
		t.Fatalf("unexpected homefeed sqlite author id fields: %#v", row)
	}
	if row.Nickname != "mock douyin homefeed author" || row.Avatar != "https://example.test/douyin-homefeed-avatar.png" || row.UserSignature != "homefeed signature" || row.IPLocation != "重庆" {
		t.Fatalf("unexpected homefeed sqlite author profile fields: %#v", row)
	}
	if row.AddTs == 0 || row.LastModifyTs == 0 {
		t.Fatalf("expected homefeed sqlite timestamps: %#v", row)
	}
}

type cliDouyinSearchResult struct {
	dataDir    string
	sqlitePath string
	output     []byte
}

func runCLIDouyinSearchWithMockServers(t *testing.T, saveDataOption string, commentsEnabled ...bool) cliDouyinSearchResult {
	t.Helper()
	repoRoot := e2e.RepoRoot(t)

	enableComments := len(commentsEnabled) > 0 && commentsEnabled[0]
	var searchCalls int
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
							"client_id": "dy-mock-client",
							"platforms": map[string]any{
								"dy": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"dy": "mock-douyin"},
						},
					},
				},
			})
		case "/api/cookies/dy":
			if got := r.URL.Query().Get("client_id"); got != "dy-mock-client" {
				t.Fatalf("unexpected client_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "sessionid=mock-session; msToken=mock-token"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/douyin/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data": map[string]string{
					"a_bogus": "mock-a-bogus",
				},
			})
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/general/search/single/":
			searchCalls++
			if r.URL.Query().Get("offset") != "0" {
				writeJSON(t, w, map[string]any{
					"status_code": 0,
					"data":        []map[string]any{},
					"extra":       map[string]string{"logid": "mock-logid-2"},
				})
				return
			}
			writeJSON(t, w, map[string]any{
				"status_code": 0,
				"data": []map[string]any{
					{"aweme_info": douyinMockAweme()},
				},
				"extra": map[string]string{"logid": "mock-logid"},
			})
		case "/aweme/v1/web/comment/list/":
			rootCommentCalls++
			writeJSON(t, w, map[string]any{
				"status_code": 0,
				"has_more":    0,
				"cursor":      20,
				"comments": []map[string]any{
					douyinMockRootComment(),
				},
			})
		case "/aweme/v1/web/comment/list/reply/":
			subCommentCalls++
			writeJSON(t, w, map[string]any{
				"status_code": 0,
				"has_more":    0,
				"cursor":      20,
				"comments": []map[string]any{
					douyinMockSubComment(),
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
platform: "dy"
keywords:
  - "mock-keyword"
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
  douyinApiUrl: %q
  douyinIndexUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, enableComments, enableComments, host, port, mockServer.URL, mockServer.URL)
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
		"MCPRO_DOUYIN_API_URL="+mockServer.URL,
		"MCPRO_DOUYIN_INDEX_URL="+mockServer.URL,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if searchCalls != 2 {
		t.Fatalf("expected two douyin search requests like Python page-then-empty flow, got %d\n%s", searchCalls, output)
	}
	if enableComments && rootCommentCalls != 1 {
		t.Fatalf("expected one douyin root comment request, got %d\n%s", rootCommentCalls, output)
	}
	if enableComments && subCommentCalls != 1 {
		t.Fatalf("expected one douyin sub comment request, got %d\n%s", subCommentCalls, output)
	}
	return cliDouyinSearchResult{
		dataDir:    dataDir,
		sqlitePath: sqlitePath,
		output:     output,
	}
}

func runCLIDouyinDetailWithMockServers(t *testing.T, saveDataOption string) cliDouyinSearchResult {
	t.Helper()
	repoRoot := e2e.RepoRoot(t)

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
							"client_id": "dy-detail-client",
							"platforms": map[string]any{
								"dy": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"dy": "mock-douyin"},
						},
					},
				},
			})
		case "/api/cookies/dy":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "sessionid=detail-session; msToken=detail-token"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/douyin/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"a_bogus": "detail-a-bogus"},
			})
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/aweme/detail/":
			detailCalls++
			if got := r.URL.Query().Get("aweme_id"); got != "aweme_detail_e2e" {
				t.Fatalf("unexpected detail aweme_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"status_code":  0,
				"aweme_detail": douyinMockDetailAweme(),
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
platform: "dy"
keywords:
  - "mock-keyword"
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
  douyinApiUrl: %q
  douyinIndexUrl: %q
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
	runCmd := exec.CommandContext(ctx, binaryPath,
		"--platform", "dy",
		"--type", "detail",
		"--urls", "aweme_detail_e2e",
		"--save_data_option", saveDataOption,
		"--no-enable_comments",
		"--no-enable_checkpoint",
	)
	runCmd.Dir = tmpDir
	runCmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+dataDir,
		"MCPRO_DOUYIN_API_URL="+mockServer.URL,
		"MCPRO_DOUYIN_INDEX_URL="+mockServer.URL,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if detailCalls != 1 {
		t.Fatalf("expected one douyin detail request, got %d\n%s", detailCalls, output)
	}
	return cliDouyinSearchResult{
		dataDir:    dataDir,
		sqlitePath: sqlitePath,
		output:     output,
	}
}

func runCLIDouyinHotSearchWithMockServers(t *testing.T, saveDataOption string) cliDouyinSearchResult {
	t.Helper()
	repoRoot := e2e.RepoRoot(t)

	var hotListCalls int
	var hotVideoListCalls int
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
							"client_id": "dy-hot-client",
							"platforms": map[string]any{
								"dy": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"dy": "mock-douyin"},
						},
					},
				},
			})
		case "/api/cookies/dy":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "sessionid=hot-session; msToken=hot-token"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/douyin/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"a_bogus": "hot-a-bogus"},
			})
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/hot/search/list/":
			hotListCalls++
			writeJSON(t, w, douyinHotSearchListPayload())
		case "/aweme/v1/web/hot/search/video/list/":
			hotVideoListCalls++
			if got := r.URL.Query().Get("hotword"); got != "mock hot douyin" {
				t.Fatalf("unexpected hotword: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"status_code": 0,
				"aweme_list": []map[string]any{
					{"aweme_id": "aweme_hot_e2e"},
				},
			})
		case "/aweme/v1/web/aweme/detail/":
			detailCalls++
			if got := r.URL.Query().Get("aweme_id"); got != "aweme_hot_e2e" {
				t.Fatalf("unexpected hot detail aweme_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"status_code":  0,
				"aweme_detail": douyinMockHotAweme(),
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
platform: "dy"
keywords:
  - "mock-keyword"
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
perNoteMaxComments: 2
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  douyinApiUrl: %q
  douyinIndexUrl: %q
  douyinHotSearchUrl: %q
`, saveDataOption, mockServer.URL, dataDir, sqlitePath, host, port, mockServer.URL, mockServer.URL, mockServer.URL+"/aweme/v1/web/hot/search/list/")
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
	runCmd := exec.CommandContext(ctx, binaryPath,
		"--platform", "dy",
		"--type", "hot_search",
		"--save_data_option", saveDataOption,
		"--hot_search_max_items", "1",
		"--hot_search_per_item_count", "1",
		"--no-enable_comments",
		"--no-enable_checkpoint",
	)
	runCmd.Dir = tmpDir
	runCmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+dataDir,
		"MCPRO_DOUYIN_API_URL="+mockServer.URL,
		"MCPRO_DOUYIN_INDEX_URL="+mockServer.URL,
		"MCPRO_DOUYIN_HOT_SEARCH_URL="+mockServer.URL+"/aweme/v1/web/hot/search/list/",
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if hotListCalls != 1 || hotVideoListCalls != 1 || detailCalls != 1 {
		t.Fatalf("unexpected hot_search calls: hot_list=%d hot_video=%d detail=%d\n%s", hotListCalls, hotVideoListCalls, detailCalls, output)
	}
	return cliDouyinSearchResult{
		dataDir:    dataDir,
		sqlitePath: sqlitePath,
		output:     output,
	}
}

func runCLIDouyinCreatorWithMockServers(t *testing.T, saveDataOption string) cliDouyinSearchResult {
	t.Helper()
	repoRoot := e2e.RepoRoot(t)

	var profileCalls int
	var postCalls int
	var commentCalls int
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
							"client_id": "dy-creator-client",
							"platforms": map[string]any{
								"dy": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"dy": "mock-douyin"},
						},
					},
				},
			})
		case "/api/cookies/dy":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "sessionid=creator-session; msToken=creator-token"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/douyin/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"a_bogus": "creator-a-bogus"},
			})
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/user/profile/other/":
			profileCalls++
			if got := r.URL.Query().Get("sec_user_id"); got != "sec-dy-creator" {
				t.Fatalf("unexpected creator sec_user_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"status_code": 0,
				"user":        douyinMockCreator(),
			})
		case "/aweme/v1/web/aweme/post/":
			postCalls++
			if got := r.URL.Query().Get("sec_user_id"); got != "sec-dy-creator" {
				t.Fatalf("unexpected creator post sec_user_id: %s", got)
			}
			if got := r.URL.Query().Get("max_cursor"); got != "0" {
				t.Fatalf("unexpected creator post max_cursor: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"status_code": 0,
				"has_more":    0,
				"max_cursor":  "18",
				"aweme_list": []map[string]any{
					douyinMockCreatorAweme(),
				},
			})
		case "/aweme/v1/web/comment/list/":
			commentCalls++
			writeJSON(t, w, map[string]any{"status_code": 0, "comments": []map[string]any{}})
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
platform: "dy"
keywords:
  - "mock-keyword"
crawlerType: "creator"
specifiedCreatorIds:
  - "sec-dy-creator"
specifiedCreatorURLs:
  - "sec-dy-creator"
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
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  douyinApiUrl: %q
  douyinIndexUrl: %q
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
	runCmd := exec.CommandContext(ctx, binaryPath,
		"--platform", "dy",
		"--type", "creator",
		"--urls", "sec-dy-creator",
		"--save_data_option", saveDataOption,
		"--no-enable_comments",
		"--no-enable_sub_comments",
		"--no-enable_checkpoint",
	)
	runCmd.Dir = tmpDir
	runCmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+dataDir,
		"MCPRO_DOUYIN_API_URL="+mockServer.URL,
		"MCPRO_DOUYIN_INDEX_URL="+mockServer.URL,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if profileCalls != 1 || postCalls != 1 || commentCalls != 0 {
		t.Fatalf("unexpected creator calls: profile=%d post=%d comments=%d\n%s", profileCalls, postCalls, commentCalls, output)
	}
	return cliDouyinSearchResult{
		dataDir:    dataDir,
		sqlitePath: sqlitePath,
		output:     output,
	}
}

func runCLIDouyinHomefeedWithMockServers(t *testing.T, saveDataOption string) cliDouyinSearchResult {
	t.Helper()
	repoRoot := e2e.RepoRoot(t)

	var feedCalls int
	var commentCalls int
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
							"client_id": "dy-homefeed-client",
							"platforms": map[string]any{
								"dy": map[string]any{"has_cookies": true},
							},
							"nicknames": map[string]string{"dy": "mock-douyin"},
						},
					},
				},
			})
		case "/api/cookies/dy":
			writeJSON(t, w, map[string]any{
				"isok": true,
				"msg":  "ok",
				"data": map[string]string{"cookies": "sessionid=homefeed-session; msToken=homefeed-token"},
			})
		case "/signsrv/pong":
			_, _ = w.Write([]byte(`{"isok":true}`))
		case "/signsrv/v1/douyin/sign":
			writeJSON(t, w, map[string]any{
				"biz_code": 0,
				"msg":      "ok",
				"isok":     true,
				"data":     map[string]string{"a_bogus": "homefeed-a-bogus"},
			})
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/module/feed/":
			feedCalls++
			switch r.URL.Query().Get("refresh_index") {
			case "0":
				body, err := json.Marshal(douyinMockHomefeedAweme())
				if err != nil {
					t.Fatal(err)
				}
				writeJSON(t, w, map[string]any{
					"StatusCode": 0,
					"cards": []map[string]any{
						{
							"type":  1,
							"aweme": string(body),
						},
					},
				})
			case "20":
				writeJSON(t, w, map[string]any{
					"StatusCode": 1,
					"cards":      []map[string]any{},
				})
			default:
				t.Fatalf("unexpected homefeed refresh_index: %s", r.URL.Query().Get("refresh_index"))
			}
		case "/aweme/v1/web/comment/list/":
			commentCalls++
			writeJSON(t, w, map[string]any{"status_code": 0, "comments": []map[string]any{}})
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
platform: "dy"
keywords:
  - "mock-keyword"
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
perNoteMaxComments: 2
crawlerTimeSleep: 0
debugRequest: false
proxyConfig:
  enableIpProxy: false
signSrvConfig:
  signSrvHost: %q
  signSrvPort: %d
platformEndpoints:
  douyinApiUrl: %q
  douyinIndexUrl: %q
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
	runCmd := exec.CommandContext(ctx, binaryPath,
		"--platform", "dy",
		"--type", "homefeed",
		"--save_data_option", saveDataOption,
		"--no-enable_comments",
		"--no-enable_sub_comments",
		"--no-enable_checkpoint",
	)
	runCmd.Dir = tmpDir
	runCmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+dataDir,
		"MCPRO_DOUYIN_API_URL="+mockServer.URL,
		"MCPRO_DOUYIN_INDEX_URL="+mockServer.URL,
	)
	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	if feedCalls != 2 || commentCalls != 0 {
		t.Fatalf("unexpected homefeed calls: feed=%d comments=%d\n%s", feedCalls, commentCalls, output)
	}
	return cliDouyinSearchResult{
		dataDir:    dataDir,
		sqlitePath: sqlitePath,
		output:     output,
	}
}

func douyinMockRootComment() map[string]any {
	return map[string]any{
		"cid":                 "dy_comment_root",
		"text":                "root douyin comment",
		"create_time":         int64(1710000100),
		"reply_comment_total": 1,
		"reply_id":            "",
		"reply_to_reply_id":   "",
		"digg_count":          7,
		"ip_label":            "北京",
		"image_list": []map[string]any{
			{
				"origin_url": map[string]any{
					"url_list": []string{
						"https://example.test/dy-comment-small.png",
						"https://example.test/dy-comment.png",
					},
				},
			},
		},
		"user": map[string]any{
			"uid":       "dy-commenter",
			"sec_uid":   "sec-dy-commenter",
			"short_id":  "654321",
			"unique_id": "mock_dy_commenter",
			"nickname":  "mock dy commenter",
			"signature": "comment signature",
			"avatar_thumb": map[string]any{
				"url_list": []string{"https://example.test/dy-comment-avatar.png"},
			},
		},
	}
}

func douyinMockSubComment() map[string]any {
	return map[string]any{
		"cid":                 "dy_comment_sub",
		"text":                "sub douyin comment",
		"create_time":         int64(1710000200),
		"reply_comment_total": 0,
		"reply_id":            "dy_comment_root",
		"reply_to_reply_id":   "dy_comment_root",
		"digg_count":          2,
		"ip_label":            "广州",
		"image_list":          []map[string]any{},
		"user": map[string]any{
			"uid":       "dy-sub-commenter",
			"sec_uid":   "sec-dy-sub-commenter",
			"short_id":  "654322",
			"unique_id": "mock_dy_sub_commenter",
			"nickname":  "mock dy sub commenter",
			"signature": "sub comment signature",
			"avatar_thumb": map[string]any{
				"url_list": []string{"https://example.test/dy-sub-comment-avatar.png"},
			},
		},
	}
}

func csvCellAt(records [][]string, row int, column string) string {
	for idx, header := range records[0] {
		if strings.TrimPrefix(header, "\ufeff") == column {
			return records[row][idx]
		}
	}
	return ""
}

func douyinMockAweme() map[string]any {
	return map[string]any{
		"aweme_id":      "aweme_e2e",
		"aweme_type":    0,
		"preview_title": "Douyin E2E mock aweme",
		"desc":          "mock douyin desc",
		"create_time":   int64(1710000000),
		"ip_label":      "上海",
		"statistics": map[string]any{
			"digg_count":    88,
			"comment_count": 5,
			"share_count":   3,
			"collect_count": 20,
		},
		"aigc_info": map[string]any{"aigc_label_type": 1},
		"video": map[string]any{
			"raw_cover": map[string]any{
				"url_list": []string{
					"https://example.test/douyin-cover-small.png",
					"https://example.test/douyin-cover.png",
				},
			},
			"play_addr_h264": map[string]any{
				"url_list": []string{
					"https://example.test/douyin-play-low.mp4",
					"https://example.test/douyin-play.mp4",
				},
			},
		},
		"author": map[string]any{
			"uid":       "dy-author",
			"sec_uid":   "sec-dy-author",
			"short_id":  "123456",
			"unique_id": "mock_douyin_author",
			"nickname":  "mock douyin author",
			"signature": "mock signature",
			"avatar_thumb": map[string]any{
				"url_list": []string{"https://example.test/douyin-avatar.png"},
			},
		},
	}
}

func douyinMockDetailAweme() map[string]any {
	aweme := douyinMockAweme()
	aweme["aweme_id"] = "aweme_detail_e2e"
	aweme["preview_title"] = "Douyin detail E2E mock aweme"
	aweme["desc"] = "mock douyin detail desc"
	aweme["create_time"] = int64(1710000300)
	aweme["ip_label"] = "杭州"
	aweme["statistics"] = map[string]any{
		"digg_count":    188,
		"comment_count": 15,
		"share_count":   13,
		"collect_count": 120,
	}
	aweme["aigc_info"] = map[string]any{"aigc_label_type": 0}
	aweme["video"] = map[string]any{
		"raw_cover": map[string]any{
			"url_list": []string{
				"https://example.test/douyin-detail-cover-small.png",
				"https://example.test/douyin-detail-cover.png",
			},
		},
		"play_addr_h264": map[string]any{
			"url_list": []string{
				"https://example.test/douyin-detail-play-low.mp4",
				"https://example.test/douyin-detail-play.mp4",
			},
		},
	}
	aweme["author"] = map[string]any{
		"uid":       "dy-detail-author",
		"sec_uid":   "sec-dy-detail-author",
		"short_id":  "223344",
		"unique_id": "mock_douyin_detail_author",
		"nickname":  "mock douyin detail author",
		"signature": "detail signature",
		"avatar_thumb": map[string]any{
			"url_list": []string{"https://example.test/douyin-detail-avatar.png"},
		},
	}
	return aweme
}

func douyinHotSearchListPayload() map[string]any {
	return map[string]any{
		"status_code": 0,
		"data": map[string]any{
			"word_list": []map[string]any{
				{
					"word":        "mock hot douyin",
					"position":    1,
					"group_id":    "hot_group_1",
					"sentence_id": "sentence_1",
					"hot_value":   "999",
					"label":       "热",
				},
			},
		},
	}
}

func douyinHotSearchResumeListPayload() map[string]any {
	return map[string]any{
		"status_code": 0,
		"data": map[string]any{
			"word_list": []map[string]any{
				{
					"word":        "mock hot douyin one",
					"position":    1,
					"group_id":    "hot_resume_group_1",
					"sentence_id": "hot_resume_sentence_1",
					"hot_value":   "1999",
					"label":       "热",
				},
				{
					"word":        "mock hot douyin two",
					"position":    2,
					"group_id":    "hot_resume_group_2",
					"sentence_id": "hot_resume_sentence_2",
					"hot_value":   "1888",
					"label":       "新",
				},
			},
		},
	}
}

func douyinMockHotAweme() map[string]any {
	aweme := douyinMockAweme()
	aweme["aweme_id"] = "aweme_hot_e2e"
	aweme["preview_title"] = "Douyin hot E2E mock aweme"
	aweme["desc"] = "mock douyin hot desc"
	aweme["create_time"] = int64(1710000400)
	aweme["ip_label"] = "深圳"
	aweme["statistics"] = map[string]any{
		"digg_count":    288,
		"comment_count": 25,
		"share_count":   23,
		"collect_count": 220,
	}
	aweme["aigc_info"] = map[string]any{"aigc_label_type": 0}
	aweme["video"] = map[string]any{
		"raw_cover": map[string]any{
			"url_list": []string{
				"https://example.test/douyin-hot-cover-small.png",
				"https://example.test/douyin-hot-cover.png",
			},
		},
		"play_addr_h264": map[string]any{
			"url_list": []string{
				"https://example.test/douyin-hot-play-low.mp4",
				"https://example.test/douyin-hot-play.mp4",
			},
		},
	}
	aweme["author"] = map[string]any{
		"uid":       "dy-hot-author",
		"sec_uid":   "sec-dy-hot-author",
		"short_id":  "334455",
		"unique_id": "mock_douyin_hot_author",
		"nickname":  "mock douyin hot author",
		"signature": "hot signature",
		"avatar_thumb": map[string]any{
			"url_list": []string{"https://example.test/douyin-hot-avatar.png"},
		},
	}
	return aweme
}

func douyinMockHotResumeAweme(awemeID, title, authorID string) map[string]any {
	aweme := douyinMockHotAweme()
	aweme["aweme_id"] = awemeID
	aweme["preview_title"] = title
	aweme["desc"] = "mock douyin hot resume desc"
	aweme["author"] = map[string]any{
		"uid":       authorID,
		"sec_uid":   "sec-" + authorID,
		"short_id":  "445566",
		"unique_id": "mock_" + strings.ReplaceAll(authorID, "-", "_"),
		"nickname":  "mock " + authorID,
		"signature": "hot resume signature",
		"avatar_thumb": map[string]any{
			"url_list": []string{"https://example.test/" + authorID + ".png"},
		},
	}
	return aweme
}

func douyinMockCreator() map[string]any {
	return map[string]any{
		"uid":         "dy-creator-user",
		"nickname":    "mock douyin creator",
		"signature":   "creator signature",
		"ip_location": "成都",
		"gender":      1,
		"avatar_larger": map[string]any{
			"url_list": []string{"https://example.test/douyin-creator-avatar-large.png"},
		},
		"statistics": map[string]any{
			"following_count": 12,
			"follower_count":  3456,
			"total_favorited": 7890,
			"aweme_count":     1,
		},
	}
}

func douyinMockCreatorAweme() map[string]any {
	return douyinMockCreatorAwemeWithID("aweme_creator_e2e", "Douyin creator E2E mock aweme")
}

func douyinMockCreatorAwemeWithID(awemeID, title string) map[string]any {
	aweme := douyinMockAweme()
	aweme["aweme_id"] = awemeID
	aweme["preview_title"] = title
	aweme["desc"] = "mock douyin creator desc"
	aweme["create_time"] = int64(1710000500)
	aweme["ip_label"] = "成都"
	aweme["statistics"] = map[string]any{
		"digg_count":    388,
		"comment_count": 35,
		"share_count":   33,
		"collect_count": 320,
	}
	aweme["aigc_info"] = map[string]any{"aigc_label_type": 0}
	aweme["video"] = map[string]any{
		"raw_cover": map[string]any{
			"url_list": []string{
				"https://example.test/douyin-creator-cover-small.png",
				"https://example.test/douyin-creator-cover.png",
			},
		},
		"play_addr_h264": map[string]any{
			"url_list": []string{
				"https://example.test/douyin-creator-play-low.mp4",
				"https://example.test/douyin-creator-play.mp4",
			},
		},
	}
	aweme["author"] = map[string]any{
		"uid":       "dy-creator-author",
		"sec_uid":   "sec-dy-creator",
		"short_id":  "445566",
		"unique_id": "mock_douyin_creator_author",
		"nickname":  "mock douyin creator author",
		"signature": "creator author signature",
		"avatar_thumb": map[string]any{
			"url_list": []string{"https://example.test/douyin-creator-avatar.png"},
		},
	}
	return aweme
}

func douyinMockHomefeedAweme() map[string]any {
	return douyinMockHomefeedAwemeWithID("aweme_homefeed_e2e", "Douyin homefeed E2E mock aweme")
}

func douyinMockHomefeedAwemeWithID(awemeID, title string) map[string]any {
	aweme := douyinMockAweme()
	aweme["aweme_id"] = awemeID
	aweme["preview_title"] = title
	aweme["desc"] = "mock douyin homefeed desc"
	aweme["create_time"] = int64(1710000600)
	aweme["ip_label"] = "重庆"
	aweme["statistics"] = map[string]any{
		"digg_count":    488,
		"comment_count": 45,
		"share_count":   43,
		"collect_count": 420,
	}
	aweme["aigc_info"] = map[string]any{"aigc_label_type": 0}
	aweme["video"] = map[string]any{
		"raw_cover": map[string]any{
			"url_list": []string{
				"https://example.test/douyin-homefeed-cover-small.png",
				"https://example.test/douyin-homefeed-cover.png",
			},
		},
		"play_addr_h264": map[string]any{
			"url_list": []string{
				"https://example.test/douyin-homefeed-play-low.mp4",
				"https://example.test/douyin-homefeed-play.mp4",
			},
		},
	}
	aweme["author"] = map[string]any{
		"uid":       "dy-homefeed-author",
		"sec_uid":   "sec-dy-homefeed-author",
		"short_id":  "556677",
		"unique_id": "mock_douyin_homefeed_author",
		"nickname":  "mock douyin homefeed author",
		"signature": "homefeed signature",
		"avatar_thumb": map[string]any{
			"url_list": []string{"https://example.test/douyin-homefeed-avatar.png"},
		},
	}
	return aweme
}

func douyinHomefeedPagePayload(aweme map[string]any) map[string]any {
	body, err := json.Marshal(aweme)
	if err != nil {
		panic(err)
	}
	return map[string]any{
		"StatusCode": 0,
		"cards": []map[string]any{
			{
				"type":  1,
				"aweme": string(body),
			},
		},
	}
}

func douyinResumeSearchPayload(awemeID, title, logID string) map[string]any {
	aweme := douyinMockAweme()
	aweme["aweme_id"] = awemeID
	aweme["preview_title"] = title
	aweme["desc"] = title + " desc"
	return map[string]any{
		"status_code": 0,
		"data": []map[string]any{
			{"aweme_info": aweme},
		},
		"extra": map[string]string{"logid": logID},
	}
}
