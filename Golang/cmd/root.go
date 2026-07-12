// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/cmd/root.go
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
	"strings"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/constant"
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	conf := config.GetConfig()
	keywords := strings.Join(conf.Keywords, ",")
	urls := ""
	disableCheckpoint := false
	disableComments := false
	disableSubComments := false
	var rootCmd = &cobra.Command{
		Use:   "mcpro",
		Short: "Media crawler program for multiple platforms (e.g. xhs, douyin, etc.)",
		Long: `Media crawler program for multiple platforms (e.g. xhs, douyin, etc.)
小红书爬虫，抖音爬虫， 快手爬虫， B站爬虫， 微博爬虫，百度贴吧...。
Github Repo: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang`,
		Run: func(cmd *cobra.Command, args []string) {
			conf.Keywords = splitCommaSeparated(keywords)
			if disableCheckpoint {
				conf.EnableCheckpoint = false
			}
			if disableComments {
				conf.EnableGetComments = false
			}
			if disableSubComments {
				conf.EnableGetSubComments = false
			}
			if urls != "" {
				overrideURLsConfig(conf, splitCommaSeparated(urls))
			}
		},
	}
	rootCmd.Flags().StringVar(&conf.Platform, "platform", conf.Platform, "crawler platform name ("+strings.Join(constant.PythonCompatiblePlatformValues, ", ")+")")
	rootCmd.Flags().StringVar(&conf.CrawlerType, "type", conf.CrawlerType, "crawler type ("+strings.Join(constant.PythonCompatibleCrawlerTypes, " | ")+")")
	rootCmd.Flags().StringVar(&keywords, "keywords", keywords, "search keywords, use ',' to split")
	rootCmd.Flags().StringVar(&urls, "urls", urls, "detail/creator mode URLs or IDs, use ',' to split")
	rootCmd.Flags().BoolVar(&conf.EnableCheckpoint, "enable_checkpoint", conf.EnableCheckpoint, "enable checkpoint resume")
	rootCmd.Flags().BoolVar(&disableCheckpoint, "no-enable_checkpoint", false, "disable checkpoint resume")
	rootCmd.Flags().StringVar(&conf.SpecifiedCheckpointID, "checkpoint_id", conf.SpecifiedCheckpointID, "specified checkpoint id")
	rootCmd.Flags().BoolVar(&conf.EnableGetComments, "enable_comments", conf.EnableGetComments, "enable comments crawling")
	rootCmd.Flags().BoolVar(&disableComments, "no-enable_comments", false, "disable comments crawling")
	rootCmd.Flags().BoolVar(&conf.EnableGetSubComments, "enable_sub_comments", conf.EnableGetSubComments, "enable sub-comments crawling")
	rootCmd.Flags().BoolVar(&disableSubComments, "no-enable_sub_comments", false, "disable sub-comments crawling")
	rootCmd.Flags().IntVar(&conf.HotSearchMaxItemsCount, "hot_search_max_items", conf.HotSearchMaxItemsCount, "hot_search max board items")
	rootCmd.Flags().IntVar(&conf.HotSearchPerItemContentCount, "hot_search_per_item_count", conf.HotSearchPerItemContentCount, "hot_search max contents per board item")
	rootCmd.Flags().IntVar(&conf.StartPage, "start", conf.StartPage, "number of start page")
	rootCmd.Flags().StringVar(&conf.SaveDataOption, "save_data_option", conf.SaveDataOption, "where to save the data ("+strings.Join(constant.PythonCompatibleSaveDataOptions, " | ")+")")

	return rootCmd
}

func splitCommaSeparated(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func overrideURLsConfig(conf *config.Config, urlList []string) {
	switch conf.CrawlerType {
	case constant.DetailCrawlerType:
		conf.SpecifiedNoteURLs = urlList
		conf.SpecifiedNoteIDs = urlList
	case constant.CreatorCrawlerType:
		conf.SpecifiedCreatorURLs = urlList
		conf.SpecifiedCreatorIDs = urlList
	}
}
