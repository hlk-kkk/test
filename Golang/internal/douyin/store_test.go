// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/douyin/store_test.go
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

package douyin

import (
	"context"
	"testing"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewDouyinStoreCreatesDBStoreWithInjectedDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Aweme{}))

	store, err := NewStore(&config.Config{
		SaveDataOption: "db",
		CrawlerType:    "search",
	}, db)
	require.NoError(t, err)
	require.IsType(t, &DBStore{}, store)

	first := testAweme("aweme_db")
	first.AddTs = 100
	first.LastModifyTs = 110
	require.NoError(t, store.SaveAweme(context.Background(), first))

	updated := testAweme("aweme_db")
	updated.Title = "updated db title"
	updated.AddTs = 999
	updated.LastModifyTs = 220
	require.NoError(t, store.SaveAweme(context.Background(), updated))

	var rows []Aweme
	require.NoError(t, db.Where("aweme_id = ?", "aweme_db").Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "updated db title", rows[0].Title)
	require.Equal(t, int64(100), rows[0].AddTs)
	require.Equal(t, int64(220), rows[0].LastModifyTs)
}

func TestDouyinDBStoreMatchesPythonQueryUpdateWithoutUniqueIndex(t *testing.T) {
	db := newDouyinTestDB(t)
	store := NewDBStore(db)
	ctx := context.Background()

	firstAweme := testAweme("aweme_query_db")
	firstAweme.AddTs = 100
	firstAweme.LastModifyTs = 100
	require.NoError(t, store.SaveAweme(ctx, firstAweme))

	secondAweme := testAweme("aweme_query_db")
	secondAweme.Title = "second aweme"
	secondAweme.AddTs = 200
	secondAweme.LastModifyTs = 200
	require.NoError(t, store.SaveAweme(ctx, secondAweme))

	var awemes []Aweme
	require.NoError(t, db.Where("aweme_id = ?", "aweme_query_db").Find(&awemes).Error)
	require.Len(t, awemes, 1)
	require.Equal(t, "second aweme", awemes[0].Title)
	require.Equal(t, int64(100), awemes[0].AddTs)
	require.Equal(t, int64(200), awemes[0].LastModifyTs)

	firstComment := testAwemeComment("comment_query_db")
	firstComment.AddTs = 100
	firstComment.LastModifyTs = 100
	require.NoError(t, store.SaveComment(ctx, firstComment))

	secondComment := testAwemeComment("comment_query_db")
	secondComment.Content = "second comment"
	secondComment.AddTs = 200
	secondComment.LastModifyTs = 200
	require.NoError(t, store.SaveComment(ctx, secondComment))

	var comments []AwemeComment
	require.NoError(t, db.Where("comment_id = ?", "comment_query_db").Find(&comments).Error)
	require.Len(t, comments, 1)
	require.Equal(t, "second comment", comments[0].Content)
	require.Equal(t, int64(100), comments[0].AddTs)
	require.Equal(t, int64(200), comments[0].LastModifyTs)

	firstCreator := testDouyinCreator("creator_query_db")
	firstCreator.AddTs = 100
	firstCreator.LastModifyTs = 100
	require.NoError(t, store.SaveCreator(ctx, firstCreator))

	secondCreator := testDouyinCreator("creator_query_db")
	secondCreator.Nickname = "second creator"
	secondCreator.AddTs = 200
	secondCreator.LastModifyTs = 200
	require.NoError(t, store.SaveCreator(ctx, secondCreator))

	var creators []Creator
	require.NoError(t, db.Where("user_id = ?", "creator_query_db").Find(&creators).Error)
	require.Len(t, creators, 1)
	require.Equal(t, "second creator", creators[0].Nickname)
	require.Equal(t, int64(100), creators[0].AddTs)
	require.Equal(t, int64(200), creators[0].LastModifyTs)
}

func TestDouyinSQLiteStorePreservesOriginalAddTsLikePython(t *testing.T) {
	db := newDouyinTestDB(t)
	store := NewSQLiteDBStore(db)
	require.NoError(t, store.AutoMigrate())
	ctx := context.Background()

	firstAweme := testAweme("aweme_sqlite")
	firstAweme.AddTs = 100
	firstAweme.LastModifyTs = 100
	require.NoError(t, store.SaveAweme(ctx, firstAweme))

	secondAweme := testAweme("aweme_sqlite")
	secondAweme.Title = "second aweme"
	secondAweme.AddTs = 200
	secondAweme.LastModifyTs = 200
	require.NoError(t, store.SaveAweme(ctx, secondAweme))

	var aweme Aweme
	require.NoError(t, db.Where("aweme_id = ?", "aweme_sqlite").First(&aweme).Error)
	require.Equal(t, "second aweme", aweme.Title)
	require.Equal(t, int64(100), aweme.AddTs)
	require.Equal(t, int64(200), aweme.LastModifyTs)

	firstComment := testAwemeComment("comment_sqlite")
	firstComment.AddTs = 100
	firstComment.LastModifyTs = 100
	require.NoError(t, store.SaveComment(ctx, firstComment))

	secondComment := testAwemeComment("comment_sqlite")
	secondComment.Content = "second comment"
	secondComment.AddTs = 200
	secondComment.LastModifyTs = 200
	require.NoError(t, store.SaveComment(ctx, secondComment))

	var comment AwemeComment
	require.NoError(t, db.Where("comment_id = ?", "comment_sqlite").First(&comment).Error)
	require.Equal(t, "second comment", comment.Content)
	require.Equal(t, int64(100), comment.AddTs)
	require.Equal(t, int64(200), comment.LastModifyTs)

	firstCreator := testDouyinCreator("creator_sqlite")
	firstCreator.AddTs = 100
	firstCreator.LastModifyTs = 100
	require.NoError(t, store.SaveCreator(ctx, firstCreator))

	secondCreator := testDouyinCreator("creator_sqlite")
	secondCreator.Nickname = "second creator"
	secondCreator.AddTs = 200
	secondCreator.LastModifyTs = 200
	require.NoError(t, store.SaveCreator(ctx, secondCreator))

	var creator Creator
	require.NoError(t, db.Where("user_id = ?", "creator_sqlite").First(&creator).Error)
	require.Equal(t, "second creator", creator.Nickname)
	require.Equal(t, int64(100), creator.AddTs)
	require.Equal(t, int64(200), creator.LastModifyTs)
}

func newDouyinTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Aweme{}, &AwemeComment{}, &Creator{}))
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_douyin_awem_aweme_i_6f7bc6 ON douyin_aweme(aweme_id);").Error)
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_douyin_awem_comment_fcd7e4 ON douyin_aweme_comment(comment_id);").Error)
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_dy_creator_user_id ON dy_creator(user_id);").Error)
	return db
}

func testAweme(awemeID string) *Aweme {
	return &Aweme{
		UserID:           "dy-user",
		SecUID:           "sec-dy-user",
		ShortUserID:      "123456",
		UserUniqueID:     "mock_dy_user",
		Nickname:         "mock dy user",
		Avatar:           "https://example.test/avatar.png",
		UserSignature:    "signature",
		IPLocation:       "上海",
		AwemeID:          awemeID,
		AwemeType:        "0",
		Title:            "db title",
		Desc:             "db desc",
		CreateTime:       "1710000000",
		LikedCount:       "88",
		CommentCount:     "5",
		ShareCount:       "3",
		CollectedCount:   "20",
		AwemeURL:         "https://www.douyin.com/video/" + awemeID,
		CoverURL:         "https://example.test/cover.png",
		VideoDownloadURL: "https://example.test/video.mp4",
		SourceKeyword:    "mock-keyword",
		IsAIGenerated:    1,
	}
}

func testAwemeComment(commentID string) *AwemeComment {
	return &AwemeComment{
		UserID:          "dy-comment-user",
		SecUID:          "sec-dy-comment-user",
		ShortUserID:     "654321",
		UserUniqueID:    "mock_dy_comment_user",
		Nickname:        "mock dy commenter",
		Avatar:          "https://example.test/comment-avatar.png",
		UserSignature:   "comment signature",
		IPLocation:      "北京",
		CommentID:       commentID,
		AwemeID:         "aweme_query_db",
		Content:         "db comment",
		CreateTime:      "1710000010",
		SubCommentCount: "0",
		LikeCount:       "1",
		Pictures:        "[]",
	}
}

func testDouyinCreator(userID string) *Creator {
	return &Creator{
		UserID:      userID,
		Nickname:    "mock dy creator",
		Avatar:      "https://example.test/creator-avatar.png",
		IPLocation:  "深圳",
		Desc:        "creator desc",
		Gender:      "m",
		Follows:     "2",
		Fans:        "10",
		Interaction: "100",
		VideosCount: "3",
	}
}
