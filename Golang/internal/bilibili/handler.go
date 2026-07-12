// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/bilibili/handler.go
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

package bilibili

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/crawler"
	bilimodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/bilibili"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	biliclient "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/bilibili"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/worker"
)

// Handler 定义爬取处理器接口
type Handler interface {
	Handle(ctx context.Context) error
	Name() string
}

// HandlerDeps 所有Handler共享的依赖
type HandlerDeps struct {
	Client     *biliclient.Client
	Processor  *Processor
	Checkpoint *crawler.CheckpointManager
	WorkerPool *worker.Pool
	Config     *config.Config
	Logger     *tools.Logger
}

func NewHandlerDeps(
	client *biliclient.Client,
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

// VideoItem 视频项（用于批量处理）
type VideoItem struct {
	Aid  int64
	Bvid string
}

// ========== SearchHandler ==========

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
	h.Logger.Info("[SearchHandler.Handle] Begin search bilibili")

	keywords := h.Config.Keywords
	maxVideos := h.Config.CrawlerMaxNotesCount
	pageSize := 20

	for _, keyword := range keywords {
		h.Logger.Info("[SearchHandler.Handle] Searching keyword: ", keyword)
		seenVideoItems := make(map[string]struct{})

		// 尝试加载断点
		var startPage int = 1
		var crawledCount int = 0
		var cp *crawler.Checkpoint
		if h.Config.EnableCheckpoint {
			cp, _ = h.Checkpoint.Load(h.Name(), keyword)
			if cp != nil {
				startPage = cp.CurrentPage
				crawledCount = cp.TotalCrawled
				h.Logger.Info("[SearchHandler.Handle] Resume from checkpoint, page: ", startPage)
			} else {
				cp = &crawler.Checkpoint{
					CrawlerType: h.Name(),
					Keyword:     keyword,
					CurrentPage: 1,
				}
			}
		}

		page := startPage
		for crawledCount < maxVideos {
			h.Logger.Info("[SearchHandler.Handle] Searching page: ", page)

			searchData, err := h.Client.SearchVideoByKeyword(keyword, page, pageSize, "")
			if err != nil {
				h.Logger.Error("[SearchHandler.Handle] Search error: ", err)
				break
			}

			if len(searchData.Result) == 0 {
				h.Logger.Info("[SearchHandler.Handle] No more data")
				break
			}

			// 构建VideoItem列表
			videoItems := make([]VideoItem, 0)
			for _, item := range searchData.Result {
				if item.Type != "video" {
					continue
				}
				videoItems = append(videoItems, VideoItem{
					Aid:  item.Aid,
					Bvid: item.Bvid,
				})
			}
			videoItems = filterSeenVideoItems(videoItems, seenVideoItems)
			videoItems = limitVideoItemsToRemaining(videoItems, maxVideos, crawledCount)

			// 处理视频
			processedCount, err := h.Processor.ProcessVideos(ctx, videoItems, keyword)
			if err != nil {
				h.Logger.Error("[SearchHandler.Handle] Process videos error: ", err)
			}

			// 处理评论
			if h.Config.EnableGetComments {
				aids := make([]int64, 0)
				for _, item := range videoItems {
					aids = append(aids, item.Aid)
				}
				h.Processor.ProcessComments(ctx, aids)
			}

			crawledCount += processedCount
			page++

			// 更新断点
			if h.Config.EnableCheckpoint {
				cp.CurrentPage = page
				cp.TotalCrawled = crawledCount
				h.Checkpoint.Save(cp)
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
	h.Logger.Info("[DetailHandler.Handle] Begin get specified videos")

	// 从配置读取视频ID列表 (Bvid)
	videoIDs := h.Config.SpecifiedNoteIDs // 复用XHS的配置字段
	if len(videoIDs) == 0 {
		h.Logger.Warn("[DetailHandler.Handle] No video IDs specified")
		return nil
	}

	var cp *crawler.Checkpoint
	if h.Config.EnableCheckpoint {
		checkpointKey := makeBilibiliHashID("detail", strings.Join(videoIDs, "|"))
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

	for _, bvid := range videoIDs {
		if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, bvid) {
			continue
		}

		processed, err := h.Processor.ProcessVideos(ctx, []VideoItem{{Bvid: bvid}}, "")
		if err != nil {
			h.Logger.Error("[DetailHandler.Handle] Process video error: ", err)
			return err
		}
		if processed == 0 {
			continue
		}

		if h.Config.EnableGetComments {
			info, err := h.Client.GetVideoInfo(bvid, 0)
			if err != nil {
				return err
			}
			if info != nil {
				h.Processor.ProcessComments(ctx, []int64{info.Aid})
			}
		}

		if h.Config.EnableCheckpoint {
			h.Checkpoint.AddCrawledNote(cp, bvid)
			cp.TotalCrawled = len(cp.CrawledNoteIDs)
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}
	}

	h.Logger.Info("[DetailHandler.Handle] Detail completed")
	return nil
}

func makeBilibiliHashID(parts ...any) string {
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, fmt.Sprint(part))
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(values, "|"))))
}

// ========== CreatorHandler ==========

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
	h.Logger.Info("[CreatorHandler.Handle] Begin get creator videos")

	// 从配置读取UP主ID列表
	creatorIDs := h.Config.SpecifiedCreatorIDs
	if len(creatorIDs) == 0 {
		h.Logger.Warn("[CreatorHandler.Handle] No creator IDs specified")
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
				if index := indexString(creatorIDs, cp.CreatorID); index >= 0 {
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

	for creatorIndex := startCreatorIndex; creatorIndex < len(creatorIDs); creatorIndex++ {
		creatorIDStr := creatorIDs[creatorIndex]
		seenVideoItems := make(map[string]struct{})
		mid, err := strconv.ParseInt(creatorIDStr, 10, 64)
		if err != nil {
			h.Logger.Error("[CreatorHandler.Handle] Invalid creator ID: ", creatorIDStr)
			continue
		}

		h.Logger.Info("[CreatorHandler.Handle] Processing creator: ", mid)
		if h.Config.EnableCheckpoint {
			if cp.CreatorID != creatorIDStr {
				cp.CurrentPage = 1
				cp.TotalCrawled = 0
				cp.CrawledNoteIDs = nil
			}
			cp.CreatorID = creatorIDStr
			if cp.CurrentPage <= 0 {
				cp.CurrentPage = 1
			}
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}

		// 获取UP主信息
		upInfo, err := h.Client.GetUpInfo(mid)
		if err != nil {
			h.Logger.Error("[CreatorHandler.Handle] Get up info error: ", err)
			continue
		}

		// 获取UP主统计
		upStat, _ := h.Client.GetUpStat(mid)
		navNum, _ := h.Client.GetSpaceNavNum(mid)

		// 保存UP主信息
		h.Processor.ProcessUpInfo(ctx, upInfo, upStat, navNum)

		// 获取UP主视频
		page := 1
		pageSize := 30
		crawledCount := 0
		maxVideos := h.Config.CrawlerMaxNotesCount
		if h.Config.EnableCheckpoint && cp.CreatorID == creatorIDStr {
			if cp.CurrentPage > 0 {
				page = cp.CurrentPage
			}
			crawledCount = cp.TotalCrawled
		}

		for crawledCount < maxVideos {
			videosData, err := h.Client.GetUpVideos(mid, page, pageSize)
			if err != nil {
				h.Logger.Error("[CreatorHandler.Handle] Get up videos error: ", err)
				break
			}

			if len(videosData.List.Vlist) == 0 {
				break
			}

			// 构建VideoItem列表
			videoItems := make([]VideoItem, 0)
			for _, v := range videosData.List.Vlist {
				videoItems = append(videoItems, VideoItem{
					Aid:  v.Aid,
					Bvid: v.Bvid,
				})
			}

			processedCount := 0
			partialPage := false
			if h.Config.EnableCheckpoint {
				processedCount, partialPage, err = h.processCreatorVideosWithCheckpoint(ctx, cp, page, crawledCount, maxVideos, videoItems)
				if err != nil {
					return err
				}
			} else {
				// 处理视频
				videoItems = filterSeenVideoItems(videoItems, seenVideoItems)
				videoItems = limitVideoItemsToRemaining(videoItems, maxVideos, crawledCount)
				processedCount, _ = h.Processor.ProcessVideos(ctx, videoItems, "")

				// 处理评论
				if h.Config.EnableGetComments {
					aids := make([]int64, 0)
					for _, item := range videoItems {
						aids = append(aids, item.Aid)
					}
					h.Processor.ProcessComments(ctx, aids)
				}
			}
			crawledCount += processedCount

			nextPage := page + 1
			checkpointPage := nextPage
			if partialPage {
				checkpointPage = page
			}
			if h.Config.EnableCheckpoint {
				cp.CurrentPage = checkpointPage
				cp.TotalCrawled = crawledCount
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}

			page = nextPage
			if partialPage {
				h.Logger.Warn("[CreatorHandler.Handle] Stop at current creator page due to partial processing")
				break
			}
			if crawledCount >= videosData.Page.Count {
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

func (h *CreatorHandler) processCreatorVideosWithCheckpoint(ctx context.Context, cp *crawler.Checkpoint, page, crawledCount, maxVideos int, videoItems []VideoItem) (int, bool, error) {
	processedCount := 0
	partialPage := false
	for index, item := range videoItems {
		if crawledCount+processedCount >= maxVideos {
			partialPage = index < len(videoItems)
			break
		}

		noteID := bilibiliVideoCheckpointNoteID(item)
		if h.Checkpoint.IsNoteCrawled(cp, noteID) {
			continue
		}

		if err := h.Processor.processOneVideo(ctx, item, ""); err != nil {
			h.Logger.Error("[CreatorHandler.Handle] Process video error: ", err)
			partialPage = true
			break
		}

		processedCount++
		h.Checkpoint.AddCrawledNote(cp, noteID)
		cp.CurrentPage = page
		cp.TotalCrawled = crawledCount + processedCount
		if err := h.Checkpoint.Save(cp); err != nil {
			return processedCount, partialPage, err
		}

		if h.Config.EnableGetComments && item.Aid > 0 {
			h.Processor.ProcessComments(ctx, []int64{item.Aid})
		}
	}
	return processedCount, partialPage, nil
}

func indexString(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

// ========== HomefeedHandler ==========

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
	h.Logger.Info("[HomefeedHandler.Handle] Begin get homefeed videos")

	freshIdx := 1
	crawledCount := 0
	maxVideos := h.Config.CrawlerMaxNotesCount
	seenVideoItems := make(map[string]struct{})
	var cp *crawler.Checkpoint
	if h.Config.EnableCheckpoint {
		var err error
		cp, err = h.Checkpoint.Load(h.Name(), "")
		if err != nil {
			return err
		}
		if cp != nil {
			if cp.CurrentPage > 0 {
				freshIdx = cp.CurrentPage
			}
			crawledCount = cp.TotalCrawled
			h.Logger.Info("[HomefeedHandler.Handle] Resume from checkpoint, fresh idx: ", freshIdx)
		} else {
			cp = &crawler.Checkpoint{
				CrawlerType:  h.Name(),
				CurrentPage:  1,
				TotalCrawled: 0,
			}
		}
	}

	for crawledCount < maxVideos {
		h.Logger.Info("[HomefeedHandler.Handle] Getting homefeed, idx: ", freshIdx)

		feedData, err := h.Client.GetHomefeedVideos(freshIdx)
		if err != nil {
			h.Logger.Error("[HomefeedHandler.Handle] Get homefeed error: ", err)
			break
		}

		if len(feedData.Item) == 0 {
			h.Logger.Info("[HomefeedHandler.Handle] No more data")
			break
		}

		// 构建VideoItem列表
		videoItems := make([]VideoItem, 0)
		for _, item := range feedData.Item {
			if item.Bvid == "" {
				continue
			}
			if item.Goto != "av" || item.ShowInfo != 1 {
				continue
			}
			videoItems = append(videoItems, VideoItem{
				Aid:  item.Id,
				Bvid: item.Bvid,
			})
		}

		processedCount := 0
		partialPage := false
		if h.Config.EnableCheckpoint {
			processedCount, partialPage, err = h.processHomefeedVideosWithCheckpoint(ctx, cp, freshIdx, crawledCount, maxVideos, videoItems)
			if err != nil {
				return err
			}
		} else {
			// 处理视频
			videoItems = filterSeenVideoItems(videoItems, seenVideoItems)
			videoItems = limitVideoItemsToRemaining(videoItems, maxVideos, crawledCount)
			processedCount, _ = h.Processor.ProcessVideos(ctx, videoItems, "")

			// 处理评论
			if h.Config.EnableGetComments {
				aids := make([]int64, 0)
				for _, item := range videoItems {
					aids = append(aids, item.Aid)
				}
				h.Processor.ProcessComments(ctx, aids)
			}
		}
		crawledCount += processedCount

		nextFreshIdx := freshIdx + 1
		checkpointFreshIdx := nextFreshIdx
		if partialPage {
			checkpointFreshIdx = freshIdx
		}
		if h.Config.EnableCheckpoint {
			cp.CurrentPage = checkpointFreshIdx
			cp.TotalCrawled = crawledCount
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}

		freshIdx = nextFreshIdx
		if partialPage {
			h.Logger.Warn("[HomefeedHandler.Handle] Stop at current homefeed page due to partial processing")
			break
		}

		// 请求间隔
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}

	h.Logger.Info("[HomefeedHandler.Handle] Homefeed completed")
	return nil
}

func (h *HomefeedHandler) processHomefeedVideosWithCheckpoint(ctx context.Context, cp *crawler.Checkpoint, freshIdx, crawledCount, maxVideos int, videoItems []VideoItem) (int, bool, error) {
	processedCount := 0
	partialPage := false
	for index, item := range videoItems {
		if crawledCount+processedCount >= maxVideos {
			partialPage = index < len(videoItems)
			break
		}

		noteID := bilibiliVideoCheckpointNoteID(item)
		if h.Checkpoint.IsNoteCrawled(cp, noteID) {
			continue
		}

		if err := h.Processor.processOneVideo(ctx, item, ""); err != nil {
			h.Logger.Error("[HomefeedHandler.Handle] Process video error: ", err)
			partialPage = true
			break
		}

		processedCount++
		h.Checkpoint.AddCrawledNote(cp, noteID)
		cp.CurrentPage = freshIdx
		cp.TotalCrawled = crawledCount + processedCount
		if err := h.Checkpoint.Save(cp); err != nil {
			return processedCount, partialPage, err
		}

		if h.Config.EnableGetComments && item.Aid > 0 {
			h.Processor.ProcessComments(ctx, []int64{item.Aid})
		}
	}
	return processedCount, partialPage, nil
}

func bilibiliVideoCheckpointNoteID(item VideoItem) string {
	if item.Bvid != "" {
		return item.Bvid
	}
	return strconv.FormatInt(item.Aid, 10)
}

func limitVideoItemsToRemaining(videoItems []VideoItem, maxVideos, crawledCount int) []VideoItem {
	remaining := maxVideos - crawledCount
	if remaining <= 0 {
		return nil
	}
	if len(videoItems) > remaining {
		return videoItems[:remaining]
	}
	return videoItems
}

func filterSeenVideoItems(videoItems []VideoItem, seen map[string]struct{}) []VideoItem {
	if len(videoItems) == 0 {
		return nil
	}
	filtered := make([]VideoItem, 0, len(videoItems))
	for _, item := range videoItems {
		noteID := bilibiliVideoCheckpointNoteID(item)
		if noteID == "" || noteID == "0" {
			continue
		}
		if _, ok := seen[noteID]; ok {
			continue
		}
		seen[noteID] = struct{}{}
		filtered = append(filtered, item)
	}
	return filtered
}

// ========== HotSearchHandler ==========

type HotSearchHandler struct {
	*HandlerDeps
}

const (
	bilibiliHotSearchCheckpointID          = "hot_search_latest"
	bilibiliHotSearchSnapshotItemsExtraKey = "hot_search_snapshot_items"
	bilibiliHotSearchSnapshotTsExtraKey    = "hot_search_snapshot_ts"
)

func NewHotSearchHandler(deps *HandlerDeps) *HotSearchHandler {
	return &HotSearchHandler{HandlerDeps: deps}
}

func (h *HotSearchHandler) Name() string {
	return "hot_search"
}

func (h *HotSearchHandler) Handle(ctx context.Context) error {
	h.Logger.Info("[HotSearchHandler.Handle] Begin get bilibili hot search")

	maxItems := h.Config.HotSearchMaxItemsCount
	if maxItems <= 0 {
		maxItems = 50
	}

	items, startIndex, cp, err := h.loadOrCreateHotSearchSnapshot(maxItems)
	if err != nil {
		h.Logger.Error("[HotSearchHandler.Handle] Get hot search error: ", err)
		return err
	}
	if len(items) == 0 {
		h.Logger.Info("[HotSearchHandler.Handle] No hot search data")
		if h.Config.EnableCheckpoint {
			_ = h.Checkpoint.Clear(bilibiliHotSearchCheckpointID)
		}
		return nil
	}

	processedCount := 0
	if cp != nil {
		processedCount = cp.TotalCrawled
	}
	seenVideoItems := make(map[string]struct{})
	for index, item := range items {
		if index < startIndex {
			continue
		}
		if processedCount >= maxItems {
			break
		}
		if item.Bvid == "" && item.Aid == 0 {
			if err := h.advanceHotSearchCheckpoint(cp, index+1, processedCount); err != nil {
				return err
			}
			continue
		}
		videoItem := VideoItem{
			Aid:  item.Aid,
			Bvid: item.Bvid,
		}
		if len(filterSeenVideoItems([]VideoItem{videoItem}, seenVideoItems)) == 0 {
			if err := h.advanceHotSearchCheckpoint(cp, index+1, processedCount); err != nil {
				return err
			}
			continue
		}

		if err := h.Processor.processOneVideo(ctx, videoItem, "bilibili_hot_search"); err != nil {
			h.Logger.Error("[HotSearchHandler.Handle] Process video error: ", err)
			return err
		}
		processedCount++
		if err := h.advanceHotSearchCheckpoint(cp, index+1, processedCount); err != nil {
			return err
		}
		if h.Config.EnableGetComments && item.Aid > 0 {
			h.Processor.ProcessComments(ctx, []int64{item.Aid})
		}
	}

	if h.Config.EnableCheckpoint {
		_ = h.Checkpoint.Clear(bilibiliHotSearchCheckpointID)
	}
	h.Logger.Info("[HotSearchHandler.Handle] Saved ", processedCount, " bilibili videos from hot search")
	return nil
}

func (h *HotSearchHandler) loadOrCreateHotSearchSnapshot(maxItems int) ([]bilimodel.VideoInfo, int, *crawler.Checkpoint, error) {
	if h.Config.EnableCheckpoint {
		cp, err := h.Checkpoint.LoadByID(bilibiliHotSearchCheckpointID)
		if err != nil {
			return nil, 0, nil, err
		}
		if cp != nil && cp.CrawlerType == h.Name() {
			items, ok := bilibiliHotSearchSnapshotFromCheckpoint(cp, h.Config.HotSearchCheckpointMaxAgeSeconds)
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
				h.Logger.Info("[HotSearchHandler.Handle] Resume hot search snapshot, video index: ", startIndex)
				return items, startIndex, cp, nil
			}
		}
	}

	hotSearchData, err := h.Client.GetHotSearchItems(1, maxItems)
	if err != nil {
		return nil, 0, nil, err
	}
	var items []bilimodel.VideoInfo
	if hotSearchData != nil {
		items = hotSearchData.List
	}
	if len(items) > maxItems {
		items = items[:maxItems]
	}
	if !h.Config.EnableCheckpoint || len(items) == 0 {
		return items, 0, nil, nil
	}

	snapshotTs := time.Now().UnixMilli()
	body, err := json.Marshal(items)
	if err != nil {
		return nil, 0, nil, err
	}
	cp := &crawler.Checkpoint{
		CrawlerType:  h.Name(),
		CurrentPage:  0,
		TotalCrawled: 0,
		Extra: map[string]string{
			bilibiliHotSearchSnapshotItemsExtraKey: string(body),
			bilibiliHotSearchSnapshotTsExtraKey:    fmt.Sprint(snapshotTs),
		},
	}
	if err := h.Checkpoint.SaveByID(bilibiliHotSearchCheckpointID, cp); err != nil {
		return nil, 0, nil, err
	}
	return items, 0, cp, nil
}

func bilibiliHotSearchSnapshotFromCheckpoint(cp *crawler.Checkpoint, maxAgeSeconds int) ([]bilimodel.VideoInfo, bool) {
	if cp == nil || cp.Extra == nil {
		return nil, false
	}
	rawItems := cp.Extra[bilibiliHotSearchSnapshotItemsExtraKey]
	if rawItems == "" {
		return nil, false
	}
	var items []bilimodel.VideoInfo
	if err := json.Unmarshal([]byte(rawItems), &items); err != nil {
		return nil, false
	}
	var snapshotTs int64
	_, _ = fmt.Sscan(cp.Extra[bilibiliHotSearchSnapshotTsExtraKey], &snapshotTs)
	if snapshotTs == 0 {
		return nil, false
	}
	if maxAgeSeconds > 0 && bilibiliHotSearchSnapshotAgeMillis(snapshotTs) > int64(maxAgeSeconds)*1000 {
		return nil, false
	}
	return items, true
}

func bilibiliHotSearchSnapshotAgeMillis(snapshotTs int64) int64 {
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
	return h.Checkpoint.SaveByID(bilibiliHotSearchCheckpointID, cp)
}
