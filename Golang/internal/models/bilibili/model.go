// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/models/bilibili/model.go
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

package bilibili

// SearchResponse 搜索响应
type SearchResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    *SearchData `json:"data"`
}

type SearchData struct {
	NumResults int          `json:"numResults"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pagesize"`
	Result     []SearchItem `json:"result"`
}

type SearchItem struct {
	Type        string `json:"type"`
	Aid         int64  `json:"aid"`
	Bvid        string `json:"bvid"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Mid         int64  `json:"mid"`
	Play        int64  `json:"play"`
	Favorites   int64  `json:"favorites"`
	VideoReview int64  `json:"video_review"`
	Like        int64  `json:"like"`
	Pubdate     int64  `json:"pubdate"`
	Duration    string `json:"duration"`
	Pic         string `json:"pic"`
	Tag         string `json:"tag"`
}

// PopularResponse 热门视频响应
type PopularResponse struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Data    *PopularData `json:"data"`
}

type PopularData struct {
	List []VideoInfo `json:"list"`
}

// VideoInfoResponse 视频详情响应
type VideoInfoResponse struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    *VideoInfo `json:"data"`
}

type VideoDetailResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    *VideoDetailData `json:"data"`
}

type VideoDetailData struct {
	View *VideoInfo `json:"View"`
}

type VideoInfo struct {
	Bvid      string    `json:"bvid"`
	Aid       int64     `json:"aid"`
	Videos    int       `json:"videos"`
	Tid       int       `json:"tid"`
	Tname     string    `json:"tname"`
	Copyright int       `json:"copyright"`
	Pic       string    `json:"pic"`
	Title     string    `json:"title"`
	Pubdate   int64     `json:"pubdate"`
	Ctime     int64     `json:"ctime"`
	Desc      string    `json:"desc"`
	Duration  int64     `json:"duration"`
	Owner     Owner     `json:"owner"`
	Stat      VideoStat `json:"stat"`
	Dynamic   string    `json:"dynamic"`
}

type Owner struct {
	Mid  int64  `json:"mid"`
	Name string `json:"name"`
	Face string `json:"face"`
}

type VideoStat struct {
	Aid      int64 `json:"aid"`
	View     int64 `json:"view"`
	Danmaku  int64 `json:"danmaku"`
	Reply    int64 `json:"reply"`
	Favorite int64 `json:"favorite"`
	Coin     int64 `json:"coin"`
	Share    int64 `json:"share"`
	Like     int64 `json:"like"`
	NowRank  int64 `json:"now_rank"`
	HisRank  int64 `json:"his_rank"`
}

// CommentResponse 评论响应
type CommentResponse struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Data    *CommentData `json:"data"`
}

type CommentData struct {
	Cursor  CommentCursor `json:"cursor"`
	Page    CommentPage   `json:"page"`
	Replies []Comment     `json:"replies"`
}

type CommentCursor struct {
	AllCount   int                `json:"all_count"`
	IsBegin    bool               `json:"is_begin"`
	IsEnd      bool               `json:"is_end"`
	Mode       int                `json:"mode"`
	Next       int                `json:"next"`
	Pagination *CommentPagination `json:"pagination"`
}

type CommentPage struct {
	Count int `json:"count"`
}

type CommentPagination struct {
	Offset string `json:"offset"`
}

type Comment struct {
	Rpid         int64          `json:"rpid"`
	Oid          int64          `json:"oid"`
	Mid          int64          `json:"mid"`
	Root         int64          `json:"root"`
	Parent       int64          `json:"parent"`
	Count        int            `json:"count"`
	Rcount       int            `json:"rcount"`
	Like         int            `json:"like"`
	Ctime        int64          `json:"ctime"`
	Content      CommentContent `json:"content"`
	Member       CommentMember  `json:"member"`
	Replies      []Comment      `json:"replies"`
	ReplyControl *ReplyControl  `json:"reply_control"`
}

type CommentContent struct {
	Message string `json:"message"`
}

type CommentMember struct {
	Mid    string `json:"mid"`
	Uname  string `json:"uname"`
	Avatar string `json:"avatar"`
}

type ReplyControl struct {
	SubReplyCount     int    `json:"sub_reply_count"`
	SubReplyEntryText string `json:"sub_reply_entry_text"`
}

// UpInfoResponse UP主信息响应
type UpInfoResponse struct {
	Code    int     `json:"code"`
	Message string  `json:"message"`
	Data    *UpInfo `json:"data"`
}

type UpInfo struct {
	Mid       int64  `json:"mid"`
	Name      string `json:"name"`
	Sex       string `json:"sex"`
	Face      string `json:"face"`
	Sign      string `json:"sign"`
	Level     int    `json:"level"`
	Birthday  string `json:"birthday"`
	Follower  int64  `json:"-"` // 需要从 stat 接口获取
	Following int64  `json:"-"`
}

// UpStatResponse UP主统计响应
type UpStatResponse struct {
	Code    int     `json:"code"`
	Message string  `json:"message"`
	Data    *UpStat `json:"data"`
}

type UpStat struct {
	Follower  int64 `json:"follower"`
	Following int64 `json:"following"`
}

// SpaceNavNumResponse UP主空间导航栏统计响应
type SpaceNavNumResponse struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Data    *SpaceNavNum `json:"data"`
}

type SpaceNavNum struct {
	Video int64 `json:"video"`
}

// SpaceVideoResponse 空间视频响应
type SpaceVideoResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    *SpaceVideoData `json:"data"`
}

type SpaceVideoData struct {
	List           SpaceVideoList  `json:"list"`
	Page           SpaceVideoPage  `json:"page"`
	EpisodicButton *EpisodicButton `json:"episodic_button"`
}

type SpaceVideoList struct {
	Vlist []SpaceVideo `json:"vlist"`
}

type SpaceVideo struct {
	Aid         int64  `json:"aid"`
	Bvid        string `json:"bvid"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Play        int64  `json:"play"`
	Comment     int64  `json:"comment"`
	Created     int64  `json:"created"`
	Pic         string `json:"pic"`
	Length      string `json:"length"`
	Author      string `json:"author"`
	Mid         int64  `json:"mid"`
}

type SpaceVideoPage struct {
	Pn    int `json:"pn"`
	Ps    int `json:"ps"`
	Count int `json:"count"`
}

type EpisodicButton struct {
	Text string `json:"text"`
	URI  string `json:"uri"`
}

// HomefeedResponse 首页推荐响应
type HomefeedResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    *HomefeedData `json:"data"`
}

type HomefeedData struct {
	Item []HomefeedItem `json:"item"`
}

type HomefeedItem struct {
	Id       int64     `json:"id"`
	Bvid     string    `json:"bvid"`
	Cid      int64     `json:"cid"`
	Goto     string    `json:"goto"`
	ShowInfo int       `json:"show_info"`
	Uri      string    `json:"uri"`
	Pic      string    `json:"pic"`
	Title    string    `json:"title"`
	Duration int64     `json:"duration"`
	Pubdate  int64     `json:"pubdate"`
	Owner    Owner     `json:"owner"`
	Stat     VideoStat `json:"stat"`
}
