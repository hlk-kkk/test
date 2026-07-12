// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/models/weibo/model.go
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

package weibo

type ConfigResponse struct {
	OK   int `json:"ok"`
	Data struct {
		Login bool `json:"login"`
	} `json:"data"`
}

type APIResponse struct {
	OK   int        `json:"ok"`
	Msg  string     `json:"msg"`
	Data SearchData `json:"data"`
}

type SearchData struct {
	Cards        []Card       `json:"cards"`
	TabsInfo     TabsInfo     `json:"tabsInfo"`
	UserInfo     UserInfo     `json:"userInfo"`
	CardlistInfo CardlistInfo `json:"cardlistInfo"`
}

type TabsInfo struct {
	Tabs []Tab `json:"tabs"`
}

type Tab struct {
	TabKey      string `json:"tabKey"`
	ContainerID string `json:"containerid"`
}

type UserInfo struct {
	ID             any    `json:"id"`
	ScreenName     string `json:"screen_name"`
	Gender         string `json:"gender"`
	AvatarHD       string `json:"avatar_hd"`
	Description    string `json:"description"`
	Source         string `json:"source"`
	FollowCount    any    `json:"follow_count"`
	FollowersCount any    `json:"followers_count"`
}

type CardlistInfo struct {
	SinceID any `json:"since_id"`
	Total   int `json:"total"`
}

type Card struct {
	CardType  int    `json:"card_type"`
	CardGroup []Card `json:"card_group"`
	Mblog     Mblog  `json:"mblog"`
}

type Mblog struct {
	ID             any      `json:"id"`
	Text           string   `json:"text"`
	CreatedAt      string   `json:"created_at"`
	AttitudesCount any      `json:"attitudes_count"`
	CommentsCount  any      `json:"comments_count"`
	RepostsCount   any      `json:"reposts_count"`
	RegionName     string   `json:"region_name"`
	Pics           []Pic    `json:"pics"`
	PageInfo       PageInfo `json:"page_info"`
	User           User     `json:"user"`
}

type Pic struct {
	URL string `json:"url"`
}

type PageInfo struct {
	Type string            `json:"type"`
	URLs map[string]string `json:"urls"`
}

type User struct {
	ID              any    `json:"id"`
	ScreenName      string `json:"screen_name"`
	Gender          string `json:"gender"`
	ProfileURL      string `json:"profile_url"`
	ProfileImageURL string `json:"profile_image_url"`
}

type CommentAPIResponse struct {
	OK   int         `json:"ok"`
	Msg  string      `json:"msg"`
	Data CommentPage `json:"data"`
}

type CommentPage struct {
	MaxID     int64         `json:"max_id"`
	MaxIDType int           `json:"max_id_type"`
	Data      []CommentItem `json:"data"`
}

type CommentItem struct {
	ID          any           `json:"id"`
	Text        string        `json:"text"`
	CreatedAt   string        `json:"created_at"`
	TotalNumber any           `json:"total_number"`
	LikeCount   any           `json:"like_count"`
	Source      string        `json:"source"`
	RootID      any           `json:"rootid"`
	User        User          `json:"user"`
	Comments    []CommentItem `json:"comments"`
}

type HotBandItem struct {
	Word       string `json:"word"`
	WordScheme string `json:"word_scheme"`
	RealPos    int    `json:"realpos"`
	Num        any    `json:"num"`
	RawHot     any    `json:"raw_hot"`
	LabelName  string `json:"label_name"`
	IconDesc   string `json:"icon_desc"`
	Mid        string `json:"mid"`
	IsAd       bool   `json:"is_ad"`
}
