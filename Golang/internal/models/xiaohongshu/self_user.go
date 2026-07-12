// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/models/xiaohongshu/self_user.go
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

package xiaohongshu

// UserSelfInfo 数据主体结构
type UserSelfInfo struct {
	Result       UserSelfResult      `json:"result"`
	BasicInfo    BasicInfoStruct     `json:"basic_info"`
	Interactions []InteractionStruct `json:"interactions"`
	Tags         []TagStruct         `json:"tags"`
	TabPublic    TabPublicStruct     `json:"tab_public"`
}

// UserSelfResult 结果信息结构
type UserSelfResult struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
	Code    int    `json:"code"`
}

// BasicInfoStruct 基础信息结构
type BasicInfoStruct struct {
	ImageB     string `json:"imageb"`
	Nickname   string `json:"nickname"`
	Images     string `json:"images"`
	RedID      string `json:"red_id"`
	Gender     int    `json:"gender"`
	IPLocation string `json:"ip_location"`
	Desc       string `json:"desc"`
}

// InteractionStruct 互动信息结构
type InteractionStruct struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Count string `json:"count"`
}

// TagStruct 标签结构
type TagStruct struct {
	Icon    string `json:"icon,omitempty"`
	Name    string `json:"name"`
	TagType string `json:"tagType"`
}

// TabPublicStruct 公开标签页结构
type TabPublicStruct struct {
	Collection      bool             `json:"collection"`
	CollectionBoard CollectionStruct `json:"collectionBoard"`
	CollectionNote  CollectionStruct `json:"collectionNote"`
}

// CollectionStruct 收藏相关结构
type CollectionStruct struct {
	Display bool `json:"display"`
	Lock    bool `json:"lock"`
	Count   int  `json:"count"`
}
