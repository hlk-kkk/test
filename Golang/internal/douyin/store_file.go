// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/douyin/store_file.go
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

func (s *FileStore) SaveAweme(ctx context.Context, aweme *Aweme) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	headers, values, row := awemeRecord(aweme)
	switch s.format {
	case "json":
		return s.saveJSON(row, "contents")
	case "csv":
		return s.saveCSV(headers, values, "contents")
	default:
		return fmt.Errorf("unsupported douyin file format: %s", s.format)
	}
}

func (s *FileStore) SaveComment(ctx context.Context, comment *AwemeComment) error {
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
		return fmt.Errorf("unsupported douyin file format: %s", s.format)
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
		return fmt.Errorf("unsupported douyin file format: %s", s.format)
	}
}

func (s *FileStore) saveJSON(row map[string]any, storeType string) error {
	path := filepath.Join(s.rootDir, "douyin", "json", s.crawlerType+"_"+storeType+"_"+time.Now().Format("2006-01-02")+".json")
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
	path := filepath.Join(s.rootDir, "douyin", "1_"+s.crawlerType+"_"+storeType+"_"+time.Now().Format("2006-01-02")+".csv")
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

func awemeRecord(aweme *Aweme) ([]string, []string, map[string]any) {
	headers := []string{
		"aweme_id",
		"aweme_type",
		"title",
		"desc",
		"create_time",
		"liked_count",
		"comment_count",
		"share_count",
		"collected_count",
		"aweme_url",
		"cover_url",
		"video_download_url",
		"source_keyword",
		"is_ai_generated",
		"user_id",
		"sec_uid",
		"short_user_id",
		"user_unique_id",
		"nickname",
		"avatar",
		"user_signature",
		"ip_location",
		"last_modify_ts",
	}
	values := []string{
		aweme.AwemeID,
		aweme.AwemeType,
		aweme.Title,
		aweme.Desc,
		aweme.CreateTime,
		aweme.LikedCount,
		aweme.CommentCount,
		aweme.ShareCount,
		aweme.CollectedCount,
		aweme.AwemeURL,
		aweme.CoverURL,
		aweme.VideoDownloadURL,
		aweme.SourceKeyword,
		fmt.Sprint(aweme.IsAIGenerated),
		aweme.UserID,
		aweme.SecUID,
		aweme.ShortUserID,
		aweme.UserUniqueID,
		aweme.Nickname,
		aweme.Avatar,
		aweme.UserSignature,
		aweme.IPLocation,
		fmt.Sprint(aweme.LastModifyTs),
	}
	row := map[string]any{
		"aweme_id":           aweme.AwemeID,
		"aweme_type":         aweme.AwemeType,
		"title":              aweme.Title,
		"desc":               aweme.Desc,
		"create_time":        aweme.CreateTime,
		"liked_count":        aweme.LikedCount,
		"comment_count":      aweme.CommentCount,
		"share_count":        aweme.ShareCount,
		"collected_count":    aweme.CollectedCount,
		"aweme_url":          aweme.AwemeURL,
		"cover_url":          aweme.CoverURL,
		"video_download_url": aweme.VideoDownloadURL,
		"source_keyword":     aweme.SourceKeyword,
		"is_ai_generated":    aweme.IsAIGenerated,
		"user_id":            aweme.UserID,
		"sec_uid":            aweme.SecUID,
		"short_user_id":      aweme.ShortUserID,
		"user_unique_id":     aweme.UserUniqueID,
		"nickname":           aweme.Nickname,
		"avatar":             aweme.Avatar,
		"user_signature":     aweme.UserSignature,
		"ip_location":        aweme.IPLocation,
		"last_modify_ts":     aweme.LastModifyTs,
	}
	return headers, values, row
}

func commentRecord(comment *AwemeComment) ([]string, []string, map[string]any) {
	headers := []string{
		"comment_id",
		"aweme_id",
		"content",
		"create_time",
		"sub_comment_count",
		"parent_comment_id",
		"reply_to_reply_id",
		"like_count",
		"pictures",
		"ip_location",
		"user_id",
		"sec_uid",
		"short_user_id",
		"user_unique_id",
		"nickname",
		"avatar",
		"user_signature",
		"last_modify_ts",
	}
	values := []string{
		comment.CommentID,
		comment.AwemeID,
		comment.Content,
		comment.CreateTime,
		comment.SubCommentCount,
		comment.ParentCommentID,
		comment.ReplyToReplyID,
		comment.LikeCount,
		comment.Pictures,
		comment.IPLocation,
		comment.UserID,
		comment.SecUID,
		comment.ShortUserID,
		comment.UserUniqueID,
		comment.Nickname,
		comment.Avatar,
		comment.UserSignature,
		fmt.Sprint(comment.LastModifyTs),
	}
	row := map[string]any{
		"comment_id":        comment.CommentID,
		"aweme_id":          comment.AwemeID,
		"content":           comment.Content,
		"create_time":       comment.CreateTime,
		"sub_comment_count": comment.SubCommentCount,
		"parent_comment_id": comment.ParentCommentID,
		"reply_to_reply_id": comment.ReplyToReplyID,
		"like_count":        comment.LikeCount,
		"pictures":          comment.Pictures,
		"ip_location":       comment.IPLocation,
		"user_id":           comment.UserID,
		"sec_uid":           comment.SecUID,
		"short_user_id":     comment.ShortUserID,
		"user_unique_id":    comment.UserUniqueID,
		"nickname":          comment.Nickname,
		"avatar":            comment.Avatar,
		"user_signature":    comment.UserSignature,
		"last_modify_ts":    comment.LastModifyTs,
	}
	return headers, values, row
}

func creatorRecord(creator *Creator) ([]string, []string, map[string]any) {
	headers := []string{
		"user_id",
		"nickname",
		"avatar",
		"ip_location",
		"desc",
		"gender",
		"follows",
		"fans",
		"interaction",
		"videos_count",
		"last_modify_ts",
	}
	values := []string{
		creator.UserID,
		creator.Nickname,
		creator.Avatar,
		creator.IPLocation,
		creator.Desc,
		creator.Gender,
		creator.Follows,
		creator.Fans,
		creator.Interaction,
		creator.VideosCount,
		fmt.Sprint(creator.LastModifyTs),
	}
	row := map[string]any{
		"user_id":        creator.UserID,
		"nickname":       creator.Nickname,
		"avatar":         creator.Avatar,
		"ip_location":    creator.IPLocation,
		"desc":           creator.Desc,
		"gender":         creator.Gender,
		"follows":        creator.Follows,
		"fans":           creator.Fans,
		"interaction":    creator.Interaction,
		"videos_count":   creator.VideosCount,
		"last_modify_ts": creator.LastModifyTs,
	}
	return headers, values, row
}

var _ Store = (*FileStore)(nil)
