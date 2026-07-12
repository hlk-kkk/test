// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/client_test.go
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
	"testing"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/constant"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/cookiebridge"
	"github.com/stretchr/testify/require"
)

func TestPlatformClientFactoriesUseSharedCookieBridgeSetup(t *testing.T) {
	originalConfig := config.Obj
	t.Cleanup(func() {
		config.Obj = originalConfig
	})

	server := cookiebridge.NewMockServer(cookiebridge.Account{
		ClientID:  "client-a",
		Connected: true,
		Cookies: map[string]string{
			constant.XhsPlatformName:       "web_session=xhs; a1=xhs-a1",
			constant.BilibiliPlatformAlias: "SESSDATA=bili",
			constant.DouyinPlatformAlias:   "sessionid=dy",
			constant.KuaishouPlatformAlias: "kuaishou.api_st=ks",
			constant.WeiboPlatformAlias:    "SUB=wb",
			constant.TiebaPlatformName:     "BDUSS=tieba",
			constant.ZhihuPlatformName:     "z_c0=zhihu",
		},
	})
	defer server.Close()

	cases := []struct {
		name     string
		platform string
		new      func() any
	}{
		{name: "xhs", platform: constant.XhsPlatformName, new: func() any { return NewXhsClient() }},
		{name: "bilibili", platform: constant.BilibiliPlatformAlias, new: func() any { return NewBilibiliClient() }},
		{name: "douyin", platform: constant.DouyinPlatformAlias, new: func() any { return NewDouyinClient() }},
		{name: "kuaishou", platform: constant.KuaishouPlatformAlias, new: func() any { return NewKuaishouClient() }},
		{name: "weibo", platform: constant.WeiboPlatformAlias, new: func() any { return NewWeiboClient() }},
		{name: "tieba", platform: constant.TiebaPlatformName, new: func() any { return NewTiebaClient() }},
		{name: "zhihu", platform: constant.ZhihuPlatformName, new: func() any { return NewZhihuClient() }},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			config.Obj = &config.Config{
				Platform:            tc.platform,
				AccountPoolSaveType: constant.AccountSaveTypeCookieBridge,
				CookieBridgeURL:     server.URL,
				SignSrvConfig: &config.SignSrvConfig{
					SignSrvHost: "127.0.0.1",
					SignSrvPort: 0,
				},
			}
			config.Obj.ApplyDefaults()

			require.NotNil(t, tc.new())
		})
	}
}
