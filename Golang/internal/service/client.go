// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/client.go
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

package service

import (
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/constant"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/account_pool"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/rpc/signsrv"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/repo"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/repo/accounts_cookies"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/bilibili"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/douyin"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/kuaishou"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/tieba"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/weibo"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/xhs"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/service/zhihu"
)

type clientBaseDeps struct {
	config            *config.Config
	accountWithIpPool *account_pool.AccountWithIpPool
}

func newClientBaseDeps() clientBaseDeps {
	cfg := config.GetConfig()
	logger := tools.GetLogger()
	var cookiesAccountRepo *accounts_cookies.CookiesAccountRepo
	if cfg.AccountPoolSaveType == constant.AccountSaveTypeDb {
		db := repo.GetDB()
		cookiesAccountRepo = accounts_cookies.NewCrawlerCookiesAccountRepo(db, logger)
	}
	accountPool := account_pool.NewAccountPool(cfg, cookiesAccountRepo)
	accountWithIpPool := account_pool.NewAccountWithIpPool(accountPool)
	return clientBaseDeps{
		config:            cfg,
		accountWithIpPool: accountWithIpPool,
	}
}

func NewXhsClient() *xhs.Client {
	deps := newClientBaseDeps()
	client := signsrv.NewSignClient(deps.config)
	return xhs.NewXhsClient(deps.accountWithIpPool, client)
}

func NewBilibiliClient() *bilibili.Client {
	deps := newClientBaseDeps()
	client := signsrv.NewSignClient(deps.config)
	return bilibili.NewBilibiliClient(deps.accountWithIpPool, client)
}

func NewDouyinClient() *douyin.Client {
	deps := newClientBaseDeps()
	client := signsrv.NewSignClient(deps.config)
	return douyin.NewDouyinClient(deps.accountWithIpPool, client)
}

func NewKuaishouClient() *kuaishou.Client {
	deps := newClientBaseDeps()
	return kuaishou.NewKuaishouClient(deps.accountWithIpPool)
}

func NewWeiboClient() *weibo.Client {
	deps := newClientBaseDeps()
	return weibo.NewWeiboClient(deps.accountWithIpPool)
}

func NewTiebaClient() *tieba.Client {
	deps := newClientBaseDeps()
	return tieba.NewTiebaClient(deps.accountWithIpPool)
}

func NewZhihuClient() *zhihu.Client {
	deps := newClientBaseDeps()
	client := signsrv.NewSignClient(deps.config)
	return zhihu.NewZhihuClient(deps.accountWithIpPool, client)
}
