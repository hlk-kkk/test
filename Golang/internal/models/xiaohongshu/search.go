// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/models/xiaohongshu/search.go
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

// NoteType represents the type of note
type NoteType string

const (
	NoteTypeNormal NoteType = "normal"
	NoteTypeVideo  NoteType = "video"
)

// SearchSortType represents the sort type for search results
type SearchSortType string

const (
	// SearchSortTypeGeneral Default sort type
	SearchSortTypeGeneral SearchSortType = "general"

	// SearchSortTypeMostPopular Sort by popularity in descending order
	SearchSortTypeMostPopular SearchSortType = "popularity_descending"

	// SearchSortTypeLatest Sort by time in descending order
	SearchSortTypeLatest SearchSortType = "time_descending"
)

// SearchNoteType represents the type of notes to search for
type SearchNoteType int

const (
	// SearchNoteTypeAll Default: search all types
	SearchNoteTypeAll SearchNoteType = iota

	// SearchNoteTypeVideo Search only video notes
	SearchNoteTypeVideo

	// SearchNoteTypeImage Search only image notes
	SearchNoteTypeImage
)

// SearchRequest represents the request data structure for searching notes by keyword
type SearchRequest struct {
	Keyword  string         `json:"keyword"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	SearchID string         `json:"search_id"`
	Sort     SearchSortType `json:"sort"`
	NoteType SearchNoteType `json:"note_type"`
}

// SearchWrapper search result wrapper
type SearchWrapper struct {
	HasMore bool               `json:"has_more"`
	Items   []SearchResultNote `json:"items"`
}

// SearchResultNote item
type SearchResultNote struct {
	XsecToken string   `json:"xsec_token"`
	ID        string   `json:"id"`
	ModelType string   `json:"model_type"`
	NoteCard  NoteCard `json:"note_card"`
}

// NoteCard note card
type NoteCard struct {
	InteractInfo InteractInfo `json:"interact_info"`
	Cover        ImageInfo    `json:"cover"`
	ImageList    []ImageItem  `json:"image_list"`
	Type         string       `json:"type"`
	DisplayTitle string       `json:"display_title"`
	User         User         `json:"user"`
}

// ImageItem image item
type ImageItem struct {
	Height   int          `json:"height"`
	Width    int          `json:"width"`
	InfoList []ImageScene `json:"info_list"`
}

// ImageScene image scene
type ImageScene struct {
	ImageScene string `json:"image_scene"`
	URL        string `json:"url"`
}
