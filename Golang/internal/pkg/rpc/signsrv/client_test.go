// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/rpc/signsrv/client_test.go
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
	"net"
	"net/url"
	"strconv"
	"testing"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	signsrvtest "github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/testsupport/signsrv"
	"github.com/stretchr/testify/require"
)

func TestClientReadsMockSignServerResponses(t *testing.T) {
	server := signsrvtest.NewMockServer()
	defer server.Close()

	host, port := signServerHostPort(t, server.URL)
	client := NewSignClient(&config.Config{
		SignSrvConfig: &config.SignSrvConfig{
			SignSrvHost: host,
			SignSrvPort: port,
		},
	})

	require.True(t, client.PongSignServer())

	xhsResp, err := client.XhsSignServer(&XhsSignRequest{
		Uri:     "/api/sns/web/v1/search/notes",
		Data:    map[string]any{"keyword": "mock"},
		Cookies: "a1=mock",
	})
	require.NoError(t, err)
	require.Equal(t, "mock-x-s", xhsResp.Result.Xs)
	require.Equal(t, "mock-x-t", xhsResp.Result.Xt)
	require.Equal(t, "mock-x-s-common", xhsResp.Result.XsCommon)
	require.Equal(t, "mock-x-b3-trace-id", xhsResp.Result.XB3TraceId)

	bilibiliResp, err := client.BilibiliSignServer(&BilibiliSignRequest{
		ReqData: map[string]string{"keyword": "mock"},
		Cookies: "SESSDATA=mock",
	})
	require.NoError(t, err)
	require.Equal(t, "1710000000", bilibiliResp.Result.Wts)
	require.Equal(t, "mock-w-rid", bilibiliResp.Result.WRid)

	douyinResp, err := client.DouyinSignServer(&DouyinSignRequest{
		Uri:         "/aweme/v1/web/general/search/single/",
		QueryParams: "keyword=mock",
		UserAgent:   "mock-agent",
		Cookies:     "sessionid=mock",
	})
	require.NoError(t, err)
	require.Equal(t, "mock-a-bogus", douyinResp.Result.ABogus)

	zhihuResp, err := client.ZhihuSignServer(&ZhihuSignRequest{
		Uri:     "/api/v4/search_v3?q=mock",
		Cookies: "_zap=mock",
	})
	require.NoError(t, err)
	require.Equal(t, "mock-x-zst-81", zhihuResp.Result.XZst81)
	require.Equal(t, "mock-x-zse-96", zhihuResp.Result.XZse96)
}

func signServerHostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)
	host, portText, err := net.SplitHostPort(parsed.Host)
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	return host, port
}
