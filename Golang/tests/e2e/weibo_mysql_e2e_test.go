// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/weibo_mysql_e2e_test.go
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
	"strings"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/e2e"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/cookiebridge"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestCLIWeiboSearchDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "wb",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "wb-unified-client",
				Connected: true,
				Nicknames: map[string]string{"wb": "mock-weibo", "weibo": "mock-weibo"},
				Cookies:   map[string]string{"wb": "SUB=unified; XSRF-TOKEN=unified", "weibo": "SUB=unified; XSRF-TOKEN=unified"},
			},
		},
		PlatformHandler: newWeiboUnifiedSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "wb",
		Keywords:             []string{"mock-wb"},
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
		NoteID         string
		Content        string
		CreateTime     int64
		CreateDateTime string
		LikedCount     string
		CommentsCount  string
		SharedCount    string
		NoteURL        string
		IPLocation     string
		ImageList      string
		VideoURL       string
		SourceKeyword  string
		UserID         string
		Nickname       string
		Gender         string
		ProfileURL     string
		Avatar         string
	}
	err = db.QueryRow(`
		SELECT note_id, content, create_time, create_date_time, liked_count, comments_count,
		       shared_count, note_url, ip_location, image_list, video_url, source_keyword,
		       user_id, nickname, gender, profile_url, avatar
		FROM weibo_note
		WHERE note_id = ?
	`, "wb_note_e2e").Scan(
		&row.NoteID,
		&row.Content,
		&row.CreateTime,
		&row.CreateDateTime,
		&row.LikedCount,
		&row.CommentsCount,
		&row.SharedCount,
		&row.NoteURL,
		&row.IPLocation,
		&row.ImageList,
		&row.VideoURL,
		&row.SourceKeyword,
		&row.UserID,
		&row.Nickname,
		&row.Gender,
		&row.ProfileURL,
		&row.Avatar,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "wb_note_e2e", row.NoteID)
	require.Equal(t, "Weibo E2E mock note", row.Content)
	require.Equal(t, int64(1710000700), row.CreateTime)
	require.Equal(t, "2024-03-10 00:11:40+08:00", row.CreateDateTime)
	require.Equal(t, "66", row.LikedCount)
	require.Equal(t, "7", row.CommentsCount)
	require.Equal(t, "8", row.SharedCount)
	require.Equal(t, "https://m.weibo.cn/detail/wb_note_e2e", row.NoteURL)
	require.Equal(t, "上海", row.IPLocation)
	require.Equal(t, "https://example.test/wb-image-1.jpg,https://example.test/wb-image-2.jpg", row.ImageList)
	require.Equal(t, "https://example.test/wb-720.mp4,https://example.test/wb-hd.mp4", row.VideoURL)
	require.Equal(t, "mock-wb", row.SourceKeyword)
	require.Equal(t, "wb-author", row.UserID)
	require.Equal(t, "mock weibo author", row.Nickname)
	require.Equal(t, "m", row.Gender)
	require.Equal(t, "https://m.weibo.cn/u/wb-author", row.ProfileURL)
	require.Equal(t, "https://example.test/wb-avatar.png", row.Avatar)
}

func TestCLIWeiboDetailDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "wb",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "wb-detail-unified-client",
				Connected: true,
				Nicknames: map[string]string{"wb": "mock-weibo", "weibo": "mock-weibo"},
				Cookies:   map[string]string{"wb": "SUB=detail-unified; XSRF-TOKEN=detail-unified", "weibo": "SUB=detail-unified; XSRF-TOKEN=detail-unified"},
			},
		},
		PlatformHandler: newWeiboUnifiedDetailMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "wb",
		Keywords:             []string{"mock-wb"},
		CrawlerType:          "detail",
		SaveDataOption:       "db",
		SpecifiedNoteIDs:     []string{"wb_detail_e2e"},
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
		NoteID         string
		Content        string
		CreateTime     int64
		CreateDateTime string
		LikedCount     string
		CommentsCount  string
		SharedCount    string
		NoteURL        string
		IPLocation     string
		ImageList      string
		VideoURL       string
		SourceKeyword  string
		UserID         string
		Nickname       string
		Gender         string
		ProfileURL     string
		Avatar         string
		AddTs          int64
		LastModifyTs   int64
	}
	err = db.QueryRow(`
		SELECT note_id, content, create_time, create_date_time, liked_count, comments_count,
		       shared_count, note_url, ip_location, image_list, video_url, source_keyword,
		       user_id, nickname, gender, profile_url, avatar, add_ts, last_modify_ts
		FROM weibo_note
		WHERE note_id = ?
	`, "wb_detail_e2e").Scan(
		&row.NoteID,
		&row.Content,
		&row.CreateTime,
		&row.CreateDateTime,
		&row.LikedCount,
		&row.CommentsCount,
		&row.SharedCount,
		&row.NoteURL,
		&row.IPLocation,
		&row.ImageList,
		&row.VideoURL,
		&row.SourceKeyword,
		&row.UserID,
		&row.Nickname,
		&row.Gender,
		&row.ProfileURL,
		&row.Avatar,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "wb_detail_e2e", row.NoteID)
	require.Equal(t, "Weibo detail E2E mock note", row.Content)
	require.Equal(t, int64(1710000800), row.CreateTime)
	require.Equal(t, "2024-03-10 00:13:20+08:00", row.CreateDateTime)
	require.Equal(t, "166", row.LikedCount)
	require.Equal(t, "17", row.CommentsCount)
	require.Equal(t, "18", row.SharedCount)
	require.Equal(t, "https://m.weibo.cn/detail/wb_detail_e2e", row.NoteURL)
	require.Equal(t, "北京", row.IPLocation)
	require.Equal(t, "https://example.test/wb-detail-image-1.jpg", row.ImageList)
	require.Equal(t, "https://example.test/wb-detail-720.mp4,https://example.test/wb-detail-ld.mp4", row.VideoURL)
	require.Equal(t, "", row.SourceKeyword)
	require.Equal(t, "wb-detail-author", row.UserID)
	require.Equal(t, "mock weibo detail author", row.Nickname)
	require.Equal(t, "f", row.Gender)
	require.Equal(t, "https://m.weibo.cn/u/wb-detail-author", row.ProfileURL)
	require.Equal(t, "https://example.test/wb-detail-avatar.png", row.Avatar)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func TestCLIWeiboDetailCommentsDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "wb",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "wb-comments-unified-client",
				Connected: true,
				Nicknames: map[string]string{"wb": "mock-weibo", "weibo": "mock-weibo"},
				Cookies:   map[string]string{"wb": "SUB=comments-unified; XSRF-TOKEN=comments-unified", "weibo": "SUB=comments-unified; XSRF-TOKEN=comments-unified"},
			},
		},
		PlatformHandler: newWeiboUnifiedDetailMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "wb",
		Keywords:             []string{"mock-wb"},
		CrawlerType:          "detail",
		SaveDataOption:       "db",
		SpecifiedNoteIDs:     []string{"wb_detail_e2e"},
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
		SELECT comment_id, note_id, content, create_time, create_date_time,
		       sub_comment_count, like_count, ip_location, parent_comment_id,
		       user_id, nickname, gender, profile_url, avatar, add_ts, last_modify_ts
		FROM weibo_note_comment
		ORDER BY comment_id
	`)
	require.NoError(t, err, "command output:\n%s", output)
	defer rows.Close()

	type commentRow struct {
		CommentID       string
		NoteID          string
		Content         string
		CreateTime      int64
		CreateDateTime  string
		SubCommentCount string
		LikeCount       string
		IPLocation      string
		ParentCommentID string
		UserID          string
		Nickname        string
		Gender          string
		ProfileURL      string
		Avatar          string
		AddTs           int64
		LastModifyTs    int64
	}
	var comments []commentRow
	for rows.Next() {
		var row commentRow
		require.NoError(t, rows.Scan(
			&row.CommentID,
			&row.NoteID,
			&row.Content,
			&row.CreateTime,
			&row.CreateDateTime,
			&row.SubCommentCount,
			&row.LikeCount,
			&row.IPLocation,
			&row.ParentCommentID,
			&row.UserID,
			&row.Nickname,
			&row.Gender,
			&row.ProfileURL,
			&row.Avatar,
			&row.AddTs,
			&row.LastModifyTs,
		))
		comments = append(comments, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, comments, 2, "command output:\n%s", output)

	root := comments[0]
	require.Equal(t, "wb_comment_root", root.CommentID)
	require.Equal(t, "wb_detail_e2e", root.NoteID)
	require.Equal(t, "root weibo comment", root.Content)
	require.Equal(t, int64(1710000900), root.CreateTime)
	require.Equal(t, "2024-03-10 00:15:00+08:00", root.CreateDateTime)
	require.Equal(t, "1", root.SubCommentCount)
	require.Equal(t, "12", root.LikeCount)
	require.Equal(t, "广东", root.IPLocation)
	require.Equal(t, "", root.ParentCommentID)
	require.Equal(t, "wb-commenter", root.UserID)
	require.Equal(t, "mock wb commenter", root.Nickname)
	require.Equal(t, "m", root.Gender)
	require.Equal(t, "https://m.weibo.cn/u/wb-commenter", root.ProfileURL)
	require.Equal(t, "https://example.test/wb-comment-avatar.png", root.Avatar)
	require.NotZero(t, root.AddTs)
	require.NotZero(t, root.LastModifyTs)

	sub := comments[1]
	require.Equal(t, "wb_comment_sub", sub.CommentID)
	require.Equal(t, "wb_detail_e2e", sub.NoteID)
	require.Equal(t, "sub weibo comment", sub.Content)
	require.Equal(t, int64(1710000910), sub.CreateTime)
	require.Equal(t, "2024-03-10 00:15:10+08:00", sub.CreateDateTime)
	require.Equal(t, "0", sub.SubCommentCount)
	require.Equal(t, "3", sub.LikeCount)
	require.Equal(t, "深圳", sub.IPLocation)
	require.Equal(t, "wb_comment_root", sub.ParentCommentID)
	require.Equal(t, "wb-sub-commenter", sub.UserID)
	require.Equal(t, "mock wb sub commenter", sub.Nickname)
	require.Equal(t, "f", sub.Gender)
	require.Equal(t, "https://m.weibo.cn/u/wb-sub-commenter", sub.ProfileURL)
	require.Equal(t, "https://example.test/wb-sub-comment-avatar.png", sub.Avatar)
	require.NotZero(t, sub.AddTs)
	require.NotZero(t, sub.LastModifyTs)
}

func TestCLIWeiboCreatorDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "wb",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "wb-creator-unified-client",
				Connected: true,
				Nicknames: map[string]string{"wb": "mock-weibo", "weibo": "mock-weibo"},
				Cookies:   map[string]string{"wb": "SUB=creator-unified; XSRF-TOKEN=creator-unified", "weibo": "SUB=creator-unified; XSRF-TOKEN=creator-unified"},
			},
		},
		PlatformHandler: newWeiboUnifiedCreatorMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "wb",
		Keywords:             []string{"mock-wb"},
		CrawlerType:          "creator",
		SaveDataOption:       "db",
		SpecifiedCreatorIDs:  []string{"wb-creator-user"},
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
		TagList      string
		AddTs        int64
		LastModifyTs int64
	}
	err = db.QueryRow(`
		SELECT user_id, nickname, avatar, ip_location, `+"`desc`"+`, gender,
		       follows, fans, tag_list, add_ts, last_modify_ts
		FROM weibo_creator
		WHERE user_id = ?
	`, "wb-creator-user").Scan(
		&creator.UserID,
		&creator.Nickname,
		&creator.Avatar,
		&creator.IPLocation,
		&creator.Description,
		&creator.Gender,
		&creator.Follows,
		&creator.Fans,
		&creator.TagList,
		&creator.AddTs,
		&creator.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "wb-creator-user", creator.UserID)
	require.Equal(t, "mock weibo creator", creator.Nickname)
	require.Equal(t, "https://example.test/wb-creator-avatar.png", creator.Avatar)
	require.Equal(t, "杭州", creator.IPLocation)
	require.Equal(t, "weibo creator signature", creator.Description)
	require.Equal(t, "女", creator.Gender)
	require.Equal(t, "12", creator.Follows)
	require.Equal(t, "3456", creator.Fans)
	require.Equal(t, "", creator.TagList)
	require.NotZero(t, creator.AddTs)
	require.NotZero(t, creator.LastModifyTs)

	var note struct {
		NoteID         string
		Content        string
		CreateTime     int64
		CreateDateTime string
		LikedCount     string
		CommentsCount  string
		SharedCount    string
		NoteURL        string
		IPLocation     string
		ImageList      string
		VideoURL       string
		SourceKeyword  string
		UserID         string
		Nickname       string
		Gender         string
		ProfileURL     string
		Avatar         string
		AddTs          int64
		LastModifyTs   int64
	}
	err = db.QueryRow(`
		SELECT note_id, content, create_time, create_date_time, liked_count, comments_count,
		       shared_count, note_url, ip_location, image_list, video_url, source_keyword,
		       user_id, nickname, gender, profile_url, avatar, add_ts, last_modify_ts
		FROM weibo_note
		WHERE note_id = ?
	`, "wb_creator_note_e2e").Scan(
		&note.NoteID,
		&note.Content,
		&note.CreateTime,
		&note.CreateDateTime,
		&note.LikedCount,
		&note.CommentsCount,
		&note.SharedCount,
		&note.NoteURL,
		&note.IPLocation,
		&note.ImageList,
		&note.VideoURL,
		&note.SourceKeyword,
		&note.UserID,
		&note.Nickname,
		&note.Gender,
		&note.ProfileURL,
		&note.Avatar,
		&note.AddTs,
		&note.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "wb_creator_note_e2e", note.NoteID)
	require.Equal(t, "Weibo creator E2E mock note", note.Content)
	require.Equal(t, int64(1710001000), note.CreateTime)
	require.Equal(t, "2024-03-10 00:16:40+08:00", note.CreateDateTime)
	require.Equal(t, "266", note.LikedCount)
	require.Equal(t, "27", note.CommentsCount)
	require.Equal(t, "28", note.SharedCount)
	require.Equal(t, "https://m.weibo.cn/detail/wb_creator_note_e2e", note.NoteURL)
	require.Equal(t, "成都", note.IPLocation)
	require.Equal(t, "https://example.test/wb-creator-image.jpg", note.ImageList)
	require.Equal(t, "https://example.test/wb-creator-720.mp4", note.VideoURL)
	require.Equal(t, "", note.SourceKeyword)
	require.Equal(t, "wb-creator-author", note.UserID)
	require.Equal(t, "mock weibo creator author", note.Nickname)
	require.Equal(t, "m", note.Gender)
	require.Equal(t, "https://m.weibo.cn/u/wb-creator-author", note.ProfileURL)
	require.Equal(t, "https://example.test/wb-creator-author-avatar.png", note.Avatar)
	require.NotZero(t, note.AddTs)
	require.NotZero(t, note.LastModifyTs)
}

func TestCLIWeiboHotSearchDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "wb",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "wb-hot-unified-client",
				Connected: true,
				Nicknames: map[string]string{"wb": "mock-weibo", "weibo": "mock-weibo"},
				Cookies:   map[string]string{"wb": "SUB=hot-unified; XSRF-TOKEN=hot-unified", "weibo": "SUB=hot-unified; XSRF-TOKEN=hot-unified"},
			},
		},
		PlatformHandler: newWeiboUnifiedHotSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:                "wb",
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

	var note struct {
		NoteID        string
		Content       string
		NoteURL       string
		IPLocation    string
		ImageList     string
		VideoURL      string
		SourceKeyword string
		UserID        string
		Nickname      string
	}
	err = db.QueryRow(`
		SELECT note_id, content, note_url, ip_location, image_list, video_url,
		       source_keyword, user_id, nickname
		FROM weibo_note
		WHERE note_id = ?
	`, "wb_hot_note_e2e").Scan(
		&note.NoteID,
		&note.Content,
		&note.NoteURL,
		&note.IPLocation,
		&note.ImageList,
		&note.VideoURL,
		&note.SourceKeyword,
		&note.UserID,
		&note.Nickname,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "wb_hot_note_e2e", note.NoteID)
	require.Equal(t, "Weibo hot search E2E mock note", note.Content)
	require.Equal(t, "https://m.weibo.cn/detail/wb_hot_note_e2e", note.NoteURL)
	require.Equal(t, "深圳", note.IPLocation)
	require.Equal(t, "https://example.test/wb-hot-image.jpg", note.ImageList)
	require.Equal(t, "https://example.test/wb-hot-720.mp4", note.VideoURL)
	require.Equal(t, "mock hot weibo", note.SourceKeyword)
	require.Equal(t, "wb-hot-author", note.UserID)
	require.Equal(t, "mock weibo hot author", note.Nickname)

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
		SnapshotDate string
		SnapshotTs   int64
		AddTs        int64
		LastModifyTs int64
	}
	err = db.QueryRow(`
		SELECT hot_item_id, platform, board_name, board_type, rank_no, title, keyword,
		       hot_value, badge, topic_id, topic_url, raw_json, snapshot_date,
		       snapshot_ts, add_ts, last_modify_ts
		FROM platform_hot_search_item
		WHERE platform = ? AND rank_no = ?
	`, "wb", 1).Scan(
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
		&item.SnapshotDate,
		&item.SnapshotTs,
		&item.AddTs,
		&item.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.NotEmpty(t, item.HotItemID)
	require.Equal(t, "wb", item.Platform)
	require.Equal(t, "hot_search", item.BoardName)
	require.Equal(t, "topic", item.BoardType)
	require.Equal(t, 1, item.RankNo)
	require.Equal(t, "mock hot weibo", item.Title)
	require.Equal(t, "mock hot weibo", item.Keyword)
	require.Equal(t, "999", item.HotValue)
	require.Equal(t, "热", item.Badge)
	require.Equal(t, "wb-hot-mid", item.TopicID)
	require.Equal(t, "https://s.weibo.com/weibo?q=mock%20hot%20weibo", item.TopicURL)
	require.Contains(t, item.RawJSON, "mock hot weibo")
	require.NotEmpty(t, item.SnapshotDate)
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
	`, "wb", "wb_hot_note_e2e").Scan(
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
	require.Equal(t, "wb", relation.Platform)
	require.Equal(t, "wb_hot_note_e2e", relation.ContentID)
	require.Equal(t, "note", relation.ContentType)
	require.Equal(t, "weibo_note", relation.ContentTable)
	require.Equal(t, 1, relation.RankNo)
	require.Equal(t, "mock hot weibo", relation.SourceKeyword)
	require.NotZero(t, relation.AddTs)
	require.NotZero(t, relation.LastModifyTs)
}

func newWeiboUnifiedSearchMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/config":
			writeJSON(t, w, map[string]any{"ok": 1, "data": map[string]bool{"login": true}})
		case "/api/container/getIndex":
			require.Equal(t, "100103type=1&q=mock-wb", r.URL.Query().Get("containerid"))
			require.Equal(t, "searchall", r.URL.Query().Get("page_type"))
			if r.URL.Query().Get("page") != "1" {
				writeJSON(t, w, map[string]any{"ok": 1, "data": map[string]any{"cards": []map[string]any{}}})
				return
			}
			writeJSON(t, w, weiboSearchPayload())
		default:
			t.Fatalf("unexpected weibo unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newWeiboUnifiedCreatorMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/config":
			writeJSON(t, w, map[string]any{"ok": 1, "data": map[string]bool{"login": true}})
		case "/api/container/getIndex":
			switch r.URL.Query().Get("containerid") {
			case "100505wb-creator-user":
				writeJSON(t, w, weiboCreatorInfoPayload())
			case "107603wb-creator-user":
				writeJSON(t, w, weiboCreatorNotesPayload())
			default:
				t.Fatalf("unexpected weibo creator containerid: %s", r.URL.Query().Get("containerid"))
			}
		default:
			t.Fatalf("unexpected weibo creator unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newWeiboUnifiedDetailMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/config":
			w.Header().Set("Content-Type", "application/json")
			writeJSON(t, w, map[string]any{"ok": 1, "data": map[string]bool{"login": true}})
		case "/detail/wb_detail_e2e":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(weiboDetailHTML()))
		case "/comments/hotflow":
			w.Header().Set("Content-Type", "application/json")
			writeJSON(t, w, weiboCommentsPayload())
		default:
			t.Fatalf("unexpected weibo detail unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newWeiboUnifiedHotSearchMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/config":
			writeJSON(t, w, map[string]any{"ok": 1, "data": map[string]bool{"login": true}})
		case "/ajax/statuses/hot_band":
			writeJSON(t, w, weiboHotBandPayload())
		case "/api/container/getIndex":
			containerID := r.URL.Query().Get("containerid")
			require.True(t, strings.Contains(containerID, "mock hot weibo"), "unexpected containerid: %s", containerID)
			require.Equal(t, "searchall", r.URL.Query().Get("page_type"))
			require.Equal(t, "1", r.URL.Query().Get("page"))
			writeJSON(t, w, weiboHotSearchPayload())
		default:
			t.Fatalf("unexpected weibo hot_search unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}
