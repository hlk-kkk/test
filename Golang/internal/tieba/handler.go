// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/tieba/handler.go
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
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/crawler"
	tiebamodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/tieba"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	tiebaclient "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/tieba"
)

type Handler interface {
	Handle(ctx context.Context) error
	Name() string
}

type HandlerDeps struct {
	Client     *tiebaclient.Client
	Store      Store
	Checkpoint *crawler.CheckpointManager
	Config     *config.Config
	Logger     *tools.Logger
}

type SearchHandler struct {
	*HandlerDeps
}

func NewSearchHandler(deps *HandlerDeps) *SearchHandler {
	return &SearchHandler{HandlerDeps: deps}
}

func (h *SearchHandler) Name() string {
	return "search"
}

func (h *SearchHandler) Handle(ctx context.Context) error {
	h.Logger.Info("[SearchHandler.Handle] Begin search tieba keywords")

	pageSize := 10
	maxNotes := h.Config.CrawlerMaxNotesCount
	if maxNotes < pageSize {
		maxNotes = pageSize
	}

	for _, keyword := range h.Config.Keywords {
		page := h.Config.StartPage
		if page <= 0 {
			page = 1
		}
		savedCount := (page - 1) * pageSize
		var cp *crawler.Checkpoint
		if h.Config.EnableCheckpoint {
			var err error
			cp, err = h.Checkpoint.Load(h.Name(), keyword)
			if err != nil {
				return err
			}
			if cp != nil {
				if cp.CurrentPage > 0 {
					page = cp.CurrentPage
				}
				savedCount = cp.TotalCrawled
				h.Logger.Info("[SearchHandler.Handle] Resume from checkpoint, keyword: ", keyword, ", page: ", page)
			} else {
				cp = &crawler.Checkpoint{
					CrawlerType:  h.Name(),
					Keyword:      keyword,
					CurrentPage:  page,
					TotalCrawled: savedCount,
				}
			}
		}
		for savedCount <= maxNotes {
			noteIDs, err := h.Client.GetNotesByKeyword(keyword, page, pageSize)
			if err != nil {
				return err
			}
			if len(noteIDs) == 0 {
				break
			}
			remaining := maxNotes - savedCount
			if remaining <= 0 {
				break
			}
			if len(noteIDs) > remaining {
				noteIDs = noteIDs[:remaining]
			}

			for _, noteID := range noteIDs {
				if noteID == "" {
					continue
				}
				if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, noteID) {
					savedCount++
					continue
				}
				if exists, err := h.Store.NoteExists(ctx, noteID); err != nil {
					return err
				} else if exists {
					savedCount++
					if h.Config.EnableCheckpoint {
						h.Checkpoint.AddCrawledNote(cp, noteID)
					}
					continue
				}
				detail, err := h.Client.GetNoteByID(noteID)
				if err != nil {
					return err
				}
				if detail == nil {
					continue
				}
				note := convertNote(detail, keyword, h.tiebaBaseURL())
				if note == nil {
					continue
				}
				if err := h.Store.SaveNote(ctx, note); err != nil {
					return err
				}
				if h.Config.EnableCheckpoint {
					h.Checkpoint.AddCrawledNote(cp, note.NoteID)
				}
				if h.Config.EnableGetComments {
					if err := h.fetchNoteComments(ctx, note); err != nil {
						return err
					}
				}
				savedCount++
			}
			page++
			if h.Config.EnableCheckpoint {
				cp.CurrentPage = page
				cp.TotalCrawled = savedCount
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}
			if h.Config.CrawlerTimeSleep > 0 {
				time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
			}
		}
	}

	h.Logger.Info("[SearchHandler.Handle] Tieba search completed")
	return nil
}

func (h *SearchHandler) tiebaBaseURL() string {
	return tiebaBaseURL(h.Config)
}

type DetailHandler struct {
	*HandlerDeps
}

func NewDetailHandler(deps *HandlerDeps) *DetailHandler {
	return &DetailHandler{HandlerDeps: deps}
}

func (h *DetailHandler) Name() string {
	return "detail"
}

func (h *DetailHandler) Handle(ctx context.Context) error {
	h.Logger.Info("[DetailHandler.Handle] Begin get tieba specified notes")

	noteIDs := h.detailNoteIDs()
	if len(noteIDs) == 0 {
		h.Logger.Warn("[DetailHandler.Handle] No note IDs specified")
		return nil
	}

	var cp *crawler.Checkpoint
	if h.Config.EnableCheckpoint {
		checkpointKey := makeTiebaHashID("detail", strings.Join(noteIDs, "|"))
		var err error
		cp, err = h.Checkpoint.Load(h.Name(), checkpointKey)
		if err != nil {
			return err
		}
		if cp != nil {
			h.Logger.Info("[DetailHandler.Handle] Resume from checkpoint, crawled: ", len(cp.CrawledNoteIDs))
		} else {
			cp = &crawler.Checkpoint{
				CrawlerType:  h.Name(),
				Keyword:      checkpointKey,
				CurrentPage:  0,
				TotalCrawled: 0,
			}
		}
	}

	for _, noteID := range noteIDs {
		if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, noteID) {
			continue
		}
		if exists, err := h.Store.NoteExists(ctx, noteID); err != nil {
			return err
		} else if exists {
			if h.Config.EnableCheckpoint {
				h.Checkpoint.AddCrawledNote(cp, noteID)
				cp.TotalCrawled = len(cp.CrawledNoteIDs)
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}
			continue
		}
		detail, err := h.Client.GetNoteByID(noteID)
		if err != nil {
			return err
		}
		if detail == nil {
			continue
		}
		note := convertNote(detail, "", tiebaBaseURL(h.Config))
		if note == nil {
			continue
		}
		if err := h.Store.SaveNote(ctx, note); err != nil {
			return err
		}
		if h.Config.EnableGetComments {
			if err := h.fetchNoteComments(ctx, note); err != nil {
				return err
			}
		}
		if h.Config.EnableCheckpoint {
			h.Checkpoint.AddCrawledNote(cp, note.NoteID)
			cp.TotalCrawled = len(cp.CrawledNoteIDs)
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}
	}

	h.Logger.Info("[DetailHandler.Handle] Tieba detail completed")
	return nil
}

func (h *DetailHandler) detailNoteIDs() []string {
	ids := make([]string, 0, len(h.Config.SpecifiedNoteIDs)+len(h.Config.SpecifiedNoteURLs))
	seen := make(map[string]struct{})
	appendID := func(value string) {
		value = tiebaNoteIDFromValue(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}
	for _, noteID := range h.Config.SpecifiedNoteIDs {
		appendID(noteID)
	}
	for _, noteURL := range h.Config.SpecifiedNoteURLs {
		appendID(noteURL)
	}
	return ids
}

func tiebaNoteIDFromValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
		value = parsed.Path
	}
	value = strings.TrimRight(value, "/")
	if idx := strings.Index(value, "?"); idx >= 0 {
		value = value[:idx]
	}
	if idx := strings.LastIndex(value, "/p/"); idx >= 0 {
		value = value[idx+len("/p/"):]
	}
	value = strings.Trim(value, "/")
	if idx := strings.LastIndex(value, "/"); idx >= 0 {
		value = value[idx+1:]
	}
	return strings.TrimSpace(value)
}

func makeTiebaHashID(parts ...any) string {
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, fmt.Sprint(part))
	}
	sum := md5.Sum([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}

type CreatorHandler struct {
	*HandlerDeps
}

func NewCreatorHandler(deps *HandlerDeps) *CreatorHandler {
	return &CreatorHandler{HandlerDeps: deps}
}

func (h *CreatorHandler) Name() string {
	return "creator"
}

func (h *CreatorHandler) Handle(ctx context.Context) error {
	h.Logger.Info("[CreatorHandler.Handle] Begin get tieba creators")

	creatorValues := h.creatorValues()
	if len(creatorValues) == 0 {
		h.Logger.Warn("[CreatorHandler.Handle] No creator URLs specified")
		return nil
	}

	var cp *crawler.Checkpoint
	startCreatorIndex := 0
	if h.Config.EnableCheckpoint {
		var err error
		cp, err = h.Checkpoint.Load(h.Name(), "")
		if err != nil {
			return err
		}
		if cp != nil {
			if cp.CreatorID != "" {
				if index := tiebaIndexString(creatorValues, cp.CreatorID); index >= 0 {
					startCreatorIndex = index
				}
			}
			h.Logger.Info("[CreatorHandler.Handle] Resume from checkpoint, creator: ", cp.CreatorID, ", page: ", cp.CurrentPage)
		} else {
			cp = &crawler.Checkpoint{
				CrawlerType:  h.Name(),
				CurrentPage:  1,
				TotalCrawled: 0,
			}
		}
	}

	for creatorIndex := startCreatorIndex; creatorIndex < len(creatorValues); creatorIndex++ {
		creatorValue := creatorValues[creatorIndex]
		if h.Config.EnableCheckpoint {
			if cp.CreatorID != creatorValue {
				cp.CurrentPage = 1
				cp.TotalCrawled = 0
				cp.CrawledNoteIDs = nil
			}
			cp.CreatorID = creatorValue
			if cp.CurrentPage <= 0 {
				cp.CurrentPage = 1
			}
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}

		creatorUser, err := h.Client.GetCreatorInfoByURL(creatorValue)
		if err != nil {
			return err
		}
		if creatorUser == nil {
			continue
		}
		creator := convertCreator(*creatorUser)
		if creator != nil {
			if err := h.Store.SaveCreator(ctx, creator); err != nil {
				return err
			}
		}
		portrait := tiebaclient.ExtractCreatorPortrait(creatorValue)
		if portrait == "" {
			continue
		}
		if err := h.fetchCreatorNotes(ctx, portrait, cp); err != nil {
			return err
		}
	}

	h.Logger.Info("[CreatorHandler.Handle] Tieba creator completed")
	return nil
}

func (h *CreatorHandler) creatorValues() []string {
	values := make([]string, 0, len(h.Config.SpecifiedCreatorIDs)+len(h.Config.SpecifiedCreatorURLs))
	seen := make(map[string]struct{})
	appendValue := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	for _, value := range h.Config.SpecifiedCreatorIDs {
		appendValue(value)
	}
	for _, value := range h.Config.SpecifiedCreatorURLs {
		appendValue(value)
	}
	return values
}

func tiebaIndexString(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

func (h *CreatorHandler) fetchCreatorNotes(ctx context.Context, portrait string, cp *crawler.Checkpoint) error {
	page := 1
	pageSize := 20
	maxNotes := h.Config.CrawlerMaxNotesCount
	if maxNotes <= 0 {
		maxNotes = pageSize
	}
	savedCount := 0
	if h.Config.EnableCheckpoint && cp != nil {
		if cp.CurrentPage > 0 {
			page = cp.CurrentPage
		}
		savedCount = cp.TotalCrawled
	}
	for savedCount < maxNotes {
		noteIDs, hasMore, err := h.Client.GetNotesByCreatorPortrait(portrait, page, pageSize)
		if err != nil {
			return err
		}
		if len(noteIDs) == 0 {
			return nil
		}
		for _, noteID := range noteIDs {
			if savedCount >= maxNotes {
				break
			}
			if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, noteID) {
				continue
			}
			if exists, err := h.Store.NoteExists(ctx, noteID); err != nil {
				return err
			} else if exists {
				savedCount++
				if h.Config.EnableCheckpoint {
					h.Checkpoint.AddCrawledNote(cp, noteID)
					cp.CurrentPage = page
					cp.TotalCrawled = savedCount
					if err := h.Checkpoint.Save(cp); err != nil {
						return err
					}
				}
				continue
			}
			detail, err := h.Client.GetNoteByID(noteID)
			if err != nil {
				return err
			}
			if detail == nil {
				continue
			}
			note := convertNote(detail, "", tiebaBaseURL(h.Config))
			if note == nil {
				continue
			}
			if err := h.Store.SaveNote(ctx, note); err != nil {
				return err
			}
			if h.Config.EnableGetComments {
				if err := h.fetchNoteComments(ctx, note); err != nil {
					return err
				}
			}
			savedCount++
			if h.Config.EnableCheckpoint {
				h.Checkpoint.AddCrawledNote(cp, note.NoteID)
				cp.CurrentPage = page
				cp.TotalCrawled = savedCount
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}
		}
		page++
		if h.Config.EnableCheckpoint {
			cp.CurrentPage = page
			cp.TotalCrawled = savedCount
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}
		if !hasMore {
			return nil
		}
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}
	return nil
}

type HotSearchHandler struct {
	*HandlerDeps
}

func NewHotSearchHandler(deps *HandlerDeps) *HotSearchHandler {
	return &HotSearchHandler{HandlerDeps: deps}
}

func (h *HotSearchHandler) Name() string {
	return "hot_search"
}

func (h *HotSearchHandler) Handle(ctx context.Context) error {
	h.Logger.Info("[HotSearchHandler.Handle] Tieba hot search placeholder is ready; platform-specific crawling is not implemented yet")
	return nil
}

func tiebaBaseURL(cfg *config.Config) string {
	baseURL := "https://tieba.baidu.com"
	if cfg != nil && cfg.PlatformEndpoints != nil && cfg.PlatformEndpoints.TiebaURL != "" {
		baseURL = cfg.PlatformEndpoints.TiebaURL
	}
	return strings.TrimRight(baseURL, "/")
}

func convertNote(page *tiebamodel.PageResponse, sourceKeyword, baseURL string) *Note {
	noteID := anyToString(page.Thread.ID)
	if noteID == "" {
		noteID = anyToString(page.Thread.TID)
	}
	if noteID == "" {
		noteID = anyToString(page.FirstFloor.TID)
	}
	if noteID == "" {
		return nil
	}
	forum := page.Forum
	if forum.Name == "" {
		forum = page.DisplayForum
	}
	tiebaName := ensureTiebaSuffix(forum.Name)
	author := userByID(page.UserList, anyToString(page.FirstFloor.AuthorID))
	publishTs := anyToInt64(page.FirstFloor.Time)
	if publishTs == 0 {
		publishTs = anyToInt64(page.Thread.CreateTime)
	}
	now := time.Now().UnixMilli()
	return &Note{
		NoteID:          noteID,
		Title:           cleanTitle(firstNonEmpty(page.Thread.Title, page.FirstFloor.Title), tiebaName),
		Desc:            contentText(firstNonNil(page.FirstFloor.Content, page.Thread.OriginThreadInfo.Abstract, page.Thread.OriginThreadInfo.Content)),
		NoteURL:         baseURL + "/p/" + noteID,
		PublishTime:     formatChinaTime(publishTs),
		UserLink:        userLink(baseURL, author),
		UserNickname:    firstNonEmpty(author.NameShow, author.Name),
		UserAvatar:      userAvatar(author),
		TiebaID:         "",
		TiebaName:       tiebaName,
		TiebaLink:       tiebaLink(baseURL, tiebaName),
		TotalReplayNum:  anyToInt(page.Thread.ReplyNum),
		TotalReplayPage: anyToInt(page.Page.TotalPage),
		IPLocation:      author.IPAddress,
		LastModifyTs:    now,
		SourceKeyword:   sourceKeyword,
	}
}

func (h *HandlerDeps) fetchNoteComments(ctx context.Context, note *Note) error {
	if note == nil || note.NoteID == "" {
		return nil
	}
	totalPages := note.TotalReplayPage
	if totalPages <= 0 {
		totalPages = 1
	}
	savedComments := 0
	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		page, err := h.Client.GetNoteComments(note.NoteID, pageNum)
		if err != nil {
			return err
		}
		if page == nil || len(page.PostList) == 0 {
			continue
		}
		rootComments := convertRootComments(page, note, tiebaBaseURL(h.Config))
		for _, comment := range rootComments {
			if err := h.Store.SaveComment(ctx, comment); err != nil {
				return err
			}
			savedComments++
			if h.Config.PerNoteMaxComments > 0 && savedComments >= h.Config.PerNoteMaxComments {
				return nil
			}
			if h.Config.EnableGetSubComments && comment.SubCommentCount > 0 {
				saved, err := h.fetchSubComments(ctx, note, comment)
				if err != nil {
					return err
				}
				savedComments += saved
				if h.Config.PerNoteMaxComments > 0 && savedComments >= h.Config.PerNoteMaxComments {
					return nil
				}
			}
		}
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}
	return nil
}

func (h *HandlerDeps) fetchSubComments(ctx context.Context, note *Note, parent *Comment) (int, error) {
	body, err := h.Client.GetNoteSubComments(note.NoteID, parent.CommentID, parent.TiebaID, 1)
	if err != nil {
		return 0, err
	}
	comments := parseTiebaSubComments(body, note, parent)
	for _, comment := range comments {
		if err := h.Store.SaveComment(ctx, comment); err != nil {
			return 0, err
		}
	}
	return len(comments), nil
}

func convertRootComments(page *tiebamodel.PageResponse, note *Note, baseURL string) []*Comment {
	forum := page.Forum
	if forum.ID == nil && page.DisplayForum.ID != nil {
		forum = page.DisplayForum
	}
	forumID := anyToString(forum.ID)
	now := time.Now().UnixMilli()
	comments := make([]*Comment, 0, len(page.PostList))
	for _, post := range page.PostList {
		commentID := anyToString(post.ID)
		if commentID == "" {
			continue
		}
		author := userByID(page.UserList, anyToString(post.AuthorID))
		comment := &Comment{
			CommentID:       commentID,
			ParentCommentID: "",
			Content:         contentText(post.Content),
			UserLink:        userLink(baseURL, author),
			UserNickname:    firstNonEmpty(author.NameShow, author.Name),
			UserAvatar:      userAvatar(author),
			TiebaID:         forumID,
			TiebaName:       note.TiebaName,
			TiebaLink:       note.TiebaLink,
			PublishTime:     formatChinaTime(anyToInt64(post.Time)),
			IPLocation:      author.IPAddress,
			SubCommentCount: anyToInt(post.SubPostNumber),
			NoteID:          note.NoteID,
			NoteURL:         note.NoteURL,
			LastModifyTs:    now,
		}
		comments = append(comments, comment)
	}
	return comments
}

func convertCreator(user tiebamodel.CreatorUser) *Creator {
	userID := anyToString(user.ID)
	if userID == "" {
		return nil
	}
	now := time.Now().UnixMilli()
	return &Creator{
		UserID:               userID,
		UserName:             user.Name,
		Nickname:             firstNonEmpty(user.NameShow, user.Name),
		Avatar:               userAvatar(tiebamodel.PageUser{Portrait: user.Portrait, UserShowInfo: user.UserShowInfo}),
		IPLocation:           user.IPAddress,
		LastModifyTs:         now,
		Gender:               tiebaGender(anyToInt(user.Sex), anyToInt(user.Gender)),
		Follows:              anyToString(user.ConcernNum),
		Fans:                 anyToString(user.FansNum),
		RegistrationDuration: anyToString(user.TBAge),
	}
}

func tiebaGender(values ...int) string {
	for _, value := range values {
		switch value {
		case 1:
			return "男"
		case 2:
			return "女"
		}
	}
	return "未知"
}

type tiebaSubCommentData struct {
	SPID     string `json:"spid"`
	ShowName string `json:"showname"`
}

var (
	subCommentItemPattern    = regexp.MustCompile(`(?is)<li\b([^>]*)>(.*?)</li>`)
	subCommentDataPattern    = regexp.MustCompile(`(?is)data-field='([^']*)'|data-field="([^"]*)"`)
	subCommentHrefPattern    = regexp.MustCompile(`(?is)<a\b[^>]*href=['"]([^'"]*)['"]`)
	subCommentAvatarPattern  = regexp.MustCompile(`(?is)<img\b[^>]*src=['"]([^'"]*)['"]`)
	subCommentContentPattern = regexp.MustCompile(`(?is)<span\b[^>]*class=['"][^'"]*lzl_content_main[^'"]*['"][^>]*>(.*?)</span>`)
	subCommentTimePattern    = regexp.MustCompile(`(?is)<span\b[^>]*class=['"][^'"]*lzl_time[^'"]*['"][^>]*>(.*?)</span>`)
	htmlTagPattern           = regexp.MustCompile(`(?s)<[^>]+>`)
)

func parseTiebaSubComments(body string, note *Note, parent *Comment) []*Comment {
	matches := subCommentItemPattern.FindAllStringSubmatch(body, -1)
	comments := make([]*Comment, 0, len(matches))
	now := time.Now().UnixMilli()
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		dataField := extractTiebaDataField(match[1])
		if dataField == "" {
			continue
		}
		var data tiebaSubCommentData
		if err := json.Unmarshal([]byte(html.UnescapeString(dataField)), &data); err != nil {
			continue
		}
		if data.SPID == "" {
			continue
		}
		itemHTML := match[2]
		comment := &Comment{
			CommentID:       data.SPID,
			ParentCommentID: parent.CommentID,
			Content:         extractTiebaHTMLText(itemHTML, subCommentContentPattern),
			UserLink:        extractTiebaHTMLAttr(itemHTML, subCommentHrefPattern),
			UserNickname:    data.ShowName,
			UserAvatar:      extractTiebaHTMLAttr(itemHTML, subCommentAvatarPattern),
			TiebaID:         parent.TiebaID,
			TiebaName:       note.TiebaName,
			TiebaLink:       note.TiebaLink,
			PublishTime:     extractTiebaHTMLText(itemHTML, subCommentTimePattern),
			IPLocation:      "",
			SubCommentCount: 0,
			NoteID:          note.NoteID,
			NoteURL:         note.NoteURL,
			LastModifyTs:    now,
		}
		comments = append(comments, comment)
	}
	return comments
}

func extractTiebaDataField(attrs string) string {
	match := subCommentDataPattern.FindStringSubmatch(attrs)
	if len(match) < 3 {
		return ""
	}
	for _, value := range match[1:] {
		if value != "" {
			return value
		}
	}
	return ""
}

func extractTiebaHTMLAttr(body string, pattern *regexp.Regexp) string {
	match := pattern.FindStringSubmatch(body)
	if len(match) < 2 {
		return ""
	}
	return html.UnescapeString(strings.TrimSpace(match[1]))
}

func extractTiebaHTMLText(body string, pattern *regexp.Regexp) string {
	match := pattern.FindStringSubmatch(body)
	if len(match) < 2 {
		return ""
	}
	return normalizeText(html.UnescapeString(htmlTagPattern.ReplaceAllString(match[1], "")))
}

func userByID(users []tiebamodel.PageUser, userID string) tiebamodel.PageUser {
	for _, user := range users {
		if anyToString(user.ID) == userID {
			return user
		}
	}
	return tiebamodel.PageUser{}
}

func userLink(baseURL string, user tiebamodel.PageUser) string {
	if user.Portrait == "" {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + "/home/main?id=" + tiebaQuote(user.Portrait)
}

func userAvatar(user tiebamodel.PageUser) string {
	if user.UserShowInfo.FeedHead.ImageData.ImgURL != "" {
		return user.UserShowInfo.FeedHead.ImageData.ImgURL
	}
	if user.Portrait == "" {
		return ""
	}
	return "https://gss0.bdstatic.com/6LZ1dD3d1sgCo2Kml5_Y_D3/sys/portrait/item/" + user.Portrait
}

func tiebaLink(baseURL, tiebaName string) string {
	if tiebaName == "" {
		return strings.TrimRight(baseURL, "/")
	}
	return strings.TrimRight(baseURL, "/") + "/f?kw=" + tiebaQuote(strings.TrimSuffix(tiebaName, "吧"))
}

func ensureTiebaSuffix(tiebaName string) string {
	tiebaName = strings.TrimSpace(tiebaName)
	if tiebaName == "" || strings.HasSuffix(tiebaName, "吧") {
		return tiebaName
	}
	return tiebaName + "吧"
}

var titleSuffixPattern = regexp.MustCompile(`_(?:百度贴吧|Baidu Tieba)$`)

func cleanTitle(title, tiebaName string) string {
	title = normalizeText(title)
	title = titleSuffixPattern.ReplaceAllString(title, "")
	for _, name := range []string{tiebaName, strings.TrimSuffix(tiebaName, "吧")} {
		if name != "" {
			title = strings.ReplaceAll(title, "【"+name+"】", "")
		}
	}
	return strings.TrimSpace(title)
}

func contentText(content any) string {
	switch typed := content.(type) {
	case string:
		return normalizeText(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := contentItemText(item); text != "" {
				parts = append(parts, text)
			}
		}
		return normalizeText(strings.Join(parts, ""))
	default:
		return ""
	}
}

func contentItemText(item any) string {
	value, ok := item.(map[string]any)
	if !ok {
		return ""
	}
	if text := anyToString(value["text"]); text != "" {
		return text
	}
	return anyToString(value["c"])
}

var whitespacePattern = regexp.MustCompile(`\s+`)

func normalizeText(value string) string {
	return strings.TrimSpace(whitespacePattern.ReplaceAllString(value, " "))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func formatChinaTime(timestamp int64) string {
	if timestamp <= 0 {
		return ""
	}
	return time.Unix(timestamp, 0).In(time.FixedZone("", 8*60*60)).Format("2006-01-02 15:04:05")
}

func tiebaQuote(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

func anyToString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return fmt.Sprintf("%.0f", typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return fmt.Sprint(typed)
	}
}

func anyToInt(value any) int {
	return int(anyToInt64(value))
}

func anyToInt64(value any) int64 {
	if value == nil {
		return 0
	}
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case string:
		parsed, _ := strconv.ParseInt(typed, 10, 64)
		return parsed
	default:
		return 0
	}
}
