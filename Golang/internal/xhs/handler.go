// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/xhs/handler.go
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

// internal/xhs/handler.go
package xhs

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/crawler"
	xhsmodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/xiaohongshu"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	xhsclient "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/xhs"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/worker"
)

// Handler 定义爬取处理器接口
type Handler interface {
	// Handle 执行爬取任务
	Handle(ctx context.Context) error
	// Name 返回handler名称
	Name() string
}

// HandlerDeps 所有Handler共享的依赖
type HandlerDeps struct {
	Client     *xhsclient.Client
	Processor  *Processor
	Checkpoint *crawler.CheckpointManager
	WorkerPool *worker.Pool
	Config     *config.Config
	Logger     *tools.Logger
}

// NewHandlerDeps 创建Handler依赖
func NewHandlerDeps(
	client *xhsclient.Client,
	processor *Processor,
	checkpoint *crawler.CheckpointManager,
	workerPool *worker.Pool,
	cfg *config.Config,
) *HandlerDeps {
	return &HandlerDeps{
		Client:     client,
		Processor:  processor,
		Checkpoint: checkpoint,
		WorkerPool: workerPool,
		Config:     cfg,
		Logger:     tools.GetLogger(),
	}
}

// NoteItem 帖子项（用于批量处理）
type NoteItem struct {
	NoteID     string
	XsecToken  string
	XsecSource string
}

// parseNoteInfoFromURL 从小红书笔记URL中解析出笔记信息
// URL格式: https://www.xiaohongshu.com/explore/{note_id}?xsec_token=xxx&xsec_source=xxx
func parseNoteInfoFromURL(noteURL string) NoteItem {
	// 提取 note_id: URL路径的最后一段
	parts := strings.Split(noteURL, "/")
	lastPart := parts[len(parts)-1]
	noteID := strings.Split(lastPart, "?")[0]

	// 解析查询参数
	xsecToken := ""
	xsecSource := ""
	if u, err := url.Parse(noteURL); err == nil {
		xsecToken = u.Query().Get("xsec_token")
		xsecSource = u.Query().Get("xsec_source")
	}

	return NoteItem{
		NoteID:     noteID,
		XsecToken:  xsecToken,
		XsecSource: xsecSource,
	}
}

// CreatorURLInfo 创作者URL解析信息
type CreatorURLInfo struct {
	CreatorID  string
	XsecToken  string
	XsecSource string
}

// parseCreatorInfoFromURL 从小红书创作者主页URL中解析出创作者信息
// URL格式: https://www.xiaohongshu.com/user/profile/{creator_id}?xsec_token=xxx&xsec_source=xxx
func parseCreatorInfoFromURL(creatorURL string) CreatorURLInfo {
	parts := strings.Split(creatorURL, "/")
	lastPart := parts[len(parts)-1]
	creatorID := strings.Split(lastPart, "?")[0]

	xsecToken := ""
	xsecSource := ""
	if u, err := url.Parse(creatorURL); err == nil {
		xsecToken = u.Query().Get("xsec_token")
		xsecSource = u.Query().Get("xsec_source")
	}

	return CreatorURLInfo{
		CreatorID:  creatorID,
		XsecToken:  xsecToken,
		XsecSource: xsecSource,
	}
}

// ========== SearchHandler ==========

// SearchHandler 搜索爬取处理器
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
	h.Logger.Info("[SearchHandler.Handle] Begin search xhs")

	keywords := h.Config.Keywords
	maxNotes := h.Config.CrawlerMaxNotesCount
	pageSize := 20

	// 尝试加载断点,支持跨关键词恢复
	var startKeywordIndex int = 0
	var cp *crawler.Checkpoint

	if h.Config.EnableCheckpoint {
		lastCheckpoint, _ := h.Checkpoint.LoadByID("search_latest")
		if lastCheckpoint != nil {
			// 找到上次爬取的关键词在列表中的位置
			for i, kw := range keywords {
				if kw == lastCheckpoint.Keyword {
					startKeywordIndex = i
					cp = lastCheckpoint
					h.Logger.Info("[SearchHandler.Handle] Resume from checkpoint, keyword: ", kw, ", page: ", cp.CurrentPage)
					break
				}
			}
			if startKeywordIndex == 0 && lastCheckpoint.Keyword != keywords[0] {
				h.Logger.Warn("[SearchHandler.Handle] Checkpoint keyword not found, starting from first keyword")
			}
		}
	}

	// 从断点位置开始爬取
	for i := startKeywordIndex; i < len(keywords); i++ {
		keyword := keywords[i]
		h.Logger.Info("[SearchHandler.Handle] Searching keyword: ", keyword)

		// 如果是新关键词,创建新的checkpoint并重置页码
		if cp == nil || cp.Keyword != keyword {
			cp = &crawler.Checkpoint{
				CrawlerType:  h.Name(),
				Keyword:      keyword,
				CurrentPage:  1,
				TotalCrawled: 0,
			}
			// 先保存checkpoint,确保后续可以恢复
			if h.Config.EnableCheckpoint {
				h.Checkpoint.Save(cp)
				h.Checkpoint.SaveByID("search_latest", cp)
			}
		}

		page := cp.CurrentPage
		crawledCount := cp.TotalCrawled

		for crawledCount < maxNotes {
			h.Logger.Info("[SearchHandler.Handle] Searching page: ", page)

			// 搜索
			searchResp, err := h.Client.SearchNotesByKeyword(&xhsmodel.SearchRequest{
				Keyword:  keyword,
				Page:     page,
				PageSize: pageSize,
				SearchID: tools.GetSearchID(),
				Sort:     xhsmodel.SearchSortTypeGeneral,
				NoteType: xhsmodel.SearchNoteTypeAll,
			})
			if err != nil {
				h.Logger.Error("[SearchHandler.Handle] Search error: ", err)
				// 打印当前进度,便于恢复
				h.Logger.Error("==================== Current Progress ====================")
				for j := 0; j < 3; j++ {
					h.Logger.Error("[SearchHandler.Handle] Current keyword: ", keyword, ", page: ", page)
				}
				h.Logger.Error("==================== Current Progress ====================")
				h.Logger.Info("[SearchHandler.Handle] You can enable checkpoint in config to resume from here")
				return err
			}

			if !searchResp.HasMore || len(searchResp.Items) == 0 {
				h.Logger.Info("[SearchHandler.Handle] No more data for keyword: ", keyword)
				break
			}

			// 过滤有效帖子
			noteItems := make([]NoteItem, 0)
			for _, item := range searchResp.Items {
				if item.ModelType == "rec_query" || item.ModelType == "hot_query" {
					continue
				}
				noteItems = append(noteItems, NoteItem{
					NoteID:     item.ID,
					XsecToken:  item.XsecToken,
					XsecSource: "pc_search",
				})
			}

			// 限制爬取数量,避免超过配额
			remainingNotes := maxNotes - crawledCount
			if remainingNotes <= 0 {
				h.Logger.Info("[SearchHandler.Handle] Reached max notes count: ", maxNotes)
				break
			}
			if len(noteItems) > remainingNotes {
				noteItems = noteItems[:remainingNotes]
			}

			// 处理帖子
			processedCount, err := h.Processor.ProcessNotes(ctx, noteItems, keyword)
			if err != nil {
				h.Logger.Error("[SearchHandler.Handle] Process notes error: ", err)
			}

			// 处理评论
			if h.Config.EnableGetComments {
				noteIDs := make([]string, 0)
				xsecTokens := make([]string, 0)
				for _, item := range noteItems {
					noteIDs = append(noteIDs, item.NoteID)
					xsecTokens = append(xsecTokens, item.XsecToken)
				}
				h.Processor.ProcessComments(ctx, noteIDs, xsecTokens)
			}

			crawledCount += processedCount
			page++

			// 更新断点(每页之后保存)
			cp.CurrentPage = page
			cp.TotalCrawled = crawledCount
			if h.Config.EnableCheckpoint {
				h.Checkpoint.Save(cp)
				h.Checkpoint.SaveByID("search_latest", cp)
			}

			// 请求间隔
			if h.Config.CrawlerTimeSleep > 0 {
				time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
			}
		}
	}

	h.Logger.Info("[SearchHandler.Handle] Search completed")
	return nil
}

// ========== DetailHandler ==========

// DetailHandler 指定帖子爬取处理器
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
	h.Logger.Info("[DetailHandler.Handle] Begin get specified notes")

	// 从配置中解析笔记信息（优先使用URL列表，兼容旧的ID列表）
	noteItems := h.parseNoteList()
	if len(noteItems) == 0 {
		h.Logger.Warn("[DetailHandler.Handle] No note URLs or IDs specified")
		return nil
	}

	var cp *crawler.Checkpoint
	if h.Config.EnableCheckpoint {
		noteIDs := make([]string, 0, len(noteItems))
		for _, item := range noteItems {
			noteIDs = append(noteIDs, item.NoteID)
		}
		checkpointKey := makeXhsHashID("detail", strings.Join(noteIDs, "|"))
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

	for _, item := range noteItems {
		if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, item.NoteID) {
			h.Logger.Info("[DetailHandler.Handle] Note already crawled, skip: ", item.NoteID)
			continue
		}

		processed, err := h.Processor.ProcessNotes(ctx, []NoteItem{item}, "")
		if err != nil {
			h.Logger.Error("[DetailHandler.Handle] Process note error: ", err)
			return err
		}
		if processed == 0 {
			continue
		}

		if h.Config.EnableGetComments {
			h.Processor.ProcessComments(ctx, []string{item.NoteID}, []string{item.XsecToken})
		}

		if h.Config.EnableCheckpoint {
			h.Checkpoint.AddCrawledNote(cp, item.NoteID)
			cp.TotalCrawled = len(cp.CrawledNoteIDs)
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}

		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}

	h.Logger.Info("[DetailHandler.Handle] Detail completed")
	return nil
}

func makeXhsHashID(parts ...any) string {
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, fmt.Sprint(part))
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(values, "|"))))
}

// parseNoteList 从配置中解析笔记列表
// 优先使用 SpecifiedNoteURLs（携带xsec_token），兼容旧的 SpecifiedNoteIDs
func (h *DetailHandler) parseNoteList() []NoteItem {
	// 优先使用URL列表（与Python版本XHS_SPECIFIED_NOTE_URL_LIST对齐）
	if len(h.Config.SpecifiedNoteURLs) > 0 {
		items := make([]NoteItem, 0, len(h.Config.SpecifiedNoteURLs))
		for _, noteURL := range h.Config.SpecifiedNoteURLs {
			item := parseNoteInfoFromURL(noteURL)
			if item.NoteID != "" {
				h.Logger.Info("[DetailHandler.parseNoteList] Parsed note URL: noteID=", item.NoteID, " xsecSource=", item.XsecSource)
				items = append(items, item)
			}
		}
		return items
	}

	// 兼容旧的纯ID列表
	if len(h.Config.SpecifiedNoteIDs) > 0 {
		h.Logger.Warn("[DetailHandler.parseNoteList] Using deprecated specifiedNoteIds, please use specifiedNoteURLs with xsec_token")
		items := make([]NoteItem, 0, len(h.Config.SpecifiedNoteIDs))
		for _, noteID := range h.Config.SpecifiedNoteIDs {
			items = append(items, NoteItem{
				NoteID:     noteID,
				XsecToken:  "",
				XsecSource: "pc_note_detail",
			})
		}
		return items
	}

	return nil
}

// ========== CreatorHandler ==========

// CreatorHandler 创作者爬取处理器
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
	h.Logger.Info("[CreatorHandler.Handle] Begin get creator notes")

	// 解析创作者列表（优先使用URL列表，兼容旧的ID列表）
	creatorList := h.parseCreatorList()
	if len(creatorList) == 0 {
		h.Logger.Warn("[CreatorHandler.Handle] No creator URLs or IDs specified")
		return nil
	}

	// 加载断点,支持跨创作者恢复
	var startCreatorIndex int = 0
	var cp *crawler.Checkpoint

	if h.Config.EnableCheckpoint {
		lastCheckpoint, _ := h.Checkpoint.LoadByID("creator_latest")
		if lastCheckpoint != nil {
			// 找到上次爬取的创作者在列表中的位置
			for i, creator := range creatorList {
				if creator.CreatorID == lastCheckpoint.CreatorID {
					startCreatorIndex = i
					cp = lastCheckpoint
					h.Logger.Info("[CreatorHandler.Handle] Resume from checkpoint, creator: ", creator.CreatorID, ", cursor: ", cp.Cursor)
					break
				}
			}
		}
	}

	// 从断点位置开始爬取
	for i := startCreatorIndex; i < len(creatorList); i++ {
		creator := creatorList[i]
		h.Logger.Info("[CreatorHandler.Handle] Processing creator: ", creator.CreatorID)

		// 如果是新创作者,创建新的checkpoint并重置游标
		if cp == nil || cp.CreatorID != creator.CreatorID {
			cp = &crawler.Checkpoint{
				CrawlerType:  h.Name(),
				CreatorID:    creator.CreatorID,
				Cursor:       "",
				TotalCrawled: 0,
			}
			// 先保存checkpoint,确保后续可以恢复
			if h.Config.EnableCheckpoint {
				h.Checkpoint.SaveByID("creator_latest", cp)
			}

			// 获取创作者信息（传入xsec_token和xsec_source）
			creatorInfo, err := h.Client.GetCreatorInfo(creator.CreatorID, creator.XsecToken, creator.XsecSource)
			if err != nil {
				h.Logger.Error("[CreatorHandler.Handle] Get creator info error: ", err)
				continue
			}

			// 保存创作者信息
			h.Processor.ProcessCreator(ctx, creatorInfo)
		}

		// 获取创作者帖子
		cursor := cp.Cursor
		pageSize := 30
		crawledCount := cp.TotalCrawled
		maxNotes := h.Config.CrawlerMaxNotesCount

		for crawledCount < maxNotes {
			h.Logger.Info("[CreatorHandler.Handle] Fetching notes, cursor: ", cursor)

			notesResp, err := h.Client.GetCreatorNotes(creator.CreatorID, cursor, creator.XsecToken, creator.XsecSource, pageSize)
			if err != nil {
				h.Logger.Error("[CreatorHandler.Handle] Get creator notes error: ", err)
				// 打印当前进度,便于恢复
				h.Logger.Error("==================== Current Progress ====================")
				for j := 0; j < 3; j++ {
					h.Logger.Error("[CreatorHandler.Handle] Current creator: ", creator.CreatorID, ", cursor: ", cursor)
				}
				h.Logger.Error("==================== Current Progress ====================")
				return err
			}

			if len(notesResp.Notes) == 0 {
				h.Logger.Info("[CreatorHandler.Handle] No more notes for creator: ", creator.CreatorID)
				break
			}

			// 构建NoteItem列表
			noteItems := make([]NoteItem, 0)
			for _, note := range notesResp.Notes {
				noteItems = append(noteItems, NoteItem{
					NoteID:     note.NoteID,
					XsecToken:  note.XsecToken,
					XsecSource: "pc_user",
				})
			}

			// 限制爬取数量
			remainingNotes := maxNotes - crawledCount
			if remainingNotes <= 0 {
				h.Logger.Info("[CreatorHandler.Handle] Reached max notes count: ", maxNotes)
				break
			}
			if len(noteItems) > remainingNotes {
				noteItems = noteItems[:remainingNotes]
			}

			// 处理帖子
			processedCount, _ := h.Processor.ProcessNotes(ctx, noteItems, "")
			crawledCount += processedCount

			// 处理评论
			if h.Config.EnableGetComments {
				noteIDs := make([]string, 0)
				xsecTokens := make([]string, 0)
				for _, item := range noteItems {
					noteIDs = append(noteIDs, item.NoteID)
					xsecTokens = append(xsecTokens, item.XsecToken)
				}
				h.Processor.ProcessComments(ctx, noteIDs, xsecTokens)
			}

			// 更新游标和计数
			cursor = notesResp.Cursor
			cp.Cursor = cursor
			cp.TotalCrawled = crawledCount
			if h.Config.EnableCheckpoint {
				h.Checkpoint.SaveByID("creator_latest", cp)
			}

			if !notesResp.HasMore {
				h.Logger.Info("[CreatorHandler.Handle] No more notes (has_more=false)")
				break
			}

			// 请求间隔
			if h.Config.CrawlerTimeSleep > 0 {
				time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
			}
		}
	}

	h.Logger.Info("[CreatorHandler.Handle] Creator completed")
	return nil
}

// parseCreatorList 从配置中解析创作者列表
// 优先使用 SpecifiedCreatorURLs（携带xsec_token），兼容旧的 SpecifiedCreatorIDs
func (h *CreatorHandler) parseCreatorList() []CreatorURLInfo {
	// 优先使用URL列表（与Python版本XHS_CREATOR_URL_LIST对齐）
	if len(h.Config.SpecifiedCreatorURLs) > 0 {
		items := make([]CreatorURLInfo, 0, len(h.Config.SpecifiedCreatorURLs))
		for _, creatorURL := range h.Config.SpecifiedCreatorURLs {
			info := parseCreatorInfoFromURL(creatorURL)
			if info.CreatorID != "" {
				h.Logger.Info("[CreatorHandler.parseCreatorList] Parsed creator URL: creatorID=", info.CreatorID, " xsecSource=", info.XsecSource)
				items = append(items, info)
			}
		}
		return items
	}

	// 兼容旧的纯ID列表
	if len(h.Config.SpecifiedCreatorIDs) > 0 {
		h.Logger.Warn("[CreatorHandler.parseCreatorList] Using deprecated specifiedCreatorIds, please use specifiedCreatorURLs with xsec_token")
		items := make([]CreatorURLInfo, 0, len(h.Config.SpecifiedCreatorIDs))
		for _, creatorID := range h.Config.SpecifiedCreatorIDs {
			items = append(items, CreatorURLInfo{
				CreatorID:  creatorID,
				XsecToken:  "",
				XsecSource: "pc_user",
			})
		}
		return items
	}

	return nil
}

// ========== HomefeedHandler ==========

// HomefeedHandler 首页推荐爬取处理器
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
	h.Logger.Info("[HomefeedHandler.Handle] Begin get homefeed notes")

	// 加载断点
	cp := &crawler.Checkpoint{
		CrawlerType:  h.Name(),
		Cursor:       "",
		CurrentPage:  0, // 用于存储 noteIndex
		TotalCrawled: 0,
	}
	if h.Config.EnableCheckpoint {
		lastCheckpoint, _ := h.Checkpoint.LoadByID("homefeed_latest")
		if lastCheckpoint != nil {
			cp = lastCheckpoint
			h.Logger.Info("[HomefeedHandler.Handle] Resume from checkpoint, cursor: ", cp.Cursor, ", index: ", cp.CurrentPage)
		}
		h.Checkpoint.SaveByID("homefeed_latest", cp)
	}

	cursor := cp.Cursor
	noteIndex := cp.CurrentPage
	noteNum := 18
	crawledCount := cp.TotalCrawled
	maxNotes := h.Config.CrawlerMaxNotesCount

	for crawledCount < maxNotes {
		h.Logger.Info("[HomefeedHandler.Handle] Getting homefeed, cursor: ", cursor, ", index: ", noteIndex)

		resp, err := h.Client.GetHomefeedNotes(xhsmodel.FeedTypeRecommend, cursor, noteIndex, noteNum)
		if err != nil {
			h.Logger.Error("[HomefeedHandler.Handle] Get homefeed error: ", err)
			// 打印当前进度,便于恢复
			h.Logger.Error("==================== Current Progress ====================")
			for j := 0; j < 3; j++ {
				h.Logger.Error("[HomefeedHandler.Handle] Current cursor: ", cursor, ", note_index: ", noteIndex)
			}
			h.Logger.Error("==================== Current Progress ====================")
			return err
		}

		if len(resp.Items) == 0 || resp.CursorScore == "" {
			h.Logger.Info("[HomefeedHandler.Handle] No more data")
			break
		}

		// 构建NoteItem列表（与Python版本对齐：使用顶层id字段作为note_id）
		noteItems := make([]NoteItem, 0)
		for _, item := range resp.Items {
			// 过滤推荐和热门查询
			if item.ModelType == "rec_query" || item.ModelType == "hot_query" {
				continue
			}
			if item.ID == "" {
				continue
			}
			noteItems = append(noteItems, NoteItem{
				NoteID:     item.ID,
				XsecToken:  item.XsecToken,
				XsecSource: "pc_feed",
			})
		}

		// 限制爬取数量
		remainingNotes := maxNotes - crawledCount
		if remainingNotes <= 0 {
			h.Logger.Info("[HomefeedHandler.Handle] Reached max notes count: ", maxNotes)
			break
		}
		if len(noteItems) > remainingNotes {
			noteItems = noteItems[:remainingNotes]
		}

		// 处理帖子
		processedCount, _ := h.Processor.ProcessNotes(ctx, noteItems, "homefeed")
		crawledCount += processedCount

		// 处理评论
		if h.Config.EnableGetComments {
			noteIDs := make([]string, 0)
			xsecTokens := make([]string, 0)
			for _, item := range noteItems {
				noteIDs = append(noteIDs, item.NoteID)
				xsecTokens = append(xsecTokens, item.XsecToken)
			}
			h.Processor.ProcessComments(ctx, noteIDs, xsecTokens)
		}

		// 更新游标和索引（与Python版本对齐：note_index += note_num）
		cursor = resp.CursorScore
		noteIndex += noteNum
		cp.Cursor = cursor
		cp.CurrentPage = noteIndex
		cp.TotalCrawled = crawledCount
		if h.Config.EnableCheckpoint {
			h.Checkpoint.SaveByID("homefeed_latest", cp)
		}

		// 请求间隔
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}

	h.Logger.Info("[HomefeedHandler.Handle] Homefeed completed")
	return nil
}

// ========== HotSearchHandler ==========

// HotSearchHandler mirrors the Python XHS hot_search placeholder.
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
	h.Logger.Info("[HotSearchHandler.Handle] XiaoHongShu hot search placeholder is ready; platform-specific crawling is not implemented yet")
	return nil
}
