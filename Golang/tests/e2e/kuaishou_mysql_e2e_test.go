// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/kuaishou_mysql_e2e_test.go
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
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/e2e"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/cookiebridge"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestCLIKuaishouSearchDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "ks",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "ks-unified-client",
				Connected: true,
				Nicknames: map[string]string{"ks": "mock-kuaishou"},
				Cookies:   map[string]string{"ks": "kpf=PC_WEB; clientid=unified; did=unified"},
			},
		},
		PlatformHandler: newKuaishouUnifiedSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "ks",
		Keywords:             []string{"mock-ks"},
		CrawlerType:          "search",
		SaveDataOption:       "db",
		CrawlerMaxNotesCount: 20,
		PlatformEndpointOverrides: map[string]string{
			"kuaishouApiUrl":     runner.BaseURL + "/graphql",
			"kuaishouRestApiUrl": runner.BaseURL,
		},
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env: append(
			mysqlServer.Env(),
			"MCPRO_KUAISHOU_API_URL="+runner.BaseURL+"/graphql",
			"MCPRO_KUAISHOU_REST_API_URL="+runner.BaseURL,
		),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	var row struct {
		VideoID       string
		VideoType     string
		Title         string
		Description   string
		CreateTime    string
		LikedCount    string
		ViewdCount    string
		VideoURL      string
		VideoCoverURL string
		VideoPlayURL  string
		SourceKeyword string
		UserID        string
		Nickname      string
		Avatar        string
	}
	err = db.QueryRow(`
		SELECT video_id, video_type, title, `+"`desc`"+`, create_time, liked_count, viewd_count,
		       video_url, video_cover_url, video_play_url, source_keyword, user_id, nickname, avatar
		FROM kuaishou_video
		WHERE video_id = ?
	`, "ks_video_e2e").Scan(
		&row.VideoID,
		&row.VideoType,
		&row.Title,
		&row.Description,
		&row.CreateTime,
		&row.LikedCount,
		&row.ViewdCount,
		&row.VideoURL,
		&row.VideoCoverURL,
		&row.VideoPlayURL,
		&row.SourceKeyword,
		&row.UserID,
		&row.Nickname,
		&row.Avatar,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "ks_video_e2e", row.VideoID)
	require.Equal(t, "photo", row.VideoType)
	require.Equal(t, "Kuaishou E2E mock video", row.Title)
	require.Equal(t, "Kuaishou E2E mock video", row.Description)
	require.Equal(t, "1710000700", row.CreateTime)
	require.Equal(t, "66", row.LikedCount)
	require.Equal(t, "777", row.ViewdCount)
	require.Equal(t, "https://www.kuaishou.com/short-video/ks_video_e2e", row.VideoURL)
	require.Equal(t, "https://example.test/ks-cover.png", row.VideoCoverURL)
	require.Equal(t, "https://example.test/ks-play.mp4", row.VideoPlayURL)
	require.Equal(t, "mock-ks", row.SourceKeyword)
	require.Equal(t, "ks-author", row.UserID)
	require.Equal(t, "mock kuaishou author", row.Nickname)
	require.Equal(t, "https://example.test/ks-avatar.png", row.Avatar)
}

func TestCLIKuaishouDetailDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "ks",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "ks-unified-detail-client",
				Connected: true,
				Nicknames: map[string]string{"ks": "mock-kuaishou"},
				Cookies:   map[string]string{"ks": "kpf=PC_WEB; clientid=detail-unified; did=detail-unified"},
			},
		},
		PlatformHandler: newKuaishouUnifiedDetailMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "ks",
		Keywords:             []string{"mock-ks"},
		CrawlerType:          "detail",
		SaveDataOption:       "db",
		SpecifiedNoteIDs:     []string{"ks_detail_e2e"},
		CrawlerMaxNotesCount: 1,
		PlatformEndpointOverrides: map[string]string{
			"kuaishouApiUrl":     runner.BaseURL + "/graphql",
			"kuaishouRestApiUrl": runner.BaseURL,
		},
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env: append(
			mysqlServer.Env(),
			"MCPRO_KUAISHOU_API_URL="+runner.BaseURL+"/graphql",
			"MCPRO_KUAISHOU_REST_API_URL="+runner.BaseURL,
		),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	var row struct {
		VideoID       string
		VideoType     string
		Title         string
		Description   string
		CreateTime    string
		LikedCount    string
		ViewdCount    string
		VideoURL      string
		VideoCoverURL string
		VideoPlayURL  string
		SourceKeyword string
		UserID        string
		Nickname      string
		Avatar        string
		AddTs         int64
		LastModifyTs  int64
	}
	err = db.QueryRow(`
		SELECT video_id, video_type, title, `+"`desc`"+`, create_time, liked_count, viewd_count,
		       video_url, video_cover_url, video_play_url, source_keyword,
		       user_id, nickname, avatar, add_ts, last_modify_ts
		FROM kuaishou_video
		WHERE video_id = ?
	`, "ks_detail_e2e").Scan(
		&row.VideoID,
		&row.VideoType,
		&row.Title,
		&row.Description,
		&row.CreateTime,
		&row.LikedCount,
		&row.ViewdCount,
		&row.VideoURL,
		&row.VideoCoverURL,
		&row.VideoPlayURL,
		&row.SourceKeyword,
		&row.UserID,
		&row.Nickname,
		&row.Avatar,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "ks_detail_e2e", row.VideoID)
	require.Equal(t, "photo", row.VideoType)
	require.Equal(t, "Kuaishou detail E2E mock video", row.Title)
	require.Equal(t, "Kuaishou detail E2E mock video", row.Description)
	require.Equal(t, "1710000800", row.CreateTime)
	require.Equal(t, "166", row.LikedCount)
	require.Equal(t, "1777", row.ViewdCount)
	require.Equal(t, "https://www.kuaishou.com/short-video/ks_detail_e2e", row.VideoURL)
	require.Equal(t, "https://example.test/ks-detail-cover.png", row.VideoCoverURL)
	require.Equal(t, "https://example.test/ks-detail-play.mp4", row.VideoPlayURL)
	require.Equal(t, "", row.SourceKeyword)
	require.Equal(t, "ks-detail-author", row.UserID)
	require.Equal(t, "mock kuaishou detail author", row.Nickname)
	require.Equal(t, "https://example.test/ks-detail-avatar.png", row.Avatar)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func TestCLIKuaishouDetailCommentsDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "ks",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "ks-comments-unified-client",
				Connected: true,
				Nicknames: map[string]string{"ks": "mock-kuaishou"},
				Cookies:   map[string]string{"ks": "kpf=PC_WEB; clientid=comments-unified; did=comments-unified"},
			},
		},
		PlatformHandler: newKuaishouUnifiedDetailMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "ks",
		Keywords:             []string{"mock-ks"},
		CrawlerType:          "detail",
		SaveDataOption:       "db",
		SpecifiedNoteIDs:     []string{"ks_detail_e2e"},
		CrawlerMaxNotesCount: 1,
		EnableGetComments:    true,
		EnableGetSubComments: true,
		PlatformEndpointOverrides: map[string]string{
			"kuaishouApiUrl":     runner.BaseURL + "/graphql",
			"kuaishouRestApiUrl": runner.BaseURL,
		},
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env: append(
			mysqlServer.Env(),
			"MCPRO_KUAISHOU_API_URL="+runner.BaseURL+"/graphql",
			"MCPRO_KUAISHOU_REST_API_URL="+runner.BaseURL,
		),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	rows, err := db.Query(`
		SELECT comment_id, parent_comment_id, video_id, content, create_time,
		       sub_comment_count, like_count, user_id, nickname, avatar,
		       add_ts, last_modify_ts
		FROM kuaishou_video_comment
		ORDER BY comment_id
	`)
	require.NoError(t, err, "command output:\n%s", output)
	defer rows.Close()

	type commentRow struct {
		CommentID       string
		ParentCommentID string
		VideoID         string
		Content         string
		CreateTime      string
		SubCommentCount string
		LikeCount       string
		UserID          string
		Nickname        string
		Avatar          string
		AddTs           int64
		LastModifyTs    int64
	}
	var comments []commentRow
	for rows.Next() {
		var row commentRow
		require.NoError(t, rows.Scan(
			&row.CommentID,
			&row.ParentCommentID,
			&row.VideoID,
			&row.Content,
			&row.CreateTime,
			&row.SubCommentCount,
			&row.LikeCount,
			&row.UserID,
			&row.Nickname,
			&row.Avatar,
			&row.AddTs,
			&row.LastModifyTs,
		))
		comments = append(comments, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, comments, 2, "command output:\n%s", output)

	root := comments[0]
	require.Equal(t, "9001", root.CommentID)
	require.Equal(t, "", root.ParentCommentID)
	require.Equal(t, "ks_detail_e2e", root.VideoID)
	require.Equal(t, "root kuaishou comment", root.Content)
	require.Equal(t, "1710000900", root.CreateTime)
	require.Equal(t, "1", root.SubCommentCount)
	require.Equal(t, "12", root.LikeCount)
	require.Equal(t, "ks-commenter", root.UserID)
	require.Equal(t, "mock ks commenter", root.Nickname)
	require.Equal(t, "https://example.test/ks-comment-avatar.png", root.Avatar)
	require.NotZero(t, root.AddTs)
	require.NotZero(t, root.LastModifyTs)

	sub := comments[1]
	require.Equal(t, "9002", sub.CommentID)
	require.Equal(t, "9001", sub.ParentCommentID)
	require.Equal(t, "ks_detail_e2e", sub.VideoID)
	require.Equal(t, "sub kuaishou comment", sub.Content)
	require.Equal(t, "1710000910", sub.CreateTime)
	require.Equal(t, "0", sub.SubCommentCount)
	require.Equal(t, "3", sub.LikeCount)
	require.Equal(t, "ks-sub-commenter", sub.UserID)
	require.Equal(t, "mock ks sub commenter", sub.Nickname)
	require.Equal(t, "https://example.test/ks-sub-comment-avatar.png", sub.Avatar)
	require.NotZero(t, sub.AddTs)
	require.NotZero(t, sub.LastModifyTs)
}

func TestCLIKuaishouCreatorDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "ks",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "ks-unified-creator-client",
				Connected: true,
				Nicknames: map[string]string{"ks": "mock-kuaishou"},
				Cookies:   map[string]string{"ks": "kpf=PC_WEB; clientid=creator-unified; did=creator-unified"},
			},
		},
		PlatformHandler: newKuaishouUnifiedCreatorMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "ks",
		Keywords:             []string{"mock-ks"},
		CrawlerType:          "creator",
		SaveDataOption:       "db",
		SpecifiedCreatorIDs:  []string{"ks-creator-user"},
		CrawlerMaxNotesCount: 1,
		PlatformEndpointOverrides: map[string]string{
			"kuaishouApiUrl":     runner.BaseURL + "/graphql",
			"kuaishouRestApiUrl": runner.BaseURL,
		},
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env: append(
			mysqlServer.Env(),
			"MCPRO_KUAISHOU_API_URL="+runner.BaseURL+"/graphql",
			"MCPRO_KUAISHOU_REST_API_URL="+runner.BaseURL,
		),
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
		VideosCount  string
		AddTs        int64
		LastModifyTs int64
	}
	err = db.QueryRow(`
		SELECT user_id, nickname, avatar, ip_location, `+"`desc`"+`, gender,
		       follows, fans, videos_count, add_ts, last_modify_ts
		FROM kuaishou_creator
		WHERE user_id = ?
	`, "ks-creator-user").Scan(
		&creator.UserID,
		&creator.Nickname,
		&creator.Avatar,
		&creator.IPLocation,
		&creator.Description,
		&creator.Gender,
		&creator.Follows,
		&creator.Fans,
		&creator.VideosCount,
		&creator.AddTs,
		&creator.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "ks-creator-user", creator.UserID)
	require.Equal(t, "mock kuaishou creator", creator.Nickname)
	require.Equal(t, "https://example.test/ks-creator-avatar.png", creator.Avatar)
	require.Equal(t, "", creator.IPLocation)
	require.Equal(t, "kuaishou creator signature", creator.Description)
	require.Equal(t, "女", creator.Gender)
	require.Equal(t, "12", creator.Follows)
	require.Equal(t, "345", creator.Fans)
	require.Equal(t, "1", creator.VideosCount)
	require.NotZero(t, creator.AddTs)
	require.NotZero(t, creator.LastModifyTs)

	var video struct {
		VideoID       string
		VideoType     string
		Title         string
		CreateTime    string
		LikedCount    string
		ViewdCount    string
		VideoURL      string
		VideoCoverURL string
		VideoPlayURL  string
		SourceKeyword string
		UserID        string
		Nickname      string
		Avatar        string
		AddTs         int64
		LastModifyTs  int64
	}
	err = db.QueryRow(`
		SELECT video_id, video_type, title, create_time, liked_count, viewd_count,
		       video_url, video_cover_url, video_play_url, source_keyword,
		       user_id, nickname, avatar, add_ts, last_modify_ts
		FROM kuaishou_video
		WHERE video_id = ?
	`, "ks_creator_video_e2e").Scan(
		&video.VideoID,
		&video.VideoType,
		&video.Title,
		&video.CreateTime,
		&video.LikedCount,
		&video.ViewdCount,
		&video.VideoURL,
		&video.VideoCoverURL,
		&video.VideoPlayURL,
		&video.SourceKeyword,
		&video.UserID,
		&video.Nickname,
		&video.Avatar,
		&video.AddTs,
		&video.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "ks_creator_video_e2e", video.VideoID)
	require.Equal(t, "photo", video.VideoType)
	require.Equal(t, "Kuaishou creator E2E mock video", video.Title)
	require.Equal(t, "1710001000", video.CreateTime)
	require.Equal(t, "266", video.LikedCount)
	require.Equal(t, "2777", video.ViewdCount)
	require.Equal(t, "https://www.kuaishou.com/short-video/ks_creator_video_e2e", video.VideoURL)
	require.Equal(t, "https://example.test/ks-creator-cover.png", video.VideoCoverURL)
	require.Equal(t, "https://example.test/ks-creator-play.mp4", video.VideoPlayURL)
	require.Equal(t, "", video.SourceKeyword)
	require.Equal(t, "ks-creator-author", video.UserID)
	require.Equal(t, "mock kuaishou creator author", video.Nickname)
	require.Equal(t, "https://example.test/ks-creator-author-avatar.png", video.Avatar)
	require.NotZero(t, video.AddTs)
	require.NotZero(t, video.LastModifyTs)
}

func TestCLIKuaishouHomefeedDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "ks",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "ks-unified-homefeed-client",
				Connected: true,
				Nicknames: map[string]string{"ks": "mock-kuaishou"},
				Cookies:   map[string]string{"ks": "kpf=PC_WEB; clientid=homefeed-unified; did=homefeed-unified"},
			},
		},
		PlatformHandler: newKuaishouUnifiedHomefeedMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "ks",
		Keywords:             []string{"mock-ks"},
		CrawlerType:          "homefeed",
		SaveDataOption:       "db",
		CrawlerMaxNotesCount: 1,
		PlatformEndpointOverrides: map[string]string{
			"kuaishouApiUrl":     runner.BaseURL + "/graphql",
			"kuaishouRestApiUrl": runner.BaseURL,
		},
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env: append(
			mysqlServer.Env(),
			"MCPRO_KUAISHOU_API_URL="+runner.BaseURL+"/graphql",
			"MCPRO_KUAISHOU_REST_API_URL="+runner.BaseURL,
		),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()
	var row struct {
		VideoID       string
		VideoType     string
		Title         string
		Description   string
		CreateTime    string
		LikedCount    string
		ViewdCount    string
		VideoURL      string
		VideoCoverURL string
		VideoPlayURL  string
		SourceKeyword string
		UserID        string
		Nickname      string
		Avatar        string
		AddTs         int64
		LastModifyTs  int64
	}
	err = db.QueryRow(`
		SELECT video_id, video_type, title, `+"`desc`"+`, create_time, liked_count, viewd_count,
		       video_url, video_cover_url, video_play_url, source_keyword,
		       user_id, nickname, avatar, add_ts, last_modify_ts
		FROM kuaishou_video
		WHERE video_id = ?
	`, "ks_homefeed_video_e2e").Scan(
		&row.VideoID,
		&row.VideoType,
		&row.Title,
		&row.Description,
		&row.CreateTime,
		&row.LikedCount,
		&row.ViewdCount,
		&row.VideoURL,
		&row.VideoCoverURL,
		&row.VideoPlayURL,
		&row.SourceKeyword,
		&row.UserID,
		&row.Nickname,
		&row.Avatar,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "ks_homefeed_video_e2e", row.VideoID)
	require.Equal(t, "photo", row.VideoType)
	require.Equal(t, "Kuaishou homefeed E2E mock video", row.Title)
	require.Equal(t, "Kuaishou homefeed E2E mock video", row.Description)
	require.Equal(t, "1710001100", row.CreateTime)
	require.Equal(t, "366", row.LikedCount)
	require.Equal(t, "3777", row.ViewdCount)
	require.Equal(t, "https://www.kuaishou.com/short-video/ks_homefeed_video_e2e", row.VideoURL)
	require.Equal(t, "https://example.test/ks-homefeed-cover.png", row.VideoCoverURL)
	require.Equal(t, "https://example.test/ks-homefeed-play.mp4", row.VideoPlayURL)
	require.Equal(t, "", row.SourceKeyword)
	require.Equal(t, "ks-homefeed-author", row.UserID)
	require.Equal(t, "mock kuaishou homefeed author", row.Nickname)
	require.Equal(t, "https://example.test/ks-homefeed-avatar.png", row.Avatar)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func TestCLIKuaishouHotSearchDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "ks",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "ks-hot-unified-client",
				Connected: true,
				Nicknames: map[string]string{"ks": "mock-kuaishou"},
				Cookies:   map[string]string{"ks": "kpf=PC_WEB; clientid=hot-unified; did=hot-unified"},
			},
		},
		PlatformHandler: newKuaishouUnifiedHotSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:               "ks",
		CrawlerType:            "hot_search",
		SaveDataOption:         "db",
		HotSearchMaxItemsCount: 1,
		PlatformEndpointOverrides: map[string]string{
			"kuaishouApiUrl":     runner.BaseURL + "/graphql",
			"kuaishouRestApiUrl": runner.BaseURL,
		},
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{
		Env: append(
			mysqlServer.Env(),
			"MCPRO_KUAISHOU_API_URL="+runner.BaseURL+"/graphql",
			"MCPRO_KUAISHOU_REST_API_URL="+runner.BaseURL,
		),
		Timeout: 20 * time.Second,
	})

	db, err := sql.Open("mysql", mysqlServer.DSN())
	require.NoError(t, err)
	defer db.Close()

	var video struct {
		VideoID       string
		VideoType     string
		Title         string
		Description   string
		CreateTime    string
		LikedCount    string
		ViewdCount    string
		VideoURL      string
		VideoCoverURL string
		VideoPlayURL  string
		SourceKeyword string
		UserID        string
		Nickname      string
		Avatar        string
	}
	err = db.QueryRow(`
		SELECT video_id, video_type, title, `+"`desc`"+`, create_time, liked_count, viewd_count,
		       video_url, video_cover_url, video_play_url, source_keyword, user_id, nickname, avatar
		FROM kuaishou_video
		WHERE video_id = ?
	`, "ks_hot_video_e2e").Scan(
		&video.VideoID,
		&video.VideoType,
		&video.Title,
		&video.Description,
		&video.CreateTime,
		&video.LikedCount,
		&video.ViewdCount,
		&video.VideoURL,
		&video.VideoCoverURL,
		&video.VideoPlayURL,
		&video.SourceKeyword,
		&video.UserID,
		&video.Nickname,
		&video.Avatar,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "ks_hot_video_e2e", video.VideoID)
	require.Equal(t, "photo", video.VideoType)
	require.Equal(t, "Kuaishou hot search E2E mock video", video.Title)
	require.Equal(t, "Kuaishou hot search E2E mock video", video.Description)
	require.Equal(t, "1710001200", video.CreateTime)
	require.Equal(t, "466", video.LikedCount)
	require.Equal(t, "4777", video.ViewdCount)
	require.Equal(t, "https://www.kuaishou.com/short-video/ks_hot_video_e2e", video.VideoURL)
	require.Equal(t, "https://example.test/ks-hot-cover.png", video.VideoCoverURL)
	require.Equal(t, "https://example.test/ks-hot-play.mp4", video.VideoPlayURL)
	require.Equal(t, "kuaishou_hot_search", video.SourceKeyword)
	require.Equal(t, "ks-hot-author", video.UserID)
	require.Equal(t, "mock kuaishou hot author", video.Nickname)
	require.Equal(t, "https://example.test/ks-hot-avatar.png", video.Avatar)

	var item struct {
		HotItemID    string
		Platform     string
		BoardName    string
		BoardType    string
		RankNo       int
		Title        string
		Keyword      string
		HotValue     string
		TopicID      string
		TopicURL     string
		SnapshotTs   int64
		AddTs        int64
		LastModifyTs int64
	}
	err = db.QueryRow(`
		SELECT hot_item_id, platform, board_name, board_type, rank_no, title, keyword,
		       hot_value, topic_id, topic_url, snapshot_ts, add_ts, last_modify_ts
		FROM platform_hot_search_item
		WHERE platform = ? AND rank_no = ?
	`, "ks", 1).Scan(
		&item.HotItemID,
		&item.Platform,
		&item.BoardName,
		&item.BoardType,
		&item.RankNo,
		&item.Title,
		&item.Keyword,
		&item.HotValue,
		&item.TopicID,
		&item.TopicURL,
		&item.SnapshotTs,
		&item.AddTs,
		&item.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.NotEmpty(t, item.HotItemID)
	require.Equal(t, "ks", item.Platform)
	require.Equal(t, "hot_feed", item.BoardName)
	require.Equal(t, "video", item.BoardType)
	require.Equal(t, 1, item.RankNo)
	require.Equal(t, "Kuaishou hot search E2E mock video", item.Title)
	require.Equal(t, "kuaishou_hot_search", item.Keyword)
	require.Equal(t, "466", item.HotValue)
	require.Equal(t, "ks_hot_video_e2e", item.TopicID)
	require.Equal(t, "https://www.kuaishou.com/short-video/ks_hot_video_e2e", item.TopicURL)
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
	`, "ks", "ks_hot_video_e2e").Scan(
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
	require.Equal(t, "ks", relation.Platform)
	require.Equal(t, "ks_hot_video_e2e", relation.ContentID)
	require.Equal(t, "video", relation.ContentType)
	require.Equal(t, "kuaishou_video", relation.ContentTable)
	require.Equal(t, 1, relation.RankNo)
	require.Equal(t, "kuaishou_hot_search", relation.SourceKeyword)
	require.NotZero(t, relation.AddTs)
	require.NotZero(t, relation.LastModifyTs)
}

func newKuaishouUnifiedSearchMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graphql" {
			t.Fatalf("unexpected kuaishou unified runner request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload), string(body))

		switch payload["operationName"] {
		case "visionProfileUserList":
			writeJSON(t, w, map[string]any{
				"data": map[string]any{
					"visionProfileUserList": map[string]any{"result": 1},
				},
			})
		case "visionSearchPhoto":
			variables, _ := payload["variables"].(map[string]any)
			pcursor, _ := variables["pcursor"].(string)
			if pcursor != "1" {
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionSearchPhoto": map[string]any{
							"result":          1,
							"feeds":           []map[string]any{},
							"searchSessionId": "ks-session-empty",
							"pcursor":         "no_more",
						},
					},
				})
				return
			}
			feeds := make([]map[string]any, 0, 21)
			for i := 0; i < 21; i++ {
				feeds = append(feeds, kuaishouMockFeed())
			}
			writeJSON(t, w, map[string]any{
				"data": map[string]any{
					"visionSearchPhoto": map[string]any{
						"result":          1,
						"feeds":           feeds,
						"searchSessionId": "ks-session-1",
						"pcursor":         "2",
					},
				},
			})
		default:
			t.Fatalf("unexpected kuaishou graphql operation: %#v", payload["operationName"])
		}
	})
}

func newKuaishouUnifiedCreatorMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graphql" {
			t.Fatalf("unexpected kuaishou creator unified runner request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload), string(body))

		switch payload["operationName"] {
		case "visionProfileUserList":
			writeJSON(t, w, map[string]any{
				"data": map[string]any{
					"visionProfileUserList": map[string]any{"result": 1},
				},
			})
		case "visionProfile":
			writeJSON(t, w, map[string]any{
				"data": map[string]any{
					"visionProfile": map[string]any{
						"userProfile": kuaishouMockCreatorProfile(),
					},
				},
			})
		case "visionProfilePhotoList":
			writeJSON(t, w, map[string]any{
				"data": map[string]any{
					"visionProfilePhotoList": map[string]any{
						"pcursor": "no_more",
						"feeds": []map[string]any{
							kuaishouMockCreatorFeed(),
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected kuaishou creator graphql operation: %#v", payload["operationName"])
		}
	})
}

func newKuaishouUnifiedDetailMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/rest/v/photo/comment/list" {
			writeJSON(t, w, map[string]any{
				"result":    1,
				"pcursorV2": "no_more",
				"rootCommentsV2": []map[string]any{
					kuaishouMockRootComment(),
				},
			})
			return
		}
		if r.URL.Path == "/rest/v/photo/comment/sublist" {
			writeJSON(t, w, map[string]any{
				"result":    1,
				"pcursorV2": "no_more",
				"subCommentsV2": []map[string]any{
					kuaishouMockSubComment(),
				},
			})
			return
		}
		if r.URL.Path != "/graphql" {
			t.Fatalf("unexpected kuaishou detail unified runner request: %s %s", r.Method, r.URL.String())
		}
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload), string(body))

		switch payload["operationName"] {
		case "visionProfileUserList":
			writeJSON(t, w, map[string]any{
				"data": map[string]any{
					"visionProfileUserList": map[string]any{"result": 1},
				},
			})
		case "visionVideoDetail":
			variables, _ := payload["variables"].(map[string]any)
			require.Equal(t, "ks_detail_e2e", variables["photoId"])
			writeJSON(t, w, map[string]any{
				"data": map[string]any{
					"visionVideoDetail": kuaishouMockDetailFeed(),
				},
			})
		default:
			t.Fatalf("unexpected kuaishou detail graphql operation: %#v", payload["operationName"])
		}
	})
}

func newKuaishouUnifiedHomefeedMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/graphql":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, json.Unmarshal(body, &payload), string(body))
			switch payload["operationName"] {
			case "visionProfileUserList":
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfileUserList": map[string]any{"result": 1},
					},
				})
			case "brilliantTypeDataQuery":
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"brilliantTypeData": map[string]any{
							"pcursor": "no_more",
							"feeds": []map[string]any{
								kuaishouMockHomefeedFeed(),
							},
						},
					},
				})
			default:
				t.Fatalf("unexpected kuaishou homefeed graphql operation: %#v", payload["operationName"])
			}
		default:
			t.Fatalf("unexpected kuaishou unified homefeed request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newKuaishouUnifiedHotSearchMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/v/feed/hot":
			writeJSON(t, w, map[string]any{
				"result":  1,
				"pcursor": "no_more",
				"feeds": []map[string]any{
					kuaishouMockHotSearchFeed(),
				},
			})
		case "/graphql":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, json.Unmarshal(body, &payload), string(body))
			switch payload["operationName"] {
			case "visionProfileUserList":
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionProfileUserList": map[string]any{"result": 1},
					},
				})
			case "visionVideoDetail":
				variables, _ := payload["variables"].(map[string]any)
				require.Equal(t, "ks_hot_video_e2e", variables["photoId"])
				writeJSON(t, w, map[string]any{
					"data": map[string]any{
						"visionVideoDetail": kuaishouMockHotSearchFeed(),
					},
				})
			default:
				t.Fatalf("unexpected kuaishou hot_search graphql operation: %#v", payload["operationName"])
			}
		default:
			t.Fatalf("unexpected kuaishou hot_search unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}
