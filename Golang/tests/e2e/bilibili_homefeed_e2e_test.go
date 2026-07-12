// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/bilibili_homefeed_e2e_test.go
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

func TestCLIBilibiliHomefeedJSONWithUnifiedRunner(t *testing.T) {
	runner := newBilibiliHomefeedRunner(t, "bili-homefeed-json-client")
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "bili",
		CrawlerType:          "homefeed",
		SaveDataOption:       "json",
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Timeout: 20 * time.Second,
	})

	rows := e2e.ReadJSONRows(t, filepath.Join(runner.DataDir, "bilibili", "json", "homefeed_contents_"+time.Now().Format("2006-01-02")+".json"), output)
	require.Len(t, rows, 1, "command output:\n%s", output)
	row := rows[0]
	require.Equal(t, "12345", row["video_id"])
	require.Equal(t, "BV1homefeed", row["bvid"])
	require.Equal(t, "Bilibili homefeed E2E mock video", row["title"])
	require.Equal(t, "mock bilibili homefeed desc", row["desc"])
	require.Equal(t, "1710000300", row["create_time"])
	require.Equal(t, "88", row["liked_count"])
	require.Equal(t, "1000", row["video_play_count"])
	require.Equal(t, "12", row["video_danmaku"])
	require.Equal(t, "5", row["video_comment"])
	require.Equal(t, "66", row["duration"])
	require.Equal(t, "https://www.bilibili.com/video/BV1homefeed", row["video_url"])
	require.Equal(t, "https://example.test/bili-home-cover.png", row["video_cover_url"])
	require.Equal(t, "", row["source_keyword"])
	require.Contains(t, row, "add_ts")
	require.Nil(t, row["add_ts"])
}

func TestCLIBilibiliHomefeedCSVWithUnifiedRunner(t *testing.T) {
	runner := newBilibiliHomefeedRunner(t, "bili-homefeed-csv-client")
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "bili",
		CrawlerType:          "homefeed",
		SaveDataOption:       "csv",
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Timeout: 20 * time.Second,
	})

	records := e2e.ReadCSVRecords(t, filepath.Join(runner.DataDir, "bilibili", "1_homefeed_contents_"+time.Now().Format("2006-01-02")+".csv"), output)
	require.Len(t, records, 2, "command output:\n%s", output)
	require.Equal(t, "12345", e2e.CSVCellAt(records, 1, "video_id"))
	require.Equal(t, "BV1homefeed", e2e.CSVCellAt(records, 1, "bvid"))
	require.Equal(t, "Bilibili homefeed E2E mock video", e2e.CSVCellAt(records, 1, "title"))
	require.Equal(t, "mock bilibili homefeed desc", e2e.CSVCellAt(records, 1, "desc"))
	require.Equal(t, "1710000300", e2e.CSVCellAt(records, 1, "create_time"))
	require.Equal(t, "88", e2e.CSVCellAt(records, 1, "liked_count"))
	require.Equal(t, "1000", e2e.CSVCellAt(records, 1, "video_play_count"))
	require.Equal(t, "12", e2e.CSVCellAt(records, 1, "video_danmaku"))
	require.Equal(t, "5", e2e.CSVCellAt(records, 1, "video_comment"))
	require.Equal(t, "66", e2e.CSVCellAt(records, 1, "duration"))
	require.Equal(t, "https://www.bilibili.com/video/BV1homefeed", e2e.CSVCellAt(records, 1, "video_url"))
	require.Equal(t, "https://example.test/bili-home-cover.png", e2e.CSVCellAt(records, 1, "video_cover_url"))
	require.Equal(t, "", e2e.CSVCellAt(records, 1, "source_keyword"))
	require.Equal(t, "", e2e.CSVCellAt(records, 1, "add_ts"))
}

func TestCLIBilibiliHomefeedSQLiteWithUnifiedRunner(t *testing.T) {
	runner := newBilibiliHomefeedRunner(t, "bili-homefeed-sqlite-client")
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "bili",
		CrawlerType:          "homefeed",
		SaveDataOption:       "sqlite",
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Timeout: 20 * time.Second,
	})

	rows := e2e.ReadSQLiteRows(t, filepath.Join(runner.WorkDir, "media_crawler.db"), "bilibili_video", "video_id", output)
	require.Len(t, rows, 1, "command output:\n%s", output)
	require.Equal(t, "12345", e2e.SQLiteCell(rows, 0, "video_id"))
	require.Equal(t, "BV1homefeed", e2e.SQLiteCell(rows, 0, "bvid"))
	require.Equal(t, "Bilibili homefeed E2E mock video", e2e.SQLiteCell(rows, 0, "title"))
	require.Equal(t, "mock bilibili homefeed desc", e2e.SQLiteCell(rows, 0, "desc"))
	require.Equal(t, "1710000300", e2e.SQLiteCell(rows, 0, "create_time"))
	require.Equal(t, "88", e2e.SQLiteCell(rows, 0, "liked_count"))
	require.Equal(t, "1000", e2e.SQLiteCell(rows, 0, "video_play_count"))
	require.Equal(t, "12", e2e.SQLiteCell(rows, 0, "video_danmaku"))
	require.Equal(t, "5", e2e.SQLiteCell(rows, 0, "video_comment"))
	require.Equal(t, "66", e2e.SQLiteCell(rows, 0, "duration"))
	require.Equal(t, "https://www.bilibili.com/video/BV1homefeed", e2e.SQLiteCell(rows, 0, "video_url"))
	require.Equal(t, "https://example.test/bili-home-cover.png", e2e.SQLiteCell(rows, 0, "video_cover_url"))
	require.Equal(t, "", e2e.SQLiteCell(rows, 0, "source_keyword"))
	require.NotEmpty(t, e2e.SQLiteCell(rows, 0, "add_ts"))
	require.NotEmpty(t, e2e.SQLiteCell(rows, 0, "last_modify_ts"))
}

func TestCLIBilibiliHomefeedDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := newBilibiliHomefeedRunner(t, "bili-homefeed-db-client")
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "bili",
		CrawlerType:          "homefeed",
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
		VideoID        string
		Bvid           string
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
		SELECT video_id, bvid, title, `+"`desc`"+`, create_time, liked_count,
		       video_play_count, video_danmaku, video_comment, duration,
		       video_url, video_cover_url, source_keyword, add_ts, last_modify_ts
		FROM bilibili_video
		WHERE video_id = ?
	`, "12345").Scan(
		&row.VideoID,
		&row.Bvid,
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
	require.Equal(t, "BV1homefeed", row.Bvid)
	require.Equal(t, "Bilibili homefeed E2E mock video", row.Title)
	require.Equal(t, "mock bilibili homefeed desc", row.Description)
	require.Equal(t, int64(1710000300), row.CreateTime)
	require.Equal(t, "88", row.LikedCount)
	require.Equal(t, "1000", row.VideoPlayCount)
	require.Equal(t, "12", row.VideoDanmaku)
	require.Equal(t, "5", row.VideoComment)
	require.Equal(t, "66", row.Duration)
	require.Equal(t, "https://www.bilibili.com/video/BV1homefeed", row.VideoURL)
	require.Equal(t, "https://example.test/bili-home-cover.png", row.VideoCoverURL)
	require.Equal(t, "", row.SourceKeyword)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func newBilibiliHomefeedRunner(t *testing.T, clientID string) *e2e.MockPlatformRunner {
	t.Helper()
	return e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "bili",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  clientID,
				Connected: true,
				Nicknames: map[string]string{"bili": "mock-bili"},
				Cookies:   map[string]string{"bili": "SESSDATA=homefeed-unified; buvid3=homefeed-unified"},
			},
		},
		PlatformHandler: newBilibiliUnifiedHomefeedMockHandler(t),
	})
}

func newBilibiliUnifiedHomefeedMockHandler(t *testing.T) http.Handler {
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
		case "/x/web-interface/wbi/index/top/feed/rcmd":
			if r.URL.Query().Get("fresh_idx") != "1" {
				writeJSON(t, w, map[string]any{
					"code":    0,
					"message": "0",
					"data":    map[string]any{"item": []map[string]any{}},
				})
				return
			}
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data": map[string]any{
					"item": []map[string]any{
						{
							"id":        int64(12345),
							"bvid":      "BV1homefeed",
							"cid":       int64(111),
							"goto":      "av",
							"show_info": 1,
							"uri":       "https://www.bilibili.com/video/BV1homefeed",
							"pic":       "https://example.test/bili-home-cover.png",
							"title":     "Bilibili homefeed E2E mock video",
							"duration":  int64(66),
							"pubdate":   int64(1710000300),
							"owner": map[string]any{
								"mid":  int64(777),
								"name": "mock bili author",
								"face": "https://example.test/bili-avatar.png",
							},
							"stat": map[string]any{
								"aid":     int64(12345),
								"view":    int64(1000),
								"danmaku": int64(12),
								"reply":   int64(5),
								"like":    int64(88),
							},
						},
						{
							"id":        int64(99999),
							"bvid":      "BVskipLive",
							"goto":      "live",
							"show_info": 1,
							"title":     "should skip live",
						},
						{
							"id":        int64(88888),
							"bvid":      "BVskipHidden",
							"goto":      "av",
							"show_info": 0,
							"title":     "should skip hidden",
						},
					},
				},
			})
		case "/x/web-interface/view/detail":
			bvid := r.URL.Query().Get("bvid")
			aid := r.URL.Query().Get("aid")
			if bvid != "" {
				require.Equal(t, "BV1homefeed", bvid)
			}
			if aid != "" {
				require.Equal(t, "12345", aid)
			}
			writeJSON(t, w, map[string]any{
				"code":    0,
				"message": "0",
				"data": map[string]any{
					"View": map[string]any{
						"bvid":     "BV1homefeed",
						"aid":      int64(12345),
						"tname":    "科技",
						"pic":      "https://example.test/bili-home-cover.png",
						"title":    "Bilibili homefeed E2E mock video",
						"pubdate":  int64(1710000300),
						"ctime":    int64(1710000300),
						"desc":     "mock bilibili homefeed desc",
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
			t.Fatalf("unexpected bilibili homefeed unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}
