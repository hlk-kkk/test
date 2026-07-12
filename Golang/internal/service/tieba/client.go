// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/tieba/client.go
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

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models"
	tiebamodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/tieba"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/account_pool"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/httpclient"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
)

const (
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
	pcSignSecret     = "36770b1f34c9bbf2e7d1a99d2b82fa9e"
)

type Client struct {
	accountWithIpPool *account_pool.AccountWithIpPool
	accountInfo       *models.AccountWithIpPoolModel
	logger            *tools.Logger
	config            *config.Config
	httpClient        *http.Client
	userAgent         string
	pcTBS             string
}

func NewTiebaClient(accountWithIpPool *account_pool.AccountWithIpPool) *Client {
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
		c.logger.Error("[TiebaClient.Initialize] update account info error: ", err)
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
	baseURL := "https://tieba.baidu.com"
	if c.config != nil && c.config.PlatformEndpoints != nil && c.config.PlatformEndpoints.TiebaURL != "" {
		baseURL = c.config.PlatformEndpoints.TiebaURL
	}
	return strings.TrimRight(baseURL, "/")
}

func (c *Client) apiURL(path string) string {
	return c.baseURL() + "/" + strings.TrimLeft(path, "/")
}

func (c *Client) headers() map[string]string {
	headers := map[string]string{
		"Accept":     "application/json, text/html, */*",
		"origin":     c.baseURL(),
		"referer":    c.baseURL(),
		"user-agent": c.userAgent,
	}
	if c.accountInfo != nil && c.accountInfo.Account != nil {
		headers["Cookie"] = c.accountInfo.Account.Cookies
	}
	return headers
}

func (c *Client) fetchPCJSON(path, method string, params, data map[string]string, useSign bool) ([]byte, error) {
	if method == "" {
		method = http.MethodGet
	}
	if params == nil {
		params = map[string]string{}
	}
	if data == nil {
		data = map[string]string{}
	}
	if useSign {
		signSource := params
		if strings.EqualFold(method, http.MethodPost) {
			signSource = data
		}
		if signSource["subapp_type"] == "" {
			signSource["subapp_type"] = "pc"
		}
		if signSource["_client_type"] == "" {
			signSource["_client_type"] = "20"
		}
		signSource["sign"] = signPCParams(signSource)
	}

	reqURL := c.apiURL(path)
	if len(params) > 0 {
		values := url.Values{}
		for key, value := range params {
			if value != "" {
				values.Set(key, value)
			}
		}
		if encoded := values.Encode(); encoded != "" {
			reqURL += "?" + encoded
		}
	}

	var body io.Reader
	if strings.EqualFold(method, http.MethodPost) {
		values := url.Values{}
		for key, value := range data {
			if value != "" {
				values.Set(key, value)
			}
		}
		body = strings.NewReader(values.Encode())
	}

	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, err
	}
	for key, value := range c.headers() {
		req.Header.Set(key, value)
	}
	if strings.EqualFold(method, http.MethodPost) {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
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

func (c *Client) Pong() bool {
	body, err := c.fetchPCJSON("/c/s/pc/sync", http.MethodGet, map[string]string{
		"subapp_type":  "pc",
		"_client_type": "20",
	}, nil, true)
	if err != nil {
		c.logger.Error("[TiebaClient.Pong] sync failed: ", err)
		return false
	}
	var resp tiebamodel.SyncResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		c.logger.Error("[TiebaClient.Pong] decode sync failed: ", err)
		return false
	}
	return resp.Data.User.IsLogin
}

func (c *Client) GetNotesByKeyword(keyword string, page, pageSize int) ([]string, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize < 20 {
		pageSize = 20
	}
	body, err := c.fetchPCJSON("/mo/q/search/multsearch", http.MethodGet, map[string]string{
		"rn":           strconv.Itoa(pageSize),
		"st":           "1",
		"word":         keyword,
		"needbrand":    "1",
		"sug_type":     "2",
		"pn":           strconv.Itoa(page),
		"come_from":    "search",
		"subapp_type":  "pc",
		"_client_type": "20",
	}, nil, true)
	if err != nil {
		return nil, err
	}
	var resp tiebamodel.SearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.ErrorCode != 0 || resp.No != 0 {
		return nil, fmt.Errorf("tieba search API error_code=%d no=%d", resp.ErrorCode, resp.No)
	}
	noteIDs := make([]string, 0, len(resp.Data.CardList))
	for _, card := range resp.Data.CardList {
		if card.CardInfo != "thread" && card.CardStyle != "thread" {
			continue
		}
		noteID := anyToString(card.Data.TID)
		if noteID != "" {
			noteIDs = append(noteIDs, noteID)
		}
	}
	return noteIDs, nil
}

func (c *Client) GetNoteByID(noteID string) (*tiebamodel.PageResponse, error) {
	return c.GetNoteComments(noteID, 1)
}

func (c *Client) GetNoteComments(noteID string, page int) (*tiebamodel.PageResponse, error) {
	if page <= 0 {
		page = 1
	}
	tbs, err := c.getPCTBS()
	if err != nil {
		return nil, err
	}
	body, err := c.fetchPCJSON("/c/f/pb/page_pc", http.MethodPost, nil, map[string]string{
		"pn":                    strconv.Itoa(page),
		"lz":                    "0",
		"r":                     "2",
		"mark_type":             "0",
		"back":                  "0",
		"fr":                    "",
		"kz":                    noteID,
		"session_request_times": "1",
		"tbs":                   tbs,
		"subapp_type":           "pc",
		"_client_type":          "20",
	}, true)
	if err != nil {
		return nil, err
	}
	var resp tiebamodel.PageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.ErrorCode != 0 || resp.No != 0 {
		return nil, fmt.Errorf("tieba detail API error_code=%d no=%d", resp.ErrorCode, resp.No)
	}
	if anyToString(resp.Thread.ID) == "" && anyToString(resp.Thread.TID) == "" && anyToString(resp.FirstFloor.TID) == "" {
		return nil, nil
	}
	return &resp, nil
}

func (c *Client) GetNoteSubComments(noteID, parentCommentID, tiebaID string, page int) (string, error) {
	if page <= 0 {
		page = 1
	}
	body, err := c.fetchPCJSON("/p/comment", http.MethodGet, map[string]string{
		"tid": noteID,
		"pid": parentCommentID,
		"fid": tiebaID,
		"pn":  strconv.Itoa(page),
	}, nil, false)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *Client) GetCreatorInfoByURL(creatorURL string) (*tiebamodel.CreatorUser, error) {
	portrait := ExtractCreatorPortrait(creatorURL)
	if portrait == "" {
		return nil, fmt.Errorf("can not extract tieba creator portrait from %q", creatorURL)
	}
	body, err := c.fetchPCJSON("/c/u/pc/homeSidebarRight", http.MethodGet, map[string]string{
		"portrait":     portrait,
		"un":           "",
		"subapp_type":  "pc",
		"_client_type": "20",
	}, nil, true)
	if err != nil {
		return nil, err
	}
	var resp tiebamodel.CreatorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if resp.ErrorCode != 0 || resp.No != 0 {
		return nil, fmt.Errorf("tieba creator API error_code=%d no=%d", resp.ErrorCode, resp.No)
	}
	if anyToString(resp.Data.User.ID) == "" {
		return nil, nil
	}
	return &resp.Data.User, nil
}

func (c *Client) GetNotesByCreatorPortrait(portrait string, page, pageSize int) ([]string, bool, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	body, err := c.fetchPCJSON("/c/u/feed/myThread", http.MethodGet, map[string]string{
		"pn":           strconv.Itoa(page),
		"rn":           strconv.Itoa(pageSize),
		"portrait":     portrait,
		"type":         "1",
		"un":           "",
		"subapp_type":  "pc",
		"_client_type": "20",
	}, nil, true)
	if err != nil {
		return nil, false, err
	}
	var resp tiebamodel.CreatorThreadsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, false, err
	}
	if resp.ErrorCode != 0 || resp.No != 0 {
		return nil, false, fmt.Errorf("tieba creator threads API error_code=%d no=%d", resp.ErrorCode, resp.No)
	}
	noteIDs := make([]string, 0, len(resp.Data.List))
	for _, item := range resp.Data.List {
		noteID := anyToString(item.ThreadInfo.TID)
		if noteID == "" {
			noteID = anyToString(item.ThreadInfo.ID)
		}
		if noteID != "" {
			noteIDs = append(noteIDs, noteID)
		}
	}
	return noteIDs, anyToInt(resp.Data.HasMore) == 1, nil
}

func ExtractCreatorPortrait(creatorURL string) string {
	creatorURL = strings.TrimSpace(creatorURL)
	if creatorURL == "" {
		return ""
	}
	if !strings.HasPrefix(creatorURL, "http://") && !strings.HasPrefix(creatorURL, "https://") {
		if before, _, ok := strings.Cut(creatorURL, "?"); ok {
			return before
		}
		return creatorURL
	}
	parsed, err := url.Parse(creatorURL)
	if err != nil {
		return ""
	}
	query := parsed.Query()
	for _, key := range []string{"id", "portrait", "un"} {
		if value := query.Get(key); value != "" {
			if before, _, ok := strings.Cut(value, "?"); ok {
				return before
			}
			return value
		}
	}
	return ""
}

func (c *Client) getPCTBS() (string, error) {
	if c.pcTBS != "" {
		return c.pcTBS, nil
	}
	body, err := c.fetchPCJSON("/c/s/pc/sync", http.MethodGet, map[string]string{
		"subapp_type":  "pc",
		"_client_type": "20",
	}, nil, true)
	if err != nil {
		return "", err
	}
	var resp tiebamodel.SyncResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if resp.Data.Anti.TBS == "" {
		return "", fmt.Errorf("can not get tieba tbs from pc sync API")
	}
	c.pcTBS = resp.Data.Anti.TBS
	return c.pcTBS, nil
}

func signPCParams(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		if key == "sign" || key == "sig" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(params[key])
	}
	builder.WriteString(pcSignSecret)
	sum := md5.Sum([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
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
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return fmt.Sprint(typed)
	}
}

func anyToInt(value any) int {
	if value == nil {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, _ := strconv.Atoi(typed)
		return parsed
	default:
		return 0
	}
}
