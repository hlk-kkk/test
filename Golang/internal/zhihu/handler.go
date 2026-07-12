// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/zhihu/handler.go
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
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/crawler"
	zhihumodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/zhihu"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	zhihuclient "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/zhihu"
)

var htmlTagRegexp = regexp.MustCompile(`<[^>]+>`)

type Handler interface {
	Handle(ctx context.Context) error
	Name() string
}

type HandlerDeps struct {
	Client     *zhihuclient.Client
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
	h.Logger.Info("[SearchHandler.Handle] Begin search zhihu keywords")

	pageSize := 20
	maxNotes := h.Config.CrawlerMaxNotesCount
	if maxNotes < pageSize {
		maxNotes = pageSize
	}

	startKeywordIndex := 0
	var cp *crawler.Checkpoint
	if h.Config.EnableCheckpoint {
		lastCheckpoint, _ := h.Checkpoint.LoadByID("search_latest")
		if lastCheckpoint != nil {
			for i, keyword := range h.Config.Keywords {
				if keyword == lastCheckpoint.Keyword {
					startKeywordIndex = i
					cp = lastCheckpoint
					h.Logger.Info("[SearchHandler.Handle] Resume from checkpoint, keyword: ", keyword, ", page: ", cp.CurrentPage)
					break
				}
			}
		}
	}

	for keywordIndex := startKeywordIndex; keywordIndex < len(h.Config.Keywords); keywordIndex++ {
		keyword := h.Config.Keywords[keywordIndex]
		seenContentIDs := make(map[string]struct{})
		page := h.Config.StartPage
		if page <= 0 {
			page = 1
		}
		savedCount := (page - 1) * pageSize
		if h.Config.EnableCheckpoint {
			if cp == nil || cp.Keyword != keyword {
				cp = &crawler.Checkpoint{
					CrawlerType:  h.Name(),
					Keyword:      keyword,
					CurrentPage:  page,
					TotalCrawled: savedCount,
				}
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
				if err := h.Checkpoint.SaveByID("search_latest", cp); err != nil {
					return err
				}
			}
			page = cp.CurrentPage
			if page <= 0 {
				page = 1
			}
			savedCount = cp.TotalCrawled
		}

		for savedCount <= maxNotes {
			objects, err := h.Client.GetNotesByKeyword(keyword, page, pageSize)
			if err != nil {
				return err
			}
			if len(objects) == 0 {
				break
			}
			objects = filterSeenContentObjects(objects, seenContentIDs)
			if len(objects) == 0 {
				break
			}
			remaining := maxNotes - savedCount
			if remaining <= 0 {
				break
			}
			if len(objects) > remaining {
				objects = objects[:remaining]
			}

			for _, object := range objects {
				content := convertContent(object, keyword, zhihuBaseURL(h.Config))
				if content == nil || content.ContentID == "" {
					continue
				}
				if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, content.ContentID) {
					savedCount++
					continue
				}
				if exists, err := h.Store.ContentExists(ctx, content.ContentID); err != nil {
					return err
				} else if exists {
					if h.Config.EnableCheckpoint {
						h.Checkpoint.AddCrawledNote(cp, content.ContentID)
					}
					savedCount++
					continue
				}
				if err := h.Store.SaveContent(ctx, content); err != nil {
					return err
				}
				if h.Config.EnableCheckpoint {
					h.Checkpoint.AddCrawledNote(cp, content.ContentID)
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
				if err := h.Checkpoint.SaveByID("search_latest", cp); err != nil {
					return err
				}
			}
			if h.Config.CrawlerTimeSleep > 0 {
				time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
			}
		}
		cp = nil
	}

	h.Logger.Info("[SearchHandler.Handle] Zhihu search completed")
	return nil
}

func filterSeenContentObjects(objects []*zhihumodel.ContentObject, seen map[string]struct{}) []*zhihumodel.ContentObject {
	if len(objects) == 0 {
		return nil
	}
	filtered := make([]*zhihumodel.ContentObject, 0, len(objects))
	for _, object := range objects {
		if object == nil {
			continue
		}
		contentID := anyToString(object.ID)
		if contentID == "" || contentID == "0" {
			continue
		}
		if _, ok := seen[contentID]; ok {
			continue
		}
		seen[contentID] = struct{}{}
		filtered = append(filtered, object)
	}
	return filtered
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
	h.Logger.Info("[DetailHandler.Handle] Begin get zhihu specified contents")

	contentURLs := h.detailContentURLs()
	if len(contentURLs) == 0 {
		h.Logger.Warn("[DetailHandler.Handle] No zhihu content URLs specified")
		return nil
	}

	var cp *crawler.Checkpoint
	if h.Config.EnableCheckpoint {
		checkpointKey := makeZhihuHashID("detail", strings.Join(contentURLs, "|"))
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

	for _, contentURL := range contentURLs {
		questionID, answerID := zhihuAnswerIDsFromURL(contentURL)
		if questionID == "" || answerID == "" {
			h.Logger.Warn("[DetailHandler.Handle] Unsupported zhihu detail URL: ", contentURL)
			continue
		}
		if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, answerID) {
			continue
		}
		if exists, err := h.Store.ContentExists(ctx, answerID); err != nil {
			return err
		} else if exists {
			if h.Config.EnableCheckpoint {
				h.Checkpoint.AddCrawledNote(cp, answerID)
				cp.TotalCrawled = len(cp.CrawledNoteIDs)
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}
			continue
		}
		object, err := h.Client.GetAnswerInfo(questionID, answerID)
		if err != nil {
			return err
		}
		content := convertContent(object, "", zhihuBaseURL(h.Config))
		if content == nil || content.ContentID == "" {
			continue
		}
		if err := h.Store.SaveContent(ctx, content); err != nil {
			return err
		}
		if h.Config.EnableGetComments {
			if err := h.fetchContentComments(ctx, content); err != nil {
				return err
			}
		}
		if h.Config.EnableCheckpoint {
			h.Checkpoint.AddCrawledNote(cp, content.ContentID)
			cp.TotalCrawled = len(cp.CrawledNoteIDs)
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}

	h.Logger.Info("[DetailHandler.Handle] Zhihu detail completed")
	return nil
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
	h.Logger.Info("[CreatorHandler.Handle] Begin get zhihu creators")

	creatorTokens := h.creatorURLTokens()
	if len(creatorTokens) == 0 {
		h.Logger.Warn("[CreatorHandler.Handle] No zhihu creators specified")
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
				if index := zhihuIndexString(creatorTokens, cp.CreatorID); index >= 0 {
					startCreatorIndex = index
				}
			}
			h.Logger.Info("[CreatorHandler.Handle] Resume from checkpoint, creator: ", cp.CreatorID, ", offset: ", cp.CurrentPage)
		} else {
			cp = &crawler.Checkpoint{
				CrawlerType:  h.Name(),
				CurrentPage:  0,
				TotalCrawled: 0,
			}
		}
	}

	for creatorIndex := startCreatorIndex; creatorIndex < len(creatorTokens); creatorIndex++ {
		urlToken := creatorTokens[creatorIndex]
		if h.Config.EnableCheckpoint {
			if cp.CreatorID != urlToken {
				cp.CurrentPage = 0
				cp.TotalCrawled = 0
				cp.CrawledNoteIDs = nil
			}
			cp.CreatorID = urlToken
			if cp.CurrentPage < 0 {
				cp.CurrentPage = 0
			}
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}

		info, err := h.Client.GetCreatorInfo(urlToken)
		if err != nil {
			return err
		}
		creator := convertCreator(info, urlToken, zhihuBaseURL(h.Config))
		if creator == nil || creator.UserID == "" {
			h.Logger.Warn("[CreatorHandler.Handle] Zhihu creator not found: ", urlToken)
			continue
		}
		if err := h.Store.SaveCreator(ctx, creator); err != nil {
			return err
		}
		if err := h.fetchCreatorAnswers(ctx, creator, cp); err != nil {
			return err
		}
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}

	h.Logger.Info("[CreatorHandler.Handle] Zhihu creator completed")
	return nil
}

func (h *CreatorHandler) creatorURLTokens() []string {
	values := make([]string, 0, len(h.Config.SpecifiedCreatorURLs)+len(h.Config.SpecifiedCreatorIDs))
	seen := make(map[string]struct{})
	appendValue := func(value string) {
		urlToken := zhihuCreatorURLToken(value)
		if urlToken == "" {
			return
		}
		if _, ok := seen[urlToken]; ok {
			return
		}
		seen[urlToken] = struct{}{}
		values = append(values, urlToken)
	}
	for _, creatorURL := range h.Config.SpecifiedCreatorURLs {
		appendValue(creatorURL)
	}
	for _, creatorID := range h.Config.SpecifiedCreatorIDs {
		appendValue(creatorID)
	}
	return values
}

func zhihuIndexString(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

func (h *CreatorHandler) fetchCreatorAnswers(ctx context.Context, creator *Creator, cp *crawler.Checkpoint) error {
	offset := 0
	limit := 20
	maxNotes := h.Config.CrawlerMaxNotesCount
	savedCount := offset
	baseURL := zhihuBaseURL(h.Config)
	if h.Config.EnableCheckpoint && cp != nil && cp.CreatorID == creator.URLToken {
		if cp.CurrentPage > 0 {
			offset = cp.CurrentPage
		}
		savedCount = cp.TotalCrawled
	}

	for savedCount <= maxNotes {
		resp, err := h.Client.GetCreatorAnswers(creator.URLToken, offset, limit)
		if err != nil {
			return err
		}
		if resp == nil || len(resp.Data) == 0 {
			return nil
		}

		pageContents := make([]*Content, 0, len(resp.Data))
		for _, object := range resp.Data {
			content := convertContent(object, "", baseURL)
			if content == nil || content.ContentID == "" {
				continue
			}
			if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, content.ContentID) {
				continue
			}
			if exists, err := h.Store.ContentExists(ctx, content.ContentID); err != nil {
				return err
			} else if exists {
				savedCount++
				if h.Config.EnableCheckpoint {
					h.Checkpoint.AddCrawledNote(cp, content.ContentID)
					cp.CurrentPage = offset
					cp.TotalCrawled = savedCount
					if err := h.Checkpoint.Save(cp); err != nil {
						return err
					}
				}
				continue
			}
			if err := h.Store.SaveContent(ctx, content); err != nil {
				return err
			}
			pageContents = append(pageContents, content)
			savedCount++
			if h.Config.EnableCheckpoint {
				h.Checkpoint.AddCrawledNote(cp, content.ContentID)
				cp.CurrentPage = offset
				cp.TotalCrawled = savedCount
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}
		}
		if h.Config.EnableGetComments {
			for _, content := range pageContents {
				if err := h.fetchContentComments(ctx, content); err != nil {
					return err
				}
			}
		}

		offset += limit
		if h.Config.EnableCheckpoint {
			cp.CurrentPage = offset
			cp.TotalCrawled = savedCount
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}
		if resp.Paging.IsEnd {
			return nil
		}
		if len(pageContents) == 0 {
			return nil
		}
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}
	return nil
}

func (h *CreatorHandler) fetchContentComments(ctx context.Context, content *Content) error {
	offset := ""
	limit := 10
	for {
		resp, err := h.Client.GetRootComments(content.ContentID, content.ContentType, offset, limit)
		if err != nil {
			return err
		}
		if resp == nil || len(resp.Data) == 0 {
			return nil
		}
		comments := convertComments(content, resp.Data, zhihuBaseURL(h.Config))
		for _, comment := range comments {
			if err := h.Store.SaveComment(ctx, comment); err != nil {
				return err
			}
		}
		if h.Config.EnableGetSubComments {
			for _, comment := range comments {
				if comment.SubCommentCount <= 0 {
					continue
				}
				if err := h.fetchChildComments(ctx, content, comment.CommentID); err != nil {
					return err
				}
			}
		}
		if resp.Paging.IsEnd {
			return nil
		}
		offset = pagingOffset(resp.Paging.Next)
		if offset == "" {
			return nil
		}
	}
}

func (h *CreatorHandler) fetchChildComments(ctx context.Context, content *Content, rootCommentID string) error {
	offset := ""
	limit := 10
	for {
		resp, err := h.Client.GetChildComments(rootCommentID, offset, limit)
		if err != nil {
			return err
		}
		if resp == nil || len(resp.Data) == 0 {
			return nil
		}
		comments := convertComments(content, resp.Data, zhihuBaseURL(h.Config))
		for _, comment := range comments {
			if err := h.Store.SaveComment(ctx, comment); err != nil {
				return err
			}
		}
		if resp.Paging.IsEnd {
			return nil
		}
		offset = pagingOffset(resp.Paging.Next)
		if offset == "" {
			return nil
		}
	}
}

type HomefeedHandler struct {
	*HandlerDeps
}

func NewHomefeedHandler(deps *HandlerDeps) *HomefeedHandler {
	return &HomefeedHandler{HandlerDeps: deps}
}

func (h *HomefeedHandler) Name() string {
	return "homefeed"
}

func (h *HomefeedHandler) Handle(ctx context.Context) error {
	h.Logger.Info("[HomefeedHandler.Handle] Begin get zhihu homefeed")

	pageNumber := 1
	afterID := 0
	endOffset := 0
	sessionToken := ""
	savedCount := 0
	maxNotes := h.Config.CrawlerMaxNotesCount
	baseURL := zhihuBaseURL(h.Config)
	var cp *crawler.Checkpoint
	if h.Config.EnableCheckpoint {
		var err error
		cp, err = h.Checkpoint.Load(h.Name(), "")
		if err != nil {
			return err
		}
		if cp != nil {
			if cp.CurrentPage > 0 {
				pageNumber = cp.CurrentPage
			}
			afterID, endOffset, sessionToken = parseZhihuHomefeedCursor(cp.Cursor)
			savedCount = cp.TotalCrawled
			h.Logger.Info("[HomefeedHandler.Handle] Resume from checkpoint, page: ", pageNumber, ", cursor: ", cp.Cursor)
		} else {
			cp = &crawler.Checkpoint{
				CrawlerType:  h.Name(),
				CurrentPage:  1,
				Cursor:       "",
				TotalCrawled: 0,
			}
		}
		if err := h.Checkpoint.Save(cp); err != nil {
			return err
		}
	}

	for savedCount <= maxNotes {
		resp, err := h.Client.GetHomefeedNotes(pageNumber, afterID, endOffset, sessionToken)
		if err != nil {
			return err
		}
		if resp == nil {
			return nil
		}
		if resp.Paging.IsEnd {
			break
		}

		pageContents := make([]*Content, 0, len(resp.Data))
		for _, item := range resp.Data {
			if item.Type != "feed" || item.Target == nil {
				continue
			}
			content := convertContent(item.Target, "", baseURL)
			if content == nil || content.ContentID == "" {
				continue
			}
			if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, content.ContentID) {
				continue
			}
			if exists, err := h.Store.ContentExists(ctx, content.ContentID); err != nil {
				return err
			} else if exists {
				savedCount++
				if h.Config.EnableCheckpoint {
					h.Checkpoint.AddCrawledNote(cp, content.ContentID)
					cp.CurrentPage = pageNumber
					cp.Cursor = formatZhihuHomefeedCursor(afterID, endOffset, sessionToken)
					cp.TotalCrawled = savedCount
					if err := h.Checkpoint.Save(cp); err != nil {
						return err
					}
				}
				continue
			}
			if err := h.Store.SaveContent(ctx, content); err != nil {
				return err
			}
			pageContents = append(pageContents, content)
			savedCount++
			if h.Config.EnableCheckpoint {
				h.Checkpoint.AddCrawledNote(cp, content.ContentID)
				cp.CurrentPage = pageNumber
				cp.Cursor = formatZhihuHomefeedCursor(afterID, endOffset, sessionToken)
				cp.TotalCrawled = savedCount
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}
		}
		if len(pageContents) == 0 {
			break
		}
		if h.Config.EnableGetComments {
			for _, content := range pageContents {
				if err := h.fetchContentComments(ctx, content); err != nil {
					return err
				}
			}
		}

		nextParams := queryParams(resp.Paging.Next)
		afterID = anyToInt(nextParams.Get("after_id"))
		endOffset = anyToInt(nextParams.Get("end_offset"))
		sessionToken = nextParams.Get("session_token")
		pageNumber++
		if h.Config.EnableCheckpoint {
			cp.CurrentPage = pageNumber
			cp.Cursor = formatZhihuHomefeedCursor(afterID, endOffset, sessionToken)
			cp.TotalCrawled = savedCount
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}

	h.Logger.Info("[HomefeedHandler.Handle] Zhihu homefeed completed")
	return nil
}

func parseZhihuHomefeedCursor(cursor string) (int, int, string) {
	if cursor == "" {
		return 0, 0, ""
	}
	parts := strings.Split(cursor, "|")
	afterID := 0
	endOffset := 0
	sessionToken := ""
	if len(parts) > 0 {
		afterID = anyToInt(parts[0])
	}
	if len(parts) > 1 {
		endOffset = anyToInt(parts[1])
	}
	if len(parts) > 2 {
		sessionToken = parts[2]
	}
	return afterID, endOffset, sessionToken
}

func formatZhihuHomefeedCursor(afterID, endOffset int, sessionToken string) string {
	return fmt.Sprintf("%d|%d|%s", afterID, endOffset, sessionToken)
}

func (h *HomefeedHandler) fetchContentComments(ctx context.Context, content *Content) error {
	offset := ""
	limit := 10
	for {
		resp, err := h.Client.GetRootComments(content.ContentID, content.ContentType, offset, limit)
		if err != nil {
			return err
		}
		if resp == nil || len(resp.Data) == 0 {
			return nil
		}
		comments := convertComments(content, resp.Data, zhihuBaseURL(h.Config))
		for _, comment := range comments {
			if err := h.Store.SaveComment(ctx, comment); err != nil {
				return err
			}
		}
		if h.Config.EnableGetSubComments {
			for _, comment := range comments {
				if comment.SubCommentCount <= 0 {
					continue
				}
				if err := h.fetchChildComments(ctx, content, comment.CommentID); err != nil {
					return err
				}
			}
		}
		if resp.Paging.IsEnd {
			return nil
		}
		offset = pagingOffset(resp.Paging.Next)
		if offset == "" {
			return nil
		}
	}
}

func (h *HomefeedHandler) fetchChildComments(ctx context.Context, content *Content, rootCommentID string) error {
	offset := ""
	limit := 10
	for {
		resp, err := h.Client.GetChildComments(rootCommentID, offset, limit)
		if err != nil {
			return err
		}
		if resp == nil || len(resp.Data) == 0 {
			return nil
		}
		comments := convertComments(content, resp.Data, zhihuBaseURL(h.Config))
		for _, comment := range comments {
			if err := h.Store.SaveComment(ctx, comment); err != nil {
				return err
			}
		}
		if resp.Paging.IsEnd {
			return nil
		}
		offset = pagingOffset(resp.Paging.Next)
		if offset == "" {
			return nil
		}
	}
}

type HotSearchHandler struct {
	*HandlerDeps
}

const (
	zhihuHotSearchCheckpointID           = "hot_search_latest"
	zhihuHotSearchSnapshotItemsExtraKey  = "hot_search_snapshot_items"
	zhihuHotSearchSnapshotTsExtraKey     = "hot_search_snapshot_ts"
	zhihuHotSearchSnapshotLimitExtraKey  = "hot_search_snapshot_limit"
	zhihuHotSearchSnapshotPeriodExtraKey = "hot_search_snapshot_period"
	zhihuHotSearchContentIDsExtraPrefix  = "hot_search_content_ids_"
)

func NewHotSearchHandler(deps *HandlerDeps) *HotSearchHandler {
	return &HotSearchHandler{HandlerDeps: deps}
}

func (h *HotSearchHandler) Name() string {
	return "hot_search"
}

func (h *HotSearchHandler) Handle(ctx context.Context) error {
	h.Logger.Info("[HotSearchHandler.Handle] Begin get zhihu hot search")

	maxItems := h.Config.HotSearchMaxItemsCount
	if maxItems <= 0 {
		maxItems = 50
	}
	perItemCount := h.Config.HotSearchPerItemContentCount
	if perItemCount <= 0 {
		perItemCount = 4
	}

	items, snapshotTs, startIndex, cp, err := h.loadOrCreateHotSearchSnapshot(maxItems)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		h.Logger.Info("[HotSearchHandler.Handle] No hot search contents")
		if h.Config.EnableCheckpoint {
			_ = h.Checkpoint.Clear(zhihuHotSearchCheckpointID)
		}
		return nil
	}

	savedCount := 0
	if cp != nil {
		savedCount = cp.TotalCrawled
	}
	for index, item := range items {
		if index >= maxItems {
			break
		}
		if index < startIndex {
			continue
		}
		questionID := anyToString(item.Question.ID)
		title := extractText(item.Question.Title)
		if questionID == "" || title == "" {
			if err := h.advanceHotSearchCheckpoint(cp, index+1, savedCount); err != nil {
				return err
			}
			continue
		}

		hotItem := buildZhihuHotSearchItem(item, index+1, snapshotTs)
		if hotItem == nil {
			continue
		}
		if err := h.Store.SaveHotSearchItem(ctx, hotItem); err != nil {
			return err
		}

		contentIDs, err := h.fetchHotQuestionAnswers(ctx, cp, hotItem, questionID, title, perItemCount)
		if err != nil {
			return err
		}
		if cp != nil {
			savedCount = cp.TotalCrawled
		} else {
			savedCount += len(contentIDs)
		}
		if err := h.advanceHotSearchCheckpoint(cp, index+1, savedCount); err != nil {
			return err
		}
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}

	if h.Config.EnableCheckpoint {
		_ = h.Checkpoint.Clear(zhihuHotSearchCheckpointID)
	}
	h.Logger.Info("[HotSearchHandler.Handle] Saved ", savedCount, " zhihu contents from hot search questions")
	return nil
}

func (h *HotSearchHandler) loadOrCreateHotSearchSnapshot(maxItems int) ([]zhihumodel.HotSearchItem, int64, int, *crawler.Checkpoint, error) {
	if h.Config.EnableCheckpoint {
		cp, err := h.Checkpoint.LoadByID(zhihuHotSearchCheckpointID)
		if err != nil {
			return nil, 0, 0, nil, err
		}
		if cp != nil && cp.CrawlerType == h.Name() {
			items, snapshotTs, ok := hotSearchSnapshotFromCheckpoint(cp, h.Config.HotSearchCheckpointMaxAgeSeconds)
			if ok {
				if len(items) > maxItems {
					items = items[:maxItems]
				}
				startIndex := cp.CurrentPage
				if startIndex < 0 {
					startIndex = 0
				}
				if startIndex > len(items) {
					startIndex = len(items)
				}
				h.Logger.Info("[HotSearchHandler.Handle] Resume hot search snapshot, question index: ", startIndex)
				return items, snapshotTs, startIndex, cp, nil
			}
		}
	}

	resp, err := h.Client.GetHotSearchItems(0, maxItems)
	if err != nil {
		return nil, 0, 0, nil, err
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, 0, 0, nil, nil
	}
	items := resp.Data
	if len(items) > maxItems {
		items = items[:maxItems]
	}
	snapshotTs := time.Now().UnixMilli()
	if !h.Config.EnableCheckpoint {
		return items, snapshotTs, 0, nil, nil
	}

	body, err := json.Marshal(items)
	if err != nil {
		return nil, 0, 0, nil, err
	}
	cp := &crawler.Checkpoint{
		CrawlerType:  h.Name(),
		CurrentPage:  0,
		TotalCrawled: 0,
		Extra: map[string]string{
			zhihuHotSearchSnapshotItemsExtraKey:  string(body),
			zhihuHotSearchSnapshotTsExtraKey:     fmt.Sprint(snapshotTs),
			zhihuHotSearchSnapshotLimitExtraKey:  fmt.Sprint(maxItems),
			zhihuHotSearchSnapshotPeriodExtraKey: "hour",
		},
	}
	if err := h.Checkpoint.SaveByID(zhihuHotSearchCheckpointID, cp); err != nil {
		return nil, 0, 0, nil, err
	}
	return items, snapshotTs, 0, cp, nil
}

func hotSearchSnapshotFromCheckpoint(cp *crawler.Checkpoint, maxAgeSeconds int) ([]zhihumodel.HotSearchItem, int64, bool) {
	if cp == nil || cp.Extra == nil {
		return nil, 0, false
	}
	rawItems := cp.Extra[zhihuHotSearchSnapshotItemsExtraKey]
	if rawItems == "" {
		return nil, 0, false
	}
	var items []zhihumodel.HotSearchItem
	if err := json.Unmarshal([]byte(rawItems), &items); err != nil {
		return nil, 0, false
	}
	var snapshotTs int64
	_, _ = fmt.Sscan(cp.Extra[zhihuHotSearchSnapshotTsExtraKey], &snapshotTs)
	if snapshotTs == 0 {
		return nil, 0, false
	}
	if maxAgeSeconds > 0 && hotSearchSnapshotAgeMillis(snapshotTs) > int64(maxAgeSeconds)*1000 {
		return nil, 0, false
	}
	return items, snapshotTs, true
}

func hotSearchSnapshotAgeMillis(snapshotTs int64) int64 {
	snapshotTsMillis := snapshotTs
	if snapshotTsMillis > 0 && snapshotTsMillis < 1_000_000_000_000 {
		snapshotTsMillis *= 1000
	}
	return time.Now().UnixMilli() - snapshotTsMillis
}

func (h *HotSearchHandler) advanceHotSearchCheckpoint(cp *crawler.Checkpoint, nextIndex, totalCrawled int) error {
	if !h.Config.EnableCheckpoint || cp == nil {
		return nil
	}
	cp.CurrentPage = nextIndex
	cp.TotalCrawled = totalCrawled
	return h.Checkpoint.SaveByID(zhihuHotSearchCheckpointID, cp)
}

func (h *HotSearchHandler) fetchHotQuestionAnswers(ctx context.Context, cp *crawler.Checkpoint, hotItem *HotSearchItem, questionID, sourceKeyword string, maxAnswers int) ([]string, error) {
	cursor := ""
	sessionID := ""
	offset := 0
	limit := 5
	order := "default"
	baseURL := zhihuBaseURL(h.Config)
	detailHandler := &DetailHandler{HandlerDeps: h.HandlerDeps}
	storedContentIDs := hotSearchContentIDsFromCheckpoint(cp, hotItem)
	storedContentIDSet := make(map[string]struct{}, len(storedContentIDs))
	for _, contentID := range storedContentIDs {
		storedContentIDSet[contentID] = struct{}{}
	}
	newContentIDs := make([]string, 0, maxAnswers)
	collectedCount := len(storedContentIDSet)

	for collectedCount < maxAnswers {
		resp, err := h.Client.GetQuestionFeeds(questionID, cursor, sessionID, limit, offset, order)
		if err != nil {
			return newContentIDs, err
		}
		if resp == nil || len(resp.Data) == 0 {
			return newContentIDs, nil
		}

		for _, feed := range resp.Data {
			if collectedCount >= maxAnswers {
				break
			}
			if feed.TargetType != "answer" || feed.Target == nil {
				continue
			}
			target := *feed.Target
			if target.Type == "" {
				target.Type = feed.TargetType
			}
			content := convertContent(&target, sourceKeyword, baseURL)
			if content == nil || content.ContentID == "" {
				continue
			}
			if _, ok := storedContentIDSet[content.ContentID]; ok {
				continue
			}
			if err := h.Store.SaveContent(ctx, content); err != nil {
				return newContentIDs, err
			}
			relation := buildZhihuHotSearchRelation(hotItem, content.ContentID, content.ContentType, sourceKeyword)
			if err := h.Store.SaveHotSearchRelation(ctx, relation); err != nil {
				return newContentIDs, err
			}
			storedContentIDSet[content.ContentID] = struct{}{}
			storedContentIDs = append(storedContentIDs, content.ContentID)
			collectedCount++
			newContentIDs = append(newContentIDs, content.ContentID)
			if err := h.saveHotSearchContentIDs(cp, hotItem, storedContentIDs); err != nil {
				return newContentIDs, err
			}
			if h.Config.EnableGetComments {
				if err := detailHandler.fetchContentComments(ctx, content); err != nil {
					return newContentIDs, err
				}
			}
		}
		if resp.Paging.IsEnd {
			return newContentIDs, nil
		}
		nextParams := queryParams(resp.Paging.Next)
		if len(nextParams) == 0 {
			return newContentIDs, nil
		}
		cursor = nextParams.Get("cursor")
		sessionID = nextParams.Get("session_id")
		offset = anyToInt(nextParams.Get("offset"))
		if nextLimit := anyToInt(nextParams.Get("limit")); nextLimit > 0 {
			limit = nextLimit
		}
		order = firstNonEmpty(nextParams.Get("order"), order)
	}
	return newContentIDs, nil
}

func hotSearchContentIDsFromCheckpoint(cp *crawler.Checkpoint, hotItem *HotSearchItem) []string {
	if cp == nil || hotItem == nil || cp.Extra == nil {
		return nil
	}
	rawIDs := cp.Extra[zhihuHotSearchContentIDsExtraPrefix+hotItem.HotItemID]
	if rawIDs == "" {
		return nil
	}
	var contentIDs []string
	if err := json.Unmarshal([]byte(rawIDs), &contentIDs); err != nil {
		return nil
	}
	deduped := make([]string, 0, len(contentIDs))
	seen := make(map[string]struct{}, len(contentIDs))
	for _, contentID := range contentIDs {
		contentID = strings.TrimSpace(contentID)
		if contentID == "" {
			continue
		}
		if _, ok := seen[contentID]; ok {
			continue
		}
		seen[contentID] = struct{}{}
		deduped = append(deduped, contentID)
	}
	return deduped
}

func (h *HotSearchHandler) saveHotSearchContentIDs(cp *crawler.Checkpoint, hotItem *HotSearchItem, contentIDs []string) error {
	if !h.Config.EnableCheckpoint || cp == nil || hotItem == nil {
		return nil
	}
	if cp.Extra == nil {
		cp.Extra = make(map[string]string)
	}
	body, err := json.Marshal(contentIDs)
	if err != nil {
		return err
	}
	cp.Extra[zhihuHotSearchContentIDsExtraPrefix+hotItem.HotItemID] = string(body)
	for _, contentID := range contentIDs {
		h.Checkpoint.AddCrawledNote(cp, contentID)
	}
	cp.TotalCrawled = len(cp.CrawledNoteIDs)
	return h.Checkpoint.SaveByID(zhihuHotSearchCheckpointID, cp)
}

func buildZhihuHotSearchItem(item zhihumodel.HotSearchItem, rankNo int, snapshotTs int64) *HotSearchItem {
	questionID := anyToString(item.Question.ID)
	title := extractText(item.Question.Title)
	if questionID == "" || title == "" {
		return nil
	}
	rawJSON := ""
	if body, err := json.Marshal(item); err == nil {
		rawJSON = string(body)
	}
	snapshotDate := time.UnixMilli(snapshotTs).Local().Format("2006-01-02")
	hotValue := anyToString(item.Score)
	if hotValue == "" || hotValue == "0" {
		hotValue = anyToString(item.Heat)
	}
	return &HotSearchItem{
		HotItemID:    makeZhihuHashID("zhihu", "hot_search", snapshotDate, snapshotTs, rankNo, title),
		Platform:     "zhihu",
		BoardName:    "hot_search",
		BoardType:    "question",
		SnapshotDate: snapshotDate,
		SnapshotTs:   snapshotTs,
		RankNo:       rankNo,
		Title:        title,
		Keyword:      title,
		HotValue:     hotValue,
		Badge:        item.Label,
		TopicID:      questionID,
		TopicURL:     "https://www.zhihu.com/question/" + questionID,
		RawJSON:      rawJSON,
	}
}

func buildZhihuHotSearchRelation(item *HotSearchItem, contentID, contentType, sourceKeyword string) *HotSearchContentRelation {
	return &HotSearchContentRelation{
		RelID:         makeZhihuHashID(item.HotItemID, item.Platform, "zhihu_content", contentID, contentType),
		HotItemID:     item.HotItemID,
		Platform:      item.Platform,
		ContentID:     contentID,
		ContentType:   contentType,
		ContentTable:  "zhihu_content",
		RankNo:        item.RankNo,
		SourceKeyword: firstNonEmpty(sourceKeyword, item.Keyword, item.Title),
	}
}

func makeZhihuHashID(parts ...any) string {
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, fmt.Sprint(part))
	}
	sum := md5.Sum([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}

func (h *DetailHandler) detailContentURLs() []string {
	values := make([]string, 0, len(h.Config.SpecifiedNoteURLs)+len(h.Config.SpecifiedNoteIDs))
	seen := make(map[string]struct{})
	appendValue := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		value = strings.Split(value, "?")[0]
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	for _, noteURL := range h.Config.SpecifiedNoteURLs {
		appendValue(noteURL)
	}
	for _, noteID := range h.Config.SpecifiedNoteIDs {
		appendValue(noteID)
	}
	return values
}

func zhihuAnswerIDsFromURL(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Path != "" {
		value = parsed.Path
	}
	value = strings.TrimRight(value, "/")
	parts := strings.Split(value, "/")
	for idx, part := range parts {
		if part == "question" && idx+3 < len(parts) && parts[idx+2] == "answer" {
			return parts[idx+1], parts[idx+3]
		}
	}
	return "", ""
}

func (h *DetailHandler) fetchContentComments(ctx context.Context, content *Content) error {
	offset := ""
	limit := 10
	for {
		resp, err := h.Client.GetRootComments(content.ContentID, content.ContentType, offset, limit)
		if err != nil {
			return err
		}
		if resp == nil || len(resp.Data) == 0 {
			return nil
		}
		comments := convertComments(content, resp.Data, zhihuBaseURL(h.Config))
		for _, comment := range comments {
			if err := h.Store.SaveComment(ctx, comment); err != nil {
				return err
			}
		}
		if h.Config.EnableGetSubComments {
			for _, comment := range comments {
				if comment.SubCommentCount <= 0 {
					continue
				}
				if err := h.fetchChildComments(ctx, content, comment.CommentID); err != nil {
					return err
				}
			}
		}
		if resp.Paging.IsEnd {
			return nil
		}
		offset = pagingOffset(resp.Paging.Next)
		if offset == "" {
			return nil
		}
	}
}

func (h *DetailHandler) fetchChildComments(ctx context.Context, content *Content, rootCommentID string) error {
	offset := ""
	limit := 10
	for {
		resp, err := h.Client.GetChildComments(rootCommentID, offset, limit)
		if err != nil {
			return err
		}
		if resp == nil || len(resp.Data) == 0 {
			return nil
		}
		comments := convertComments(content, resp.Data, zhihuBaseURL(h.Config))
		for _, comment := range comments {
			if err := h.Store.SaveComment(ctx, comment); err != nil {
				return err
			}
		}
		if resp.Paging.IsEnd {
			return nil
		}
		offset = pagingOffset(resp.Paging.Next)
		if offset == "" {
			return nil
		}
	}
}

func convertContent(object *zhihumodel.ContentObject, sourceKeyword, baseURL string) *Content {
	if object == nil {
		return nil
	}
	switch object.Type {
	case "answer":
		return convertAnswer(object, sourceKeyword, baseURL)
	case "article":
		return convertArticle(object, sourceKeyword, baseURL)
	case "zvideo":
		return convertZvideo(object, sourceKeyword, baseURL)
	default:
		return nil
	}
}

func convertAnswer(answer *zhihumodel.ContentObject, sourceKeyword, baseURL string) *Content {
	contentID := anyToString(answer.ID)
	questionID := anyToString(answer.Question.ID)
	author := extractAuthor(answer.Author, baseURL)
	return &Content{
		ContentID:     contentID,
		ContentType:   answer.Type,
		ContentText:   extractText(answer.Content),
		ContentURL:    fmt.Sprintf("%s/question/%s/answer/%s", baseURL, questionID, contentID),
		QuestionID:    questionID,
		Title:         extractText(answer.Title),
		Desc:          extractText(firstNonEmpty(answer.Description, answer.Excerpt)),
		CreatedTime:   firstNonEmpty(anyToString(answer.CreatedTime), anyToString(answer.CreatedTimeCamel)),
		UpdatedTime:   firstNonEmpty(anyToString(answer.UpdatedTime), anyToString(answer.UpdatedTimeCamel)),
		VoteupCount:   anyToInt(firstAny(answer.VoteupCount, answer.VoteupCountCamel)),
		CommentCount:  anyToInt(firstAny(answer.CommentCount, answer.CommentCountCamel)),
		SourceKeyword: sourceKeyword,
		UserID:        author.userID,
		UserLink:      author.userLink,
		UserNickname:  author.nickname,
		UserAvatar:    author.avatar,
		UserURLToken:  author.urlToken,
	}
}

func convertArticle(article *zhihumodel.ContentObject, sourceKeyword, baseURL string) *Content {
	contentID := anyToString(article.ID)
	author := extractAuthor(article.Author, baseURL)
	return &Content{
		ContentID:     contentID,
		ContentType:   article.Type,
		ContentText:   extractText(article.Content),
		ContentURL:    fmt.Sprintf("%s/p/%s", baseURL, contentID),
		Title:         extractText(article.Title),
		Desc:          extractText(article.Excerpt),
		CreatedTime:   firstNonEmpty(anyToString(article.CreatedTime), anyToString(article.Created), anyToString(article.CreatedTimeCamel)),
		UpdatedTime:   firstNonEmpty(anyToString(article.UpdatedTime), anyToString(article.Updated), anyToString(article.UpdatedTimeCamel)),
		VoteupCount:   anyToInt(firstAny(article.VoteupCount, article.VoteupCountCamel)),
		CommentCount:  anyToInt(firstAny(article.CommentCount, article.CommentCountCamel)),
		SourceKeyword: sourceKeyword,
		UserID:        author.userID,
		UserLink:      author.userLink,
		UserNickname:  author.nickname,
		UserAvatar:    author.avatar,
		UserURLToken:  author.urlToken,
	}
}

func convertZvideo(zvideo *zhihumodel.ContentObject, sourceKeyword, baseURL string) *Content {
	contentID := anyToString(zvideo.ID)
	author := extractAuthor(zvideo.Author, baseURL)
	return &Content{
		ContentID:     contentID,
		ContentType:   zvideo.Type,
		ContentURL:    fmt.Sprintf("%s/zvideo/%s", baseURL, contentID),
		Title:         extractText(zvideo.Title),
		Desc:          extractText(zvideo.Description),
		CreatedTime:   firstNonEmpty(anyToString(zvideo.CreatedTime), anyToString(zvideo.CreatedTimeCamel)),
		UpdatedTime:   firstNonEmpty(anyToString(zvideo.UpdatedTime), anyToString(zvideo.UpdatedTimeCamel)),
		VoteupCount:   anyToInt(firstAny(zvideo.VoteupCount, zvideo.VoteupCountCamel)),
		CommentCount:  anyToInt(firstAny(zvideo.CommentCount, zvideo.CommentCountCamel)),
		SourceKeyword: sourceKeyword,
		UserID:        author.userID,
		UserLink:      author.userLink,
		UserNickname:  author.nickname,
		UserAvatar:    author.avatar,
		UserURLToken:  author.urlToken,
	}
}

func convertComments(content *Content, objects []zhihumodel.CommentObject, baseURL string) []*Comment {
	comments := make([]*Comment, 0, len(objects))
	for _, object := range objects {
		if object.Type != "comment" {
			continue
		}
		author := extractAuthor(object.Author, baseURL)
		comments = append(comments, &Comment{
			CommentID:       anyToString(object.ID),
			ParentCommentID: anyToString(object.ReplyCommentID),
			Content:         extractText(object.Content),
			PublishTime:     anyToString(object.CreatedTime),
			IPLocation:      commentIPLocation(object.CommentTag),
			SubCommentCount: anyToInt(object.ChildCommentCount),
			LikeCount:       anyToInt(object.LikeCount),
			DislikeCount:    anyToInt(object.DislikeCount),
			ContentID:       content.ContentID,
			ContentType:     content.ContentType,
			UserID:          author.userID,
			UserLink:        author.userLink,
			UserNickname:    author.nickname,
			UserAvatar:      author.avatar,
		})
	}
	return comments
}

func convertCreator(info *zhihumodel.CreatorInfo, urlToken, baseURL string) *Creator {
	if info == nil {
		return nil
	}
	return &Creator{
		UserID:         info.ID,
		UserLink:       baseURL + "/people/" + urlToken,
		UserNickname:   info.Name,
		UserAvatar:     firstNonEmpty(info.AvatarURL, info.AvatarURLSnake),
		URLToken:       firstNonEmpty(info.URLToken, info.URLTokenSnake, urlToken),
		Gender:         zhihuGender(info.Gender),
		IPLocation:     info.IPInfo,
		Follows:        anyToInt(info.FollowingCount),
		Fans:           anyToInt(info.FollowerCount),
		AnwserCount:    anyToInt(info.AnswerCount),
		VideoCount:     anyToInt(info.ZvideoCount),
		QuestionCount:  anyToInt(info.QuestionCount),
		ArticleCount:   anyToInt(info.ArticlesCount),
		ColumnCount:    anyToInt(info.ColumnsCount),
		GetVoteupCount: anyToInt(info.VoteupCount),
	}
}

func zhihuCreatorURLToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.Split(value, "?")[0]
	if parsed, err := url.Parse(value); err == nil && parsed.Path != "" {
		value = parsed.Path
	}
	value = strings.Trim(value, "/")
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "/")
	return parts[len(parts)-1]
}

func zhihuGender(value any) string {
	if value == nil || strings.TrimSpace(anyToString(value)) == "" {
		return "未知"
	}
	switch anyToString(value) {
	case "1":
		return "男"
	case "0":
		return "女"
	default:
		return "未知"
	}
}

func commentIPLocation(tags []zhihumodel.CommentTag) string {
	for _, tag := range tags {
		if tag.Type == "ip_info" {
			return tag.Text
		}
	}
	return ""
}

func pagingOffset(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("offset")
}

func queryParams(rawURL string) url.Values {
	if rawURL == "" {
		return url.Values{}
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return url.Values{}
	}
	return parsed.Query()
}

type extractedAuthor struct {
	userID   string
	userLink string
	nickname string
	avatar   string
	urlToken string
}

func extractAuthor(author zhihumodel.Author, baseURL string) extractedAuthor {
	if author.IsAnonymous {
		return extractedAuthor{
			userID:   "ANONYMOUS",
			nickname: "匿名用户",
			avatar:   firstNonEmpty(author.AvatarURL, author.AvatarURL2),
		}
	}
	if author.ID == "" && author.Member != nil {
		author = *author.Member
	}
	urlToken := firstNonEmpty(author.URLToken, author.URLToken2)
	userLink := ""
	if urlToken != "" {
		userLink = baseURL + "/people/" + urlToken
	}
	return extractedAuthor{
		userID:   author.ID,
		userLink: userLink,
		nickname: author.Name,
		avatar:   firstNonEmpty(author.AvatarURL, author.AvatarURL2),
		urlToken: urlToken,
	}
}

func extractText(value string) string {
	value = htmlTagRegexp.ReplaceAllString(value, "")
	value = html.UnescapeString(value)
	return strings.TrimSpace(value)
}

func zhihuBaseURL(cfg *config.Config) string {
	baseURL := "https://www.zhihu.com"
	if cfg != nil && cfg.PlatformEndpoints != nil && cfg.PlatformEndpoints.ZhihuURL != "" {
		baseURL = cfg.PlatformEndpoints.ZhihuURL
	}
	return strings.TrimRight(baseURL, "/")
}

func anyToString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case jsonNumber:
		return string(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprint(typed)
	case int:
		return fmt.Sprint(typed)
	case int64:
		return fmt.Sprint(typed)
	case uint64:
		return fmt.Sprint(typed)
	default:
		return fmt.Sprint(typed)
	}
}

type jsonNumber string

func anyToInt(value any) int {
	switch typed := value.(type) {
	case nil:
		return 0
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		var parsed int
		_, _ = fmt.Sscan(typed, &parsed)
		return parsed
	default:
		var parsed int
		_, _ = fmt.Sscan(fmt.Sprint(typed), &parsed)
		return parsed
	}
}

func firstAny(values ...any) any {
	for _, value := range values {
		if anyToString(value) != "" && anyToString(value) != "0" {
			return value
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
