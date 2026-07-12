// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/xhs/store_sqlite.go
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

package xhs

import (
	"context"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SQLiteDBStore uses the same public store contract as DBStore, but avoids
// MySQL-specific upsert syntax so local SQLite runs match Python's update flow.
type SQLiteDBStore struct {
	db *gorm.DB
}

func NewSQLiteDBStoreFromPath(dbPath string) (*SQLiteDBStore, error) {
	if dbPath == "" {
		dbPath = "media_crawler.db"
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	store := NewSQLiteDBStore(db)
	if err := store.AutoMigrate(); err != nil {
		return nil, err
	}
	return store, nil
}

func NewSQLiteDBStore(db *gorm.DB) *SQLiteDBStore {
	return &SQLiteDBStore{db: db}
}

func (s *SQLiteDBStore) AutoMigrate() error {
	if err := s.db.AutoMigrate(&Note{}, &NoteComment{}, &Creator{}); err != nil {
		return err
	}
	statements := []string{
		"CREATE INDEX IF NOT EXISTS idx_xhs_note_note_id_209457 ON xhs_note (note_id)",
		"CREATE INDEX IF NOT EXISTS idx_xhs_note_time_eaa910 ON xhs_note (time)",
		"CREATE INDEX IF NOT EXISTS idx_xhs_note_co_comment_8e8349 ON xhs_note_comment (comment_id)",
		"CREATE INDEX IF NOT EXISTS idx_xhs_note_co_create__204f8d ON xhs_note_comment (create_time)",
		"CREATE INDEX IF NOT EXISTS idx_xhs_creator_user_id ON xhs_creator (user_id)",
	}
	for _, statement := range statements {
		if err := s.db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteDBStore) SaveNote(ctx context.Context, note *Note) error {
	var existing Note
	err := s.db.WithContext(ctx).Where("note_id = ?", note.NoteID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.WithContext(ctx).Create(note).Error
	}
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).
		Model(&Note{}).
		Where("note_id = ?", note.NoteID).
		Select("*").
		Omit("id", "add_ts").
		Updates(note).Error
}

func (s *SQLiteDBStore) NoteExists(ctx context.Context, noteID string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&Note{}).Where("note_id = ?", noteID).Count(&count).Error
	return count > 0, err
}

func (s *SQLiteDBStore) SaveComment(ctx context.Context, comment *NoteComment) error {
	var existing NoteComment
	err := s.db.WithContext(ctx).Where("comment_id = ?", comment.CommentID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.WithContext(ctx).Create(comment).Error
	}
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).
		Model(&NoteComment{}).
		Where("comment_id = ?", comment.CommentID).
		Select("*").
		Omit("id", "add_ts").
		Updates(comment).Error
}

func (s *SQLiteDBStore) SaveCreator(ctx context.Context, creator *Creator) error {
	var existing Creator
	err := s.db.WithContext(ctx).Where("user_id = ?", creator.UserID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.WithContext(ctx).Create(creator).Error
	}
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).
		Model(&Creator{}).
		Where("user_id = ?", creator.UserID).
		Select("*").
		Omit("id", "add_ts").
		Updates(creator).Error
}

var _ Store = (*SQLiteDBStore)(nil)
