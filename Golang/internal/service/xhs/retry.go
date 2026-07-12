// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/xhs/retry.go
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
	"context"
	"math/rand"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
)

const (
	// MaxRetries 最大重试次数
	MaxRetries = 5

	// MinRetryDelay 最小重试延迟(秒)
	MinRetryDelay = 2

	// MaxRetryDelay 最大重试延迟(秒)
	MaxRetryDelay = 10
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries  int
	MinDelay    time.Duration
	MaxDelay    time.Duration
	OnRetry     func(attempt int, err error)
	ShouldRetry func(err error) bool
	BeforeRetry func(attempt int) error
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: MaxRetries,
		MinDelay:   MinRetryDelay * time.Second,
		MaxDelay:   MaxRetryDelay * time.Second,
		ShouldRetry: func(err error) bool {
			if xhsErr, ok := IsXhsError(err); ok {
				return xhsErr.IsRetryable()
			}
			return false
		},
	}
}

// RetryFunc 可重试的函数类型
type RetryFunc func(ctx context.Context) (interface{}, error)

// WithRetry 重试装饰器
func WithRetry(ctx context.Context, fn RetryFunc, config *RetryConfig) (interface{}, error) {
	logger := tools.GetLogger()

	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// 如果不是第一次尝试,先执行延迟
		if attempt > 0 {
			// 随机延迟时间,避免请求过于规律
			delay := randomDelay(config.MinDelay, config.MaxDelay)
			logger.Info("[WithRetry] Attempt ", attempt, " failed, retrying after ", delay)

			// 调用重试回调
			if config.OnRetry != nil {
				config.OnRetry(attempt, lastErr)
			}

			// 重试前的预处理
			if config.BeforeRetry != nil {
				if err := config.BeforeRetry(attempt); err != nil {
					logger.Error("[WithRetry] BeforeRetry error: ", err)
					return nil, err
				}
			}

			// 等待延迟
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		// 执行函数
		result, err := fn(ctx)
		if err == nil {
			if attempt > 0 {
				logger.Info("[WithRetry] Request succeeded after ", attempt, " retries")
			}
			return result, nil
		}

		lastErr = err

		// 判断是否应该重试
		if config.ShouldRetry != nil && !config.ShouldRetry(err) {
			logger.Warn("[WithRetry] Error is not retryable: ", err)
			return nil, err
		}

		// 如果是最后一次尝试,直接返回错误
		if attempt >= config.MaxRetries {
			logger.Error("[WithRetry] Max retries (", config.MaxRetries, ") exceeded, last error: ", err)
			return nil, err
		}

		logger.Warn("[WithRetry] Attempt ", attempt+1, " failed: ", err)
	}

	return nil, lastErr
}

// randomDelay 生成随机延迟时间
func randomDelay(min, max time.Duration) time.Duration {
	if min >= max {
		return min
	}
	delta := max - min
	return min + time.Duration(rand.Int63n(int64(delta)))
}
