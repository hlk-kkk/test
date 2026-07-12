// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/douyin_mysql_e2e_test.go
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

package e2e_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/e2e"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/cookiebridge"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestCLIDouyinSearchDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "dy",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "dy-unified-client",
				Connected: true,
				Nicknames: map[string]string{"dy": "mock-douyin"},
				Cookies:   map[string]string{"dy": "sessionid=unified; msToken=unified"},
			},
		},
		PlatformHandler: newDouyinUnifiedSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "dy",
		Keywords:             []string{"mock-keyword"},
		CrawlerType:          "search",
		SaveDataOption:       "db",
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env:     mysqlServer.Env(),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	var row struct {
		AwemeID          string
		AwemeType        string
		Title            string
		Description      string
		CreateTime       int64
		LikedCount       string
		CommentCount     string
		ShareCount       string
		CollectedCount   string
		AwemeURL         string
		CoverURL         string
		VideoDownloadURL string
		SourceKeyword    string
		IsAIGenerated    int
		UserID           string
		SecUID           string
		ShortUserID      string
		UserUniqueID     string
		Nickname         string
		Avatar           string
		UserSignature    string
		IPLocation       string
	}
	err = db.QueryRow(`
		SELECT aweme_id, aweme_type, title, `+"`desc`"+`, create_time, liked_count, comment_count,
		       share_count, collected_count, aweme_url, cover_url, video_download_url,
		       source_keyword, is_ai_generated, user_id, sec_uid, short_user_id,
		       user_unique_id, nickname, avatar, user_signature, ip_location
		FROM douyin_aweme
		WHERE aweme_id = ?
	`, "aweme_e2e").Scan(
		&row.AwemeID,
		&row.AwemeType,
		&row.Title,
		&row.Description,
		&row.CreateTime,
		&row.LikedCount,
		&row.CommentCount,
		&row.ShareCount,
		&row.CollectedCount,
		&row.AwemeURL,
		&row.CoverURL,
		&row.VideoDownloadURL,
		&row.SourceKeyword,
		&row.IsAIGenerated,
		&row.UserID,
		&row.SecUID,
		&row.ShortUserID,
		&row.UserUniqueID,
		&row.Nickname,
		&row.Avatar,
		&row.UserSignature,
		&row.IPLocation,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "aweme_e2e", row.AwemeID)
	require.Equal(t, "0", row.AwemeType)
	require.Equal(t, "Douyin E2E mock aweme", row.Title)
	require.Equal(t, "mock douyin desc", row.Description)
	require.Equal(t, int64(1710000000), row.CreateTime)
	require.Equal(t, "88", row.LikedCount)
	require.Equal(t, "5", row.CommentCount)
	require.Equal(t, "3", row.ShareCount)
	require.Equal(t, "20", row.CollectedCount)
	require.Equal(t, "https://www.douyin.com/video/aweme_e2e", row.AwemeURL)
	require.Equal(t, "https://example.test/douyin-cover.png", row.CoverURL)
	require.Equal(t, "https://example.test/douyin-play.mp4", row.VideoDownloadURL)
	require.Equal(t, "mock-keyword", row.SourceKeyword)
	require.Equal(t, 1, row.IsAIGenerated)
	require.Equal(t, "dy-author", row.UserID)
	require.Equal(t, "sec-dy-author", row.SecUID)
	require.Equal(t, "123456", row.ShortUserID)
	require.Equal(t, "mock_douyin_author", row.UserUniqueID)
	require.Equal(t, "mock douyin author", row.Nickname)
	require.Equal(t, "https://example.test/douyin-avatar.png", row.Avatar)
	require.Equal(t, "mock signature", row.UserSignature)
	require.Equal(t, "上海", row.IPLocation)
}

func TestCLIDouyinSearchCommentsDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "dy",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "dy-comments-unified-client",
				Connected: true,
				Nicknames: map[string]string{"dy": "mock-douyin"},
				Cookies:   map[string]string{"dy": "sessionid=comments-unified; msToken=comments-unified"},
			},
		},
		PlatformHandler: newDouyinUnifiedSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "dy",
		Keywords:             []string{"mock-keyword"},
		CrawlerType:          "search",
		SaveDataOption:       "db",
		CrawlerMaxNotesCount: 1,
		EnableGetComments:    true,
		EnableGetSubComments: true,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env:     mysqlServer.Env(),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	rows, err := db.Query(`
		SELECT comment_id, aweme_id, content, create_time, sub_comment_count,
		       parent_comment_id, reply_to_reply_id, like_count, pictures,
		       ip_location, user_id, sec_uid, short_user_id, user_unique_id,
		       nickname, avatar, user_signature, add_ts, last_modify_ts
		FROM douyin_aweme_comment
		ORDER BY comment_id
	`)
	require.NoError(t, err, "command output:\n%s", output)
	defer rows.Close()

	type commentRow struct {
		CommentID       string
		AwemeID         string
		Content         string
		CreateTime      int64
		SubCommentCount string
		ParentCommentID string
		ReplyToReplyID  string
		LikeCount       string
		Pictures        string
		IPLocation      string
		UserID          string
		SecUID          string
		ShortUserID     string
		UserUniqueID    string
		Nickname        string
		Avatar          string
		UserSignature   string
		AddTs           int64
		LastModifyTs    int64
	}
	var comments []commentRow
	for rows.Next() {
		var row commentRow
		require.NoError(t, rows.Scan(
			&row.CommentID,
			&row.AwemeID,
			&row.Content,
			&row.CreateTime,
			&row.SubCommentCount,
			&row.ParentCommentID,
			&row.ReplyToReplyID,
			&row.LikeCount,
			&row.Pictures,
			&row.IPLocation,
			&row.UserID,
			&row.SecUID,
			&row.ShortUserID,
			&row.UserUniqueID,
			&row.Nickname,
			&row.Avatar,
			&row.UserSignature,
			&row.AddTs,
			&row.LastModifyTs,
		))
		comments = append(comments, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, comments, 2, "command output:\n%s", output)

	root := comments[0]
	require.Equal(t, "dy_comment_root", root.CommentID)
	require.Equal(t, "aweme_e2e", root.AwemeID)
	require.Equal(t, "root douyin comment", root.Content)
	require.Equal(t, int64(1710000100), root.CreateTime)
	require.Equal(t, "1", root.SubCommentCount)
	require.Equal(t, "", root.ParentCommentID)
	require.Equal(t, "", root.ReplyToReplyID)
	require.Equal(t, "7", root.LikeCount)
	require.Equal(t, "https://example.test/dy-comment.png", root.Pictures)
	require.Equal(t, "北京", root.IPLocation)
	require.Equal(t, "dy-commenter", root.UserID)
	require.Equal(t, "sec-dy-commenter", root.SecUID)
	require.Equal(t, "654321", root.ShortUserID)
	require.Equal(t, "mock_dy_commenter", root.UserUniqueID)
	require.Equal(t, "mock dy commenter", root.Nickname)
	require.Equal(t, "https://example.test/dy-comment-avatar.png", root.Avatar)
	require.Equal(t, "comment signature", root.UserSignature)
	require.NotZero(t, root.AddTs)
	require.NotZero(t, root.LastModifyTs)

	sub := comments[1]
	require.Equal(t, "dy_comment_sub", sub.CommentID)
	require.Equal(t, "aweme_e2e", sub.AwemeID)
	require.Equal(t, "sub douyin comment", sub.Content)
	require.Equal(t, int64(1710000200), sub.CreateTime)
	require.Equal(t, "0", sub.SubCommentCount)
	require.Equal(t, "dy_comment_root", sub.ParentCommentID)
	require.Equal(t, "dy_comment_root", sub.ReplyToReplyID)
	require.Equal(t, "2", sub.LikeCount)
	require.Equal(t, "", sub.Pictures)
	require.Equal(t, "广州", sub.IPLocation)
	require.Equal(t, "dy-sub-commenter", sub.UserID)
	require.Equal(t, "sec-dy-sub-commenter", sub.SecUID)
	require.Equal(t, "654322", sub.ShortUserID)
	require.Equal(t, "mock_dy_sub_commenter", sub.UserUniqueID)
	require.Equal(t, "mock dy sub commenter", sub.Nickname)
	require.Equal(t, "https://example.test/dy-sub-comment-avatar.png", sub.Avatar)
	require.Equal(t, "sub comment signature", sub.UserSignature)
	require.NotZero(t, sub.AddTs)
	require.NotZero(t, sub.LastModifyTs)
}

func TestCLIDouyinDetailDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "dy",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "dy-unified-detail-client",
				Connected: true,
				Nicknames: map[string]string{"dy": "mock-douyin"},
				Cookies:   map[string]string{"dy": "sessionid=unified-detail; msToken=unified-detail"},
			},
		},
		PlatformHandler: newDouyinUnifiedDetailMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "dy",
		Keywords:             []string{"mock-keyword"},
		CrawlerType:          "detail",
		SaveDataOption:       "db",
		SpecifiedNoteIDs:     []string{"aweme_detail_e2e"},
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env:     mysqlServer.Env(),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	var row struct {
		AwemeID          string
		AwemeType        string
		Title            string
		Description      string
		CreateTime       int64
		LikedCount       string
		CommentCount     string
		ShareCount       string
		CollectedCount   string
		AwemeURL         string
		CoverURL         string
		VideoDownloadURL string
		SourceKeyword    string
		IsAIGenerated    int
		UserID           string
		SecUID           string
		ShortUserID      string
		UserUniqueID     string
		Nickname         string
		Avatar           string
		UserSignature    string
		IPLocation       string
		AddTs            int64
		LastModifyTs     int64
	}
	err = db.QueryRow(`
		SELECT aweme_id, aweme_type, title, `+"`desc`"+`, create_time, liked_count, comment_count,
		       share_count, collected_count, aweme_url, cover_url, video_download_url,
		       source_keyword, is_ai_generated, user_id, sec_uid, short_user_id,
		       user_unique_id, nickname, avatar, user_signature, ip_location,
		       add_ts, last_modify_ts
		FROM douyin_aweme
		WHERE aweme_id = ?
	`, "aweme_detail_e2e").Scan(
		&row.AwemeID,
		&row.AwemeType,
		&row.Title,
		&row.Description,
		&row.CreateTime,
		&row.LikedCount,
		&row.CommentCount,
		&row.ShareCount,
		&row.CollectedCount,
		&row.AwemeURL,
		&row.CoverURL,
		&row.VideoDownloadURL,
		&row.SourceKeyword,
		&row.IsAIGenerated,
		&row.UserID,
		&row.SecUID,
		&row.ShortUserID,
		&row.UserUniqueID,
		&row.Nickname,
		&row.Avatar,
		&row.UserSignature,
		&row.IPLocation,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "aweme_detail_e2e", row.AwemeID)
	require.Equal(t, "0", row.AwemeType)
	require.Equal(t, "Douyin detail E2E mock aweme", row.Title)
	require.Equal(t, "mock douyin detail desc", row.Description)
	require.Equal(t, int64(1710000300), row.CreateTime)
	require.Equal(t, "188", row.LikedCount)
	require.Equal(t, "15", row.CommentCount)
	require.Equal(t, "13", row.ShareCount)
	require.Equal(t, "120", row.CollectedCount)
	require.Equal(t, "https://www.douyin.com/video/aweme_detail_e2e", row.AwemeURL)
	require.Equal(t, "https://example.test/douyin-detail-cover.png", row.CoverURL)
	require.Equal(t, "https://example.test/douyin-detail-play.mp4", row.VideoDownloadURL)
	require.Equal(t, "", row.SourceKeyword)
	require.Equal(t, 0, row.IsAIGenerated)
	require.Equal(t, "dy-detail-author", row.UserID)
	require.Equal(t, "sec-dy-detail-author", row.SecUID)
	require.Equal(t, "223344", row.ShortUserID)
	require.Equal(t, "mock_douyin_detail_author", row.UserUniqueID)
	require.Equal(t, "mock douyin detail author", row.Nickname)
	require.Equal(t, "https://example.test/douyin-detail-avatar.png", row.Avatar)
	require.Equal(t, "detail signature", row.UserSignature)
	require.Equal(t, "杭州", row.IPLocation)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func TestCLIDouyinCreatorDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "dy",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "dy-unified-creator-client",
				Connected: true,
				Nicknames: map[string]string{"dy": "mock-douyin"},
				Cookies:   map[string]string{"dy": "sessionid=unified-creator; msToken=unified-creator"},
			},
		},
		PlatformHandler: newDouyinUnifiedCreatorMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "dy",
		Keywords:             []string{"mock-keyword"},
		CrawlerType:          "creator",
		SaveDataOption:       "db",
		SpecifiedCreatorIDs:  []string{"sec-dy-creator"},
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env:     mysqlServer.Env(),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()

	var creator struct {
		UserID       string
		Nickname     string
		Avatar       string
		IPLocation   string
		Description  string
		Gender       string
		Follows      string
		Fans         string
		Interaction  string
		VideosCount  string
		AddTs        int64
		LastModifyTs int64
	}
	err = db.QueryRow(`
		SELECT user_id, nickname, avatar, ip_location, `+"`desc`"+`, gender,
		       follows, fans, interaction, videos_count, add_ts, last_modify_ts
		FROM dy_creator
		WHERE user_id = ?
	`, "dy-creator-user").Scan(
		&creator.UserID,
		&creator.Nickname,
		&creator.Avatar,
		&creator.IPLocation,
		&creator.Description,
		&creator.Gender,
		&creator.Follows,
		&creator.Fans,
		&creator.Interaction,
		&creator.VideosCount,
		&creator.AddTs,
		&creator.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "dy-creator-user", creator.UserID)
	require.Equal(t, "mock douyin creator", creator.Nickname)
	require.Equal(t, "https://example.test/douyin-creator-avatar-large.png", creator.Avatar)
	require.Equal(t, "成都", creator.IPLocation)
	require.Equal(t, "creator signature", creator.Description)
	require.Equal(t, "男", creator.Gender)
	require.Equal(t, "12", creator.Follows)
	require.Equal(t, "3456", creator.Fans)
	require.Equal(t, "7890", creator.Interaction)
	require.Equal(t, "1", creator.VideosCount)
	require.NotZero(t, creator.AddTs)
	require.NotZero(t, creator.LastModifyTs)

	var aweme struct {
		AwemeID          string
		Title            string
		Description      string
		CreateTime       int64
		LikedCount       string
		CommentCount     string
		ShareCount       string
		CollectedCount   string
		CoverURL         string
		VideoDownloadURL string
		SourceKeyword    string
		IsAIGenerated    int
		UserID           string
		SecUID           string
		ShortUserID      string
		UserUniqueID     string
		Nickname         string
		Avatar           string
		UserSignature    string
		IPLocation       string
		AddTs            int64
		LastModifyTs     int64
	}
	err = db.QueryRow(`
		SELECT aweme_id, title, `+"`desc`"+`, create_time, liked_count, comment_count,
		       share_count, collected_count, cover_url, video_download_url,
		       source_keyword, is_ai_generated, user_id, sec_uid, short_user_id,
		       user_unique_id, nickname, avatar, user_signature, ip_location,
		       add_ts, last_modify_ts
		FROM douyin_aweme
		WHERE aweme_id = ?
	`, "aweme_creator_e2e").Scan(
		&aweme.AwemeID,
		&aweme.Title,
		&aweme.Description,
		&aweme.CreateTime,
		&aweme.LikedCount,
		&aweme.CommentCount,
		&aweme.ShareCount,
		&aweme.CollectedCount,
		&aweme.CoverURL,
		&aweme.VideoDownloadURL,
		&aweme.SourceKeyword,
		&aweme.IsAIGenerated,
		&aweme.UserID,
		&aweme.SecUID,
		&aweme.ShortUserID,
		&aweme.UserUniqueID,
		&aweme.Nickname,
		&aweme.Avatar,
		&aweme.UserSignature,
		&aweme.IPLocation,
		&aweme.AddTs,
		&aweme.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "aweme_creator_e2e", aweme.AwemeID)
	require.Equal(t, "Douyin creator E2E mock aweme", aweme.Title)
	require.Equal(t, "mock douyin creator desc", aweme.Description)
	require.Equal(t, int64(1710000500), aweme.CreateTime)
	require.Equal(t, "388", aweme.LikedCount)
	require.Equal(t, "35", aweme.CommentCount)
	require.Equal(t, "33", aweme.ShareCount)
	require.Equal(t, "320", aweme.CollectedCount)
	require.Equal(t, "https://example.test/douyin-creator-cover.png", aweme.CoverURL)
	require.Equal(t, "https://example.test/douyin-creator-play.mp4", aweme.VideoDownloadURL)
	require.Equal(t, "", aweme.SourceKeyword)
	require.Equal(t, 0, aweme.IsAIGenerated)
	require.Equal(t, "dy-creator-author", aweme.UserID)
	require.Equal(t, "sec-dy-creator", aweme.SecUID)
	require.Equal(t, "445566", aweme.ShortUserID)
	require.Equal(t, "mock_douyin_creator_author", aweme.UserUniqueID)
	require.Equal(t, "mock douyin creator author", aweme.Nickname)
	require.Equal(t, "https://example.test/douyin-creator-avatar.png", aweme.Avatar)
	require.Equal(t, "creator author signature", aweme.UserSignature)
	require.Equal(t, "成都", aweme.IPLocation)
	require.NotZero(t, aweme.AddTs)
	require.NotZero(t, aweme.LastModifyTs)
}

func TestCLIDouyinHomefeedDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "dy",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "dy-unified-homefeed-client",
				Connected: true,
				Nicknames: map[string]string{"dy": "mock-douyin"},
				Cookies:   map[string]string{"dy": "sessionid=unified-homefeed; msToken=unified-homefeed"},
			},
		},
		PlatformHandler: newDouyinUnifiedHomefeedMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "dy",
		Keywords:             []string{"unused"},
		CrawlerType:          "homefeed",
		SaveDataOption:       "db",
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env:     mysqlServer.Env(),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	var row struct {
		AwemeID          string
		Title            string
		Description      string
		CreateTime       int64
		LikedCount       string
		CommentCount     string
		ShareCount       string
		CollectedCount   string
		CoverURL         string
		VideoDownloadURL string
		SourceKeyword    string
		IsAIGenerated    int
		UserID           string
		SecUID           string
		ShortUserID      string
		UserUniqueID     string
		Nickname         string
		Avatar           string
		UserSignature    string
		IPLocation       string
		AddTs            int64
		LastModifyTs     int64
	}
	err = db.QueryRow(`
		SELECT aweme_id, title, `+"`desc`"+`, create_time, liked_count, comment_count,
		       share_count, collected_count, cover_url, video_download_url,
		       source_keyword, is_ai_generated, user_id, sec_uid, short_user_id,
		       user_unique_id, nickname, avatar, user_signature, ip_location,
		       add_ts, last_modify_ts
		FROM douyin_aweme
		WHERE aweme_id = ?
	`, "aweme_homefeed_e2e").Scan(
		&row.AwemeID,
		&row.Title,
		&row.Description,
		&row.CreateTime,
		&row.LikedCount,
		&row.CommentCount,
		&row.ShareCount,
		&row.CollectedCount,
		&row.CoverURL,
		&row.VideoDownloadURL,
		&row.SourceKeyword,
		&row.IsAIGenerated,
		&row.UserID,
		&row.SecUID,
		&row.ShortUserID,
		&row.UserUniqueID,
		&row.Nickname,
		&row.Avatar,
		&row.UserSignature,
		&row.IPLocation,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "aweme_homefeed_e2e", row.AwemeID)
	require.Equal(t, "Douyin homefeed E2E mock aweme", row.Title)
	require.Equal(t, "mock douyin homefeed desc", row.Description)
	require.Equal(t, int64(1710000600), row.CreateTime)
	require.Equal(t, "488", row.LikedCount)
	require.Equal(t, "45", row.CommentCount)
	require.Equal(t, "43", row.ShareCount)
	require.Equal(t, "420", row.CollectedCount)
	require.Equal(t, "https://example.test/douyin-homefeed-cover.png", row.CoverURL)
	require.Equal(t, "https://example.test/douyin-homefeed-play.mp4", row.VideoDownloadURL)
	require.Equal(t, "", row.SourceKeyword)
	require.Equal(t, 0, row.IsAIGenerated)
	require.Equal(t, "dy-homefeed-author", row.UserID)
	require.Equal(t, "sec-dy-homefeed-author", row.SecUID)
	require.Equal(t, "556677", row.ShortUserID)
	require.Equal(t, "mock_douyin_homefeed_author", row.UserUniqueID)
	require.Equal(t, "mock douyin homefeed author", row.Nickname)
	require.Equal(t, "https://example.test/douyin-homefeed-avatar.png", row.Avatar)
	require.Equal(t, "homefeed signature", row.UserSignature)
	require.Equal(t, "重庆", row.IPLocation)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func TestCLIDouyinHotSearchDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "dy",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "dy-unified-hot-client",
				Connected: true,
				Nicknames: map[string]string{"dy": "mock-douyin"},
				Cookies:   map[string]string{"dy": "sessionid=unified-hot; msToken=unified-hot"},
			},
		},
		PlatformHandler: newDouyinUnifiedHotSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:                "dy",
		Keywords:                []string{"unused"},
		CrawlerType:             "hot_search",
		SaveDataOption:          "db",
		HotSearchMaxItemsCount:  1,
		HotSearchPerItemContent: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env:     mysqlServer.Env(),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	var row struct {
		AwemeID          string
		Title            string
		Description      string
		CreateTime       int64
		LikedCount       string
		CommentCount     string
		ShareCount       string
		CollectedCount   string
		AwemeURL         string
		CoverURL         string
		VideoDownloadURL string
		SourceKeyword    string
		IsAIGenerated    int
		UserID           string
		SecUID           string
		ShortUserID      string
		UserUniqueID     string
		Nickname         string
		Avatar           string
		UserSignature    string
		IPLocation       string
		AddTs            int64
		LastModifyTs     int64
	}
	err = db.QueryRow(`
		SELECT aweme_id, title, `+"`desc`"+`, create_time, liked_count, comment_count,
		       share_count, collected_count, aweme_url, cover_url, video_download_url,
		       source_keyword, is_ai_generated, user_id, sec_uid, short_user_id,
		       user_unique_id, nickname, avatar, user_signature, ip_location,
		       add_ts, last_modify_ts
		FROM douyin_aweme
		WHERE aweme_id = ?
	`, "aweme_hot_e2e").Scan(
		&row.AwemeID,
		&row.Title,
		&row.Description,
		&row.CreateTime,
		&row.LikedCount,
		&row.CommentCount,
		&row.ShareCount,
		&row.CollectedCount,
		&row.AwemeURL,
		&row.CoverURL,
		&row.VideoDownloadURL,
		&row.SourceKeyword,
		&row.IsAIGenerated,
		&row.UserID,
		&row.SecUID,
		&row.ShortUserID,
		&row.UserUniqueID,
		&row.Nickname,
		&row.Avatar,
		&row.UserSignature,
		&row.IPLocation,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "aweme_hot_e2e", row.AwemeID)
	require.Equal(t, "Douyin hot E2E mock aweme", row.Title)
	require.Equal(t, "mock douyin hot desc", row.Description)
	require.Equal(t, int64(1710000400), row.CreateTime)
	require.Equal(t, "288", row.LikedCount)
	require.Equal(t, "25", row.CommentCount)
	require.Equal(t, "23", row.ShareCount)
	require.Equal(t, "220", row.CollectedCount)
	require.Equal(t, "https://www.douyin.com/video/aweme_hot_e2e", row.AwemeURL)
	require.Equal(t, "https://example.test/douyin-hot-cover.png", row.CoverURL)
	require.Equal(t, "https://example.test/douyin-hot-play.mp4", row.VideoDownloadURL)
	require.Equal(t, "mock hot douyin", row.SourceKeyword)
	require.Equal(t, 0, row.IsAIGenerated)
	require.Equal(t, "dy-hot-author", row.UserID)
	require.Equal(t, "sec-dy-hot-author", row.SecUID)
	require.Equal(t, "334455", row.ShortUserID)
	require.Equal(t, "mock_douyin_hot_author", row.UserUniqueID)
	require.Equal(t, "mock douyin hot author", row.Nickname)
	require.Equal(t, "https://example.test/douyin-hot-avatar.png", row.Avatar)
	require.Equal(t, "hot signature", row.UserSignature)
	require.Equal(t, "深圳", row.IPLocation)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func newDouyinUnifiedSearchMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/general/search/single/":
			if r.URL.Query().Get("offset") != "0" {
				writeJSON(t, w, map[string]any{
					"status_code": 0,
					"data":        []map[string]any{},
					"extra":       map[string]string{"logid": "mock-logid-2"},
				})
				return
			}
			writeJSON(t, w, map[string]any{
				"status_code": 0,
				"data": []map[string]any{
					{"aweme_info": douyinMockAweme()},
				},
				"extra": map[string]string{"logid": "mock-logid"},
			})
		case "/aweme/v1/web/comment/list/":
			writeJSON(t, w, map[string]any{
				"status_code": 0,
				"has_more":    0,
				"cursor":      20,
				"comments": []map[string]any{
					douyinMockRootComment(),
				},
			})
		case "/aweme/v1/web/comment/list/reply/":
			writeJSON(t, w, map[string]any{
				"status_code": 0,
				"has_more":    0,
				"cursor":      20,
				"comments": []map[string]any{
					douyinMockSubComment(),
				},
			})
		default:
			t.Fatalf("unexpected douyin unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newDouyinUnifiedCreatorMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/user/profile/other/":
			require.Equal(t, "sec-dy-creator", r.URL.Query().Get("sec_user_id"))
			writeJSON(t, w, map[string]any{
				"status_code": 0,
				"user":        douyinMockCreator(),
			})
		case "/aweme/v1/web/aweme/post/":
			require.Equal(t, "sec-dy-creator", r.URL.Query().Get("sec_user_id"))
			require.Equal(t, "0", r.URL.Query().Get("max_cursor"))
			writeJSON(t, w, map[string]any{
				"status_code": 0,
				"has_more":    0,
				"max_cursor":  "18",
				"aweme_list": []map[string]any{
					douyinMockCreatorAweme(),
				},
			})
		default:
			t.Fatalf("unexpected douyin unified creator request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newDouyinUnifiedDetailMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/aweme/detail/":
			if got := r.URL.Query().Get("aweme_id"); got != "aweme_detail_e2e" {
				t.Fatalf("unexpected douyin detail aweme_id: %s", got)
			}
			writeJSON(t, w, map[string]any{
				"status_code":  0,
				"aweme_detail": douyinMockDetailAweme(),
			})
		default:
			t.Fatalf("unexpected douyin unified detail request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newDouyinUnifiedHomefeedMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/module/feed/":
			switch r.URL.Query().Get("refresh_index") {
			case "0":
				body, err := json.Marshal(douyinMockHomefeedAweme())
				require.NoError(t, err)
				writeJSON(t, w, map[string]any{
					"StatusCode": 0,
					"cards": []map[string]any{
						{
							"type":  1,
							"aweme": string(body),
						},
					},
				})
			case "20":
				writeJSON(t, w, map[string]any{
					"StatusCode": 1,
					"cards":      []map[string]any{},
				})
			default:
				t.Fatalf("unexpected douyin homefeed refresh_index: %s", r.URL.Query().Get("refresh_index"))
			}
		default:
			t.Fatalf("unexpected douyin unified homefeed request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newDouyinUnifiedHotSearchMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/aweme/v1/web/history/read/":
			writeJSON(t, w, map[string]any{"status_code": 0, "status_msg": ""})
		case "/aweme/v1/web/hot/search/list/":
			writeJSON(t, w, douyinHotSearchListPayload())
		case "/aweme/v1/web/hot/search/video/list/":
			require.Equal(t, "mock hot douyin", r.URL.Query().Get("hotword"))
			writeJSON(t, w, map[string]any{
				"status_code": 0,
				"aweme_list": []map[string]any{
					{"aweme_id": "aweme_hot_e2e"},
				},
			})
		case "/aweme/v1/web/aweme/detail/":
			require.Equal(t, "aweme_hot_e2e", r.URL.Query().Get("aweme_id"))
			writeJSON(t, w, map[string]any{
				"status_code":  0,
				"aweme_detail": douyinMockHotAweme(),
			})
		default:
			t.Fatalf("unexpected douyin unified hot_search request: %s %s", r.Method, r.URL.String())
		}
	})
}
