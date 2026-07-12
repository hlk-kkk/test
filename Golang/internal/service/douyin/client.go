// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/douyin/client.go
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

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models"
	douyinmodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/douyin"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/account_pool"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/httpclient"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/rpc/signsrv"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
)

const defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"

type Client struct {
	accountWithIpPool *account_pool.AccountWithIpPool
	accountInfo       *models.AccountWithIpPoolModel
	logger            *tools.Logger
	config            *config.Config
	signSrvClient     *signsrv.Client
	httpClient        *http.Client
	userAgent         string
}

func NewDouyinClient(accountWithIpPool *account_pool.AccountWithIpPool, signSrvClient *signsrv.Client) *Client {
	cfg := config.GetConfig()
	cfg.ApplyDefaults()
	return &Client{
		accountWithIpPool: accountWithIpPool,
		logger:            tools.GetLogger(),
		config:            cfg,
		signSrvClient:     signSrvClient,
		httpClient:        httpclient.NewStandardClient(time.Duration(cfg.RequestTimeOut) * time.Second),
		userAgent:         defaultUserAgent,
	}
}

func (c *Client) Initialize() {
	if err := c.updateAccountInfo(); err != nil {
		c.logger.Error("[DouyinClient.Initialize] update account info error: ", err)
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

func (c *Client) apiURL(path string) string {
	baseURL := "https://www.douyin.com"
	if c.config != nil && c.config.PlatformEndpoints != nil && c.config.PlatformEndpoints.DouyinApiURL != "" {
		baseURL = c.config.PlatformEndpoints.DouyinApiURL
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func (c *Client) headers() map[string]string {
	headers := map[string]string{
		"Accept":          "application/json",
		"Content-Type":    "application/json",
		"accept-language": "zh-CN,zh;q=0.9",
		"origin":          "https://www.douyin.com",
		"referer":         "https://www.douyin.com/user/self",
		"user-agent":      c.userAgent,
	}
	if c.accountInfo != nil && c.accountInfo.Account != nil {
		headers["Cookie"] = c.accountInfo.Account.Cookies
	}
	return headers
}

func (c *Client) cookies() string {
	if c.accountInfo != nil && c.accountInfo.Account != nil {
		return c.accountInfo.Account.Cookies
	}
	return ""
}

func (c *Client) get(path string, params map[string]string) ([]byte, error) {
	preparedParams, err := c.prepareURLParams(path, params)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, c.apiURL(path), nil)
	if err != nil {
		return nil, err
	}
	query := req.URL.Query()
	for key, value := range preparedParams {
		query.Set(key, value)
	}
	req.URL.RawQuery = query.Encode()
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

func (c *Client) post(path string, params map[string]string) ([]byte, error) {
	preparedParams, err := c.prepareURLParams(path, params)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.apiURL(path), nil)
	if err != nil {
		return nil, err
	}
	query := req.URL.Query()
	for key, value := range preparedParams {
		query.Set(key, value)
	}
	req.URL.RawQuery = query.Encode()
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

func (c *Client) prepareURLParams(path string, params map[string]string) (map[string]string, error) {
	finalParams := map[string]string{}
	for key, value := range params {
		finalParams[key] = value
	}
	for key, value := range c.commonParams() {
		finalParams[key] = value
	}
	for key, value := range c.verifyParams() {
		finalParams[key] = value
	}

	query := url.Values{}
	for key, value := range finalParams {
		query.Set(key, value)
	}
	signResp, err := c.signSrvClient.DouyinSignServer(&signsrv.DouyinSignRequest{
		Uri:         path,
		QueryParams: query.Encode(),
		UserAgent:   c.userAgent,
		Cookies:     c.cookies(),
	})
	if err != nil {
		return nil, err
	}
	if !strings.Contains(path, "/v1/web/general/search/single/") && signResp.Result != nil && signResp.Result.ABogus != "" {
		finalParams["a_bogus"] = signResp.Result.ABogus
	}
	return finalParams, nil
}

func (c *Client) commonParams() map[string]string {
	return map[string]string{
		"device_platform":             "webapp",
		"aid":                         "6383",
		"channel":                     "channel_pc_web",
		"publish_video_strategy_type": "2",
		"update_version_code":         "170400",
		"pc_client_type":              "1",
		"version_code":                "170400",
		"version_name":                "17.4.0",
		"cookie_enabled":              "true",
		"screen_width":                "2560",
		"screen_height":               "1440",
		"browser_language":            "zh-CN",
		"browser_platform":            "MacIntel",
		"browser_name":                "Chrome",
		"browser_version":             "135.0.0.0",
		"browser_online":              "true",
		"engine_name":                 "Blink",
		"engine_version":              "135.0.0.0",
		"os_name":                     "Mac+OS",
		"os_version":                  "10.15.7",
		"cpu_core_num":                "8",
		"device_memory":               "8",
		"platform":                    "PC",
		"downlink":                    "4.45",
		"effective_type":              "4g",
		"round_trip_time":             "100",
	}
}

func (c *Client) verifyParams() map[string]string {
	return map[string]string{
		"webid":   fallbackCookieValue(c.cookies(), "webid", randomDigits(19)),
		"msToken": fallbackCookieValue(c.cookies(), "msToken", randomToken(128)),
	}
}

func fallbackCookieValue(cookieHeader, key, fallback string) string {
	for _, part := range strings.Split(cookieHeader, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && name == key && value != "" {
			return value
		}
	}
	return fallback
}

func randomDigits(length int) string {
	var builder strings.Builder
	for builder.Len() < length {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return strconv.FormatInt(time.Now().UnixNano(), 10)[:length]
		}
		builder.WriteByte(byte('0' + n.Int64()))
	}
	return builder.String()
}

func randomToken(length int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var builder strings.Builder
	for builder.Len() < length {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return strings.Repeat("a", length)
		}
		builder.WriteByte(alphabet[n.Int64()])
	}
	return builder.String()
}

func (c *Client) Pong() bool {
	if !c.signSrvClient.PongSignServer() {
		c.logger.Error("[DouyinClient.Pong] sign server pong failed")
		return false
	}
	body, err := c.get("/aweme/v1/web/history/read/", map[string]string{
		"max_cursor": "0",
		"count":      "20",
	})
	if err != nil {
		c.logger.Error("[DouyinClient.Pong] history read failed: ", err)
		return false
	}
	var resp douyinmodel.HistoryReadResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		c.logger.Error("[DouyinClient.Pong] decode history read failed: ", err)
		return false
	}
	return resp.StatusCode == 0
}

func (c *Client) SearchInfoByKeyword(keyword string, offset int, searchID string) (*douyinmodel.SearchResponse, error) {
	params := map[string]string{
		"search_channel":       "aweme_general",
		"enable_history":       "1",
		"keyword":              keyword,
		"search_source":        "tab_search",
		"query_correct_type":   "1",
		"is_filter_search":     "0",
		"from_group_id":        "7378810571505847586",
		"offset":               strconv.Itoa(offset),
		"count":                "10",
		"need_filter_settings": "1",
		"list_type":            "multi",
		"search_id":            searchID,
	}

	body, err := c.get("/aweme/v1/web/general/search/single/", params)
	if err != nil {
		return nil, err
	}
	var resp douyinmodel.SearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetVideoByID(awemeID string) (*douyinmodel.AwemeInfo, error) {
	body, err := c.get("/aweme/v1/web/aweme/detail/", map[string]string{
		"aweme_id": awemeID,
	})
	if err != nil {
		return nil, err
	}
	var resp douyinmodel.AwemeDetailResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.AwemeDetail, nil
}

func (c *Client) GetAwemeComments(awemeID string, cursor int64) (*douyinmodel.CommentResponse, error) {
	body, err := c.get("/aweme/v1/web/comment/list/", map[string]string{
		"aweme_id":  awemeID,
		"cursor":    strconv.FormatInt(cursor, 10),
		"count":     "20",
		"item_type": "0",
	})
	if err != nil {
		return nil, err
	}
	var resp douyinmodel.CommentResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetSubComments(commentID string, cursor int64) (*douyinmodel.CommentResponse, error) {
	body, err := c.get("/aweme/v1/web/comment/list/reply/", map[string]string{
		"comment_id": commentID,
		"cursor":     strconv.FormatInt(cursor, 10),
		"count":      "20",
		"item_type":  "0",
	})
	if err != nil {
		return nil, err
	}
	var resp douyinmodel.CommentResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetHotSearchItems() ([]douyinmodel.HotSearchItem, error) {
	body, err := c.get("/aweme/v1/web/hot/search/list/", nil)
	if err != nil {
		return nil, err
	}
	var resp douyinmodel.HotSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	items := make([]douyinmodel.HotSearchItem, 0, len(resp.Data.WordList))
	for _, item := range resp.Data.WordList {
		if item.Word == "" || item.Position == 0 || item.GroupID == "" {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (c *Client) GetHotSearchAwemeIDs(hotword string, offset, count int) ([]string, error) {
	body, err := c.get("/aweme/v1/web/hot/search/video/list/", map[string]string{
		"hotword": hotword,
		"offset":  strconv.Itoa(offset),
		"count":   strconv.Itoa(count),
	})
	if err != nil {
		return nil, err
	}
	var resp douyinmodel.HotSearchAwemeListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(resp.AwemeList))
	for _, item := range resp.AwemeList {
		if item.AwemeID != "" {
			ids = append(ids, item.AwemeID)
		}
	}
	return ids, nil
}

func (c *Client) GetUserInfo(secUserID string) (*douyinmodel.CreatorInfo, error) {
	body, err := c.get("/aweme/v1/web/user/profile/other/", map[string]string{
		"sec_user_id":                 secUserID,
		"publish_video_strategy_type": "2",
		"personal_center_strategy":    "1",
	})
	if err != nil {
		return nil, err
	}
	var resp douyinmodel.UserProfileResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.User.UID == "" {
		return nil, nil
	}
	return &resp.User, nil
}

func (c *Client) GetUserAwemePosts(secUserID, maxCursor string) (*douyinmodel.UserAwemePostsResponse, error) {
	if maxCursor == "" {
		maxCursor = "0"
	}
	body, err := c.get("/aweme/v1/web/aweme/post/", map[string]string{
		"sec_user_id":                 secUserID,
		"count":                       "18",
		"max_cursor":                  maxCursor,
		"locate_query":                "false",
		"publish_video_strategy_type": "2",
	})
	if err != nil {
		return nil, err
	}
	var resp douyinmodel.UserAwemePostsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetHomefeedAwemeList(refreshIndex, count int) (*douyinmodel.HomefeedResponse, error) {
	body, err := c.post("/aweme/v1/web/module/feed/", map[string]string{
		"device_platform": "webapp",
		"aid":             "6383",
		"channel":         "channel_pc_web",
		"module_id":       "3003101",
		"count":           strconv.Itoa(count),
		"refresh_index":   strconv.Itoa(refreshIndex),
		"tag_id":          "0",
	})
	if err != nil {
		return nil, err
	}
	var resp douyinmodel.HomefeedResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
