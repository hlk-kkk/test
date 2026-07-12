// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/kuaishou/handler.go
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

package kuaishou

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/crawler"
	kuaishoumodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/kuaishou"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	kuaishouclient "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/kuaishou"
)

type Handler interface {
	Handle(ctx context.Context) error
	Name() string
}

type HandlerDeps struct {
	Client     *kuaishouclient.Client
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
	h.Logger.Info("[SearchHandler.Handle] Begin search kuaishou")

	pageSize := 20
	maxVideos := h.Config.CrawlerMaxNotesCount
	if maxVideos < pageSize {
		maxVideos = pageSize
	}

	for _, keyword := range h.Config.Keywords {
		page := h.Config.StartPage
		if page <= 0 {
			page = 1
		}
		savedCount := (page - 1) * pageSize
		searchSessionID := ""
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
				if cp.Extra != nil {
					searchSessionID = cp.Extra["search_session_id"]
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
		seenVideos := make(map[string]struct{})

		for savedCount <= maxVideos {
			resp, err := h.Client.SearchInfoByKeyword(keyword, fmt.Sprint(page), searchSessionID)
			if err != nil {
				return err
			}
			if resp == nil || resp.Result != 1 || len(resp.Feeds) == 0 {
				break
			}
			searchSessionID = resp.SearchSessionID

			pageVideoIDs := make([]string, 0, len(resp.Feeds))
			for _, feed := range resp.Feeds {
				video := convertFeed(feed, keyword)
				if video == nil || video.VideoID == "" {
					continue
				}
				if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, video.VideoID) {
					savedCount++
					continue
				}
				if _, ok := seenVideos[video.VideoID]; ok {
					savedCount++
					continue
				}
				if exists, err := h.Store.VideoExists(ctx, video.VideoID); err != nil {
					return err
				} else if exists {
					savedCount++
					seenVideos[video.VideoID] = struct{}{}
					if h.Config.EnableCheckpoint {
						h.Checkpoint.AddCrawledNote(cp, video.VideoID)
					}
					continue
				}
				if err := h.Store.SaveVideo(ctx, video); err != nil {
					return err
				}
				seenVideos[video.VideoID] = struct{}{}
				pageVideoIDs = append(pageVideoIDs, video.VideoID)
				if h.Config.EnableCheckpoint {
					h.Checkpoint.AddCrawledNote(cp, video.VideoID)
				}
				savedCount++
			}
			if h.Config.EnableGetComments {
				for _, videoID := range pageVideoIDs {
					if err := h.fetchVideoComments(ctx, videoID); err != nil {
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
				cp.Extra["search_session_id"] = searchSessionID
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}
			if h.Config.CrawlerTimeSleep > 0 {
				time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
			}
		}
	}

	h.Logger.Info("[SearchHandler.Handle] Kuaishou search completed")
	return nil
}

func convertFeed(feed kuaishoumodel.Feed, sourceKeyword string) *Video {
	if feed.Photo.ID == "" {
		return nil
	}
	caption := truncateRunes(feed.Photo.Caption, 500)
	now := time.Now().UnixMilli()
	return &Video{
		UserID:        feed.Author.ID,
		Nickname:      feed.Author.Name,
		Avatar:        feed.Author.HeaderURL,
		LastModifyTs:  now,
		VideoID:       feed.Photo.ID,
		VideoType:     feed.Type,
		Title:         caption,
		Desc:          caption,
		CreateTime:    fmt.Sprint(feed.Photo.Timestamp),
		LikedCount:    fmt.Sprint(feed.Photo.RealLikeCount),
		ViewdCount:    fmt.Sprint(feed.Photo.ViewCount),
		VideoURL:      "https://www.kuaishou.com/short-video/" + feed.Photo.ID,
		VideoCoverURL: feed.Photo.CoverURL,
		VideoPlayURL:  feed.Photo.PhotoURL,
		SourceKeyword: sourceKeyword,
	}
}

func (h *SearchHandler) fetchVideoComments(ctx context.Context, videoID string) error {
	pcursor := ""
	savedComments := 0
	for pcursor != "no_more" {
		resp, err := h.Client.GetVideoCommentsV2(videoID, pcursor)
		if err != nil {
			return err
		}
		if resp == nil {
			return nil
		}
		pcursor = resp.PCursorV2
		if pcursor == "" {
			pcursor = "no_more"
		}
		if len(resp.RootCommentsV2) == 0 {
			return nil
		}
		for _, item := range resp.RootCommentsV2 {
			comment := convertCommentV2(videoID, item, "")
			if comment == nil {
				continue
			}
			if err := h.Store.SaveComment(ctx, comment); err != nil {
				return err
			}
			savedComments++
			if h.Config.EnableGetSubComments && item.HasSubComments {
				if err := h.fetchVideoSubComments(ctx, videoID, comment.CommentID); err != nil {
					return err
				}
			}
			if h.Config.PerNoteMaxComments > 0 && savedComments >= h.Config.PerNoteMaxComments {
				return nil
			}
		}
	}
	return nil
}

func (h *SearchHandler) fetchVideoSubComments(ctx context.Context, videoID, rootCommentID string) error {
	pcursor := ""
	for pcursor != "no_more" {
		resp, err := h.Client.GetVideoSubCommentsV2(videoID, rootCommentID, pcursor)
		if err != nil {
			return err
		}
		if resp == nil {
			return nil
		}
		pcursor = resp.PCursorV2
		if pcursor == "" {
			pcursor = "no_more"
		}
		if len(resp.SubCommentsV2) == 0 {
			return nil
		}
		for _, item := range resp.SubCommentsV2 {
			comment := convertCommentV2(videoID, item, rootCommentID)
			if comment == nil {
				continue
			}
			if err := h.Store.SaveComment(ctx, comment); err != nil {
				return err
			}
		}
	}
	return nil
}

func convertCommentV2(videoID string, item kuaishoumodel.CommentV2, parentCommentID string) *VideoComment {
	if item.CommentID == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	return &VideoComment{
		UserID:          item.AuthorID,
		Nickname:        item.AuthorName,
		Avatar:          item.HeadURL,
		LastModifyTs:    now,
		CommentID:       fmt.Sprint(item.CommentID),
		ParentCommentID: parentCommentID,
		VideoID:         videoID,
		Content:         item.Content,
		CreateTime:      fmt.Sprint(item.Timestamp),
		SubCommentCount: fmt.Sprint(item.CommentCount),
		LikeCount:       fmt.Sprint(item.LikeCount),
	}
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
	h.Logger.Info("[DetailHandler.Handle] Begin get kuaishou specified videos")

	videoIDs := h.detailVideoIDs()
	if len(videoIDs) == 0 {
		h.Logger.Warn("[DetailHandler.Handle] No video IDs specified")
		return nil
	}

	var cp *crawler.Checkpoint
	if h.Config.EnableCheckpoint {
		checkpointKey := makeKuaishouHashID("detail", strings.Join(videoIDs, "|"))
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

	for _, videoID := range videoIDs {
		if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, videoID) {
			continue
		}
		if exists, err := h.Store.VideoExists(ctx, videoID); err != nil {
			return err
		} else if exists {
			if h.Config.EnableCheckpoint {
				h.Checkpoint.AddCrawledNote(cp, videoID)
				cp.TotalCrawled = len(cp.CrawledNoteIDs)
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}
			continue
		}
		feed, err := h.Client.GetVideoInfo(videoID)
		if err != nil {
			return err
		}
		if feed == nil || feed.Photo.ID == "" {
			continue
		}
		video := convertFeed(*feed, "")
		if err := h.Store.SaveVideo(ctx, video); err != nil {
			return err
		}
		if h.Config.EnableGetComments {
			searchHandler := &SearchHandler{HandlerDeps: h.HandlerDeps}
			if err := searchHandler.fetchVideoComments(ctx, video.VideoID); err != nil {
				return err
			}
		}
		if h.Config.EnableCheckpoint {
			h.Checkpoint.AddCrawledNote(cp, video.VideoID)
			cp.TotalCrawled = len(cp.CrawledNoteIDs)
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}
	}

	h.Logger.Info("[DetailHandler.Handle] Kuaishou detail completed")
	return nil
}

func (h *DetailHandler) detailVideoIDs() []string {
	rawItems := h.Config.SpecifiedNoteIDs
	if len(rawItems) == 0 {
		rawItems = h.Config.SpecifiedNoteURLs
	}
	ids := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		if videoID := extractKuaishouVideoID(raw); videoID != "" {
			ids = append(ids, videoID)
		}
	}
	return ids
}

func extractKuaishouVideoID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err == nil {
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		for i, part := range parts {
			if part == "short-video" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}
	if before, _, ok := strings.Cut(raw, "?"); ok {
		raw = before
	}
	raw = strings.Trim(raw, "/")
	if idx := strings.LastIndex(raw, "/short-video/"); idx >= 0 {
		return strings.Trim(raw[idx+len("/short-video/"):], "/")
	}
	return raw
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
	h.Logger.Info("[CreatorHandler.Handle] Begin get kuaishou creators")

	creatorIDs := h.creatorIDs()
	if len(creatorIDs) == 0 {
		h.Logger.Warn("[CreatorHandler.Handle] No creator IDs specified")
		return nil
	}

	maxVideos := h.Config.CrawlerMaxNotesCount
	if maxVideos <= 0 {
		maxVideos = 20
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
				if index := kuaishouIndexString(creatorIDs, cp.CreatorID); index >= 0 {
					startCreatorIndex = index
				}
			}
			h.Logger.Info("[CreatorHandler.Handle] Resume from checkpoint, creator: ", cp.CreatorID, ", cursor: ", cp.Cursor)
		} else {
			cp = &crawler.Checkpoint{
				CrawlerType:  h.Name(),
				Cursor:       "",
				TotalCrawled: 0,
			}
		}
	}

	for creatorIndex := startCreatorIndex; creatorIndex < len(creatorIDs); creatorIndex++ {
		userID := creatorIDs[creatorIndex]
		if h.Config.EnableCheckpoint {
			if cp.CreatorID != userID {
				cp.Cursor = ""
				cp.TotalCrawled = 0
				cp.CrawledNoteIDs = nil
			}
			cp.CreatorID = userID
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}

		profile, err := h.Client.GetCreatorInfo(userID)
		if err != nil {
			return err
		}
		if profile != nil {
			if err := h.Store.SaveCreator(ctx, convertCreatorProfile(userID, *profile)); err != nil {
				return err
			}
		}

		pcursor := ""
		savedCount := 0
		if h.Config.EnableCheckpoint && cp.CreatorID == userID {
			pcursor = cp.Cursor
			savedCount = cp.TotalCrawled
		}
		for savedCount <= maxVideos {
			resp, err := h.Client.GetVideoByCreator(userID, pcursor)
			if err != nil {
				return err
			}
			if resp == nil || len(resp.Feeds) == 0 {
				break
			}

			videoIDs := make([]string, 0, len(resp.Feeds))
			for _, feed := range resp.Feeds {
				video := convertFeed(feed, "")
				if video == nil || video.VideoID == "" {
					continue
				}
				if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, video.VideoID) {
					continue
				}
				if err := h.Store.SaveVideo(ctx, video); err != nil {
					return err
				}
				videoIDs = append(videoIDs, video.VideoID)
				savedCount++
				if h.Config.EnableCheckpoint {
					h.Checkpoint.AddCrawledNote(cp, video.VideoID)
					cp.Cursor = pcursor
					cp.TotalCrawled = savedCount
					if err := h.Checkpoint.Save(cp); err != nil {
						return err
					}
				}
			}
			if h.Config.EnableGetComments {
				searchHandler := &SearchHandler{HandlerDeps: h.HandlerDeps}
				for _, videoID := range videoIDs {
					if err := searchHandler.fetchVideoComments(ctx, videoID); err != nil {
						return err
					}
				}
			}

			nextCursor := resp.PCursor
			if h.Config.EnableCheckpoint {
				cp.Cursor = nextCursor
				cp.TotalCrawled = savedCount
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}

			pcursor = nextCursor
			if pcursor == "" || pcursor == "no_more" {
				break
			}
		}
	}

	h.Logger.Info("[CreatorHandler.Handle] Kuaishou creator completed")
	return nil
}

func (h *CreatorHandler) creatorIDs() []string {
	rawItems := h.Config.SpecifiedCreatorIDs
	if len(rawItems) == 0 {
		rawItems = h.Config.SpecifiedCreatorURLs
	}
	ids := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		if creatorID := extractKuaishouCreatorID(raw); creatorID != "" {
			ids = append(ids, creatorID)
		}
	}
	return ids
}

func kuaishouIndexString(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
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
	h.Logger.Info("[HomefeedHandler.Handle] Begin get kuaishou homefeed videos")

	maxVideos := h.Config.CrawlerMaxNotesCount
	if maxVideos <= 0 {
		maxVideos = 20
	}
	pcursor := ""
	savedCount := 0
	searchHandler := &SearchHandler{HandlerDeps: h.HandlerDeps}
	var cp *crawler.Checkpoint
	if h.Config.EnableCheckpoint {
		var err error
		cp, err = h.Checkpoint.Load(h.Name(), "")
		if err != nil {
			return err
		}
		if cp != nil {
			pcursor = cp.Cursor
			savedCount = cp.TotalCrawled
			h.Logger.Info("[HomefeedHandler.Handle] Resume from checkpoint, cursor: ", pcursor)
		} else {
			cp = &crawler.Checkpoint{
				CrawlerType:  h.Name(),
				Cursor:       "",
				TotalCrawled: 0,
			}
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}
	}

	for savedCount < maxVideos {
		resp, err := h.Client.GetHomefeedVideos(pcursor)
		if err != nil {
			return err
		}
		if resp == nil || len(resp.Feeds) == 0 {
			break
		}

		videoIDs := make([]string, 0, len(resp.Feeds))
		partialPage := false
		for index, feed := range resp.Feeds {
			if savedCount >= maxVideos {
				partialPage = index < len(resp.Feeds)
				break
			}
			video := convertFeed(feed, "")
			if video == nil || video.VideoID == "" {
				continue
			}
			if h.Config.EnableCheckpoint && h.Checkpoint.IsNoteCrawled(cp, video.VideoID) {
				continue
			}
			if exists, err := h.Store.VideoExists(ctx, video.VideoID); err != nil {
				return err
			} else if exists {
				savedCount++
				if h.Config.EnableCheckpoint {
					h.Checkpoint.AddCrawledNote(cp, video.VideoID)
					cp.Cursor = pcursor
					cp.TotalCrawled = savedCount
					if err := h.Checkpoint.Save(cp); err != nil {
						return err
					}
				}
				continue
			}
			if err := h.Store.SaveVideo(ctx, video); err != nil {
				return err
			}
			videoIDs = append(videoIDs, video.VideoID)
			savedCount++
			if h.Config.EnableCheckpoint {
				h.Checkpoint.AddCrawledNote(cp, video.VideoID)
				cp.Cursor = pcursor
				cp.TotalCrawled = savedCount
				if err := h.Checkpoint.Save(cp); err != nil {
					return err
				}
			}
		}

		if h.Config.EnableGetComments {
			for _, videoID := range videoIDs {
				if err := searchHandler.fetchVideoComments(ctx, videoID); err != nil {
					return err
				}
			}
		}

		nextCursor := resp.PCursor
		checkpointCursor := nextCursor
		if partialPage {
			checkpointCursor = pcursor
		}
		if h.Config.EnableCheckpoint {
			cp.Cursor = checkpointCursor
			cp.TotalCrawled = savedCount
			if err := h.Checkpoint.Save(cp); err != nil {
				return err
			}
		}

		pcursor = nextCursor
		if partialPage {
			h.Logger.Warn("[HomefeedHandler.Handle] Stop at current homefeed cursor due to partial processing")
			break
		}
		if pcursor == "" || pcursor == "no_more" {
			break
		}
		if h.Config.CrawlerTimeSleep > 0 {
			time.Sleep(time.Duration(h.Config.CrawlerTimeSleep) * time.Second)
		}
	}

	h.Logger.Info("[HomefeedHandler.Handle] Kuaishou homefeed completed")
	return nil
}

type HotSearchHandler struct {
	*HandlerDeps
}

const (
	kuaishouHotSearchCheckpointID          = "hot_search_latest"
	kuaishouHotSearchSnapshotItemsExtraKey = "hot_search_snapshot_items"
	kuaishouHotSearchSnapshotTsExtraKey    = "hot_search_snapshot_ts"
)

func NewHotSearchHandler(deps *HandlerDeps) *HotSearchHandler {
	return &HotSearchHandler{HandlerDeps: deps}
}

func (h *HotSearchHandler) Name() string {
	return "hot_search"
}

func (h *HotSearchHandler) Handle(ctx context.Context) error {
	h.Logger.Info("[HotSearchHandler.Handle] Begin get kuaishou hot search")

	maxItems := h.Config.HotSearchMaxItemsCount
	if maxItems <= 0 {
		maxItems = 50
	}

	feeds, snapshotTs, startIndex, cp, err := h.loadOrCreateHotSearchSnapshot(maxItems)
	if err != nil {
		return err
	}
	if len(feeds) == 0 {
		h.Logger.Info("[HotSearchHandler.Handle] No hot search data")
		if h.Config.EnableCheckpoint {
			_ = h.Checkpoint.Clear(kuaishouHotSearchCheckpointID)
		}
		return nil
	}

	searchHandler := &SearchHandler{HandlerDeps: h.HandlerDeps}
	savedCount := 0
	if cp != nil {
		savedCount = cp.TotalCrawled
	}
	for index, feed := range feeds {
		if index < startIndex {
			continue
		}
		if savedCount >= maxItems {
			break
		}
		videoID := feed.Photo.ID
		if videoID == "" {
			if err := h.advanceHotSearchCheckpoint(cp, index+1, savedCount); err != nil {
				return err
			}
			continue
		}

		hotItem := buildKuaishouHotSearchItem(feed, index+1, snapshotTs)
		if err := h.Store.SaveHotSearchItem(ctx, hotItem); err != nil {
			return err
		}

		detail, err := h.Client.GetVideoInfo(videoID)
		if err != nil {
			return err
		}
		if detail == nil || detail.Photo.ID == "" {
			continue
		}

		video := convertFeed(*detail, "kuaishou_hot_search")
		if err := h.Store.SaveVideo(ctx, video); err != nil {
			return err
		}
		relation := buildKuaishouHotSearchRelation(hotItem, videoID)
		if err := h.Store.SaveHotSearchRelation(ctx, relation); err != nil {
			return err
		}
		if h.Config.EnableGetComments {
			if err := searchHandler.fetchVideoComments(ctx, video.VideoID); err != nil {
				return err
			}
		}
		savedCount++
		if err := h.advanceHotSearchCheckpoint(cp, index+1, savedCount); err != nil {
			return err
		}
	}

	if h.Config.EnableCheckpoint {
		_ = h.Checkpoint.Clear(kuaishouHotSearchCheckpointID)
	}
	h.Logger.Info("[HotSearchHandler.Handle] Saved ", savedCount, " kuaishou videos from hot search")
	return nil
}

func (h *HotSearchHandler) loadOrCreateHotSearchSnapshot(maxItems int) ([]kuaishoumodel.Feed, int64, int, *crawler.Checkpoint, error) {
	if h.Config.EnableCheckpoint {
		cp, err := h.Checkpoint.LoadByID(kuaishouHotSearchCheckpointID)
		if err != nil {
			return nil, 0, 0, nil, err
		}
		if cp != nil && cp.CrawlerType == h.Name() {
			feeds, snapshotTs, ok := kuaishouHotSearchSnapshotFromCheckpoint(cp, h.Config.HotSearchCheckpointMaxAgeSeconds)
			if ok {
				if len(feeds) > maxItems {
					feeds = feeds[:maxItems]
				}
				startIndex := cp.CurrentPage
				if startIndex < 0 {
					startIndex = 0
				}
				if startIndex > len(feeds) {
					startIndex = len(feeds)
				}
				h.Logger.Info("[HotSearchHandler.Handle] Resume hot search snapshot, feed index: ", startIndex)
				return feeds, snapshotTs, startIndex, cp, nil
			}
		}
	}

	resp, err := h.Client.GetHotSearchItems("")
	if err != nil {
		return nil, 0, 0, nil, err
	}
	var feeds []kuaishoumodel.Feed
	if resp != nil {
		feeds = resp.Feeds
	}
	if len(feeds) > maxItems {
		feeds = feeds[:maxItems]
	}
	snapshotTs := time.Now().UnixMilli()
	if !h.Config.EnableCheckpoint || len(feeds) == 0 {
		return feeds, snapshotTs, 0, nil, nil
	}

	body, err := json.Marshal(feeds)
	if err != nil {
		return nil, 0, 0, nil, err
	}
	cp := &crawler.Checkpoint{
		CrawlerType:  h.Name(),
		CurrentPage:  0,
		TotalCrawled: 0,
		Extra: map[string]string{
			kuaishouHotSearchSnapshotItemsExtraKey: string(body),
			kuaishouHotSearchSnapshotTsExtraKey:    fmt.Sprint(snapshotTs),
		},
	}
	if err := h.Checkpoint.SaveByID(kuaishouHotSearchCheckpointID, cp); err != nil {
		return nil, 0, 0, nil, err
	}
	return feeds, snapshotTs, 0, cp, nil
}

func kuaishouHotSearchSnapshotFromCheckpoint(cp *crawler.Checkpoint, maxAgeSeconds int) ([]kuaishoumodel.Feed, int64, bool) {
	if cp == nil || cp.Extra == nil {
		return nil, 0, false
	}
	rawItems := cp.Extra[kuaishouHotSearchSnapshotItemsExtraKey]
	if rawItems == "" {
		return nil, 0, false
	}
	var feeds []kuaishoumodel.Feed
	if err := json.Unmarshal([]byte(rawItems), &feeds); err != nil {
		return nil, 0, false
	}
	var snapshotTs int64
	_, _ = fmt.Sscan(cp.Extra[kuaishouHotSearchSnapshotTsExtraKey], &snapshotTs)
	if snapshotTs == 0 {
		return nil, 0, false
	}
	if maxAgeSeconds > 0 && kuaishouHotSearchSnapshotAgeMillis(snapshotTs) > int64(maxAgeSeconds)*1000 {
		return nil, 0, false
	}
	return feeds, snapshotTs, true
}

func kuaishouHotSearchSnapshotAgeMillis(snapshotTs int64) int64 {
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
	return h.Checkpoint.SaveByID(kuaishouHotSearchCheckpointID, cp)
}

func buildKuaishouHotSearchItem(feed kuaishoumodel.Feed, rankNo int, snapshotTs int64) *HotSearchItem {
	title := feed.Photo.Caption
	if title == "" {
		title = "kuaishou_hot_search"
	}
	rawJSON := ""
	if body, err := json.Marshal(feed); err == nil {
		rawJSON = string(body)
	}
	snapshotDate := time.UnixMilli(snapshotTs).Local().Format("2006-01-02")
	hotValue := fmt.Sprint(feed.Photo.RealLikeCount)
	if feed.Photo.RealLikeCount == 0 {
		hotValue = fmt.Sprint(feed.Photo.ViewCount)
	}
	hotItemID := makeKuaishouHashID("ks", "hot_feed", snapshotDate, snapshotTs, rankNo, title)
	return &HotSearchItem{
		HotItemID:    hotItemID,
		Platform:     "ks",
		BoardName:    "hot_feed",
		BoardType:    "video",
		SnapshotDate: snapshotDate,
		SnapshotTs:   snapshotTs,
		RankNo:       rankNo,
		Title:        title,
		Keyword:      "kuaishou_hot_search",
		HotValue:     hotValue,
		Badge:        "",
		TopicID:      feed.Photo.ID,
		TopicURL:     "https://www.kuaishou.com/short-video/" + feed.Photo.ID,
		RawJSON:      rawJSON,
	}
}

func buildKuaishouHotSearchRelation(item *HotSearchItem, videoID string) *HotSearchContentRelation {
	return &HotSearchContentRelation{
		RelID:         makeKuaishouHashID(item.HotItemID, item.Platform, "kuaishou_video", videoID, "video"),
		HotItemID:     item.HotItemID,
		Platform:      item.Platform,
		ContentID:     videoID,
		ContentType:   "video",
		ContentTable:  "kuaishou_video",
		RankNo:        item.RankNo,
		SourceKeyword: "kuaishou_hot_search",
	}
}

func makeKuaishouHashID(parts ...any) string {
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, fmt.Sprint(part))
	}
	sum := md5.Sum([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}

func extractKuaishouCreatorID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err == nil {
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		for i, part := range parts {
			if part == "profile" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}
	if before, _, ok := strings.Cut(raw, "?"); ok {
		raw = before
	}
	raw = strings.Trim(raw, "/")
	if idx := strings.LastIndex(raw, "/profile/"); idx >= 0 {
		return strings.Trim(raw[idx+len("/profile/"):], "/")
	}
	return raw
}

func convertCreatorProfile(userID string, profile kuaishoumodel.CreatorProfile) *Creator {
	gender := "男"
	if profile.Profile.Gender == "F" {
		gender = "女"
	}
	return &Creator{
		UserID:       userID,
		Nickname:     profile.Profile.UserName,
		Avatar:       profile.Profile.HeadURL,
		IPLocation:   "",
		LastModifyTs: time.Now().UnixMilli(),
		Desc:         profile.Profile.UserText,
		Gender:       gender,
		Follows:      fmt.Sprint(profile.OwnerCount.Follow),
		Fans:         fmt.Sprint(profile.OwnerCount.Fan),
		VideosCount:  fmt.Sprint(profile.OwnerCount.PhotoPublic),
	}
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
