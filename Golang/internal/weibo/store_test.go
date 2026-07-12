// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/weibo/store_test.go
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

package weibo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWeiboDBStoreMatchesPythonQueryUpdateWithoutUniqueIndex(t *testing.T) {
	db := newWeiboTestDB(t)
	store := NewDBStore(db)
	ctx := context.Background()

	firstNote := testWeiboNote("wb_note_db", func(note *Note) {
		note.Content = "first note"
		note.AddTs = 100
		note.LastModifyTs = 100
	})
	require.NoError(t, store.SaveNote(ctx, firstNote))

	secondNote := testWeiboNote("wb_note_db", func(note *Note) {
		note.Content = "second note"
		note.AddTs = 200
		note.LastModifyTs = 200
	})
	require.NoError(t, store.SaveNote(ctx, secondNote))

	var notes []Note
	require.NoError(t, db.Where("note_id = ?", "wb_note_db").Find(&notes).Error)
	require.Len(t, notes, 1)
	require.Equal(t, "second note", notes[0].Content)
	require.Equal(t, int64(100), notes[0].AddTs)
	require.Equal(t, int64(200), notes[0].LastModifyTs)

	firstComment := testWeiboComment("wb_comment_db", func(comment *Comment) {
		comment.Content = "first comment"
		comment.AddTs = 100
		comment.LastModifyTs = 100
	})
	require.NoError(t, store.SaveComment(ctx, firstComment))

	secondComment := testWeiboComment("wb_comment_db", func(comment *Comment) {
		comment.Content = "second comment"
		comment.AddTs = 200
		comment.LastModifyTs = 200
	})
	require.NoError(t, store.SaveComment(ctx, secondComment))

	var comments []Comment
	require.NoError(t, db.Where("comment_id = ?", "wb_comment_db").Find(&comments).Error)
	require.Len(t, comments, 1)
	require.Equal(t, "second comment", comments[0].Content)
	require.Equal(t, int64(100), comments[0].AddTs)
	require.Equal(t, int64(200), comments[0].LastModifyTs)

	firstCreator := testWeiboCreator("wb_creator_db", func(creator *Creator) {
		creator.Nickname = "first creator"
		creator.AddTs = 100
		creator.LastModifyTs = 100
	})
	require.NoError(t, store.SaveCreator(ctx, firstCreator))

	secondCreator := testWeiboCreator("wb_creator_db", func(creator *Creator) {
		creator.Nickname = "second creator"
		creator.AddTs = 200
		creator.LastModifyTs = 200
	})
	require.NoError(t, store.SaveCreator(ctx, secondCreator))

	var creators []Creator
	require.NoError(t, db.Where("user_id = ?", "wb_creator_db").Find(&creators).Error)
	require.Len(t, creators, 1)
	require.Equal(t, "second creator", creators[0].Nickname)
	require.Equal(t, int64(100), creators[0].AddTs)
	require.Equal(t, int64(200), creators[0].LastModifyTs)

	firstHotItem := testWeiboHotSearchItem("wb_hot_item_db", func(item *HotSearchItem) {
		item.Title = "first hot"
		item.AddTs = 100
		item.LastModifyTs = 100
	})
	require.NoError(t, store.SaveHotSearchItem(ctx, firstHotItem))

	secondHotItem := testWeiboHotSearchItem("wb_hot_item_db", func(item *HotSearchItem) {
		item.Title = "second hot"
		item.AddTs = 200
		item.LastModifyTs = 200
	})
	require.NoError(t, store.SaveHotSearchItem(ctx, secondHotItem))

	var hotItems []HotSearchItem
	require.NoError(t, db.Where("hot_item_id = ?", "wb_hot_item_db").Find(&hotItems).Error)
	require.Len(t, hotItems, 1)
	require.Equal(t, "second hot", hotItems[0].Title)
	require.Equal(t, int64(100), hotItems[0].AddTs)
	require.Equal(t, int64(200), hotItems[0].LastModifyTs)

	firstRelation := testWeiboHotSearchRelation("wb_hot_rel_db", func(relation *HotSearchContentRelation) {
		relation.RankNo = 1
		relation.AddTs = 100
		relation.LastModifyTs = 100
	})
	require.NoError(t, store.SaveHotSearchRelation(ctx, firstRelation))

	secondRelation := testWeiboHotSearchRelation("wb_hot_rel_db", func(relation *HotSearchContentRelation) {
		relation.RankNo = 2
		relation.AddTs = 200
		relation.LastModifyTs = 200
	})
	require.NoError(t, store.SaveHotSearchRelation(ctx, secondRelation))

	var relations []HotSearchContentRelation
	require.NoError(t, db.Where("rel_id = ?", "wb_hot_rel_db").Find(&relations).Error)
	require.Len(t, relations, 1)
	require.Equal(t, 2, relations[0].RankNo)
	require.Equal(t, int64(100), relations[0].AddTs)
	require.Equal(t, int64(200), relations[0].LastModifyTs)
}

func TestWeiboSQLiteStorePreservesOriginalAddTsLikePython(t *testing.T) {
	db := newWeiboTestDB(t)
	store := NewSQLiteDBStore(db)
	require.NoError(t, store.AutoMigrate())
	ctx := context.Background()

	firstNote := testWeiboNote("wb_note_sqlite", func(note *Note) {
		note.Content = "first note"
		note.AddTs = 100
		note.LastModifyTs = 100
	})
	require.NoError(t, store.SaveNote(ctx, firstNote))

	secondNote := testWeiboNote("wb_note_sqlite", func(note *Note) {
		note.Content = "second note"
		note.AddTs = 200
		note.LastModifyTs = 200
	})
	require.NoError(t, store.SaveNote(ctx, secondNote))

	var note Note
	require.NoError(t, db.Where("note_id = ?", "wb_note_sqlite").First(&note).Error)
	require.Equal(t, "second note", note.Content)
	require.Equal(t, int64(100), note.AddTs)
	require.Equal(t, int64(200), note.LastModifyTs)

	firstComment := testWeiboComment("wb_comment_sqlite", func(comment *Comment) {
		comment.Content = "first comment"
		comment.AddTs = 100
		comment.LastModifyTs = 100
	})
	require.NoError(t, store.SaveComment(ctx, firstComment))

	secondComment := testWeiboComment("wb_comment_sqlite", func(comment *Comment) {
		comment.Content = "second comment"
		comment.AddTs = 200
		comment.LastModifyTs = 200
	})
	require.NoError(t, store.SaveComment(ctx, secondComment))

	var comment Comment
	require.NoError(t, db.Where("comment_id = ?", "wb_comment_sqlite").First(&comment).Error)
	require.Equal(t, "second comment", comment.Content)
	require.Equal(t, int64(100), comment.AddTs)
	require.Equal(t, int64(200), comment.LastModifyTs)

	firstCreator := testWeiboCreator("wb_creator_sqlite", func(creator *Creator) {
		creator.Nickname = "first creator"
		creator.AddTs = 100
		creator.LastModifyTs = 100
	})
	require.NoError(t, store.SaveCreator(ctx, firstCreator))

	secondCreator := testWeiboCreator("wb_creator_sqlite", func(creator *Creator) {
		creator.Nickname = "second creator"
		creator.AddTs = 200
		creator.LastModifyTs = 200
	})
	require.NoError(t, store.SaveCreator(ctx, secondCreator))

	var creator Creator
	require.NoError(t, db.Where("user_id = ?", "wb_creator_sqlite").First(&creator).Error)
	require.Equal(t, "second creator", creator.Nickname)
	require.Equal(t, int64(100), creator.AddTs)
	require.Equal(t, int64(200), creator.LastModifyTs)
}

func newWeiboTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Note{}, &Comment{}, &Creator{}, &HotSearchItem{}, &HotSearchContentRelation{}))
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_weibo_note_note_id_f95b1a ON weibo_note(note_id);").Error)
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_weibo_note__comment_c7611c ON weibo_note_comment(comment_id);").Error)
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_weibo_creator_user_id ON weibo_creator(user_id);").Error)
	return db
}

func testWeiboNote(noteID string, mutate func(*Note)) *Note {
	note := &Note{
		UserID:         "wb_user",
		Nickname:       "wb author",
		Avatar:         "https://example.test/wb-avatar.png",
		Gender:         "m",
		ProfileURL:     "https://weibo.com/u/wb_user",
		IPLocation:     "北京",
		AddTs:          100,
		LastModifyTs:   100,
		NoteID:         noteID,
		Content:        "wb content",
		CreateTime:     1700000000,
		CreateDateTime: "2023-11-14 22:13:20",
		LikedCount:     "10",
		CommentsCount:  "2",
		SharedCount:    "1",
		NoteURL:        "https://weibo.com/123/" + noteID,
		ImageList:      "[]",
		VideoURL:       "",
		SourceKeyword:  "mock",
	}
	if mutate != nil {
		mutate(note)
	}
	return note
}

func testWeiboComment(commentID string, mutate func(*Comment)) *Comment {
	comment := &Comment{
		UserID:          "wb_comment_user",
		Nickname:        "wb commenter",
		Avatar:          "https://example.test/wb-commenter.png",
		Gender:          "f",
		ProfileURL:      "https://weibo.com/u/wb_comment_user",
		IPLocation:      "上海",
		AddTs:           100,
		LastModifyTs:    100,
		CommentID:       commentID,
		NoteID:          "wb_note_db",
		Content:         "wb comment",
		CreateTime:      1700000010,
		CreateDateTime:  "2023-11-14 22:13:30",
		SubCommentCount: "0",
		LikeCount:       "1",
	}
	if mutate != nil {
		mutate(comment)
	}
	return comment
}

func testWeiboCreator(userID string, mutate func(*Creator)) *Creator {
	creator := &Creator{
		UserID:       userID,
		Nickname:     "wb creator",
		Avatar:       "https://example.test/wb-creator.png",
		IPLocation:   "广州",
		AddTs:        100,
		LastModifyTs: 100,
		Desc:         "wb creator desc",
		Gender:       "m",
		Follows:      "2",
		Fans:         "10",
		TagList:      "tech",
	}
	if mutate != nil {
		mutate(creator)
	}
	return creator
}

func testWeiboHotSearchItem(hotItemID string, mutate func(*HotSearchItem)) *HotSearchItem {
	item := &HotSearchItem{
		HotItemID:    hotItemID,
		Platform:     "wb",
		BoardName:    "hot_search",
		SnapshotDate: "2026-06-21",
		SnapshotTs:   1782000000,
		RankNo:       1,
		Title:        "wb hot",
		Keyword:      "wb hot",
		HotValue:     "100",
		AddTs:        100,
		LastModifyTs: 100,
	}
	if mutate != nil {
		mutate(item)
	}
	return item
}

func testWeiboHotSearchRelation(relID string, mutate func(*HotSearchContentRelation)) *HotSearchContentRelation {
	relation := &HotSearchContentRelation{
		RelID:         relID,
		HotItemID:     "wb_hot_item_db",
		Platform:      "wb",
		ContentID:     "wb_note_db",
		ContentType:   "note",
		ContentTable:  "weibo_note",
		RankNo:        1,
		SourceKeyword: "wb hot",
		AddTs:         100,
		LastModifyTs:  100,
	}
	if mutate != nil {
		mutate(relation)
	}
	return relation
}
