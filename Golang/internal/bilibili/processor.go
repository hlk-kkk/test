// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/bilibili/processor.go
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
	"strconv"
	"sync/atomic"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	bilimodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/bilibili"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	biliclient "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/bilibili"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/worker"
)

// Processor 数据处理器
type Processor struct {
	client     *biliclient.Client
	store      Store
	workerPool *worker.Pool
	config     *config.Config
	logger     *tools.Logger
}

func NewProcessor(client *biliclient.Client, store Store, workerPool *worker.Pool, cfg *config.Config) *Processor {
	return &Processor{
		client:     client,
		store:      store,
		workerPool: workerPool,
		config:     cfg,
		logger:     tools.GetLogger(),
	}
}

// ProcessVideos 批量处理视频
func (p *Processor) ProcessVideos(ctx context.Context, videoItems []VideoItem, sourceKeyword string) (int, error) {
	if len(videoItems) == 0 {
		return 0, nil
	}

	p.logger.Info("[Processor.ProcessVideos] Processing ", len(videoItems), " videos")

	var successCount int32
	tasks := make([]worker.Task, 0, len(videoItems))

	for _, item := range videoItems {
		videoItem := item // 捕获变量
		tasks = append(tasks, func(ctx context.Context) error {
			err := p.processOneVideo(ctx, videoItem, sourceKeyword)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
			return err
		})
	}

	errs := p.workerPool.Submit(ctx, tasks)
	if len(errs) > 0 {
		p.logger.Error("[Processor.ProcessVideos] Failed count: ", len(errs))
	}

	return int(successCount), nil
}

// processOneVideo 处理单个视频
func (p *Processor) processOneVideo(ctx context.Context, item VideoItem, sourceKeyword string) error {
	videoID := strconv.FormatInt(item.Aid, 10)
	if videoID == "0" && item.Bvid != "" {
		videoID = item.Bvid
	}

	// 检查是否已存在
	exists, _ := p.store.VideoExists(ctx, videoID)
	if exists {
		p.logger.Info("[Processor.processOneVideo] Video already exists: ", videoID)
		return nil
	}

	// 获取视频详情
	videoInfo, err := p.client.GetVideoInfo(item.Bvid, item.Aid)
	if err != nil {
		p.logger.Error("[Processor.processOneVideo] Get video info failed: ", videoID, err)
		return err
	}

	// 转换为存储模型
	video := p.convertVideoInfoToModel(videoInfo)
	video.SourceKeyword = sourceKeyword
	video.VideoURL = "https://www.bilibili.com/video/" + videoInfo.Bvid

	// 保存
	if err := p.store.SaveVideo(ctx, video); err != nil {
		p.logger.Error("[Processor.processOneVideo] Save video failed: ", videoID, err)
		return err
	}

	p.logger.Info("[Processor.processOneVideo] Saved video: ", videoID)
	return nil
}

// convertVideoInfoToModel 将API响应转换为存储模型
func (p *Processor) convertVideoInfoToModel(info *bilimodel.VideoInfo) *Video {
	now := time.Now().Unix()

	return &Video{
		VideoID:        strconv.FormatInt(info.Aid, 10),
		Bvid:           info.Bvid,
		UserID:         strconv.FormatInt(info.Owner.Mid, 10),
		Nickname:       info.Owner.Name,
		Avatar:         info.Owner.Face,
		AddTs:          now,
		LastModifyTs:   now,
		VideoType:      info.Tname,
		Title:          info.Title,
		Desc:           info.Desc,
		CreateTime:     info.Pubdate,
		LikedCount:     strconv.FormatInt(info.Stat.Like, 10),
		VideoPlayCount: strconv.FormatInt(info.Stat.View, 10),
		VideoDanmaku:   strconv.FormatInt(info.Stat.Danmaku, 10),
		VideoComment:   strconv.FormatInt(info.Stat.Reply, 10),
		VideoDuration:  strconv.FormatInt(info.Duration, 10),
		VideoCover:     info.Pic,
	}
}

// ProcessComments 批量处理视频评论
func (p *Processor) ProcessComments(ctx context.Context, aids []int64) {
	if len(aids) == 0 || !p.config.EnableGetComments {
		return
	}

	p.logger.Info("[Processor.ProcessComments] Processing comments for ", len(aids), " videos")

	tasks := make([]worker.Task, 0, len(aids))
	for _, aid := range aids {
		videoAid := aid
		tasks = append(tasks, func(ctx context.Context) error {
			return p.processVideoComments(ctx, videoAid)
		})
	}

	p.workerPool.Submit(ctx, tasks)
}

// processVideoComments 处理单个视频的所有评论
func (p *Processor) processVideoComments(ctx context.Context, aid int64) error {
	cursor := 0
	totalComments := 0
	maxComments := p.config.PerNoteMaxComments
	if maxComments <= 0 {
		maxComments = 100
	}
	pageSize := 20

	for totalComments < maxComments {
		commentData, err := p.client.GetVideoComments(aid, cursor, pageSize)
		if err != nil {
			p.logger.Error("[Processor.processVideoComments] Get comments failed: ", aid, err)
			return err
		}

		if commentData == nil || len(commentData.Replies) == 0 {
			break
		}

		// 保存评论
		for _, reply := range commentData.Replies {
			commentModel := p.convertCommentToModel(aid, &reply, 0)
			p.store.SaveComment(ctx, commentModel)
			totalComments++

			// 处理子评论
			if p.config.EnableGetSubComments && reply.Rcount > 0 {
				if err := p.processSubComments(ctx, aid, reply.Rpid); err != nil {
					p.logger.Error("[Processor.processVideoComments] Get sub comments failed: ", aid, reply.Rpid, err)
					return err
				}
			}
		}

		if commentData.Cursor.IsEnd {
			break
		}
		cursor = commentData.Cursor.Next
	}

	p.logger.Info("[Processor.processVideoComments] Processed ", totalComments, " comments for video: ", aid)
	return nil
}

func (p *Processor) processSubComments(ctx context.Context, aid int64, rootRpid int64) error {
	pageNum := 1
	pageSize := 10

	for {
		commentData, err := p.client.GetVideoSubComments(aid, rootRpid, pageNum, pageSize)
		if err != nil {
			return err
		}
		if commentData == nil || len(commentData.Replies) == 0 {
			return nil
		}

		for _, subReply := range commentData.Replies {
			subCommentModel := p.convertCommentToModel(aid, &subReply, rootRpid)
			p.store.SaveComment(ctx, subCommentModel)
		}

		if commentData.Page.Count <= pageNum*pageSize {
			return nil
		}
		pageNum++
	}
}

// convertCommentToModel 转换评论为存储模型
func (p *Processor) convertCommentToModel(aid int64, comment *bilimodel.Comment, parentRpid int64) *VideoComment {
	now := time.Now().Unix()

	parentCommentID := ""
	if parentRpid > 0 {
		parentCommentID = strconv.FormatInt(parentRpid, 10)
	}

	return &VideoComment{
		CommentID:       strconv.FormatInt(comment.Rpid, 10),
		VideoID:         strconv.FormatInt(aid, 10),
		UserID:          comment.Member.Mid,
		Nickname:        comment.Member.Uname,
		Avatar:          comment.Member.Avatar,
		AddTs:           now,
		LastModifyTs:    now,
		Content:         comment.Content.Message,
		CreateTime:      comment.Ctime,
		SubCommentCount: strconv.Itoa(comment.Rcount),
		ParentCommentID: parentCommentID,
		LikeCount:       strconv.Itoa(comment.Like),
	}
}

// ProcessUpInfo 处理UP主信息
func (p *Processor) ProcessUpInfo(ctx context.Context, upInfo *bilimodel.UpInfo, upStat *bilimodel.UpStat, navNum *bilimodel.SpaceNavNum) error {
	if upInfo == nil {
		return nil
	}

	now := time.Now().Unix()
	info := &UpInfo{
		UserID:       strconv.FormatInt(upInfo.Mid, 10),
		Nickname:     upInfo.Name,
		Avatar:       upInfo.Face,
		AddTs:        now,
		LastModifyTs: now,
		Description:  upInfo.Sign,
	}

	if upStat != nil {
		info.FollowerCount = upStat.Follower
		info.FollowingCount = upStat.Following
	}
	if navNum != nil {
		info.ContentCount = navNum.Video
	}

	return p.store.SaveUpInfo(ctx, info)
}
