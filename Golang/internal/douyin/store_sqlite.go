// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/douyin/store_sqlite.go
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
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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
	if err := s.db.AutoMigrate(&Aweme{}, &AwemeComment{}, &Creator{}); err != nil {
		return err
	}
	statements := []string{
		"CREATE INDEX IF NOT EXISTS idx_douyin_awem_aweme_i_6f7bc6 ON douyin_aweme (aweme_id)",
		"CREATE INDEX IF NOT EXISTS idx_douyin_awem_create__299dfe ON douyin_aweme (create_time)",
		"CREATE INDEX IF NOT EXISTS idx_douyin_awem_comment_fcd7e4 ON douyin_aweme_comment (comment_id)",
		"CREATE INDEX IF NOT EXISTS idx_douyin_awem_aweme_i_c50049 ON douyin_aweme_comment (aweme_id)",
		"CREATE INDEX IF NOT EXISTS idx_dy_creator_user_id ON dy_creator (user_id)",
	}
	for _, statement := range statements {
		if err := s.db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteDBStore) SaveAweme(ctx context.Context, aweme *Aweme) error {
	now := time.Now().UnixMilli()
	if aweme.LastModifyTs == 0 {
		aweme.LastModifyTs = now
	}

	var existing Aweme
	err := s.db.WithContext(ctx).Where("aweme_id = ?", aweme.AwemeID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if aweme.AddTs == 0 {
			aweme.AddTs = now
		}
		return s.db.WithContext(ctx).Create(aweme).Error
	}
	if err != nil {
		return err
	}

	aweme.ID = existing.ID
	aweme.AddTs = existing.AddTs
	return s.db.WithContext(ctx).
		Model(&Aweme{}).
		Where("aweme_id = ?", aweme.AwemeID).
		Select("*").
		Omit("id").
		Updates(aweme).Error
}

func (s *SQLiteDBStore) SaveComment(ctx context.Context, comment *AwemeComment) error {
	now := time.Now().UnixMilli()
	if comment.LastModifyTs == 0 {
		comment.LastModifyTs = now
	}

	var existing AwemeComment
	err := s.db.WithContext(ctx).Where("comment_id = ?", comment.CommentID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if comment.AddTs == 0 {
			comment.AddTs = now
		}
		return s.db.WithContext(ctx).Create(comment).Error
	}
	if err != nil {
		return err
	}

	comment.ID = existing.ID
	comment.AddTs = existing.AddTs
	return s.db.WithContext(ctx).
		Model(&AwemeComment{}).
		Where("comment_id = ?", comment.CommentID).
		Select("*").
		Omit("id").
		Updates(comment).Error
}

func (s *SQLiteDBStore) SaveCreator(ctx context.Context, creator *Creator) error {
	now := time.Now().UnixMilli()
	if creator.LastModifyTs == 0 {
		creator.LastModifyTs = now
	}

	var existing Creator
	err := s.db.WithContext(ctx).Where("user_id = ?", creator.UserID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if creator.AddTs == 0 {
			creator.AddTs = now
		}
		return s.db.WithContext(ctx).Create(creator).Error
	}
	if err != nil {
		return err
	}

	creator.ID = existing.ID
	creator.AddTs = existing.AddTs
	return s.db.WithContext(ctx).
		Model(&Creator{}).
		Where("user_id = ?", creator.UserID).
		Select("*").
		Omit("id").
		Updates(creator).Error
}

var _ Store = (*SQLiteDBStore)(nil)
