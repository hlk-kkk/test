// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/tieba/model.go
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

type Note struct {
	ID              int64  `gorm:"primaryKey;autoIncrement;column:id"`
	NoteID          string `gorm:"column:note_id;not null;index"`
	Title           string `gorm:"column:title;not null"`
	Desc            string `gorm:"column:desc;type:longtext"`
	NoteURL         string `gorm:"column:note_url;not null"`
	PublishTime     string `gorm:"column:publish_time;not null;index"`
	UserLink        string `gorm:"column:user_link"`
	UserNickname    string `gorm:"column:user_nickname"`
	UserAvatar      string `gorm:"column:user_avatar"`
	TiebaID         string `gorm:"column:tieba_id"`
	TiebaName       string `gorm:"column:tieba_name;not null"`
	TiebaLink       string `gorm:"column:tieba_link;not null"`
	TotalReplayNum  int    `gorm:"column:total_replay_num;default:0"`
	TotalReplayPage int    `gorm:"column:total_replay_page;default:0"`
	IPLocation      string `gorm:"column:ip_location"`
	AddTs           int64  `gorm:"column:add_ts;not null"`
	LastModifyTs    int64  `gorm:"column:last_modify_ts;not null"`
	SourceKeyword   string `gorm:"column:source_keyword"`
}

func (Note) TableName() string { return "tieba_note" }

type Comment struct {
	ID              int64  `gorm:"primaryKey;autoIncrement;column:id"`
	CommentID       string `gorm:"column:comment_id;not null;index"`
	ParentCommentID string `gorm:"column:parent_comment_id"`
	Content         string `gorm:"column:content;type:longtext;not null"`
	UserLink        string `gorm:"column:user_link"`
	UserNickname    string `gorm:"column:user_nickname"`
	UserAvatar      string `gorm:"column:user_avatar"`
	TiebaID         string `gorm:"column:tieba_id"`
	TiebaName       string `gorm:"column:tieba_name;not null"`
	TiebaLink       string `gorm:"column:tieba_link;not null"`
	PublishTime     string `gorm:"column:publish_time;index"`
	IPLocation      string `gorm:"column:ip_location"`
	SubCommentCount int    `gorm:"column:sub_comment_count;default:0"`
	NoteID          string `gorm:"column:note_id;not null;index"`
	NoteURL         string `gorm:"column:note_url;not null"`
	AddTs           int64  `gorm:"column:add_ts;not null"`
	LastModifyTs    int64  `gorm:"column:last_modify_ts;not null"`
}

func (Comment) TableName() string { return "tieba_comment" }

type Creator struct {
	ID                   int64  `gorm:"primaryKey;autoIncrement;column:id"`
	UserID               string `gorm:"column:user_id;not null"`
	UserName             string `gorm:"column:user_name;not null"`
	Nickname             string `gorm:"column:nickname"`
	Avatar               string `gorm:"column:avatar"`
	IPLocation           string `gorm:"column:ip_location"`
	AddTs                int64  `gorm:"column:add_ts;not null"`
	LastModifyTs         int64  `gorm:"column:last_modify_ts;not null"`
	Gender               string `gorm:"column:gender"`
	Follows              string `gorm:"column:follows"`
	Fans                 string `gorm:"column:fans"`
	RegistrationDuration string `gorm:"column:registration_duration"`
}

func (Creator) TableName() string { return "tieba_creator" }
