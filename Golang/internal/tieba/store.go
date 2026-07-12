// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/tieba/store.go
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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/constant"
	"gorm.io/gorm"
)

type Store interface {
	SaveNote(ctx context.Context, note *Note) error
	SaveComment(ctx context.Context, comment *Comment) error
	SaveCreator(ctx context.Context, creator *Creator) error
	NoteExists(ctx context.Context, noteID string) (bool, error)
}

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
	case constant.SaveDataToJSON:
		return NewFileStore(cfg.DataDir, cfg.CrawlerType, constant.SaveDataToJSON), nil
	case constant.SaveDataToCSV:
		return NewFileStore(cfg.DataDir, cfg.CrawlerType, constant.SaveDataToCSV), nil
	case constant.SaveDataToSQLite:
		return NewSQLiteDBStoreFromPath(cfg.SqliteDBPath)
	default:
		return nil, fmt.Errorf("tieba store for saveDataOption=%q is not implemented yet", cfg.SaveDataOption)
	}
}

func NewDBStore(db *gorm.DB) *DBStore {
	return &DBStore{db: db}
}

func (s *DBStore) AutoMigrate() error {
	return s.db.AutoMigrate(&Note{}, &Comment{}, &Creator{})
}

func (s *DBStore) SaveNote(ctx context.Context, note *Note) error {
	now := time.Now().UnixMilli()
	if note.LastModifyTs == 0 {
		note.LastModifyTs = now
	}

	var existing Note
	err := s.db.WithContext(ctx).Where("note_id = ?", note.NoteID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if note.AddTs == 0 {
			note.AddTs = now
		}
		return s.db.WithContext(ctx).Create(note).Error
	}
	if err != nil {
		return err
	}

	note.ID = existing.ID
	note.AddTs = existing.AddTs
	return s.db.WithContext(ctx).
		Model(&Note{}).
		Where("note_id = ?", note.NoteID).
		Select("*").
		Omit("id").
		Updates(note).Error
}

func (s *DBStore) SaveComment(ctx context.Context, comment *Comment) error {
	now := time.Now().UnixMilli()
	if comment.LastModifyTs == 0 {
		comment.LastModifyTs = now
	}

	var existing Comment
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
		Model(&Comment{}).
		Where("comment_id = ?", comment.CommentID).
		Select("*").
		Omit("id").
		Updates(comment).Error
}

func (s *DBStore) SaveCreator(ctx context.Context, creator *Creator) error {
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

func (s *DBStore) NoteExists(ctx context.Context, noteID string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&Note{}).Where("note_id = ?", noteID).Count(&count).Error
	return count > 0, err
}
