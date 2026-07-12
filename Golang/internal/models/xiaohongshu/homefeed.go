// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/models/xiaohongshu/homefeed.go
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

// FeedType 首页推荐类型
type FeedType string

const (
	FeedTypeRecommend FeedType = "homefeed_recommend"
)

// HomefeedResponse 首页推荐响应
type HomefeedResponse struct {
	Items       []HomefeedItem `json:"items"`
	CursorScore string         `json:"cursor_score"`
}

// HomefeedItem 首页推荐项
type HomefeedItem struct {
	ID        string           `json:"id"`
	ModelType string           `json:"model_type"`
	XsecToken string           `json:"xsec_token"`
	NoteCard  HomefeedNoteCard `json:"note_card"`
}

// HomefeedNoteCard 首页推荐的笔记卡片
type HomefeedNoteCard struct {
	NoteID       string       `json:"note_id"`
	DisplayTitle string       `json:"display_title"`
	Cover        ImageInfo    `json:"cover"`
	Type         string       `json:"type"`
	InteractInfo InteractInfo `json:"interact_info"`
	User         User         `json:"user"`
}

// HomefeedRequest 首页推荐请求
type HomefeedRequest struct {
	Category          string   `json:"category"`
	CursorScore       string   `json:"cursor_score"`
	ImageFormats      []string `json:"image_formats"`
	NeedFilterImage   bool     `json:"need_filter_image"`
	NeedNum           int      `json:"need_num"`
	NoteIndex         int      `json:"note_index"`
	Num               int      `json:"num"`
	RefreshType       int      `json:"refresh_type"`
	SearchKey         string   `json:"search_key"`
	UnreadBeginNoteID string   `json:"unread_begin_note_id"`
	UnreadEndNoteID   string   `json:"unread_end_note_id"`
	UnreadNoteCount   int      `json:"unread_note_count"`
}
