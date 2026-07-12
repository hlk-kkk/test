// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/e2e/config.go
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

package e2e

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type CrawlerConfigOptions struct {
	Platform                  string
	Keywords                  []string
	CrawlerType               string
	SaveDataOption            string
	CrawlerMaxNotesCount      int
	HotSearchMaxItemsCount    int
	HotSearchPerItemContent   int
	EnableGetComments         bool
	EnableGetSubComments      bool
	EnableCheckpoint          bool
	SpecifiedNoteIDs          []string
	SpecifiedCreatorIDs       []string
	PlatformEndpointOverrides map[string]string
	ExtraConfigYAML           string
}

func (r *MockPlatformRunner) WriteCrawlerConfig(t testing.TB, opts CrawlerConfigOptions) string {
	t.Helper()

	host, port := r.HostPort(t)
	if opts.Platform == "" {
		opts.Platform = "xhs"
	}
	if len(opts.Keywords) == 0 {
		opts.Keywords = []string{"mock"}
	}
	if opts.CrawlerType == "" {
		opts.CrawlerType = "search"
	}
	if opts.SaveDataOption == "" {
		opts.SaveDataOption = "json"
	}
	if opts.CrawlerMaxNotesCount == 0 {
		opts.CrawlerMaxNotesCount = 1
	}
	if opts.HotSearchMaxItemsCount == 0 {
		opts.HotSearchMaxItemsCount = 1
	}
	if opts.HotSearchPerItemContent == 0 {
		opts.HotSearchPerItemContent = 1
	}

	var builder strings.Builder
	writeScalar(&builder, "platform", opts.Platform)
	writeStringList(&builder, "keywords", opts.Keywords)
	writeScalar(&builder, "crawlerType", opts.CrawlerType)
	writeScalar(&builder, "saveDataOption", opts.SaveDataOption)
	writeScalar(&builder, "accountPoolSaveType", "cookie_bridge")
	writeScalar(&builder, "cookieBridgeURL", r.BaseURL)
	writeScalar(&builder, "dataDir", r.DataDir)
	writeScalar(&builder, "sqliteDbPath", filepath.Join(r.WorkDir, "media_crawler.db"))
	writeInt(&builder, "startPage", 1)
	writeInt(&builder, "crawlerMaxNotesCount", opts.CrawlerMaxNotesCount)
	writeInt(&builder, "maxConcurrencyNum", 1)
	writeInt(&builder, "requestTimeOut", 2)
	writeInt(&builder, "hotSearchMaxItemsCount", opts.HotSearchMaxItemsCount)
	writeInt(&builder, "hotSearchPerItemContentCount", opts.HotSearchPerItemContent)
	writeBool(&builder, "enableGetComments", opts.EnableGetComments)
	writeBool(&builder, "enableGetSubComments", opts.EnableGetSubComments)
	writeBool(&builder, "enableCheckpoint", opts.EnableCheckpoint)
	writeInt(&builder, "perNoteMaxComments", 2)
	writeInt(&builder, "crawlerTimeSleep", 0)
	writeStringList(&builder, "specifiedNoteIds", opts.SpecifiedNoteIDs)
	writeStringList(&builder, "specifiedNoteURLs", opts.SpecifiedNoteIDs)
	writeStringList(&builder, "specifiedCreatorIds", opts.SpecifiedCreatorIDs)
	writeStringList(&builder, "specifiedCreatorURLs", opts.SpecifiedCreatorIDs)
	builder.WriteString("proxyConfig:\n")
	builder.WriteString("  enableIpProxy: false\n")
	builder.WriteString("signSrvConfig:\n")
	builder.WriteString(fmt.Sprintf("  signSrvHost: %q\n", host))
	builder.WriteString(fmt.Sprintf("  signSrvPort: %s\n", port))
	builder.WriteString("platformEndpoints:\n")
	for _, endpoint := range platformEndpointLines(r.BaseURL, opts.PlatformEndpointOverrides) {
		builder.WriteString(endpoint)
	}
	if strings.TrimSpace(opts.ExtraConfigYAML) != "" {
		builder.WriteString(strings.TrimRight(opts.ExtraConfigYAML, "\n"))
		builder.WriteString("\n")
	}
	return r.WriteConfig(t, builder.String())
}

func platformEndpointLines(baseURL string, overrides map[string]string) []string {
	endpoints := map[string]string{
		"xhsApiUrl":          baseURL,
		"xhsIndexUrl":        baseURL,
		"bilibiliApiUrl":     baseURL,
		"bilibiliIndexUrl":   baseURL,
		"bilibiliSpaceUrl":   baseURL,
		"douyinApiUrl":       baseURL,
		"douyinIndexUrl":     baseURL,
		"kuaishouApiUrl":     baseURL,
		"kuaishouRestApiUrl": baseURL,
		"weiboApiUrl":        baseURL,
		"weiboHotSearchUrl":  baseURL,
		"weiboHotBandUrl":    baseURL + "/ajax/statuses/hot_band",
		"tiebaUrl":           baseURL,
		"zhihuUrl":           baseURL,
		"zhihuZhuanlanUrl":   baseURL,
	}
	for key, value := range overrides {
		endpoints[key] = value
	}
	keys := make([]string, 0, len(endpoints))
	for key := range endpoints {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("  %s: %q\n", key, endpoints[key]))
	}
	return lines
}

func writeScalar(builder *strings.Builder, key, value string) {
	builder.WriteString(fmt.Sprintf("%s: %q\n", key, value))
}

func writeInt(builder *strings.Builder, key string, value int) {
	builder.WriteString(fmt.Sprintf("%s: %d\n", key, value))
}

func writeBool(builder *strings.Builder, key string, value bool) {
	builder.WriteString(fmt.Sprintf("%s: %t\n", key, value))
}

func writeStringList(builder *strings.Builder, key string, values []string) {
	builder.WriteString(key + ":\n")
	for _, value := range values {
		builder.WriteString(fmt.Sprintf("  - %q\n", value))
	}
}
