// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/service/xhs/client_test.go
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

// internal/service/xhs/client_test.go
package xhs

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientUsesConfiguredEndpointURLs(t *testing.T) {
	cfg := &config.Config{
		PlatformEndpoints: &config.PlatformEndpointsConfig{
			XhsApiURL:   "http://127.0.0.1:18081",
			XhsIndexURL: "http://127.0.0.1:18082",
		},
	}
	cfg.ApplyDefaults()
	client := &Client{config: cfg}

	require.Equal(t, "http://127.0.0.1:18081/api/sns/web/v1/search/notes", client.apiURL("/api/sns/web/v1/search/notes"))
	require.Equal(t, "http://127.0.0.1:18082/explore/note1", client.indexURL("/explore/note1"))
}

func TestClientValidateXhsRespRejectsSuccessFalseOnStatusOK(t *testing.T) {
	client := &Client{logger: tools.GetLogger()}
	resp := &http.Response{StatusCode: http.StatusOK}
	body := []byte(`{"code":9999,"success":false,"msg":"business failed","data":{}}`)

	err := client.validateXhsResp(nil, resp, body, false)

	require.Error(t, err)
	xhsErr, ok := IsXhsError(err)
	require.True(t, ok)
	require.Equal(t, UnknownError, xhsErr.Type)
	require.Equal(t, 9999, xhsErr.Code)
}

// TestWithRetry_Success 测试重试成功场景
func TestWithRetry_Success(t *testing.T) {
	t.Run("第一次就成功", func(t *testing.T) {
		ctx := context.Background()
		callCount := 0

		fn := func(ctx context.Context) (interface{}, error) {
			callCount++
			return "success", nil
		}

		result, err := WithRetry(ctx, fn, DefaultRetryConfig())

		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, 1, callCount, "应该只调用一次")
	})

	t.Run("第二次重试成功", func(t *testing.T) {
		ctx := context.Background()
		callCount := 0

		fn := func(ctx context.Context) (interface{}, error) {
			callCount++
			if callCount == 1 {
				// 第一次失败,返回可重试错误
				return nil, NewXhsError(0, "temporary error", NetworkError)
			}
			return "success", nil
		}

		config := DefaultRetryConfig()
		config.MinDelay = 10 * time.Millisecond
		config.MaxDelay = 20 * time.Millisecond

		result, err := WithRetry(ctx, fn, config)

		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, 2, callCount, "应该调用两次")
	})

	t.Run("最后一次重试成功", func(t *testing.T) {
		ctx := context.Background()
		callCount := 0
		maxRetries := 3

		fn := func(ctx context.Context) (interface{}, error) {
			callCount++
			if callCount <= maxRetries {
				return nil, NewXhsError(0, "temporary error", NetworkError)
			}
			return "success", nil
		}

		config := DefaultRetryConfig()
		config.MaxRetries = maxRetries
		config.MinDelay = 10 * time.Millisecond
		config.MaxDelay = 20 * time.Millisecond

		result, err := WithRetry(ctx, fn, config)

		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, maxRetries+1, callCount)
	})
}

// TestWithRetry_MaxRetriesExceeded 测试超过最大重试次数
func TestWithRetry_MaxRetriesExceeded(t *testing.T) {
	t.Run("所有重试都失败", func(t *testing.T) {
		ctx := context.Background()
		callCount := 0
		maxRetries := 3

		fn := func(ctx context.Context) (interface{}, error) {
			callCount++
			return nil, NewXhsError(0, "always fail", NetworkError)
		}

		config := DefaultRetryConfig()
		config.MaxRetries = maxRetries
		config.MinDelay = 10 * time.Millisecond
		config.MaxDelay = 20 * time.Millisecond

		result, err := WithRetry(ctx, fn, config)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, maxRetries+1, callCount, "应该调用 maxRetries+1 次")

		// 验证返回的是 XhsError
		xhsErr, ok := IsXhsError(err)
		assert.True(t, ok, "应该返回 XhsError")
		assert.Equal(t, NetworkError, xhsErr.Type)
	})

	t.Run("达到最大重试次数后立即返回", func(t *testing.T) {
		ctx := context.Background()
		maxRetries := 2

		fn := func(ctx context.Context) (interface{}, error) {
			return nil, NewXhsError(0, "error", NetworkError)
		}

		config := DefaultRetryConfig()
		config.MaxRetries = maxRetries
		config.MinDelay = 10 * time.Millisecond
		config.MaxDelay = 20 * time.Millisecond

		start := time.Now()
		_, err := WithRetry(ctx, fn, config)
		elapsed := time.Since(start)

		assert.Error(t, err)
		// 验证总时间大约是 2 次延迟(10-20ms * 2)
		assert.Less(t, elapsed, 100*time.Millisecond, "超过最大重试后应立即返回")
	})
}

// TestWithRetry_NonRetryableError 测试不可重试错误
func TestWithRetry_NonRetryableError(t *testing.T) {
	t.Run("UnknownError不重试", func(t *testing.T) {
		ctx := context.Background()
		callCount := 0

		fn := func(ctx context.Context) (interface{}, error) {
			callCount++
			return nil, NewXhsError(999, "unknown error", UnknownError)
		}

		config := DefaultRetryConfig()
		config.MinDelay = 10 * time.Millisecond
		config.MaxDelay = 20 * time.Millisecond

		result, err := WithRetry(ctx, fn, config)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, 1, callCount, "不可重试错误应该只调用一次")
	})

	t.Run("普通error不重试", func(t *testing.T) {
		ctx := context.Background()
		callCount := 0

		fn := func(ctx context.Context) (interface{}, error) {
			callCount++
			return nil, errors.New("regular error")
		}

		result, err := WithRetry(ctx, fn, DefaultRetryConfig())

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, 1, callCount, "非XhsError应该只调用一次")
	})
}

// TestWithRetry_ContextCancellation 测试上下文取消
func TestWithRetry_ContextCancellation(t *testing.T) {
	t.Run("延迟期间取消上下文", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		callCount := 0

		fn := func(ctx context.Context) (interface{}, error) {
			callCount++
			if callCount == 1 {
				// 第一次失败后,在延迟期间取消上下文
				go func() {
					time.Sleep(50 * time.Millisecond)
					cancel()
				}()
				return nil, NewXhsError(0, "error", NetworkError)
			}
			return "success", nil
		}

		config := DefaultRetryConfig()
		config.MinDelay = 1 * time.Second
		config.MaxDelay = 2 * time.Second

		result, err := WithRetry(ctx, fn, config)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, context.Canceled, err)
		assert.Equal(t, 1, callCount, "上下文取消后不应该重试")
	})
}

// TestWithRetry_Callbacks 测试回调函数
func TestWithRetry_Callbacks(t *testing.T) {
	t.Run("OnRetry回调", func(t *testing.T) {
		ctx := context.Background()
		var retryAttempts []int
		var retryErrors []error

		fn := func(ctx context.Context) (interface{}, error) {
			if len(retryAttempts) < 2 {
				return nil, NewXhsError(0, "error", NetworkError)
			}
			return "success", nil
		}

		config := DefaultRetryConfig()
		config.MinDelay = 10 * time.Millisecond
		config.MaxDelay = 20 * time.Millisecond
		config.OnRetry = func(attempt int, err error) {
			retryAttempts = append(retryAttempts, attempt)
			retryErrors = append(retryErrors, err)
		}

		result, err := WithRetry(ctx, fn, config)

		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, []int{1, 2}, retryAttempts, "应该记录重试次数")
		assert.Len(t, retryErrors, 2, "应该记录重试错误")
	})

	t.Run("BeforeRetry回调", func(t *testing.T) {
		ctx := context.Background()
		var beforeRetryAttempts []int

		fn := func(ctx context.Context) (interface{}, error) {
			if len(beforeRetryAttempts) < 2 {
				return nil, NewXhsError(0, "error", NetworkError)
			}
			return "success", nil
		}

		config := DefaultRetryConfig()
		config.MinDelay = 10 * time.Millisecond
		config.MaxDelay = 20 * time.Millisecond
		config.BeforeRetry = func(attempt int) error {
			beforeRetryAttempts = append(beforeRetryAttempts, attempt)
			return nil
		}

		result, err := WithRetry(ctx, fn, config)

		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, []int{1, 2}, beforeRetryAttempts)
	})

	t.Run("BeforeRetry返回错误", func(t *testing.T) {
		ctx := context.Background()

		fn := func(ctx context.Context) (interface{}, error) {
			return nil, NewXhsError(0, "error", NetworkError)
		}

		config := DefaultRetryConfig()
		config.MinDelay = 10 * time.Millisecond
		config.MaxDelay = 20 * time.Millisecond
		config.BeforeRetry = func(attempt int) error {
			return errors.New("before retry failed")
		}

		result, err := WithRetry(ctx, fn, config)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "before retry failed", err.Error())
	})
}

// TestWithRetry_CustomConfig 测试自定义配置
func TestWithRetry_CustomConfig(t *testing.T) {
	t.Run("自定义重试次数", func(t *testing.T) {
		ctx := context.Background()
		callCount := 0

		fn := func(ctx context.Context) (interface{}, error) {
			callCount++
			return nil, NewXhsError(0, "error", NetworkError)
		}

		config := &RetryConfig{
			MaxRetries: 2,
			MinDelay:   10 * time.Millisecond,
			MaxDelay:   20 * time.Millisecond,
			ShouldRetry: func(err error) bool {
				return true
			},
		}

		_, err := WithRetry(ctx, fn, config)

		assert.Error(t, err)
		assert.Equal(t, 3, callCount, "MaxRetries=2 应该调用3次")
	})

	t.Run("自定义ShouldRetry", func(t *testing.T) {
		ctx := context.Background()
		callCount := 0

		fn := func(ctx context.Context) (interface{}, error) {
			callCount++
			return nil, errors.New("custom error")
		}

		config := &RetryConfig{
			MaxRetries: 5,
			MinDelay:   10 * time.Millisecond,
			MaxDelay:   20 * time.Millisecond,
			ShouldRetry: func(err error) bool {
				return err.Error() == "custom error"
			},
		}

		_, err := WithRetry(ctx, fn, config)

		assert.Error(t, err)
		assert.Greater(t, callCount, 1, "自定义ShouldRetry应该允许重试")
	})
}

// TestRandomDelay 测试随机延迟
func TestRandomDelay(t *testing.T) {
	t.Run("延迟在范围内", func(t *testing.T) {
		min := 100 * time.Millisecond
		max := 200 * time.Millisecond

		for i := 0; i < 10; i++ {
			delay := randomDelay(min, max)
			assert.GreaterOrEqual(t, delay, min, "延迟应该 >= 最小值")
			assert.LessOrEqual(t, delay, max, "延迟应该 <= 最大值")
		}
	})

	t.Run("min >= max 时返回min", func(t *testing.T) {
		min := 200 * time.Millisecond
		max := 100 * time.Millisecond

		delay := randomDelay(min, max)
		assert.Equal(t, min, delay)
	})

	t.Run("min == max 时返回相同值", func(t *testing.T) {
		value := 150 * time.Millisecond

		delay := randomDelay(value, value)
		assert.Equal(t, value, delay)
	})
}

// TestDefaultRetryConfig 测试默认配置
func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, MaxRetries, config.MaxRetries)
	assert.Equal(t, MinRetryDelay*time.Second, config.MinDelay)
	assert.Equal(t, MaxRetryDelay*time.Second, config.MaxDelay)
	assert.NotNil(t, config.ShouldRetry)

	// 测试默认ShouldRetry行为
	t.Run("默认ShouldRetry - 可重试错误", func(t *testing.T) {
		retryableErrors := []*XhsError{
			NewXhsError(0, "network", NetworkError),
			NewXhsError(0, "sign", SignError),
			NewXhsError(0, "frequency", AccessFrequencyError),
		}

		for _, err := range retryableErrors {
			assert.True(t, config.ShouldRetry(err),
				"错误类型 %s 应该可重试", err.Type)
		}
	})

	t.Run("默认ShouldRetry - 普通错误", func(t *testing.T) {
		err := errors.New("regular error")
		assert.False(t, config.ShouldRetry(err), "普通错误不应该重试")
	})
}

// BenchmarkWithRetry 重试机制性能测试
func BenchmarkWithRetry(b *testing.B) {
	ctx := context.Background()

	b.Run("无重试-成功", func(b *testing.B) {
		fn := func(ctx context.Context) (interface{}, error) {
			return "success", nil
		}

		config := DefaultRetryConfig()
		config.MinDelay = 1 * time.Millisecond
		config.MaxDelay = 2 * time.Millisecond

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			WithRetry(ctx, fn, config)
		}
	})

	b.Run("一次重试", func(b *testing.B) {
		fn := func(ctx context.Context) (interface{}, error) {
			return nil, NewXhsError(0, "error", NetworkError)
		}

		config := DefaultRetryConfig()
		config.MaxRetries = 1
		config.MinDelay = 1 * time.Millisecond
		config.MaxDelay = 2 * time.Millisecond

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			WithRetry(ctx, fn, config)
		}
	})
}
