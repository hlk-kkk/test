// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/constant/constant.go
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

package constant

// 所有平台名称常量
const (
	XhsPlatformName       = "xhs"
	DouyinPlatformName    = "douyin"
	DouyinPlatformAlias   = "dy"
	KuaishouPlatformName  = "kuaishou"
	KuaishouPlatformAlias = "ks"
	BilibiliPlatformName  = "bilibili"
	BilibiliPlatformAlias = "bili"
	WeiboPlatformName     = "weibo"
	WeiboPlatformAlias    = "wb"
	TiebaPlatformName     = "tieba"
	ZhihuPlatformName     = "zhihu"
)

// Checkpoint 平台名称对齐 Python 版本的 checkpoint platform 字段。
const (
	XhsCheckpointPlatform            = XhsPlatformName
	DouyinCheckpointPlatform         = DouyinPlatformAlias
	KuaishouCheckpointPlatform       = KuaishouPlatformAlias
	BilibiliCheckpointPlatform       = BilibiliPlatformAlias
	BilibiliLegacyCheckpointPlatform = BilibiliPlatformName
	WeiboCheckpointPlatform          = WeiboPlatformAlias
	TiebaCheckpointPlatform          = TiebaPlatformName
	ZhihuCheckpointPlatform          = ZhihuPlatformName
)

// 爬取类型常量
const (
	SearchCrawlerType    = "search"
	DetailCrawlerType    = "detail"
	CreatorCrawlerType   = "creator"
	HomefeedCrawlerType  = "homefeed"
	HotSearchCrawlerType = "hot_search"
)

// 保存数据类型常量
const (
	SaveDataToDb     = "db"
	SaveDataToSQLite = "sqlite"
	SaveDataToJSON   = "json"
	SaveDataToCSV    = "csv"
)

var PythonCompatiblePlatformValues = []string{
	XhsPlatformName,
	DouyinPlatformAlias,
	KuaishouPlatformAlias,
	BilibiliPlatformAlias,
	WeiboPlatformAlias,
	TiebaPlatformName,
	ZhihuPlatformName,
}

var CheckpointPlatformValues = []string{
	XhsCheckpointPlatform,
	DouyinCheckpointPlatform,
	KuaishouCheckpointPlatform,
	BilibiliCheckpointPlatform,
	WeiboCheckpointPlatform,
	TiebaCheckpointPlatform,
	ZhihuCheckpointPlatform,
}

var PythonCompatibleCrawlerTypes = []string{
	SearchCrawlerType,
	DetailCrawlerType,
	CreatorCrawlerType,
	HomefeedCrawlerType,
	HotSearchCrawlerType,
}

var PythonCompatibleSaveDataOptions = []string{
	SaveDataToDb,
	SaveDataToSQLite,
	SaveDataToJSON,
	SaveDataToCSV,
}
