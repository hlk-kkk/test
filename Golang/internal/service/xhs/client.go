// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/xhs/client.go
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

package xhs

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	config "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models"
	xhsmodel "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models/xiaohongshu"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/account_pool"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/httpclient"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/rpc/signsrv"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
)

const (
	ApiUrl   = "https://edith.xiaohongshu.com"
	IndexUrl = "https://www.xiaohongshu.com"
)

// Client xhs Request client struct
type Client struct {
	// clientConfig Request client config
	clientConfig *httpclient.ClientConfig

	// accountWithIpPool account with ip pool struct object
	accountWithIpPool *account_pool.AccountWithIpPool

	// accountInfo account info struct
	accountInfo *models.AccountWithIpPoolModel

	// logger record log
	logger *tools.Logger

	// config global config
	config *config.Config

	// sign srv client
	signSrvClient *signsrv.Client
}

// NewXhsClient create xhs Request client
func NewXhsClient(accountWithIpPool *account_pool.AccountWithIpPool, signSrvClient *signsrv.Client) *Client {
	globalConfig := config.GetConfig()
	clientConfig := &httpclient.ClientConfig{
		TimeOut: time.Duration(globalConfig.RequestTimeOut) * time.Second,
		Headers: make(map[string]string),
	}
	xhsClient := Client{
		logger:            tools.GetLogger(),
		config:            globalConfig,
		clientConfig:      clientConfig,
		accountWithIpPool: accountWithIpPool,
		signSrvClient:     signSrvClient,
	}
	xhsClient.addRequestHeaders()
	return &xhsClient
}

func (c *Client) apiURL(path string) string {
	baseURL := ApiUrl
	if c != nil && c.config != nil && c.config.PlatformEndpoints != nil && c.config.PlatformEndpoints.XhsApiURL != "" {
		baseURL = c.config.PlatformEndpoints.XhsApiURL
	}
	return joinBaseURL(baseURL, path)
}

func (c *Client) indexURL(path string) string {
	baseURL := IndexUrl
	if c != nil && c.config != nil && c.config.PlatformEndpoints != nil && c.config.PlatformEndpoints.XhsIndexURL != "" {
		baseURL = c.config.PlatformEndpoints.XhsIndexURL
	}
	return joinBaseURL(baseURL, path)
}

func joinBaseURL(baseURL, path string) string {
	if path == "" {
		return strings.TrimRight(baseURL, "/")
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

// Initialize account info, if failed, panic
func (c *Client) Initialize() {
	err := c.updateAccountInfo()
	if err != nil {
		c.logger.Error("[Client.Initialize] update account info error: ", err)
		panic(err)
	}
}

// addRequestHeaders add Request headers (与 Python 版本对齐)
func (c *Client) addRequestHeaders() {
	c.clientConfig.SetHeader("Accept", "application/json, text/plain, */*")
	c.clientConfig.SetHeader("Accept-Language", "zh-CN,zh;q=0.9")
	c.clientConfig.SetHeader("Cache-Control", "no-cache")
	c.clientConfig.SetHeader("Content-Type", "application/json;charset=UTF-8")
	c.clientConfig.SetHeader("Origin", "https://www.xiaohongshu.com")
	c.clientConfig.SetHeader("Pragma", "no-cache")
	c.clientConfig.SetHeader("Priority", "u=1, i")
	c.clientConfig.SetHeader("Referer", "https://www.xiaohongshu.com/")
	c.clientConfig.SetHeader("Sec-Ch-Ua", `"Chromium";v="136", "Google Chrome";v="136", "Not.A/Brand";v="99"`)
	c.clientConfig.SetHeader("Sec-Ch-Ua-Mobile", "?0")
	c.clientConfig.SetHeader("Sec-Ch-Ua-Platform", `"Windows"`)
	c.clientConfig.SetHeader("Sec-Fetch-Dest", "empty")
	c.clientConfig.SetHeader("Sec-Fetch-Mode", "cors")
	c.clientConfig.SetHeader("Sec-Fetch-Site", "same-site")
	c.clientConfig.SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")
}

// updateAccountInfo update account info
func (c *Client) updateAccountInfo() error {
	accountInfo, err := c.accountWithIpPool.GetAccountWithIpInfo()
	if err != nil {
		c.logger.Error("[Client.updateAccountInfo] Get account with ip info error: ", err)
		return err
	}
	c.accountInfo = accountInfo
	return nil
}

// markAccountInvalid mark account invalid
func (c *Client) markAccountInvalid(accountInfo *models.AccountWithIpPoolModel) {
	if c.accountWithIpPool == nil {
		return
	}
	if !c.Pong() {
		// if login status is invalid, mark account invalid ( sometimes only Get Post detail blocked, actual cookies can be used, so add a layer of judgment here)
		c.logger.Info("[Client.markAccountInvalid] account login status is invalid, mark account invalid")
		c.accountWithIpPool.MarkAccountInvalid(accountInfo)
	}
	c.accountWithIpPool.MarkIpInvalid(accountInfo.IpInfo)
}

// getIpProxy Get ip proxy
func (c *Client) getIpProxy() string {
	if c.accountInfo.IpInfo != nil {
		return c.accountInfo.IpInfo.FormatProxyToString()
	}
	return ""
}

// checkIpExpired check whether ip is expired, if expired, we will retry to Get a new ip and update account ip info
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

// validateXhsResp validate xhs response
func (c *Client) validateXhsResp(requestHeaders map[string]string, resp *http.Response, bodyBytes []byte, isHtml bool) error {
	// xhs出现验证码
	if resp.StatusCode == 471 || resp.StatusCode == 461 {
		VerifyType := resp.Header.Get("Verifytype")
		VerifyUuid := resp.Header.Get("Verifyuuid")
		c.logger.Warn("[Client.validateXhsResp] 出现验证码，请求失败, VerifyType: ", VerifyType, " VerifyUuid: ", VerifyUuid)
		return ParseVerifyError(resp.StatusCode, VerifyType, VerifyUuid)
	}

	// 说明是获取HTML页面内容，检查状态码
	if isHtml {
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		c.logger.Error("[Client.validateXhsResp] get html page, get unknown error")
		return WrapNetworkError(errors.New("unknown error"))
	}

	respStruct := &xhsmodel.Response[any]{}
	err := c.unmarshalResponseData(bodyBytes, respStruct)
	if err != nil {
		c.logger.Error("[Client.validateXhsResp] unmarshal response data error: ", err)
		return WrapNetworkError(err)
	}

	if respStruct.Success {
		return nil
	}

	// 使用 ParseXhsError 统一处理错误
	xhsErr := ParseXhsError(respStruct.Code, respStruct.Msg)

	// 记录具体的错误信息
	switch xhsErr.Type {
	case IPBlockError:
		c.logger.Warn("[Client.validateXhsResp] IP被封禁")
	case NoteSecreteError:
		c.logger.Warn("[Client.validateXhsResp] 笔记内容不可见")
	case SignError:
		c.logger.Warn("[Client.validateXhsResp] 签名验证失败")
	case SessionExpiredError:
		c.logger.Warn("[Client.validateXhsResp] 登录已过期")
	case AccessFrequencyError:
		c.logger.Warn("[Client.validateXhsResp] 访问频率异常")
	default:
		c.logger.Warn("[Client.validateXhsResp] 未知错误码: ", respStruct.Code)
	}

	return xhsErr
}

// Request do Request
func (c *Client) Request(method, url string, data []byte, headers map[string]string, isHtml bool) (string, error) {
	var (
		err      error
		response *http.Response
	)

	err = c.checkIpExpired()
	if err != nil {
		return "", err
	}

	// create http client and set headers
	httpClient, err := httpclient.NewHttpClient(c.clientConfig.TimeOut, c.getIpProxy())
	if err != nil {
		c.logger.Error("[Client.Request] create http client error: ", err)
		return "", err
	}
	for key, value := range headers {
		httpClient.SetHeader(key, value)
	}

	// only set debug log when debugRequest is true
	if c.config.DebugRequest {
		c.logger.Debug("[Client.Request] Request url: ", url, " method: ", method, " data: ", string(data), " headers: ", headers)
	}

	// do Request
	switch method {
	case http.MethodGet:
		response, err = httpClient.Get(url)
	case http.MethodPost:
		response, err = httpClient.Post(url, data)
	default:
		c.logger.Error("[Client.Request] unsupported Request method")
		panic("unsupported Request method")
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
		c.logger.Debug("[Client.Request] Request response status: ", response.Status, " headers: ", response.Header, " body: ", string(bodyBytes))
	}

	// validate xhs response
	err = c.validateXhsResp(headers, response, bodyBytes, isHtml)
	if err != nil {
		c.logger.Error("[Client.Request] validate xhs response error: ", err)
		return "", err
	}

	return string(bodyBytes), nil
}

// preHeaders Request params signature
func (c *Client) preHeaders(uri string, data any) (map[string]string, error) {
	req := signsrv.XhsSignRequest{
		Uri:     uri,
		Data:    data,
		Cookies: c.accountInfo.Account.Cookies,
	}
	resp, err := c.signSrvClient.XhsSignServer(&req)
	if err != nil {
		c.logger.Error("[Client.preHeaders] sign Request error: ", err)
		return map[string]string{}, err
	}
	headers := c.clientConfig.GetHeaders()
	headers["X-S"] = resp.Result.Xs
	headers["X-T"] = resp.Result.Xt
	headers["X-S-Common"] = resp.Result.XsCommon
	headers["X-B3-Traceid"] = resp.Result.XB3TraceId
	headers["Cookie"] = c.accountInfo.Account.Cookies

	return headers, nil
}

// Get do Get Request with retry and account switch
func (c *Client) Get(uri string, params map[string]any) (string, error) {
	if params != nil {
		uri = uri + "?" + tools.EncodeQueryParams(params)
	}

	// 使用重试机制
	ctx := context.Background()
	config := DefaultRetryConfig()

	// 在重试失败后切换账号
	config.BeforeRetry = func(attempt int) error {
		if attempt > 0 {
			c.logger.Warn("[Client.Get] Retry attempt ", attempt, ", switching account...")
			// 标记当前账号失效
			c.markAccountInvalid(c.accountInfo)
			// 更新账号信息
			err := c.updateAccountInfo()
			if err != nil {
				c.logger.Error("[Client.Get] update account info error: ", err)
				return err
			}
			c.logger.Info("[Client.Get] Switched to new account")
		}
		return nil
	}

	result, err := WithRetry(ctx, func(ctx context.Context) (interface{}, error) {
		headers, err := c.preHeaders(uri, nil)
		if err != nil {
			c.logger.Error("[Client.Get] pre headers error: ", err)
			return nil, WrapNetworkError(err)
		}
		return c.Request(http.MethodGet, c.apiURL(uri), nil, headers, false)
	}, config)

	if err != nil {
		return "", err
	}

	return result.(string), nil
}

// Post do Post Request with retry and account switch
func (c *Client) Post(uri string, data any) (string, error) {
	// 使用重试机制
	ctx := context.Background()
	config := DefaultRetryConfig()

	// 在重试失败后切换账号
	config.BeforeRetry = func(attempt int) error {
		if attempt > 0 {
			c.logger.Warn("[Client.Post] Retry attempt ", attempt, ", switching account...")
			// 标记当前账号失效
			c.markAccountInvalid(c.accountInfo)
			// 更新账号信息
			err := c.updateAccountInfo()
			if err != nil {
				c.logger.Error("[Client.Post] update account info error: ", err)
				return err
			}
			c.logger.Info("[Client.Post] Switched to new account")
		}
		return nil
	}

	result, err := WithRetry(ctx, func(ctx context.Context) (interface{}, error) {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			c.logger.Error("[Client.Post] marshal Request data error: ", err)
			return nil, WrapNetworkError(err)
		}

		headers, err := c.preHeaders(uri, data)
		if err != nil {
			c.logger.Error("[Client.Post] pre headers error: ", err)
			return nil, WrapNetworkError(err)
		}

		return c.Request(http.MethodPost, c.apiURL(uri), jsonBytes, headers, false)
	}, config)

	if err != nil {
		return "", err
	}

	return result.(string), nil
}

// Pong account login status check
func (c *Client) Pong() bool {
	// ping sign srv is alive
	if !c.signSrvClient.PongSignServer() {
		c.logger.Error("[Client.Pong] sign server pong failed")
		return false
	}

	// check account login status
	c.logger.Info("[Client.Pong] Begin to check login state....")
	if c.accountInfo == nil {
		c.logger.Error("[Client.Pong] account info is empty")
		return false
	}

	resp, err := c.QuerySelfInfo()
	if err != nil {
		c.logger.Error("[Client.Pong] query self info error: ", err)
		return false
	}

	// 注意:检查的是data.result.success,而不是外层的success
	// 外层success=false只是表示接口调用,内层result.success才是真正的登录状态
	if resp.Data != nil && resp.Data.Result.Success {
		c.logger.Info("[Client.Pong] account login status is valid")
		return true
	}

	c.logger.Error("[Client.Pong] account login status is invalid, result.success=", resp.Data != nil && resp.Data.Result.Success)
	return false
}

// unmarshalResponseData unmarshal response data
func (c *Client) unmarshalResponseData(data []byte, resp any) error {
	err := json.Unmarshal(data, resp)
	if err != nil {
		c.logger.Error("[Client.unmarshalResponseData] unmarshal response data error: ", err)
		return err
	}
	return nil
}

// QuerySelfInfo query self info
func (c *Client) QuerySelfInfo() (*xhsmodel.Response[*xhsmodel.UserSelfInfo], error) {
	uri := "/api/sns/web/v1/user/selfinfo"
	respData, err := c.Get(uri, nil)
	if err != nil {
		c.logger.Error("[Client.QuerySelfInfo] query self info error: ", err)
		return nil, err
	}

	var respStruct xhsmodel.Response[*xhsmodel.UserSelfInfo]
	err = c.unmarshalResponseData([]byte(respData), &respStruct)
	if err != nil {
		return nil, err
	}

	return &respStruct, nil
}

// SearchNotesByKeyword search notes by keyword
func (c *Client) SearchNotesByKeyword(req *xhsmodel.SearchRequest) (*xhsmodel.SearchWrapper, error) {
	uri := "/api/sns/web/v1/search/notes"

	respData, err := c.Post(uri, req)
	if err != nil {
		return nil, err
	}

	var respStruct xhsmodel.Response[*xhsmodel.SearchWrapper]
	err = c.unmarshalResponseData([]byte(respData), &respStruct)
	if err != nil {
		return nil, err
	}
	return respStruct.Data, nil
}

// GetNoteDetailFromApi get note detail from api
func (c *Client) GetNoteDetailFromApi(noteId, xsecToken, xsecSource string) (*xhsmodel.NoteDetail, error) {
	uri := "/api/sns/web/v1/feed"

	req := xhsmodel.NoteDetailApiFeedRequest{
		SourceNoteID: noteId,
		ImageFormats: []string{"jpg", "webp", "avif"},
		Extra: struct {
			NeedBodyTopic int `json:"need_body_topic"`
		}{NeedBodyTopic: 1},
		XsecToken:  xsecToken,
		XsecSource: xsecSource,
	}

	respData, err := c.Post(uri, req)
	if err != nil {
		return nil, err
	}

	var respStruct xhsmodel.Response[*xhsmodel.NoteDetailResponseFromApi]
	err = c.unmarshalResponseData([]byte(respData), &respStruct)
	if err != nil {
		return nil, err
	}

	if len(respStruct.Data.Items) > 0 {
		return &respStruct.Data.Items[0].NoteCard, nil
	}

	return nil, errors.New("no note detail found")
}

// GetNoteDetailFromHtml get note detail from html
func (c *Client) GetNoteDetailFromHtml(noteId, xsecToken, xsecSource string) (*xhsmodel.NoteDetail, error) {
	url := c.indexURL("/explore/" + noteId + "?xsec_token=" + xsecToken + "&xsec_source=" + xsecSource)
	headers := c.clientConfig.GetHeaders()
	headers["Cookie"] = c.accountInfo.Account.Cookies
	respData, err := c.Request(http.MethodGet, url, nil, headers, true)
	if err != nil {
		return nil, err
	}

	if strings.Contains(respData, "noteDetailMap") {
		// use regex to get state
		re := regexp.MustCompile(`window.__INITIAL_STATE__=({.*})</script>`)
		matches := re.FindStringSubmatch(respData)
		if len(matches) < 2 {
			return nil, errors.New("failed to extract initial state")
		}
		pageInitData := matches[1]
		// Replace 'undefined' with 'null' to avoid JSON parsing errors
		pageInitData = strings.ReplaceAll(pageInitData, "undefined", "null")

		var respStruct xhsmodel.NoteDetailResponseFromHtml
		err = c.unmarshalResponseData([]byte(pageInitData), &respStruct)
		if err != nil {
			return nil, err
		}
		noteDetail := respStruct.Note.NoteDetailMap.Data[noteId].Note
		return &noteDetail, nil
	}
	return nil, errors.New("get note detail from html error")
}

// GetNoteComments 获取帖子一级评论
func (c *Client) GetNoteComments(noteID, cursor, xsecToken string) (*xhsmodel.CommentResponse, error) {
	uri := "/api/sns/web/v2/comment/page"
	params := map[string]any{
		"note_id":        noteID,
		"cursor":         cursor,
		"top_comment_id": "",
		"image_formats":  "jpg,webp,avif",
	}
	if xsecToken != "" {
		params["xsec_token"] = xsecToken
	}

	respData, err := c.Get(uri, params)
	if err != nil {
		return nil, err
	}

	var respStruct xhsmodel.Response[*xhsmodel.CommentResponse]
	err = c.unmarshalResponseData([]byte(respData), &respStruct)
	if err != nil {
		return nil, err
	}

	return respStruct.Data, nil
}

// GetNoteSubComments 获取子评论
func (c *Client) GetNoteSubComments(noteID, rootCommentID, cursor, xsecToken string, num int) (*xhsmodel.CommentResponse, error) {
	uri := "/api/sns/web/v2/comment/sub/page"
	params := map[string]any{
		"note_id":         noteID,
		"root_comment_id": rootCommentID,
		"num":             num,
		"cursor":          cursor,
	}
	if xsecToken != "" {
		params["xsec_token"] = xsecToken
	}

	respData, err := c.Get(uri, params)
	if err != nil {
		return nil, err
	}

	var respStruct xhsmodel.Response[*xhsmodel.CommentResponse]
	err = c.unmarshalResponseData([]byte(respData), &respStruct)
	if err != nil {
		return nil, err
	}

	return respStruct.Data, nil
}

// GetCreatorInfo 获取创作者信息（从HTML解析）
func (c *Client) GetCreatorInfo(userID, xsecToken, xsecSource string) (*xhsmodel.CreatorInfo, error) {
	url := c.indexURL("/user/profile/" + userID)
	if xsecToken != "" {
		url += "?xsec_token=" + xsecToken + "&xsec_source=" + xsecSource
	}

	headers := c.clientConfig.GetHeaders()
	headers["Cookie"] = c.accountInfo.Account.Cookies

	respData, err := c.Request(http.MethodGet, url, nil, headers, true)
	if err != nil {
		return nil, err
	}

	return c.extractCreatorFromHtml(userID, respData)
}

// extractCreatorFromHtml 从HTML中提取创作者信息
func (c *Client) extractCreatorFromHtml(userID, htmlContent string) (*xhsmodel.CreatorInfo, error) {
	re := regexp.MustCompile(`window.__INITIAL_STATE__=({.*})</script>`)
	matches := re.FindStringSubmatch(htmlContent)
	if len(matches) < 2 {
		return nil, errors.New("failed to extract initial state for creator")
	}

	pageInitData := matches[1]
	pageInitData = strings.ReplaceAll(pageInitData, "undefined", "null")

	// 解析用户信息
	var data map[string]interface{}
	if err := c.unmarshalResponseData([]byte(pageInitData), &data); err != nil {
		return nil, err
	}

	userInfo := &xhsmodel.CreatorInfo{UserID: userID}

	if user, ok := data["user"].(map[string]interface{}); ok {
		if userDetail, ok := user["userPageData"].(map[string]interface{}); ok {
			if basicInfo, ok := userDetail["basicInfo"].(map[string]interface{}); ok {
				if nickname, ok := basicInfo["nickname"].(string); ok {
					userInfo.Nickname = nickname
				}
				if avatar, ok := basicInfo["imageb"].(string); ok && avatar != "" {
					userInfo.Avatar = avatar
				} else if avatar, ok := basicInfo["images"].(string); ok {
					userInfo.Avatar = avatar
				}
				if desc, ok := basicInfo["desc"].(string); ok {
					userInfo.Desc = desc
				}
				if gender, ok := basicInfo["gender"].(float64); ok {
					if gender == 1 {
						userInfo.Gender = "女"
					} else {
						userInfo.Gender = "男"
					}
				}
				if ipLocation, ok := basicInfo["ipLocation"].(string); ok {
					userInfo.IPLocation = ipLocation
				}
			}
			if interactions, ok := userDetail["interactions"].([]interface{}); ok {
				for _, inter := range interactions {
					if interMap, ok := inter.(map[string]interface{}); ok {
						name, _ := interMap["name"].(string)
						interactionType, _ := interMap["type"].(string)
						count, _ := interMap["count"].(string)
						switch {
						case name == "关注" || interactionType == "follows":
							userInfo.Follows = count
						case name == "粉丝" || interactionType == "fans":
							userInfo.Fans = count
						case name == "获赞与收藏" || interactionType == "interaction":
							userInfo.Interaction = count
						}
					}
				}
			}
			if tags, ok := userDetail["tags"].([]interface{}); ok {
				tagParts := make([]string, 0, len(tags))
				for _, tag := range tags {
					tagMap, ok := tag.(map[string]interface{})
					if !ok {
						continue
					}
					tagType, _ := tagMap["tagType"].(string)
					name, _ := tagMap["name"].(string)
					if tagType == "" {
						continue
					}
					tagTypeJSON, _ := json.Marshal(tagType)
					nameJSON, _ := json.Marshal(name)
					tagParts = append(tagParts, string(tagTypeJSON)+": "+string(nameJSON))
				}
				if len(tagParts) > 0 {
					userInfo.TagList = "{" + strings.Join(tagParts, ", ") + "}"
				}
			}
			if userInfo.TagList == "" {
				if tagList, ok := userDetail["tagList"].(string); ok {
					userInfo.TagList = tagList
				}
			}
			if userInfo.Follows == "" || userInfo.Fans == "" || userInfo.Interaction == "" {
				if interactions, ok := userDetail["interactions"].([]interface{}); ok {
					for _, inter := range interactions {
						interMap, ok := inter.(map[string]interface{})
						if !ok {
							continue
						}
						name, _ := interMap["name"].(string)
						count, _ := interMap["count"].(string)
						switch name {
						case "关注":
							userInfo.Follows = count
						case "粉丝":
							userInfo.Fans = count
						case "获赞与收藏":
							userInfo.Interaction = count
						}
					}
				}
			}
		}
	}

	return userInfo, nil
}

// GetCreatorNotes 获取创作者的帖子列表
func (c *Client) GetCreatorNotes(creatorID, cursor, xsecToken, xsecSource string, pageSize int) (*xhsmodel.CreatorNotesResponse, error) {
	uri := "/api/sns/web/v1/user_posted"
	params := map[string]any{
		"num":           pageSize,
		"cursor":        cursor,
		"user_id":       creatorID,
		"image_formats": "jpg,webp,avif",
	}
	if xsecToken != "" {
		params["xsec_token"] = xsecToken
	}
	if xsecSource != "" {
		params["xsec_source"] = xsecSource
	}

	respData, err := c.Get(uri, params)
	if err != nil {
		return nil, err
	}

	var respStruct xhsmodel.Response[*xhsmodel.CreatorNotesResponse]
	err = c.unmarshalResponseData([]byte(respData), &respStruct)
	if err != nil {
		return nil, err
	}

	return respStruct.Data, nil
}

// GetHomefeedNotes 获取首页推荐帖子
func (c *Client) GetHomefeedNotes(category xhsmodel.FeedType, cursor string, noteIndex, noteNum int) (*xhsmodel.HomefeedResponse, error) {
	uri := "/api/sns/web/v1/homefeed"
	data := xhsmodel.HomefeedRequest{
		Category:          string(category),
		CursorScore:       cursor,
		ImageFormats:      []string{"jpg", "webp", "avif"},
		NeedFilterImage:   false,
		NeedNum:           noteNum,
		NoteIndex:         noteIndex,
		Num:               noteNum,
		RefreshType:       3,
		SearchKey:         "",
		UnreadBeginNoteID: "",
		UnreadEndNoteID:   "",
		UnreadNoteCount:   0,
	}

	respData, err := c.Post(uri, data)
	if err != nil {
		return nil, err
	}

	var respStruct xhsmodel.Response[*xhsmodel.HomefeedResponse]
	err = c.unmarshalResponseData([]byte(respData), &respStruct)
	if err != nil {
		return nil, err
	}

	return respStruct.Data, nil
}
