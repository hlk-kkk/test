// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/xhs/store_test.go
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

// internal/xhs/store_test.go
package xhs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "Failed to create in-memory database")

	// 自动迁移表结构
	err = db.AutoMigrate(&Note{}, &NoteComment{}, &Creator{})
	require.NoError(t, err, "Failed to migrate tables")

	// 为 SQLite 添加唯一约束
	// Note 表需要 note_id 唯一
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_xhs_note_note_id ON xhs_note(note_id);")
	// Comment 表需要 comment_id 唯一
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_xhs_note_comment_comment_id ON xhs_note_comment(comment_id);")
	// Creator 表需要 user_id 唯一
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_xhs_creator_user_id ON xhs_creator(user_id);")

	return db
}

// TestDBStore_SaveNote 测试保存笔记
func TestDBStore_SaveNote(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLiteDBStore(db) // 使用 SQLite 兼容的 Store
	ctx := context.Background()

	now := time.Now().Unix()

	t.Run("创建新笔记", func(t *testing.T) {
		note := &Note{
			UserID:         "user123",
			Nickname:       "测试用户",
			Avatar:         "https://example.com/avatar.jpg",
			IPLocation:     "上海",
			AddTs:          now,
			LastModifyTs:   now,
			NoteID:         "note123",
			Type:           "normal",
			Title:          "测试笔记",
			Desc:           "这是一个测试笔记",
			Time:           now,
			LastUpdateTime: now,
			LikedCount:     "100",
			CollectedCount: "50",
			CommentCount:   "20",
			ShareCount:     "10",
			ImageList:      `["img1.jpg", "img2.jpg"]`,
			TagList:        `["标签1", "标签2"]`,
			NoteURL:        "https://www.xiaohongshu.com/note123",
			SourceKeyword:  "测试关键词",
		}

		err := store.SaveNote(ctx, note)
		assert.NoError(t, err, "SaveNote should not return error")

		// 验证笔记已保存
		exists, err := store.NoteExists(ctx, "note123")
		assert.NoError(t, err)
		assert.True(t, exists, "Note should exist after saving")
	})

	t.Run("更新已存在的笔记", func(t *testing.T) {
		// 先创建一个笔记
		note := &Note{
			UserID:         "user456",
			Nickname:       "用户456",
			NoteID:         "note456",
			Title:          "原始标题",
			Desc:           "原始描述",
			AddTs:          now,
			LastModifyTs:   now,
			Time:           now,
			LastUpdateTime: now,
		}
		err := store.SaveNote(ctx, note)
		require.NoError(t, err)

		// 更新笔记
		note.Title = "更新后的标题"
		note.Desc = "更新后的描述"
		note.LikedCount = "200"
		err = store.SaveNote(ctx, note)
		assert.NoError(t, err)

		// 验证更新成功
		var updated Note
		err = db.Where("note_id = ?", "note456").First(&updated).Error
		require.NoError(t, err)
		assert.Equal(t, "更新后的标题", updated.Title)
		assert.Equal(t, "更新后的描述", updated.Desc)
		assert.Equal(t, "200", updated.LikedCount)
	})

	t.Run("去重逻辑 - 相同 note_id", func(t *testing.T) {
		note1 := &Note{
			UserID:         "user789",
			NoteID:         "note789",
			Title:          "第一次保存",
			AddTs:          now,
			LastModifyTs:   now,
			Time:           now,
			LastUpdateTime: now,
		}
		err := store.SaveNote(ctx, note1)
		require.NoError(t, err)

		note2 := &Note{
			UserID:         "user789",
			NoteID:         "note789", // 相同的 note_id
			Title:          "第二次保存",
			AddTs:          now,
			LastModifyTs:   now,
			Time:           now,
			LastUpdateTime: now,
		}
		err = store.SaveNote(ctx, note2)
		assert.NoError(t, err)

		// 验证只有一条记录
		var count int64
		err = db.Model(&Note{}).Where("note_id = ?", "note789").Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "Should have only one record")

		// 验证标题已更新
		var updated Note
		err = db.Where("note_id = ?", "note789").First(&updated).Error
		require.NoError(t, err)
		assert.Equal(t, "第二次保存", updated.Title)
	})
}

func TestDBStore_SaveNoteMatchesPythonQueryUpdateWithoutUniqueIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Note{}))
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_xhs_note_note_id_209457 ON xhs_note(note_id);").Error)

	store := NewDBStore(db)
	ctx := context.Background()
	first := &Note{
		UserID:         "user_db",
		NoteID:         "note_db",
		Title:          "first title",
		AddTs:          100,
		LastModifyTs:   100,
		Time:           100,
		LastUpdateTime: 100,
	}
	require.NoError(t, store.SaveNote(ctx, first))

	second := &Note{
		UserID:         "user_db",
		NoteID:         "note_db",
		Title:          "second title",
		AddTs:          200,
		LastModifyTs:   200,
		Time:           200,
		LastUpdateTime: 200,
	}
	require.NoError(t, store.SaveNote(ctx, second))

	var rows []Note
	require.NoError(t, db.Where("note_id = ?", "note_db").Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "second title", rows[0].Title)
	require.Equal(t, int64(100), rows[0].AddTs, "DB updates should preserve the original add_ts like Python")
	require.Equal(t, int64(200), rows[0].LastModifyTs)
}

func TestDBStore_SaveCommentMatchesPythonQueryUpdateWithoutUniqueIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&NoteComment{}))
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_xhs_note_co_comment_8e8349 ON xhs_note_comment(comment_id);").Error)

	store := NewDBStore(db)
	ctx := context.Background()
	first := &NoteComment{
		UserID:          "comment_user_db",
		CommentID:       "comment_db",
		NoteID:          "note_db",
		Content:         "first comment",
		AddTs:           100,
		LastModifyTs:    100,
		CreateTime:      100,
		SubCommentCount: "0",
		LikeCount:       "1",
		NoteURL:         "https://example.test/note_db",
	}
	require.NoError(t, store.SaveComment(ctx, first))

	second := &NoteComment{
		UserID:          "comment_user_db",
		CommentID:       "comment_db",
		NoteID:          "note_db",
		Content:         "second comment",
		AddTs:           200,
		LastModifyTs:    200,
		CreateTime:      200,
		SubCommentCount: "0",
		LikeCount:       "2",
		NoteURL:         "https://example.test/note_db",
	}
	require.NoError(t, store.SaveComment(ctx, second))

	var rows []NoteComment
	require.NoError(t, db.Where("comment_id = ?", "comment_db").Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "second comment", rows[0].Content)
	require.Equal(t, int64(100), rows[0].AddTs, "DB updates should preserve the original add_ts like Python")
	require.Equal(t, int64(200), rows[0].LastModifyTs)
	require.Equal(t, "2", rows[0].LikeCount)
}

func TestDBStore_SaveCreatorMatchesPythonQueryUpdateWithoutUniqueIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Creator{}))
	require.NoError(t, db.Exec("CREATE INDEX IF NOT EXISTS idx_xhs_creator_user_id ON xhs_creator(user_id);").Error)

	store := NewDBStore(db)
	ctx := context.Background()
	first := &Creator{
		UserID:       "creator_db",
		Nickname:     "first creator",
		AddTs:        100,
		LastModifyTs: 100,
		Fans:         "10",
	}
	require.NoError(t, store.SaveCreator(ctx, first))

	second := &Creator{
		UserID:       "creator_db",
		Nickname:     "second creator",
		AddTs:        200,
		LastModifyTs: 200,
		Fans:         "20",
	}
	require.NoError(t, store.SaveCreator(ctx, second))

	var rows []Creator
	require.NoError(t, db.Where("user_id = ?", "creator_db").Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "second creator", rows[0].Nickname)
	require.Equal(t, int64(100), rows[0].AddTs, "DB updates should preserve the original add_ts like Python")
	require.Equal(t, int64(200), rows[0].LastModifyTs)
	require.Equal(t, "20", rows[0].Fans)
}

func TestSQLiteDBStorePreservesOriginalAddTsLikePython(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	store := NewSQLiteDBStore(db)
	require.NoError(t, store.AutoMigrate())
	ctx := context.Background()
	builder := NewTestDataBuilder()

	firstNote := builder.BuildNote("note_sqlite_add_ts", func(note *Note) {
		note.Title = "first note"
		note.AddTs = 100
		note.LastModifyTs = 100
	})
	require.NoError(t, store.SaveNote(ctx, firstNote))

	secondNote := builder.BuildNote("note_sqlite_add_ts", func(note *Note) {
		note.Title = "second note"
		note.AddTs = 200
		note.LastModifyTs = 200
	})
	require.NoError(t, store.SaveNote(ctx, secondNote))

	var note Note
	require.NoError(t, db.Where("note_id = ?", "note_sqlite_add_ts").First(&note).Error)
	require.Equal(t, "second note", note.Title)
	require.Equal(t, int64(100), note.AddTs)
	require.Equal(t, int64(200), note.LastModifyTs)

	firstComment := builder.BuildComment("comment_sqlite_add_ts", "note_sqlite_add_ts", func(comment *NoteComment) {
		comment.Content = "first comment"
		comment.AddTs = 100
		comment.LastModifyTs = 100
	})
	require.NoError(t, store.SaveComment(ctx, firstComment))

	secondComment := builder.BuildComment("comment_sqlite_add_ts", "note_sqlite_add_ts", func(comment *NoteComment) {
		comment.Content = "second comment"
		comment.AddTs = 200
		comment.LastModifyTs = 200
	})
	require.NoError(t, store.SaveComment(ctx, secondComment))

	var comment NoteComment
	require.NoError(t, db.Where("comment_id = ?", "comment_sqlite_add_ts").First(&comment).Error)
	require.Equal(t, "second comment", comment.Content)
	require.Equal(t, int64(100), comment.AddTs)
	require.Equal(t, int64(200), comment.LastModifyTs)

	firstCreator := builder.BuildCreator("creator_sqlite_add_ts", func(creator *Creator) {
		creator.Nickname = "first creator"
		creator.AddTs = 100
		creator.LastModifyTs = 100
	})
	require.NoError(t, store.SaveCreator(ctx, firstCreator))

	secondCreator := builder.BuildCreator("creator_sqlite_add_ts", func(creator *Creator) {
		creator.Nickname = "second creator"
		creator.AddTs = 200
		creator.LastModifyTs = 200
	})
	require.NoError(t, store.SaveCreator(ctx, secondCreator))

	var creator Creator
	require.NoError(t, db.Where("user_id = ?", "creator_sqlite_add_ts").First(&creator).Error)
	require.Equal(t, "second creator", creator.Nickname)
	require.Equal(t, int64(100), creator.AddTs)
	require.Equal(t, int64(200), creator.LastModifyTs)
}

// TestDBStore_NoteExists 测试笔记存在性检查
func TestDBStore_NoteExists(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLiteDBStore(db) // 使用 SQLite 兼容的 Store
	ctx := context.Background()

	now := time.Now().Unix()

	t.Run("笔记不存在", func(t *testing.T) {
		exists, err := store.NoteExists(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.False(t, exists, "Non-existent note should return false")
	})

	t.Run("笔记存在", func(t *testing.T) {
		note := &Note{
			UserID:         "user_exists",
			NoteID:         "note_exists",
			Title:          "存在的笔记",
			AddTs:          now,
			LastModifyTs:   now,
			Time:           now,
			LastUpdateTime: now,
		}
		err := store.SaveNote(ctx, note)
		require.NoError(t, err)

		exists, err := store.NoteExists(ctx, "note_exists")
		assert.NoError(t, err)
		assert.True(t, exists, "Existing note should return true")
	})
}

// TestDBStore_SaveComment 测试保存评论
func TestDBStore_SaveComment(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLiteDBStore(db) // 使用 SQLite 兼容的 Store
	ctx := context.Background()

	now := time.Now().Unix()

	t.Run("创建新评论", func(t *testing.T) {
		comment := &NoteComment{
			UserID:          "comment_user1",
			Nickname:        "评论用户1",
			Avatar:          "https://example.com/avatar.jpg",
			IPLocation:      "北京",
			AddTs:           now,
			LastModifyTs:    now,
			CommentID:       "comment123",
			CreateTime:      now,
			NoteID:          "note123",
			Content:         "这是一条测试评论",
			SubCommentCount: "5",
			LikeCount:       "10",
			NoteURL:         "https://www.xiaohongshu.com/note123",
		}

		err := store.SaveComment(ctx, comment)
		assert.NoError(t, err, "SaveComment should not return error")

		// 验证评论已保存
		var saved NoteComment
		err = db.Where("comment_id = ?", "comment123").First(&saved).Error
		require.NoError(t, err)
		assert.Equal(t, "这是一条测试评论", saved.Content)
	})

	t.Run("更新已存在的评论", func(t *testing.T) {
		comment := &NoteComment{
			UserID:          "comment_user2",
			CommentID:       "comment456",
			NoteID:          "note456",
			Content:         "原始评论内容",
			AddTs:           now,
			LastModifyTs:    now,
			CreateTime:      now,
			SubCommentCount: "0",
			LikeCount:       "0",
			NoteURL:         "https://www.xiaohongshu.com/note456",
		}
		err := store.SaveComment(ctx, comment)
		require.NoError(t, err)

		// 更新评论
		comment.Content = "更新后的评论内容"
		comment.LikeCount = "20"
		err = store.SaveComment(ctx, comment)
		assert.NoError(t, err)

		// 验证更新成功
		var updated NoteComment
		err = db.Where("comment_id = ?", "comment456").First(&updated).Error
		require.NoError(t, err)
		assert.Equal(t, "更新后的评论内容", updated.Content)
		assert.Equal(t, "20", updated.LikeCount)
	})

	t.Run("去重逻辑 - 相同 comment_id", func(t *testing.T) {
		comment1 := &NoteComment{
			UserID:          "comment_user3",
			CommentID:       "comment789",
			NoteID:          "note789",
			Content:         "第一条评论",
			AddTs:           now,
			LastModifyTs:    now,
			CreateTime:      now,
			SubCommentCount: "0",
			LikeCount:       "0",
			NoteURL:         "https://www.xiaohongshu.com/note789",
		}
		err := store.SaveComment(ctx, comment1)
		require.NoError(t, err)

		comment2 := &NoteComment{
			UserID:          "comment_user3",
			CommentID:       "comment789", // 相同的 comment_id
			NoteID:          "note789",
			Content:         "第二条评论",
			AddTs:           now,
			LastModifyTs:    now,
			CreateTime:      now,
			SubCommentCount: "0",
			LikeCount:       "0",
			NoteURL:         "https://www.xiaohongshu.com/note789",
		}
		err = store.SaveComment(ctx, comment2)
		assert.NoError(t, err)

		// 验证只有一条记录
		var count int64
		err = db.Model(&NoteComment{}).Where("comment_id = ?", "comment789").Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "Should have only one record")

		// 验证内容已更新
		var updated NoteComment
		err = db.Where("comment_id = ?", "comment789").First(&updated).Error
		require.NoError(t, err)
		assert.Equal(t, "第二条评论", updated.Content)
	})
}

// TestDBStore_SaveCreator 测试保存创作者
func TestDBStore_SaveCreator(t *testing.T) {
	db := setupTestDB(t)
	store := NewSQLiteDBStore(db) // 使用 SQLite 兼容的 Store
	ctx := context.Background()

	now := time.Now().Unix()

	t.Run("创建新创作者", func(t *testing.T) {
		creator := &Creator{
			UserID:       "creator123",
			Nickname:     "创作者123",
			Avatar:       "https://example.com/creator.jpg",
			IPLocation:   "广州",
			AddTs:        now,
			LastModifyTs: now,
			Desc:         "这是一个测试创作者",
			Gender:       "女",
			Follows:      "100",
			Fans:         "10000",
			Interaction:  "50000",
			TagList:      `["美妆", "时尚"]`,
		}

		err := store.SaveCreator(ctx, creator)
		assert.NoError(t, err, "SaveCreator should not return error")

		// 验证创作者已保存
		var saved Creator
		err = db.Where("user_id = ?", "creator123").First(&saved).Error
		require.NoError(t, err)
		assert.Equal(t, "创作者123", saved.Nickname)
	})

	t.Run("更新已存在的创作者", func(t *testing.T) {
		creator := &Creator{
			UserID:       "creator456",
			Nickname:     "原始昵称",
			AddTs:        now,
			LastModifyTs: now,
			Follows:      "50",
			Fans:         "5000",
			Interaction:  "25000",
		}
		err := store.SaveCreator(ctx, creator)
		require.NoError(t, err)

		// 更新创作者
		creator.Nickname = "更新后的昵称"
		creator.Fans = "15000"
		err = store.SaveCreator(ctx, creator)
		assert.NoError(t, err)

		// 验证更新成功
		var updated Creator
		err = db.Where("user_id = ?", "creator456").First(&updated).Error
		require.NoError(t, err)
		assert.Equal(t, "更新后的昵称", updated.Nickname)
		assert.Equal(t, "15000", updated.Fans)
	})

	t.Run("去重逻辑 - 相同 user_id", func(t *testing.T) {
		creator1 := &Creator{
			UserID:       "creator789",
			Nickname:     "第一次保存",
			AddTs:        now,
			LastModifyTs: now,
			Follows:      "10",
			Fans:         "100",
			Interaction:  "500",
		}
		err := store.SaveCreator(ctx, creator1)
		require.NoError(t, err)

		creator2 := &Creator{
			UserID:       "creator789", // 相同的 user_id
			Nickname:     "第二次保存",
			AddTs:        now,
			LastModifyTs: now,
			Follows:      "20",
			Fans:         "200",
			Interaction:  "1000",
		}
		err = store.SaveCreator(ctx, creator2)
		assert.NoError(t, err)

		// 验证只有一条记录
		var count int64
		err = db.Model(&Creator{}).Where("user_id = ?", "creator789").Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "Should have only one record")

		// 验证昵称已更新
		var updated Creator
		err = db.Where("user_id = ?", "creator789").First(&updated).Error
		require.NoError(t, err)
		assert.Equal(t, "第二次保存", updated.Nickname)
	})
}

// BenchmarkDBStore_SaveNote 性能测试 - 保存笔记
func BenchmarkDBStore_SaveNote(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&Note{})
	store := NewDBStore(db)
	ctx := context.Background()

	now := time.Now().Unix()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		note := &Note{
			UserID:         "bench_user",
			NoteID:         "bench_note",
			Title:          "Benchmark Note",
			AddTs:          now,
			LastModifyTs:   now,
			Time:           now,
			LastUpdateTime: now,
		}
		store.SaveNote(ctx, note)
	}
}

// BenchmarkDBStore_NoteExists 性能测试 - 检查笔记存在性
func BenchmarkDBStore_NoteExists(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&Note{})
	store := NewDBStore(db)
	ctx := context.Background()

	now := time.Now().Unix()

	// 先保存一条笔记
	note := &Note{
		UserID:         "bench_user",
		NoteID:         "bench_note",
		Title:          "Benchmark Note",
		AddTs:          now,
		LastModifyTs:   now,
		Time:           now,
		LastUpdateTime: now,
	}
	store.SaveNote(ctx, note)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.NoteExists(ctx, "bench_note")
	}
}
