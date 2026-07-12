// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/kuaishou/store_test.go
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

package kuaishou

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestKuaishouDBStoreMatchesPythonQueryUpdateWithoutUniqueIndex(t *testing.T) {
	db := newKuaishouTestDB(t)
	store := NewDBStore(db)
	ctx := context.Background()

	firstVideo := testKuaishouVideo("ks_video_db", func(video *Video) {
		video.Title = "first video"
		video.AddTs = 100
		video.LastModifyTs = 100
	})
	require.NoError(t, store.SaveVideo(ctx, firstVideo))

	secondVideo := testKuaishouVideo("ks_video_db", func(video *Video) {
		video.Title = "second video"
		video.AddTs = 200
		video.LastModifyTs = 200
	})
	require.NoError(t, store.SaveVideo(ctx, secondVideo))

	var videos []Video
	require.NoError(t, db.Where("video_id = ?", "ks_video_db").Find(&videos).Error)
	require.Len(t, videos, 1)
	require.Equal(t, "second video", videos[0].Title)
	require.Equal(t, int64(100), videos[0].AddTs)
	require.Equal(t, int64(200), videos[0].LastModifyTs)

	firstComment := testKuaishouComment("ks_comment_db", func(comment *VideoComment) {
		comment.Content = "first comment"
		comment.AddTs = 100
		comment.LastModifyTs = 100
	})
	require.NoError(t, store.SaveComment(ctx, firstComment))

	secondComment := testKuaishouComment("ks_comment_db", func(comment *VideoComment) {
		comment.Content = "second comment"
		comment.AddTs = 200
		comment.LastModifyTs = 200
	})
	require.NoError(t, store.SaveComment(ctx, secondComment))

	var comments []VideoComment
	require.NoError(t, db.Where("comment_id = ?", "ks_comment_db").Find(&comments).Error)
	require.Len(t, comments, 1)
	require.Equal(t, "second comment", comments[0].Content)
	require.Equal(t, int64(100), comments[0].AddTs)
	require.Equal(t, int64(200), comments[0].LastModifyTs)

	firstCreator := testKuaishouCreator("ks_creator_db", func(creator *Creator) {
		creator.Nickname = "first creator"
		creator.AddTs = 100
		creator.LastModifyTs = 100
	})
	require.NoError(t, store.SaveCreator(ctx, firstCreator))

	secondCreator := testKuaishouCreator("ks_creator_db", func(creator *Creator) {
		creator.Nickname = "second creator"
		creator.AddTs = 200
		creator.LastModifyTs = 200
	})
	require.NoError(t, store.SaveCreator(ctx, secondCreator))

	var creators []Creator
	require.NoError(t, db.Where("user_id = ?", "ks_creator_db").Find(&creators).Error)
	require.Len(t, creators, 1)
	require.Equal(t, "second creator", creators[0].Nickname)
	require.Equal(t, int64(100), creators[0].AddTs)
	require.Equal(t, int64(200), creators[0].LastModifyTs)
}

func TestKuaishouSQLiteStorePreservesOriginalAddTsLikePython(t *testing.T) {
	db := newKuaishouTestDB(t)
	store := NewSQLiteDBStore(db)
	require.NoError(t, store.AutoMigrate())
	ctx := context.Background()

	firstVideo := testKuaishouVideo("ks_video_sqlite", func(video *Video) {
		video.Title = "first video"
		video.AddTs = 100
		video.LastModifyTs = 100
	})
	require.NoError(t, store.SaveVideo(ctx, firstVideo))

	secondVideo := testKuaishouVideo("ks_video_sqlite", func(video *Video) {
		video.Title = "second video"
		video.AddTs = 200
		video.LastModifyTs = 200
	})
	require.NoError(t, store.SaveVideo(ctx, secondVideo))

	var video Video
	require.NoError(t, db.Where("video_id = ?", "ks_video_sqlite").First(&video).Error)
	require.Equal(t, "second video", video.Title)
	require.Equal(t, int64(100), video.AddTs)
	require.Equal(t, int64(200), video.LastModifyTs)

	firstComment := testKuaishouComment("ks_comment_sqlite", func(comment *VideoComment) {
		comment.Content = "first comment"
		comment.AddTs = 100
		comment.LastModifyTs = 100
	})
	require.NoError(t, store.SaveComment(ctx, firstComment))

	secondComment := testKuaishouComment("ks_comment_sqlite", func(comment *VideoComment) {
		comment.Content = "second comment"
		comment.AddTs = 200
		comment.LastModifyTs = 200
	})
	require.NoError(t, store.SaveComment(ctx, secondComment))

	var comment VideoComment
	require.NoError(t, db.Where("comment_id = ?", "ks_comment_sqlite").First(&comment).Error)
	require.Equal(t, "second comment", comment.Content)
	require.Equal(t, int64(100), comment.AddTs)
	require.Equal(t, int64(200), comment.LastModifyTs)

	firstCreator := testKuaishouCreator("ks_creator_sqlite", func(creator *Creator) {
		creator.Nickname = "first creator"
		creator.AddTs = 100
		creator.LastModifyTs = 100
	})
	require.NoError(t, store.SaveCreator(ctx, firstCreator))

	secondCreator := testKuaishouCreator("ks_creator_sqlite", func(creator *Creator) {
		creator.Nickname = "second creator"
		creator.AddTs = 200
		creator.LastModifyTs = 200
	})
	require.NoError(t, store.SaveCreator(ctx, secondCreator))

	var creator Creator
	require.NoError(t, db.Where("user_id = ?", "ks_creator_sqlite").First(&creator).Error)
	require.Equal(t, "second creator", creator.Nickname)
	require.Equal(t, int64(100), creator.AddTs)
	require.Equal(t, int64(200), creator.LastModifyTs)
}

func newKuaishouTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Video{}, &VideoComment{}, &Creator{}, &HotSearchItem{}, &HotSearchContentRelation{}))
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_kuaishou_vi_video_i_c5c6a6 ON kuaishou_video(video_id);").Error)
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_kuaishou_vi_comment_ed48fa ON kuaishou_video_comment(comment_id);").Error)
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_kuaishou_creator_user_id ON kuaishou_creator(user_id);").Error)
	return db
}

func testKuaishouVideo(videoID string, mutate func(*Video)) *Video {
	video := &Video{
		UserID:        "ks_user",
		Nickname:      "ks author",
		Avatar:        "https://example.test/ks-avatar.png",
		AddTs:         100,
		LastModifyTs:  100,
		VideoID:       videoID,
		VideoType:     "NORMAL",
		Title:         "ks title",
		Desc:          "ks desc",
		CreateTime:    "1700000000",
		LikedCount:    "10",
		ViewdCount:    "100",
		VideoURL:      "https://www.kuaishou.com/short-video/" + videoID,
		VideoCoverURL: "https://example.test/ks-cover.png",
		VideoPlayURL:  "https://example.test/ks-play.mp4",
		SourceKeyword: "mock",
	}
	if mutate != nil {
		mutate(video)
	}
	return video
}

func testKuaishouComment(commentID string, mutate func(*VideoComment)) *VideoComment {
	comment := &VideoComment{
		UserID:          "ks_comment_user",
		Nickname:        "ks commenter",
		Avatar:          "https://example.test/ks-commenter.png",
		AddTs:           100,
		LastModifyTs:    100,
		CommentID:       commentID,
		VideoID:         "ks_video_db",
		Content:         "ks comment",
		CreateTime:      "1700000010",
		SubCommentCount: "0",
		LikeCount:       "1",
	}
	if mutate != nil {
		mutate(comment)
	}
	return comment
}

func testKuaishouCreator(userID string, mutate func(*Creator)) *Creator {
	creator := &Creator{
		UserID:       userID,
		Nickname:     "ks creator",
		Avatar:       "https://example.test/ks-creator.png",
		IPLocation:   "北京",
		AddTs:        100,
		LastModifyTs: 100,
		Desc:         "ks creator desc",
		Gender:       "m",
		Follows:      "2",
		Fans:         "10",
		VideosCount:  "3",
	}
	if mutate != nil {
		mutate(creator)
	}
	return creator
}
