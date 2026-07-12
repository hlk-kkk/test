// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/models/ip_info.go
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

package models

import "time"

type IpInfoModel struct {
	Ip            string `json:"ip"`
	Port          int    `json:"port"`
	User          string `json:"user"`
	Protocol      string `json:"protocol"`
	Password      string `json:"password"`
	ExpiredTimeTs int64  `json:"expired_time_ts"`
}

// Expired check if the ip info is expired
// if current timestamp is greater than invalid_time_stamps, return true, else return false
func (a *IpInfoModel) Expired() bool {
	return a.ExpiredTimeTs < time.Now().Unix()
}

// FormatProxyToString get the proxy string, eg: http://user:password@ip:port
func (a *IpInfoModel) FormatProxyToString() string {
	return a.Protocol + "://" + a.User + ":" + a.Password + "@" + a.Ip + ":" + string(rune(a.Port))
}
