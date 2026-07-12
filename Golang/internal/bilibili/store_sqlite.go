// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/bilibili/store_sqlite.go
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

package bilibili

import (
	"context"
	"os"
	"path/filepath"

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
	if err := s.db.AutoMigrate(&Video{}, &VideoComment{}, &UpInfo{}); err != nil {
		return err
	}
	statements := []string{
		"CREATE INDEX IF NOT EXISTS idx_bilibili_vi_video_i_31c36e ON bilibili_video (video_id)",
		"CREATE INDEX IF NOT EXISTS idx_bilibili_vi_create__73e0ec ON bilibili_video (create_time)",
		"CREATE INDEX IF NOT EXISTS idx_bilibili_vi_comment_41c34e ON bilibili_video_comment (comment_id)",
		"CREATE INDEX IF NOT EXISTS idx_bilibili_vi_video_i_f22873 ON bilibili_video_comment (video_id)",
		"CREATE INDEX IF NOT EXISTS idx_bilibili_vi_user_123456 ON bilibili_up_info (user_id)",
	}
	for _, statement := range statements {
		if err := s.db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteDBStore) SaveVideo(ctx context.Context, video *Video) error {
	var existing Video
	err := s.db.WithContext(ctx).Where("video_id = ?", video.VideoID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.WithContext(ctx).Create(video).Error
	}
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).
		Model(&Video{}).
		Where("video_id = ?", video.VideoID).
		Select("*").
		Omit("id", "add_ts").
		Updates(video).Error
}

func (s *SQLiteDBStore) VideoExists(ctx context.Context, videoID string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&Video{}).Where("video_id = ?", videoID).Count(&count).Error
	return count > 0, err
}

func (s *SQLiteDBStore) SaveComment(ctx context.Context, comment *VideoComment) error {
	var existing VideoComment
	err := s.db.WithContext(ctx).Where("comment_id = ?", comment.CommentID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.WithContext(ctx).Create(comment).Error
	}
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).
		Model(&VideoComment{}).
		Where("comment_id = ?", comment.CommentID).
		Select("*").
		Omit("id", "add_ts").
		Updates(comment).Error
}

func (s *SQLiteDBStore) SaveUpInfo(ctx context.Context, upInfo *UpInfo) error {
	var existing UpInfo
	err := s.db.WithContext(ctx).Where("user_id = ?", upInfo.UserID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.WithContext(ctx).Create(upInfo).Error
	}
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).
		Model(&UpInfo{}).
		Where("user_id = ?", upInfo.UserID).
		Select("*").
		Omit("id", "add_ts").
		Updates(upInfo).Error
}

var _ Store = (*SQLiteDBStore)(nil)
