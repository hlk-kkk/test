// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/cmd/root_test.go
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

package cmd

import (
	"bytes"
	"testing"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/constant"
	"github.com/stretchr/testify/require"
)

func resetTestConfig() *config.Config {
	cfg := &config.Config{
		Platform:             "xhs",
		CrawlerType:          "search",
		SaveDataOption:       "db",
		StartPage:            1,
		CrawlerMaxNotesCount: 20,
		MaxConcurrencyNum:    1,
		RequestTimeOut:       5,
		EnableGetComments:    true,
		EnableGetSubComments: false,
		CrawlerTimeSleep:     0,
		Keywords:             []string{"deepseek"},
		DbConfig:             &config.DbConfig{},
		ProxyConfig:          &config.ProxyConfig{},
		SignSrvConfig:        &config.SignSrvConfig{},
	}
	config.Obj = cfg
	return cfg
}

func TestRootCmdExposesPythonCompatibleFlags(t *testing.T) {
	resetTestConfig()

	root := RootCmd()

	for _, name := range []string{
		"platform",
		"type",
		"keywords",
		"urls",
		"enable_checkpoint",
		"checkpoint_id",
		"enable_comments",
		"enable_sub_comments",
		"hot_search_max_items",
		"hot_search_per_item_count",
		"save_data_option",
	} {
		require.NotNil(t, root.Flags().Lookup(name), "missing Python-compatible flag --%s", name)
	}
}

func TestRootCmdHelpDocumentsPythonCompatibleValues(t *testing.T) {
	resetTestConfig()
	root := RootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--help"})

	err := root.Execute()

	require.NoError(t, err)
	help := out.String()
	require.Contains(t, help, "xhs, dy, ks, bili, wb, tieba, zhihu")
	require.Contains(t, help, "search | detail | creator | homefeed | hot_search")
	require.Contains(t, help, "db | sqlite | json | csv")
}

func TestRootCmdParsesPythonCompatiblePlatforms(t *testing.T) {
	for _, platform := range constant.PythonCompatiblePlatformValues {
		t.Run(platform, func(t *testing.T) {
			cfg := resetTestConfig()
			root := RootCmd()
			root.SetArgs([]string{"--platform", platform})

			err := root.Execute()

			require.NoError(t, err)
			require.Equal(t, platform, cfg.Platform)
		})
	}
}

func TestRootCmdParsesPythonCompatibleCrawlerTypes(t *testing.T) {
	for _, crawlerType := range constant.PythonCompatibleCrawlerTypes {
		t.Run(crawlerType, func(t *testing.T) {
			cfg := resetTestConfig()
			root := RootCmd()
			root.SetArgs([]string{"--type", crawlerType})

			err := root.Execute()

			require.NoError(t, err)
			require.Equal(t, crawlerType, cfg.CrawlerType)
		})
	}
}

func TestRootCmdParsesKeywordsLikePython(t *testing.T) {
	cfg := resetTestConfig()
	root := RootCmd()
	root.SetArgs([]string{"--platform", "xhs", "--type", "search", "--keywords", "AI,deepseek, chatgpt"})

	err := root.Execute()

	require.NoError(t, err)
	require.Equal(t, []string{"AI", "deepseek", "chatgpt"}, cfg.Keywords)
}

func TestRootCmdMapsURLsForXHSDetail(t *testing.T) {
	cfg := resetTestConfig()
	root := RootCmd()
	root.SetArgs([]string{
		"--platform", "xhs",
		"--type", "detail",
		"--urls", "https://www.xiaohongshu.com/explore/note1?xsec_token=t1&xsec_source=pc_search,https://www.xiaohongshu.com/explore/note2?xsec_token=t2&xsec_source=pc_search",
	})

	err := root.Execute()

	require.NoError(t, err)
	require.Equal(t, []string{
		"https://www.xiaohongshu.com/explore/note1?xsec_token=t1&xsec_source=pc_search",
		"https://www.xiaohongshu.com/explore/note2?xsec_token=t2&xsec_source=pc_search",
	}, cfg.SpecifiedNoteURLs)
}
