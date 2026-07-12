// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/models/douyin/model.go
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

package douyin

type SearchResponse struct {
	StatusCode int          `json:"status_code"`
	StatusMsg  string       `json:"status_msg"`
	Data       []SearchItem `json:"data"`
	Extra      SearchExtra  `json:"extra"`
}

type SearchExtra struct {
	LogID string `json:"logid"`
}

type SearchItem struct {
	AwemeInfo *AwemeInfo `json:"aweme_info"`
}

type HistoryReadResponse struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type AwemeDetailResponse struct {
	StatusCode  int        `json:"status_code"`
	StatusMsg   string     `json:"status_msg"`
	AwemeDetail *AwemeInfo `json:"aweme_detail"`
}

type AwemeInfo struct {
	AwemeID      string     `json:"aweme_id"`
	AwemeType    int        `json:"aweme_type"`
	PreviewTitle string     `json:"preview_title"`
	Desc         string     `json:"desc"`
	CreateTime   int64      `json:"create_time"`
	IPLabel      string     `json:"ip_label"`
	Statistics   Statistics `json:"statistics"`
	AIGCInfo     AIGCInfo   `json:"aigc_info"`
	Video        Video      `json:"video"`
	Author       Author     `json:"author"`
}

type Statistics struct {
	DiggCount    int64 `json:"digg_count"`
	CommentCount int64 `json:"comment_count"`
	ShareCount   int64 `json:"share_count"`
	CollectCount int64 `json:"collect_count"`
}

type AIGCInfo struct {
	AIGCLabelType int `json:"aigc_label_type"`
}

type Video struct {
	RawCover     URLList `json:"raw_cover"`
	OriginCover  URLList `json:"origin_cover"`
	PlayAddrH264 URLList `json:"play_addr_h264"`
	PlayAddr256  URLList `json:"play_addr_256"`
	PlayAddr     URLList `json:"play_addr"`
}

type URLList struct {
	URLList []string `json:"url_list"`
}

type Author struct {
	UID         string  `json:"uid"`
	SecUID      string  `json:"sec_uid"`
	ShortID     string  `json:"short_id"`
	UniqueID    string  `json:"unique_id"`
	Nickname    string  `json:"nickname"`
	Signature   string  `json:"signature"`
	AvatarThumb URLList `json:"avatar_thumb"`
}

type UserProfileResponse struct {
	StatusCode int         `json:"status_code"`
	StatusMsg  string      `json:"status_msg"`
	User       CreatorInfo `json:"user"`
}

type CreatorInfo struct {
	UID          string            `json:"uid"`
	Nickname     string            `json:"nickname"`
	Signature    string            `json:"signature"`
	IPLocation   string            `json:"ip_location"`
	Gender       int               `json:"gender"`
	AvatarLarger URLList           `json:"avatar_larger"`
	Statistics   CreatorStatistics `json:"statistics"`
}

type CreatorStatistics struct {
	FollowingCount int64 `json:"following_count"`
	FollowerCount  int64 `json:"follower_count"`
	TotalFavorited int64 `json:"total_favorited"`
	AwemeCount     int64 `json:"aweme_count"`
}

type UserAwemePostsResponse struct {
	StatusCode int         `json:"status_code"`
	StatusMsg  string      `json:"status_msg"`
	HasMore    int         `json:"has_more"`
	MaxCursor  string      `json:"max_cursor"`
	AwemeList  []AwemeInfo `json:"aweme_list"`
}

type HomefeedResponse struct {
	StatusCode int            `json:"StatusCode"`
	Cards      []HomefeedCard `json:"cards"`
}

type HomefeedCard struct {
	Type  int    `json:"type"`
	Aweme string `json:"aweme"`
}

type CommentResponse struct {
	StatusCode int           `json:"status_code"`
	HasMore    int           `json:"has_more"`
	Cursor     int64         `json:"cursor"`
	Comments   []CommentInfo `json:"comments"`
}

type CommentInfo struct {
	CID               string         `json:"cid"`
	Text              string         `json:"text"`
	CreateTime        int64          `json:"create_time"`
	ReplyCommentTotal int            `json:"reply_comment_total"`
	ReplyID           string         `json:"reply_id"`
	ReplyToReplyID    string         `json:"reply_to_reply_id"`
	DiggCount         int64          `json:"digg_count"`
	IPLabel           string         `json:"ip_label"`
	ImageList         []CommentImage `json:"image_list"`
	User              Author         `json:"user"`
}

type CommentImage struct {
	OriginURL URLList `json:"origin_url"`
}

type HotSearchResponse struct {
	StatusCode int           `json:"status_code"`
	StatusMsg  string        `json:"status_msg"`
	Data       HotSearchData `json:"data"`
}

type HotSearchData struct {
	WordList []HotSearchItem `json:"word_list"`
}

type HotSearchItem struct {
	Word       string `json:"word"`
	Position   int    `json:"position"`
	GroupID    string `json:"group_id"`
	SentenceID string `json:"sentence_id"`
	HotValue   string `json:"hot_value"`
	Label      string `json:"label"`
}

type HotSearchAwemeListResponse struct {
	StatusCode int                 `json:"status_code"`
	StatusMsg  string              `json:"status_msg"`
	AwemeList  []HotSearchAwemeRef `json:"aweme_list"`
}

type HotSearchAwemeRef struct {
	AwemeID string `json:"aweme_id"`
}
