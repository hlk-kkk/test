// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/kuaishou/store_sqlite.go
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

package kuaishou

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
	if err := s.db.AutoMigrate(&Video{}, &VideoComment{}, &Creator{}, &HotSearchItem{}, &HotSearchContentRelation{}); err != nil {
		return err
	}
	statements := []string{
		"CREATE INDEX IF NOT EXISTS idx_kuaishou_vi_video_i_c5c6a6 ON kuaishou_video (video_id)",
		"CREATE INDEX IF NOT EXISTS idx_kuaishou_vi_create__a10dee ON kuaishou_video (create_time)",
		"CREATE INDEX IF NOT EXISTS idx_kuaishou_vi_comment_ed48fa ON kuaishou_video_comment (comment_id)",
		"CREATE INDEX IF NOT EXISTS idx_kuaishou_vi_video_i_e50914 ON kuaishou_video_comment (video_id)",
		"CREATE INDEX IF NOT EXISTS idx_kuaishou_creator_user_id ON kuaishou_creator (user_id)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_hot_search_item_id ON platform_hot_search_item (hot_item_id)",
		"CREATE INDEX IF NOT EXISTS idx_hot_search_item_board ON platform_hot_search_item (platform, board_name, snapshot_date, rank_no)",
		"CREATE INDEX IF NOT EXISTS idx_hot_search_item_title ON platform_hot_search_item (platform, title)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_hot_search_rel_id ON platform_hot_search_content_rel (rel_id)",
		"CREATE INDEX IF NOT EXISTS idx_hot_search_rel_item ON platform_hot_search_content_rel (hot_item_id)",
		"CREATE INDEX IF NOT EXISTS idx_hot_search_rel_content ON platform_hot_search_content_rel (platform, content_table, content_id)",
	}
	for _, statement := range statements {
		if err := s.db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteDBStore) SaveVideo(ctx context.Context, video *Video) error {
	now := time.Now().UnixMilli()
	if video.LastModifyTs == 0 {
		video.LastModifyTs = now
	}

	var existing Video
	err := s.db.WithContext(ctx).Where("video_id = ?", video.VideoID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if video.AddTs == 0 {
			video.AddTs = now
		}
		return s.db.WithContext(ctx).Create(video).Error
	}
	if err != nil {
		return err
	}

	video.ID = existing.ID
	video.AddTs = existing.AddTs
	return s.db.WithContext(ctx).
		Model(&Video{}).
		Where("video_id = ?", video.VideoID).
		Select("*").
		Omit("id").
		Updates(video).Error
}

func (s *SQLiteDBStore) VideoExists(ctx context.Context, videoID string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&Video{}).Where("video_id = ?", videoID).Count(&count).Error
	return count > 0, err
}

func (s *SQLiteDBStore) SaveComment(ctx context.Context, comment *VideoComment) error {
	now := time.Now().UnixMilli()
	if comment.LastModifyTs == 0 {
		comment.LastModifyTs = now
	}

	var existing VideoComment
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
		Model(&VideoComment{}).
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

func (s *SQLiteDBStore) SaveHotSearchItem(ctx context.Context, item *HotSearchItem) error {
	now := time.Now().UnixMilli()
	if item.LastModifyTs == 0 {
		item.LastModifyTs = now
	}

	var existing HotSearchItem
	err := s.db.WithContext(ctx).Where("hot_item_id = ?", item.HotItemID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if item.AddTs == 0 {
			item.AddTs = now
		}
		return s.db.WithContext(ctx).Create(item).Error
	}
	if err != nil {
		return err
	}

	item.ID = existing.ID
	item.AddTs = existing.AddTs
	return s.db.WithContext(ctx).
		Model(&HotSearchItem{}).
		Where("hot_item_id = ?", item.HotItemID).
		Select("*").
		Omit("id").
		Updates(item).Error
}

func (s *SQLiteDBStore) SaveHotSearchRelation(ctx context.Context, relation *HotSearchContentRelation) error {
	now := time.Now().UnixMilli()
	if relation.LastModifyTs == 0 {
		relation.LastModifyTs = now
	}

	var existing HotSearchContentRelation
	err := s.db.WithContext(ctx).Where("rel_id = ?", relation.RelID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		if relation.AddTs == 0 {
			relation.AddTs = now
		}
		return s.db.WithContext(ctx).Create(relation).Error
	}
	if err != nil {
		return err
	}

	relation.ID = existing.ID
	relation.AddTs = existing.AddTs
	return s.db.WithContext(ctx).
		Model(&HotSearchContentRelation{}).
		Where("rel_id = ?", relation.RelID).
		Select("*").
		Omit("id").
		Updates(relation).Error
}

var _ Store = (*SQLiteDBStore)(nil)
