// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/account_pool/account_pool_test.go
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

package account_pool

import (
	"testing"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/cookiebridge"
	"github.com/stretchr/testify/require"
)

func TestAccountPoolLoadsAccountsFromCookieBridge(t *testing.T) {
	server := cookiebridge.NewMockServer(cookiebridge.Account{
		ClientID:  "client-a",
		Connected: true,
		Nicknames: map[string]string{"xhs": "小红书账号A"},
		Cookies:   map[string]string{"xhs": "a=1; b=2"},
	})
	defer server.Close()

	pool := NewAccountPool(&config.Config{
		Platform:            "xhs",
		AccountPoolSaveType: "cookie_bridge",
		CookieBridgeURL:     server.URL,
	}, nil)

	account, err := pool.GetActiveAccount()

	require.NoError(t, err)
	require.Equal(t, "小红书账号A", account.AccountName)
	require.Equal(t, "a=1; b=2", account.Cookies)
	require.Equal(t, "xhs", account.PlatformName)
}
