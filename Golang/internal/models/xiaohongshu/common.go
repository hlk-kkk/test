// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/models/xiaohongshu/common.go
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

// Response generic response structure
type Response[T any] struct {
	Code    int    `json:"code"`
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	Data    T      `json:"data"`
}

// ImageInfo 图片详细信息
type ImageInfo struct {
	ImageScene string `json:"image_scene"`
	URL        string `json:"url"`
}

// InteractInfo 交互信息
type InteractInfo struct {
	CollectedCount string `json:"collected_count"`
	CommentCount   string `json:"comment_count"`
	ShareCount     string `json:"share_count"`
	Followed       bool   `json:"followed"`
	Relation       string `json:"relation"`
	Liked          bool   `json:"liked"`
	LikedCount     string `json:"liked_count"`
	Collected      bool   `json:"collected"`
}

// TagInfo 标签信息
type TagInfo struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// User user info
type User struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	UserID   string `json:"user_id"`
}
