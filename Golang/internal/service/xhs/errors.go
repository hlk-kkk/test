// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/xhs/errors.go
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

package xhs

import (
	"errors"
	"fmt"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/constant"
)

// XhsError 小红书 API 错误类型
type XhsError struct {
	Code    int
	Message string
	Type    ErrorType
}

func (e *XhsError) Error() string {
	return fmt.Sprintf("XHS Error [%s]: %s (code: %d)", e.Type, e.Message, e.Code)
}

// ErrorType 错误类型枚举
type ErrorType string

const (
	// IPBlockError IP 被封禁
	IPBlockError ErrorType = "ip_block"

	// SignError 签名失败
	SignError ErrorType = "sign_error"

	// NoteSecreteError 笔记不可见
	NoteSecreteError ErrorType = "note_secrete"

	// AccessFrequencyError 访问频率异常
	AccessFrequencyError ErrorType = "access_frequency"

	// SessionExpiredError 会话过期
	SessionExpiredError ErrorType = "session_expired"

	// NeedVerifyError 需要验证码
	NeedVerifyError ErrorType = "need_verify"

	// UnknownError 未知错误
	UnknownError ErrorType = "unknown"

	// NetworkError 网络错误
	NetworkError ErrorType = "network"
)

// NewXhsError 创建 XhsError
func NewXhsError(code int, message string, errType ErrorType) *XhsError {
	return &XhsError{
		Code:    code,
		Message: message,
		Type:    errType,
	}
}

// IsRetryable 判断错误是否可以重试
func (e *XhsError) IsRetryable() bool {
	switch e.Type {
	case AccessFrequencyError, NetworkError, SignError:
		return true
	case IPBlockError, SessionExpiredError, NeedVerifyError, NoteSecreteError:
		// 这些错误需要切换账号或IP,但也可以重试
		return true
	default:
		return false
	}
}

// NeedSwitchAccount 判断是否需要切换账号
func (e *XhsError) NeedSwitchAccount() bool {
	switch e.Type {
	case SessionExpiredError, AccessFrequencyError, NeedVerifyError:
		return true
	default:
		return false
	}
}

// NeedSwitchIP 判断是否需要切换IP
func (e *XhsError) NeedSwitchIP() bool {
	return e.Type == IPBlockError
}

// ParseXhsError 根据错误码解析错误类型
func ParseXhsError(code int, message string) *XhsError {
	var errType ErrorType

	switch constant.ErrorEnum(code) {
	case constant.IpBlock:
		errType = IPBlockError
	case constant.NoteSecreteFault:
		errType = NoteSecreteError
	case constant.SignFault:
		errType = SignError
	case constant.SessionExpired:
		errType = SessionExpiredError
	case constant.AccessFrequencyError:
		errType = AccessFrequencyError
	default:
		errType = UnknownError
	}

	return NewXhsError(code, message, errType)
}

// ParseVerifyError 解析验证码错误
func ParseVerifyError(statusCode int, verifyType, verifyUUID string) *XhsError {
	message := fmt.Sprintf("需要验证码 (status: %d, type: %s, uuid: %s)",
		statusCode, verifyType, verifyUUID)
	return NewXhsError(statusCode, message, NeedVerifyError)
}

// WrapNetworkError 包装网络错误
func WrapNetworkError(err error) *XhsError {
	if err == nil {
		return nil
	}
	return NewXhsError(0, err.Error(), NetworkError)
}

// IsXhsError 判断是否是 XhsError
func IsXhsError(err error) (*XhsError, bool) {
	var xhsErr *XhsError
	if errors.As(err, &xhsErr) {
		return xhsErr, true
	}
	return nil, false
}
