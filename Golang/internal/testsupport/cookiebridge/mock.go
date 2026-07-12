// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/testsupport/cookiebridge/mock.go
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
	"net/http/httptest"
	"strings"
)

type Account struct {
	ClientID  string
	Connected bool
	Nicknames map[string]string
	Cookies   map[string]string
}

func NewMockServer(accounts ...Account) *httptest.Server {
	return httptest.NewServer(NewHandler(accounts...))
}

func NewHandler(accounts ...Account) http.Handler {
	normalized := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.ClientID == "" {
			account.ClientID = "mock-client"
		}
		if account.Nicknames == nil {
			account.Nicknames = map[string]string{}
		}
		if account.Cookies == nil {
			account.Cookies = map[string]string{}
		}
		normalized = append(normalized, account)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/" || r.URL.Path == "/ping":
			writeOK(w, map[string]any{"message": "pong"})
		case r.URL.Path == "/api/status":
			writeOK(w, statusPayload(normalized))
		case r.URL.Path == "/api/accounts":
			writeOK(w, accountsPayload(normalized))
		case strings.HasPrefix(r.URL.Path, "/api/accounts/"):
			writeAccountDetail(w, normalized, strings.TrimPrefix(r.URL.Path, "/api/accounts/"))
		case r.URL.Path == "/api/cookies/all":
			writeAllCookies(w, normalized, r.URL.Query().Get("client_id"))
		case strings.HasPrefix(r.URL.Path, "/api/cookies/"):
			platform := strings.TrimPrefix(r.URL.Path, "/api/cookies/")
			writePlatformCookie(w, normalized, platform, r.URL.Query().Get("client_id"))
		default:
			http.NotFound(w, r)
		}
	})
}

func statusPayload(accounts []Account) map[string]any {
	clients := map[string]any{}
	connectedCount := 0
	for _, account := range accounts {
		if account.Connected {
			connectedCount++
		}
		clients[account.ClientID] = map[string]any{
			"connected": account.Connected,
			"nicknames": account.Nicknames,
			"last_seen": 1710000000,
			"platforms": platformStatus(account),
		}
	}
	return map[string]any{
		"extension_connected": connectedCount > 0,
		"connected_count":     connectedCount,
		"total_clients":       len(accounts),
		"clients":             clients,
	}
}

func accountsPayload(accounts []Account) map[string]any {
	result := make([]map[string]any, 0, len(accounts))
	for _, account := range accounts {
		result = append(result, map[string]any{
			"client_id":    account.ClientID,
			"connected":    account.Connected,
			"nicknames":    account.Nicknames,
			"platforms":    accountPlatforms(account),
			"connected_at": 1710000000,
			"last_seen":    1710000000,
		})
	}
	return map[string]any{"accounts": result, "total": len(result)}
}

func writeAccountDetail(w http.ResponseWriter, accounts []Account, clientID string) {
	for _, account := range accounts {
		if account.ClientID == clientID {
			writeOK(w, map[string]any{
				"client_id":    account.ClientID,
				"connected":    account.Connected,
				"nicknames":    account.Nicknames,
				"platforms":    accountPlatforms(account),
				"connected_at": 1710000000,
				"last_seen":    1710000000,
			})
			return
		}
	}
	writeError(w, http.StatusOK, 103, "Account Not Found")
}

func writeAllCookies(w http.ResponseWriter, accounts []Account, clientID string) {
	platforms := map[string]any{}
	for _, account := range accounts {
		if clientID != "" && account.ClientID != clientID {
			continue
		}
		for platform, cookies := range account.Cookies {
			if _, exists := platforms[platform]; exists || cookies == "" {
				continue
			}
			platforms[platform] = cookiePayload(platform, account.ClientID, cookies)
		}
		if clientID != "" {
			break
		}
	}
	writeOK(w, map[string]any{"platforms": platforms})
}

func writePlatformCookie(w http.ResponseWriter, accounts []Account, platform, clientID string) {
	for _, account := range accounts {
		if clientID != "" && account.ClientID != clientID {
			continue
		}
		cookies := account.Cookies[platform]
		if cookies == "" {
			continue
		}
		writeOK(w, cookiePayload(platform, account.ClientID, cookies))
		return
	}
	writeError(w, http.StatusOK, 103, "Cookies Not Found")
}

func accountPlatforms(account Account) map[string]map[string]bool {
	platforms := make(map[string]map[string]bool, len(account.Cookies))
	for platform, cookies := range account.Cookies {
		platforms[platform] = map[string]bool{"has_cookies": cookies != ""}
	}
	return platforms
}

func platformStatus(account Account) map[string]any {
	platforms := make(map[string]any, len(account.Cookies))
	for platform, cookies := range account.Cookies {
		platforms[platform] = map[string]any{
			"has_cookies": cookies != "",
			"source":      "mock",
			"fresh":       true,
			"age_seconds": 0,
		}
	}
	return platforms
}

func cookiePayload(platform, clientID, cookies string) map[string]any {
	return map[string]any{
		"platform":  platform,
		"cookies":   cookies,
		"source":    "mock",
		"client_id": clientID,
	}
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, map[string]any{
		"biz_code": 0,
		"msg":      "OK!",
		"isok":     true,
		"data":     data,
	})
}

func writeError(w http.ResponseWriter, statusCode, bizCode int, msg string) {
	writeJSON(w, statusCode, map[string]any{
		"biz_code": bizCode,
		"msg":      msg,
		"isok":     false,
		"extra":    map[string]any{},
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
