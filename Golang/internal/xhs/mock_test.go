// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/xhs/mock_test.go
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

// internal/xhs/mock_test.go
package xhs

import (
	"context"
	"errors"
	"sync"
)

// MockStore 是 Store 接口的 Mock 实现,用于测试
type MockStore struct {
	mu sync.RWMutex

	// 存储数据
	notes    map[string]*Note
	comments map[string]*NoteComment
	creators map[string]*Creator

	// 错误注入
	saveNoteError    error
	saveCommentError error
	saveCreatorError error
	noteExistsError  error

	// 调用计数
	saveNoteCalls    int
	saveCommentCalls int
	saveCreatorCalls int
	noteExistsCalls  int
}

// NewMockStore 创建新的 MockStore
func NewMockStore() *MockStore {
	return &MockStore{
		notes:    make(map[string]*Note),
		comments: make(map[string]*NoteComment),
		creators: make(map[string]*Creator),
	}
}

// SaveNote 模拟保存笔记
func (m *MockStore) SaveNote(ctx context.Context, note *Note) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saveNoteCalls++

	if m.saveNoteError != nil {
		return m.saveNoteError
	}

	// 复制一份数据
	noteCopy := *note
	m.notes[note.NoteID] = &noteCopy
	return nil
}

// NoteExists 模拟检查笔记是否存在
func (m *MockStore) NoteExists(ctx context.Context, noteID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.noteExistsCalls++

	if m.noteExistsError != nil {
		return false, m.noteExistsError
	}

	_, exists := m.notes[noteID]
	return exists, nil
}

// SaveComment 模拟保存评论
func (m *MockStore) SaveComment(ctx context.Context, comment *NoteComment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saveCommentCalls++

	if m.saveCommentError != nil {
		return m.saveCommentError
	}

	// 复制一份数据
	commentCopy := *comment
	m.comments[comment.CommentID] = &commentCopy
	return nil
}

// SaveCreator 模拟保存创作者
func (m *MockStore) SaveCreator(ctx context.Context, creator *Creator) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saveCreatorCalls++

	if m.saveCreatorError != nil {
		return m.saveCreatorError
	}

	// 复制一份数据
	creatorCopy := *creator
	m.creators[creator.UserID] = &creatorCopy
	return nil
}

// SetSaveNoteError 设置 SaveNote 错误
func (m *MockStore) SetSaveNoteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveNoteError = err
}

// SetSaveCommentError 设置 SaveComment 错误
func (m *MockStore) SetSaveCommentError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveCommentError = err
}

// SetSaveCreatorError 设置 SaveCreator 错误
func (m *MockStore) SetSaveCreatorError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveCreatorError = err
}

// SetNoteExistsError 设置 NoteExists 错误
func (m *MockStore) SetNoteExistsError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.noteExistsError = err
}

// GetNote 获取笔记(用于测试验证)
func (m *MockStore) GetNote(noteID string) *Note {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.notes[noteID]
}

// GetComment 获取评论(用于测试验证)
func (m *MockStore) GetComment(commentID string) *NoteComment {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.comments[commentID]
}

// GetCreator 获取创作者(用于测试验证)
func (m *MockStore) GetCreator(userID string) *Creator {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.creators[userID]
}

// GetSaveNoteCalls 获取 SaveNote 调用次数
func (m *MockStore) GetSaveNoteCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.saveNoteCalls
}

// GetSaveCommentCalls 获取 SaveComment 调用次数
func (m *MockStore) GetSaveCommentCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.saveCommentCalls
}

// GetSaveCreatorCalls 获取 SaveCreator 调用次数
func (m *MockStore) GetSaveCreatorCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.saveCreatorCalls
}

// GetNoteExistsCalls 获取 NoteExists 调用次数
func (m *MockStore) GetNoteExistsCalls() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.noteExistsCalls
}

// Reset 重置 Mock 状态
func (m *MockStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.notes = make(map[string]*Note)
	m.comments = make(map[string]*NoteComment)
	m.creators = make(map[string]*Creator)

	m.saveNoteError = nil
	m.saveCommentError = nil
	m.saveCreatorError = nil
	m.noteExistsError = nil

	m.saveNoteCalls = 0
	m.saveCommentCalls = 0
	m.saveCreatorCalls = 0
	m.noteExistsCalls = 0
}

// 常见的测试错误
var (
	ErrMockSaveFailed   = errors.New("mock save failed")
	ErrMockQueryFailed  = errors.New("mock query failed")
	ErrMockNetworkError = errors.New("mock network error")
	ErrMockTimeout      = errors.New("mock timeout")
)
