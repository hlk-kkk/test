// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/bilibili/store.go
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

// internal/bilibili/store.go
package bilibili

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/constant"
	"gorm.io/gorm"
)

// Store 存储接口
type Store interface {
	// Video 相关
	SaveVideo(ctx context.Context, video *Video) error
	VideoExists(ctx context.Context, videoID string) (bool, error)

	// Comment 相关
	SaveComment(ctx context.Context, comment *VideoComment) error

	// UpInfo 相关
	SaveUpInfo(ctx context.Context, upInfo *UpInfo) error
}

// DBStore 数据库存储实现
type DBStore struct {
	db *gorm.DB
}

func NewStore(cfg *config.Config, db *gorm.DB) (Store, error) {
	if cfg == nil {
		cfg = &config.Config{}
	}
	cfg.ApplyDefaults()
	switch strings.ToLower(cfg.SaveDataOption) {
	case "", constant.SaveDataToDb:
		if db == nil {
			return nil, errors.New("db store requires non-nil gorm db")
		}
		store := NewDBStore(db)
		if err := store.AutoMigrate(); err != nil {
			return nil, err
		}
		return store, nil
	case constant.SaveDataToSQLite:
		return NewSQLiteDBStoreFromPath(cfg.SqliteDBPath)
	case constant.SaveDataToJSON:
		return NewFileStore(cfg.DataDir, cfg.CrawlerType, constant.SaveDataToJSON), nil
	case constant.SaveDataToCSV:
		return NewFileStore(cfg.DataDir, cfg.CrawlerType, constant.SaveDataToCSV), nil
	default:
		return nil, fmt.Errorf("unsupported save data option: %s", cfg.SaveDataOption)
	}
}

// NewDBStore 创建数据库存储
func NewDBStore(db *gorm.DB) *DBStore {
	return &DBStore{db: db}
}

func (s *DBStore) AutoMigrate() error {
	return s.db.AutoMigrate(&Video{}, &VideoComment{}, &UpInfo{})
}

// SaveVideo 保存视频（匹配 Python 的查询后插入/更新语义）
func (s *DBStore) SaveVideo(ctx context.Context, video *Video) error {
	db := s.db.WithContext(ctx)
	var existing Video
	err := db.Where("video_id = ?", video.VideoID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(video).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&Video{}).
		Where("video_id = ?", video.VideoID).
		Select("*").
		Omit("id", "add_ts").
		Updates(video).Error
}

// VideoExists 检查视频是否存在
func (s *DBStore) VideoExists(ctx context.Context, videoID string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&Video{}).Where("video_id = ?", videoID).Count(&count).Error
	return count > 0, err
}

// SaveComment 保存评论（匹配 Python 的查询后插入/更新语义）
func (s *DBStore) SaveComment(ctx context.Context, comment *VideoComment) error {
	db := s.db.WithContext(ctx)
	var existing VideoComment
	err := db.Where("comment_id = ?", comment.CommentID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(comment).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&VideoComment{}).
		Where("comment_id = ?", comment.CommentID).
		Select("*").
		Omit("id", "add_ts").
		Updates(comment).Error
}

// SaveUpInfo 保存UP主信息（匹配 Python 的查询后插入/更新语义）
func (s *DBStore) SaveUpInfo(ctx context.Context, upInfo *UpInfo) error {
	db := s.db.WithContext(ctx)
	var existing UpInfo
	err := db.Where("user_id = ?", upInfo.UserID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(upInfo).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&UpInfo{}).
		Where("user_id = ?", upInfo.UserID).
		Select("*").
		Omit("id", "add_ts").
		Updates(upInfo).Error
}
