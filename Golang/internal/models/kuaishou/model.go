// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/models/kuaishou/model.go
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

import (
	"encoding/json"
	"strconv"
)

type GraphQLError struct {
	Message string `json:"message"`
}

type ProfileUserListGraphQLResponse struct {
	Errors []GraphQLError `json:"errors"`
	Data   struct {
		VisionProfileUserList ProfileUserListResult `json:"visionProfileUserList"`
	} `json:"data"`
}

type ProfileUserListResult struct {
	Result int `json:"result"`
}

type SearchGraphQLResponse struct {
	Errors []GraphQLError `json:"errors"`
	Data   struct {
		VisionSearchPhoto SearchResult `json:"visionSearchPhoto"`
	} `json:"data"`
}

type DetailGraphQLResponse struct {
	Errors []GraphQLError `json:"errors"`
	Data   struct {
		VisionVideoDetail Feed `json:"visionVideoDetail"`
	} `json:"data"`
}

type ProfileGraphQLResponse struct {
	Errors []GraphQLError `json:"errors"`
	Data   struct {
		VisionProfile ProfileResult `json:"visionProfile"`
	} `json:"data"`
}

type ProfileResult struct {
	UserProfile CreatorProfile `json:"userProfile"`
}

type CreatorProfile struct {
	OwnerCount CreatorOwnerCount  `json:"ownerCount"`
	Profile    CreatorProfileInfo `json:"profile"`
}

type CreatorOwnerCount struct {
	Follow      int64 `json:"follow"`
	Fan         int64 `json:"fan"`
	PhotoPublic int64 `json:"photo_public"`
}

type CreatorProfileInfo struct {
	UserName string `json:"user_name"`
	Gender   string `json:"gender"`
	HeadURL  string `json:"headurl"`
	UserText string `json:"user_text"`
}

type ProfilePhotoListGraphQLResponse struct {
	Errors []GraphQLError `json:"errors"`
	Data   struct {
		VisionProfilePhotoList ProfilePhotoListResult `json:"visionProfilePhotoList"`
	} `json:"data"`
}

type HomefeedGraphQLResponse struct {
	Errors []GraphQLError `json:"errors"`
	Data   struct {
		BrilliantTypeData HomefeedResult `json:"brilliantTypeData"`
	} `json:"data"`
}

type ProfilePhotoListResult struct {
	PCursor string `json:"pcursor"`
	Feeds   []Feed `json:"feeds"`
}

type HomefeedResult struct {
	PCursor string `json:"pcursor"`
	Feeds   []Feed `json:"feeds"`
}

type HotFeedV2Response struct {
	Result  int    `json:"result"`
	PCursor string `json:"pcursor"`
	Feeds   []Feed `json:"feeds"`
}

type SearchResult struct {
	Result          int    `json:"result"`
	Feeds           []Feed `json:"feeds"`
	SearchSessionID string `json:"searchSessionId"`
	PCursor         string `json:"pcursor"`
}

type Feed struct {
	Type   string `json:"type"`
	Author Author `json:"author"`
	Photo  Photo  `json:"photo"`
}

func (f *Feed) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type   json.RawMessage `json:"type"`
		Author Author          `json:"author"`
		Photo  Photo           `json:"photo"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	f.Type = flexibleKuaishouString(raw.Type)
	f.Author = raw.Author
	f.Photo = raw.Photo
	return nil
}

func flexibleKuaishouString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var integer int64
	if err := json.Unmarshal(raw, &integer); err == nil {
		return strconv.FormatInt(integer, 10)
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	return string(raw)
}

type Author struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	HeaderURL string `json:"headerUrl"`
}

type Photo struct {
	ID            string `json:"id"`
	Caption       string `json:"caption"`
	Timestamp     int64  `json:"timestamp"`
	RealLikeCount int64  `json:"realLikeCount"`
	ViewCount     int64  `json:"viewCount"`
	CoverURL      string `json:"coverUrl"`
	PhotoURL      string `json:"photoUrl"`
}

type CommentListV2Response struct {
	Result         int         `json:"result"`
	PCursorV2      string      `json:"pcursorV2"`
	RootCommentsV2 []CommentV2 `json:"rootCommentsV2"`
}

type SubCommentListV2Response struct {
	Result        int         `json:"result"`
	PCursorV2     string      `json:"pcursorV2"`
	SubCommentsV2 []CommentV2 `json:"subCommentsV2"`
}

type CommentV2 struct {
	CommentID      int64  `json:"comment_id"`
	Content        string `json:"content"`
	Timestamp      int64  `json:"timestamp"`
	CommentCount   int64  `json:"commentCount"`
	LikeCount      int64  `json:"likeCount"`
	AuthorID       string `json:"author_id"`
	AuthorName     string `json:"author_name"`
	HeadURL        string `json:"headurl"`
	HasSubComments bool   `json:"hasSubComments"`
}
