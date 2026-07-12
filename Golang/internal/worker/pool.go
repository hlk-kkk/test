// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/worker/pool.go
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

// internal/worker/pool.go
package worker

import (
	"context"
	"sync"
)

// Task 任务函数类型
type Task func(ctx context.Context) error

// Pool 基于channel的worker池
type Pool struct {
	size   int
	logger interface {
		Info(args ...interface{})
		Error(args ...interface{})
	}
}

// NewPool 创建worker池
func NewPool(size int) *Pool {
	if size <= 0 {
		size = 5
	}
	return &Pool{
		size: size,
	}
}

// Submit 提交任务并等待全部完成
func (p *Pool) Submit(ctx context.Context, tasks []Task) []error {
	if len(tasks) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	taskCh := make(chan Task, len(tasks))
	errCh := make(chan error, len(tasks))

	// 启动workers
	for i := 0; i < p.size; i++ {
		go p.worker(ctx, &wg, taskCh, errCh)
	}

	// 提交任务
	for _, task := range tasks {
		wg.Add(1)
		taskCh <- task
	}
	close(taskCh)

	// 等待完成
	wg.Wait()
	close(errCh)

	// 收集错误
	return p.collectErrors(errCh)
}

// worker 工作协程
func (p *Pool) worker(ctx context.Context, wg *sync.WaitGroup, taskCh <-chan Task, errCh chan<- error) {
	for task := range taskCh {
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			wg.Done()
			return
		default:
			if err := task(ctx); err != nil {
				errCh <- err
			}
			wg.Done()
		}
	}
}

// collectErrors 收集错误
func (p *Pool) collectErrors(errCh <-chan error) []error {
	var errs []error
	for err := range errCh {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
