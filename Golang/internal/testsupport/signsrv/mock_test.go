// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/testsupport/signsrv/mock_test.go
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
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMockServerImplementsSignServiceAPIs(t *testing.T) {
	server := NewMockServer()
	defer server.Close()

	pong := getJSONMap(t, http.MethodGet, server.URL+"/signsrv/pong", nil)
	require.True(t, pong["isok"].(bool))
	require.EqualValues(t, 0, pong["biz_code"])
	require.Equal(t, "pong", pong["data"].(map[string]any)["message"])

	xhs := getJSONMap(t, http.MethodPost, server.URL+"/signsrv/v1/xhs/sign", map[string]any{
		"uri":     "/api/sns/web/v1/search/notes",
		"data":    map[string]any{"keyword": "mock"},
		"cookies": "a1=mock",
	})
	require.True(t, xhs["isok"].(bool))
	require.Equal(t, "mock-x-s", xhs["data"].(map[string]any)["x_s"])
	require.Equal(t, "mock-x-t", xhs["data"].(map[string]any)["x_t"])
	require.Equal(t, "mock-x-s-common", xhs["data"].(map[string]any)["x_s_common"])
	require.Equal(t, "mock-x-b3-trace-id", xhs["data"].(map[string]any)["x_b3_trace_id"])

	bilibili := getJSONMap(t, http.MethodPost, server.URL+"/signsrv/v1/bilibili/sign", map[string]any{
		"req_data": map[string]string{"keyword": "mock"},
		"cookies":  "SESSDATA=mock",
	})
	require.True(t, bilibili["isok"].(bool))
	require.Equal(t, "1710000000", bilibili["data"].(map[string]any)["wts"])
	require.Equal(t, "mock-w-rid", bilibili["data"].(map[string]any)["w_rid"])

	douyin := getJSONMap(t, http.MethodPost, server.URL+"/signsrv/v1/douyin/sign", map[string]any{
		"uri":          "/aweme/v1/web/general/search/single/",
		"query_params": "keyword=mock",
		"user_agent":   "mock-agent",
		"cookies":      "sessionid=mock",
	})
	require.True(t, douyin["isok"].(bool))
	require.Equal(t, "mock-a-bogus", douyin["data"].(map[string]any)["a_bogus"])

	zhihu := getJSONMap(t, http.MethodPost, server.URL+"/signsrv/v1/zhihu/sign", map[string]any{
		"uri":     "/api/v4/search_v3?q=mock",
		"cookies": "_zap=mock",
	})
	require.True(t, zhihu["isok"].(bool))
	require.Equal(t, "mock-x-zst-81", zhihu["data"].(map[string]any)["x_zst_81"])
	require.Equal(t, "mock-x-zse-96", zhihu["data"].(map[string]any)["x_zse_96"])
}

func getJSONMap(t *testing.T, method, requestURL string, body any) map[string]any {
	t.Helper()

	var bodyReader *bytes.Reader
	if body == nil {
		bodyReader = bytes.NewReader(nil)
	} else {
		bodyBytes, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(bodyBytes)
	}
	req, err := http.NewRequest(method, requestURL, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	return payload
}
