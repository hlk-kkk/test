// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/testsupport/signsrv/mock.go
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
	"net/http"
	"net/http/httptest"
)

func NewMockServer() *httptest.Server {
	return httptest.NewServer(NewHandler())
}

func NewHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/signsrv/pong":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			writeOK(w, map[string]string{"message": "pong"})
		case "/signsrv/v1/xhs/sign":
			writeSignResponse(w, r, map[string]string{
				"x_s":           "mock-x-s",
				"x_t":           "mock-x-t",
				"x_s_common":    "mock-x-s-common",
				"x_b3_trace_id": "mock-x-b3-trace-id",
			})
		case "/signsrv/v1/bilibili/sign":
			writeSignResponse(w, r, map[string]string{
				"wts":   "1710000000",
				"w_rid": "mock-w-rid",
			})
		case "/signsrv/v1/douyin/sign":
			writeSignResponse(w, r, map[string]string{
				"a_bogus": "mock-a-bogus",
			})
		case "/signsrv/v1/zhihu/sign":
			writeSignResponse(w, r, map[string]string{
				"x_zst_81": "mock-x-zst-81",
				"x_zse_96": "mock-x-zse-96",
			})
		default:
			http.NotFound(w, r)
		}
	})
}

func writeSignResponse(w http.ResponseWriter, r *http.Request, data map[string]string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeOK(w, data)
}

func writeOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"biz_code": 0,
		"msg":      "success",
		"isok":     true,
		"data":     data,
	})
}
