// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/bilibili/file_store_test.go
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

package bilibili

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewBiliStoreCreatesJSONStoreWithoutDB(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.Config{
		SaveDataOption: "json",
		CrawlerType:    "search",
		DataDir:        dataDir,
	}
	store, err := NewStore(cfg, nil)
	require.NoError(t, err)

	err = store.SaveVideo(context.Background(), testVideo("video_json", func(video *Video) {
		video.Title = "Bili JSON标题"
	}))
	require.NoError(t, err)

	jsonPath := filepath.Join(dataDir, "bilibili", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var rows []map[string]any
	require.NoError(t, json.Unmarshal(body, &rows))
	require.Len(t, rows, 1)
	require.Equal(t, "video_json", rows[0]["video_id"])
	require.Equal(t, "Bili JSON标题", rows[0]["title"])
}

func TestNewBiliStoreCreatesCSVStoreWithoutDB(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.Config{
		SaveDataOption: "csv",
		CrawlerType:    "search",
		DataDir:        dataDir,
	}
	store, err := NewStore(cfg, nil)
	require.NoError(t, err)

	err = store.SaveVideo(context.Background(), testVideo("video_csv", func(video *Video) {
		video.Title = "Bili CSV标题"
	}))
	require.NoError(t, err)

	csvPath := filepath.Join(dataDir, "bilibili", "1_search_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(csvPath)
	require.NoError(t, err)
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Contains(t, records[0], "video_id")
	require.Contains(t, records[0], "title")
	require.Contains(t, records[1], "video_csv")
	require.Contains(t, records[1], "Bili CSV标题")
}

func TestNewBiliStoreCreatesSQLiteStoreWithoutMySQL(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "media_crawler.db")
	cfg := &config.Config{
		SaveDataOption: "sqlite",
		SqliteDBPath:   dbPath,
		CrawlerType:    "search",
	}
	store, err := NewStore(cfg, nil)
	require.NoError(t, err)
	require.IsType(t, &SQLiteDBStore{}, store)

	ctx := context.Background()
	video := testVideo("video_sqlite", func(video *Video) {
		video.Title = "sqlite title"
	})
	require.NoError(t, store.SaveVideo(ctx, video))

	video.Title = "sqlite updated title"
	require.NoError(t, store.SaveVideo(ctx, video))

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)
	var rows []Video
	require.NoError(t, db.Where("video_id = ?", "video_sqlite").Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "sqlite updated title", rows[0].Title)
}

func TestBiliDBStoreSaveVideoMatchesPythonQueryUpdateWithoutUniqueIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Video{}))
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_bilibili_vi_video_i_31c36e ON bilibili_video(video_id);").Error)

	store := NewDBStore(db)
	ctx := context.Background()
	first := testVideo("video_db", func(video *Video) {
		video.Title = "first title"
		video.AddTs = 100
		video.LastModifyTs = 100
	})
	require.NoError(t, store.SaveVideo(ctx, first))

	second := testVideo("video_db", func(video *Video) {
		video.Title = "second title"
		video.AddTs = 200
		video.LastModifyTs = 200
	})
	require.NoError(t, store.SaveVideo(ctx, second))

	var rows []Video
	require.NoError(t, db.Where("video_id = ?", "video_db").Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "second title", rows[0].Title)
	require.Equal(t, int64(100), rows[0].AddTs, "DB updates should preserve the original add_ts like Python")
	require.Equal(t, int64(200), rows[0].LastModifyTs)
}

func TestBiliDBStoreSaveCommentMatchesPythonQueryUpdateWithoutUniqueIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&VideoComment{}))
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_bilibili_vi_comment_41c34e ON bilibili_video_comment(comment_id);").Error)

	store := NewDBStore(db)
	ctx := context.Background()
	first := testVideoComment("comment_db", func(comment *VideoComment) {
		comment.Content = "first comment"
		comment.AddTs = 100
		comment.LastModifyTs = 100
		comment.LikeCount = "1"
	})
	require.NoError(t, store.SaveComment(ctx, first))

	second := testVideoComment("comment_db", func(comment *VideoComment) {
		comment.Content = "second comment"
		comment.AddTs = 200
		comment.LastModifyTs = 200
		comment.LikeCount = "2"
	})
	require.NoError(t, store.SaveComment(ctx, second))

	var rows []VideoComment
	require.NoError(t, db.Where("comment_id = ?", "comment_db").Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "second comment", rows[0].Content)
	require.Equal(t, int64(100), rows[0].AddTs, "DB updates should preserve the original add_ts like Python")
	require.Equal(t, int64(200), rows[0].LastModifyTs)
	require.Equal(t, "2", rows[0].LikeCount)
}

func TestBiliDBStoreSaveUpInfoMatchesPythonQueryUpdateWithoutUniqueIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&UpInfo{}))
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_bilibili_vi_user_123456 ON bilibili_up_info(user_id);").Error)

	store := NewDBStore(db)
	ctx := context.Background()
	first := testUpInfo("up_db", func(upInfo *UpInfo) {
		upInfo.Nickname = "first up"
		upInfo.AddTs = 100
		upInfo.LastModifyTs = 100
		upInfo.FollowerCount = 10
	})
	require.NoError(t, store.SaveUpInfo(ctx, first))

	second := testUpInfo("up_db", func(upInfo *UpInfo) {
		upInfo.Nickname = "second up"
		upInfo.AddTs = 200
		upInfo.LastModifyTs = 200
		upInfo.FollowerCount = 20
	})
	require.NoError(t, store.SaveUpInfo(ctx, second))

	var rows []UpInfo
	require.NoError(t, db.Where("user_id = ?", "up_db").Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "second up", rows[0].Nickname)
	require.Equal(t, int64(100), rows[0].AddTs, "DB updates should preserve the original add_ts like Python")
	require.Equal(t, int64(200), rows[0].LastModifyTs)
	require.Equal(t, int64(20), rows[0].FollowerCount)
}

func TestBiliSQLiteStorePreservesOriginalAddTsLikePython(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	store := NewSQLiteDBStore(db)
	require.NoError(t, store.AutoMigrate())
	ctx := context.Background()

	firstVideo := testVideo("video_sqlite_add_ts", func(video *Video) {
		video.Title = "first video"
		video.AddTs = 100
		video.LastModifyTs = 100
	})
	require.NoError(t, store.SaveVideo(ctx, firstVideo))

	secondVideo := testVideo("video_sqlite_add_ts", func(video *Video) {
		video.Title = "second video"
		video.AddTs = 200
		video.LastModifyTs = 200
	})
	require.NoError(t, store.SaveVideo(ctx, secondVideo))

	var video Video
	require.NoError(t, db.Where("video_id = ?", "video_sqlite_add_ts").First(&video).Error)
	require.Equal(t, "second video", video.Title)
	require.Equal(t, int64(100), video.AddTs)
	require.Equal(t, int64(200), video.LastModifyTs)

	firstComment := testVideoComment("comment_sqlite_add_ts", func(comment *VideoComment) {
		comment.Content = "first comment"
		comment.AddTs = 100
		comment.LastModifyTs = 100
	})
	require.NoError(t, store.SaveComment(ctx, firstComment))

	secondComment := testVideoComment("comment_sqlite_add_ts", func(comment *VideoComment) {
		comment.Content = "second comment"
		comment.AddTs = 200
		comment.LastModifyTs = 200
	})
	require.NoError(t, store.SaveComment(ctx, secondComment))

	var comment VideoComment
	require.NoError(t, db.Where("comment_id = ?", "comment_sqlite_add_ts").First(&comment).Error)
	require.Equal(t, "second comment", comment.Content)
	require.Equal(t, int64(100), comment.AddTs)
	require.Equal(t, int64(200), comment.LastModifyTs)

	firstUpInfo := testUpInfo("up_sqlite_add_ts", func(upInfo *UpInfo) {
		upInfo.Nickname = "first up"
		upInfo.AddTs = 100
		upInfo.LastModifyTs = 100
	})
	require.NoError(t, store.SaveUpInfo(ctx, firstUpInfo))

	secondUpInfo := testUpInfo("up_sqlite_add_ts", func(upInfo *UpInfo) {
		upInfo.Nickname = "second up"
		upInfo.AddTs = 200
		upInfo.LastModifyTs = 200
	})
	require.NoError(t, store.SaveUpInfo(ctx, secondUpInfo))

	var upInfo UpInfo
	require.NoError(t, db.Where("user_id = ?", "up_sqlite_add_ts").First(&upInfo).Error)
	require.Equal(t, "second up", upInfo.Nickname)
	require.Equal(t, int64(100), upInfo.AddTs)
	require.Equal(t, int64(200), upInfo.LastModifyTs)
}

func TestNewBiliCrawlerUsesConfiguredStoreFactory(t *testing.T) {
	cfg := &config.Config{
		SaveDataOption:    "json",
		CrawlerType:       "search",
		DataDir:           t.TempDir(),
		MaxConcurrencyNum: 1,
	}

	crawler := NewCrawler(CrawlerOptions{Config: cfg})

	require.IsType(t, &FileStore{}, crawler.store)
}

func testVideo(videoID string, mutate func(*Video)) *Video {
	now := time.Now().Unix()
	video := &Video{
		UserID:         "bili_user",
		Nickname:       "bili author",
		Avatar:         "https://example.test/bili-avatar.png",
		AddTs:          now,
		LastModifyTs:   now,
		VideoID:        videoID,
		Bvid:           "BV1mock",
		VideoType:      "科技",
		Title:          "bili title",
		Desc:           "bili desc",
		CreateTime:     now,
		LikedCount:     "10",
		VideoPlayCount: "100",
		VideoDanmaku:   "2",
		VideoComment:   "3",
		VideoDuration:  "60",
		VideoURL:       "https://www.bilibili.com/video/BV1mock",
		VideoCover:     "https://example.test/cover.png",
		SourceKeyword:  "mock",
	}
	if mutate != nil {
		mutate(video)
	}
	return video
}

func testVideoComment(commentID string, mutate func(*VideoComment)) *VideoComment {
	now := time.Now().Unix()
	comment := &VideoComment{
		UserID:          "bili_comment_user",
		Nickname:        "bili commenter",
		Avatar:          "https://example.test/bili-commenter.png",
		AddTs:           now,
		LastModifyTs:    now,
		CommentID:       commentID,
		VideoID:         "video_db",
		Content:         "bili comment",
		CreateTime:      now,
		SubCommentCount: "0",
		LikeCount:       "1",
	}
	if mutate != nil {
		mutate(comment)
	}
	return comment
}

func testUpInfo(userID string, mutate func(*UpInfo)) *UpInfo {
	now := time.Now().Unix()
	upInfo := &UpInfo{
		UserID:         userID,
		Nickname:       "bili up",
		Avatar:         "https://example.test/bili-up.png",
		AddTs:          now,
		LastModifyTs:   now,
		Description:    "bili up desc",
		FollowerCount:  10,
		FollowingCount: 2,
		ContentCount:   3,
	}
	if mutate != nil {
		mutate(upInfo)
	}
	return upInfo
}
