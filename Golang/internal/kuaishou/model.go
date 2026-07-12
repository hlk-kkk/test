// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/kuaishou/model.go
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

package kuaishou

type Video struct {
	ID            int64  `gorm:"primaryKey;autoIncrement;column:id"`
	UserID        string `gorm:"column:user_id"`
	Nickname      string `gorm:"column:nickname"`
	Avatar        string `gorm:"column:avatar"`
	AddTs         int64  `gorm:"column:add_ts;not null"`
	LastModifyTs  int64  `gorm:"column:last_modify_ts;not null"`
	VideoID       string `gorm:"column:video_id;not null;index"`
	VideoType     string `gorm:"column:video_type;not null"`
	Title         string `gorm:"column:title"`
	Desc          string `gorm:"column:desc;type:longtext"`
	CreateTime    string `gorm:"column:create_time;type:integer;not null;index"`
	LikedCount    string `gorm:"column:liked_count"`
	ViewdCount    string `gorm:"column:viewd_count"`
	VideoURL      string `gorm:"column:video_url;type:longtext"`
	VideoCoverURL string `gorm:"column:video_cover_url;type:longtext"`
	VideoPlayURL  string `gorm:"column:video_play_url;type:longtext"`
	SourceKeyword string `gorm:"column:source_keyword"`
}

func (Video) TableName() string { return "kuaishou_video" }

type VideoComment struct {
	ID              int64  `gorm:"primaryKey;autoIncrement;column:id"`
	UserID          string `gorm:"column:user_id"`
	Nickname        string `gorm:"column:nickname"`
	Avatar          string `gorm:"column:avatar"`
	AddTs           int64  `gorm:"column:add_ts;not null"`
	LastModifyTs    int64  `gorm:"column:last_modify_ts;not null"`
	CommentID       string `gorm:"column:comment_id;not null;index"`
	ParentCommentID string `gorm:"column:parent_comment_id"`
	VideoID         string `gorm:"column:video_id;not null;index"`
	Content         string `gorm:"column:content;type:longtext"`
	CreateTime      string `gorm:"column:create_time;type:integer;not null"`
	SubCommentCount string `gorm:"column:sub_comment_count;not null"`
	LikeCount       string `gorm:"column:like_count;not null;default:'0'"`
}

func (VideoComment) TableName() string { return "kuaishou_video_comment" }

type Creator struct {
	ID           int64  `gorm:"primaryKey;autoIncrement;column:id"`
	UserID       string `gorm:"column:user_id;not null;index"`
	Nickname     string `gorm:"column:nickname"`
	Avatar       string `gorm:"column:avatar"`
	IPLocation   string `gorm:"column:ip_location"`
	AddTs        int64  `gorm:"column:add_ts;not null"`
	LastModifyTs int64  `gorm:"column:last_modify_ts;not null"`
	Desc         string `gorm:"column:desc;type:longtext"`
	Gender       string `gorm:"column:gender"`
	Follows      string `gorm:"column:follows"`
	Fans         string `gorm:"column:fans"`
	VideosCount  string `gorm:"column:videos_count"`
}

func (Creator) TableName() string { return "kuaishou_creator" }

type HotSearchItem struct {
	ID           int64  `gorm:"primaryKey;autoIncrement;column:id"`
	HotItemID    string `gorm:"column:hot_item_id;type:varchar(191);not null;uniqueIndex"`
	Platform     string `gorm:"column:platform;not null"`
	BoardName    string `gorm:"column:board_name;not null;default:'hot_search'"`
	BoardType    string `gorm:"column:board_type;not null;default:''"`
	SnapshotDate string `gorm:"column:snapshot_date;not null"`
	SnapshotTs   int64  `gorm:"column:snapshot_ts;not null"`
	RankNo       int    `gorm:"column:rank_no;not null;default:0"`
	Title        string `gorm:"column:title;not null;default:''"`
	Keyword      string `gorm:"column:keyword;not null;default:''"`
	HotValue     string `gorm:"column:hot_value;not null;default:''"`
	Badge        string `gorm:"column:badge;not null;default:''"`
	TopicID      string `gorm:"column:topic_id;not null;default:''"`
	TopicURL     string `gorm:"column:topic_url;not null;default:''"`
	RawJSON      string `gorm:"column:raw_json"`
	AddTs        int64  `gorm:"column:add_ts;not null"`
	LastModifyTs int64  `gorm:"column:last_modify_ts;not null"`
}

func (HotSearchItem) TableName() string { return "platform_hot_search_item" }

type HotSearchContentRelation struct {
	ID            int64  `gorm:"primaryKey;autoIncrement;column:id"`
	RelID         string `gorm:"column:rel_id;type:varchar(191);not null;uniqueIndex"`
	HotItemID     string `gorm:"column:hot_item_id;type:varchar(191);not null;index"`
	Platform      string `gorm:"column:platform;not null"`
	ContentID     string `gorm:"column:content_id;not null"`
	ContentType   string `gorm:"column:content_type;not null;default:''"`
	ContentTable  string `gorm:"column:content_table;not null;default:''"`
	RankNo        int    `gorm:"column:rank_no;not null;default:0"`
	SourceKeyword string `gorm:"column:source_keyword;not null;default:''"`
	AddTs         int64  `gorm:"column:add_ts;not null"`
	LastModifyTs  int64  `gorm:"column:last_modify_ts;not null"`
}

func (HotSearchContentRelation) TableName() string { return "platform_hot_search_content_rel" }
