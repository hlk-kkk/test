// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/weibo/handler.go
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

package weibo

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/crawler"
	weibomodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/weibo"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	weiboclient "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/weibo"
)

type Handler interface {
	Handle(ctx context.Context) error
	Name() string
}

type HandlerDeps struct {
	Client     *weiboclient.Client
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
	h.Logger.Info("[SearchHandler.Handle] Begin search weibo")

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
		seenNotes := make(map[string]struct{})

		for savedCount <= maxNotes {
			cards, err := h.Client.SearchNotes(keyword, page)
			if err != nil {
				return err
			}
			notes := extractNotes(cards, keyword)
			if len(notes) == 0 {
				break
			}

			remaining := maxNotes - savedCount
			if remaining <= 0 {
				break
			}
			if len(notes) > remaining {
				notes = notes[:remaining]
			}

			pageNoteIDs := make([]string, 0, len(notes))
			for _, note := range notes {
				if note == nil || note.NoteID == "" {
					continue
				}
				if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, note.NoteID) {
					savedCount++
					continue
				}
				if _, ok := seenNotes[note.NoteID]; ok {
					savedCount++
					continue
				}
				if exists, err := h.Store.NoteExists(ctx, note.NoteID); err != nil {
					return err
				} else if exists {
					savedCount++
					seenNotes[note.NoteID] = struct{}{}
					continue
				}
				if err := h.Store.SaveNote(ctx, note); err != nil {
					return err
				}
				pageNoteIDs = append(pageNoteIDs, note.NoteID)
				seenNotes[note.NoteID] = struct{}{}
				if h.Config.EnableCheckpoint {
					h.Checkpoint.AddCrawledNote(cp, note.NoteID)
				}
				savedCount++
			}
			if h.Config.EnableGetComments {
				for _, noteID := range pageNoteIDs {
					if err := h.fetchNoteComments(ctx, noteID); err != nil {
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
			if h.Config.CrawlerTimeSleep > 0 {
				time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
			}
		}
	}

	h.Logger.Info("[SearchHandler.Handle] Weibo search completed")
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
	h.Logger.Info("[DetailHandler.Handle] Begin get weibo specified notes")

	noteIDs := h.detailNoteIDs()
	if len(noteIDs) == 0 {
		h.Logger.Warn("[DetailHandler.Handle] No note IDs specified")
		return nil
	}

	var cp *crawler.Checkpoint
	if h.Config.EnableCheckpoint {
		checkpointKey := makeWeiboHashID("detail", strings.Join(noteIDs, "|"))
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
		mblog, err := h.Client.GetNoteInfoByID(noteID)
		if err != nil {
			return err
		}
		if mblog == nil {
			continue
		}
		note := convertMblog(*mblog, "")
		if note == nil {
			continue
		}
		if err := h.Store.SaveNote(ctx, note); err != nil {
			return err
		}
		if h.Config.EnableGetComments {
			if err := h.fetchNoteComments(ctx, note.NoteID); err != nil {
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

	h.Logger.Info("[DetailHandler.Handle] Weibo detail completed")
	return nil
}

func (h *DetailHandler) detailNoteIDs() []string {
	ids := make([]string, 0, len(h.Config.SpecifiedNoteIDs)+len(h.Config.SpecifiedNoteURLs))
	seen := make(map[string]struct{})
	appendID := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		value = strings.TrimRight(value, "/")
		if idx := strings.LastIndex(value, "/detail/"); idx >= 0 {
			value = value[idx+len("/detail/"):]
		}
		if idx := strings.Index(value, "?"); idx >= 0 {
			value = value[:idx]
		}
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
	h.Logger.Info("[CreatorHandler.Handle] Begin get weibo creators")

	creatorIDs := h.creatorIDs()
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
				if index := weiboIndexString(creatorIDs, cp.CreatorID); index >= 0 {
					startCreatorIndex = index
				}
			}
			h.Logger.Info("[CreatorHandler.Handle] Resume from checkpoint, creator: ", cp.CreatorID, ", since_id: ", cp.Cursor)
		} else {
			cp = &crawler.Checkpoint{
				CrawlerType:  h.Name(),
				Cursor:       "0",
				TotalCrawled: 0,
			}
		}
	}

	for creatorIndex := startCreatorIndex; creatorIndex < len(creatorIDs); creatorIndex++ {
		creatorID := creatorIDs[creatorIndex]
		if h.Config.EnableCheckpoint {
			if cp.CreatorID != creatorID {
				cp.Cursor = "0"
				cp.TotalCrawled = 0
				cp.CrawledNoteIDs = nil
			}
			cp.CreatorID = creatorID
			if cp.Cursor == "" {
				cp.Cursor = "0"
			}
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}

		creatorData, err := h.Client.GetCreatorInfoByID(creatorID)
		if err != nil {
			return err
		}
		if creatorData == nil {
			continue
		}
		creator := convertCreator(creatorData.UserInfo)
		if creator != nil {
			if err := h.Store.SaveCreator(ctx, creator); err != nil {
				return err
			}
		}

		containerID := "107603" + creatorID
		for _, tab := range creatorData.TabsInfo.Tabs {
			if tab.TabKey == "weibo" && tab.ContainerID != "" {
				containerID = tab.ContainerID
				break
			}
		}
		if err := h.fetchCreatorNotes(ctx, creatorID, containerID, cp); err != nil {
			return err
		}
	}

	h.Logger.Info("[CreatorHandler.Handle] Weibo creator completed")
	return nil
}

func (h *CreatorHandler) creatorIDs() []string {
	ids := make([]string, 0, len(h.Config.SpecifiedCreatorIDs)+len(h.Config.SpecifiedCreatorURLs))
	seen := make(map[string]struct{})
	appendID := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		value = strings.TrimRight(value, "/")
		if idx := strings.LastIndex(value, "/u/"); idx >= 0 {
			value = value[idx+len("/u/"):]
		}
		if idx := strings.LastIndex(value, "/profile/"); idx >= 0 {
			value = value[idx+len("/profile/"):]
		}
		if idx := strings.Index(value, "?"); idx >= 0 {
			value = value[:idx]
		}
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}
	for _, creatorID := range h.Config.SpecifiedCreatorIDs {
		appendID(creatorID)
	}
	for _, creatorURL := range h.Config.SpecifiedCreatorURLs {
		appendID(creatorURL)
	}
	return ids
}

func weiboIndexString(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

func (h *CreatorHandler) fetchCreatorNotes(ctx context.Context, creatorID, containerID string, cp *crawler.Checkpoint) error {
	sinceID := "0"
	totalCount := 0
	maxNotes := h.Config.CrawlerMaxNotesCount
	if maxNotes <= 0 {
		maxNotes = 40
	}
	if h.Config.EnableCheckpoint && cp != nil && cp.CreatorID == creatorID {
		if cp.Cursor != "" {
			sinceID = cp.Cursor
		}
		totalCount = cp.TotalCrawled
	}

	for totalCount <= maxNotes {
		data, err := h.Client.GetNotesByCreator(creatorID, containerID, sinceID)
		if err != nil {
			return err
		}
		if data == nil || len(data.Cards) == 0 {
			return nil
		}

		notes := extractDirectNotes(data.Cards, "")
		noteIDs := make([]string, 0, len(notes))
		for _, note := range notes {
			if note == nil || note.NoteID == "" {
				continue
			}
			if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, note.NoteID) {
				continue
			}
			if exists, err := h.Store.NoteExists(ctx, note.NoteID); err != nil {
				return err
			} else if exists {
				if h.Config.EnableCheckpoint {
					h.Checkpoint.AddCrawledNote(cp, note.NoteID)
					cp.Cursor = sinceID
					cp.TotalCrawled = totalCount
					if err := h.Checkpoint.Save(cp); err != nil {
						return err
					}
				}
				continue
			}
			if err := h.Store.SaveNote(ctx, note); err != nil {
				return err
			}
			noteIDs = append(noteIDs, note.NoteID)
			if h.Config.EnableCheckpoint {
				h.Checkpoint.AddCrawledNote(cp, note.NoteID)
				cp.Cursor = sinceID
				cp.TotalCrawled = totalCount
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}
		}
		if h.Config.EnableGetComments {
			for _, noteID := range noteIDs {
				if err := h.fetchNoteComments(ctx, noteID); err != nil {
					return err
				}
			}
		}

		totalCount += 10
		sinceID = anyToString(data.CardlistInfo.SinceID)
		if h.Config.EnableCheckpoint {
			cp.Cursor = sinceID
			cp.TotalCrawled = totalCount
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}
		if data.CardlistInfo.Total <= totalCount {
			return nil
		}
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}
	return nil
}

func convertCreator(user weibomodel.UserInfo) *Creator {
	userID := anyToString(user.ID)
	if userID == "" {
		return nil
	}
	gender := "男"
	if user.Gender == "f" {
		gender = "女"
	}
	now := time.Now().UnixMilli()
	return &Creator{
		UserID:       userID,
		Nickname:     user.ScreenName,
		Avatar:       user.AvatarHD,
		IPLocation:   strings.TrimPrefix(user.Source, "来自"),
		LastModifyTs: now,
		Desc:         user.Description,
		Gender:       gender,
		Follows:      anyToString(user.FollowCount),
		Fans:         anyToString(user.FollowersCount),
		TagList:      "",
	}
}

type HotSearchHandler struct {
	*HandlerDeps
}

const (
	weiboHotSearchCheckpointID          = "hot_search_latest"
	weiboHotSearchSnapshotItemsExtraKey = "hot_search_snapshot_items"
	weiboHotSearchSnapshotTsExtraKey    = "hot_search_snapshot_ts"
)

func NewHotSearchHandler(deps *HandlerDeps) *HotSearchHandler {
	return &HotSearchHandler{HandlerDeps: deps}
}

func (h *HotSearchHandler) Name() string {
	return "hot_search"
}

func (h *HotSearchHandler) Handle(ctx context.Context) error {
	h.Logger.Info("[HotSearchHandler.Handle] Begin get weibo hot search")

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
		h.Logger.Info("[HotSearchHandler.Handle] No hot search topics")
		if h.Config.EnableCheckpoint {
			_ = h.Checkpoint.Clear(weiboHotSearchCheckpointID)
		}
		return nil
	}

	processedItems := 0
	savedNotes := 0
	if cp != nil {
		savedNotes = cp.TotalCrawled
	}
	for index, item := range items {
		if index < startIndex {
			continue
		}
		if processedItems >= maxItems {
			break
		}
		hotWord := strings.TrimSpace(item.Word)
		if hotWord == "" || item.RealPos == 0 || item.IsAd {
			if err := h.advanceHotSearchCheckpoint(cp, index+1, savedNotes); err != nil {
				return err
			}
			continue
		}

		hotItem := buildWeiboHotSearchItem(item, snapshotTs)
		if hotItem != nil {
			if err := h.Store.SaveHotSearchItem(ctx, hotItem); err != nil {
				return err
			}
		}

		cards, err := h.Client.SearchNotes(hotWord, 1)
		if err != nil {
			return err
		}
		notes := extractNotes(cards, hotWord)
		if len(notes) > perItemCount {
			notes = notes[:perItemCount]
		}

		noteIDs := make([]string, 0, len(notes))
		for _, note := range notes {
			if note == nil || note.NoteID == "" {
				continue
			}
			if err := h.Store.SaveNote(ctx, note); err != nil {
				return err
			}
			if hotItem != nil {
				relation := buildWeiboHotSearchRelation(hotItem, note.NoteID, hotWord)
				if err := h.Store.SaveHotSearchRelation(ctx, relation); err != nil {
					return err
				}
			}
			noteIDs = append(noteIDs, note.NoteID)
			savedNotes++
		}

		if err := h.advanceHotSearchCheckpoint(cp, index+1, savedNotes); err != nil {
			return err
		}
		if h.Config.EnableGetComments {
			for _, noteID := range noteIDs {
				if err := h.fetchNoteComments(ctx, noteID); err != nil {
					return err
				}
			}
		}
		processedItems++
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}

	if h.Config.EnableCheckpoint {
		_ = h.Checkpoint.Clear(weiboHotSearchCheckpointID)
	}
	h.Logger.Info("[HotSearchHandler.Handle] Saved ", savedNotes, " weibo notes from hot search topics")
	return nil
}

func (h *HotSearchHandler) loadOrCreateHotSearchSnapshot(maxItems int) ([]weibomodel.HotBandItem, int64, int, *crawler.Checkpoint, error) {
	if h.Config.EnableCheckpoint {
		cp, err := h.Checkpoint.LoadByID(weiboHotSearchCheckpointID)
		if err != nil {
			return nil, 0, 0, nil, err
		}
		if cp != nil && cp.CrawlerType == h.Name() {
			items, snapshotTs, ok := weiboHotSearchSnapshotFromCheckpoint(cp, h.Config.HotSearchCheckpointMaxAgeSeconds)
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
				h.Logger.Info("[HotSearchHandler.Handle] Resume hot search snapshot, topic index: ", startIndex)
				return items, snapshotTs, startIndex, cp, nil
			}
		}
	}

	items, err := h.Client.GetHotSearchItems()
	if err != nil {
		return nil, 0, 0, nil, err
	}
	if len(items) > maxItems {
		items = items[:maxItems]
	}
	snapshotTs := time.Now().UnixMilli()
	if !h.Config.EnableCheckpoint || len(items) == 0 {
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
			weiboHotSearchSnapshotItemsExtraKey: string(body),
			weiboHotSearchSnapshotTsExtraKey:    fmt.Sprint(snapshotTs),
		},
	}
	if err := h.Checkpoint.SaveByID(weiboHotSearchCheckpointID, cp); err != nil {
		return nil, 0, 0, nil, err
	}
	return items, snapshotTs, 0, cp, nil
}

func weiboHotSearchSnapshotFromCheckpoint(cp *crawler.Checkpoint, maxAgeSeconds int) ([]weibomodel.HotBandItem, int64, bool) {
	if cp == nil || cp.Extra == nil {
		return nil, 0, false
	}
	rawItems := cp.Extra[weiboHotSearchSnapshotItemsExtraKey]
	if rawItems == "" {
		return nil, 0, false
	}
	var items []weibomodel.HotBandItem
	if err := json.Unmarshal([]byte(rawItems), &items); err != nil {
		return nil, 0, false
	}
	var snapshotTs int64
	_, _ = fmt.Sscan(cp.Extra[weiboHotSearchSnapshotTsExtraKey], &snapshotTs)
	if snapshotTs == 0 {
		return nil, 0, false
	}
	if maxAgeSeconds > 0 && weiboHotSearchSnapshotAgeMillis(snapshotTs) > int64(maxAgeSeconds)*1000 {
		return nil, 0, false
	}
	return items, snapshotTs, true
}

func weiboHotSearchSnapshotAgeMillis(snapshotTs int64) int64 {
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
	return h.Checkpoint.SaveByID(weiboHotSearchCheckpointID, cp)
}

func buildWeiboHotSearchItem(item weibomodel.HotBandItem, snapshotTs int64) *HotSearchItem {
	title := strings.TrimSpace(item.Word)
	if title == "" {
		return nil
	}
	wordScheme := strings.TrimSpace(item.WordScheme)
	if wordScheme == "" {
		wordScheme = title
	}
	hotValue := anyToString(item.Num)
	if hotValue == "" {
		hotValue = anyToString(item.RawHot)
	}
	badge := item.LabelName
	if badge == "" {
		badge = item.IconDesc
	}
	topicID := item.Mid
	if topicID == "" {
		topicID = wordScheme
	}
	rawJSON := ""
	if body, err := json.Marshal(item); err == nil {
		rawJSON = string(body)
	}
	snapshotDate := time.UnixMilli(snapshotTs).Local().Format("2006-01-02")
	return &HotSearchItem{
		HotItemID:    makeWeiboHashID("wb", "hot_search", snapshotDate, snapshotTs, item.RealPos, title),
		Platform:     "wb",
		BoardName:    "hot_search",
		BoardType:    "topic",
		SnapshotDate: snapshotDate,
		SnapshotTs:   snapshotTs,
		RankNo:       item.RealPos,
		Title:        title,
		Keyword:      title,
		HotValue:     hotValue,
		Badge:        badge,
		TopicID:      topicID,
		TopicURL:     "https://s.weibo.com/weibo?q=" + weiboQuote(wordScheme),
		RawJSON:      rawJSON,
	}
}

func buildWeiboHotSearchRelation(item *HotSearchItem, noteID, sourceKeyword string) *HotSearchContentRelation {
	return &HotSearchContentRelation{
		RelID:         makeWeiboHashID(item.HotItemID, item.Platform, "weibo_note", noteID, "note"),
		HotItemID:     item.HotItemID,
		Platform:      item.Platform,
		ContentID:     noteID,
		ContentType:   "note",
		ContentTable:  "weibo_note",
		RankNo:        item.RankNo,
		SourceKeyword: sourceKeyword,
	}
}

func makeWeiboHashID(parts ...any) string {
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, fmt.Sprint(part))
	}
	sum := md5.Sum([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}

func weiboQuote(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

func (h *HandlerDeps) fetchNoteComments(ctx context.Context, noteID string) error {
	maxID := int64(-1)
	maxIDType := 0
	savedComments := 0

	for {
		page, err := h.Client.GetNoteComments(noteID, maxID, maxIDType)
		if err != nil {
			return err
		}
		if page == nil || len(page.Data) == 0 {
			return nil
		}

		for _, item := range page.Data {
			comment := convertComment(noteID, item)
			if comment == nil {
				continue
			}
			if err := h.Store.SaveComment(ctx, comment); err != nil {
				return err
			}
			savedComments++
			if h.Config.EnableGetSubComments {
				for _, subItem := range item.Comments {
					subComment := convertComment(noteID, subItem)
					if subComment == nil {
						continue
					}
					if err := h.Store.SaveComment(ctx, subComment); err != nil {
						return err
					}
					savedComments++
				}
			}
			if h.Config.PerNoteMaxComments > 0 && savedComments >= h.Config.PerNoteMaxComments {
				return nil
			}
		}

		maxID = page.MaxID
		maxIDType = page.MaxIDType
		if maxID == 0 {
			return nil
		}
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}
}

func extractNotes(cards []weibomodel.Card, sourceKeyword string) []*Note {
	notes := make([]*Note, 0)
	for _, card := range cards {
		if card.CardType == 9 {
			if note := convertMblog(card.Mblog, sourceKeyword); note != nil {
				notes = append(notes, note)
			}
		}
		for _, child := range card.CardGroup {
			if child.CardType == 9 {
				if note := convertMblog(child.Mblog, sourceKeyword); note != nil {
					notes = append(notes, note)
				}
			}
		}
	}
	return notes
}

func extractDirectNotes(cards []weibomodel.Card, sourceKeyword string) []*Note {
	notes := make([]*Note, 0)
	for _, card := range cards {
		if card.CardType != 9 {
			continue
		}
		if note := convertMblog(card.Mblog, sourceKeyword); note != nil {
			notes = append(notes, note)
		}
	}
	return notes
}

func convertMblog(mblog weibomodel.Mblog, sourceKeyword string) *Note {
	noteID := anyToString(mblog.ID)
	if noteID == "" {
		return nil
	}
	createTime, createDateTime := parseWeiboTime(mblog.CreatedAt)
	now := time.Now().UnixMilli()
	return &Note{
		UserID:         anyToString(mblog.User.ID),
		Nickname:       mblog.User.ScreenName,
		Avatar:         mblog.User.ProfileImageURL,
		Gender:         mblog.User.Gender,
		ProfileURL:     mblog.User.ProfileURL,
		IPLocation:     strings.TrimPrefix(mblog.RegionName, "发布于 "),
		LastModifyTs:   now,
		NoteID:         noteID,
		Content:        stripHTML(mblog.Text),
		CreateTime:     createTime,
		CreateDateTime: createDateTime,
		LikedCount:     anyToString(mblog.AttitudesCount),
		CommentsCount:  anyToString(mblog.CommentsCount),
		SharedCount:    anyToString(mblog.RepostsCount),
		NoteURL:        "https://m.weibo.cn/detail/" + noteID,
		ImageList:      strings.Join(imageList(mblog.Pics), ","),
		VideoURL:       strings.Join(videoURLList(mblog.PageInfo), ","),
		SourceKeyword:  sourceKeyword,
	}
}

func convertComment(noteID string, item weibomodel.CommentItem) *Comment {
	commentID := anyToString(item.ID)
	if commentID == "" {
		return nil
	}
	createTime, createDateTime := parseWeiboTime(item.CreatedAt)
	now := time.Now().UnixMilli()
	return &Comment{
		UserID:          anyToString(item.User.ID),
		Nickname:        item.User.ScreenName,
		Avatar:          item.User.ProfileImageURL,
		Gender:          item.User.Gender,
		ProfileURL:      item.User.ProfileURL,
		IPLocation:      strings.TrimPrefix(item.Source, "来自"),
		LastModifyTs:    now,
		CommentID:       commentID,
		NoteID:          noteID,
		Content:         stripHTML(item.Text),
		CreateTime:      createTime,
		CreateDateTime:  createDateTime,
		SubCommentCount: anyToString(item.TotalNumber),
		ParentCommentID: anyToString(item.RootID),
		LikeCount:       anyToString(item.LikeCount),
	}
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
	default:
		return fmt.Sprint(typed)
	}
}

var htmlTagPattern = regexp.MustCompile(`<.*?>`)

func stripHTML(value string) string {
	return htmlTagPattern.ReplaceAllString(value, "")
}

func imageList(pics []weibomodel.Pic) []string {
	images := make([]string, 0, len(pics))
	for _, pic := range pics {
		if pic.URL != "" {
			images = append(images, pic.URL)
		}
	}
	return images
}

func videoURLList(pageInfo weibomodel.PageInfo) []string {
	if pageInfo.Type != "video" {
		return nil
	}
	keys := []string{"mp4_720p_mp4", "mp4_hd_mp4", "mp4_ld_mp4"}
	urls := make([]string, 0, len(keys))
	for _, key := range keys {
		if value := pageInfo.URLs[key]; value != "" {
			urls = append(urls, value)
		}
	}
	return urls
}

func parseWeiboTime(value string) (int64, string) {
	const layout = "Mon Jan 02 15:04:05 -0700 2006"
	parsed, err := time.Parse(layout, value)
	if err != nil {
		return 0, ""
	}
	china := parsed.In(time.FixedZone("", 8*60*60))
	pythonTimestamp := time.Date(
		parsed.Year(),
		parsed.Month(),
		parsed.Day(),
		parsed.Hour(),
		parsed.Minute(),
		parsed.Second(),
		0,
		time.UTC,
	).Unix()
	return pythonTimestamp, china.Format("2006-01-02 15:04:05-07:00")
}
