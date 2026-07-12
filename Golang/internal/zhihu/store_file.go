// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/zhihu/store_file.go
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

package zhihu

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

func (s *FileStore) SaveContent(ctx context.Context, content *Content) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if content.LastModifyTs == 0 {
		content.LastModifyTs = time.Now().UnixMilli()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	headers, values, row := contentRecord(content)
	switch s.format {
	case "json":
		return s.saveJSON(row, "contents")
	case "csv":
		return s.saveCSV(headers, values, "contents")
	default:
		return fmt.Errorf("unsupported zhihu file format: %s", s.format)
	}
}

func (s *FileStore) SaveComment(ctx context.Context, comment *Comment) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if comment.LastModifyTs == 0 {
		comment.LastModifyTs = time.Now().UnixMilli()
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
		return fmt.Errorf("unsupported zhihu file format: %s", s.format)
	}
}

func (s *FileStore) SaveCreator(ctx context.Context, creator *Creator) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if creator.LastModifyTs == 0 {
		creator.LastModifyTs = time.Now().UnixMilli()
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
		return fmt.Errorf("unsupported zhihu file format: %s", s.format)
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

func (s *FileStore) ContentExists(ctx context.Context, contentID string) (bool, error) {
	return false, nil
}

func (s *FileStore) saveJSON(row map[string]any, storeType string) error {
	path := filepath.Join(s.rootDir, "zhihu", "json", s.crawlerType+"_"+storeType+"_"+time.Now().Format("2006-01-02")+".json")
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
	path := filepath.Join(s.rootDir, "zhihu", "1_"+s.crawlerType+"_"+storeType+"_"+time.Now().Format("2006-01-02")+".csv")
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

func contentRecord(content *Content) ([]string, []string, map[string]any) {
	headers := []string{
		"content_id",
		"content_type",
		"content_text",
		"content_url",
		"question_id",
		"title",
		"desc",
		"created_time",
		"updated_time",
		"voteup_count",
		"comment_count",
		"source_keyword",
		"user_id",
		"user_link",
		"user_nickname",
		"user_avatar",
		"user_url_token",
		"last_modify_ts",
	}
	values := []string{
		content.ContentID,
		content.ContentType,
		content.ContentText,
		content.ContentURL,
		content.QuestionID,
		content.Title,
		content.Desc,
		content.CreatedTime,
		content.UpdatedTime,
		fmt.Sprint(content.VoteupCount),
		fmt.Sprint(content.CommentCount),
		content.SourceKeyword,
		content.UserID,
		content.UserLink,
		content.UserNickname,
		content.UserAvatar,
		content.UserURLToken,
		fmt.Sprint(content.LastModifyTs),
	}
	row := map[string]any{
		"content_id":     content.ContentID,
		"content_type":   content.ContentType,
		"content_text":   content.ContentText,
		"content_url":    content.ContentURL,
		"question_id":    content.QuestionID,
		"title":          content.Title,
		"desc":           content.Desc,
		"created_time":   numericStringOrValue(content.CreatedTime),
		"updated_time":   numericStringOrValue(content.UpdatedTime),
		"voteup_count":   content.VoteupCount,
		"comment_count":  content.CommentCount,
		"source_keyword": content.SourceKeyword,
		"user_id":        content.UserID,
		"user_link":      content.UserLink,
		"user_nickname":  content.UserNickname,
		"user_avatar":    content.UserAvatar,
		"user_url_token": content.UserURLToken,
		"last_modify_ts": content.LastModifyTs,
	}
	return headers, values, row
}

func commentRecord(comment *Comment) ([]string, []string, map[string]any) {
	headers := []string{
		"comment_id",
		"parent_comment_id",
		"content",
		"publish_time",
		"ip_location",
		"sub_comment_count",
		"like_count",
		"dislike_count",
		"content_id",
		"content_type",
		"user_id",
		"user_link",
		"user_nickname",
		"user_avatar",
		"last_modify_ts",
	}
	values := []string{
		comment.CommentID,
		comment.ParentCommentID,
		comment.Content,
		comment.PublishTime,
		comment.IPLocation,
		fmt.Sprint(comment.SubCommentCount),
		fmt.Sprint(comment.LikeCount),
		fmt.Sprint(comment.DislikeCount),
		comment.ContentID,
		comment.ContentType,
		comment.UserID,
		comment.UserLink,
		comment.UserNickname,
		comment.UserAvatar,
		fmt.Sprint(comment.LastModifyTs),
	}
	row := map[string]any{
		"comment_id":        comment.CommentID,
		"parent_comment_id": comment.ParentCommentID,
		"content":           comment.Content,
		"publish_time":      numericStringOrValue(comment.PublishTime),
		"ip_location":       comment.IPLocation,
		"sub_comment_count": comment.SubCommentCount,
		"like_count":        comment.LikeCount,
		"dislike_count":     comment.DislikeCount,
		"content_id":        comment.ContentID,
		"content_type":      comment.ContentType,
		"user_id":           comment.UserID,
		"user_link":         comment.UserLink,
		"user_nickname":     comment.UserNickname,
		"user_avatar":       comment.UserAvatar,
		"last_modify_ts":    comment.LastModifyTs,
	}
	return headers, values, row
}

func creatorRecord(creator *Creator) ([]string, []string, map[string]any) {
	headers := []string{
		"user_id",
		"user_link",
		"user_nickname",
		"user_avatar",
		"url_token",
		"gender",
		"ip_location",
		"follows",
		"fans",
		"anwser_count",
		"video_count",
		"question_count",
		"article_count",
		"column_count",
		"get_voteup_count",
		"last_modify_ts",
	}
	values := []string{
		creator.UserID,
		creator.UserLink,
		creator.UserNickname,
		creator.UserAvatar,
		creator.URLToken,
		creator.Gender,
		creator.IPLocation,
		fmt.Sprint(creator.Follows),
		fmt.Sprint(creator.Fans),
		fmt.Sprint(creator.AnwserCount),
		fmt.Sprint(creator.VideoCount),
		fmt.Sprint(creator.QuestionCount),
		fmt.Sprint(creator.ArticleCount),
		fmt.Sprint(creator.ColumnCount),
		fmt.Sprint(creator.GetVoteupCount),
		fmt.Sprint(creator.LastModifyTs),
	}
	row := map[string]any{
		"user_id":          creator.UserID,
		"user_link":        creator.UserLink,
		"user_nickname":    creator.UserNickname,
		"user_avatar":      creator.UserAvatar,
		"url_token":        creator.URLToken,
		"gender":           creator.Gender,
		"ip_location":      creator.IPLocation,
		"follows":          creator.Follows,
		"fans":             creator.Fans,
		"anwser_count":     creator.AnwserCount,
		"video_count":      creator.VideoCount,
		"question_count":   creator.QuestionCount,
		"article_count":    creator.ArticleCount,
		"column_count":     creator.ColumnCount,
		"get_voteup_count": creator.GetVoteupCount,
		"last_modify_ts":   creator.LastModifyTs,
	}
	return headers, values, row
}

func numericStringOrValue(value string) any {
	if value == "" {
		return value
	}
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parsed
	}
	return value
}

var _ Store = (*FileStore)(nil)
