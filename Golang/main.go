// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/main.go
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

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/cmd"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/constant"
	bilibiliCrawler "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/bilibili"
	douyinCrawler "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/douyin"
	kuaishouCrawler "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/kuaishou"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/repo"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service"
	tiebaCrawler "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/tieba"
	weiboCrawler "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/weibo"
	xhsCrawler "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/xhs"
	zhihuCrawler "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/zhihu"
	"gorm.io/gorm"
)

func main() {
	// 初始化配置
	config.InitConfig()

	// 从命令行获取参数
	if err := cmd.RootCmd().Execute(); err != nil {
		fmt.Println("init command args err:", err)
	}

	cfg := config.GetConfig()
	db := dbForSaveOption(cfg)

	switch cfg.Platform {
	case constant.XhsPlatformName:
		crawler := xhsCrawler.NewCrawler(xhsCrawler.CrawlerOptions{
			DB:     db,
			Client: service.NewXhsClient(),
			Config: cfg,
		})
		crawler.Initialize()
		crawler.Start(context.Background())
	case constant.BilibiliPlatformName, constant.BilibiliPlatformAlias:
		crawler := bilibiliCrawler.NewCrawler(bilibiliCrawler.CrawlerOptions{
			DB:     db,
			Client: service.NewBilibiliClient(),
			Config: cfg,
		})
		crawler.Initialize()
		crawler.Start(context.Background())
	case constant.DouyinPlatformName, constant.DouyinPlatformAlias:
		crawler := douyinCrawler.NewCrawler(douyinCrawler.CrawlerOptions{
			DB:     db,
			Client: service.NewDouyinClient(),
			Config: cfg,
		})
		crawler.Initialize()
		crawler.Start(context.Background())
	case constant.KuaishouPlatformName, constant.KuaishouPlatformAlias:
		crawler := kuaishouCrawler.NewCrawler(kuaishouCrawler.CrawlerOptions{
			DB:     db,
			Client: service.NewKuaishouClient(),
			Config: cfg,
		})
		crawler.Initialize()
		crawler.Start(context.Background())
	case constant.WeiboPlatformName, constant.WeiboPlatformAlias:
		crawler := weiboCrawler.NewCrawler(weiboCrawler.CrawlerOptions{
			DB:     db,
			Client: service.NewWeiboClient(),
			Config: cfg,
		})
		crawler.Initialize()
		crawler.Start(context.Background())
	case constant.TiebaPlatformName:
		crawler := tiebaCrawler.NewCrawler(tiebaCrawler.CrawlerOptions{
			DB:     db,
			Client: service.NewTiebaClient(),
			Config: cfg,
		})
		crawler.Initialize()
		crawler.Start(context.Background())
	case constant.ZhihuPlatformName:
		crawler := zhihuCrawler.NewCrawler(zhihuCrawler.CrawlerOptions{
			DB:     db,
			Client: service.NewZhihuClient(),
			Config: cfg,
		})
		crawler.Initialize()
		crawler.Start(context.Background())
	default:
		fmt.Printf("platform %q not supported\n", cfg.Platform)
		os.Exit(1)
	}
}

func dbForSaveOption(cfg *config.Config) *gorm.DB {
	if cfg == nil {
		return repo.GetDB()
	}
	switch strings.ToLower(cfg.SaveDataOption) {
	case "", constant.SaveDataToDb:
		return repo.GetDB()
	default:
		return nil
	}
}
