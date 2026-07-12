// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/kuaishou/client.go
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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models"
	kuaishoumodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/kuaishou"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/account_pool"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/httpclient"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
)

const defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"

type Client struct {
	accountWithIpPool *account_pool.AccountWithIpPool
	accountInfo       *models.AccountWithIpPoolModel
	logger            *tools.Logger
	config            *config.Config
	httpClient        *http.Client
	userAgent         string
}

func NewKuaishouClient(accountWithIpPool *account_pool.AccountWithIpPool) *Client {
	cfg := config.GetConfig()
	cfg.ApplyDefaults()
	return &Client{
		accountWithIpPool: accountWithIpPool,
		logger:            tools.GetLogger(),
		config:            cfg,
		httpClient:        httpclient.NewStandardClient(time.Duration(cfg.RequestTimeOut) * time.Second),
		userAgent:         defaultUserAgent,
	}
}

func (c *Client) Initialize() {
	if err := c.updateAccountInfo(); err != nil {
		c.logger.Error("[KuaishouClient.Initialize] update account info error: ", err)
		panic(err)
	}
}

func (c *Client) updateAccountInfo() error {
	accountInfo, err := c.accountWithIpPool.GetAccountWithIpInfo()
	if err != nil {
		return err
	}
	c.accountInfo = accountInfo
	return nil
}

func (c *Client) graphQLURL() string {
	if c.config != nil && c.config.PlatformEndpoints != nil && c.config.PlatformEndpoints.KuaishouApiURL != "" {
		return c.config.PlatformEndpoints.KuaishouApiURL
	}
	return "https://www.kuaishou.com/graphql"
}

func (c *Client) restURL(path string) string {
	baseURL := "https://www.kuaishou.com"
	if c.config != nil && c.config.PlatformEndpoints != nil && c.config.PlatformEndpoints.KuaishouRestURL != "" {
		baseURL = c.config.PlatformEndpoints.KuaishouRestURL
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func (c *Client) headers() map[string]string {
	headers := map[string]string{
		"Accept":          "application/json",
		"Content-Type":    "application/json",
		"accept-language": "zh-CN,zh;q=0.9",
		"origin":          "https://www.kuaishou.com",
		"referer":         c.restURL("/"),
		"user-agent":      c.userAgent,
	}
	if c.accountInfo != nil && c.accountInfo.Account != nil {
		headers["Cookie"] = c.accountInfo.Account.Cookies
	}
	return headers
}

func (c *Client) graphQL(operationName string, variables map[string]any) ([]byte, error) {
	if variables == nil {
		variables = map[string]any{}
	}
	payload := map[string]any{
		"operationName": operationName,
		"variables":     variables,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.graphQLURL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for key, value := range c.headers() {
		req.Header.Set(key, value)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) postV2(path string, payload map[string]any) ([]byte, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.restURL(path), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for key, value := range c.headers() {
		req.Header.Set(key, value)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(resp.Status)
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result struct {
		Result int `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Result != 1 {
		return nil, fmt.Errorf("kuaishou v2 API result=%d", result.Result)
	}
	return body, nil
}

func (c *Client) Pong() bool {
	body, err := c.graphQL("visionProfileUserList", map[string]any{})
	if err != nil {
		c.logger.Error("[KuaishouClient.Pong] profile user list failed: ", err)
		return false
	}
	var resp kuaishoumodel.ProfileUserListGraphQLResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		c.logger.Error("[KuaishouClient.Pong] decode profile user list failed: ", err)
		return false
	}
	if err := graphQLError(resp.Errors); err != nil {
		c.logger.Error("[KuaishouClient.Pong] graphql error: ", err)
		return false
	}
	return resp.Data.VisionProfileUserList.Result == 1
}

func (c *Client) SearchInfoByKeyword(keyword, pcursor, searchSessionID string) (*kuaishoumodel.SearchResult, error) {
	if pcursor == "" {
		pcursor = "1"
	}
	body, err := c.graphQL("visionSearchPhoto", map[string]any{
		"keyword":         keyword,
		"pcursor":         pcursor,
		"searchSessionId": searchSessionID,
	})
	if err != nil {
		return nil, err
	}
	var resp kuaishoumodel.SearchGraphQLResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if err := graphQLError(resp.Errors); err != nil {
		return nil, err
	}
	return &resp.Data.VisionSearchPhoto, nil
}

func (c *Client) GetVideoInfo(photoID string) (*kuaishoumodel.Feed, error) {
	body, err := c.graphQL("visionVideoDetail", map[string]any{
		"photoId": photoID,
		"page":    "search",
	})
	if err != nil {
		return nil, err
	}
	var resp kuaishoumodel.DetailGraphQLResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if err := graphQLError(resp.Errors); err != nil {
		return nil, err
	}
	if resp.Data.VisionVideoDetail.Photo.ID == "" {
		return nil, nil
	}
	return &resp.Data.VisionVideoDetail, nil
}

func (c *Client) GetCreatorInfo(userID string) (*kuaishoumodel.CreatorProfile, error) {
	body, err := c.graphQL("visionProfile", map[string]any{
		"userId": userID,
	})
	if err != nil {
		return nil, err
	}
	var resp kuaishoumodel.ProfileGraphQLResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if err := graphQLError(resp.Errors); err != nil {
		return nil, err
	}
	if resp.Data.VisionProfile.UserProfile.Profile.UserName == "" {
		return nil, nil
	}
	return &resp.Data.VisionProfile.UserProfile, nil
}

func (c *Client) GetVideoByCreator(userID, pcursor string) (*kuaishoumodel.ProfilePhotoListResult, error) {
	body, err := c.graphQL("visionProfilePhotoList", map[string]any{
		"page":    "profile",
		"pcursor": pcursor,
		"userId":  userID,
	})
	if err != nil {
		return nil, err
	}
	var resp kuaishoumodel.ProfilePhotoListGraphQLResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if err := graphQLError(resp.Errors); err != nil {
		return nil, err
	}
	return &resp.Data.VisionProfilePhotoList, nil
}

func (c *Client) GetHomefeedVideos(pcursor string) (*kuaishoumodel.HomefeedResult, error) {
	body, err := c.graphQL("brilliantTypeDataQuery", map[string]any{
		"pcursor":      pcursor,
		"hotChannelId": "00",
		"page":         "brilliant",
	})
	if err != nil {
		return nil, err
	}
	var resp kuaishoumodel.HomefeedGraphQLResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if err := graphQLError(resp.Errors); err != nil {
		return nil, err
	}
	return &resp.Data.BrilliantTypeData, nil
}

func (c *Client) GetHotSearchItems(pcursor string) (*kuaishoumodel.HotFeedV2Response, error) {
	payload := map[string]any{}
	if pcursor != "" {
		payload["pcursor"] = pcursor
	}
	body, err := c.postV2("/rest/v/feed/hot", payload)
	if err != nil {
		return nil, err
	}
	var resp kuaishoumodel.HotFeedV2Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetVideoCommentsV2(photoID, pcursor string) (*kuaishoumodel.CommentListV2Response, error) {
	body, err := c.postV2("/rest/v/photo/comment/list", map[string]any{
		"photoId": photoID,
		"pcursor": pcursor,
	})
	if err != nil {
		return nil, err
	}
	var resp kuaishoumodel.CommentListV2Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetVideoSubCommentsV2(photoID, rootCommentID, pcursor string) (*kuaishoumodel.SubCommentListV2Response, error) {
	var rootComment any = rootCommentID
	if value, err := strconv.ParseInt(rootCommentID, 10, 64); err == nil {
		rootComment = value
	}
	body, err := c.postV2("/rest/v/photo/comment/sublist", map[string]any{
		"photoId":       photoID,
		"pcursor":       pcursor,
		"rootCommentId": rootComment,
	})
	if err != nil {
		return nil, err
	}
	var resp kuaishoumodel.SubCommentListV2Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func graphQLError(errList []kuaishoumodel.GraphQLError) error {
	if len(errList) == 0 {
		return nil
	}
	if errList[0].Message == "" {
		return fmt.Errorf("graphql returned %d errors", len(errList))
	}
	return errors.New(errList[0].Message)
}
