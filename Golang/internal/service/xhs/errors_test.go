// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/xhs/errors_test.go
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

// internal/service/xhs/errors_test.go
package xhs

import (
	"errors"
	"testing"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/constant"
	"github.com/stretchr/testify/assert"
)

// TestXhsError_Error 测试错误消息格式
func TestXhsError_Error(t *testing.T) {
	err := NewXhsError(401, "Unauthorized", SessionExpiredError)
	expected := "XHS Error [session_expired]: Unauthorized (code: 401)"
	assert.Equal(t, expected, err.Error())
}

// TestXhsError_IsRetryable 测试错误是否可重试
func TestXhsError_IsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		errType   ErrorType
		retryable bool
	}{
		{"NetworkError可重试", NetworkError, true},
		{"SignError可重试", SignError, true},
		{"AccessFrequencyError可重试", AccessFrequencyError, true},
		{"IPBlockError可重试", IPBlockError, true},
		{"SessionExpiredError可重试", SessionExpiredError, true},
		{"NeedVerifyError可重试", NeedVerifyError, true},
		{"NoteSecreteError可重试", NoteSecreteError, true},
		{"UnknownError不可重试", UnknownError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewXhsError(0, "test", tt.errType)
			assert.Equal(t, tt.retryable, err.IsRetryable())
		})
	}
}

// TestXhsError_NeedSwitchAccount 测试是否需要切换账号
func TestXhsError_NeedSwitchAccount(t *testing.T) {
	tests := []struct {
		name       string
		errType    ErrorType
		needSwitch bool
	}{
		{"SessionExpiredError需要切换", SessionExpiredError, true},
		{"AccessFrequencyError需要切换", AccessFrequencyError, true},
		{"NeedVerifyError需要切换", NeedVerifyError, true},
		{"IPBlockError不需要切换账号", IPBlockError, false},
		{"NetworkError不需要切换账号", NetworkError, false},
		{"SignError不需要切换账号", SignError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewXhsError(0, "test", tt.errType)
			assert.Equal(t, tt.needSwitch, err.NeedSwitchAccount())
		})
	}
}

// TestXhsError_NeedSwitchIP 测试是否需要切换IP
func TestXhsError_NeedSwitchIP(t *testing.T) {
	tests := []struct {
		name       string
		errType    ErrorType
		needSwitch bool
	}{
		{"IPBlockError需要切换IP", IPBlockError, true},
		{"SessionExpiredError不需要切换IP", SessionExpiredError, false},
		{"NetworkError不需要切换IP", NetworkError, false},
		{"SignError不需要切换IP", SignError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewXhsError(0, "test", tt.errType)
			assert.Equal(t, tt.needSwitch, err.NeedSwitchIP())
		})
	}
}

// TestParseXhsError 测试错误码解析
func TestParseXhsError(t *testing.T) {
	tests := []struct {
		name         string
		code         int
		message      string
		expectedType ErrorType
	}{
		{
			name:         "IP被封",
			code:         int(constant.IpBlock),
			message:      "IP blocked",
			expectedType: IPBlockError,
		},
		{
			name:         "签名错误",
			code:         int(constant.SignFault),
			message:      "Sign failed",
			expectedType: SignError,
		},
		{
			name:         "会话过期",
			code:         int(constant.SessionExpired),
			message:      "Session expired",
			expectedType: SessionExpiredError,
		},
		{
			name:         "访问频率异常",
			code:         int(constant.AccessFrequencyError),
			message:      "Access too frequently",
			expectedType: AccessFrequencyError,
		},
		{
			name:         "笔记不可见",
			code:         int(constant.NoteSecreteFault),
			message:      "Note is private",
			expectedType: NoteSecreteError,
		},
		{
			name:         "未知错误",
			code:         999,
			message:      "Unknown error",
			expectedType: UnknownError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseXhsError(tt.code, tt.message)
			assert.Equal(t, tt.code, err.Code)
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.expectedType, err.Type)
		})
	}
}

// TestParseVerifyError 测试验证码错误解析
func TestParseVerifyError(t *testing.T) {
	err := ParseVerifyError(461, "slider", "uuid-123")

	assert.Equal(t, 461, err.Code)
	assert.Equal(t, NeedVerifyError, err.Type)
	assert.Contains(t, err.Message, "需要验证码")
	assert.Contains(t, err.Message, "461")
	assert.Contains(t, err.Message, "slider")
	assert.Contains(t, err.Message, "uuid-123")
}

// TestWrapNetworkError 测试网络错误包装
func TestWrapNetworkError(t *testing.T) {
	t.Run("包装非nil错误", func(t *testing.T) {
		originalErr := errors.New("connection timeout")
		xhsErr := WrapNetworkError(originalErr)

		assert.NotNil(t, xhsErr)
		assert.Equal(t, NetworkError, xhsErr.Type)
		assert.Equal(t, "connection timeout", xhsErr.Message)
		assert.Equal(t, 0, xhsErr.Code)
	})

	t.Run("包装nil错误", func(t *testing.T) {
		xhsErr := WrapNetworkError(nil)
		assert.Nil(t, xhsErr)
	})
}

// TestIsXhsError 测试错误类型判断
func TestIsXhsError(t *testing.T) {
	t.Run("是XhsError", func(t *testing.T) {
		originalErr := NewXhsError(401, "Unauthorized", SessionExpiredError)

		xhsErr, ok := IsXhsError(originalErr)
		assert.True(t, ok)
		assert.NotNil(t, xhsErr)
		assert.Equal(t, SessionExpiredError, xhsErr.Type)
	})

	t.Run("不是XhsError", func(t *testing.T) {
		originalErr := errors.New("regular error")

		xhsErr, ok := IsXhsError(originalErr)
		assert.False(t, ok)
		assert.Nil(t, xhsErr)
	})

	t.Run("nil错误", func(t *testing.T) {
		xhsErr, ok := IsXhsError(nil)
		assert.False(t, ok)
		assert.Nil(t, xhsErr)
	})
}

// TestErrorTypeConstants 测试错误类型常量
func TestErrorTypeConstants(t *testing.T) {
	// 确保错误类型常量正确定义
	assert.Equal(t, ErrorType("ip_block"), IPBlockError)
	assert.Equal(t, ErrorType("sign_error"), SignError)
	assert.Equal(t, ErrorType("note_secrete"), NoteSecreteError)
	assert.Equal(t, ErrorType("access_frequency"), AccessFrequencyError)
	assert.Equal(t, ErrorType("session_expired"), SessionExpiredError)
	assert.Equal(t, ErrorType("need_verify"), NeedVerifyError)
	assert.Equal(t, ErrorType("unknown"), UnknownError)
	assert.Equal(t, ErrorType("network"), NetworkError)
}

// TestErrorChaining 测试错误链
func TestErrorChaining(t *testing.T) {
	t.Run("errors.As识别XhsError", func(t *testing.T) {
		originalErr := NewXhsError(401, "Unauthorized", SessionExpiredError)

		var xhsErr *XhsError
		ok := errors.As(originalErr, &xhsErr)

		assert.True(t, ok)
		assert.Equal(t, SessionExpiredError, xhsErr.Type)
	})

	t.Run("errors.Is不适用于XhsError", func(t *testing.T) {
		err1 := NewXhsError(401, "Unauthorized", SessionExpiredError)
		err2 := NewXhsError(401, "Unauthorized", SessionExpiredError)

		// XhsError 没有实现 Is 方法,所以这里应该返回 false
		assert.False(t, errors.Is(err1, err2))
	})
}

// TestErrorScenarios 测试真实场景
func TestErrorScenarios(t *testing.T) {
	t.Run("场景1: IP被封 - 需要切换IP和重试", func(t *testing.T) {
		err := ParseXhsError(int(constant.IpBlock), "IP blocked")

		assert.True(t, err.IsRetryable(), "IP被封应该可重试")
		assert.True(t, err.NeedSwitchIP(), "IP被封需要切换IP")
		assert.False(t, err.NeedSwitchAccount(), "IP被封不需要切换账号")
	})

	t.Run("场景2: 会话过期 - 需要切换账号", func(t *testing.T) {
		err := ParseXhsError(int(constant.SessionExpired), "Session expired")

		assert.True(t, err.IsRetryable(), "会话过期应该可重试")
		assert.True(t, err.NeedSwitchAccount(), "会话过期需要切换账号")
		assert.False(t, err.NeedSwitchIP(), "会话过期不需要切换IP")
	})

	t.Run("场景3: 访问频率异常 - 需要切换账号和重试", func(t *testing.T) {
		err := ParseXhsError(int(constant.AccessFrequencyError), "Access too frequently")

		assert.True(t, err.IsRetryable(), "访问频率异常应该可重试")
		assert.True(t, err.NeedSwitchAccount(), "访问频率异常需要切换账号")
		assert.False(t, err.NeedSwitchIP(), "访问频率异常不需要切换IP")
	})

	t.Run("场景4: 签名错误 - 只需重试", func(t *testing.T) {
		err := ParseXhsError(int(constant.SignFault), "Sign failed")

		assert.True(t, err.IsRetryable(), "签名错误应该可重试")
		assert.False(t, err.NeedSwitchAccount(), "签名错误不需要切换账号")
		assert.False(t, err.NeedSwitchIP(), "签名错误不需要切换IP")
	})

	t.Run("场景5: 需要验证码 - 需要切换账号", func(t *testing.T) {
		err := ParseVerifyError(461, "slider", "uuid-123")

		assert.True(t, err.IsRetryable(), "需要验证码应该可重试")
		assert.True(t, err.NeedSwitchAccount(), "需要验证码需要切换账号")
		assert.False(t, err.NeedSwitchIP(), "需要验证码不需要切换IP")
	})

	t.Run("场景6: 网络错误 - 只需重试", func(t *testing.T) {
		err := WrapNetworkError(errors.New("connection timeout"))

		assert.True(t, err.IsRetryable(), "网络错误应该可重试")
		assert.False(t, err.NeedSwitchAccount(), "网络错误不需要切换账号")
		assert.False(t, err.NeedSwitchIP(), "网络错误不需要切换IP")
	})
}
