// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/bilibili/client.go
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

package bilibili

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	config "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models"
	bilimodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/bilibili"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/account_pool"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/httpclient"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/rpc/signsrv"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
)

const (
	ApiUrl   = "https://api.bilibili.com"
	IndexUrl = "https://www.bilibili.com"
	SpaceUrl = "https://space.bilibili.com"
)

type Client struct {
	clientConfig      *httpclient.ClientConfig
	accountWithIpPool *account_pool.AccountWithIpPool
	accountInfo       *models.AccountWithIpPoolModel
	logger            *tools.Logger
	config            *config.Config
	signSrvClient     *signsrv.Client
}

func NewBilibiliClient(accountWithIpPool *account_pool.AccountWithIpPool, signSrvClient *signsrv.Client) *Client {
	globalConfig := config.GetConfig()
	clientConfig := &httpclient.ClientConfig{
		TimeOut: time.Duration(globalConfig.RequestTimeOut) * time.Second,
		Headers: make(map[string]string),
	}
	client := &Client{
		logger:            tools.GetLogger(),
		config:            globalConfig,
		clientConfig:      clientConfig,
		accountWithIpPool: accountWithIpPool,
		signSrvClient:     signSrvClient,
	}
	client.addRequestHeaders()
	return client
}

func (c *Client) apiURL(path string) string {
	baseURL := ApiUrl
	if c != nil && c.config != nil && c.config.PlatformEndpoints != nil && c.config.PlatformEndpoints.BilibiliApiURL != "" {
		baseURL = c.config.PlatformEndpoints.BilibiliApiURL
	}
	return joinBaseURL(baseURL, path)
}

func (c *Client) indexURL(path string) string {
	baseURL := IndexUrl
	if c != nil && c.config != nil && c.config.PlatformEndpoints != nil && c.config.PlatformEndpoints.BilibiliIndexURL != "" {
		baseURL = c.config.PlatformEndpoints.BilibiliIndexURL
	}
	return joinBaseURL(baseURL, path)
}

func (c *Client) spaceURL(path string) string {
	baseURL := SpaceUrl
	if c != nil && c.config != nil && c.config.PlatformEndpoints != nil && c.config.PlatformEndpoints.BilibiliSpaceURL != "" {
		baseURL = c.config.PlatformEndpoints.BilibiliSpaceURL
	}
	return joinBaseURL(baseURL, path)
}

func joinBaseURL(baseURL, path string) string {
	if path == "" {
		return strings.TrimRight(baseURL, "/")
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func (c *Client) Initialize() {
	err := c.updateAccountInfo()
	if err != nil {
		c.logger.Error("[Client.Initialize] update account info error: ", err)
		panic(err)
	}
}

func (c *Client) addRequestHeaders() {
	c.clientConfig.SetHeader("referer", "https://www.bilibili.com/")
	c.clientConfig.SetHeader("origin", "https://www.bilibili.com")
	c.clientConfig.SetHeader("Accept-Language", "zh-CN,zh;q=0.9")
	c.clientConfig.SetHeader("Accept", "application/json, text/plain, */*")
	c.clientConfig.SetHeader("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36")
}

func (c *Client) updateAccountInfo() error {
	accountInfo, err := c.accountWithIpPool.GetAccountWithIpInfo()
	if err != nil {
		c.logger.Error("[Client.updateAccountInfo] Get account with ip info error: ", err)
		return err
	}
	c.accountInfo = accountInfo
	return nil
}

func (c *Client) getIpProxy() string {
	if c.accountInfo.IpInfo != nil {
		return c.accountInfo.IpInfo.FormatProxyToString()
	}
	return ""
}

func (c *Client) checkIpExpired() error {
	if c.config.ProxyConfig.EnableIpProxy && c.accountInfo.IpInfo != nil && c.accountInfo.IpInfo.Expired() {
		c.logger.Info("[Client.checkIpExpired] ip is expired, reload ip")
		ip, err := c.accountWithIpPool.ProxyIpPool.GetProxy()
		if err != nil {
			c.logger.Error("[Client.checkIpExpired] Get ip error: ", err)
			return err
		}
		c.accountInfo.IpInfo = ip
	}
	return nil
}

// preRequestData 预处理请求数据（添加WBI签名）
func (c *Client) preRequestData(params map[string]string) (map[string]string, error) {
	signReq := &signsrv.BilibiliSignRequest{
		ReqData: params,
		Cookies: c.accountInfo.Account.Cookies,
	}
	signResp, err := c.signSrvClient.BilibiliSignServer(signReq)
	if err != nil {
		c.logger.Error("[Client.preRequestData] sign request error: ", err)
		return nil, err
	}
	// 添加签名参数
	params["wts"] = signResp.Result.Wts
	params["w_rid"] = signResp.Result.WRid
	return params, nil
}

// Request 发送HTTP请求
func (c *Client) Request(method, url string, headers map[string]string) (string, error) {
	var (
		err      error
		response *http.Response
	)

	err = c.checkIpExpired()
	if err != nil {
		return "", err
	}

	httpClient, err := httpclient.NewHttpClient(c.clientConfig.TimeOut, c.getIpProxy())
	if err != nil {
		c.logger.Error("[Client.Request] create http client error: ", err)
		return "", err
	}
	for key, value := range headers {
		httpClient.SetHeader(key, value)
	}

	if c.config.DebugRequest {
		c.logger.Debug("[Client.Request] Request url: ", url, " method: ", method)
	}

	switch method {
	case http.MethodGet:
		response, err = httpClient.Get(url)
	default:
		c.logger.Error("[Client.Request] unsupported Request method")
		return "", errors.New("unsupported request method")
	}

	if err != nil {
		c.logger.Error("[Client.Request] Request error: ", err)
		return "", err
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		c.logger.Error("[Client.Request] read response body error: ", err)
		return "", err
	}
	defer response.Body.Close()

	if c.config.DebugRequest {
		c.logger.Debug("[Client.Request] Response: ", string(bodyBytes))
	}

	return string(bodyBytes), nil
}

// Get 发送GET请求
func (c *Client) Get(uri string, params map[string]string) (string, error) {
	// 添加WBI签名
	signedParams, err := c.preRequestData(params)
	if err != nil {
		return "", err
	}

	// 构建URL
	queryStr := tools.EncodeQueryParamsStr(signedParams)
	url := c.apiURL(uri) + "?" + queryStr

	headers := c.clientConfig.GetHeaders()
	headers["Cookie"] = c.accountInfo.Account.Cookies

	return c.Request(http.MethodGet, url, headers)
}

func (c *Client) unmarshalResponseData(data []byte, resp any) error {
	return json.Unmarshal(data, resp)
}

// Pong 检查登录状态
func (c *Client) Pong() bool {
	if !c.signSrvClient.PongSignServer() {
		c.logger.Error("[Client.Pong] sign server pong failed")
		return false
	}
	c.logger.Info("[Client.Pong] Begin to check login state....")
	if c.accountInfo == nil {
		c.logger.Error("[Client.Pong] account info is empty")
		return false
	}
	// 简单检查cookie是否存在
	if c.accountInfo.Account.Cookies == "" {
		return false
	}
	return true
}

// SearchVideoByKeyword 搜索视频
func (c *Client) SearchVideoByKeyword(keyword string, page, pageSize int, order string) (*bilimodel.SearchData, error) {
	uri := "/x/web-interface/wbi/search/type"
	params := map[string]string{
		"search_type": "video",
		"keyword":     keyword,
		"page":        strconv.Itoa(page),
		"page_size":   strconv.Itoa(pageSize),
		"order":       order,
	}

	respData, err := c.Get(uri, params)
	if err != nil {
		return nil, err
	}

	var resp bilimodel.SearchResponse
	if err := c.unmarshalResponseData([]byte(respData), &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, errors.New(resp.Message)
	}

	return resp.Data, nil
}

// GetHotSearchItems 获取B站热门视频列表
func (c *Client) GetHotSearchItems(page, pageSize int) (*bilimodel.PopularData, error) {
	params := map[string]string{
		"ps": strconv.Itoa(pageSize),
		"pn": strconv.Itoa(page),
	}
	url := c.apiURL("/x/web-interface/popular") + "?" + tools.EncodeQueryParamsStr(params)
	headers := c.clientConfig.GetHeaders()
	headers["Referer"] = "https://www.bilibili.com/v/popular/all/"
	headers["Cookie"] = c.accountInfo.Account.Cookies

	respData, err := c.Request(http.MethodGet, url, headers)
	if err != nil {
		return nil, err
	}

	var resp bilimodel.PopularResponse
	if err := c.unmarshalResponseData([]byte(respData), &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, errors.New(resp.Message)
	}

	return resp.Data, nil
}

// GetVideoInfo 获取视频详情
func (c *Client) GetVideoInfo(bvid string, aid int64) (*bilimodel.VideoInfo, error) {
	uri := "/x/web-interface/view/detail"
	params := map[string]string{}
	if bvid != "" {
		params["bvid"] = bvid
	} else if aid > 0 {
		params["aid"] = strconv.FormatInt(aid, 10)
	}

	respData, err := c.Get(uri, params)
	if err != nil {
		return nil, err
	}

	var resp bilimodel.VideoDetailResponse
	if err := c.unmarshalResponseData([]byte(respData), &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, errors.New(resp.Message)
	}

	if resp.Data == nil {
		return nil, nil
	}
	return resp.Data.View, nil
}

// GetVideoComments 获取视频评论
func (c *Client) GetVideoComments(aid int64, cursor int, pageSize int) (*bilimodel.CommentData, error) {
	uri := "/x/v2/reply/wbi/main"
	params := map[string]string{
		"type": "1",
		"oid":  strconv.FormatInt(aid, 10),
		"mode": "3",
		"plat": "1",
		"ps":   strconv.Itoa(pageSize),
		"next": strconv.Itoa(cursor),
	}

	respData, err := c.Get(uri, params)
	if err != nil {
		return nil, err
	}

	var resp bilimodel.CommentResponse
	if err := c.unmarshalResponseData([]byte(respData), &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, errors.New(resp.Message)
	}

	return resp.Data, nil
}

// GetVideoSubComments 获取视频子评论
func (c *Client) GetVideoSubComments(aid int64, rootRpid int64, pageNum int, pageSize int) (*bilimodel.CommentData, error) {
	uri := "/x/v2/reply/reply"
	params := map[string]string{
		"type": "1",
		"oid":  strconv.FormatInt(aid, 10),
		"mode": "3",
		"ps":   strconv.Itoa(pageSize),
		"pn":   strconv.Itoa(pageNum),
		"root": strconv.FormatInt(rootRpid, 10),
	}

	respData, err := c.Get(uri, params)
	if err != nil {
		return nil, err
	}

	var resp bilimodel.CommentResponse
	if err := c.unmarshalResponseData([]byte(respData), &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, errors.New(resp.Message)
	}

	return resp.Data, nil
}

// GetUpInfo 获取UP主信息
func (c *Client) GetUpInfo(mid int64) (*bilimodel.UpInfo, error) {
	uri := "/x/space/wbi/acc/info"
	params := map[string]string{
		"mid": strconv.FormatInt(mid, 10),
	}

	respData, err := c.Get(uri, params)
	if err != nil {
		return nil, err
	}

	var resp bilimodel.UpInfoResponse
	if err := c.unmarshalResponseData([]byte(respData), &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, errors.New(resp.Message)
	}

	return resp.Data, nil
}

// GetUpStat 获取UP主统计信息
func (c *Client) GetUpStat(mid int64) (*bilimodel.UpStat, error) {
	uri := "/x/relation/stat"
	params := map[string]string{
		"vmid": strconv.FormatInt(mid, 10),
	}

	respData, err := c.Get(uri, params)
	if err != nil {
		return nil, err
	}

	var resp bilimodel.UpStatResponse
	if err := c.unmarshalResponseData([]byte(respData), &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, errors.New(resp.Message)
	}

	return resp.Data, nil
}

// GetSpaceNavNum 获取UP主空间导航栏统计
func (c *Client) GetSpaceNavNum(mid int64) (*bilimodel.SpaceNavNum, error) {
	uri := "/x/space/navnum"
	params := map[string]string{
		"mid": strconv.FormatInt(mid, 10),
	}

	respData, err := c.Get(uri, params)
	if err != nil {
		return nil, err
	}

	var resp bilimodel.SpaceNavNumResponse
	if err := c.unmarshalResponseData([]byte(respData), &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, errors.New(resp.Message)
	}

	return resp.Data, nil
}

// GetUpVideos 获取UP主视频列表
func (c *Client) GetUpVideos(mid int64, page, pageSize int) (*bilimodel.SpaceVideoData, error) {
	uri := "/x/space/wbi/arc/search"
	params := map[string]string{
		"mid": strconv.FormatInt(mid, 10),
		"pn":  strconv.Itoa(page),
		"ps":  strconv.Itoa(pageSize),
	}

	respData, err := c.Get(uri, params)
	if err != nil {
		return nil, err
	}

	var resp bilimodel.SpaceVideoResponse
	if err := c.unmarshalResponseData([]byte(respData), &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, errors.New(resp.Message)
	}

	return resp.Data, nil
}

// GetHomefeedVideos 获取首页推荐
func (c *Client) GetHomefeedVideos(freshIdx int) (*bilimodel.HomefeedData, error) {
	uri := "/x/web-interface/wbi/index/top/feed/rcmd"
	params := map[string]string{
		"fresh_type":   "3",
		"ps":           "12",
		"fresh_idx":    strconv.Itoa(freshIdx),
		"fresh_idx_1h": strconv.Itoa(freshIdx),
	}

	respData, err := c.Get(uri, params)
	if err != nil {
		return nil, err
	}

	var resp bilimodel.HomefeedResponse
	if err := c.unmarshalResponseData([]byte(respData), &resp); err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, errors.New(resp.Message)
	}

	return resp.Data, nil
}
