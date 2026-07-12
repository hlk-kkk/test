// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/zhihu/client.go
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

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models"
	zhihumodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/zhihu"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/account_pool"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/httpclient"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/rpc/signsrv"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
)

const defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"

var initialDataRegexp = regexp.MustCompile(`(?s)<script[^>]*id=["']js-initialData["'][^>]*>(.*?)</script>`)

type Client struct {
	accountWithIpPool *account_pool.AccountWithIpPool
	accountInfo       *models.AccountWithIpPoolModel
	signClient        *signsrv.Client
	logger            *tools.Logger
	config            *config.Config
	httpClient        *http.Client
	userAgent         string
}

func NewZhihuClient(accountWithIpPool *account_pool.AccountWithIpPool, signClient *signsrv.Client) *Client {
	cfg := config.GetConfig()
	cfg.ApplyDefaults()
	return &Client{
		accountWithIpPool: accountWithIpPool,
		signClient:        signClient,
		logger:            tools.GetLogger(),
		config:            cfg,
		httpClient:        httpclient.NewStandardClient(time.Duration(cfg.RequestTimeOut) * time.Second),
		userAgent:         defaultUserAgent,
	}
}

func (c *Client) Initialize() {
	if err := c.updateAccountInfo(); err != nil {
		c.logger.Error("[ZhihuClient.Initialize] update account info error: ", err)
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

func (c *Client) baseURL() string {
	baseURL := "https://www.zhihu.com"
	if c.config != nil && c.config.PlatformEndpoints != nil && c.config.PlatformEndpoints.ZhihuURL != "" {
		baseURL = c.config.PlatformEndpoints.ZhihuURL
	}
	return strings.TrimRight(baseURL, "/")
}

func (c *Client) apiURL(path string) string {
	return c.baseURL() + "/" + strings.TrimLeft(path, "/")
}

func (c *Client) cookies() string {
	if c.accountInfo != nil && c.accountInfo.Account != nil {
		return c.accountInfo.Account.Cookies
	}
	return ""
}

func (c *Client) headers(signURI string) map[string]string {
	headers := map[string]string{
		"accept":           "*/*",
		"accept-language":  "zh-CN,zh;q=0.9",
		"origin":           c.baseURL(),
		"priority":         "u=1, i",
		"referer":          c.baseURL() + "/search?q=python&time_interval=a_year&type=content",
		"user-agent":       c.userAgent,
		"x-api-version":    "3.0.91",
		"x-app-za":         "OS=Web",
		"x-requested-with": "fetch",
		"x-zse-93":         "101_3_3.0",
	}
	if cookies := c.cookies(); cookies != "" {
		headers["Cookie"] = cookies
	}
	if c.signClient != nil && signURI != "" {
		signResp, err := c.signClient.ZhihuSignServer(&signsrv.ZhihuSignRequest{
			Uri:     signURI,
			Cookies: c.cookies(),
		})
		if err == nil && signResp != nil && signResp.Result != nil {
			headers["x-zst-81"] = signResp.Result.XZst81
			headers["x-zse-96"] = signResp.Result.XZse96
		}
	}
	return headers
}

func (c *Client) getJSON(path string, params url.Values, target any) error {
	body, err := c.getBody(path, params)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func (c *Client) getBody(path string, params url.Values) ([]byte, error) {
	reqURL := c.apiURL(path)
	signURI := "/" + strings.TrimLeft(path, "/")
	if len(params) > 0 {
		encoded := params.Encode()
		reqURL += "?" + encoded
		signURI += "?" + encoded
	}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range c.headers(signURI) {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("zhihu request %s failed: %s %s", reqURL, resp.Status, string(body))
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) Pong() bool {
	var resp zhihumodel.MeResponse
	if err := c.getJSON("/api/v4/me", nil, &resp); err != nil {
		c.logger.Error("[ZhihuClient.Pong] me request failed: ", err)
		return false
	}
	return resp.UID != "" || resp.Name != ""
}

func (c *Client) GetNotesByKeyword(keyword string, page, pageSize int) ([]*zhihumodel.ContentObject, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize < 20 {
		pageSize = 20
	}
	params := url.Values{}
	params.Set("gk_version", "gz-gaokao")
	params.Set("t", "general")
	params.Set("q", keyword)
	params.Set("correction", "1")
	params.Set("offset", strconv.Itoa((page-1)*pageSize))
	params.Set("limit", strconv.Itoa(pageSize))
	params.Set("filter_fields", "")
	params.Set("lc_idx", strconv.Itoa((page-1)*pageSize))
	params.Set("show_all_topics", "0")
	params.Set("search_source", "Filter")
	params.Set("time_interval", "")
	params.Set("sort", "")
	params.Set("vertical", "")

	var resp zhihumodel.SearchResponse
	if err := c.getJSON("/api/v4/search_v3", params, &resp); err != nil {
		return nil, err
	}
	contents := make([]*zhihumodel.ContentObject, 0, len(resp.Data))
	for _, item := range resp.Data {
		if item.Type != "search_result" && item.Type != "zvideo" {
			continue
		}
		if item.Object == nil {
			continue
		}
		switch item.Object.Type {
		case "answer", "article", "zvideo":
			contents = append(contents, item.Object)
		}
	}
	return contents, nil
}

func (c *Client) GetAnswerInfo(questionID, answerID string) (*zhihumodel.ContentObject, error) {
	body, err := c.getBody(fmt.Sprintf("/question/%s/answer/%s", questionID, answerID), nil)
	if err != nil {
		return nil, err
	}
	return extractAnswerFromHTML(string(body))
}

func (c *Client) GetCreatorInfo(urlToken string) (*zhihumodel.CreatorInfo, error) {
	body, err := c.getBody(fmt.Sprintf("/people/%s", url.PathEscape(urlToken)), nil)
	if err != nil {
		return nil, err
	}
	return extractCreatorFromHTML(urlToken, string(body))
}

func (c *Client) GetCreatorAnswers(urlToken string, offset, limit int) (*zhihumodel.CreatorAnswersResponse, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{}
	params.Set("include", "data[*].is_normal,admin_closed_comment,reward_info,is_collapsed,annotation_action,annotation_detail,collapse_reason,collapsed_by,suggest_edit,comment_count,can_comment,content,editable_content,attachment,voteup_count,reshipment_settings,comment_permission,created_time,updated_time,review_info,excerpt,paid_info,reaction_instruction,is_labeled,label_info,relationship.is_authorized,voting,is_author,is_thanked,is_nothelp;data[*].vessay_info;data[*].author.badge[?(type=best_answerer)].topics;data[*].author.vip_info;data[*].question.has_publishing_draft,relationship")
	params.Set("offset", strconv.Itoa(offset))
	params.Set("limit", strconv.Itoa(limit))
	params.Set("order_by", "created")
	var resp zhihumodel.CreatorAnswersResponse
	if err := c.getJSON(fmt.Sprintf("/api/v4/members/%s/answers", url.PathEscape(urlToken)), params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetHomefeedNotes(pageNumber, afterID, endOffset int, sessionToken string) (*zhihumodel.HomefeedResponse, error) {
	if pageNumber <= 0 {
		pageNumber = 1
	}
	if afterID < 0 {
		afterID = 0
	}
	if endOffset < 0 {
		endOffset = 0
	}
	params := url.Values{}
	params.Set("action", "down")
	params.Set("desktop", "true")
	params.Set("after_id", strconv.Itoa(afterID))
	params.Set("end_offset", strconv.Itoa(endOffset))
	params.Set("page_number", strconv.Itoa(pageNumber))
	params.Set("session_token", sessionToken)
	var resp zhihumodel.HomefeedResponse
	if err := c.getJSON("/api/v3/feed/topstory/recommend", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetHotSearchItems(offset, limit int) (*zhihumodel.HotSearchResponse, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{}
	params.Set("domain", "0")
	params.Set("period", "hour")
	params.Set("offset", strconv.Itoa(offset))
	params.Set("limit", strconv.Itoa(limit))
	var resp zhihumodel.HotSearchResponse
	if err := c.getJSON("/api/v4/creators/rank/hot", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetQuestionFeeds(questionID, cursor, sessionID string, limit, offset int, order string) (*zhihumodel.QuestionFeedsResponse, error) {
	if limit <= 0 {
		limit = 5
	}
	if offset < 0 {
		offset = 0
	}
	if order == "" {
		order = "default"
	}
	params := url.Values{}
	params.Set("cursor", cursor)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))
	params.Set("order", order)
	params.Set("session_id", sessionID)
	params.Set("ws_qiangzhisafe", "1")
	params.Set("platform", "desktop")
	params.Set("include", "data[*].is_normal,admin_closed_comment,reward_info,is_collapsed,annotation_action,annotation_detail,collapse_reason,is_sticky,collapsed_by,suggest_edit,comment_count,can_comment,content,editable_content,attachment,voteup_count,reshipment_settings,comment_permission,created_time,updated_time,review_info,relevant_info,question,excerpt,is_labeled,paid_info,paid_info_content,reaction_instruction,relationship.is_authorized,is_author,voting,is_thanked,is_nothelp;data[*].author.follower_count,vip_info,kvip_info,badge[*].topics;data[*].settings.table_of_content.enabled")
	var resp zhihumodel.QuestionFeedsResponse
	if err := c.getJSON(fmt.Sprintf("/api/v4/questions/%s/feeds", url.PathEscape(questionID)), params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetRootComments(contentID, contentType, offset string, limit int) (*zhihumodel.CommentResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	params := url.Values{}
	params.Set("order", "score")
	params.Set("offset", offset)
	params.Set("limit", strconv.Itoa(limit))
	var resp zhihumodel.CommentResponse
	if err := c.getJSON(fmt.Sprintf("/api/v4/comment_v5/%ss/%s/root_comment", contentType, contentID), params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetChildComments(rootCommentID, offset string, limit int) (*zhihumodel.CommentResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	params := url.Values{}
	params.Set("order", "sort")
	params.Set("offset", offset)
	params.Set("limit", strconv.Itoa(limit))
	var resp zhihumodel.CommentResponse
	if err := c.getJSON(fmt.Sprintf("/api/v4/comment_v5/comment/%s/child_comment", rootCommentID), params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func extractAnswerFromHTML(body string) (*zhihumodel.ContentObject, error) {
	matches := initialDataRegexp.FindStringSubmatch(body)
	if len(matches) < 2 {
		return nil, nil
	}
	script := strings.TrimSpace(html.UnescapeString(matches[1]))
	if script == "" {
		return nil, nil
	}
	var initial struct {
		InitialState struct {
			Entities struct {
				Answers map[string]*zhihumodel.ContentObject `json:"answers"`
			} `json:"entities"`
		} `json:"initialState"`
	}
	if err := json.Unmarshal([]byte(script), &initial); err != nil {
		return nil, err
	}
	for _, answer := range initial.InitialState.Entities.Answers {
		return answer, nil
	}
	return nil, nil
}

func extractCreatorFromHTML(urlToken, body string) (*zhihumodel.CreatorInfo, error) {
	matches := initialDataRegexp.FindStringSubmatch(body)
	if len(matches) < 2 {
		return nil, nil
	}
	script := strings.TrimSpace(html.UnescapeString(matches[1]))
	if script == "" {
		return nil, nil
	}
	var initial struct {
		InitialState struct {
			Entities struct {
				Users map[string]*zhihumodel.CreatorInfo `json:"users"`
			} `json:"entities"`
		} `json:"initialState"`
	}
	if err := json.Unmarshal([]byte(script), &initial); err != nil {
		return nil, err
	}
	return initial.InitialState.Entities.Users[urlToken], nil
}

func (c *Client) EnsureReady() error {
	if c.accountInfo == nil || c.accountInfo.Account == nil {
		return errors.New("zhihu client has no account info")
	}
	return nil
}
