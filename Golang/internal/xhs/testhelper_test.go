// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/xhs/testhelper_test.go
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

// internal/xhs/testhelper_test.go
package xhs

import (
	"fmt"
	"time"
)

// TestDataBuilder 测试数据构建器
type TestDataBuilder struct {
	timestamp int64
}

// NewTestDataBuilder 创建测试数据构建器
func NewTestDataBuilder() *TestDataBuilder {
	return &TestDataBuilder{
		timestamp: time.Now().Unix(),
	}
}

// BuildNote 构建测试笔记
func (b *TestDataBuilder) BuildNote(noteID string, opts ...NoteOption) *Note {
	note := &Note{
		UserID:         "test_user_" + noteID,
		Nickname:       "测试用户",
		Avatar:         "https://example.com/avatar.jpg",
		IPLocation:     "上海",
		AddTs:          b.timestamp,
		LastModifyTs:   b.timestamp,
		NoteID:         noteID,
		Type:           "normal",
		Title:          "测试笔记 " + noteID,
		Desc:           "这是测试笔记的描述",
		Time:           b.timestamp,
		LastUpdateTime: b.timestamp,
		LikedCount:     "100",
		CollectedCount: "50",
		CommentCount:   "20",
		ShareCount:     "10",
		ImageList:      `["img1.jpg", "img2.jpg"]`,
		TagList:        `["标签1", "标签2"]`,
		NoteURL:        fmt.Sprintf("https://www.xiaohongshu.com/%s", noteID),
		SourceKeyword:  "测试关键词",
	}

	// 应用选项
	for _, opt := range opts {
		opt(note)
	}

	return note
}

// BuildComment 构建测试评论
func (b *TestDataBuilder) BuildComment(commentID, noteID string, opts ...CommentOption) *NoteComment {
	comment := &NoteComment{
		UserID:          "test_user_" + commentID,
		Nickname:        "评论用户",
		Avatar:          "https://example.com/avatar.jpg",
		IPLocation:      "北京",
		AddTs:           b.timestamp,
		LastModifyTs:    b.timestamp,
		CommentID:       commentID,
		CreateTime:      b.timestamp,
		NoteID:          noteID,
		Content:         "这是测试评论 " + commentID,
		SubCommentCount: "0",
		LikeCount:       "5",
		NoteURL:         fmt.Sprintf("https://www.xiaohongshu.com/%s", noteID),
	}

	// 应用选项
	for _, opt := range opts {
		opt(comment)
	}

	return comment
}

// BuildCreator 构建测试创作者
func (b *TestDataBuilder) BuildCreator(userID string, opts ...CreatorOption) *Creator {
	creator := &Creator{
		UserID:       userID,
		Nickname:     "创作者 " + userID,
		Avatar:       "https://example.com/creator.jpg",
		IPLocation:   "深圳",
		AddTs:        b.timestamp,
		LastModifyTs: b.timestamp,
		Desc:         "这是测试创作者的个人简介",
		Gender:       "女",
		Follows:      "500",
		Fans:         "50000",
		Interaction:  "100000",
		TagList:      `["美妆", "时尚", "穿搭"]`,
	}

	// 应用选项
	for _, opt := range opts {
		opt(creator)
	}

	return creator
}

// BuildNotes 批量构建测试笔记
func (b *TestDataBuilder) BuildNotes(count int) []*Note {
	notes := make([]*Note, count)
	for i := 0; i < count; i++ {
		noteID := fmt.Sprintf("note_%d", i)
		notes[i] = b.BuildNote(noteID)
	}
	return notes
}

// BuildComments 批量构建测试评论
func (b *TestDataBuilder) BuildComments(noteID string, count int) []*NoteComment {
	comments := make([]*NoteComment, count)
	for i := 0; i < count; i++ {
		commentID := fmt.Sprintf("comment_%d", i)
		comments[i] = b.BuildComment(commentID, noteID)
	}
	return comments
}

// NoteOption 笔记配置选项
type NoteOption func(*Note)

// WithNoteTitle 设置笔记标题
func WithNoteTitle(title string) NoteOption {
	return func(n *Note) {
		n.Title = title
	}
}

// WithNoteDesc 设置笔记描述
func WithNoteDesc(desc string) NoteOption {
	return func(n *Note) {
		n.Desc = desc
	}
}

// WithNoteType 设置笔记类型
func WithNoteType(noteType string) NoteOption {
	return func(n *Note) {
		n.Type = noteType
	}
}

// WithNoteLikedCount 设置点赞数
func WithNoteLikedCount(count string) NoteOption {
	return func(n *Note) {
		n.LikedCount = count
	}
}

// WithNoteCollectedCount 设置收藏数
func WithNoteCollectedCount(count string) NoteOption {
	return func(n *Note) {
		n.CollectedCount = count
	}
}

// WithNoteCommentCount 设置评论数
func WithNoteCommentCount(count string) NoteOption {
	return func(n *Note) {
		n.CommentCount = count
	}
}

// WithNoteSourceKeyword 设置来源关键词
func WithNoteSourceKeyword(keyword string) NoteOption {
	return func(n *Note) {
		n.SourceKeyword = keyword
	}
}

// WithNoteVideoURL 设置视频URL
func WithNoteVideoURL(url string) NoteOption {
	return func(n *Note) {
		n.VideoURL = url
	}
}

// CommentOption 评论配置选项
type CommentOption func(*NoteComment)

// WithCommentContent 设置评论内容
func WithCommentContent(content string) CommentOption {
	return func(c *NoteComment) {
		c.Content = content
	}
}

// WithCommentLikeCount 设置评论点赞数
func WithCommentLikeCount(count string) CommentOption {
	return func(c *NoteComment) {
		c.LikeCount = count
	}
}

// WithCommentSubCount 设置子评论数
func WithCommentSubCount(count string) CommentOption {
	return func(c *NoteComment) {
		c.SubCommentCount = count
	}
}

// WithParentCommentID 设置父评论ID
func WithParentCommentID(parentID string) CommentOption {
	return func(c *NoteComment) {
		c.ParentCommentID = parentID
	}
}

// CreatorOption 创作者配置选项
type CreatorOption func(*Creator)

// WithCreatorNickname 设置创作者昵称
func WithCreatorNickname(nickname string) CreatorOption {
	return func(c *Creator) {
		c.Nickname = nickname
	}
}

// WithCreatorGender 设置创作者性别
func WithCreatorGender(gender string) CreatorOption {
	return func(c *Creator) {
		c.Gender = gender
	}
}

// WithCreatorFollows 设置关注数
func WithCreatorFollows(follows string) CreatorOption {
	return func(c *Creator) {
		c.Follows = follows
	}
}

// WithCreatorFans 设置粉丝数
func WithCreatorFans(fans string) CreatorOption {
	return func(c *Creator) {
		c.Fans = fans
	}
}

// WithCreatorInteraction 设置互动数
func WithCreatorInteraction(interaction string) CreatorOption {
	return func(c *Creator) {
		c.Interaction = interaction
	}
}

// WithCreatorDesc 设置创作者简介
func WithCreatorDesc(desc string) CreatorOption {
	return func(c *Creator) {
		c.Desc = desc
	}
}

// WithCreatorTagList 设置标签列表
func WithCreatorTagList(tagList string) CreatorOption {
	return func(c *Creator) {
		c.TagList = tagList
	}
}

// AssertNoteEqual 断言两个笔记相等(忽略ID和时间戳)
func AssertNoteEqual(expected, actual *Note) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	return expected.UserID == actual.UserID &&
		expected.NoteID == actual.NoteID &&
		expected.Title == actual.Title &&
		expected.Desc == actual.Desc &&
		expected.Type == actual.Type
}

// AssertCommentEqual 断言两个评论相等(忽略ID和时间戳)
func AssertCommentEqual(expected, actual *NoteComment) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	return expected.UserID == actual.UserID &&
		expected.CommentID == actual.CommentID &&
		expected.NoteID == actual.NoteID &&
		expected.Content == actual.Content
}

// AssertCreatorEqual 断言两个创作者相等(忽略ID和时间戳)
func AssertCreatorEqual(expected, actual *Creator) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	return expected.UserID == actual.UserID &&
		expected.Nickname == actual.Nickname &&
		expected.Gender == actual.Gender
}
