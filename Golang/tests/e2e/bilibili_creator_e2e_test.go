// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/bilibili_creator_e2e_test.go
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
	"path/filepath"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/e2e"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/cookiebridge"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestCLIBilibiliCreatorCSVWithUnifiedRunner(t *testing.T) {
	runner := newBilibiliCreatorRunner(t, "bili-creator-csv-client")
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "bili",
		CrawlerType:          "creator",
		SaveDataOption:       "csv",
		SpecifiedCreatorIDs:  []string{"777"},
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Timeout: 20 * time.Second,
	})

	creatorCSV := filepath.Join(runner.DataDir, "bilibili", "1_creator_creators_"+time.Now().Format("2006-01-02")+".csv")
	creatorRecords := e2e.ReadCSVRecords(t, creatorCSV, output)
	require.Len(t, creatorRecords, 2, "command output:\n%s", output)
	require.Equal(t, "777", e2e.CSVCellAt(creatorRecords, 1, "user_id"))
	require.Equal(t, "mock bili author", e2e.CSVCellAt(creatorRecords, 1, "nickname"))
	require.Equal(t, "https://example.test/bili-avatar.png", e2e.CSVCellAt(creatorRecords, 1, "avatar"))
	require.Equal(t, "creator desc from mock", e2e.CSVCellAt(creatorRecords, 1, "description"))
	require.Equal(t, "1234", e2e.CSVCellAt(creatorRecords, 1, "follower_count"))
	require.Equal(t, "56", e2e.CSVCellAt(creatorRecords, 1, "following_count"))
	require.Equal(t, "7", e2e.CSVCellAt(creatorRecords, 1, "content_count"))
	require.Equal(t, "", e2e.CSVCellAt(creatorRecords, 1, "add_ts"))
	require.NotEmpty(t, e2e.CSVCellAt(creatorRecords, 1, "last_modify_ts"))

	contentCSV := filepath.Join(runner.DataDir, "bilibili", "1_creator_contents_"+time.Now().Format("2006-01-02")+".csv")
	contentRecords := e2e.ReadCSVRecords(t, contentCSV, output)
	require.Len(t, contentRecords, 2, "command output:\n%s", output)
	require.Equal(t, "12345", e2e.CSVCellAt(contentRecords, 1, "video_id"))
	require.Equal(t, "BV1mockbili", e2e.CSVCellAt(contentRecords, 1, "bvid"))
	require.Equal(t, "Bilibili E2E mock video", e2e.CSVCellAt(contentRecords, 1, "title"))
	require.Equal(t, "mock bilibili desc", e2e.CSVCellAt(contentRecords, 1, "desc"))
	require.Equal(t, "1710000000", e2e.CSVCellAt(contentRecords, 1, "create_time"))
	require.Equal(t, "88", e2e.CSVCellAt(contentRecords, 1, "liked_count"))
	require.Equal(t, "1000", e2e.CSVCellAt(contentRecords, 1, "video_play_count"))
	require.Equal(t, "12", e2e.CSVCellAt(contentRecords, 1, "video_danmaku"))
	require.Equal(t, "5", e2e.CSVCellAt(contentRecords, 1, "video_comment"))
	require.Equal(t, "66", e2e.CSVCellAt(contentRecords, 1, "duration"))
	require.Equal(t, "https://www.bilibili.com/video/BV1mockbili", e2e.CSVCellAt(contentRecords, 1, "video_url"))
	require.Equal(t, "https://example.test/bili-cover.png", e2e.CSVCellAt(contentRecords, 1, "video_cover_url"))
	require.Equal(t, "", e2e.CSVCellAt(contentRecords, 1, "source_keyword"))
	require.Equal(t, "", e2e.CSVCellAt(contentRecords, 1, "add_ts"))
}

func TestCLIBilibiliCreatorJSONWithUnifiedRunner(t *testing.T) {
	runner := newBilibiliCreatorRunner(t, "bili-creator-json-client")
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "bili",
		CrawlerType:          "creator",
		SaveDataOption:       "json",
		SpecifiedCreatorIDs:  []string{"777"},
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Timeout: 20 * time.Second,
	})

	creatorJSON := filepath.Join(runner.DataDir, "bilibili", "json", "creator_creators_"+time.Now().Format("2006-01-02")+".json")
	creatorRows := e2e.ReadJSONRows(t, creatorJSON, output)
	require.Len(t, creatorRows, 1, "command output:\n%s", output)
	creator := creatorRows[0]
	require.Equal(t, "777", creator["user_id"])
	require.Equal(t, "mock bili author", creator["nickname"])
	require.Equal(t, "https://example.test/bili-avatar.png", creator["avatar"])
	require.Equal(t, "creator desc from mock", creator["description"])
	require.Equal(t, float64(1234), creator["follower_count"])
	require.Equal(t, float64(56), creator["following_count"])
	require.Equal(t, float64(7), creator["content_count"])
	require.Contains(t, creator, "add_ts")
	require.Nil(t, creator["add_ts"])
	require.NotEmpty(t, creator["last_modify_ts"])

	contentJSON := filepath.Join(runner.DataDir, "bilibili", "json", "creator_contents_"+time.Now().Format("2006-01-02")+".json")
	contentRows := e2e.ReadJSONRows(t, contentJSON, output)
	require.Len(t, contentRows, 1, "command output:\n%s", output)
	content := contentRows[0]
	require.Equal(t, "12345", content["video_id"])
	require.Equal(t, "BV1mockbili", content["bvid"])
	require.Equal(t, "Bilibili E2E mock video", content["title"])
	require.Equal(t, "mock bilibili desc", content["desc"])
	require.Equal(t, "1710000000", content["create_time"])
	require.Equal(t, "88", content["liked_count"])
	require.Equal(t, "1000", content["video_play_count"])
	require.Equal(t, "12", content["video_danmaku"])
	require.Equal(t, "5", content["video_comment"])
	require.Equal(t, "66", content["duration"])
	require.Equal(t, "https://www.bilibili.com/video/BV1mockbili", content["video_url"])
	require.Equal(t, "https://example.test/bili-cover.png", content["video_cover_url"])
	require.Equal(t, "", content["source_keyword"])
	require.Contains(t, content, "add_ts")
	require.Nil(t, content["add_ts"])
}

func TestCLIBilibiliCreatorSQLiteWithUnifiedRunner(t *testing.T) {
	runner := newBilibiliCreatorRunner(t, "bili-creator-sqlite-client")
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "bili",
		CrawlerType:          "creator",
		SaveDataOption:       "sqlite",
		SpecifiedCreatorIDs:  []string{"777"},
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Timeout: 20 * time.Second,
	})

	creatorRows := e2e.ReadSQLiteRows(t, filepath.Join(runner.WorkDir, "media_crawler.db"), "bilibili_up_info", "user_id", output)
	require.Len(t, creatorRows, 1, "command output:\n%s", output)
	require.Equal(t, "777", e2e.SQLiteCell(creatorRows, 0, "user_id"))
	require.Equal(t, "mock bili author", e2e.SQLiteCell(creatorRows, 0, "nickname"))
	require.Equal(t, "https://example.test/bili-avatar.png", e2e.SQLiteCell(creatorRows, 0, "avatar"))
	require.Equal(t, "creator desc from mock", e2e.SQLiteCell(creatorRows, 0, "description"))
	require.Equal(t, "1234", e2e.SQLiteCell(creatorRows, 0, "follower_count"))
	require.Equal(t, "56", e2e.SQLiteCell(creatorRows, 0, "following_count"))
	require.Equal(t, "7", e2e.SQLiteCell(creatorRows, 0, "content_count"))
	require.NotEmpty(t, e2e.SQLiteCell(creatorRows, 0, "add_ts"))
	require.NotEmpty(t, e2e.SQLiteCell(creatorRows, 0, "last_modify_ts"))

	contentRows := e2e.ReadSQLiteRows(t, filepath.Join(runner.WorkDir, "media_crawler.db"), "bilibili_video", "video_id", output)
	require.Len(t, contentRows, 1, "command output:\n%s", output)
	require.Equal(t, "12345", e2e.SQLiteCell(contentRows, 0, "video_id"))
	require.Equal(t, "BV1mockbili", e2e.SQLiteCell(contentRows, 0, "bvid"))
	require.Equal(t, "Bilibili E2E mock video", e2e.SQLiteCell(contentRows, 0, "title"))
	require.Equal(t, "mock bilibili desc", e2e.SQLiteCell(contentRows, 0, "desc"))
	require.Equal(t, "1710000000", e2e.SQLiteCell(contentRows, 0, "create_time"))
	require.Equal(t, "88", e2e.SQLiteCell(contentRows, 0, "liked_count"))
	require.Equal(t, "1000", e2e.SQLiteCell(contentRows, 0, "video_play_count"))
	require.Equal(t, "12", e2e.SQLiteCell(contentRows, 0, "video_danmaku"))
	require.Equal(t, "5", e2e.SQLiteCell(contentRows, 0, "video_comment"))
	require.Equal(t, "66", e2e.SQLiteCell(contentRows, 0, "duration"))
	require.Equal(t, "https://www.bilibili.com/video/BV1mockbili", e2e.SQLiteCell(contentRows, 0, "video_url"))
	require.Equal(t, "https://example.test/bili-cover.png", e2e.SQLiteCell(contentRows, 0, "video_cover_url"))
	require.Equal(t, "", e2e.SQLiteCell(contentRows, 0, "source_keyword"))
}

func TestCLIBilibiliCreatorDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := newBilibiliCreatorRunner(t, "bili-creator-db-client")
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "bili",
		CrawlerType:          "creator",
		SaveDataOption:       "db",
		SpecifiedCreatorIDs:  []string{"777"},
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env:     mysqlServer.Env(),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()

	var creator struct {
		UserID         string
		Nickname       string
		Avatar         string
		Description    string
		FollowerCount  string
		FollowingCount string
		ContentCount   string
		AddTs          int64
		LastModifyTs   int64
	}
	err = db.QueryRow(`
		SELECT user_id, nickname, avatar, description, follower_count,
		       following_count, content_count, add_ts, last_modify_ts
		FROM bilibili_up_info
		WHERE user_id = ?
	`, "777").Scan(
		&creator.UserID,
		&creator.Nickname,
		&creator.Avatar,
		&creator.Description,
		&creator.FollowerCount,
		&creator.FollowingCount,
		&creator.ContentCount,
		&creator.AddTs,
		&creator.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "777", creator.UserID)
	require.Equal(t, "mock bili author", creator.Nickname)
	require.Equal(t, "https://example.test/bili-avatar.png", creator.Avatar)
	require.Equal(t, "creator desc from mock", creator.Description)
	require.Equal(t, "1234", creator.FollowerCount)
	require.Equal(t, "56", creator.FollowingCount)
	require.Equal(t, "7", creator.ContentCount)
	require.NotZero(t, creator.AddTs)
	require.NotZero(t, creator.LastModifyTs)

	var content struct {
		VideoID       string
		Bvid          string
		Title         string
		Description   string
		CreateTime    int64
		LikedCount    string
		PlayCount     string
		DanmakuCount  string
		CommentCount  string
		Duration      string
		VideoURL      string
		VideoCoverURL string
		SourceKeyword string
	}
	err = db.QueryRow(`
		SELECT video_id, bvid, title, `+"`desc`"+`, create_time, liked_count,
		       video_play_count, video_danmaku, video_comment, duration,
		       video_url, video_cover_url, source_keyword
		FROM bilibili_video
		WHERE video_id = ?
	`, "12345").Scan(
		&content.VideoID,
		&content.Bvid,
		&content.Title,
		&content.Description,
		&content.CreateTime,
		&content.LikedCount,
		&content.PlayCount,
		&content.DanmakuCount,
		&content.CommentCount,
		&content.Duration,
		&content.VideoURL,
		&content.VideoCoverURL,
		&content.SourceKeyword,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "12345", content.VideoID)
	require.Equal(t, "BV1mockbili", content.Bvid)
	require.Equal(t, "Bilibili E2E mock video", content.Title)
	require.Equal(t, "mock bilibili desc", content.Description)
	require.Equal(t, int64(1710000000), content.CreateTime)
	require.Equal(t, "88", content.LikedCount)
	require.Equal(t, "1000", content.PlayCount)
	require.Equal(t, "12", content.DanmakuCount)
	require.Equal(t, "5", content.CommentCount)
	require.Equal(t, "66", content.Duration)
	require.Equal(t, "https://www.bilibili.com/video/BV1mockbili", content.VideoURL)
	require.Equal(t, "https://example.test/bili-cover.png", content.VideoCoverURL)
	require.Equal(t, "", content.SourceKeyword)
}

func newBilibiliCreatorRunner(t *testing.T, clientID string) *e2e.MockPlatformRunner {
	t.Helper()
	return e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "bili",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  clientID,
				Connected: true,
				Nicknames: map[string]string{"bili": "mock-bili"},
				Cookies:   map[string]string{"bili": "SESSDATA=creator-unified; buvid3=creator-unified"},
			},
		},
		PlatformHandler: newBilibiliUnifiedCreatorMockHandler(t),
	})
}

func newBilibiliUnifiedCreatorMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/x/web-interface/nav":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data":    map[string]any{"isLogin": true, "uname": "mock-bili"},
			})
		case "/x/space/wbi/acc/info":
			require.Equal(t, "777", r.URL.Query().Get("mid"))
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data": map[string]any{
					"mid":  int64(777),
					"name": "mock bili author",
					"face": "https://example.test/bili-avatar.png",
					"sign": "creator desc from mock",
				},
			})
		case "/x/relation/stat":
			require.Equal(t, "777", r.URL.Query().Get("vmid"))
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data":    map[string]any{"follower": int64(1234), "following": int64(56)},
			})
		case "/x/space/navnum":
			require.Equal(t, "777", r.URL.Query().Get("mid"))
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data":    map[string]any{"video": int64(7)},
			})
		case "/x/space/wbi/arc/search":
			require.Equal(t, "777", r.URL.Query().Get("mid"))
			if r.URL.Query().Get("pn") != "1" {
				writeJSON(t, w, map[string]any{
					"code":    0,
					"message": "0",
					"data": map[string]any{
						"list": map[string]any{"vlist": []map[string]any{}},
						"page": map[string]any{"pn": 2, "ps": 30, "count": 1},
					},
				})
				return
			}
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data": map[string]any{
					"list": map[string]any{
						"vlist": []map[string]any{
							{
								"aid":         int64(12345),
								"bvid":        "BV1mockbili",
								"title":       "Bilibili E2E mock video",
								"description": "mock bilibili desc",
								"play":        int64(1000),
								"comment":     int64(5),
								"created":     int64(1710000000),
								"pic":         "https://example.test/bili-cover.png",
								"length":      "01:06",
								"author":      "mock bili author",
								"mid":         int64(777),
							},
						},
					},
					"page": map[string]any{"pn": 1, "ps": 30, "count": 1},
				},
			})
		case "/x/web-interface/view/detail":
			bvid := r.URL.Query().Get("bvid")
			aid := r.URL.Query().Get("aid")
			if bvid != "" {
				require.Equal(t, "BV1mockbili", bvid)
			}
			if aid != "" {
				require.Equal(t, "12345", aid)
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
			t.Fatalf("unexpected bilibili creator unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}
