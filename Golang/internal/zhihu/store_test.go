// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/zhihu/store_test.go
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

package zhihu

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestZhihuDBStoreMatchesPythonQueryUpdateWithoutUniqueIndex(t *testing.T) {
	db := newZhihuTestDB(t)
	store := NewDBStore(db)
	ctx := context.Background()

	firstContent := testZhihuContent("zh_content_db", func(content *Content) {
		content.Title = "first content"
		content.AddTs = 100
		content.LastModifyTs = 100
	})
	require.NoError(t, store.SaveContent(ctx, firstContent))

	secondContent := testZhihuContent("zh_content_db", func(content *Content) {
		content.Title = "second content"
		content.AddTs = 200
		content.LastModifyTs = 200
	})
	require.NoError(t, store.SaveContent(ctx, secondContent))

	var contents []Content
	require.NoError(t, db.Where("content_id = ?", "zh_content_db").Find(&contents).Error)
	require.Len(t, contents, 1)
	require.Equal(t, "second content", contents[0].Title)
	require.Equal(t, int64(100), contents[0].AddTs)
	require.Equal(t, int64(200), contents[0].LastModifyTs)

	firstComment := testZhihuComment("zh_comment_db", func(comment *Comment) {
		comment.Content = "first comment"
		comment.AddTs = 100
		comment.LastModifyTs = 100
	})
	require.NoError(t, store.SaveComment(ctx, firstComment))

	secondComment := testZhihuComment("zh_comment_db", func(comment *Comment) {
		comment.Content = "second comment"
		comment.AddTs = 200
		comment.LastModifyTs = 200
	})
	require.NoError(t, store.SaveComment(ctx, secondComment))

	var comments []Comment
	require.NoError(t, db.Where("comment_id = ?", "zh_comment_db").Find(&comments).Error)
	require.Len(t, comments, 1)
	require.Equal(t, "second comment", comments[0].Content)
	require.Equal(t, int64(100), comments[0].AddTs)
	require.Equal(t, int64(200), comments[0].LastModifyTs)

	firstCreator := testZhihuCreator("zh_creator_db", func(creator *Creator) {
		creator.UserNickname = "first creator"
		creator.AddTs = 100
		creator.LastModifyTs = 100
	})
	require.NoError(t, store.SaveCreator(ctx, firstCreator))

	secondCreator := testZhihuCreator("zh_creator_db", func(creator *Creator) {
		creator.UserNickname = "second creator"
		creator.AddTs = 200
		creator.LastModifyTs = 200
	})
	require.NoError(t, store.SaveCreator(ctx, secondCreator))

	var creators []Creator
	require.NoError(t, db.Where("user_id = ?", "zh_creator_db").Find(&creators).Error)
	require.Len(t, creators, 1)
	require.Equal(t, "second creator", creators[0].UserNickname)
	require.Equal(t, int64(100), creators[0].AddTs)
	require.Equal(t, int64(200), creators[0].LastModifyTs)

	firstHotItem := testZhihuHotSearchItem("zh_hot_item_db", func(item *HotSearchItem) {
		item.Title = "first hot"
		item.AddTs = 100
		item.LastModifyTs = 100
	})
	require.NoError(t, store.SaveHotSearchItem(ctx, firstHotItem))

	secondHotItem := testZhihuHotSearchItem("zh_hot_item_db", func(item *HotSearchItem) {
		item.Title = "second hot"
		item.AddTs = 200
		item.LastModifyTs = 200
	})
	require.NoError(t, store.SaveHotSearchItem(ctx, secondHotItem))

	var hotItems []HotSearchItem
	require.NoError(t, db.Where("hot_item_id = ?", "zh_hot_item_db").Find(&hotItems).Error)
	require.Len(t, hotItems, 1)
	require.Equal(t, "second hot", hotItems[0].Title)
	require.Equal(t, int64(100), hotItems[0].AddTs)
	require.Equal(t, int64(200), hotItems[0].LastModifyTs)

	firstRelation := testZhihuHotSearchRelation("zh_hot_rel_db", func(relation *HotSearchContentRelation) {
		relation.RankNo = 1
		relation.AddTs = 100
		relation.LastModifyTs = 100
	})
	require.NoError(t, store.SaveHotSearchRelation(ctx, firstRelation))

	secondRelation := testZhihuHotSearchRelation("zh_hot_rel_db", func(relation *HotSearchContentRelation) {
		relation.RankNo = 2
		relation.AddTs = 200
		relation.LastModifyTs = 200
	})
	require.NoError(t, store.SaveHotSearchRelation(ctx, secondRelation))

	var relations []HotSearchContentRelation
	require.NoError(t, db.Where("rel_id = ?", "zh_hot_rel_db").Find(&relations).Error)
	require.Len(t, relations, 1)
	require.Equal(t, 2, relations[0].RankNo)
	require.Equal(t, int64(100), relations[0].AddTs)
	require.Equal(t, int64(200), relations[0].LastModifyTs)
}

func TestZhihuSQLiteStorePreservesOriginalAddTsLikePython(t *testing.T) {
	db := newZhihuTestDB(t)
	store := NewSQLiteDBStore(db)
	require.NoError(t, store.AutoMigrate())
	ctx := context.Background()

	firstContent := testZhihuContent("zh_content_sqlite", func(content *Content) {
		content.Title = "first content"
		content.AddTs = 100
		content.LastModifyTs = 100
	})
	require.NoError(t, store.SaveContent(ctx, firstContent))

	secondContent := testZhihuContent("zh_content_sqlite", func(content *Content) {
		content.Title = "second content"
		content.AddTs = 200
		content.LastModifyTs = 200
	})
	require.NoError(t, store.SaveContent(ctx, secondContent))

	var content Content
	require.NoError(t, db.Where("content_id = ?", "zh_content_sqlite").First(&content).Error)
	require.Equal(t, "second content", content.Title)
	require.Equal(t, int64(100), content.AddTs)
	require.Equal(t, int64(200), content.LastModifyTs)

	firstComment := testZhihuComment("zh_comment_sqlite", func(comment *Comment) {
		comment.Content = "first comment"
		comment.AddTs = 100
		comment.LastModifyTs = 100
	})
	require.NoError(t, store.SaveComment(ctx, firstComment))

	secondComment := testZhihuComment("zh_comment_sqlite", func(comment *Comment) {
		comment.Content = "second comment"
		comment.AddTs = 200
		comment.LastModifyTs = 200
	})
	require.NoError(t, store.SaveComment(ctx, secondComment))

	var comment Comment
	require.NoError(t, db.Where("comment_id = ?", "zh_comment_sqlite").First(&comment).Error)
	require.Equal(t, "second comment", comment.Content)
	require.Equal(t, int64(100), comment.AddTs)
	require.Equal(t, int64(200), comment.LastModifyTs)

	firstCreator := testZhihuCreator("zh_creator_sqlite", func(creator *Creator) {
		creator.UserNickname = "first creator"
		creator.AddTs = 100
		creator.LastModifyTs = 100
	})
	require.NoError(t, store.SaveCreator(ctx, firstCreator))

	secondCreator := testZhihuCreator("zh_creator_sqlite", func(creator *Creator) {
		creator.UserNickname = "second creator"
		creator.AddTs = 200
		creator.LastModifyTs = 200
	})
	require.NoError(t, store.SaveCreator(ctx, secondCreator))

	var creator Creator
	require.NoError(t, db.Where("user_id = ?", "zh_creator_sqlite").First(&creator).Error)
	require.Equal(t, "second creator", creator.UserNickname)
	require.Equal(t, int64(100), creator.AddTs)
	require.Equal(t, int64(200), creator.LastModifyTs)
}

func newZhihuTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Content{}, &Comment{}, &Creator{}, &HotSearchItem{}, &HotSearchContentRelation{}))
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_zhihu_content_content_id ON zhihu_content(content_id);").Error)
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_zhihu_comment_comment_id ON zhihu_comment(comment_id);").Error)
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_zhihu_creator_user_id ON zhihu_creator(user_id);").Error)
	return db
}

func testZhihuContent(contentID string, mutate func(*Content)) *Content {
	content := &Content{
		ContentID:     contentID,
		ContentType:   "answer",
		ContentText:   "zh content text",
		ContentURL:    "https://www.zhihu.com/question/1/answer/" + contentID,
		QuestionID:    "question-1",
		Title:         "zh content",
		Desc:          "zh desc",
		CreatedTime:   "1700000000",
		UpdatedTime:   "1700000010",
		VoteupCount:   10,
		CommentCount:  2,
		SourceKeyword: "mock",
		UserID:        "zh_user",
		UserLink:      "https://www.zhihu.com/people/zh_user",
		UserNickname:  "zh author",
		UserAvatar:    "https://example.test/zh-avatar.png",
		UserURLToken:  "zh_user",
		AddTs:         100,
		LastModifyTs:  100,
	}
	if mutate != nil {
		mutate(content)
	}
	return content
}

func testZhihuComment(commentID string, mutate func(*Comment)) *Comment {
	comment := &Comment{
		CommentID:       commentID,
		Content:         "zh comment",
		PublishTime:     "1700000020",
		SubCommentCount: 0,
		LikeCount:       1,
		ContentID:       "zh_content_db",
		ContentType:     "answer",
		UserID:          "zh_comment_user",
		UserLink:        "https://www.zhihu.com/people/zh_comment_user",
		UserNickname:    "zh commenter",
		UserAvatar:      "https://example.test/zh-commenter.png",
		AddTs:           100,
		LastModifyTs:    100,
	}
	if mutate != nil {
		mutate(comment)
	}
	return comment
}

func testZhihuCreator(userID string, mutate func(*Creator)) *Creator {
	creator := &Creator{
		UserID:         userID,
		UserLink:       "https://www.zhihu.com/people/" + userID,
		UserNickname:   "zh creator",
		UserAvatar:     "https://example.test/zh-creator.png",
		URLToken:       userID,
		Gender:         "m",
		IPLocation:     "北京",
		Follows:        2,
		Fans:           10,
		AnwserCount:    3,
		VideoCount:     1,
		QuestionCount:  4,
		ArticleCount:   5,
		ColumnCount:    6,
		GetVoteupCount: 100,
		AddTs:          100,
		LastModifyTs:   100,
	}
	if mutate != nil {
		mutate(creator)
	}
	return creator
}

func testZhihuHotSearchItem(hotItemID string, mutate func(*HotSearchItem)) *HotSearchItem {
	item := &HotSearchItem{
		HotItemID:    hotItemID,
		Platform:     "zhihu",
		BoardName:    "hot_search",
		SnapshotDate: "2026-06-21",
		SnapshotTs:   1782000000,
		RankNo:       1,
		Title:        "zh hot",
		Keyword:      "zh hot",
		HotValue:     "100",
		AddTs:        100,
		LastModifyTs: 100,
	}
	if mutate != nil {
		mutate(item)
	}
	return item
}

func testZhihuHotSearchRelation(relID string, mutate func(*HotSearchContentRelation)) *HotSearchContentRelation {
	relation := &HotSearchContentRelation{
		RelID:         relID,
		HotItemID:     "zh_hot_item_db",
		Platform:      "zhihu",
		ContentID:     "zh_content_db",
		ContentType:   "answer",
		ContentTable:  "zhihu_content",
		RankNo:        1,
		SourceKeyword: "zh hot",
		AddTs:         100,
		LastModifyTs:  100,
	}
	if mutate != nil {
		mutate(relation)
	}
	return relation
}
