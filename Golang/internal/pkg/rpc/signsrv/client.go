// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/rpc/signsrv/client.go
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

package signsrv

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/httpclient"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	"github.com/google/wire"
	"io"
	"net/http"
	"time"
)

const RequestTimeOut = 30 * time.Second

var ProviderSet = wire.NewSet(NewSignClient)

type Client struct {
	logger   *tools.Logger
	endPoint string
}

// NewSignClient create a new client
func NewSignClient(config *config.Config) *Client {
	return &Client{
		logger:   tools.GetLogger(),
		endPoint: fmt.Sprintf("http://%s:%d", config.SignSrvConfig.SignSrvHost, config.SignSrvConfig.SignSrvPort),
	}
}

// request do request
func (c *Client) request(method, uri string, data []byte, headers map[string]string) (string, error) {
	var (
		err      error
		response *http.Response
	)
	// create http client and set headers
	httpClient, err := httpclient.NewHttpClient(RequestTimeOut, "")
	if err != nil {
		c.logger.Error("[Client.request] create http client error: ", err)
		return "", err
	}
	for key, value := range headers {
		httpClient.SetHeader(key, value)
	}
	httpClient.SetHeader("Content-Type", "application/json")

	url := c.endPoint + uri

	// do request
	switch method {
	case http.MethodGet:
		response, err = httpClient.Get(url)
	case http.MethodPost:
		response, err = httpClient.Post(url, data)
	default:
		c.logger.Error("[Client.request] unsupported request method")
		panic("unsupported request method")
	}
	if err != nil {
		c.logger.Error("[Client.request] request error: ", err)
		return "", err
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		c.logger.Error("[Client.request] read response body error: ", err)
		return "", err
	}
	defer response.Body.Close()

	return string(bodyBytes), nil
}

// XhsSignServer xhs sign server
func (c *Client) XhsSignServer(signReq *XhsSignRequest) (*XhsSignResponse, error) {
	uri := "/signsrv/v1/xhs/sign"

	jsonBytes, err := json.Marshal(signReq)
	if err != nil {
		c.logger.Error("[XhsClient.XhsSignServer] marshal request data error: ", err)
		return nil, err
	}

	body, err := c.request(http.MethodPost, uri, jsonBytes, nil)
	if err != nil {
		c.logger.Error("[XhsClient.XhsSignServer] request error: ", err)
		return nil, err
	}

	signResp := &XhsSignResponse{}
	err = json.Unmarshal([]byte(body), signResp)
	if err != nil {
		c.logger.Error("[XhsClient.XhsSignServer] unmarshal response error: ", err)
		return nil, err
	}

	if !signResp.IsOk {
		c.logger.Error("[XhsClient.XhsSignServer] sign response error: ", signResp.Msg)
		return nil, errors.New(signResp.Msg)
	}

	return signResp, nil
}

// BilibiliSignServer bilibili sign server
func (c *Client) BilibiliSignServer(signReq *BilibiliSignRequest) (*BilibiliSignResponse, error) {
	uri := "/signsrv/v1/bilibili/sign"

	jsonBytes, err := json.Marshal(signReq)
	if err != nil {
		c.logger.Error("[BilibiliClient.BilibiliSignServer] marshal request data error: ", err)
		return nil, err
	}

	body, err := c.request(http.MethodPost, uri, jsonBytes, nil)
	if err != nil {
		c.logger.Error("[BilibiliClient.BilibiliSignServer] request error: ", err)
		return nil, err
	}

	signResp := &BilibiliSignResponse{}
	err = json.Unmarshal([]byte(body), signResp)
	if err != nil {
		c.logger.Error("[BilibiliClient.BilibiliSignServer] unmarshal response error: ", err)
		return nil, err
	}

	if !signResp.IsOk {
		c.logger.Error("[BilibiliClient.BilibiliSignServer] sign response error: ", signResp.Msg)
		return nil, errors.New(signResp.Msg)
	}

	return signResp, nil
}

// DouyinSignServer douyin sign server
func (c *Client) DouyinSignServer(signReq *DouyinSignRequest) (*DouyinSignResponse, error) {
	uri := "/signsrv/v1/douyin/sign"

	jsonBytes, err := json.Marshal(signReq)
	if err != nil {
		c.logger.Error("[DouyinClient.DouyinSignServer] marshal request data error: ", err)
		return nil, err
	}

	body, err := c.request(http.MethodPost, uri, jsonBytes, nil)
	if err != nil {
		c.logger.Error("[DouyinClient.DouyinSignServer] request error: ", err)
		return nil, err
	}

	signResp := &DouyinSignResponse{}
	err = json.Unmarshal([]byte(body), signResp)
	if err != nil {
		c.logger.Error("[DouyinClient.DouyinSignServer] unmarshal response error: ", err)
		return nil, err
	}

	if !signResp.IsOk {
		c.logger.Error("[DouyinClient.DouyinSignServer] sign response error: ", signResp.Msg)
		return nil, errors.New(signResp.Msg)
	}

	return signResp, nil
}

// ZhihuSignServer zhihu sign server
func (c *Client) ZhihuSignServer(signReq *ZhihuSignRequest) (*ZhihuSignResponse, error) {
	uri := "/signsrv/v1/zhihu/sign"

	jsonBytes, err := json.Marshal(signReq)
	if err != nil {
		c.logger.Error("[ZhihuClient.ZhihuSignServer] marshal request data error: ", err)
		return nil, err
	}

	body, err := c.request(http.MethodPost, uri, jsonBytes, nil)
	if err != nil {
		c.logger.Error("[ZhihuClient.ZhihuSignServer] request error: ", err)
		return nil, err
	}

	signResp := &ZhihuSignResponse{}
	err = json.Unmarshal([]byte(body), signResp)
	if err != nil {
		c.logger.Error("[ZhihuClient.ZhihuSignServer] unmarshal response error: ", err)
		return nil, err
	}

	if !signResp.IsOk {
		c.logger.Error("[ZhihuClient.ZhihuSignServer] sign response error: ", signResp.Msg)
		return nil, errors.New(signResp.Msg)
	}

	return signResp, nil
}

// PongSignServer pong sign server
func (c *Client) PongSignServer() bool {
	uri := "/signsrv/pong"
	_, err := c.request(http.MethodGet, uri, nil, nil)
	if err != nil {
		c.logger.Error("[XhsClient.PongSignServer] request error: ", err)
		return false
	}
	return true
}
