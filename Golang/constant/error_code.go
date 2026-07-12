// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/constant/error_code.go
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

package constant

// ErrorEnum 错误码枚举
type ErrorEnum int

const (
	// IpBlock 网络连接异常，请检查网络设置或重启试试
	IpBlock ErrorEnum = 300012

	// NoteSecreteFault 当前内容无法展示
	NoteSecreteFault ErrorEnum = -510001

	// SignFault 浏览器异常，请尝试关闭/卸载风险插件或重启试试！
	SignFault ErrorEnum = 300015

	// SessionExpired 登录已过期
	SessionExpired ErrorEnum = -100

	// AccessFrequencyError 访问频次异常，请勿频繁操作或重启试试
	AccessFrequencyError ErrorEnum = 300013
)
