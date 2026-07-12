// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/weibo/client.go
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

package weibo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models"
	weibomodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/weibo"
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

func NewWeiboClient(accountWithIpPool *account_pool.AccountWithIpPool) *Client {
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
		c.logger.Error("[WeiboClient.Initialize] update account info error: ", err)
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
	baseURL := "https://m.weibo.cn"
	if c.config != nil && c.config.PlatformEndpoints != nil && c.config.PlatformEndpoints.WeiboApiURL != "" {
		baseURL = c.config.PlatformEndpoints.WeiboApiURL
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func (c *Client) hotBandURL() string {
	if c.config != nil && c.config.PlatformEndpoints != nil && c.config.PlatformEndpoints.WeiboHotBandURL != "" {
		return c.config.PlatformEndpoints.WeiboHotBandURL
	}
	return "https://weibo.com/ajax/statuses/hot_band"
}

func (c *Client) headers() map[string]string {
	headers := map[string]string{
		"Content-Type": "application/json;charset=UTF-8",
		"Accept":       "application/json, text/plain, */*",
		"origin":       "https://m.weibo.cn/",
		"referer":      "https://m.weibo.cn/",
		"user-agent":   c.userAgent,
	}
	if c.accountInfo != nil && c.accountInfo.Account != nil {
		headers["Cookie"] = c.accountInfo.Account.Cookies
	}
	return headers
}

func (c *Client) getURL(rawURL string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if params != nil {
		query := req.URL.Query()
		for key, value := range params {
			query.Set(key, value)
		}
		req.URL.RawQuery = query.Encode()
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

func (c *Client) get(path string, params map[string]string) ([]byte, error) {
	return c.getURL(c.apiURL(path), params)
}

func (c *Client) getData(path string, params map[string]string) (weibomodel.SearchData, error) {
	body, err := c.get(path, params)
	if err != nil {
		return weibomodel.SearchData{}, err
	}
	var resp weibomodel.APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return weibomodel.SearchData{}, err
	}
	if resp.OK != 0 && resp.OK != 1 {
		if resp.Msg == "" {
			resp.Msg = "unknown error"
		}
		return weibomodel.SearchData{}, fmt.Errorf("weibo API ok=%d: %s", resp.OK, resp.Msg)
	}
	return resp.Data, nil
}

func (c *Client) Pong() bool {
	body, err := c.get("/api/config", nil)
	if err != nil {
		c.logger.Error("[WeiboClient.Pong] config check failed: ", err)
		return false
	}
	var resp weibomodel.ConfigResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		c.logger.Error("[WeiboClient.Pong] decode config failed: ", err)
		return false
	}
	return resp.Data.Login
}

func (c *Client) SearchNotes(keyword string, page int) ([]weibomodel.Card, error) {
	if page <= 0 {
		page = 1
	}
	data, err := c.getData("/api/container/getIndex", map[string]string{
		"containerid": "100103type=1&q=" + keyword,
		"page_type":   "searchall",
		"page":        strconv.Itoa(page),
	})
	if err != nil {
		return nil, err
	}
	return data.Cards, nil
}

func (c *Client) GetHotSearchItems() ([]weibomodel.HotBandItem, error) {
	body, err := c.getURL(c.hotBandURL(), nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		OK   int             `json:"ok"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.OK != 1 {
		if resp.Msg == "" {
			resp.Msg = "get weibo hot band failed"
		}
		return nil, fmt.Errorf("weibo hot band API ok=%d: %s", resp.OK, resp.Msg)
	}
	var items []weibomodel.HotBandItem
	if err := json.Unmarshal(resp.Data, &items); err == nil {
		return filterHotBandItems(items), nil
	}
	var wrapped struct {
		BandList []weibomodel.HotBandItem `json:"band_list"`
	}
	if err := json.Unmarshal(resp.Data, &wrapped); err != nil {
		return nil, err
	}
	return filterHotBandItems(wrapped.BandList), nil
}

func (c *Client) GetCreatorInfoByID(creatorID string) (*weibomodel.SearchData, error) {
	data, err := c.getData("/api/container/getIndex", map[string]string{
		"jumpfrom":    "weibocom",
		"type":        "uid",
		"value":       creatorID,
		"containerid": "100505" + creatorID,
	})
	if err != nil {
		return nil, err
	}
	if anyToString(data.UserInfo.ID) == "" {
		return nil, nil
	}
	return &data, nil
}

func (c *Client) GetNotesByCreator(creatorID, containerID, sinceID string) (*weibomodel.SearchData, error) {
	if containerID == "" {
		containerID = "107603" + creatorID
	}
	if sinceID == "" {
		sinceID = "0"
	}
	data, err := c.getData("/api/container/getIndex", map[string]string{
		"jumpfrom":    "weibocom",
		"type":        "uid",
		"value":       creatorID,
		"containerid": containerID,
		"since_id":    sinceID,
	})
	if err != nil {
		return nil, err
	}
	return &data, nil
}

var renderDataPattern = regexp.MustCompile(`(?s)var \$render_data = (\[.*?\])\[0\]`)

func (c *Client) GetNoteInfoByID(noteID string) (*weibomodel.Mblog, error) {
	body, err := c.get("/detail/"+noteID, nil)
	if err != nil {
		return nil, err
	}
	matches := renderDataPattern.FindSubmatch(body)
	if len(matches) < 2 {
		return nil, nil
	}
	var payload []struct {
		Status weibomodel.Mblog `json:"status"`
	}
	if err := json.Unmarshal(matches[1], &payload); err != nil {
		return nil, err
	}
	if len(payload) == 0 || anyToString(payload[0].Status.ID) == "" {
		return nil, nil
	}
	return &payload[0].Status, nil
}

func (c *Client) GetNoteComments(noteID string, maxID int64, maxIDType int) (*weibomodel.CommentPage, error) {
	params := map[string]string{
		"id":          noteID,
		"mid":         noteID,
		"max_id_type": strconv.Itoa(maxIDType),
	}
	if maxID > 0 {
		params["max_id"] = strconv.FormatInt(maxID, 10)
	}
	body, err := c.get("/comments/hotflow", params)
	if err != nil {
		return nil, err
	}
	var resp weibomodel.CommentAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.OK != 0 && resp.OK != 1 {
		if resp.Msg == "" {
			resp.Msg = "unknown error"
		}
		return nil, fmt.Errorf("weibo comments API ok=%d: %s", resp.OK, resp.Msg)
	}
	return &resp.Data, nil
}

func filterHotBandItems(items []weibomodel.HotBandItem) []weibomodel.HotBandItem {
	filtered := make([]weibomodel.HotBandItem, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Word) == "" || item.RealPos == 0 || item.IsAd {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func anyToString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return fmt.Sprintf("%.0f", typed)
	default:
		return fmt.Sprint(typed)
	}
}
