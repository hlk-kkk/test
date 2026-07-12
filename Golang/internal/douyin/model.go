// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/douyin/model.go
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

type Aweme struct {
	ID               int64  `gorm:"primaryKey;autoIncrement;column:id"`
	UserID           string `gorm:"column:user_id"`
	SecUID           string `gorm:"column:sec_uid"`
	ShortUserID      string `gorm:"column:short_user_id"`
	UserUniqueID     string `gorm:"column:user_unique_id"`
	Nickname         string `gorm:"column:nickname"`
	Avatar           string `gorm:"column:avatar"`
	UserSignature    string `gorm:"column:user_signature"`
	IPLocation       string `gorm:"column:ip_location"`
	AddTs            int64  `gorm:"column:add_ts;not null"`
	LastModifyTs     int64  `gorm:"column:last_modify_ts;not null"`
	AwemeID          string `gorm:"column:aweme_id;not null;index"`
	AwemeType        string `gorm:"column:aweme_type;not null"`
	Title            string `gorm:"column:title"`
	Desc             string `gorm:"column:desc;type:longtext"`
	CreateTime       string `gorm:"column:create_time;type:integer;not null;index"`
	LikedCount       string `gorm:"column:liked_count"`
	CommentCount     string `gorm:"column:comment_count"`
	ShareCount       string `gorm:"column:share_count"`
	CollectedCount   string `gorm:"column:collected_count"`
	AwemeURL         string `gorm:"column:aweme_url;type:longtext"`
	CoverURL         string `gorm:"column:cover_url;type:longtext"`
	VideoDownloadURL string `gorm:"column:video_download_url;type:longtext"`
	SourceKeyword    string `gorm:"column:source_keyword"`
	IsAIGenerated    int    `gorm:"column:is_ai_generated;not null;default:0"`
}

func (Aweme) TableName() string { return "douyin_aweme" }

type AwemeComment struct {
	ID              int64  `gorm:"primaryKey;autoIncrement;column:id"`
	UserID          string `gorm:"column:user_id"`
	SecUID          string `gorm:"column:sec_uid"`
	ShortUserID     string `gorm:"column:short_user_id"`
	UserUniqueID    string `gorm:"column:user_unique_id"`
	Nickname        string `gorm:"column:nickname"`
	Avatar          string `gorm:"column:avatar"`
	UserSignature   string `gorm:"column:user_signature"`
	IPLocation      string `gorm:"column:ip_location"`
	AddTs           int64  `gorm:"column:add_ts;not null"`
	LastModifyTs    int64  `gorm:"column:last_modify_ts;not null"`
	CommentID       string `gorm:"column:comment_id;not null;index"`
	AwemeID         string `gorm:"column:aweme_id;not null;index"`
	Content         string `gorm:"column:content;type:longtext"`
	CreateTime      string `gorm:"column:create_time;type:integer;not null"`
	SubCommentCount string `gorm:"column:sub_comment_count;not null"`
	ParentCommentID string `gorm:"column:parent_comment_id"`
	LikeCount       string `gorm:"column:like_count;not null;default:'0'"`
	Pictures        string `gorm:"column:pictures;not null;default:''"`
	ReplyToReplyID  string `gorm:"column:reply_to_reply_id"`
}

func (AwemeComment) TableName() string { return "douyin_aweme_comment" }

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
	Interaction  string `gorm:"column:interaction"`
	VideosCount  string `gorm:"column:videos_count"`
}

func (Creator) TableName() string { return "dy_creator" }
