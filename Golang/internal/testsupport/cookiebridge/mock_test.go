// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/testsupport/cookiebridge/mock_test.go
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

package cookiebridge

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMockServerImplementsCookieBridgeHealthAndAccountAPIs(t *testing.T) {
	server := NewMockServer(
		Account{
			ClientID:  "client-a",
			Connected: true,
			Nicknames: map[string]string{"xhs": "小红书账号A"},
			Cookies: map[string]string{
				"xhs":  "web_session=session-a; a1=a1-a",
				"bili": "SESSDATA=bili-a",
			},
		},
		Account{
			ClientID:  "client-b",
			Connected: false,
			Nicknames: map[string]string{"xhs": "小红书账号B"},
			Cookies: map[string]string{
				"xhs": "web_session=session-b; a1=a1-b",
			},
		},
	)
	defer server.Close()

	ping := getJSONMap(t, server.URL+"/ping")
	require.True(t, ping["isok"].(bool))
	require.EqualValues(t, 0, ping["biz_code"])
	require.Equal(t, "pong", nestedString(t, ping, "data", "message"))

	status := getJSONMap(t, server.URL+"/api/status")
	require.True(t, status["isok"].(bool))
	require.EqualValues(t, 2, nestedFloat(t, status, "data", "total_clients"))
	require.EqualValues(t, 1, nestedFloat(t, status, "data", "connected_count"))
	require.Equal(t, "小红书账号A", nestedString(t, status, "data", "clients", "client-a", "nicknames", "xhs"))
	require.True(t, nestedBool(t, status, "data", "clients", "client-a", "platforms", "xhs", "has_cookies"))

	accounts := getJSONMap(t, server.URL+"/api/accounts")
	require.True(t, accounts["isok"].(bool))
	require.EqualValues(t, 2, nestedFloat(t, accounts, "data", "total"))
	require.Equal(t, "client-a", nestedString(t, accounts, "data", "accounts", "0", "client_id"))
	require.True(t, nestedBool(t, accounts, "data", "accounts", "0", "platforms", "xhs", "has_cookies"))

	cookieForClient := getJSONMap(t, server.URL+"/api/cookies/xhs?client_id=client-b")
	require.True(t, cookieForClient["isok"].(bool))
	require.Equal(t, "xhs", nestedString(t, cookieForClient, "data", "platform"))
	require.Equal(t, "client-b", nestedString(t, cookieForClient, "data", "client_id"))
	require.Equal(t, "web_session=session-b; a1=a1-b", nestedString(t, cookieForClient, "data", "cookies"))

	anyCookie := getJSONMap(t, server.URL+"/api/cookies/bili")
	require.True(t, anyCookie["isok"].(bool))
	require.Equal(t, "client-a", nestedString(t, anyCookie, "data", "client_id"))
	require.Equal(t, "SESSDATA=bili-a", nestedString(t, anyCookie, "data", "cookies"))
}

func getJSONMap(t *testing.T, requestURL string) map[string]any {
	t.Helper()

	resp, err := http.Get(requestURL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	return payload
}

func nestedString(t *testing.T, root map[string]any, path ...string) string {
	t.Helper()
	value := nestedValue(t, root, path...)
	result, ok := value.(string)
	require.Truef(t, ok, "expected string at %v, got %T", path, value)
	return result
}

func nestedBool(t *testing.T, root map[string]any, path ...string) bool {
	t.Helper()
	value := nestedValue(t, root, path...)
	result, ok := value.(bool)
	require.Truef(t, ok, "expected bool at %v, got %T", path, value)
	return result
}

func nestedFloat(t *testing.T, root map[string]any, path ...string) float64 {
	t.Helper()
	value := nestedValue(t, root, path...)
	result, ok := value.(float64)
	require.Truef(t, ok, "expected float64 at %v, got %T", path, value)
	return result
}

func nestedValue(t *testing.T, root map[string]any, path ...string) any {
	t.Helper()

	var current any = root
	for _, key := range path {
		switch typed := current.(type) {
		case map[string]any:
			current = typed[key]
		case []any:
			require.Len(t, key, 1)
			index := int(key[0] - '0')
			require.GreaterOrEqual(t, index, 0)
			require.Less(t, index, len(typed))
			current = typed[index]
		default:
			t.Fatalf("cannot descend into %T at %q in path %v", current, key, path)
		}
		require.NotNilf(t, current, "missing value at %q in path %v", key, path)
	}
	return current
}
