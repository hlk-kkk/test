// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/xhs_unified_runner_test.go
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
	"path/filepath"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/e2e"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/cookiebridge"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestCLIXhsSearchJSONWithUnifiedRunner(t *testing.T) {
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "xhs",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "xhs-unified-client",
				Connected: true,
				Nicknames: map[string]string{"xhs": "mock-xhs"},
				Cookies:   map[string]string{"xhs": "web_session=unified; a1=unified"},
			},
		},
		PlatformHandler: newXHSUnifiedSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "xhs",
		Keywords:             []string{"mock-keyword"},
		CrawlerType:          "search",
		SaveDataOption:       "json",
		CrawlerMaxNotesCount: 1,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{Timeout: 20 * time.Second})
	rows := e2e.ReadJSONRows(
		t,
		filepath.Join(runner.DataDir, "xhs", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json"),
		output,
	)

	require.Len(t, rows, 1)
	require.Equal(t, "note_e2e", rows[0]["note_id"])
	require.Equal(t, "E2E mock note", rows[0]["title"])
	require.Equal(t, "mock-keyword", rows[0]["source_keyword"])
	require.Equal(t, "https://www.xiaohongshu.com/explore/note_e2e?xsec_token=token_e2e&xsec_source=pc_search", rows[0]["note_url"])
	require.NotContains(t, rows[0], "add_ts")
	require.Equal(t, "1710000000000", rows[0]["time"])
	require.Equal(t, "1710000001000", rows[0]["last_update_time"])
}

func TestCLIXhsSearchDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "xhs",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "xhs-unified-client",
				Connected: true,
				Nicknames: map[string]string{"xhs": "mock-xhs"},
				Cookies:   map[string]string{"xhs": "web_session=unified; a1=unified"},
			},
		},
		PlatformHandler: newXHSUnifiedSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "xhs",
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
		NoteID         string
		Title          string
		SourceKeyword  string
		NoteURL        string
		Time           int64
		LastUpdateTime int64
	}
	err = db.QueryRow(`
		SELECT note_id, title, source_keyword, note_url, time, last_update_time
		FROM xhs_note
		WHERE note_id = ?
	`, "note_e2e").Scan(
		&row.NoteID,
		&row.Title,
		&row.SourceKeyword,
		&row.NoteURL,
		&row.Time,
		&row.LastUpdateTime,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "note_e2e", row.NoteID)
	require.Equal(t, "E2E mock note", row.Title)
	require.Equal(t, "mock-keyword", row.SourceKeyword)
	require.Equal(t, "https://www.xiaohongshu.com/explore/note_e2e?xsec_token=token_e2e&xsec_source=pc_search", row.NoteURL)
	require.Equal(t, int64(1710000000000), row.Time)
	require.Equal(t, int64(1710000001000), row.LastUpdateTime)
}

func TestCLIXhsSearchCommentsCSVWithUnifiedRunner(t *testing.T) {
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "xhs",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "xhs-comments-csv-client",
				Connected: true,
				Nicknames: map[string]string{"xhs": "mock-xhs"},
				Cookies:   map[string]string{"xhs": "web_session=comments-csv; a1=comments-csv"},
			},
		},
		PlatformHandler: newXHSUnifiedSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "xhs",
		Keywords:             []string{"mock-keyword"},
		CrawlerType:          "search",
		SaveDataOption:       "csv",
		CrawlerMaxNotesCount: 1,
		EnableGetComments:    true,
		EnableGetSubComments: true,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{Timeout: 20 * time.Second})
	records := e2e.ReadCSVRecords(
		t,
		filepath.Join(runner.DataDir, "xhs", "1_search_comments_"+time.Now().Format("2006-01-02")+".csv"),
		output,
	)

	require.Len(t, records, 3)
	require.Equal(t, "comment_root", e2e.CSVCellAt(records, 1, "comment_id"))
	require.Equal(t, "note_e2e", e2e.CSVCellAt(records, 1, "note_id"))
	require.Equal(t, "root comment from unified mock", e2e.CSVCellAt(records, 1, "content"))
	require.Equal(t, "1710000002000", e2e.CSVCellAt(records, 1, "create_time"))
	require.Equal(t, "https://example.test/comment-root.png", e2e.CSVCellAt(records, 1, "pictures"))
	require.Equal(t, "https://www.xiaohongshu.com/explore/note_e2e?xsec_token=token_e2e&xsec_source=pc_search", e2e.CSVCellAt(records, 1, "note_url"))
	require.Equal(t, "comment_sub", e2e.CSVCellAt(records, 2, "comment_id"))
	require.Equal(t, "comment_root", e2e.CSVCellAt(records, 2, "parent_comment_id"))
	require.Equal(t, "comment_root", e2e.CSVCellAt(records, 2, "target_comment_id"))
}

func TestCLIXhsSearchCommentsSQLiteWithUnifiedRunner(t *testing.T) {
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "xhs",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "xhs-comments-sqlite-client",
				Connected: true,
				Nicknames: map[string]string{"xhs": "mock-xhs"},
				Cookies:   map[string]string{"xhs": "web_session=comments-sqlite; a1=comments-sqlite"},
			},
		},
		PlatformHandler: newXHSUnifiedSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "xhs",
		Keywords:             []string{"mock-keyword"},
		CrawlerType:          "search",
		SaveDataOption:       "sqlite",
		CrawlerMaxNotesCount: 1,
		EnableGetComments:    true,
		EnableGetSubComments: true,
	})

	output := runner.RunCLI(t, e2e.BuildCLI(t, e2e.RepoRoot(t)), e2e.RunOptions{Timeout: 20 * time.Second})
	rows := e2e.ReadSQLiteRows(t, filepath.Join(runner.WorkDir, "media_crawler.db"), "xhs_note_comment", "comment_id", output)

	require.Len(t, rows, 2)
	require.Equal(t, "comment_root", e2e.SQLiteCell(rows, 0, "comment_id"))
	require.Equal(t, "note_e2e", e2e.SQLiteCell(rows, 0, "note_id"))
	require.Equal(t, "root comment from unified mock", e2e.SQLiteCell(rows, 0, "content"))
	require.Equal(t, "1710000002000", e2e.SQLiteCell(rows, 0, "create_time"))
	require.Equal(t, "https://example.test/comment-root.png", e2e.SQLiteCell(rows, 0, "pictures"))
	require.Equal(t, "https://www.xiaohongshu.com/explore/note_e2e?xsec_token=token_e2e&xsec_source=pc_search", e2e.SQLiteCell(rows, 0, "note_url"))
	require.Equal(t, "comment_sub", e2e.SQLiteCell(rows, 1, "comment_id"))
	require.Equal(t, "comment_root", e2e.SQLiteCell(rows, 1, "parent_comment_id"))
	require.Equal(t, "comment_root", e2e.SQLiteCell(rows, 1, "target_comment_id"))
}

func TestCLIXhsSearchCommentsDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "xhs",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "xhs-comments-db-client",
				Connected: true,
				Nicknames: map[string]string{"xhs": "mock-xhs"},
				Cookies:   map[string]string{"xhs": "web_session=comments-db; a1=comments-db"},
			},
		},
		PlatformHandler: newXHSUnifiedSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "xhs",
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
		SELECT comment_id, note_id, content, create_time, sub_comment_count,
		       pictures, parent_comment_id, target_comment_id, like_count,
		       note_url, user_id, nickname, avatar, ip_location, add_ts, last_modify_ts
		FROM xhs_note_comment
		ORDER BY comment_id
	`)
	require.NoError(t, err, "command output:\n%s", output)
	defer rows.Close()

	type commentRow struct {
		CommentID       string
		NoteID          string
		Content         string
		CreateTime      int64
		SubCommentCount string
		Pictures        string
		ParentCommentID string
		TargetCommentID string
		LikeCount       string
		NoteURL         string
		UserID          string
		Nickname        string
		Avatar          string
		IPLocation      string
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
			&row.SubCommentCount,
			&row.Pictures,
			&row.ParentCommentID,
			&row.TargetCommentID,
			&row.LikeCount,
			&row.NoteURL,
			&row.UserID,
			&row.Nickname,
			&row.Avatar,
			&row.IPLocation,
			&row.AddTs,
			&row.LastModifyTs,
		))
		comments = append(comments, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, comments, 2)

	require.Equal(t, "comment_root", comments[0].CommentID)
	require.Equal(t, "note_e2e", comments[0].NoteID)
	require.Equal(t, "root comment from unified mock", comments[0].Content)
	require.Equal(t, int64(1710000002000), comments[0].CreateTime)
	require.Equal(t, "1", comments[0].SubCommentCount)
	require.Equal(t, "https://example.test/comment-root.png", comments[0].Pictures)
	require.Equal(t, "", comments[0].ParentCommentID)
	require.Equal(t, "", comments[0].TargetCommentID)
	require.Equal(t, "4", comments[0].LikeCount)
	require.Equal(t, "https://www.xiaohongshu.com/explore/note_e2e?xsec_token=token_e2e&xsec_source=pc_search", comments[0].NoteURL)
	require.Equal(t, "commenter_root", comments[0].UserID)
	require.Equal(t, "root commenter", comments[0].Nickname)
	require.Equal(t, "https://example.test/commenter-root.png", comments[0].Avatar)
	require.Equal(t, "北京", comments[0].IPLocation)
	require.NotZero(t, comments[0].AddTs)
	require.NotZero(t, comments[0].LastModifyTs)

	require.Equal(t, "comment_sub", comments[1].CommentID)
	require.Equal(t, "sub comment from unified mock", comments[1].Content)
	require.Equal(t, int64(1710000003000), comments[1].CreateTime)
	require.Equal(t, "0", comments[1].SubCommentCount)
	require.Equal(t, "comment_root", comments[1].ParentCommentID)
	require.Equal(t, "comment_root", comments[1].TargetCommentID)
	require.Equal(t, "2", comments[1].LikeCount)
	require.Equal(t, "commenter_sub", comments[1].UserID)
	require.Equal(t, "sub commenter", comments[1].Nickname)
	require.Equal(t, "广东", comments[1].IPLocation)
	require.NotZero(t, comments[1].AddTs)
	require.NotZero(t, comments[1].LastModifyTs)
}

func TestCLIXhsDetailDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "xhs",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "xhs-unified-detail-client",
				Connected: true,
				Nicknames: map[string]string{"xhs": "mock-xhs"},
				Cookies:   map[string]string{"xhs": "web_session=detail-unified; a1=detail-unified"},
			},
		},
		PlatformHandler: newXHSUnifiedDetailMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "xhs",
		Keywords:             []string{"unused"},
		CrawlerType:          "detail",
		SaveDataOption:       "db",
		SpecifiedNoteIDs:     []string{"https://www.xiaohongshu.com/explore/xhs_detail_e2e?xsec_token=token_detail_e2e&xsec_source=pc_note_detail"},
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
		Type           string
		Title          string
		Description    string
		VideoURL       string
		Time           int64
		LastUpdateTime int64
		LikedCount     string
		CollectedCount string
		CommentCount   string
		ShareCount     string
		ImageList      string
		TagList        string
		NoteURL        string
		SourceKeyword  string
		UserID         string
		Nickname       string
		Avatar         string
		IPLocation     string
		AddTs          int64
		LastModifyTs   int64
	}
	err = db.QueryRow(`
		SELECT note_id, type, title, `+"`desc`"+`, video_url, time, last_update_time,
		       liked_count, collected_count, comment_count, share_count, image_list,
		       tag_list, note_url, source_keyword, user_id, nickname, avatar,
		       ip_location, add_ts, last_modify_ts
		FROM xhs_note
		WHERE note_id = ?
	`, "xhs_detail_e2e").Scan(
		&row.NoteID,
		&row.Type,
		&row.Title,
		&row.Description,
		&row.VideoURL,
		&row.Time,
		&row.LastUpdateTime,
		&row.LikedCount,
		&row.CollectedCount,
		&row.CommentCount,
		&row.ShareCount,
		&row.ImageList,
		&row.TagList,
		&row.NoteURL,
		&row.SourceKeyword,
		&row.UserID,
		&row.Nickname,
		&row.Avatar,
		&row.IPLocation,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "xhs_detail_e2e", row.NoteID)
	require.Equal(t, "normal", row.Type)
	require.Equal(t, "XHS detail note", row.Title)
	require.Equal(t, "resume desc from local server", row.Description)
	require.Equal(t, "", row.VideoURL)
	require.Equal(t, int64(1710000000000), row.Time)
	require.Equal(t, int64(1710000001000), row.LastUpdateTime)
	require.Equal(t, "8", row.LikedCount)
	require.Equal(t, "5", row.CollectedCount)
	require.Equal(t, "3", row.CommentCount)
	require.Equal(t, "2", row.ShareCount)
	require.Equal(t, "https://example.test/xhs_detail_e2e.png", row.ImageList)
	require.Equal(t, "resume", row.TagList)
	require.Equal(t, "https://www.xiaohongshu.com/explore/xhs_detail_e2e?xsec_token=token_detail_e2e&xsec_source=pc_note_detail", row.NoteURL)
	require.Equal(t, "", row.SourceKeyword)
	require.Equal(t, "author_resume", row.UserID)
	require.Equal(t, "resume author", row.Nickname)
	require.Equal(t, "https://example.test/resume-avatar.png", row.Avatar)
	require.Equal(t, "上海", row.IPLocation)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func TestCLIXhsCreatorDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "xhs",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "xhs-unified-creator-client",
				Connected: true,
				Nicknames: map[string]string{"xhs": "mock-xhs"},
				Cookies:   map[string]string{"xhs": "web_session=creator-unified; a1=creator-unified"},
			},
		},
		PlatformHandler: newXHSUnifiedCreatorMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "xhs",
		Keywords:             []string{"unused"},
		CrawlerType:          "creator",
		SaveDataOption:       "db",
		SpecifiedCreatorIDs:  []string{"https://www.xiaohongshu.com/user/profile/xhs_creator_e2e?xsec_token=token_creator_profile&xsec_source=pc_user"},
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
		TagList      string
		AddTs        int64
		LastModifyTs int64
	}
	err = db.QueryRow(`
		SELECT user_id, nickname, avatar, ip_location, `+"`desc`"+`, gender,
		       follows, fans, interaction, tag_list, add_ts, last_modify_ts
		FROM xhs_creator
		WHERE user_id = ?
	`, "xhs_creator_e2e").Scan(
		&creator.UserID,
		&creator.Nickname,
		&creator.Avatar,
		&creator.IPLocation,
		&creator.Description,
		&creator.Gender,
		&creator.Follows,
		&creator.Fans,
		&creator.Interaction,
		&creator.TagList,
		&creator.AddTs,
		&creator.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "xhs_creator_e2e", creator.UserID)
	require.Equal(t, "XHS creator mock", creator.Nickname)
	require.Equal(t, "https://example.test/xhs-creator-images.png", creator.Avatar)
	require.Equal(t, "上海", creator.IPLocation)
	require.Equal(t, "creator desc from mock", creator.Description)
	require.Equal(t, "女", creator.Gender)
	require.Equal(t, "12", creator.Follows)
	require.Equal(t, "345", creator.Fans)
	require.Equal(t, "678", creator.Interaction)
	require.Equal(t, `{"profession": "摄影师", "location": "上海"}`, creator.TagList)
	require.NotZero(t, creator.AddTs)
	require.NotZero(t, creator.LastModifyTs)

	var note struct {
		NoteID         string
		Title          string
		NoteURL        string
		SourceKeyword  string
		Time           int64
		LastUpdateTime int64
		AddTs          int64
		LastModifyTs   int64
	}
	err = db.QueryRow(`
		SELECT note_id, title, note_url, source_keyword, time, last_update_time,
		       add_ts, last_modify_ts
		FROM xhs_note
		WHERE note_id = ?
	`, "xhs_creator_note_e2e").Scan(
		&note.NoteID,
		&note.Title,
		&note.NoteURL,
		&note.SourceKeyword,
		&note.Time,
		&note.LastUpdateTime,
		&note.AddTs,
		&note.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "xhs_creator_note_e2e", note.NoteID)
	require.Equal(t, "XHS creator note", note.Title)
	require.Equal(t, "https://www.xiaohongshu.com/explore/xhs_creator_note_e2e?xsec_token=token_creator_note&xsec_source=pc_user", note.NoteURL)
	require.Equal(t, "", note.SourceKeyword)
	require.Equal(t, int64(1710000000000), note.Time)
	require.Equal(t, int64(1710000001000), note.LastUpdateTime)
	require.NotZero(t, note.AddTs)
	require.NotZero(t, note.LastModifyTs)
}

func TestCLIXhsHomefeedDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "xhs",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "xhs-unified-homefeed-client",
				Connected: true,
				Nicknames: map[string]string{"xhs": "mock-xhs"},
				Cookies:   map[string]string{"xhs": "web_session=homefeed-unified; a1=homefeed-unified"},
			},
		},
		PlatformHandler: newXHSUnifiedHomefeedMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "xhs",
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
		NoteID         string
		Title          string
		NoteURL        string
		SourceKeyword  string
		Time           int64
		LastUpdateTime int64
		AddTs          int64
		LastModifyTs   int64
	}
	err = db.QueryRow(`
		SELECT note_id, title, note_url, source_keyword, time, last_update_time,
		       add_ts, last_modify_ts
		FROM xhs_note
		WHERE note_id = ?
	`, "xhs_homefeed_note_e2e").Scan(
		&row.NoteID,
		&row.Title,
		&row.NoteURL,
		&row.SourceKeyword,
		&row.Time,
		&row.LastUpdateTime,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "xhs_homefeed_note_e2e", row.NoteID)
	require.Equal(t, "XHS homefeed note", row.Title)
	require.Equal(t, "https://www.xiaohongshu.com/explore/xhs_homefeed_note_e2e?xsec_token=token_homefeed_note&xsec_source=pc_feed", row.NoteURL)
	require.Equal(t, "homefeed", row.SourceKeyword)
	require.Equal(t, int64(1710000000000), row.Time)
	require.Equal(t, int64(1710000001000), row.LastUpdateTime)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func newXHSUnifiedSearchMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		payload := decodeJSONPayload(t, r)

		switch r.URL.Path {
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{
						"success": true,
						"message": "ok",
						"code":    0,
					},
				},
			})
		case "/api/sns/web/v1/search/notes":
			if page := int(payloadNumber(payload, "page")); page != 1 {
				writeJSON(t, w, map[string]any{
					"code":    0,
					"success": true,
					"msg":     "ok",
					"data":    map[string]any{"has_more": false, "items": []map[string]any{}},
				})
				return
			}
			writeJSON(t, w, xhsResumeSearchPayload("note_e2e", "token_e2e"))
		case "/api/sns/web/v1/feed":
			if got := payloadString(payload, "source_note_id"); got != "note_e2e" {
				t.Fatalf("unexpected xhs feed source_note_id: %s", got)
			}
			writeJSON(t, w, xhsResumeFeedPayload("note_e2e", "E2E mock note", "token_e2e"))
		case "/api/sns/web/v2/comment/page":
			require.Equal(t, "note_e2e", r.URL.Query().Get("note_id"))
			require.Equal(t, "token_e2e", r.URL.Query().Get("xsec_token"))
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"cursor":   "",
					"has_more": false,
					"comments": []map[string]any{
						{
							"id":                "comment_root",
							"note_id":           "note_e2e",
							"content":           "root comment from unified mock",
							"create_time":       int64(1710000002000),
							"like_count":        "4",
							"sub_comment_count": "1",
							"pictures": []map[string]string{
								{"url_default": "https://example.test/comment-root.png"},
							},
							"user_info": map[string]string{
								"user_id":  "commenter_root",
								"nickname": "root commenter",
								"image":    "https://example.test/commenter-root.png",
							},
							"ip_location": "北京",
						},
					},
				},
			})
		case "/api/sns/web/v2/comment/sub/page":
			require.Equal(t, "note_e2e", r.URL.Query().Get("note_id"))
			require.Equal(t, "comment_root", r.URL.Query().Get("root_comment_id"))
			require.Equal(t, "token_e2e", r.URL.Query().Get("xsec_token"))
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"cursor":   "",
					"has_more": false,
					"comments": []map[string]any{
						{
							"id":                "comment_sub",
							"note_id":           "note_e2e",
							"content":           "sub comment from unified mock",
							"create_time":       int64(1710000003000),
							"like_count":        "2",
							"sub_comment_count": "0",
							"pictures":          []map[string]string{},
							"target_comment": map[string]string{
								"id": "comment_root",
							},
							"user_info": map[string]string{
								"user_id":  "commenter_sub",
								"nickname": "sub commenter",
								"image":    "https://example.test/commenter-sub.png",
							},
							"ip_location": "广东",
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected xhs unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newXHSUnifiedDetailMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		payload := decodeJSONPayload(t, r)

		switch r.URL.Path {
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{
						"success": true,
						"message": "ok",
						"code":    0,
					},
				},
			})
		case "/api/sns/web/v1/feed":
			require.Equal(t, "xhs_detail_e2e", payloadString(payload, "source_note_id"))
			require.Equal(t, "token_detail_e2e", payloadString(payload, "xsec_token"))
			require.Equal(t, "pc_note_detail", payloadString(payload, "xsec_source"))
			writeJSON(t, w, xhsResumeFeedPayload("xhs_detail_e2e", "XHS detail note", "token_detail_e2e"))
		default:
			t.Fatalf("unexpected xhs detail unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newXHSUnifiedCreatorMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		payload := decodeJSONPayload(t, r)

		switch r.URL.Path {
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{
						"success": true,
						"message": "ok",
						"code":    0,
					},
				},
			})
		case "/user/profile/xhs_creator_e2e":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(xhsCreatorHTML()))
		case "/api/sns/web/v1/user_posted":
			require.Equal(t, "xhs_creator_e2e", r.URL.Query().Get("user_id"))
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"cursor":   "",
					"has_more": false,
					"notes": []map[string]any{
						{
							"note_id":       "xhs_creator_note_e2e",
							"xsec_token":    "token_creator_note",
							"xsec_source":   "pc_user",
							"display_title": "creator list note",
							"type":          "normal",
							"cover": map[string]string{
								"url": "https://example.test/xhs-creator-cover.png",
							},
						},
					},
				},
			})
		case "/api/sns/web/v1/feed":
			require.Equal(t, "xhs_creator_note_e2e", payloadString(payload, "source_note_id"))
			require.Equal(t, "token_creator_note", payloadString(payload, "xsec_token"))
			require.Equal(t, "pc_user", payloadString(payload, "xsec_source"))
			writeJSON(t, w, xhsResumeFeedPayload("xhs_creator_note_e2e", "XHS creator note", "token_creator_note"))
		default:
			t.Fatalf("unexpected xhs creator unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newXHSUnifiedHomefeedMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		payload := decodeJSONPayload(t, r)

		switch r.URL.Path {
		case "/api/sns/web/v1/user/selfinfo":
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"result": map[string]any{
						"success": true,
						"message": "ok",
						"code":    0,
					},
				},
			})
		case "/api/sns/web/v1/homefeed":
			require.Equal(t, "homefeed_recommend", payloadString(payload, "category"))
			writeJSON(t, w, map[string]any{
				"code":    0,
				"success": true,
				"msg":     "ok",
				"data": map[string]any{
					"cursor_score": "homefeed_cursor_1",
					"items": []map[string]any{
						{
							"id":         "xhs_homefeed_note_e2e",
							"model_type": "note",
							"xsec_token": "token_homefeed_note",
							"note_card": map[string]any{
								"note_id":       "xhs_homefeed_note_e2e",
								"display_title": "homefeed list note",
								"type":          "normal",
								"user": map[string]string{
									"user_id":  "author_homefeed",
									"nickname": "homefeed author",
									"avatar":   "https://example.test/homefeed-avatar.png",
								},
							},
						},
					},
				},
			})
		case "/api/sns/web/v1/feed":
			require.Equal(t, "xhs_homefeed_note_e2e", payloadString(payload, "source_note_id"))
			require.Equal(t, "token_homefeed_note", payloadString(payload, "xsec_token"))
			writeJSON(t, w, xhsResumeFeedPayload("xhs_homefeed_note_e2e", "XHS homefeed note", "token_homefeed_note"))
		default:
			t.Fatalf("unexpected xhs homefeed unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func decodeJSONPayload(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
		t.Fatalf("decode request JSON for %s: %v", r.URL.Path, err)
	}
	return payload
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key].(string)
	if !ok {
		return ""
	}
	return value
}
