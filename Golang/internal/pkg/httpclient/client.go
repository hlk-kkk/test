// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/httpclient/client.go
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

package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const testBlockExternalHTTPEnv = "MCPRO_TEST_BLOCK_EXTERNAL_HTTP"

// HttpClient 封装常用的HTTP客户端操作
type HttpClient struct {
	client  *http.Client
	headers map[string]string
}

// NewHttpClient 创建一个新的HttpClient实例
func NewHttpClient(timeout time.Duration, proxyStr string) (*HttpClient, error) {
	transport := newTransport()

	// 如果设置了代理
	if proxyStr != "" {
		proxyURL, err := url.Parse(proxyStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy URL: %v", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	if testBlockExternalHTTPEnabled() {
		transport.DialContext = guardedDialContext()
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return &HttpClient{
		client:  client,
		headers: make(map[string]string),
	}, nil
}

func NewStandardClient(timeout time.Duration) *http.Client {
	client := &http.Client{Timeout: timeout}
	if testBlockExternalHTTPEnabled() {
		client.Transport = &http.Transport{DialContext: guardedDialContext()}
	}
	return client
}

func newTransport() *http.Transport {
	return &http.Transport{}
}

func testBlockExternalHTTPEnabled() bool {
	return os.Getenv(testBlockExternalHTTPEnv) == "1"
}

func guardedDialContext() func(context.Context, string, string) (net.Conn, error) {
	dialer := &net.Dialer{}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if !isLoopbackAddress(address) {
			return nil, fmt.Errorf("blocked external HTTP request to %s; automated tests must use loopback mock servers", address)
		}
		return dialer.DialContext(ctx, network, address)
	}
}

func isLoopbackAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// SetHeader 设置一个全局的请求头
func (h *HttpClient) SetHeader(key, value string) {
	h.headers[key] = value
}

// Get 发起GET请求
func (h *HttpClient) Get(url string) (*http.Response, error) {
	return h.doRequest("GET", url, nil)
}

// Post 发起POST请求
func (h *HttpClient) Post(url string, data []byte) (*http.Response, error) {
	return h.doRequest("POST", url, data)
}

// doRequest 处理HTTP请求的通用方法
func (h *HttpClient) doRequest(method, url string, data []byte) (*http.Response, error) {
	var req *http.Request
	var err error

	if data != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(data))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, err
	}

	// 设置全局请求头
	for key, value := range h.headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
