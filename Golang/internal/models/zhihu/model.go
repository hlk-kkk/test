// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/models/zhihu/model.go
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

package zhihu

type SearchResponse struct {
	Data []SearchItem `json:"data"`
}

type CreatorAnswersResponse struct {
	Paging Paging           `json:"paging"`
	Data   []*ContentObject `json:"data"`
}

type HomefeedResponse struct {
	Paging Paging         `json:"paging"`
	Data   []HomefeedItem `json:"data"`
}

type HomefeedItem struct {
	Type   string         `json:"type"`
	Target *ContentObject `json:"target"`
}

type HotSearchResponse struct {
	Data []HotSearchItem `json:"data"`
}

type HotSearchItem struct {
	Question Question `json:"question"`
	Score    any      `json:"score"`
	Heat     any      `json:"heat"`
	Label    string   `json:"label"`
}

type QuestionFeedsResponse struct {
	Paging Paging         `json:"paging"`
	Data   []QuestionFeed `json:"data"`
}

type QuestionFeed struct {
	TargetType string         `json:"target_type"`
	Target     *ContentObject `json:"target"`
}

type SearchItem struct {
	Type   string         `json:"type"`
	Object *ContentObject `json:"object"`
}

type ContentObject struct {
	Type              string   `json:"type"`
	ID                any      `json:"id"`
	Content           string   `json:"content"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	Excerpt           string   `json:"excerpt"`
	CreatedTime       any      `json:"created_time"`
	CreatedTimeCamel  any      `json:"createdTime"`
	Created           any      `json:"created"`
	UpdatedTime       any      `json:"updated_time"`
	UpdatedTimeCamel  any      `json:"updatedTime"`
	Updated           any      `json:"updated"`
	VoteupCount       any      `json:"voteup_count"`
	VoteupCountCamel  any      `json:"voteupCount"`
	CommentCount      any      `json:"comment_count"`
	CommentCountCamel any      `json:"commentCount"`
	Question          Question `json:"question"`
	Author            Author   `json:"author"`
}

type Question struct {
	ID    any    `json:"id"`
	Title string `json:"title"`
}

type Author struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	AvatarURL   string  `json:"avatar_url"`
	AvatarURL2  string  `json:"avatarUrl"`
	URLToken    string  `json:"url_token"`
	URLToken2   string  `json:"urlToken"`
	IsAnonymous bool    `json:"is_anonymous"`
	Member      *Author `json:"member"`
}

type CreatorInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	AvatarURL      string `json:"avatarUrl"`
	AvatarURLSnake string `json:"avatar_url"`
	URLToken       string `json:"urlToken"`
	URLTokenSnake  string `json:"url_token"`
	Gender         any    `json:"gender"`
	IPInfo         string `json:"ipInfo"`
	FollowingCount any    `json:"followingCount"`
	FollowerCount  any    `json:"followerCount"`
	AnswerCount    any    `json:"answerCount"`
	ZvideoCount    any    `json:"zvideoCount"`
	QuestionCount  any    `json:"questionCount"`
	ArticlesCount  any    `json:"articlesCount"`
	ColumnsCount   any    `json:"columnsCount"`
	VoteupCount    any    `json:"voteupCount"`
}

type MeResponse struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
}

type CommentResponse struct {
	Paging Paging          `json:"paging"`
	Data   []CommentObject `json:"data"`
}

type Paging struct {
	IsEnd bool   `json:"is_end"`
	Next  string `json:"next"`
}

type CommentObject struct {
	Type              string       `json:"type"`
	ID                any          `json:"id"`
	ReplyCommentID    any          `json:"reply_comment_id"`
	Content           string       `json:"content"`
	CreatedTime       any          `json:"created_time"`
	CommentTag        []CommentTag `json:"comment_tag"`
	ChildCommentCount any          `json:"child_comment_count"`
	LikeCount         any          `json:"like_count"`
	DislikeCount      any          `json:"dislike_count"`
	Author            Author       `json:"author"`
}

type CommentTag struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
