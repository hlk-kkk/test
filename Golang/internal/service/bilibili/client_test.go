// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/bilibili/client_test.go
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
	"testing"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/stretchr/testify/require"
)

func TestClientUsesConfiguredEndpointURLs(t *testing.T) {
	cfg := &config.Config{
		PlatformEndpoints: &config.PlatformEndpointsConfig{
			BilibiliApiURL:   "http://127.0.0.1:18083",
			BilibiliIndexURL: "http://127.0.0.1:18084",
			BilibiliSpaceURL: "http://127.0.0.1:18085",
		},
	}
	cfg.ApplyDefaults()
	client := &Client{config: cfg}

	require.Equal(t, "http://127.0.0.1:18083/x/web-interface/view", client.apiURL("/x/web-interface/view"))
	require.Equal(t, "http://127.0.0.1:18084/video/BV123", client.indexURL("/video/BV123"))
	require.Equal(t, "http://127.0.0.1:18085/12345", client.spaceURL("/12345"))
}
