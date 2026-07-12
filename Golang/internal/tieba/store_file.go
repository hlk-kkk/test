// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/tieba/store_file.go
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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
		return fmt.Errorf("unsupported tieba file format: %s", s.format)
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
		return fmt.Errorf("unsupported tieba file format: %s", s.format)
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
		return fmt.Errorf("unsupported tieba file format: %s", s.format)
	}
}

func (s *FileStore) NoteExists(ctx context.Context, noteID string) (bool, error) {
	return false, nil
}

func (s *FileStore) saveJSON(row map[string]any, storeType string) error {
	path := filepath.Join(s.rootDir, "tieba", "json", s.crawlerType+"_"+storeType+"_"+time.Now().Format("2006-01-02")+".json")
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
	path := filepath.Join(s.rootDir, "tieba", "1_"+s.crawlerType+"_"+storeType+"_"+time.Now().Format("2006-01-02")+".csv")
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
		"title",
		"desc",
		"note_url",
		"publish_time",
		"user_link",
		"user_nickname",
		"user_avatar",
		"tieba_name",
		"tieba_link",
		"total_replay_num",
		"total_replay_page",
		"ip_location",
		"source_keyword",
		"last_modify_ts",
	}
	values := []string{
		note.NoteID,
		note.Title,
		note.Desc,
		note.NoteURL,
		note.PublishTime,
		note.UserLink,
		note.UserNickname,
		note.UserAvatar,
		note.TiebaName,
		note.TiebaLink,
		fmt.Sprint(note.TotalReplayNum),
		fmt.Sprint(note.TotalReplayPage),
		note.IPLocation,
		note.SourceKeyword,
		fmt.Sprint(note.LastModifyTs),
	}
	row := map[string]any{
		"note_id":           note.NoteID,
		"title":             note.Title,
		"desc":              note.Desc,
		"note_url":          note.NoteURL,
		"publish_time":      note.PublishTime,
		"user_link":         note.UserLink,
		"user_nickname":     note.UserNickname,
		"user_avatar":       note.UserAvatar,
		"tieba_name":        note.TiebaName,
		"tieba_link":        note.TiebaLink,
		"total_replay_num":  note.TotalReplayNum,
		"total_replay_page": note.TotalReplayPage,
		"ip_location":       note.IPLocation,
		"source_keyword":    note.SourceKeyword,
		"last_modify_ts":    note.LastModifyTs,
	}
	return headers, values, row
}

func commentRecord(comment *Comment) ([]string, []string, map[string]any) {
	headers := []string{
		"comment_id",
		"parent_comment_id",
		"content",
		"user_link",
		"user_nickname",
		"user_avatar",
		"publish_time",
		"ip_location",
		"sub_comment_count",
		"note_id",
		"note_url",
		"tieba_id",
		"tieba_name",
		"tieba_link",
		"last_modify_ts",
	}
	values := []string{
		comment.CommentID,
		comment.ParentCommentID,
		comment.Content,
		comment.UserLink,
		comment.UserNickname,
		comment.UserAvatar,
		comment.PublishTime,
		comment.IPLocation,
		fmt.Sprint(comment.SubCommentCount),
		comment.NoteID,
		comment.NoteURL,
		comment.TiebaID,
		comment.TiebaName,
		comment.TiebaLink,
		fmt.Sprint(comment.LastModifyTs),
	}
	row := map[string]any{
		"comment_id":        comment.CommentID,
		"parent_comment_id": comment.ParentCommentID,
		"content":           comment.Content,
		"user_link":         comment.UserLink,
		"user_nickname":     comment.UserNickname,
		"user_avatar":       comment.UserAvatar,
		"publish_time":      comment.PublishTime,
		"ip_location":       comment.IPLocation,
		"sub_comment_count": comment.SubCommentCount,
		"note_id":           comment.NoteID,
		"note_url":          comment.NoteURL,
		"tieba_id":          comment.TiebaID,
		"tieba_name":        comment.TiebaName,
		"tieba_link":        comment.TiebaLink,
		"last_modify_ts":    comment.LastModifyTs,
	}
	return headers, values, row
}

func creatorRecord(creator *Creator) ([]string, []string, map[string]any) {
	headers := []string{
		"user_id",
		"user_name",
		"nickname",
		"gender",
		"avatar",
		"ip_location",
		"follows",
		"fans",
		"registration_duration",
		"last_modify_ts",
	}
	values := []string{
		creator.UserID,
		creator.UserName,
		creator.Nickname,
		creator.Gender,
		creator.Avatar,
		creator.IPLocation,
		creator.Follows,
		creator.Fans,
		creator.RegistrationDuration,
		fmt.Sprint(creator.LastModifyTs),
	}
	row := map[string]any{
		"user_id":               creator.UserID,
		"user_name":             creator.UserName,
		"nickname":              creator.Nickname,
		"gender":                creator.Gender,
		"avatar":                creator.Avatar,
		"ip_location":           creator.IPLocation,
		"follows":               creatorNumber(creator.Follows),
		"fans":                  creatorNumber(creator.Fans),
		"registration_duration": creator.RegistrationDuration,
		"last_modify_ts":        creator.LastModifyTs,
	}
	return headers, values, row
}

func creatorNumber(value string) any {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return value
	}
	return parsed
}

var _ Store = (*FileStore)(nil)
