// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/tieba/store_test.go
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

package tieba

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTiebaDBStoreMatchesPythonQueryUpdateWithoutUniqueIndex(t *testing.T) {
	db := newTiebaTestDB(t)
	store := NewDBStore(db)
	ctx := context.Background()

	firstNote := testTiebaNote("tb_note_db", func(note *Note) {
		note.Title = "first note"
		note.AddTs = 100
		note.LastModifyTs = 100
	})
	require.NoError(t, store.SaveNote(ctx, firstNote))

	secondNote := testTiebaNote("tb_note_db", func(note *Note) {
		note.Title = "second note"
		note.AddTs = 200
		note.LastModifyTs = 200
	})
	require.NoError(t, store.SaveNote(ctx, secondNote))

	var notes []Note
	require.NoError(t, db.Where("note_id = ?", "tb_note_db").Find(&notes).Error)
	require.Len(t, notes, 1)
	require.Equal(t, "second note", notes[0].Title)
	require.Equal(t, int64(100), notes[0].AddTs)
	require.Equal(t, int64(200), notes[0].LastModifyTs)

	firstComment := testTiebaComment("tb_comment_db", func(comment *Comment) {
		comment.Content = "first comment"
		comment.AddTs = 100
		comment.LastModifyTs = 100
	})
	require.NoError(t, store.SaveComment(ctx, firstComment))

	secondComment := testTiebaComment("tb_comment_db", func(comment *Comment) {
		comment.Content = "second comment"
		comment.AddTs = 200
		comment.LastModifyTs = 200
	})
	require.NoError(t, store.SaveComment(ctx, secondComment))

	var comments []Comment
	require.NoError(t, db.Where("comment_id = ?", "tb_comment_db").Find(&comments).Error)
	require.Len(t, comments, 1)
	require.Equal(t, "second comment", comments[0].Content)
	require.Equal(t, int64(100), comments[0].AddTs)
	require.Equal(t, int64(200), comments[0].LastModifyTs)

	firstCreator := testTiebaCreator("tb_creator_db", func(creator *Creator) {
		creator.Nickname = "first creator"
		creator.AddTs = 100
		creator.LastModifyTs = 100
	})
	require.NoError(t, store.SaveCreator(ctx, firstCreator))

	secondCreator := testTiebaCreator("tb_creator_db", func(creator *Creator) {
		creator.Nickname = "second creator"
		creator.AddTs = 200
		creator.LastModifyTs = 200
	})
	require.NoError(t, store.SaveCreator(ctx, secondCreator))

	var creators []Creator
	require.NoError(t, db.Where("user_id = ?", "tb_creator_db").Find(&creators).Error)
	require.Len(t, creators, 1)
	require.Equal(t, "second creator", creators[0].Nickname)
	require.Equal(t, int64(100), creators[0].AddTs)
	require.Equal(t, int64(200), creators[0].LastModifyTs)
}

func TestTiebaSQLiteStorePreservesOriginalAddTsLikePython(t *testing.T) {
	db := newTiebaTestDB(t)
	store := NewSQLiteDBStore(db)
	require.NoError(t, store.AutoMigrate())
	ctx := context.Background()

	firstNote := testTiebaNote("tb_note_sqlite", func(note *Note) {
		note.Title = "first note"
		note.AddTs = 100
		note.LastModifyTs = 100
	})
	require.NoError(t, store.SaveNote(ctx, firstNote))

	secondNote := testTiebaNote("tb_note_sqlite", func(note *Note) {
		note.Title = "second note"
		note.AddTs = 200
		note.LastModifyTs = 200
	})
	require.NoError(t, store.SaveNote(ctx, secondNote))

	var note Note
	require.NoError(t, db.Where("note_id = ?", "tb_note_sqlite").First(&note).Error)
	require.Equal(t, "second note", note.Title)
	require.Equal(t, int64(100), note.AddTs)
	require.Equal(t, int64(200), note.LastModifyTs)

	firstComment := testTiebaComment("tb_comment_sqlite", func(comment *Comment) {
		comment.Content = "first comment"
		comment.AddTs = 100
		comment.LastModifyTs = 100
	})
	require.NoError(t, store.SaveComment(ctx, firstComment))

	secondComment := testTiebaComment("tb_comment_sqlite", func(comment *Comment) {
		comment.Content = "second comment"
		comment.AddTs = 200
		comment.LastModifyTs = 200
	})
	require.NoError(t, store.SaveComment(ctx, secondComment))

	var comment Comment
	require.NoError(t, db.Where("comment_id = ?", "tb_comment_sqlite").First(&comment).Error)
	require.Equal(t, "second comment", comment.Content)
	require.Equal(t, int64(100), comment.AddTs)
	require.Equal(t, int64(200), comment.LastModifyTs)

	firstCreator := testTiebaCreator("tb_creator_sqlite", func(creator *Creator) {
		creator.Nickname = "first creator"
		creator.AddTs = 100
		creator.LastModifyTs = 100
	})
	require.NoError(t, store.SaveCreator(ctx, firstCreator))

	secondCreator := testTiebaCreator("tb_creator_sqlite", func(creator *Creator) {
		creator.Nickname = "second creator"
		creator.AddTs = 200
		creator.LastModifyTs = 200
	})
	require.NoError(t, store.SaveCreator(ctx, secondCreator))

	var creator Creator
	require.NoError(t, db.Where("user_id = ?", "tb_creator_sqlite").First(&creator).Error)
	require.Equal(t, "second creator", creator.Nickname)
	require.Equal(t, int64(100), creator.AddTs)
	require.Equal(t, int64(200), creator.LastModifyTs)
}

func newTiebaTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Note{}, &Comment{}, &Creator{}))
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_tieba_note_note_id ON tieba_note(note_id);").Error)
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_tieba_comment_comment_id ON tieba_comment(comment_id);").Error)
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_tieba_creator_user_id ON tieba_creator(user_id);").Error)
	return db
}

func testTiebaNote(noteID string, mutate func(*Note)) *Note {
	note := &Note{
		NoteID:          noteID,
		Title:           "tb title",
		Desc:            "tb desc",
		NoteURL:         "https://tieba.baidu.com/p/" + noteID,
		PublishTime:     "2026-06-21 06:00",
		UserLink:        "https://tieba.baidu.com/home/main?id=tb_user",
		UserNickname:    "tb author",
		UserAvatar:      "https://example.test/tb-avatar.png",
		TiebaID:         "tb_forum",
		TiebaName:       "mock吧",
		TiebaLink:       "https://tieba.baidu.com/f?kw=mock",
		TotalReplayNum:  2,
		TotalReplayPage: 1,
		IPLocation:      "北京",
		AddTs:           100,
		LastModifyTs:    100,
		SourceKeyword:   "mock",
	}
	if mutate != nil {
		mutate(note)
	}
	return note
}

func testTiebaComment(commentID string, mutate func(*Comment)) *Comment {
	comment := &Comment{
		CommentID:       commentID,
		Content:         "tb comment",
		UserLink:        "https://tieba.baidu.com/home/main?id=tb_comment_user",
		UserNickname:    "tb commenter",
		UserAvatar:      "https://example.test/tb-commenter.png",
		TiebaID:         "tb_forum",
		TiebaName:       "mock吧",
		TiebaLink:       "https://tieba.baidu.com/f?kw=mock",
		PublishTime:     "2026-06-21 06:01",
		IPLocation:      "上海",
		SubCommentCount: 0,
		NoteID:          "tb_note_db",
		NoteURL:         "https://tieba.baidu.com/p/tb_note_db",
		AddTs:           100,
		LastModifyTs:    100,
	}
	if mutate != nil {
		mutate(comment)
	}
	return comment
}

func testTiebaCreator(userID string, mutate func(*Creator)) *Creator {
	creator := &Creator{
		UserID:               userID,
		UserName:             userID,
		Nickname:             "tb creator",
		Avatar:               "https://example.test/tb-creator.png",
		IPLocation:           "广州",
		AddTs:                100,
		LastModifyTs:         100,
		Gender:               "m",
		Follows:              "2",
		Fans:                 "10",
		RegistrationDuration: "10年",
	}
	if mutate != nil {
		mutate(creator)
	}
	return creator
}
