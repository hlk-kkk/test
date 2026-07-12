// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/xhs/mock_store_test.go
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

// internal/xhs/mock_store_test.go
package xhs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockStore_SaveNote 测试 MockStore 的 SaveNote 功能
func TestMockStore_SaveNote(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	now := time.Now().Unix()

	t.Run("正常保存笔记", func(t *testing.T) {
		note := &Note{
			UserID:         "user123",
			NoteID:         "note123",
			Title:          "测试笔记",
			AddTs:          now,
			LastModifyTs:   now,
			Time:           now,
			LastUpdateTime: now,
		}

		err := store.SaveNote(ctx, note)
		assert.NoError(t, err)

		// 验证保存成功
		saved := store.GetNote("note123")
		require.NotNil(t, saved)
		assert.Equal(t, "测试笔记", saved.Title)
		assert.Equal(t, 1, store.GetSaveNoteCalls())
	})

	t.Run("注入错误", func(t *testing.T) {
		store.Reset()
		store.SetSaveNoteError(ErrMockSaveFailed)

		note := &Note{
			NoteID: "note_error",
			Title:  "错误笔记",
		}

		err := store.SaveNote(ctx, note)
		assert.ErrorIs(t, err, ErrMockSaveFailed)

		// 验证没有保存
		saved := store.GetNote("note_error")
		assert.Nil(t, saved)
		assert.Equal(t, 1, store.GetSaveNoteCalls())
	})

	t.Run("更新已存在的笔记", func(t *testing.T) {
		store.Reset()

		note := &Note{
			NoteID: "note_update",
			Title:  "原始标题",
		}
		store.SaveNote(ctx, note)

		// 更新笔记
		note.Title = "更新后的标题"
		err := store.SaveNote(ctx, note)
		assert.NoError(t, err)

		// 验证更新成功
		saved := store.GetNote("note_update")
		assert.Equal(t, "更新后的标题", saved.Title)
		assert.Equal(t, 2, store.GetSaveNoteCalls())
	})
}

// TestMockStore_NoteExists 测试 MockStore 的 NoteExists 功能
func TestMockStore_NoteExists(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	t.Run("笔记不存在", func(t *testing.T) {
		exists, err := store.NoteExists(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.False(t, exists)
		assert.Equal(t, 1, store.GetNoteExistsCalls())
	})

	t.Run("笔记存在", func(t *testing.T) {
		store.Reset()

		note := &Note{
			NoteID: "note_exists",
			Title:  "存在的笔记",
		}
		store.SaveNote(ctx, note)

		exists, err := store.NoteExists(ctx, "note_exists")
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, 1, store.GetNoteExistsCalls())
	})

	t.Run("注入错误", func(t *testing.T) {
		store.Reset()
		store.SetNoteExistsError(ErrMockQueryFailed)

		exists, err := store.NoteExists(ctx, "any_note")
		assert.ErrorIs(t, err, ErrMockQueryFailed)
		assert.False(t, exists)
		assert.Equal(t, 1, store.GetNoteExistsCalls())
	})
}

// TestMockStore_SaveComment 测试 MockStore 的 SaveComment 功能
func TestMockStore_SaveComment(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	now := time.Now().Unix()

	t.Run("正常保存评论", func(t *testing.T) {
		comment := &NoteComment{
			UserID:          "user123",
			CommentID:       "comment123",
			NoteID:          "note123",
			Content:         "测试评论",
			AddTs:           now,
			LastModifyTs:    now,
			CreateTime:      now,
			SubCommentCount: "0",
			LikeCount:       "0",
			NoteURL:         "https://www.xiaohongshu.com/note123",
		}

		err := store.SaveComment(ctx, comment)
		assert.NoError(t, err)

		// 验证保存成功
		saved := store.GetComment("comment123")
		require.NotNil(t, saved)
		assert.Equal(t, "测试评论", saved.Content)
		assert.Equal(t, 1, store.GetSaveCommentCalls())
	})

	t.Run("注入错误", func(t *testing.T) {
		store.Reset()
		store.SetSaveCommentError(ErrMockSaveFailed)

		comment := &NoteComment{
			CommentID: "comment_error",
			Content:   "错误评论",
		}

		err := store.SaveComment(ctx, comment)
		assert.ErrorIs(t, err, ErrMockSaveFailed)

		// 验证没有保存
		saved := store.GetComment("comment_error")
		assert.Nil(t, saved)
		assert.Equal(t, 1, store.GetSaveCommentCalls())
	})
}

// TestMockStore_SaveCreator 测试 MockStore 的 SaveCreator 功能
func TestMockStore_SaveCreator(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	now := time.Now().Unix()

	t.Run("正常保存创作者", func(t *testing.T) {
		creator := &Creator{
			UserID:       "creator123",
			Nickname:     "创作者123",
			Avatar:       "https://example.com/avatar.jpg",
			AddTs:        now,
			LastModifyTs: now,
			Follows:      "100",
			Fans:         "10000",
			Interaction:  "50000",
		}

		err := store.SaveCreator(ctx, creator)
		assert.NoError(t, err)

		// 验证保存成功
		saved := store.GetCreator("creator123")
		require.NotNil(t, saved)
		assert.Equal(t, "创作者123", saved.Nickname)
		assert.Equal(t, 1, store.GetSaveCreatorCalls())
	})

	t.Run("注入错误", func(t *testing.T) {
		store.Reset()
		store.SetSaveCreatorError(ErrMockSaveFailed)

		creator := &Creator{
			UserID:   "creator_error",
			Nickname: "错误创作者",
		}

		err := store.SaveCreator(ctx, creator)
		assert.ErrorIs(t, err, ErrMockSaveFailed)

		// 验证没有保存
		saved := store.GetCreator("creator_error")
		assert.Nil(t, saved)
		assert.Equal(t, 1, store.GetSaveCreatorCalls())
	})
}

// TestMockStore_ConcurrentAccess 测试 MockStore 的并发访问安全性
func TestMockStore_ConcurrentAccess(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	now := time.Now().Unix()

	// 并发写入笔记
	t.Run("并发写入笔记", func(t *testing.T) {
		store.Reset()

		const goroutines = 10
		done := make(chan bool, goroutines)

		for i := 0; i < goroutines; i++ {
			go func(id int) {
				note := &Note{
					UserID:         "concurrent_user",
					NoteID:         "concurrent_note_" + string(rune('0'+id)),
					Title:          "并发笔记",
					AddTs:          now,
					LastModifyTs:   now,
					Time:           now,
					LastUpdateTime: now,
				}
				store.SaveNote(ctx, note)
				done <- true
			}(i)
		}

		// 等待所有 goroutine 完成
		for i := 0; i < goroutines; i++ {
			<-done
		}

		assert.Equal(t, goroutines, store.GetSaveNoteCalls())
	})

	// 并发读写
	t.Run("并发读写", func(t *testing.T) {
		store.Reset()

		note := &Note{
			NoteID: "read_write_note",
			Title:  "读写笔记",
		}
		store.SaveNote(ctx, note)

		const goroutines = 20
		done := make(chan bool, goroutines)

		// 10个读,10个写
		for i := 0; i < goroutines/2; i++ {
			// 读操作
			go func() {
				store.NoteExists(ctx, "read_write_note")
				done <- true
			}()

			// 写操作
			go func() {
				note := &Note{
					NoteID: "read_write_note",
					Title:  "更新的笔记",
				}
				store.SaveNote(ctx, note)
				done <- true
			}()
		}

		// 等待所有 goroutine 完成
		for i := 0; i < goroutines; i++ {
			<-done
		}

		// 验证所有操作都执行了
		assert.Equal(t, goroutines/2, store.GetNoteExistsCalls())
		assert.True(t, store.GetSaveNoteCalls() >= goroutines/2)
	})
}

// TestMockStore_Reset 测试 MockStore 的 Reset 功能
func TestMockStore_Reset(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// 添加一些数据
	note := &Note{NoteID: "note1", Title: "笔记1"}
	store.SaveNote(ctx, note)

	comment := &NoteComment{CommentID: "comment1", Content: "评论1"}
	store.SaveComment(ctx, comment)

	creator := &Creator{UserID: "creator1", Nickname: "创作者1"}
	store.SaveCreator(ctx, creator)

	// 设置错误
	store.SetSaveNoteError(ErrMockSaveFailed)

	// 重置
	store.Reset()

	// 验证所有数据和状态都被清空
	assert.Nil(t, store.GetNote("note1"))
	assert.Nil(t, store.GetComment("comment1"))
	assert.Nil(t, store.GetCreator("creator1"))

	assert.Equal(t, 0, store.GetSaveNoteCalls())
	assert.Equal(t, 0, store.GetSaveCommentCalls())
	assert.Equal(t, 0, store.GetSaveCreatorCalls())
	assert.Equal(t, 0, store.GetNoteExistsCalls())

	// 验证错误注入也被清除
	err := store.SaveNote(ctx, &Note{NoteID: "new_note"})
	assert.NoError(t, err)
}
