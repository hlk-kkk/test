// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/bilibili_mysql_e2e_test.go
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
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/e2e"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/cookiebridge"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestCLIBilibiliSearchDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "bili",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "bili-unified-client",
				Connected: true,
				Nicknames: map[string]string{"bili": "mock-bili"},
				Cookies:   map[string]string{"bili": "SESSDATA=unified; buvid3=unified"},
			},
		},
		PlatformHandler: newBilibiliUnifiedSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "bili",
		Keywords:             []string{"mock-keyword"},
		CrawlerType:          "search",
		SaveDataOption:       "db",
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env:     mysqlServer.Env(),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	var row struct {
		VideoID       string
		Bvid          string
		Title         string
		VideoType     string
		SourceKeyword string
		CreateTime    int64
		Duration      string
		VideoCoverURL string
	}
	err = db.QueryRow(`
		SELECT video_id, bvid, title, video_type, source_keyword, create_time, duration, video_cover_url
		FROM bilibili_video
		WHERE video_id = ?
	`, "12345").Scan(
		&row.VideoID,
		&row.Bvid,
		&row.Title,
		&row.VideoType,
		&row.SourceKeyword,
		&row.CreateTime,
		&row.Duration,
		&row.VideoCoverURL,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "12345", row.VideoID)
	require.Equal(t, "BV1mockbili", row.Bvid)
	require.Equal(t, "Bilibili E2E mock video", row.Title)
	require.Equal(t, "科技", row.VideoType)
	require.Equal(t, "mock-keyword", row.SourceKeyword)
	require.Equal(t, int64(1710000000), row.CreateTime)
	require.Equal(t, "66", row.Duration)
	require.Equal(t, "https://example.test/bili-cover.png", row.VideoCoverURL)
}

func TestCLIBilibiliSearchCommentsDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "bili",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "bili-comments-unified-client",
				Connected: true,
				Nicknames: map[string]string{"bili": "mock-bili"},
				Cookies:   map[string]string{"bili": "SESSDATA=comments-unified; buvid3=comments-unified"},
			},
		},
		PlatformHandler: newBilibiliUnifiedSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "bili",
		Keywords:             []string{"mock-keyword"},
		CrawlerType:          "search",
		SaveDataOption:       "db",
		CrawlerMaxNotesCount: 1,
		EnableGetComments:    true,
		EnableGetSubComments: true,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env:     mysqlServer.Env(),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	rows, err := db.Query(`
		SELECT comment_id, video_id, content, create_time, sub_comment_count,
		       like_count, parent_comment_id, user_id, nickname, avatar,
		       add_ts, last_modify_ts
		FROM bilibili_video_comment
		ORDER BY comment_id
	`)
	require.NoError(t, err, "command output:\n%s", output)
	defer rows.Close()

	type commentRow struct {
		CommentID       string
		VideoID         string
		Content         string
		CreateTime      int64
		SubCommentCount string
		LikeCount       string
		ParentCommentID string
		UserID          string
		Nickname        string
		Avatar          string
		AddTs           int64
		LastModifyTs    int64
	}
	var comments []commentRow
	for rows.Next() {
		var row commentRow
		require.NoError(t, rows.Scan(
			&row.CommentID,
			&row.VideoID,
			&row.Content,
			&row.CreateTime,
			&row.SubCommentCount,
			&row.LikeCount,
			&row.ParentCommentID,
			&row.UserID,
			&row.Nickname,
			&row.Avatar,
			&row.AddTs,
			&row.LastModifyTs,
		))
		comments = append(comments, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, comments, 2, "command output:\n%s", output)

	root := comments[0]
	require.Equal(t, "9001", root.CommentID)
	require.Equal(t, "12345", root.VideoID)
	require.Equal(t, "root bili comment", root.Content)
	require.Equal(t, int64(1710000100), root.CreateTime)
	require.Equal(t, "1", root.SubCommentCount)
	require.Equal(t, "4", root.LikeCount)
	require.Equal(t, "", root.ParentCommentID)
	require.Equal(t, "8801", root.UserID)
	require.Equal(t, "bili root commenter", root.Nickname)
	require.Equal(t, "https://example.test/bili-root.png", root.Avatar)
	require.NotZero(t, root.AddTs)
	require.NotZero(t, root.LastModifyTs)

	sub := comments[1]
	require.Equal(t, "9002", sub.CommentID)
	require.Equal(t, "12345", sub.VideoID)
	require.Equal(t, "sub bili comment", sub.Content)
	require.Equal(t, int64(1710000200), sub.CreateTime)
	require.Equal(t, "0", sub.SubCommentCount)
	require.Equal(t, "2", sub.LikeCount)
	require.Equal(t, "9001", sub.ParentCommentID)
	require.Equal(t, "8802", sub.UserID)
	require.Equal(t, "bili sub commenter", sub.Nickname)
	require.Equal(t, "https://example.test/bili-sub.png", sub.Avatar)
	require.NotZero(t, sub.AddTs)
	require.NotZero(t, sub.LastModifyTs)
}

func TestCLIBilibiliDetailDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "bili",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "bili-unified-detail-client",
				Connected: true,
				Nicknames: map[string]string{"bili": "mock-bili"},
				Cookies:   map[string]string{"bili": "SESSDATA=unified-detail; buvid3=unified-detail"},
			},
		},
		PlatformHandler: newBilibiliUnifiedDetailMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "bili",
		Keywords:             []string{"mock-keyword"},
		CrawlerType:          "detail",
		SaveDataOption:       "db",
		SpecifiedNoteIDs:     []string{"BV1mockbili"},
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env:     mysqlServer.Env(),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	var row struct {
		VideoID        string
		Bvid           string
		UserID         string
		Nickname       string
		Avatar         string
		VideoType      string
		Title          string
		Description    string
		CreateTime     int64
		LikedCount     string
		VideoPlayCount string
		VideoDanmaku   string
		VideoComment   string
		Duration       string
		VideoURL       string
		VideoCoverURL  string
		SourceKeyword  string
		AddTs          int64
		LastModifyTs   int64
	}
	err = db.QueryRow(`
		SELECT video_id, bvid, user_id, nickname, avatar, video_type, title, `+"`desc`"+`,
		       create_time, liked_count, video_play_count, video_danmaku,
		       video_comment, duration, video_url, video_cover_url, source_keyword,
		       add_ts, last_modify_ts
		FROM bilibili_video
		WHERE video_id = ?
	`, "12345").Scan(
		&row.VideoID,
		&row.Bvid,
		&row.UserID,
		&row.Nickname,
		&row.Avatar,
		&row.VideoType,
		&row.Title,
		&row.Description,
		&row.CreateTime,
		&row.LikedCount,
		&row.VideoPlayCount,
		&row.VideoDanmaku,
		&row.VideoComment,
		&row.Duration,
		&row.VideoURL,
		&row.VideoCoverURL,
		&row.SourceKeyword,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "12345", row.VideoID)
	require.Equal(t, "BV1mockbili", row.Bvid)
	require.Equal(t, "777", row.UserID)
	require.Equal(t, "mock bili author", row.Nickname)
	require.Equal(t, "https://example.test/bili-avatar.png", row.Avatar)
	require.Equal(t, "科技", row.VideoType)
	require.Equal(t, "Bilibili E2E mock video", row.Title)
	require.Equal(t, "mock bilibili desc", row.Description)
	require.Equal(t, int64(1710000000), row.CreateTime)
	require.Equal(t, "88", row.LikedCount)
	require.Equal(t, "1000", row.VideoPlayCount)
	require.Equal(t, "12", row.VideoDanmaku)
	require.Equal(t, "5", row.VideoComment)
	require.Equal(t, "66", row.Duration)
	require.Equal(t, "https://www.bilibili.com/video/BV1mockbili", row.VideoURL)
	require.Equal(t, "https://example.test/bili-cover.png", row.VideoCoverURL)
	require.Equal(t, "", row.SourceKeyword)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func TestCLIBilibiliHotSearchDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "bili",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "bili-hot-unified-client",
				Connected: true,
				Nicknames: map[string]string{"bili": "mock-bili"},
				Cookies:   map[string]string{"bili": "SESSDATA=hot-unified; buvid3=hot-unified"},
			},
		},
		PlatformHandler: newBilibiliUnifiedHotSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:               "bili",
		CrawlerType:            "hot_search",
		SaveDataOption:         "db",
		HotSearchMaxItemsCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env:     mysqlServer.Env(),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	var row struct {
		VideoID       string
		Bvid          string
		Title         string
		SourceKeyword string
		CreateTime    int64
		Duration      string
		VideoURL      string
		VideoCoverURL string
		AddTs         int64
		LastModifyTs  int64
	}
	err = db.QueryRow(`
		SELECT video_id, bvid, title, source_keyword, create_time, duration,
		       video_url, video_cover_url, add_ts, last_modify_ts
		FROM bilibili_video
		WHERE video_id = ?
	`, "12345").Scan(
		&row.VideoID,
		&row.Bvid,
		&row.Title,
		&row.SourceKeyword,
		&row.CreateTime,
		&row.Duration,
		&row.VideoURL,
		&row.VideoCoverURL,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "12345", row.VideoID)
	require.Equal(t, "BV1mockbili", row.Bvid)
	require.Equal(t, "Bilibili E2E mock video", row.Title)
	require.Equal(t, "bilibili_hot_search", row.SourceKeyword)
	require.Equal(t, int64(1710000000), row.CreateTime)
	require.Equal(t, "66", row.Duration)
	require.Equal(t, "https://www.bilibili.com/video/BV1mockbili", row.VideoURL)
	require.Equal(t, "https://example.test/bili-cover.png", row.VideoCoverURL)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func TestCLIBilibiliHotSearchCommentsDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "bili",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "bili-hot-comments-unified-client",
				Connected: true,
				Nicknames: map[string]string{"bili": "mock-bili"},
				Cookies:   map[string]string{"bili": "SESSDATA=hot-comments-unified; buvid3=hot-comments-unified"},
			},
		},
		PlatformHandler: newBilibiliUnifiedHotSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:               "bili",
		CrawlerType:            "hot_search",
		SaveDataOption:         "db",
		HotSearchMaxItemsCount: 1,
		EnableGetComments:      true,
		EnableGetSubComments:   true,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env:     mysqlServer.Env(),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	rows, err := db.Query(`
		SELECT comment_id, video_id, content, create_time, sub_comment_count,
		       like_count, parent_comment_id, user_id, nickname, avatar,
		       add_ts, last_modify_ts
		FROM bilibili_video_comment
		ORDER BY comment_id
	`)
	require.NoError(t, err, "command output:\n%s", output)
	defer rows.Close()

	type commentRow struct {
		CommentID       string
		VideoID         string
		Content         string
		CreateTime      int64
		SubCommentCount string
		LikeCount       string
		ParentCommentID string
		UserID          string
		Nickname        string
		Avatar          string
		AddTs           int64
		LastModifyTs    int64
	}
	var comments []commentRow
	for rows.Next() {
		var row commentRow
		require.NoError(t, rows.Scan(
			&row.CommentID,
			&row.VideoID,
			&row.Content,
			&row.CreateTime,
			&row.SubCommentCount,
			&row.LikeCount,
			&row.ParentCommentID,
			&row.UserID,
			&row.Nickname,
			&row.Avatar,
			&row.AddTs,
			&row.LastModifyTs,
		))
		comments = append(comments, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, comments, 2, "command output:\n%s", output)

	require.Equal(t, "9001", comments[0].CommentID)
	require.Equal(t, "12345", comments[0].VideoID)
	require.Equal(t, "root bili comment", comments[0].Content)
	require.Equal(t, int64(1710000100), comments[0].CreateTime)
	require.Equal(t, "1", comments[0].SubCommentCount)
	require.Equal(t, "4", comments[0].LikeCount)
	require.Equal(t, "", comments[0].ParentCommentID)
	require.Equal(t, "8801", comments[0].UserID)
	require.Equal(t, "bili root commenter", comments[0].Nickname)
	require.Equal(t, "https://example.test/bili-root.png", comments[0].Avatar)
	require.NotZero(t, comments[0].AddTs)
	require.NotZero(t, comments[0].LastModifyTs)

	require.Equal(t, "9002", comments[1].CommentID)
	require.Equal(t, "12345", comments[1].VideoID)
	require.Equal(t, "sub bili comment", comments[1].Content)
	require.Equal(t, int64(1710000200), comments[1].CreateTime)
	require.Equal(t, "0", comments[1].SubCommentCount)
	require.Equal(t, "2", comments[1].LikeCount)
	require.Equal(t, "9001", comments[1].ParentCommentID)
	require.Equal(t, "8802", comments[1].UserID)
	require.Equal(t, "bili sub commenter", comments[1].Nickname)
	require.Equal(t, "https://example.test/bili-sub.png", comments[1].Avatar)
	require.NotZero(t, comments[1].AddTs)
	require.NotZero(t, comments[1].LastModifyTs)
}

func newBilibiliUnifiedSearchMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/x/web-interface/wbi/search/type":
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
			t.Fatalf("unexpected bilibili unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newBilibiliUnifiedDetailMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/x/web-interface/view/detail":
			if got := r.URL.Query().Get("bvid"); got != "BV1mockbili" {
				t.Fatalf("unexpected bilibili detail bvid: %s", got)
			}
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
		default:
			t.Fatalf("unexpected bilibili unified detail request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newBilibiliUnifiedHotSearchMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/x/web-interface/popular":
			require.Equal(t, "1", r.URL.Query().Get("pn"))
			require.Equal(t, "1", r.URL.Query().Get("ps"))
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
					},
				},
			})
		case "/x/web-interface/view/detail":
			require.Equal(t, "BV1mockbili", r.URL.Query().Get("bvid"))
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
			require.Equal(t, "12345", r.URL.Query().Get("oid"))
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
			require.Equal(t, "12345", r.URL.Query().Get("oid"))
			require.Equal(t, "9001", r.URL.Query().Get("root"))
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
			t.Fatalf("unexpected bilibili hot_search unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}
