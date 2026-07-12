// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/config/config_test.go
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

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigApplyDefaultsSetsPlatformEndpoints(t *testing.T) {
	cfg := &Config{}

	cfg.ApplyDefaults()

	require.NotNil(t, cfg.PlatformEndpoints)
	require.Equal(t, "https://edith.xiaohongshu.com", cfg.PlatformEndpoints.XhsApiURL)
	require.Equal(t, "https://www.xiaohongshu.com", cfg.PlatformEndpoints.XhsIndexURL)
	require.Equal(t, "https://api.bilibili.com", cfg.PlatformEndpoints.BilibiliApiURL)
	require.Equal(t, "https://www.bilibili.com", cfg.PlatformEndpoints.BilibiliIndexURL)
	require.Equal(t, "https://space.bilibili.com", cfg.PlatformEndpoints.BilibiliSpaceURL)
}

func TestConfigApplyDefaultsReadsMockEndpointEnv(t *testing.T) {
	t.Setenv("MCPRO_XHS_API_URL", "http://127.0.0.1:18081")
	t.Setenv("MCPRO_XHS_INDEX_URL", "http://127.0.0.1:18082")
	t.Setenv("MCPRO_BILIBILI_API_URL", "http://127.0.0.1:18083")
	t.Setenv("MCPRO_BILIBILI_INDEX_URL", "http://127.0.0.1:18084")
	t.Setenv("MCPRO_BILIBILI_SPACE_URL", "http://127.0.0.1:18085")
	cfg := &Config{}

	cfg.ApplyDefaults()

	require.Equal(t, "http://127.0.0.1:18081", cfg.PlatformEndpoints.XhsApiURL)
	require.Equal(t, "http://127.0.0.1:18082", cfg.PlatformEndpoints.XhsIndexURL)
	require.Equal(t, "http://127.0.0.1:18083", cfg.PlatformEndpoints.BilibiliApiURL)
	require.Equal(t, "http://127.0.0.1:18084", cfg.PlatformEndpoints.BilibiliIndexURL)
	require.Equal(t, "http://127.0.0.1:18085", cfg.PlatformEndpoints.BilibiliSpaceURL)
}
