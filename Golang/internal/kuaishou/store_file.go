// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/kuaishou/store_file.go
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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileStore struct {
	rootDir     string
	crawlerType string
	format      string
	mu          sync.Mutex
}

func NewFileStore(rootDir, crawlerType, format string) *FileStore {
	if rootDir == "" {
		rootDir = "data"
	}
	if crawlerType == "" {
		crawlerType = "search"
	}
	return &FileStore{rootDir: rootDir, crawlerType: crawlerType, format: format}
}

func (s *FileStore) SaveVideo(ctx context.Context, video *Video) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	headers, values, row := videoRecord(video)
	switch s.format {
	case "json":
		return s.saveJSON(row, "contents")
	case "csv":
		return s.saveCSV(headers, values, "contents")
	default:
		return fmt.Errorf("unsupported kuaishou file format: %s", s.format)
	}
}

func (s *FileStore) SaveComment(ctx context.Context, comment *VideoComment) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	headers, values, row := commentRecord(comment)
	switch s.format {
	case "json":
		return s.saveJSON(row, "comments")
	case "csv":
		return s.saveCSV(headers, values, "comments")
	default:
		return fmt.Errorf("unsupported kuaishou file format: %s", s.format)
	}
}

func (s *FileStore) SaveCreator(ctx context.Context, creator *Creator) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if s.format == "csv" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, _, row := creatorRecord(creator)
	switch s.format {
	case "json":
		return s.saveJSON(row, "creator")
	default:
		return fmt.Errorf("unsupported kuaishou file format: %s", s.format)
	}
}

func (s *FileStore) SaveHotSearchItem(ctx context.Context, item *HotSearchItem) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return nil
}

func (s *FileStore) SaveHotSearchRelation(ctx context.Context, relation *HotSearchContentRelation) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return nil
}

func (s *FileStore) VideoExists(ctx context.Context, videoID string) (bool, error) {
	return false, nil
}

func (s *FileStore) saveJSON(row map[string]any, storeType string) error {
	path := filepath.Join(s.rootDir, "kuaishou", "json", s.crawlerType+"_"+storeType+"_"+time.Now().Format("2006-01-02")+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	rows := make([]map[string]any, 0)
	if body, err := os.ReadFile(path); err == nil && len(body) > 0 {
		if err := json.Unmarshal(body, &rows); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	rows = append(rows, row)
	body, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0644)
}

func (s *FileStore) saveCSV(headers, values []string, storeType string) error {
	path := filepath.Join(s.rootDir, "kuaishou", "1_"+s.crawlerType+"_"+storeType+"_"+time.Now().Format("2006-01-02")+".csv")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	needsHeader := true
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		needsHeader = false
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if needsHeader {
		if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
			return err
		}
	}

	writer := csv.NewWriter(file)
	if needsHeader {
		if err := writer.Write(headers); err != nil {
			return err
		}
	}
	if err := writer.Write(values); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func videoRecord(video *Video) ([]string, []string, map[string]any) {
	headers := []string{
		"video_id",
		"video_type",
		"title",
		"desc",
		"create_time",
		"liked_count",
		"viewd_count",
		"video_url",
		"video_cover_url",
		"video_play_url",
		"source_keyword",
		"user_id",
		"nickname",
		"avatar",
		"last_modify_ts",
	}
	values := []string{
		video.VideoID,
		video.VideoType,
		video.Title,
		video.Desc,
		video.CreateTime,
		video.LikedCount,
		video.ViewdCount,
		video.VideoURL,
		video.VideoCoverURL,
		video.VideoPlayURL,
		video.SourceKeyword,
		video.UserID,
		video.Nickname,
		video.Avatar,
		fmt.Sprint(video.LastModifyTs),
	}
	row := map[string]any{
		"video_id":        video.VideoID,
		"video_type":      video.VideoType,
		"title":           video.Title,
		"desc":            video.Desc,
		"create_time":     video.CreateTime,
		"liked_count":     video.LikedCount,
		"viewd_count":     video.ViewdCount,
		"video_url":       video.VideoURL,
		"video_cover_url": video.VideoCoverURL,
		"video_play_url":  video.VideoPlayURL,
		"source_keyword":  video.SourceKeyword,
		"user_id":         video.UserID,
		"nickname":        video.Nickname,
		"avatar":          video.Avatar,
		"last_modify_ts":  video.LastModifyTs,
	}
	return headers, values, row
}

func commentRecord(comment *VideoComment) ([]string, []string, map[string]any) {
	headers := []string{
		"comment_id",
		"parent_comment_id",
		"video_id",
		"content",
		"create_time",
		"sub_comment_count",
		"like_count",
		"user_id",
		"nickname",
		"avatar",
		"last_modify_ts",
	}
	values := []string{
		comment.CommentID,
		comment.ParentCommentID,
		comment.VideoID,
		comment.Content,
		comment.CreateTime,
		comment.SubCommentCount,
		comment.LikeCount,
		comment.UserID,
		comment.Nickname,
		comment.Avatar,
		fmt.Sprint(comment.LastModifyTs),
	}
	row := map[string]any{
		"comment_id":        comment.CommentID,
		"parent_comment_id": comment.ParentCommentID,
		"video_id":          comment.VideoID,
		"content":           comment.Content,
		"create_time":       comment.CreateTime,
		"sub_comment_count": comment.SubCommentCount,
		"like_count":        comment.LikeCount,
		"user_id":           comment.UserID,
		"nickname":          comment.Nickname,
		"avatar":            comment.Avatar,
		"last_modify_ts":    comment.LastModifyTs,
	}
	return headers, values, row
}

func creatorRecord(creator *Creator) ([]string, []string, map[string]any) {
	headers := []string{
		"user_id",
		"nickname",
		"avatar",
		"gender",
		"desc",
		"ip_location",
		"follows",
		"fans",
		"videos_count",
		"last_modify_ts",
	}
	values := []string{
		creator.UserID,
		creator.Nickname,
		creator.Avatar,
		creator.Gender,
		creator.Desc,
		creator.IPLocation,
		creator.Follows,
		creator.Fans,
		creator.VideosCount,
		fmt.Sprint(creator.LastModifyTs),
	}
	row := map[string]any{
		"user_id":        creator.UserID,
		"nickname":       creator.Nickname,
		"avatar":         creator.Avatar,
		"gender":         creator.Gender,
		"desc":           creator.Desc,
		"ip_location":    creator.IPLocation,
		"follows":        creator.Follows,
		"fans":           creator.Fans,
		"videos_count":   creator.VideosCount,
		"last_modify_ts": creator.LastModifyTs,
	}
	return headers, values, row
}
