// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/rpc/signsrv/model.go
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

type XhsSignResult struct {
	Xs         string `json:"x_s"`
	Xt         string `json:"x_t"`
	XsCommon   string `json:"x_s_common"`
	XB3TraceId string `json:"x_b3_trace_id"`
}

type XhsSignRequest struct {
	Uri     string `json:"uri"`
	Data    any    `json:"data"`
	Cookies string `json:"cookies"`
}

type XhsSignResponse struct {
	BizCode int            `json:"biz_code"`
	Msg     string         `json:"msg"`
	IsOk    bool           `json:"isok"`
	Result  *XhsSignResult `json:"data"`
}

type BilibiliSignResult struct {
	Wts  string `json:"wts"`
	WRid string `json:"w_rid"`
}

type BilibiliSignRequest struct {
	ReqData map[string]string `json:"req_data"`
	Cookies string            `json:"cookies"`
}

type BilibiliSignResponse struct {
	BizCode int                 `json:"biz_code"`
	Msg     string              `json:"msg"`
	IsOk    bool                `json:"isok"`
	Result  *BilibiliSignResult `json:"data"`
}

type DouyinSignResult struct {
	ABogus string `json:"a_bogus"`
}

type DouyinSignRequest struct {
	Uri         string `json:"uri"`
	QueryParams string `json:"query_params"`
	UserAgent   string `json:"user_agent"`
	Cookies     string `json:"cookies"`
}

type DouyinSignResponse struct {
	BizCode int               `json:"biz_code"`
	Msg     string            `json:"msg"`
	IsOk    bool              `json:"isok"`
	Result  *DouyinSignResult `json:"data"`
}

type ZhihuSignResult struct {
	XZst81 string `json:"x_zst_81"`
	XZse96 string `json:"x_zse_96"`
}

type ZhihuSignRequest struct {
	Uri     string `json:"uri"`
	Cookies string `json:"cookies"`
}

type ZhihuSignResponse struct {
	BizCode int              `json:"biz_code"`
	Msg     string           `json:"msg"`
	IsOk    bool             `json:"isok"`
	Result  *ZhihuSignResult `json:"data"`
}
