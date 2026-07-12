// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/models/tieba/model.go
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

package tieba

type SyncResponse struct {
	ErrorCode int      `json:"error_code"`
	No        int      `json:"no"`
	Data      SyncData `json:"data"`
}

type SyncData struct {
	Anti SyncAnti `json:"anti"`
	User SyncUser `json:"user"`
}

type SyncAnti struct {
	TBS string `json:"tbs"`
}

type SyncUser struct {
	IsLogin bool `json:"is_login"`
}

type SearchResponse struct {
	ErrorCode int        `json:"error_code"`
	No        int        `json:"no"`
	Data      SearchData `json:"data"`
}

type SearchData struct {
	CardList []SearchCard `json:"card_list"`
}

type SearchCard struct {
	CardInfo  string       `json:"cardInfo"`
	CardStyle string       `json:"cardStyle"`
	Data      SearchThread `json:"data"`
}

type SearchThread struct {
	TID        any        `json:"tid"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Time       any        `json:"time"`
	CreateTime any        `json:"create_time"`
	ForumName  string     `json:"forum_name"`
	PostNum    any        `json:"post_num"`
	User       SearchUser `json:"user"`
}

type SearchUser struct {
	ShowNickname string `json:"show_nickname"`
	UserName     string `json:"user_name"`
	Portrait     string `json:"portrait"`
	PortraitH    string `json:"portraith"`
}

type PageResponse struct {
	ErrorCode    int           `json:"error_code"`
	No           int           `json:"no"`
	Thread       Thread        `json:"thread"`
	FirstFloor   FirstFloor    `json:"first_floor"`
	Forum        Forum         `json:"forum"`
	DisplayForum Forum         `json:"display_forum"`
	Page         Page          `json:"page"`
	UserList     []PageUser    `json:"user_list"`
	PostList     []CommentPost `json:"post_list"`
}

type Thread struct {
	ID               any              `json:"id"`
	TID              any              `json:"tid"`
	Title            string           `json:"title"`
	ReplyNum         any              `json:"reply_num"`
	CreateTime       any              `json:"create_time"`
	OriginThreadInfo OriginThreadInfo `json:"origin_thread_info"`
}

type OriginThreadInfo struct {
	Abstract any `json:"abstract"`
	Content  any `json:"content"`
}

type FirstFloor struct {
	TID      any    `json:"tid"`
	Title    string `json:"title"`
	AuthorID any    `json:"author_id"`
	Content  any    `json:"content"`
	Time     any    `json:"time"`
}

type Forum struct {
	ID   any    `json:"id"`
	Name string `json:"name"`
}

type Page struct {
	TotalPage any `json:"total_page"`
}

type PageUser struct {
	ID           any          `json:"id"`
	Name         string       `json:"name"`
	NameShow     string       `json:"name_show"`
	Portrait     string       `json:"portrait"`
	IPAddress    string       `json:"ip_address"`
	UserShowInfo UserShowInfo `json:"user_show_info"`
}

type UserShowInfo struct {
	FeedHead FeedHead `json:"feed_head"`
}

type FeedHead struct {
	ImageData ImageData `json:"image_data"`
}

type ImageData struct {
	ImgURL string `json:"img_url"`
}

type CommentPost struct {
	ID            any `json:"id"`
	AuthorID      any `json:"author_id"`
	SubPostNumber any `json:"sub_post_number"`
	Content       any `json:"content"`
	Time          any `json:"time"`
}

type CreatorResponse struct {
	ErrorCode int         `json:"error_code"`
	No        int         `json:"no"`
	Data      CreatorData `json:"data"`
}

type CreatorData struct {
	User CreatorUser `json:"user"`
}

type CreatorUser struct {
	ID           any          `json:"id"`
	Name         string       `json:"name"`
	NameShow     string       `json:"name_show"`
	Portrait     string       `json:"portrait"`
	Sex          any          `json:"sex"`
	Gender       any          `json:"gender"`
	IPAddress    string       `json:"ip_address"`
	ConcernNum   any          `json:"concern_num"`
	FansNum      any          `json:"fans_num"`
	TBAge        any          `json:"tb_age"`
	UserShowInfo UserShowInfo `json:"user_show_info"`
}

type CreatorThreadsResponse struct {
	ErrorCode int                `json:"error_code"`
	No        int                `json:"no"`
	Data      CreatorThreadsData `json:"data"`
}

type CreatorThreadsData struct {
	HasMore any                 `json:"has_more"`
	List    []CreatorThreadItem `json:"list"`
}

type CreatorThreadItem struct {
	ThreadInfo CreatorThreadInfo `json:"thread_info"`
}

type CreatorThreadInfo struct {
	TID any `json:"tid"`
	ID  any `json:"id"`
}
