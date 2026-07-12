// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/xhs/xhs.go
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

// internal/xhs/xhs.go
package xhs

import (
	"context"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/constant"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/crawler"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	xhsclient "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/xhs"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/worker"
	"gorm.io/gorm"
)

// CrawlerOptions Crawler配置选项
type CrawlerOptions struct {
	DB         *gorm.DB
	Client     *xhsclient.Client
	WorkerSize int
	Config     *config.Config
}

// Crawler XHS爬虫主结构
type Crawler struct {
	client     *xhsclient.Client
	store      Store
	processor  *Processor
	checkpoint *crawler.CheckpointManager
	workerPool *worker.Pool
	handlers   map[string]Handler
	config     *config.Config
	logger     *tools.Logger
}

// NewCrawler 创建XHS爬虫
func NewCrawler(opts CrawlerOptions) *Crawler {
	cfg := opts.Config
	if cfg == nil {
		cfg = config.GetConfig()
	}
	cfg.ApplyDefaults()

	// 设置默认值
	workerSize := opts.WorkerSize
	if workerSize <= 0 {
		workerSize = cfg.MaxConcurrencyNum
	}
	if workerSize <= 0 {
		workerSize = 5
	}

	store, err := NewStore(cfg, opts.DB)
	if err != nil {
		panic(err)
	}
	checkpoint := crawler.NewCheckpointManager(constant.XhsCheckpointPlatform, "data/checkpoints")
	workerPool := worker.NewPool(workerSize)

	c := &Crawler{
		client:     opts.Client,
		store:      store,
		checkpoint: checkpoint,
		workerPool: workerPool,
		config:     cfg,
		logger:     tools.GetLogger(),
		handlers:   make(map[string]Handler),
	}

	// 创建Processor
	c.processor = NewProcessor(c.client, c.store, c.workerPool, c.config)

	// 注册Handlers
	deps := NewHandlerDeps(c.client, c.processor, c.checkpoint, c.workerPool, c.config)
	c.handlers[constant.SearchCrawlerType] = NewSearchHandler(deps)
	c.handlers[constant.DetailCrawlerType] = NewDetailHandler(deps)
	c.handlers[constant.CreatorCrawlerType] = NewCreatorHandler(deps)
	c.handlers[constant.HomefeedCrawlerType] = NewHomefeedHandler(deps)
	c.handlers[constant.HotSearchCrawlerType] = NewHotSearchHandler(deps)

	return c
}

// Initialize 初始化
func (c *Crawler) Initialize() {
	c.logger.Info("[Crawler.Initialize] Initializing XHS crawler")
	c.client.Initialize()
}

// Start 启动爬虫
func (c *Crawler) Start(ctx context.Context) {
	c.logger.Info("[Crawler.Start] Starting XHS crawler")

	// 检查登录状态
	if !c.client.Pong() {
		c.logger.Error("[Crawler.Start] API client pong failed, please check account")
		return
	}

	// 根据配置选择Handler
	crawlerType := c.config.CrawlerType
	handler, ok := c.handlers[crawlerType]
	if !ok {
		c.logger.Error("[Crawler.Start] Unknown crawler type: ", crawlerType)
		return
	}

	c.logger.Info("[Crawler.Start] Using handler: ", handler.Name())

	// 执行爬取
	if err := handler.Handle(ctx); err != nil {
		c.logger.Error("[Crawler.Start] Handler error: ", err)
		return
	}

	c.logger.Info("[Crawler.Start] XHS crawler finished")
}
