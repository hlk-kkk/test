// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/douyin/handler.go
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

package douyin

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/crawler"
	douyinmodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/douyin"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	douyinclient "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/douyin"
)

type Handler interface {
	Handle(ctx context.Context) error
	Name() string
}

type HandlerDeps struct {
	Client     *douyinclient.Client
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
	h.Logger.Info("[SearchHandler.Handle] Begin search douyin")

	pageSize := 20
	maxAwemes := h.Config.CrawlerMaxNotesCount
	if maxAwemes < pageSize {
		maxAwemes = pageSize
	}

	for _, keyword := range h.Config.Keywords {
		page := h.Config.StartPage
		if page <= 0 {
			page = 1
		}
		savedCount := 0
		searchID := ""
		var cp *crawler.Checkpoint
		if h.Config.EnableCheckpoint {
			cp, _ = h.Checkpoint.Load(h.Name(), keyword)
			if cp != nil {
				if cp.CurrentPage > 0 {
					page = cp.CurrentPage
				}
				savedCount = cp.TotalCrawled
				if cp.Extra != nil {
					searchID = cp.Extra["search_id"]
				}
				h.Logger.Info("[SearchHandler.Handle] Resume from checkpoint, keyword: ", keyword, ", page: ", page)
			} else {
				cp = &crawler.Checkpoint{
					CrawlerType:  h.Name(),
					Keyword:      keyword,
					CurrentPage:  page,
					TotalCrawled: savedCount,
					Extra:        map[string]string{},
				}
			}
		}
		for savedCount <= maxAwemes {
			offset := (page - 1) * pageSize
			searchResp, err := h.Client.SearchInfoByKeyword(keyword, offset, searchID)
			if err != nil {
				return err
			}
			searchID = searchResp.Extra.LogID
			if len(searchResp.Data) == 0 {
				break
			}

			pageAwemeIDs := make([]string, 0, len(searchResp.Data))
			for _, item := range searchResp.Data {
				if item.AwemeInfo == nil || item.AwemeInfo.AwemeID == "" {
					continue
				}
				if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, item.AwemeInfo.AwemeID) {
					savedCount++
					continue
				}
				aweme := convertAwemeInfo(item.AwemeInfo, keyword)
				if err := h.Store.SaveAweme(ctx, aweme); err != nil {
					return err
				}
				if h.Config.EnableCheckpoint {
					h.Checkpoint.AddCrawledNote(cp, aweme.AwemeID)
				}
				pageAwemeIDs = append(pageAwemeIDs, aweme.AwemeID)
				savedCount++
			}
			if h.Config.EnableGetComments {
				for _, awemeID := range pageAwemeIDs {
					if err := h.fetchAwemeComments(ctx, awemeID); err != nil {
						return err
					}
				}
			}

			page++
			if h.Config.EnableCheckpoint {
				cp.CurrentPage = page
				cp.TotalCrawled = savedCount
				if cp.Extra == nil {
					cp.Extra = map[string]string{}
				}
				cp.Extra["search_id"] = searchID
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}
			if h.Config.CrawlerTimeSleep > 0 {
				time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
			}
		}
	}

	h.Logger.Info("[SearchHandler.Handle] Douyin search completed")
	return nil
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
	h.Logger.Info("[DetailHandler.Handle] Begin get douyin specified awemes")

	awemeIDs := h.detailAwemeIDs()
	if len(awemeIDs) == 0 {
		h.Logger.Warn("[DetailHandler.Handle] No aweme IDs specified")
		return nil
	}

	var cp *crawler.Checkpoint
	if h.Config.EnableCheckpoint {
		checkpointKey := makeDouyinHashID("detail", strings.Join(awemeIDs, "|"))
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

	for _, awemeID := range awemeIDs {
		if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, awemeID) {
			continue
		}
		info, err := h.Client.GetVideoByID(awemeID)
		if err != nil {
			return err
		}
		if info == nil || info.AwemeID == "" {
			continue
		}
		aweme := convertAwemeInfo(info, "")
		if err := h.Store.SaveAweme(ctx, aweme); err != nil {
			return err
		}
		if h.Config.EnableGetComments {
			searchHandler := &SearchHandler{HandlerDeps: h.HandlerDeps}
			if err := searchHandler.fetchAwemeComments(ctx, aweme.AwemeID); err != nil {
				return err
			}
		}
		if h.Config.EnableCheckpoint {
			h.Checkpoint.AddCrawledNote(cp, aweme.AwemeID)
			cp.TotalCrawled = len(cp.CrawledNoteIDs)
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}
	}

	h.Logger.Info("[DetailHandler.Handle] Douyin detail completed")
	return nil
}

func (h *DetailHandler) detailAwemeIDs() []string {
	rawItems := h.Config.SpecifiedNoteURLs
	if len(rawItems) == 0 {
		rawItems = h.Config.SpecifiedNoteIDs
	}
	ids := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		if awemeID := extractDouyinAwemeID(raw); awemeID != "" {
			ids = append(ids, awemeID)
		}
	}
	return ids
}

func extractDouyinAwemeID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err == nil {
		if modalID := parsed.Query().Get("modal_id"); modalID != "" {
			return modalID
		}
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		for i, part := range parts {
			if part == "video" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}
	if before, _, ok := strings.Cut(raw, "?"); ok {
		raw = before
	}
	raw = strings.Trim(raw, "/")
	if idx := strings.LastIndex(raw, "/video/"); idx >= 0 {
		return strings.Trim(raw[idx+len("/video/"):], "/")
	}
	return raw
}

func makeDouyinHashID(parts ...any) string {
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, fmt.Sprint(part))
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(values, "|"))))
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
	h.Logger.Info("[CreatorHandler.Handle] Begin get douyin creators")

	creatorIDs := h.creatorIDs()
	if len(creatorIDs) == 0 {
		h.Logger.Warn("[CreatorHandler.Handle] No creator IDs specified")
		return nil
	}

	maxAwemes := h.Config.CrawlerMaxNotesCount
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
				if index := douyinIndexString(creatorIDs, cp.CreatorID); index >= 0 {
					startCreatorIndex = index
				}
			}
			h.Logger.Info("[CreatorHandler.Handle] Resume from checkpoint, creator: ", cp.CreatorID, ", cursor: ", cp.Cursor)
		} else {
			cp = &crawler.Checkpoint{
				CrawlerType:  h.Name(),
				Cursor:       "0",
				TotalCrawled: 0,
			}
		}
	}

	for creatorIndex := startCreatorIndex; creatorIndex < len(creatorIDs); creatorIndex++ {
		secUserID := creatorIDs[creatorIndex]
		if h.Config.EnableCheckpoint {
			if cp.CreatorID != secUserID {
				cp.Cursor = "0"
				cp.TotalCrawled = 0
				cp.CrawledNoteIDs = nil
			}
			cp.CreatorID = secUserID
			if cp.Cursor == "" {
				cp.Cursor = "0"
			}
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}

		creatorInfo, err := h.Client.GetUserInfo(secUserID)
		if err != nil {
			return err
		}
		if creatorInfo != nil {
			creator := convertCreatorInfo(creatorInfo)
			if err := h.Store.SaveCreator(ctx, creator); err != nil {
				return err
			}
		}

		maxCursor := "0"
		savedCount := 0
		if h.Config.EnableCheckpoint && cp.CreatorID == secUserID {
			if cp.Cursor != "" {
				maxCursor = cp.Cursor
			}
			savedCount = cp.TotalCrawled
		}
		for savedCount <= maxAwemes {
			postsResp, err := h.Client.GetUserAwemePosts(secUserID, maxCursor)
			if err != nil {
				return err
			}
			if postsResp == nil || len(postsResp.AwemeList) == 0 {
				break
			}
			pageAwemeIDs := make([]string, 0, len(postsResp.AwemeList))
			for _, item := range postsResp.AwemeList {
				if item.AwemeID == "" {
					continue
				}
				if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, item.AwemeID) {
					continue
				}
				aweme := convertAwemeInfo(&item, "")
				if err := h.Store.SaveAweme(ctx, aweme); err != nil {
					return err
				}
				pageAwemeIDs = append(pageAwemeIDs, aweme.AwemeID)
				savedCount++
				if h.Config.EnableCheckpoint {
					h.Checkpoint.AddCrawledNote(cp, aweme.AwemeID)
					cp.Cursor = maxCursor
					cp.TotalCrawled = savedCount
					if err := h.Checkpoint.Save(cp); err != nil {
						return err
					}
				}
			}
			if h.Config.EnableGetComments {
				searchHandler := &SearchHandler{HandlerDeps: h.HandlerDeps}
				for _, awemeID := range pageAwemeIDs {
					if err := searchHandler.fetchAwemeComments(ctx, awemeID); err != nil {
						return err
					}
				}
			}
			nextCursor := postsResp.MaxCursor
			if h.Config.EnableCheckpoint {
				cp.Cursor = nextCursor
				cp.TotalCrawled = savedCount
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}
			if postsResp.HasMore != 1 {
				break
			}
			maxCursor = nextCursor
			if maxCursor == "" {
				break
			}
			if h.Config.CrawlerTimeSleep > 0 {
				time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
			}
		}
	}

	h.Logger.Info("[CreatorHandler.Handle] Douyin creator completed")
	return nil
}

func (h *CreatorHandler) creatorIDs() []string {
	rawItems := h.Config.SpecifiedCreatorURLs
	if len(rawItems) == 0 {
		rawItems = h.Config.SpecifiedCreatorIDs
	}
	ids := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		item := strings.TrimSpace(raw)
		if item != "" {
			ids = append(ids, item)
		}
	}
	return ids
}

func douyinIndexString(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

type HotSearchHandler struct {
	*HandlerDeps
}

const (
	douyinHotSearchCheckpointID          = "hot_search_latest"
	douyinHotSearchSnapshotItemsExtraKey = "hot_search_snapshot_items"
	douyinHotSearchSnapshotTsExtraKey    = "hot_search_snapshot_ts"
)

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
	h.Logger.Info("[HomefeedHandler.Handle] Begin get douyin homefeed awemes")

	refreshIndex := 0
	perPageCount := 20
	maxAwemes := h.Config.CrawlerMaxNotesCount
	savedCount := 0
	var cp *crawler.Checkpoint
	if h.Config.EnableCheckpoint {
		var err error
		cp, err = h.Checkpoint.Load(h.Name(), "")
		if err != nil {
			return err
		}
		if cp != nil {
			if cp.CurrentPage > 0 {
				refreshIndex = cp.CurrentPage
			}
			savedCount = cp.TotalCrawled
			h.Logger.Info("[HomefeedHandler.Handle] Resume from checkpoint, refresh index: ", refreshIndex)
		} else {
			cp = &crawler.Checkpoint{
				CrawlerType:  h.Name(),
				CurrentPage:  0,
				TotalCrawled: 0,
			}
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}
	}

	for savedCount <= maxAwemes {
		resp, err := h.Client.GetHomefeedAwemeList(refreshIndex, perPageCount)
		if err != nil {
			return err
		}
		if resp == nil || resp.StatusCode != 0 {
			break
		}
		if len(resp.Cards) == 0 {
			break
		}

		pageAwemeIDs := make([]string, 0, len(resp.Cards))
		partialPage := false
		if h.Config.EnableCheckpoint {
			var processedIDs []string
			var err error
			processedIDs, savedCount, partialPage, err = h.processHomefeedCardsWithCheckpoint(ctx, cp, refreshIndex, savedCount, maxAwemes, resp.Cards)
			if err != nil {
				return err
			}
			pageAwemeIDs = processedIDs
		} else {
			for _, card := range resp.Cards {
				if card.Type != 1 || card.Aweme == "" {
					continue
				}
				if savedCount >= maxAwemes {
					break
				}
				var info douyinmodel.AwemeInfo
				if err := json.Unmarshal([]byte(card.Aweme), &info); err != nil {
					return err
				}
				if info.AwemeID == "" {
					continue
				}
				aweme := convertAwemeInfo(&info, "")
				if err := h.Store.SaveAweme(ctx, aweme); err != nil {
					return err
				}
				pageAwemeIDs = append(pageAwemeIDs, aweme.AwemeID)
				savedCount++
			}
			if h.Config.EnableGetComments {
				searchHandler := &SearchHandler{HandlerDeps: h.HandlerDeps}
				for _, awemeID := range pageAwemeIDs {
					if err := searchHandler.fetchAwemeComments(ctx, awemeID); err != nil {
						return err
					}
				}
			}
		}

		nextRefreshIndex := refreshIndex + perPageCount
		checkpointRefreshIndex := nextRefreshIndex
		if partialPage {
			checkpointRefreshIndex = refreshIndex
		}
		if h.Config.EnableCheckpoint {
			cp.CurrentPage = checkpointRefreshIndex
			cp.TotalCrawled = savedCount
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}

		refreshIndex = nextRefreshIndex
		if partialPage {
			h.Logger.Warn("[HomefeedHandler.Handle] Stop at current homefeed refresh index due to partial processing")
			break
		}
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}

	h.Logger.Info("[HomefeedHandler.Handle] Douyin homefeed completed")
	return nil
}

func (h *HomefeedHandler) processHomefeedCardsWithCheckpoint(ctx context.Context, cp *crawler.Checkpoint, refreshIndex, savedCount, maxAwemes int, cards []douyinmodel.HomefeedCard) ([]string, int, bool, error) {
	pageAwemeIDs := make([]string, 0, len(cards))
	partialPage := false
	for index, card := range cards {
		if card.Type != 1 || card.Aweme == "" {
			continue
		}
		if savedCount >= maxAwemes {
			partialPage = index < len(cards)
			break
		}

		var info douyinmodel.AwemeInfo
		if err := json.Unmarshal([]byte(card.Aweme), &info); err != nil {
			return pageAwemeIDs, savedCount, partialPage, err
		}
		if info.AwemeID == "" {
			continue
		}
		if h.Checkpoint.IsNoteCrawled(cp, info.AwemeID) {
			continue
		}

		aweme := convertAwemeInfo(&info, "")
		if err := h.Store.SaveAweme(ctx, aweme); err != nil {
			return pageAwemeIDs, savedCount, partialPage, err
		}
		h.Checkpoint.AddCrawledNote(cp, aweme.AwemeID)
		savedCount++
		cp.CurrentPage = refreshIndex
		cp.TotalCrawled = savedCount
		if err := h.Checkpoint.Save(cp); err != nil {
			return pageAwemeIDs, savedCount, partialPage, err
		}

		pageAwemeIDs = append(pageAwemeIDs, aweme.AwemeID)
		if h.Config.EnableGetComments {
			searchHandler := &SearchHandler{HandlerDeps: h.HandlerDeps}
			if err := searchHandler.fetchAwemeComments(ctx, aweme.AwemeID); err != nil {
				return pageAwemeIDs, savedCount, partialPage, err
			}
		}
	}
	return pageAwemeIDs, savedCount, partialPage, nil
}

func NewHotSearchHandler(deps *HandlerDeps) *HotSearchHandler {
	return &HotSearchHandler{HandlerDeps: deps}
}

func (h *HotSearchHandler) Name() string {
	return "hot_search"
}

func (h *HotSearchHandler) Handle(ctx context.Context) error {
	h.Logger.Info("[HotSearchHandler.Handle] Begin get douyin hot search")

	maxItems := h.Config.HotSearchMaxItemsCount
	if maxItems <= 0 {
		maxItems = 50
	}
	perItemCount := h.Config.HotSearchPerItemContentCount
	if perItemCount <= 0 {
		perItemCount = 4
	}

	hotItems, startIndex, cp, err := h.loadOrCreateHotSearchSnapshot(maxItems)
	if err != nil {
		return err
	}
	if len(hotItems) == 0 {
		h.Logger.Info("[HotSearchHandler.Handle] No hot search data")
		if h.Config.EnableCheckpoint {
			_ = h.Checkpoint.Clear(douyinHotSearchCheckpointID)
		}
		return nil
	}

	savedCount := 0
	if cp != nil {
		savedCount = cp.TotalCrawled
	}
	for index, hotItem := range hotItems {
		if index < startIndex {
			continue
		}
		if savedCount >= maxItems {
			break
		}
		awemeIDs, err := h.Client.GetHotSearchAwemeIDs(hotItem.Word, 0, perItemCount)
		if err != nil {
			return err
		}
		if len(awemeIDs) > perItemCount {
			awemeIDs = awemeIDs[:perItemCount]
		}
		for _, awemeID := range awemeIDs {
			info, err := h.Client.GetVideoByID(awemeID)
			if err != nil {
				return err
			}
			if info == nil || info.AwemeID == "" {
				continue
			}
			aweme := convertAwemeInfo(info, hotItem.Word)
			if err := h.Store.SaveAweme(ctx, aweme); err != nil {
				return err
			}
			if h.Config.EnableGetComments {
				searchHandler := &SearchHandler{HandlerDeps: h.HandlerDeps}
				if err := searchHandler.fetchAwemeComments(ctx, aweme.AwemeID); err != nil {
					return err
				}
			}
		}
		savedCount++
		if err := h.advanceHotSearchCheckpoint(cp, index+1, savedCount); err != nil {
			return err
		}
	}

	if h.Config.EnableCheckpoint {
		_ = h.Checkpoint.Clear(douyinHotSearchCheckpointID)
	}
	h.Logger.Info("[HotSearchHandler.Handle] Saved ", savedCount, " douyin hot search items")
	return nil
}

func (h *HotSearchHandler) loadOrCreateHotSearchSnapshot(maxItems int) ([]douyinmodel.HotSearchItem, int, *crawler.Checkpoint, error) {
	if h.Config.EnableCheckpoint {
		cp, err := h.Checkpoint.LoadByID(douyinHotSearchCheckpointID)
		if err != nil {
			return nil, 0, nil, err
		}
		if cp != nil && cp.CrawlerType == h.Name() {
			items, ok := douyinHotSearchSnapshotFromCheckpoint(cp, h.Config.HotSearchCheckpointMaxAgeSeconds)
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
				h.Logger.Info("[HotSearchHandler.Handle] Resume hot search snapshot, hot word index: ", startIndex)
				return items, startIndex, cp, nil
			}
		}
	}

	items, err := h.Client.GetHotSearchItems()
	if err != nil {
		return nil, 0, nil, err
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
			douyinHotSearchSnapshotItemsExtraKey: string(body),
			douyinHotSearchSnapshotTsExtraKey:    fmt.Sprint(snapshotTs),
		},
	}
	if err := h.Checkpoint.SaveByID(douyinHotSearchCheckpointID, cp); err != nil {
		return nil, 0, nil, err
	}
	return items, 0, cp, nil
}

func douyinHotSearchSnapshotFromCheckpoint(cp *crawler.Checkpoint, maxAgeSeconds int) ([]douyinmodel.HotSearchItem, bool) {
	if cp == nil || cp.Extra == nil {
		return nil, false
	}
	rawItems := cp.Extra[douyinHotSearchSnapshotItemsExtraKey]
	if rawItems == "" {
		return nil, false
	}
	var items []douyinmodel.HotSearchItem
	if err := json.Unmarshal([]byte(rawItems), &items); err != nil {
		return nil, false
	}
	var snapshotTs int64
	_, _ = fmt.Sscan(cp.Extra[douyinHotSearchSnapshotTsExtraKey], &snapshotTs)
	if snapshotTs == 0 {
		return nil, false
	}
	if maxAgeSeconds > 0 && douyinHotSearchSnapshotAgeMillis(snapshotTs) > int64(maxAgeSeconds)*1000 {
		return nil, false
	}
	return items, true
}

func douyinHotSearchSnapshotAgeMillis(snapshotTs int64) int64 {
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
	return h.Checkpoint.SaveByID(douyinHotSearchCheckpointID, cp)
}

func (h *SearchHandler) fetchAwemeComments(ctx context.Context, awemeID string) error {
	cursor := int64(0)
	totalSaved := 0
	for {
		resp, err := h.Client.GetAwemeComments(awemeID, cursor)
		if err != nil {
			return err
		}
		if len(resp.Comments) == 0 {
			if resp.HasMore == 0 {
				return nil
			}
			cursor = resp.Cursor
			continue
		}

		for _, item := range resp.Comments {
			comment := convertCommentInfo(awemeID, item)
			if err := h.Store.SaveComment(ctx, comment); err != nil {
				return err
			}
			totalSaved++
		}
		if h.Config.PerNoteMaxComments > 0 && totalSaved >= h.Config.PerNoteMaxComments {
			return nil
		}
		if h.Config.EnableGetSubComments {
			for _, item := range resp.Comments {
				if item.ReplyCommentTotal <= 0 {
					continue
				}
				saved, err := h.fetchSubComments(ctx, awemeID, item.CID)
				if err != nil {
					return err
				}
				totalSaved += saved
				if h.Config.PerNoteMaxComments > 0 && totalSaved >= h.Config.PerNoteMaxComments {
					return nil
				}
			}
		}
		if resp.HasMore == 0 {
			return nil
		}
		cursor = resp.Cursor
	}
}

func (h *SearchHandler) fetchSubComments(ctx context.Context, awemeID, commentID string) (int, error) {
	cursor := int64(0)
	totalSaved := 0
	for {
		resp, err := h.Client.GetSubComments(commentID, cursor)
		if err != nil {
			return totalSaved, err
		}
		for _, item := range resp.Comments {
			comment := convertCommentInfo(awemeID, item)
			if err := h.Store.SaveComment(ctx, comment); err != nil {
				return totalSaved, err
			}
			totalSaved++
		}
		if resp.HasMore == 0 {
			return totalSaved, nil
		}
		cursor = resp.Cursor
	}
}

func convertAwemeInfo(info *douyinmodel.AwemeInfo, sourceKeyword string) *Aweme {
	title := info.PreviewTitle
	if title == "" {
		title = info.Desc
	}
	return &Aweme{
		AwemeID:          info.AwemeID,
		AwemeType:        strconv.Itoa(info.AwemeType),
		Title:            title,
		Desc:             info.Desc,
		CreateTime:       strconv.FormatInt(info.CreateTime, 10),
		LikedCount:       strconv.FormatInt(info.Statistics.DiggCount, 10),
		CommentCount:     strconv.FormatInt(info.Statistics.CommentCount, 10),
		ShareCount:       strconv.FormatInt(info.Statistics.ShareCount, 10),
		CollectedCount:   strconv.FormatInt(info.Statistics.CollectCount, 10),
		AwemeURL:         "https://www.douyin.com/video/" + info.AwemeID,
		CoverURL:         extractCoverURL(info.Video),
		VideoDownloadURL: extractVideoDownloadURL(info.Video),
		SourceKeyword:    sourceKeyword,
		IsAIGenerated:    info.AIGCInfo.AIGCLabelType,
		UserID:           info.Author.UID,
		SecUID:           info.Author.SecUID,
		ShortUserID:      info.Author.ShortID,
		UserUniqueID:     info.Author.UniqueID,
		Nickname:         info.Author.Nickname,
		Avatar:           firstURL(info.Author.AvatarThumb.URLList),
		UserSignature:    info.Author.Signature,
		IPLocation:       info.IPLabel,
		LastModifyTs:     time.Now().UnixMilli(),
	}
}

func convertCommentInfo(awemeID string, info douyinmodel.CommentInfo) *AwemeComment {
	return &AwemeComment{
		CommentID:       info.CID,
		AwemeID:         awemeID,
		Content:         info.Text,
		CreateTime:      strconv.FormatInt(info.CreateTime, 10),
		SubCommentCount: strconv.Itoa(info.ReplyCommentTotal),
		ParentCommentID: info.ReplyID,
		ReplyToReplyID:  info.ReplyToReplyID,
		LikeCount:       strconv.FormatInt(info.DiggCount, 10),
		Pictures:        extractCommentPictures(info.ImageList),
		IPLocation:      info.IPLabel,
		UserID:          info.User.UID,
		SecUID:          info.User.SecUID,
		ShortUserID:     info.User.ShortID,
		UserUniqueID:    info.User.UniqueID,
		Nickname:        info.User.Nickname,
		Avatar:          firstURL(info.User.AvatarThumb.URLList),
		UserSignature:   info.User.Signature,
		LastModifyTs:    time.Now().UnixMilli(),
	}
}

func convertCreatorInfo(info *douyinmodel.CreatorInfo) *Creator {
	return &Creator{
		UserID:       info.UID,
		Nickname:     info.Nickname,
		Avatar:       firstURL(info.AvatarLarger.URLList),
		IPLocation:   info.IPLocation,
		Desc:         info.Signature,
		Gender:       douyinGender(info.Gender),
		Follows:      strconv.FormatInt(info.Statistics.FollowingCount, 10),
		Fans:         strconv.FormatInt(info.Statistics.FollowerCount, 10),
		Interaction:  strconv.FormatInt(info.Statistics.TotalFavorited, 10),
		VideosCount:  strconv.FormatInt(info.Statistics.AwemeCount, 10),
		LastModifyTs: time.Now().UnixMilli(),
	}
}

func douyinGender(gender int) string {
	switch gender {
	case 1:
		return "男"
	case 2:
		return "女"
	default:
		return "未知"
	}
}

func extractCommentPictures(images []douyinmodel.CommentImage) string {
	pictures := make([]string, 0, len(images))
	for _, image := range images {
		urls := image.OriginURL.URLList
		if len(urls) > 1 {
			pictures = append(pictures, urls[1])
		}
	}
	return strings.Join(pictures, ",")
}

func extractCoverURL(video douyinmodel.Video) string {
	urls := video.RawCover.URLList
	if len(urls) == 0 {
		urls = video.OriginCover.URLList
	}
	if len(urls) > 1 {
		return urls[1]
	}
	return ""
}

func extractVideoDownloadURL(video douyinmodel.Video) string {
	urls := video.PlayAddrH264.URLList
	if len(urls) == 0 {
		urls = video.PlayAddr256.URLList
	}
	if len(urls) == 0 {
		urls = video.PlayAddr.URLList
	}
	if len(urls) >= 2 {
		return urls[len(urls)-1]
	}
	return ""
}

func firstURL(urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}
