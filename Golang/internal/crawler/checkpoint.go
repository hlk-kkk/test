// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/crawler/checkpoint.go
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

// internal/crawler/checkpoint.go
package crawler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Checkpoint 断点数据结构
type Checkpoint struct {
	ID             string            `json:"id"`
	Platform       string            `json:"platform"`
	CrawlerType    string            `json:"crawler_type"`
	Keyword        string            `json:"keyword,omitempty"`
	CreatorID      string            `json:"creator_id,omitempty"`
	CurrentPage    int               `json:"current_page"`
	Cursor         string            `json:"cursor,omitempty"` // 游标分页(用于Creator/Homefeed)
	TotalCrawled   int               `json:"total_crawled"`
	CrawledNoteIDs []string          `json:"crawled_note_ids,omitempty"`
	UpdatedAt      int64             `json:"updated_at"`
	Extra          map[string]string `json:"extra,omitempty"`
}

// CheckpointManager 断点续爬管理器
type CheckpointManager struct {
	baseDir         string
	platform        string
	legacyPlatforms []string
	mu              sync.RWMutex
}

// NewCheckpointManager 创建断点管理器
func NewCheckpointManager(platform string, baseDir string, legacyPlatforms ...string) *CheckpointManager {
	if baseDir == "" {
		baseDir = "data/checkpoints"
	}
	// 确保目录存在
	os.MkdirAll(baseDir, 0755)
	return &CheckpointManager{
		baseDir:         baseDir,
		platform:        platform,
		legacyPlatforms: legacyPlatforms,
	}
}

// generateID 生成检查点ID
func (m *CheckpointManager) generateID(crawlerType, key string) string {
	timestamp := time.Now().Format("20060102150405")
	if key != "" {
		return fmt.Sprintf("%s_%s_%s_%s", m.platform, crawlerType, key, timestamp)
	}
	return fmt.Sprintf("%s_%s_%s", m.platform, crawlerType, timestamp)
}

// getFilePath 获取检查点文件路径
func (m *CheckpointManager) getFilePath(id string) string {
	return filepath.Join(m.baseDir, id+".json")
}

func (m *CheckpointManager) checkpointKey(cp *Checkpoint) string {
	if cp.Keyword != "" {
		return cp.Keyword
	}
	if cp.CreatorID != "" {
		return cp.CreatorID
	}
	return ""
}

// Save 保存断点
func (m *CheckpointManager) Save(cp *Checkpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cp.ID == "" {
		cp.ID = m.generateID(cp.CrawlerType, m.checkpointKey(cp))
	}
	cp.UpdatedAt = time.Now().Unix()
	cp.Platform = m.platform

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal checkpoint failed: %w", err)
	}

	return os.WriteFile(m.getFilePath(cp.ID), data, 0644)
}

// SaveByID 使用指定ID保存断点(用于保存特殊checkpoint如"search_latest")
func (m *CheckpointManager) SaveByID(id string, cp *Checkpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cp.ID = id
	cp.UpdatedAt = time.Now().Unix()
	cp.Platform = m.platform

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal checkpoint failed: %w", err)
	}

	return os.WriteFile(m.getFilePath(id), data, 0644)
}

// Load 加载断点（按crawlerType和key查找最新的）
func (m *CheckpointManager) Load(crawlerType, key string) (*Checkpoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	matches := make([]string, 0)
	for _, platform := range m.loadPlatforms() {
		pattern := filepath.Join(m.baseDir, fmt.Sprintf("%s_%s_%s_*.json", platform, crawlerType, key))
		if key == "" {
			pattern = filepath.Join(m.baseDir, fmt.Sprintf("%s_%s_*.json", platform, crawlerType))
		}

		platformMatches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		matches = append(matches, platformMatches...)
	}

	if len(matches) == 0 {
		return nil, nil
	}

	// 找最新的文件
	var latestFile string
	var latestTime time.Time
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestFile = match
		}
	}

	if latestFile == "" {
		return nil, nil
	}

	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, err
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("unmarshal checkpoint failed: %w", err)
	}

	return &cp, nil
}

func (m *CheckpointManager) loadPlatforms() []string {
	platforms := make([]string, 0, 1+len(m.legacyPlatforms))
	platforms = append(platforms, m.platform)
	seen := map[string]struct{}{m.platform: {}}
	for _, platform := range m.legacyPlatforms {
		if platform == "" {
			continue
		}
		if _, ok := seen[platform]; ok {
			continue
		}
		platforms = append(platforms, platform)
		seen[platform] = struct{}{}
	}
	return platforms
}

// LoadByID 按ID加载断点
func (m *CheckpointManager) LoadByID(id string) (*Checkpoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := os.ReadFile(m.getFilePath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("unmarshal checkpoint failed: %w", err)
	}

	return &cp, nil
}

// Clear 清除断点
func (m *CheckpointManager) Clear(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	err := os.Remove(m.getFilePath(id))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// AddCrawledNote 添加已爬取的笔记ID
func (m *CheckpointManager) AddCrawledNote(cp *Checkpoint, noteID string) {
	for _, id := range cp.CrawledNoteIDs {
		if id == noteID {
			return
		}
	}
	cp.CrawledNoteIDs = append(cp.CrawledNoteIDs, noteID)
}

// IsNoteCrawled 检查笔记是否已爬取
func (m *CheckpointManager) IsNoteCrawled(cp *Checkpoint, noteID string) bool {
	if cp == nil {
		return false
	}
	for _, id := range cp.CrawledNoteIDs {
		if id == noteID {
			return true
		}
	}
	return false
}
