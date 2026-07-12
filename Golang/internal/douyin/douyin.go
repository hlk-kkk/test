// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/douyin/douyin.go
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

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/constant"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/crawler"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	douyinclient "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/douyin"
	"gorm.io/gorm"
)

type CrawlerOptions struct {
	DB     *gorm.DB
	Client *douyinclient.Client
	Config *config.Config
}

type Crawler struct {
	client     *douyinclient.Client
	store      Store
	checkpoint *crawler.CheckpointManager
	handlers   map[string]Handler
	config     *config.Config
	logger     *tools.Logger
}

func NewCrawler(opts CrawlerOptions) *Crawler {
	cfg := opts.Config
	if cfg == nil {
		cfg = config.GetConfig()
	}
	cfg.ApplyDefaults()

	store, err := NewStore(cfg, opts.DB)
	if err != nil {
		panic(err)
	}
	checkpoint := crawler.NewCheckpointManager(constant.DouyinCheckpointPlatform, "data/checkpoints")

	c := &Crawler{
		client:     opts.Client,
		store:      store,
		checkpoint: checkpoint,
		handlers:   make(map[string]Handler),
		config:     cfg,
		logger:     tools.GetLogger(),
	}

	deps := &HandlerDeps{
		Client:     c.client,
		Store:      c.store,
		Checkpoint: c.checkpoint,
		Config:     c.config,
		Logger:     c.logger,
	}
	c.handlers[constant.SearchCrawlerType] = NewSearchHandler(deps)
	c.handlers[constant.DetailCrawlerType] = NewDetailHandler(deps)
	c.handlers[constant.CreatorCrawlerType] = NewCreatorHandler(deps)
	c.handlers[constant.HomefeedCrawlerType] = NewHomefeedHandler(deps)
	c.handlers[constant.HotSearchCrawlerType] = NewHotSearchHandler(deps)
	return c
}

func (c *Crawler) Initialize() {
	c.logger.Info("[Crawler.Initialize] Initializing Douyin crawler")
	c.client.Initialize()
}

func (c *Crawler) Start(ctx context.Context) {
	c.logger.Info("[Crawler.Start] Starting Douyin crawler")
	if !c.client.Pong() {
		c.logger.Error("[Crawler.Start] API client pong failed, please check account")
		return
	}

	handler, ok := c.handlers[c.config.CrawlerType]
	if !ok {
		c.logger.Error("[Crawler.Start] Unknown crawler type: ", c.config.CrawlerType)
		return
	}
	c.logger.Info("[Crawler.Start] Using handler: ", handler.Name())
	if err := handler.Handle(ctx); err != nil {
		c.logger.Error("[Crawler.Start] Handler error: ", err)
		return
	}
	c.logger.Info("[Crawler.Start] Douyin crawler finished")
}
