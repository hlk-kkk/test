// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/weibo/store_file.go
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

package weibo

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

func (s *FileStore) SaveNote(ctx context.Context, note *Note) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	headers, values, row := noteRecord(note)
	switch s.format {
	case "json":
		return s.saveJSON(row, "contents")
	case "csv":
		return s.saveCSV(headers, values, "contents")
	default:
		return fmt.Errorf("unsupported weibo file format: %s", s.format)
	}
}

func (s *FileStore) SaveComment(ctx context.Context, comment *Comment) error {
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
		return fmt.Errorf("unsupported weibo file format: %s", s.format)
	}
}

func (s *FileStore) SaveCreator(ctx context.Context, creator *Creator) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	headers, values, row := creatorRecord(creator)
	switch s.format {
	case "json":
		return s.saveJSON(row, "creator")
	case "csv":
		return s.saveCSV(headers, values, "creator")
	default:
		return fmt.Errorf("unsupported weibo file format: %s", s.format)
	}
}

func (s *FileStore) SaveHotSearchItem(ctx context.Context, item *HotSearchItem) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (s *FileStore) SaveHotSearchRelation(ctx context.Context, relation *HotSearchContentRelation) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (s *FileStore) NoteExists(ctx context.Context, noteID string) (bool, error) {
	return false, nil
}

func (s *FileStore) saveJSON(row map[string]any, storeType string) error {
	path := filepath.Join(s.rootDir, "weibo", "json", s.crawlerType+"_"+storeType+"_"+time.Now().Format("2006-01-02")+".json")
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
	path := filepath.Join(s.rootDir, "weibo", s.crawlerType+"_"+storeType+"_"+time.Now().Format("2006-01-02")+".csv")
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

func noteRecord(note *Note) ([]string, []string, map[string]any) {
	headers := []string{
		"note_id",
		"content",
		"create_time",
		"create_date_time",
		"liked_count",
		"comments_count",
		"shared_count",
		"note_url",
		"ip_location",
		"image_list",
		"video_url",
		"user_id",
		"nickname",
		"gender",
		"profile_url",
		"avatar",
		"source_keyword",
		"add_ts",
		"last_modify_ts",
	}
	values := []string{
		note.NoteID,
		note.Content,
		fmt.Sprint(note.CreateTime),
		note.CreateDateTime,
		note.LikedCount,
		note.CommentsCount,
		note.SharedCount,
		note.NoteURL,
		note.IPLocation,
		note.ImageList,
		note.VideoURL,
		note.UserID,
		note.Nickname,
		note.Gender,
		note.ProfileURL,
		note.Avatar,
		note.SourceKeyword,
		"",
		fmt.Sprint(note.LastModifyTs),
	}
	row := map[string]any{
		"note_id":          note.NoteID,
		"content":          note.Content,
		"create_time":      fmt.Sprint(note.CreateTime),
		"create_date_time": note.CreateDateTime,
		"liked_count":      note.LikedCount,
		"comments_count":   note.CommentsCount,
		"shared_count":     note.SharedCount,
		"note_url":         note.NoteURL,
		"ip_location":      note.IPLocation,
		"image_list":       note.ImageList,
		"video_url":        note.VideoURL,
		"user_id":          note.UserID,
		"nickname":         note.Nickname,
		"gender":           note.Gender,
		"profile_url":      note.ProfileURL,
		"avatar":           note.Avatar,
		"source_keyword":   note.SourceKeyword,
		"add_ts":           nil,
		"last_modify_ts":   note.LastModifyTs,
	}
	return headers, values, row
}

func commentRecord(comment *Comment) ([]string, []string, map[string]any) {
	headers := []string{
		"comment_id",
		"note_id",
		"content",
		"create_time",
		"create_date_time",
		"sub_comment_count",
		"like_count",
		"ip_location",
		"parent_comment_id",
		"user_id",
		"nickname",
		"gender",
		"profile_url",
		"avatar",
		"sub_comments",
		"add_ts",
		"last_modify_ts",
	}
	values := []string{
		comment.CommentID,
		comment.NoteID,
		comment.Content,
		fmt.Sprint(comment.CreateTime),
		comment.CreateDateTime,
		comment.SubCommentCount,
		comment.LikeCount,
		comment.IPLocation,
		comment.ParentCommentID,
		comment.UserID,
		comment.Nickname,
		comment.Gender,
		comment.ProfileURL,
		comment.Avatar,
		"",
		"",
		fmt.Sprint(comment.LastModifyTs),
	}
	row := map[string]any{
		"comment_id":        comment.CommentID,
		"note_id":           comment.NoteID,
		"content":           comment.Content,
		"create_time":       fmt.Sprint(comment.CreateTime),
		"create_date_time":  comment.CreateDateTime,
		"sub_comment_count": comment.SubCommentCount,
		"like_count":        comment.LikeCount,
		"ip_location":       comment.IPLocation,
		"parent_comment_id": comment.ParentCommentID,
		"user_id":           comment.UserID,
		"nickname":          comment.Nickname,
		"gender":            comment.Gender,
		"profile_url":       comment.ProfileURL,
		"avatar":            comment.Avatar,
		"sub_comments":      nil,
		"add_ts":            nil,
		"last_modify_ts":    comment.LastModifyTs,
	}
	return headers, values, row
}

func creatorRecord(creator *Creator) ([]string, []string, map[string]any) {
	headers := []string{
		"user_id",
		"nickname",
		"gender",
		"avatar",
		"desc",
		"ip_location",
		"follows",
		"fans",
		"tag_list",
		"add_ts",
		"last_modify_ts",
	}
	values := []string{
		creator.UserID,
		creator.Nickname,
		creator.Gender,
		creator.Avatar,
		creator.Desc,
		creator.IPLocation,
		creator.Follows,
		creator.Fans,
		creator.TagList,
		"",
		fmt.Sprint(creator.LastModifyTs),
	}
	row := map[string]any{
		"user_id":        creator.UserID,
		"nickname":       creator.Nickname,
		"gender":         creator.Gender,
		"avatar":         creator.Avatar,
		"desc":           creator.Desc,
		"ip_location":    creator.IPLocation,
		"follows":        creator.Follows,
		"fans":           creator.Fans,
		"tag_list":       creator.TagList,
		"add_ts":         nil,
		"last_modify_ts": creator.LastModifyTs,
	}
	return headers, values, row
}

var _ Store = (*FileStore)(nil)
