// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/tests/e2e/tieba_mysql_e2e_test.go
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

func TestCLITiebaSearchDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "tieba",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "tieba-unified-client",
				Connected: true,
				Nicknames: map[string]string{"tieba": "mock-tieba"},
				Cookies:   map[string]string{"tieba": "BDUSS=unified; STOKEN=unified"},
			},
		},
		PlatformHandler: newTiebaUnifiedSearchMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "tieba",
		Keywords:             []string{"mock-tieba"},
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
		NoteID          string
		Title           string
		Description     string
		NoteURL         string
		PublishTime     string
		UserLink        string
		UserNickname    string
		UserAvatar      string
		TiebaName       string
		TiebaLink       string
		TotalReplayNum  int
		TotalReplayPage int
		IPLocation      string
		SourceKeyword   string
		AddTs           int64
		LastModifyTs    int64
	}
	err = db.QueryRow(`
		SELECT note_id, title, `+"`desc`"+`, note_url, publish_time, user_link, user_nickname,
		       user_avatar, tieba_name, tieba_link, total_replay_num, total_replay_page,
		       ip_location, source_keyword, add_ts, last_modify_ts
		FROM tieba_note
		WHERE note_id = ?
	`, "tb_note_e2e").Scan(
		&row.NoteID,
		&row.Title,
		&row.Description,
		&row.NoteURL,
		&row.PublishTime,
		&row.UserLink,
		&row.UserNickname,
		&row.UserAvatar,
		&row.TiebaName,
		&row.TiebaLink,
		&row.TotalReplayNum,
		&row.TotalReplayPage,
		&row.IPLocation,
		&row.SourceKeyword,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "tb_note_e2e", row.NoteID)
	require.Equal(t, "Tieba E2E mock note", row.Title)
	require.Equal(t, "Tieba E2E mock content", row.Description)
	require.Equal(t, runner.BaseURL+"/p/tb_note_e2e", row.NoteURL)
	require.Equal(t, "2024-03-10 00:21:40", row.PublishTime)
	require.Equal(t, runner.BaseURL+"/home/main?id=tb-author-portrait", row.UserLink)
	require.Equal(t, "mock tieba author", row.UserNickname)
	require.Equal(t, "https://example.test/tieba-avatar.png", row.UserAvatar)
	require.Equal(t, "mock tieba吧", row.TiebaName)
	require.Equal(t, runner.BaseURL+"/f?kw=mock%20tieba", row.TiebaLink)
	require.Equal(t, 12, row.TotalReplayNum)
	require.Equal(t, 2, row.TotalReplayPage)
	require.Equal(t, "广东", row.IPLocation)
	require.Equal(t, "mock-tieba", row.SourceKeyword)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func TestCLITiebaDetailDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "tieba",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "tieba-detail-unified-client",
				Connected: true,
				Nicknames: map[string]string{"tieba": "mock-tieba"},
				Cookies:   map[string]string{"tieba": "BDUSS=detail-unified; STOKEN=detail-unified"},
			},
		},
		PlatformHandler: newTiebaUnifiedDetailMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "tieba",
		Keywords:             []string{"mock-tieba"},
		CrawlerType:          "detail",
		SaveDataOption:       "db",
		SpecifiedNoteIDs:     []string{"tb_note_e2e"},
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
		NoteID          string
		Title           string
		Description     string
		NoteURL         string
		PublishTime     string
		UserLink        string
		UserNickname    string
		UserAvatar      string
		TiebaName       string
		TiebaLink       string
		TotalReplayNum  int
		TotalReplayPage int
		IPLocation      string
		SourceKeyword   string
		AddTs           int64
		LastModifyTs    int64
	}
	err = db.QueryRow(`
		SELECT note_id, title, `+"`desc`"+`, note_url, publish_time, user_link, user_nickname,
		       user_avatar, tieba_name, tieba_link, total_replay_num, total_replay_page,
		       ip_location, source_keyword, add_ts, last_modify_ts
		FROM tieba_note
		WHERE note_id = ?
	`, "tb_note_e2e").Scan(
		&row.NoteID,
		&row.Title,
		&row.Description,
		&row.NoteURL,
		&row.PublishTime,
		&row.UserLink,
		&row.UserNickname,
		&row.UserAvatar,
		&row.TiebaName,
		&row.TiebaLink,
		&row.TotalReplayNum,
		&row.TotalReplayPage,
		&row.IPLocation,
		&row.SourceKeyword,
		&row.AddTs,
		&row.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "tb_note_e2e", row.NoteID)
	require.Equal(t, "Tieba E2E mock note", row.Title)
	require.Equal(t, "Tieba E2E mock content", row.Description)
	require.Equal(t, runner.BaseURL+"/p/tb_note_e2e", row.NoteURL)
	require.Equal(t, "2024-03-10 00:21:40", row.PublishTime)
	require.Equal(t, runner.BaseURL+"/home/main?id=tb-author-portrait", row.UserLink)
	require.Equal(t, "mock tieba author", row.UserNickname)
	require.Equal(t, "https://example.test/tieba-avatar.png", row.UserAvatar)
	require.Equal(t, "mock tieba吧", row.TiebaName)
	require.Equal(t, runner.BaseURL+"/f?kw=mock%20tieba", row.TiebaLink)
	require.Equal(t, 12, row.TotalReplayNum)
	require.Equal(t, 1, row.TotalReplayPage)
	require.Equal(t, "广东", row.IPLocation)
	require.Equal(t, "", row.SourceKeyword)
	require.NotZero(t, row.AddTs)
	require.NotZero(t, row.LastModifyTs)
}

func TestCLITiebaDetailCommentsDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "tieba",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "tieba-comments-unified-client",
				Connected: true,
				Nicknames: map[string]string{"tieba": "mock-tieba"},
				Cookies:   map[string]string{"tieba": "BDUSS=comments-unified; STOKEN=comments-unified"},
			},
		},
		PlatformHandler: newTiebaUnifiedDetailMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "tieba",
		Keywords:             []string{"mock-tieba"},
		CrawlerType:          "detail",
		SaveDataOption:       "db",
		SpecifiedNoteIDs:     []string{"tb_note_e2e"},
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
		SELECT comment_id, parent_comment_id, content, user_link, user_nickname,
		       user_avatar, publish_time, ip_location, sub_comment_count,
		       note_id, note_url, tieba_id, tieba_name, tieba_link,
		       add_ts, last_modify_ts
		FROM tieba_comment
		ORDER BY comment_id
	`)
	require.NoError(t, err, "command output:\n%s", output)
	defer rows.Close()

	type commentRow struct {
		CommentID       string
		ParentCommentID string
		Content         string
		UserLink        string
		UserNickname    string
		UserAvatar      string
		PublishTime     string
		IPLocation      string
		SubCommentCount int
		NoteID          string
		NoteURL         string
		TiebaID         string
		TiebaName       string
		TiebaLink       string
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
			&row.UserLink,
			&row.UserNickname,
			&row.UserAvatar,
			&row.PublishTime,
			&row.IPLocation,
			&row.SubCommentCount,
			&row.NoteID,
			&row.NoteURL,
			&row.TiebaID,
			&row.TiebaName,
			&row.TiebaLink,
			&row.AddTs,
			&row.LastModifyTs,
		))
		comments = append(comments, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, comments, 2, "command output:\n%s", output)

	root := comments[0]
	require.Equal(t, "tb_comment_root", root.CommentID)
	require.Equal(t, "", root.ParentCommentID)
	require.Equal(t, "root tieba comment", root.Content)
	require.Equal(t, runner.BaseURL+"/home/main?id=tb-commenter-portrait", root.UserLink)
	require.Equal(t, "mock tieba commenter", root.UserNickname)
	require.Equal(t, "https://example.test/tieba-comment-avatar.png", root.UserAvatar)
	require.Equal(t, "2024-03-10 00:23:20", root.PublishTime)
	require.Equal(t, "上海", root.IPLocation)
	require.Equal(t, 1, root.SubCommentCount)
	require.Equal(t, "tb_note_e2e", root.NoteID)
	require.Equal(t, runner.BaseURL+"/p/tb_note_e2e", root.NoteURL)
	require.Equal(t, "tb-forum-id", root.TiebaID)
	require.Equal(t, "mock tieba吧", root.TiebaName)
	require.Equal(t, runner.BaseURL+"/f?kw=mock%20tieba", root.TiebaLink)
	require.NotZero(t, root.AddTs)
	require.NotZero(t, root.LastModifyTs)

	sub := comments[1]
	require.Equal(t, "tb_comment_sub", sub.CommentID)
	require.Equal(t, "tb_comment_root", sub.ParentCommentID)
	require.Equal(t, "sub tieba comment", sub.Content)
	require.Equal(t, "/home/main?id=tb-sub-commenter-portrait", sub.UserLink)
	require.Equal(t, "mock tieba sub commenter", sub.UserNickname)
	require.Equal(t, "https://example.test/tieba-sub-comment-avatar.png", sub.UserAvatar)
	require.Equal(t, "2024-03-10 00:24", sub.PublishTime)
	require.Equal(t, "", sub.IPLocation)
	require.Equal(t, 0, sub.SubCommentCount)
	require.Equal(t, "tb_note_e2e", sub.NoteID)
	require.Equal(t, runner.BaseURL+"/p/tb_note_e2e", sub.NoteURL)
	require.Equal(t, "tb-forum-id", sub.TiebaID)
	require.Equal(t, "mock tieba吧", sub.TiebaName)
	require.Equal(t, runner.BaseURL+"/f?kw=mock%20tieba", sub.TiebaLink)
	require.NotZero(t, sub.AddTs)
	require.NotZero(t, sub.LastModifyTs)
}

func TestCLITiebaCreatorDBWithUnifiedRunner(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	runner := e2e.NewMockPlatformRunner(t, e2e.MockPlatformOptions{
		Platform: "tieba",
		Accounts: []cookiebridge.Account{
			{
				ClientID:  "tieba-creator-unified-client",
				Connected: true,
				Nicknames: map[string]string{"tieba": "mock-tieba"},
				Cookies:   map[string]string{"tieba": "BDUSS=creator-unified; STOKEN=creator-unified"},
			},
		},
		PlatformHandler: newTiebaUnifiedCreatorMockHandler(t),
	})
	runner.WriteCrawlerConfig(t, e2e.CrawlerConfigOptions{
		Platform:             "tieba",
		Keywords:             []string{"mock-tieba"},
		CrawlerType:          "creator",
		SaveDataOption:       "db",
		SpecifiedCreatorIDs:  []string{"tb-creator-portrait"},
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
		UserID               string
		UserName             string
		Nickname             string
		Avatar               string
		IPLocation           string
		Gender               string
		Follows              string
		Fans                 string
		RegistrationDuration string
		AddTs                int64
		LastModifyTs         int64
	}
	err = db.QueryRow(`
		SELECT user_id, user_name, nickname, avatar, ip_location, gender,
		       follows, fans, registration_duration, add_ts, last_modify_ts
		FROM tieba_creator
		WHERE user_id = ?
	`, "tb-creator-user").Scan(
		&creator.UserID,
		&creator.UserName,
		&creator.Nickname,
		&creator.Avatar,
		&creator.IPLocation,
		&creator.Gender,
		&creator.Follows,
		&creator.Fans,
		&creator.RegistrationDuration,
		&creator.AddTs,
		&creator.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "tb-creator-user", creator.UserID)
	require.Equal(t, "mock_tieba_creator", creator.UserName)
	require.Equal(t, "mock tieba creator", creator.Nickname)
	require.Equal(t, "https://example.test/tieba-creator-avatar.png", creator.Avatar)
	require.Equal(t, "浙江", creator.IPLocation)
	require.Equal(t, "女", creator.Gender)
	require.Equal(t, "12", creator.Follows)
	require.Equal(t, "345", creator.Fans)
	require.Equal(t, "5.6年", creator.RegistrationDuration)
	require.NotZero(t, creator.AddTs)
	require.NotZero(t, creator.LastModifyTs)

	var note struct {
		NoteID          string
		Title           string
		Description     string
		NoteURL         string
		PublishTime     string
		UserLink        string
		UserNickname    string
		UserAvatar      string
		TiebaName       string
		TiebaLink       string
		TotalReplayNum  int
		TotalReplayPage int
		IPLocation      string
		SourceKeyword   string
		AddTs           int64
		LastModifyTs    int64
	}
	err = db.QueryRow(`
		SELECT note_id, title, `+"`desc`"+`, note_url, publish_time, user_link, user_nickname,
		       user_avatar, tieba_name, tieba_link, total_replay_num, total_replay_page,
		       ip_location, source_keyword, add_ts, last_modify_ts
		FROM tieba_note
		WHERE note_id = ?
	`, "tb_note_e2e").Scan(
		&note.NoteID,
		&note.Title,
		&note.Description,
		&note.NoteURL,
		&note.PublishTime,
		&note.UserLink,
		&note.UserNickname,
		&note.UserAvatar,
		&note.TiebaName,
		&note.TiebaLink,
		&note.TotalReplayNum,
		&note.TotalReplayPage,
		&note.IPLocation,
		&note.SourceKeyword,
		&note.AddTs,
		&note.LastModifyTs,
	)
	require.NoError(t, err, "command output:\n%s", output)
	require.Equal(t, "tb_note_e2e", note.NoteID)
	require.Equal(t, "Tieba E2E mock note", note.Title)
	require.Equal(t, "Tieba E2E mock content", note.Description)
	require.Equal(t, runner.BaseURL+"/p/tb_note_e2e", note.NoteURL)
	require.Equal(t, "2024-03-10 00:21:40", note.PublishTime)
	require.Equal(t, runner.BaseURL+"/home/main?id=tb-author-portrait", note.UserLink)
	require.Equal(t, "mock tieba author", note.UserNickname)
	require.Equal(t, "https://example.test/tieba-avatar.png", note.UserAvatar)
	require.Equal(t, "mock tieba吧", note.TiebaName)
	require.Equal(t, runner.BaseURL+"/f?kw=mock%20tieba", note.TiebaLink)
	require.Equal(t, 12, note.TotalReplayNum)
	require.Equal(t, 1, note.TotalReplayPage)
	require.Equal(t, "广东", note.IPLocation)
	require.Equal(t, "", note.SourceKeyword)
	require.NotZero(t, note.AddTs)
	require.NotZero(t, note.LastModifyTs)
}

func newTiebaUnifiedSearchMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/c/s/pc/sync":
			writeJSON(t, w, map[string]any{
				"error_code": 0,
				"data": map[string]any{
					"anti": map[string]string{"tbs": "mock-tieba-tbs"},
					"user": map[string]bool{"is_login": true},
				},
			})
		case "/mo/q/search/multsearch":
			require.Equal(t, "mock-tieba", r.URL.Query().Get("word"))
			if r.URL.Query().Get("pn") != "1" {
				writeJSON(t, w, map[string]any{"error_code": 0, "data": map[string]any{"card_list": []map[string]any{}}})
				return
			}
			writeJSON(t, w, tiebaSearchPayload())
		case "/c/f/pb/page_pc":
			require.NoError(t, r.ParseForm())
			require.Equal(t, "tb_note_e2e", r.Form.Get("kz"))
			writeJSON(t, w, tiebaDetailPayload(2, false))
		default:
			t.Fatalf("unexpected tieba unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newTiebaUnifiedCreatorMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/c/s/pc/sync":
			writeJSON(t, w, map[string]any{
				"error_code": 0,
				"data": map[string]any{
					"anti": map[string]string{"tbs": "mock-tieba-tbs"},
					"user": map[string]bool{"is_login": true},
				},
			})
		case "/c/u/pc/homeSidebarRight":
			require.Equal(t, "tb-creator-portrait", r.URL.Query().Get("portrait"))
			writeJSON(t, w, tiebaCreatorPayload())
		case "/c/u/feed/myThread":
			require.Equal(t, "tb-creator-portrait", r.URL.Query().Get("portrait"))
			require.Equal(t, "1", r.URL.Query().Get("pn"))
			writeJSON(t, w, tiebaCreatorThreadsPayload())
		case "/c/f/pb/page_pc":
			require.NoError(t, r.ParseForm())
			require.Equal(t, "tb_note_e2e", r.Form.Get("kz"))
			writeJSON(t, w, tiebaDetailPayload(1, false))
		default:
			t.Fatalf("unexpected tieba creator unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}

func newTiebaUnifiedDetailMockHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/c/s/pc/sync":
			writeJSON(t, w, map[string]any{
				"error_code": 0,
				"data": map[string]any{
					"anti": map[string]string{"tbs": "mock-tieba-tbs"},
					"user": map[string]bool{"is_login": true},
				},
			})
		case "/c/f/pb/page_pc":
			require.NoError(t, r.ParseForm())
			require.Equal(t, "tb_note_e2e", r.Form.Get("kz"))
			writeJSON(t, w, tiebaDetailPayload(1, true))
		case "/p/comment":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(tiebaSubCommentsHTML()))
		default:
			t.Fatalf("unexpected tieba detail unified runner request: %s %s", r.Method, r.URL.String())
		}
	})
}
