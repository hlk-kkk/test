// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/zhihu_mysql_e2e_test.go
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
	"net/http"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/e2e"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/cookiebridge"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestCLIZhihuSearchDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "zhihu",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "zhihu-unified-client",
				Connected: true,
				Nicknames: map[string]string{"zhihu": "mock-zhihu"},
				Cookies:   map[string]string{"zhihu": "z_c0=unified; q_c1=unified"},
			},
		},
		PlatformHandler: newZhihuUnifiedSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "zhihu",
		Keywords:             []string{"mock-zhihu"},
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
		ContentID     string
		ContentType   string
		ContentText   string
		ContentURL    string
		QuestionID    string
		Title         string
		Description   string
		CreatedTime   string
		UpdatedTime   string
		VoteupCount   int
		CommentCount  int
		SourceKeyword string
		UserID        string
		UserLink      string
		UserNickname  string
		UserAvatar    string
		UserURLToken  string
		AddTs         int64
		LastModifyTs  int64
	}
	err = db.QueryRow(`
		SELECT content_id, content_type, content_text, content_url, question_id, title, `+"`desc`"+`,
		       created_time, updated_time, voteup_count, comment_count, source_keyword,
		       user_id, user_link, user_nickname, user_avatar, user_url_token, add_ts, last_modify_ts
		FROM zhihu_content
		WHERE content_id = ?
	`, "zh_answer_e2e").Scan(
		&row.ContentID,
		&row.ContentType,
		&row.ContentText,
		&row.ContentURL,
		&row.QuestionID,
		&row.Title,
		&row.Description,
		&row.CreatedTime,
		&row.UpdatedTime,
		&row.VoteupCount,
		&row.CommentCount,
		&row.SourceKeyword,
		&row.UserID,
		&row.UserLink,
		&row.UserNickname,
		&row.UserAvatar,
		&row.UserURLToken,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "zh_answer_e2e", row.ContentID)
	require.Equal(t, "answer", row.ContentType)
	require.Equal(t, "Zhihu answer E2E content", row.ContentText)
	require.Equal(t, runner.BaseURL+"/question/zh_question_e2e/answer/zh_answer_e2e", row.ContentURL)
	require.Equal(t, "zh_question_e2e", row.QuestionID)
	require.Equal(t, "Zhihu E2E question", row.Title)
	require.Equal(t, "Zhihu answer description", row.Description)
	require.Equal(t, "1710001500", row.CreatedTime)
	require.Equal(t, "1710001600", row.UpdatedTime)
	require.Equal(t, 88, row.VoteupCount)
	require.Equal(t, 5, row.CommentCount)
	require.Equal(t, "mock-zhihu", row.SourceKeyword)
	require.Equal(t, "zh-author", row.UserID)
	require.Equal(t, runner.BaseURL+"/people/mock-zh-author", row.UserLink)
	require.Equal(t, "mock zhihu author", row.UserNickname)
	require.Equal(t, "https://example.test/zhihu-avatar.png", row.UserAvatar)
	require.Equal(t, "mock-zh-author", row.UserURLToken)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func TestCLIZhihuDetailDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "zhihu",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "zhihu-detail-unified-client",
				Connected: true,
				Nicknames: map[string]string{"zhihu": "mock-zhihu"},
				Cookies:   map[string]string{"zhihu": "z_c0=detail-unified; q_c1=detail-unified"},
			},
		},
		PlatformHandler: newZhihuUnifiedDetailMockHandler(t),
	})
	detailURL := runner.BaseURL + "/question/zh_question_detail_e2e/answer/zh_answer_detail_e2e"
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "zhihu",
		Keywords:             []string{"unused"},
		CrawlerType:          "detail",
		SaveDataOption:       "db",
		SpecifiedNoteIDs:     []string{detailURL},
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env: append(
			mysqlServer.Env(),
			"MCPRO_ZHIHU_URL="+runner.BaseURL,
			"MCPRO_ZHIHU_ZHUANLAN_URL="+runner.BaseURL,
		),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	var row struct {
		ContentID     string
		ContentType   string
		ContentText   string
		ContentURL    string
		QuestionID    string
		Title         string
		Description   string
		CreatedTime   string
		UpdatedTime   string
		VoteupCount   int
		CommentCount  int
		SourceKeyword string
		UserID        string
		UserLink      string
		UserNickname  string
		UserAvatar    string
		UserURLToken  string
		AddTs         int64
		LastModifyTs  int64
	}
	err = db.QueryRow(`
		SELECT content_id, content_type, content_text, content_url, question_id, title, `+"`desc`"+`,
		       created_time, updated_time, voteup_count, comment_count, source_keyword,
		       user_id, user_link, user_nickname, user_avatar, user_url_token, add_ts, last_modify_ts
		FROM zhihu_content
		WHERE content_id = ?
	`, "zh_answer_detail_e2e").Scan(
		&row.ContentID,
		&row.ContentType,
		&row.ContentText,
		&row.ContentURL,
		&row.QuestionID,
		&row.Title,
		&row.Description,
		&row.CreatedTime,
		&row.UpdatedTime,
		&row.VoteupCount,
		&row.CommentCount,
		&row.SourceKeyword,
		&row.UserID,
		&row.UserLink,
		&row.UserNickname,
		&row.UserAvatar,
		&row.UserURLToken,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "zh_answer_detail_e2e", row.ContentID)
	require.Equal(t, "answer", row.ContentType)
	require.Equal(t, "Zhihu detail E2E content", row.ContentText)
	require.Equal(t, runner.BaseURL+"/question/zh_question_detail_e2e/answer/zh_answer_detail_e2e", row.ContentURL)
	require.Equal(t, "zh_question_detail_e2e", row.QuestionID)
	require.Equal(t, "Zhihu detail question", row.Title)
	require.Equal(t, "Zhihu detail description", row.Description)
	require.Equal(t, "1710001700", row.CreatedTime)
	require.Equal(t, "1710001800", row.UpdatedTime)
	require.Equal(t, 188, row.VoteupCount)
	require.Equal(t, 15, row.CommentCount)
	require.Equal(t, "", row.SourceKeyword)
	require.Equal(t, "zh-detail-author", row.UserID)
	require.Equal(t, runner.BaseURL+"/people/mock-zh-detail-author", row.UserLink)
	require.Equal(t, "mock zhihu detail author", row.UserNickname)
	require.Equal(t, "https://example.test/zhihu-detail-avatar.png", row.UserAvatar)
	require.Equal(t, "mock-zh-detail-author", row.UserURLToken)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func TestCLIZhihuDetailCommentsDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "zhihu",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "zhihu-comments-unified-client",
				Connected: true,
				Nicknames: map[string]string{"zhihu": "mock-zhihu"},
				Cookies:   map[string]string{"zhihu": "z_c0=comments-unified; q_c1=comments-unified"},
			},
		},
		PlatformHandler: newZhihuUnifiedDetailMockHandler(t),
	})
	detailURL := runner.BaseURL + "/question/zh_question_detail_e2e/answer/zh_answer_detail_e2e"
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "zhihu",
		Keywords:             []string{"unused"},
		CrawlerType:          "detail",
		SaveDataOption:       "db",
		SpecifiedNoteIDs:     []string{detailURL},
		CrawlerMaxNotesCount: 1,
		EnableGetComments:    true,
		EnableGetSubComments: true,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env: append(
			mysqlServer.Env(),
			"MCPRO_ZHIHU_URL="+runner.BaseURL,
			"MCPRO_ZHIHU_ZHUANLAN_URL="+runner.BaseURL,
		),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	rows, err := db.Query(`
		SELECT comment_id, parent_comment_id, content, publish_time, ip_location,
		       sub_comment_count, like_count, dislike_count, content_id, content_type,
		       user_id, user_link, user_nickname, user_avatar, add_ts, last_modify_ts
		FROM zhihu_comment
		ORDER BY comment_id
	`)
	require.NoError(t, err, "command output:\n%s", output)
	defer rows.Close()

	type commentRow struct {
		CommentID       string
		ParentCommentID string
		Content         string
		PublishTime     string
		IPLocation      string
		SubCommentCount int
		LikeCount       int
		DislikeCount    int
		ContentID       string
		ContentType     string
		UserID          string
		UserLink        string
		UserNickname    string
		UserAvatar      string
		AddTs           int64
		LastModifyTs    int64
	}
	var comments []commentRow
	for rows.Next() {
		var row commentRow
		require.NoError(t, rows.Scan(
			&row.CommentID,
			&row.ParentCommentID,
			&row.Content,
			&row.PublishTime,
			&row.IPLocation,
			&row.SubCommentCount,
			&row.LikeCount,
			&row.DislikeCount,
			&row.ContentID,
			&row.ContentType,
			&row.UserID,
			&row.UserLink,
			&row.UserNickname,
			&row.UserAvatar,
			&row.AddTs,
			&row.LastModifyTs,
		))
		comments = append(comments, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, comments, 2, "command output:\n%s", output)

	root := comments[0]
	require.Equal(t, "zh_comment_root", root.CommentID)
	require.Equal(t, "", root.ParentCommentID)
	require.Equal(t, "root zhihu comment", root.Content)
	require.Equal(t, "1710001900", root.PublishTime)
	require.Equal(t, "广东", root.IPLocation)
	require.Equal(t, 1, root.SubCommentCount)
	require.Equal(t, 12, root.LikeCount)
	require.Equal(t, 2, root.DislikeCount)
	require.Equal(t, "zh_answer_detail_e2e", root.ContentID)
	require.Equal(t, "answer", root.ContentType)
	require.Equal(t, "zh-commenter", root.UserID)
	require.Equal(t, runner.BaseURL+"/people/mock-zh-commenter", root.UserLink)
	require.Equal(t, "mock zhihu commenter", root.UserNickname)
	require.Equal(t, "https://example.test/zhihu-comment-avatar.png", root.UserAvatar)
	require.NotZero(t, root.AddTs)
	require.NotZero(t, root.LastModifyTs)

	sub := comments[1]
	require.Equal(t, "zh_comment_sub", sub.CommentID)
	require.Equal(t, "zh_comment_root", sub.ParentCommentID)
	require.Equal(t, "sub zhihu comment", sub.Content)
	require.Equal(t, "1710001910", sub.PublishTime)
	require.Equal(t, "深圳", sub.IPLocation)
	require.Equal(t, 0, sub.SubCommentCount)
	require.Equal(t, 3, sub.LikeCount)
	require.Equal(t, 0, sub.DislikeCount)
	require.Equal(t, "zh_answer_detail_e2e", sub.ContentID)
	require.Equal(t, "answer", sub.ContentType)
	require.Equal(t, "zh-sub-commenter", sub.UserID)
	require.Equal(t, runner.BaseURL+"/people/mock-zh-sub-commenter", sub.UserLink)
	require.Equal(t, "mock zhihu sub commenter", sub.UserNickname)
	require.Equal(t, "https://example.test/zhihu-sub-comment-avatar.png", sub.UserAvatar)
	require.NotZero(t, sub.AddTs)
	require.NotZero(t, sub.LastModifyTs)
}

func TestCLIZhihuCreatorDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "zhihu",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "zhihu-creator-unified-client",
				Connected: true,
				Nicknames: map[string]string{"zhihu": "mock-zhihu"},
				Cookies:   map[string]string{"zhihu": "z_c0=creator-unified; q_c1=creator-unified"},
			},
		},
		PlatformHandler: newZhihuUnifiedCreatorMockHandler(t),
	})
	creatorURL := runner.BaseURL + "/people/mock-zh-creator"
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "zhihu",
		Keywords:             []string{"unused"},
		CrawlerType:          "creator",
		SaveDataOption:       "db",
		SpecifiedCreatorIDs:  []string{creatorURL},
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env: append(
			mysqlServer.Env(),
			"MCPRO_ZHIHU_URL="+runner.BaseURL,
			"MCPRO_ZHIHU_ZHUANLAN_URL="+runner.BaseURL,
		),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()

	var creator struct {
		UserID         string
		UserLink       string
		UserNickname   string
		UserAvatar     string
		URLToken       string
		Gender         string
		IPLocation     string
		Follows        int
		Fans           int
		AnwserCount    int
		VideoCount     int
		QuestionCount  int
		ArticleCount   int
		ColumnCount    int
		GetVoteupCount int
		AddTs          int64
		LastModifyTs   int64
	}
	err = db.QueryRow(`
		SELECT user_id, user_link, user_nickname, user_avatar, url_token, gender,
		       ip_location, follows, fans, anwser_count, video_count, question_count,
		       article_count, column_count, get_voteup_count, add_ts, last_modify_ts
		FROM zhihu_creator
		WHERE user_id = ?
	`, "zh-creator-user").Scan(
		&creator.UserID,
		&creator.UserLink,
		&creator.UserNickname,
		&creator.UserAvatar,
		&creator.URLToken,
		&creator.Gender,
		&creator.IPLocation,
		&creator.Follows,
		&creator.Fans,
		&creator.AnwserCount,
		&creator.VideoCount,
		&creator.QuestionCount,
		&creator.ArticleCount,
		&creator.ColumnCount,
		&creator.GetVoteupCount,
		&creator.AddTs,
		&creator.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "zh-creator-user", creator.UserID)
	require.Equal(t, runner.BaseURL+"/people/mock-zh-creator", creator.UserLink)
	require.Equal(t, "mock zhihu creator", creator.UserNickname)
	require.Equal(t, "https://example.test/zhihu-creator-avatar.png", creator.UserAvatar)
	require.Equal(t, "mock-zh-creator", creator.URLToken)
	require.Equal(t, "男", creator.Gender)
	require.Equal(t, "浙江", creator.IPLocation)
	require.Equal(t, 12, creator.Follows)
	require.Equal(t, 345, creator.Fans)
	require.Equal(t, 7, creator.AnwserCount)
	require.Equal(t, 2, creator.VideoCount)
	require.Equal(t, 3, creator.QuestionCount)
	require.Equal(t, 4, creator.ArticleCount)
	require.Equal(t, 1, creator.ColumnCount)
	require.Equal(t, 9876, creator.GetVoteupCount)
	require.NotZero(t, creator.AddTs)
	require.NotZero(t, creator.LastModifyTs)

	var content struct {
		ContentID     string
		ContentType   string
		ContentText   string
		ContentURL    string
		QuestionID    string
		Title         string
		Description   string
		CreatedTime   string
		UpdatedTime   string
		VoteupCount   int
		CommentCount  int
		SourceKeyword string
		UserID        string
		UserLink      string
		UserNickname  string
		UserAvatar    string
		UserURLToken  string
		AddTs         int64
		LastModifyTs  int64
	}
	err = db.QueryRow(`
		SELECT content_id, content_type, content_text, content_url, question_id, title, `+"`desc`"+`,
		       created_time, updated_time, voteup_count, comment_count, source_keyword,
		       user_id, user_link, user_nickname, user_avatar, user_url_token, add_ts, last_modify_ts
		FROM zhihu_content
		WHERE content_id = ?
	`, "zh_creator_answer_e2e").Scan(
		&content.ContentID,
		&content.ContentType,
		&content.ContentText,
		&content.ContentURL,
		&content.QuestionID,
		&content.Title,
		&content.Description,
		&content.CreatedTime,
		&content.UpdatedTime,
		&content.VoteupCount,
		&content.CommentCount,
		&content.SourceKeyword,
		&content.UserID,
		&content.UserLink,
		&content.UserNickname,
		&content.UserAvatar,
		&content.UserURLToken,
		&content.AddTs,
		&content.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "zh_creator_answer_e2e", content.ContentID)
	require.Equal(t, "answer", content.ContentType)
	require.Equal(t, "Zhihu creator answer E2E content", content.ContentText)
	require.Equal(t, runner.BaseURL+"/question/zh_creator_question_e2e/answer/zh_creator_answer_e2e", content.ContentURL)
	require.Equal(t, "zh_creator_question_e2e", content.QuestionID)
	require.Equal(t, "Zhihu creator question", content.Title)
	require.Equal(t, "Zhihu creator answer description", content.Description)
	require.Equal(t, "1710002000", content.CreatedTime)
	require.Equal(t, "1710002100", content.UpdatedTime)
	require.Equal(t, 288, content.VoteupCount)
	require.Equal(t, 25, content.CommentCount)
	require.Equal(t, "", content.SourceKeyword)
	require.Equal(t, "zh-creator-user", content.UserID)
	require.Equal(t, runner.BaseURL+"/people/mock-zh-creator", content.UserLink)
	require.Equal(t, "mock zhihu creator", content.UserNickname)
	require.Equal(t, "https://example.test/zhihu-creator-avatar.png", content.UserAvatar)
	require.Equal(t, "mock-zh-creator", content.UserURLToken)
	require.NotZero(t, content.AddTs)
	require.NotZero(t, content.LastModifyTs)
}

func TestCLIZhihuHomefeedDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "zhihu",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "zhihu-homefeed-unified-client",
				Connected: true,
				Nicknames: map[string]string{"zhihu": "mock-zhihu"},
				Cookies:   map[string]string{"zhihu": "z_c0=homefeed-unified; q_c1=homefeed-unified"},
			},
		},
		PlatformHandler: newZhihuUnifiedHomefeedMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "zhihu",
		Keywords:             []string{"unused"},
		CrawlerType:          "homefeed",
		SaveDataOption:       "db",
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env: append(
			mysqlServer.Env(),
			"MCPRO_ZHIHU_URL="+runner.BaseURL,
			"MCPRO_ZHIHU_ZHUANLAN_URL="+runner.BaseURL,
		),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	var row struct {
		ContentID     string
		ContentType   string
		ContentText   string
		ContentURL    string
		QuestionID    string
		Title         string
		Description   string
		CreatedTime   string
		UpdatedTime   string
		VoteupCount   int
		CommentCount  int
		SourceKeyword string
		UserID        string
		UserLink      string
		UserNickname  string
		UserAvatar    string
		UserURLToken  string
		AddTs         int64
		LastModifyTs  int64
	}
	err = db.QueryRow(`
		SELECT content_id, content_type, content_text, content_url, question_id, title, `+"`desc`"+`,
		       created_time, updated_time, voteup_count, comment_count, source_keyword,
		       user_id, user_link, user_nickname, user_avatar, user_url_token, add_ts, last_modify_ts
		FROM zhihu_content
		WHERE content_id = ?
	`, "zh_homefeed_answer_e2e").Scan(
		&row.ContentID,
		&row.ContentType,
		&row.ContentText,
		&row.ContentURL,
		&row.QuestionID,
		&row.Title,
		&row.Description,
		&row.CreatedTime,
		&row.UpdatedTime,
		&row.VoteupCount,
		&row.CommentCount,
		&row.SourceKeyword,
		&row.UserID,
		&row.UserLink,
		&row.UserNickname,
		&row.UserAvatar,
		&row.UserURLToken,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "zh_homefeed_answer_e2e", row.ContentID)
	require.Equal(t, "answer", row.ContentType)
	require.Equal(t, "Zhihu homefeed answer E2E content", row.ContentText)
	require.Equal(t, runner.BaseURL+"/question/zh_homefeed_question_e2e/answer/zh_homefeed_answer_e2e", row.ContentURL)
	require.Equal(t, "zh_homefeed_question_e2e", row.QuestionID)
	require.Equal(t, "Zhihu homefeed question", row.Title)
	require.Equal(t, "Zhihu homefeed answer description", row.Description)
	require.Equal(t, "1710002200", row.CreatedTime)
	require.Equal(t, "1710002300", row.UpdatedTime)
	require.Equal(t, 388, row.VoteupCount)
	require.Equal(t, 35, row.CommentCount)
	require.Equal(t, "", row.SourceKeyword)
	require.Equal(t, "zh-homefeed-author", row.UserID)
	require.Equal(t, runner.BaseURL+"/people/mock-zh-homefeed-author", row.UserLink)
	require.Equal(t, "mock zhihu homefeed author", row.UserNickname)
	require.Equal(t, "https://example.test/zhihu-homefeed-avatar.png", row.UserAvatar)
	require.Equal(t, "mock-zh-homefeed-author", row.UserURLToken)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func TestCLIZhihuHotSearchDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "zhihu",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "zhihu-hot-unified-client",
				Connected: true,
				Nicknames: map[string]string{"zhihu": "mock-zhihu"},
				Cookies:   map[string]string{"zhihu": "z_c0=hot-unified; q_c1=hot-unified"},
			},
		},
		PlatformHandler: newZhihuUnifiedHotSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:                "zhihu",
		CrawlerType:             "hot_search",
		SaveDataOption:          "db",
		HotSearchMaxItemsCount:  1,
		HotSearchPerItemContent: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env: append(
			mysqlServer.Env(),
			"MCPRO_ZHIHU_URL="+runner.BaseURL,
			"MCPRO_ZHIHU_ZHUANLAN_URL="+runner.BaseURL,
		),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()

	var content struct {
		ContentID     string
		ContentType   string
		ContentText   string
		ContentURL    string
		QuestionID    string
		Title         string
		Description   string
		CreatedTime   string
		UpdatedTime   string
		VoteupCount   int
		CommentCount  int
		SourceKeyword string
		UserID        string
		UserLink      string
		UserNickname  string
		UserAvatar    string
		UserURLToken  string
	}
	err = db.QueryRow(`
		SELECT content_id, content_type, content_text, content_url, question_id, title, `+"`desc`"+`,
		       created_time, updated_time, voteup_count, comment_count, source_keyword,
		       user_id, user_link, user_nickname, user_avatar, user_url_token
		FROM zhihu_content
		WHERE content_id = ?
	`, "zh_hot_answer_e2e").Scan(
		&content.ContentID,
		&content.ContentType,
		&content.ContentText,
		&content.ContentURL,
		&content.QuestionID,
		&content.Title,
		&content.Description,
		&content.CreatedTime,
		&content.UpdatedTime,
		&content.VoteupCount,
		&content.CommentCount,
		&content.SourceKeyword,
		&content.UserID,
		&content.UserLink,
		&content.UserNickname,
		&content.UserAvatar,
		&content.UserURLToken,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "zh_hot_answer_e2e", content.ContentID)
	require.Equal(t, "answer", content.ContentType)
	require.Equal(t, "Zhihu hot answer E2E content", content.ContentText)
	require.Equal(t, runner.BaseURL+"/question/zh_hot_question_e2e/answer/zh_hot_answer_e2e", content.ContentURL)
	require.Equal(t, "zh_hot_question_e2e", content.QuestionID)
	require.Equal(t, "Zhihu hot question", content.Title)
	require.Equal(t, "Zhihu hot answer description", content.Description)
	require.Equal(t, "1710002400", content.CreatedTime)
	require.Equal(t, "1710002500", content.UpdatedTime)
	require.Equal(t, 488, content.VoteupCount)
	require.Equal(t, 45, content.CommentCount)
	require.Equal(t, "Zhihu hot question", content.SourceKeyword)
	require.Equal(t, "zh-hot-author", content.UserID)
	require.Equal(t, runner.BaseURL+"/people/mock-zh-hot-author", content.UserLink)
	require.Equal(t, "mock zhihu hot author", content.UserNickname)
	require.Equal(t, "https://example.test/zhihu-hot-avatar.png", content.UserAvatar)
	require.Equal(t, "mock-zh-hot-author", content.UserURLToken)

	var item struct {
		HotItemID    string
		Platform     string
		BoardName    string
		BoardType    string
		RankNo       int
		Title        string
		Keyword      string
		HotValue     string
		Badge        string
		TopicID      string
		TopicURL     string
		RawJSON      string
		SnapshotTs   int64
		AddTs        int64
		LastModifyTs int64
	}
	err = db.QueryRow(`
		SELECT hot_item_id, platform, board_name, board_type, rank_no, title, keyword,
		       hot_value, badge, topic_id, topic_url, raw_json, snapshot_ts, add_ts,
		       last_modify_ts
		FROM platform_hot_search_item
		WHERE platform = ? AND rank_no = ?
	`, "zhihu", 1).Scan(
		&item.HotItemID,
		&item.Platform,
		&item.BoardName,
		&item.BoardType,
		&item.RankNo,
		&item.Title,
		&item.Keyword,
		&item.HotValue,
		&item.Badge,
		&item.TopicID,
		&item.TopicURL,
		&item.RawJSON,
		&item.SnapshotTs,
		&item.AddTs,
		&item.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.NotEmpty(t, item.HotItemID)
	require.Equal(t, "zhihu", item.Platform)
	require.Equal(t, "hot_search", item.BoardName)
	require.Equal(t, "question", item.BoardType)
	require.Equal(t, 1, item.RankNo)
	require.Equal(t, "Zhihu hot question", item.Title)
	require.Equal(t, "Zhihu hot question", item.Keyword)
	require.Equal(t, "12345", item.HotValue)
	require.Equal(t, "热", item.Badge)
	require.Equal(t, "zh_hot_question_e2e", item.TopicID)
	require.Equal(t, "https://www.zhihu.com/question/zh_hot_question_e2e", item.TopicURL)
	require.Contains(t, item.RawJSON, "zh_hot_question_e2e")
	require.NotZero(t, item.SnapshotTs)
	require.NotZero(t, item.AddTs)
	require.NotZero(t, item.LastModifyTs)

	var relation struct {
		RelID         string
		HotItemID     string
		Platform      string
		ContentID     string
		ContentType   string
		ContentTable  string
		RankNo        int
		SourceKeyword string
		AddTs         int64
		LastModifyTs  int64
	}
	err = db.QueryRow(`
		SELECT rel_id, hot_item_id, platform, content_id, content_type, content_table,
		       rank_no, source_keyword, add_ts, last_modify_ts
		FROM platform_hot_search_content_rel
		WHERE platform = ? AND content_id = ?
	`, "zhihu", "zh_hot_answer_e2e").Scan(
		&relation.RelID,
		&relation.HotItemID,
		&relation.Platform,
		&relation.ContentID,
		&relation.ContentType,
		&relation.ContentTable,
		&relation.RankNo,
		&relation.SourceKeyword,
		&relation.AddTs,
		&relation.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.NotEmpty(t, relation.RelID)
	require.Equal(t, item.HotItemID, relation.HotItemID)
	require.Equal(t, "zhihu", relation.Platform)
	require.Equal(t, "zh_hot_answer_e2e", relation.ContentID)
	require.Equal(t, "answer", relation.ContentType)
	require.Equal(t, "zhihu_content", relation.ContentTable)
	require.Equal(t, 1, relation.RankNo)
	require.Equal(t, "Zhihu hot question", relation.SourceKeyword)
	require.NotZero(t, relation.AddTs)
	require.NotZero(t, relation.LastModifyTs)
}

func newZhihuUnifiedSearchMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v4/me":
			writeJSON(t, w, map[string]any{"uid": "zh-current-user", "name": "mock zhihu account"})
		case "/api/v4/search_v3":
			require.Equal(t, "mock-zhihu", r.URL.Query().Get("q"))
			if r.URL.Query().Get("offset") != "0" {
				writeJSON(t, w, map[string]any{"data": []map[string]any{}})
				return
			}
			writeJSON(t, w, zhihuSearchPayload())
		default:
			t.Fatalf("unexpected zhihu unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newZhihuUnifiedCreatorMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/me":
			w.Header().Set("Content-Type", "application/json")
			writeJSON(t, w, map[string]any{"uid": "zh-current-user", "name": "mock zhihu account"})
		case "/people/mock-zh-creator":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(zhihuCreatorHTML()))
		case "/api/v4/members/mock-zh-creator/answers":
			w.Header().Set("Content-Type", "application/json")
			writeJSON(t, w, zhihuCreatorAnswersPayload())
		default:
			t.Fatalf("unexpected zhihu creator unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newZhihuUnifiedHomefeedMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v4/me":
			writeJSON(t, w, map[string]any{"uid": "zh-current-user", "name": "mock zhihu account"})
		case "/api/v3/feed/topstory/recommend":
			switch r.URL.Query().Get("page_number") {
			case "1":
				writeJSON(t, w, zhihuHomefeedPayload("http://"+r.Host))
			case "2":
				writeJSON(t, w, map[string]any{
					"paging": map[string]any{"is_end": true, "next": ""},
					"data":   []map[string]any{},
				})
			default:
				t.Fatalf("unexpected zhihu homefeed page_number: %s (%s)", r.URL.Query().Get("page_number"), r.URL.RawQuery)
			}
		default:
			t.Fatalf("unexpected zhihu homefeed unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newZhihuUnifiedDetailMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/me":
			w.Header().Set("Content-Type", "application/json")
			writeJSON(t, w, map[string]any{"uid": "zh-current-user", "name": "mock zhihu account"})
		case "/question/zh_question_detail_e2e/answer/zh_answer_detail_e2e":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(zhihuDetailHTML()))
		case "/api/v4/comment_v5/answers/zh_answer_detail_e2e/root_comment":
			w.Header().Set("Content-Type", "application/json")
			writeJSON(t, w, zhihuRootCommentsPayload())
		case "/api/v4/comment_v5/comment/zh_comment_root/child_comment":
			w.Header().Set("Content-Type", "application/json")
			writeJSON(t, w, zhihuChildCommentsPayload())
		default:
			t.Fatalf("unexpected zhihu detail unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newZhihuUnifiedHotSearchMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v4/me":
			writeJSON(t, w, map[string]any{"uid": "zh-current-user", "name": "mock zhihu account"})
		case "/api/v4/creators/rank/hot":
			writeJSON(t, w, zhihuHotSearchPayload())
		case "/api/v4/questions/zh_hot_question_e2e/feeds":
			writeJSON(t, w, zhihuHotQuestionFeedsPayload())
		default:
			t.Fatalf("unexpected zhihu hot_search unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}
