// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/xhs/processor.go
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

// internal/xhs/processor.go
package xhs

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	xhsmodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/xiaohongshu"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	xhsclient "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/xhs"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/worker"
)

// Processor 数据处理器
type Processor struct {
	client     *xhsclient.Client
	store      Store
	workerPool *worker.Pool
	config     *config.Config
	logger     *tools.Logger
}

// NewProcessor 创建处理器
func NewProcessor(client *xhsclient.Client, store Store, workerPool *worker.Pool, cfg *config.Config) *Processor {
	return &Processor{
		client:     client,
		store:      store,
		workerPool: workerPool,
		config:     cfg,
		logger:     tools.GetLogger(),
	}
}

// ProcessNotes 批量处理帖子
func (p *Processor) ProcessNotes(ctx context.Context, noteItems []NoteItem, sourceKeyword string) (int, error) {
	if len(noteItems) == 0 {
		return 0, nil
	}

	p.logger.Info("[Processor.ProcessNotes] Processing ", len(noteItems), " notes")

	var successCount int32
	tasks := make([]worker.Task, 0, len(noteItems))

	for _, item := range noteItems {
		noteItem := item // 捕获变量
		tasks = append(tasks, func(ctx context.Context) error {
			err := p.processOneNote(ctx, noteItem, sourceKeyword)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
			return err
		})
	}

	errs := p.workerPool.Submit(ctx, tasks)
	if len(errs) > 0 {
		p.logger.Error("[Processor.ProcessNotes] Failed count: ", len(errs))
	}

	return int(successCount), nil
}

// processOneNote 处理单个帖子
func (p *Processor) processOneNote(ctx context.Context, item NoteItem, sourceKeyword string) error {
	// 检查是否已存在
	exists, _ := p.store.NoteExists(ctx, item.NoteID)
	if exists {
		p.logger.Info("[Processor.processOneNote] Note already exists: ", item.NoteID)
		return nil
	}

	// 获取帖子详情（API优先，失败后再尝试HTML，与Python版本对齐）
	noteDetail, err := p.client.GetNoteDetailFromApi(item.NoteID, item.XsecToken, item.XsecSource)
	if err != nil {
		p.logger.Warn("[Processor.processOneNote] Get from API failed, try HTML: ", item.NoteID)
		noteDetail, err = p.client.GetNoteDetailFromHtml(item.NoteID, item.XsecToken, item.XsecSource)
		if err != nil {
			p.logger.Error("[Processor.processOneNote] Get note detail failed: ", item.NoteID, err)
			return err
		}
	}

	// 转换为存储模型
	note := p.convertNoteDetailToNote(noteDetail)
	note.SourceKeyword = sourceKeyword
	note.NoteURL = "https://www.xiaohongshu.com/explore/" + item.NoteID + "?xsec_token=" + item.XsecToken + "&xsec_source=" + item.XsecSource

	// 保存
	if err := p.store.SaveNote(ctx, note); err != nil {
		p.logger.Error("[Processor.processOneNote] Save note failed: ", item.NoteID, err)
		return err
	}

	p.logger.Info("[Processor.processOneNote] Saved note: ", item.NoteID)
	return nil
}

// convertNoteDetailToNote 将API响应转换为存储模型
func (p *Processor) convertNoteDetailToNote(noteDetail *xhsmodel.NoteDetail) *Note {
	now := time.Now().Unix()

	// 获取图片列表
	imgList := make([]string, 0)
	for _, img := range noteDetail.ImageList {
		imgList = append(imgList, img.URL)
	}

	// 获取标签列表
	tagList := make([]string, 0)
	for _, tag := range noteDetail.TagList {
		if tag.Type == "topic" {
			tagList = append(tagList, tag.Name)
		}
	}

	// 获取视频URL
	videoURL := ""
	if noteDetail.Type == "video" {
		originVideoKey := noteDetail.Video.Consumer.OriginVideoKey
		if originVideoKey != "" {
			videoURL = "http://sns-video-bd.xhscdn.com/" + originVideoKey
		} else if len(noteDetail.Video.Media.Stream.H264) > 0 {
			videoURL = noteDetail.Video.Media.Stream.H264[0].MasterUrl
		}
	}

	return &Note{
		NoteID:         noteDetail.NoteId,
		UserID:         noteDetail.User.UserID,
		Nickname:       noteDetail.User.Nickname,
		Avatar:         noteDetail.User.Avatar,
		IPLocation:     noteDetail.IpLocation,
		AddTs:          now,
		LastModifyTs:   now,
		Type:           noteDetail.Type,
		Title:          noteDetail.Title,
		Desc:           noteDetail.Desc,
		VideoURL:       videoURL,
		Time:           noteDetail.Time,
		LastUpdateTime: noteDetail.LastUpdateTime,
		LikedCount:     noteDetail.InteractInfo.LikedCount,
		CollectedCount: noteDetail.InteractInfo.CollectedCount,
		CommentCount:   noteDetail.InteractInfo.CommentCount,
		ShareCount:     noteDetail.InteractInfo.ShareCount,
		ImageList:      strings.Join(imgList, ","),
		TagList:        strings.Join(tagList, ","),
	}
}

// ProcessComments 批量处理帖子评论
func (p *Processor) ProcessComments(ctx context.Context, noteIDs []string, xsecTokens []string) {
	if len(noteIDs) == 0 || !p.config.EnableGetComments {
		return
	}

	p.logger.Info("[Processor.ProcessComments] Processing comments for ", len(noteIDs), " notes")

	tasks := make([]worker.Task, 0, len(noteIDs))
	for i, noteID := range noteIDs {
		nid := noteID
		token := ""
		if i < len(xsecTokens) {
			token = xsecTokens[i]
		}

		tasks = append(tasks, func(ctx context.Context) error {
			return p.processNoteComments(ctx, nid, token)
		})
	}

	p.workerPool.Submit(ctx, tasks)
}

// processNoteComments 处理单个帖子的所有评论
func (p *Processor) processNoteComments(ctx context.Context, noteID, xsecToken string) error {
	cursor := ""
	totalComments := 0
	maxComments := p.config.PerNoteMaxComments
	if maxComments <= 0 {
		maxComments = 100 // 默认值
	}

	for totalComments < maxComments {
		// 获取一级评论
		resp, err := p.client.GetNoteComments(noteID, cursor, xsecToken)
		if err != nil {
			p.logger.Error("[Processor.processNoteComments] Get comments failed: ", noteID, err)
			return err
		}

		if len(resp.Comments) == 0 {
			break
		}

		// 保存评论
		for _, comment := range resp.Comments {
			commentModel := p.convertCommentToModel(noteID, &comment, "", xsecToken)
			if err := p.store.SaveComment(ctx, commentModel); err != nil {
				p.logger.Error("[Processor.processNoteComments] Save comment failed: ", comment.ID, " noteID: ", noteID, " err: ", err)
			}
			totalComments++

			// 获取子评论
			if p.config.EnableGetSubComments && comment.SubCommentCount != "0" {
				p.processSubComments(ctx, noteID, comment.ID, xsecToken)
			}
		}

		if !resp.HasMore || cursor == resp.Cursor {
			break
		}
		cursor = resp.Cursor
	}

	p.logger.Info("[Processor.processNoteComments] Processed ", totalComments, " comments for note: ", noteID)
	return nil
}

// processSubComments 处理子评论
func (p *Processor) processSubComments(ctx context.Context, noteID, rootCommentID, xsecToken string) {
	cursor := ""
	for {
		resp, err := p.client.GetNoteSubComments(noteID, rootCommentID, cursor, xsecToken, 10)
		if err != nil {
			p.logger.Error("[Processor.processSubComments] Get sub comments failed: ", noteID, " rootCommentID: ", rootCommentID, " err: ", err)
			break
		}

		if len(resp.Comments) == 0 {
			break
		}

		for _, comment := range resp.Comments {
			commentModel := p.convertCommentToModel(noteID, &comment, rootCommentID, xsecToken)
			if err := p.store.SaveComment(ctx, commentModel); err != nil {
				p.logger.Error("[Processor.processSubComments] Save sub comment failed: ", comment.ID, " noteID: ", noteID, " err: ", err)
			}
		}

		if !resp.HasMore {
			break
		}
		cursor = resp.Cursor
	}
}

// extractPictureURLs 从评论的 Pictures json.RawMessage 中提取图片URL列表
// Python版本: ",".join([item.get("url_default", "") for item in comment_item.get("pictures", [])])
func (p *Processor) extractPictureURLs(pictures json.RawMessage) string {
	if len(pictures) == 0 || string(pictures) == "null" || string(pictures) == "" {
		return ""
	}

	var picList []struct {
		URLDefault string `json:"url_default"`
	}
	if err := json.Unmarshal(pictures, &picList); err != nil {
		p.logger.Warn("[Processor.extractPictureURLs] Failed to parse pictures: ", err)
		return string(pictures)
	}

	urls := make([]string, 0, len(picList))
	for _, pic := range picList {
		urls = append(urls, pic.URLDefault)
	}
	return strings.Join(urls, ",")
}

// convertCommentToModel 转换评论为存储模型
func (p *Processor) convertCommentToModel(noteID string, comment *xhsmodel.Comment, parentCommentID string, xsecToken string) *NoteComment {
	now := time.Now().Unix()

	userID := ""
	nickname := ""
	avatar := ""
	if comment.UserInfo != nil {
		userID = comment.UserInfo.UserID
		nickname = comment.UserInfo.Nickname
		avatar = comment.UserInfo.Image
	}

	// 提取 target_comment_id（与Python版本对齐）
	targetCommentID := ""
	if comment.TargetComment != nil {
		targetCommentID = comment.TargetComment.ID
	}

	return &NoteComment{
		UserID:          userID,
		Nickname:        nickname,
		Avatar:          avatar,
		IPLocation:      comment.IPLocation,
		AddTs:           now,
		LastModifyTs:    now,
		CommentID:       comment.ID,
		CreateTime:      comment.CreateTime,
		NoteID:          noteID,
		Content:         comment.Content,
		SubCommentCount: comment.SubCommentCount,
		Pictures:        p.extractPictureURLs(comment.Pictures),
		ParentCommentID: parentCommentID,
		TargetCommentID: targetCommentID,
		LikeCount:       comment.LikeCount,
		NoteURL:         "https://www.xiaohongshu.com/explore/" + noteID + "?xsec_token=" + xsecToken + "&xsec_source=pc_search",
	}
}

// ProcessCreator 处理创作者信息
func (p *Processor) ProcessCreator(ctx context.Context, creatorInfo *xhsmodel.CreatorInfo) error {
	if creatorInfo == nil {
		return nil
	}

	now := time.Now().Unix()
	creator := &Creator{
		UserID:       creatorInfo.UserID,
		Nickname:     creatorInfo.Nickname,
		Avatar:       creatorInfo.Avatar,
		IPLocation:   creatorInfo.IPLocation,
		AddTs:        now,
		LastModifyTs: now,
		Desc:         creatorInfo.Desc,
		Gender:       creatorInfo.Gender,
		Follows:      creatorInfo.Follows,
		Fans:         creatorInfo.Fans,
		Interaction:  creatorInfo.Interaction,
		TagList:      creatorInfo.TagList,
	}

	return p.store.SaveCreator(ctx, creator)
}
