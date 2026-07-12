// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/bilibili/model.go
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

// internal/bilibili/model.go
package bilibili

// Video 对应 bilibili_video 表
type Video struct {
	ID             int64  `gorm:"primaryKey;autoIncrement;column:id"`
	UserID         string `gorm:"column:user_id;not null"`
	Nickname       string `gorm:"column:nickname"`
	Avatar         string `gorm:"column:avatar"`
	AddTs          int64  `gorm:"column:add_ts;not null"`
	LastModifyTs   int64  `gorm:"column:last_modify_ts;not null"`
	VideoID        string `gorm:"column:video_id;not null;index"`
	Bvid           string `gorm:"column:bvid"`
	VideoType      string `gorm:"column:video_type"`
	Title          string `gorm:"column:title;type:longtext"`
	Desc           string `gorm:"column:desc;type:longtext"`
	CreateTime     int64  `gorm:"column:create_time;not null;index"`
	LikedCount     string `gorm:"column:liked_count"`
	VideoPlayCount string `gorm:"column:video_play_count"`
	VideoDanmaku   string `gorm:"column:video_danmaku"`
	VideoComment   string `gorm:"column:video_comment"`
	VideoDuration  string `gorm:"column:duration"`
	VideoURL       string `gorm:"column:video_url;type:longtext"`
	VideoCover     string `gorm:"column:video_cover_url;type:longtext"`
	SourceKeyword  string `gorm:"column:source_keyword"`
}

func (Video) TableName() string { return "bilibili_video" }

// VideoComment 对应 bilibili_video_comment 表
type VideoComment struct {
	ID              int64  `gorm:"primaryKey;autoIncrement;column:id"`
	UserID          string `gorm:"column:user_id;not null"`
	Nickname        string `gorm:"column:nickname"`
	Avatar          string `gorm:"column:avatar"`
	AddTs           int64  `gorm:"column:add_ts;not null"`
	LastModifyTs    int64  `gorm:"column:last_modify_ts;not null"`
	CommentID       string `gorm:"column:comment_id;not null;index"`
	VideoID         string `gorm:"column:video_id;not null"`
	Content         string `gorm:"column:content;type:longtext;not null"`
	CreateTime      int64  `gorm:"column:create_time;not null;index"`
	SubCommentCount string `gorm:"column:sub_comment_count;not null"`
	ParentCommentID string `gorm:"column:parent_comment_id"`
	LikeCount       string `gorm:"column:like_count;not null;default:'0'"`
}

func (VideoComment) TableName() string { return "bilibili_video_comment" }

// UpInfo 对应 bilibili_up_info 表
type UpInfo struct {
	ID             int64  `gorm:"primaryKey;autoIncrement;column:id"`
	UserID         string `gorm:"column:user_id;not null;index"`
	Nickname       string `gorm:"column:nickname"`
	Avatar         string `gorm:"column:avatar"`
	AddTs          int64  `gorm:"column:add_ts;not null"`
	LastModifyTs   int64  `gorm:"column:last_modify_ts;not null"`
	Description    string `gorm:"column:description;type:longtext"`
	FollowerCount  int64  `gorm:"column:follower_count"`
	FollowingCount int64  `gorm:"column:following_count"`
	ContentCount   int64  `gorm:"column:content_count"`
}

func (UpInfo) TableName() string { return "bilibili_up_info" }
