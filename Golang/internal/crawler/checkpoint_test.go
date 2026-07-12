// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/crawler/checkpoint_test.go
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

// internal/crawler/checkpoint_test.go
package crawler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestCheckpointManager 创建测试用的 CheckpointManager
func setupTestCheckpointManager(t *testing.T) (*CheckpointManager, string, func()) {
	// 创建临时目录
	tempDir := t.TempDir()

	manager := NewCheckpointManager("xhs", tempDir)

	cleanup := func() {
		// t.TempDir() 会自动清理,这里不需要额外操作
	}

	return manager, tempDir, cleanup
}

// TestNewCheckpointManager 测试创建管理器
func TestNewCheckpointManager(t *testing.T) {
	t.Run("默认base目录", func(t *testing.T) {
		manager := NewCheckpointManager("xhs", "")
		assert.Equal(t, "data/checkpoints", manager.baseDir)
		assert.Equal(t, "xhs", manager.platform)
	})

	t.Run("自定义base目录", func(t *testing.T) {
		tempDir := t.TempDir()
		manager := NewCheckpointManager("xhs", tempDir)
		assert.Equal(t, tempDir, manager.baseDir)
		assert.Equal(t, "xhs", manager.platform)
	})

	t.Run("创建目录", func(t *testing.T) {
		tempDir := t.TempDir()
		customDir := filepath.Join(tempDir, "custom", "checkpoints")

		manager := NewCheckpointManager("xhs", customDir)
		assert.Equal(t, customDir, manager.baseDir)

		// 验证目录已创建
		_, err := os.Stat(customDir)
		assert.NoError(t, err, "目录应该已创建")
	})
}

// TestCheckpointManager_Save 测试保存断点
func TestCheckpointManager_Save(t *testing.T) {
	manager, tempDir, cleanup := setupTestCheckpointManager(t)
	defer cleanup()

	t.Run("保存基本断点", func(t *testing.T) {
		cp := &Checkpoint{
			Platform:     "xhs",
			CrawlerType:  "search",
			Keyword:      "美妆",
			CurrentPage:  3,
			TotalCrawled: 60,
		}

		err := manager.Save(cp)
		require.NoError(t, err, "保存断点应该成功")

		// 验证ID已生成
		assert.NotEmpty(t, cp.ID, "ID应该被自动生成")

		// 验证UpdatedAt已设置
		assert.Greater(t, cp.UpdatedAt, int64(0), "UpdatedAt应该被设置")

		// 验证Platform已设置
		assert.Equal(t, "xhs", cp.Platform)

		// 验证文件已创建
		filePath := filepath.Join(tempDir, cp.ID+".json")
		_, err = os.Stat(filePath)
		assert.NoError(t, err, "断点文件应该存在")
	})

	t.Run("保存包含已爬笔记ID的断点", func(t *testing.T) {
		cp := &Checkpoint{
			Platform:       "xhs",
			CrawlerType:    "search",
			Keyword:        "护肤",
			CurrentPage:    2,
			TotalCrawled:   40,
			CrawledNoteIDs: []string{"note1", "note2", "note3"},
		}

		err := manager.Save(cp)
		require.NoError(t, err)

		// 加载验证
		loaded, err := manager.LoadByID(cp.ID)
		require.NoError(t, err)
		require.NotNil(t, loaded)

		assert.Equal(t, []string{"note1", "note2", "note3"}, loaded.CrawledNoteIDs)
	})

	t.Run("保存带Extra字段的断点", func(t *testing.T) {
		cp := &Checkpoint{
			Platform:    "xhs",
			CrawlerType: "creator",
			CreatorID:   "creator123",
			CurrentPage: 1,
			Extra: map[string]string{
				"sort_type": "latest",
				"filter":    "video",
			},
		}

		err := manager.Save(cp)
		require.NoError(t, err)

		// 加载验证
		loaded, err := manager.LoadByID(cp.ID)
		require.NoError(t, err)
		require.NotNil(t, loaded)

		assert.Equal(t, "latest", loaded.Extra["sort_type"])
		assert.Equal(t, "video", loaded.Extra["filter"])
	})

	t.Run("更新已有断点", func(t *testing.T) {
		cp := &Checkpoint{
			Platform:    "xhs",
			CrawlerType: "search",
			Keyword:     "测试",
			CurrentPage: 1,
		}

		// 第一次保存
		err := manager.Save(cp)
		require.NoError(t, err)
		firstID := cp.ID
		firstTime := cp.UpdatedAt

		// 等待一秒确保时间戳不同
		time.Sleep(1 * time.Second)

		// 更新页码
		cp.CurrentPage = 5

		// 第二次保存
		err = manager.Save(cp)
		require.NoError(t, err)

		// 验证ID相同
		assert.Equal(t, firstID, cp.ID, "更新时ID应该保持不变")

		// 验证UpdatedAt已更新
		assert.Greater(t, cp.UpdatedAt, firstTime, "UpdatedAt应该被更新")

		// 加载验证
		loaded, err := manager.LoadByID(cp.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, loaded.CurrentPage)
	})
}

// TestCheckpointManager_Load 测试加载断点
func TestCheckpointManager_Load(t *testing.T) {
	manager, _, cleanup := setupTestCheckpointManager(t)
	defer cleanup()

	t.Run("加载不存在的断点", func(t *testing.T) {
		cp, err := manager.Load("search", "不存在的关键词")
		assert.NoError(t, err)
		assert.Nil(t, cp, "不存在的断点应该返回nil")
	})

	t.Run("加载已存在的断点", func(t *testing.T) {
		// 先保存
		original := &Checkpoint{
			CrawlerType:  "search",
			Keyword:      "加载测试",
			CurrentPage:  3,
			TotalCrawled: 60,
		}
		err := manager.Save(original)
		require.NoError(t, err)

		// 加载
		loaded, err := manager.Load("search", "加载测试")
		require.NoError(t, err)
		require.NotNil(t, loaded)

		// 验证数据
		assert.Equal(t, "search", loaded.CrawlerType)
		assert.Equal(t, "加载测试", loaded.Keyword)
		assert.Equal(t, 3, loaded.CurrentPage)
		assert.Equal(t, 60, loaded.TotalCrawled)
	})

	t.Run("加载最新的断点", func(t *testing.T) {
		keyword := "多个断点"

		// 保存第一个断点
		cp1 := &Checkpoint{
			CrawlerType: "search",
			Keyword:     keyword,
			CurrentPage: 1,
		}
		err := manager.Save(cp1)
		require.NoError(t, err)

		// 等待确保时间戳不同
		time.Sleep(100 * time.Millisecond)

		// 保存第二个断点
		cp2 := &Checkpoint{
			CrawlerType: "search",
			Keyword:     keyword,
			CurrentPage: 5,
		}
		err = manager.Save(cp2)
		require.NoError(t, err)

		// 加载应该返回最新的
		loaded, err := manager.Load("search", keyword)
		require.NoError(t, err)
		require.NotNil(t, loaded)

		assert.Equal(t, 5, loaded.CurrentPage, "应该加载最新的断点")
	})

	t.Run("加载creator类型断点", func(t *testing.T) {
		cp := &Checkpoint{
			CrawlerType: "creator",
			CreatorID:   "creator_test",
			CurrentPage: 2,
		}
		err := manager.Save(cp)
		require.NoError(t, err)

		// 注意: Load方法使用key参数,对于creator类型应该传CreatorID
		loaded, err := manager.Load("creator", "creator_test")
		require.NoError(t, err)
		require.NotNil(t, loaded)

		assert.Equal(t, "creator", loaded.CrawlerType)
		assert.Equal(t, 2, loaded.CurrentPage)
	})
}

func TestCheckpointManager_LoadSearchesLegacyPlatformAliases(t *testing.T) {
	tempDir := t.TempDir()
	legacyManager := NewCheckpointManager("bilibili", tempDir)
	legacyCheckpoint := &Checkpoint{
		CrawlerType:  "search",
		Keyword:      "legacy-bili",
		CurrentPage:  3,
		TotalCrawled: 30,
	}
	require.NoError(t, legacyManager.Save(legacyCheckpoint))

	canonicalManager := NewCheckpointManager("bili", tempDir, "bilibili")
	loaded, err := canonicalManager.Load("search", "legacy-bili")

	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, legacyCheckpoint.ID, loaded.ID)
	assert.Equal(t, 3, loaded.CurrentPage)
}

// TestCheckpointManager_LoadByID 测试按ID加载
func TestCheckpointManager_LoadByID(t *testing.T) {
	manager, _, cleanup := setupTestCheckpointManager(t)
	defer cleanup()

	t.Run("加载已存在的ID", func(t *testing.T) {
		cp := &Checkpoint{
			CrawlerType: "search",
			Keyword:     "test_id",
			CurrentPage: 7,
		}
		err := manager.Save(cp)
		require.NoError(t, err)

		loaded, err := manager.LoadByID(cp.ID)
		require.NoError(t, err)
		require.NotNil(t, loaded)

		assert.Equal(t, cp.ID, loaded.ID)
		assert.Equal(t, 7, loaded.CurrentPage)
	})

	t.Run("加载不存在的ID", func(t *testing.T) {
		loaded, err := manager.LoadByID("nonexistent_id")
		assert.NoError(t, err)
		assert.Nil(t, loaded)
	})
}

// TestCheckpointManager_Clear 测试清除断点
func TestCheckpointManager_Clear(t *testing.T) {
	manager, tempDir, cleanup := setupTestCheckpointManager(t)
	defer cleanup()

	t.Run("清除已存在的断点", func(t *testing.T) {
		cp := &Checkpoint{
			CrawlerType: "search",
			Keyword:     "清除测试",
			CurrentPage: 3,
		}
		err := manager.Save(cp)
		require.NoError(t, err)

		// 验证文件存在
		filePath := filepath.Join(tempDir, cp.ID+".json")
		_, err = os.Stat(filePath)
		require.NoError(t, err)

		// 清除
		err = manager.Clear(cp.ID)
		assert.NoError(t, err)

		// 验证文件已删除
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err), "文件应该已被删除")

		// 加载应该返回nil
		loaded, err := manager.LoadByID(cp.ID)
		assert.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("清除不存在的断点", func(t *testing.T) {
		err := manager.Clear("nonexistent_id")
		assert.NoError(t, err, "清除不存在的断点应该不报错")
	})
}

// TestCheckpointManager_AddCrawledNote 测试添加已爬笔记
func TestCheckpointManager_AddCrawledNote(t *testing.T) {
	manager, _, cleanup := setupTestCheckpointManager(t)
	defer cleanup()

	t.Run("添加新笔记ID", func(t *testing.T) {
		cp := &Checkpoint{
			CrawlerType:    "search",
			Keyword:        "test",
			CrawledNoteIDs: []string{},
		}

		manager.AddCrawledNote(cp, "note1")
		assert.Equal(t, []string{"note1"}, cp.CrawledNoteIDs)

		manager.AddCrawledNote(cp, "note2")
		assert.Equal(t, []string{"note1", "note2"}, cp.CrawledNoteIDs)

		manager.AddCrawledNote(cp, "note3")
		assert.Equal(t, []string{"note1", "note2", "note3"}, cp.CrawledNoteIDs)
	})

	t.Run("添加重复的笔记ID", func(t *testing.T) {
		cp := &Checkpoint{
			CrawlerType:    "search",
			Keyword:        "test",
			CrawledNoteIDs: []string{"note1", "note2"},
		}

		// 添加已存在的
		manager.AddCrawledNote(cp, "note1")
		assert.Equal(t, []string{"note1", "note2"}, cp.CrawledNoteIDs, "不应该重复添加")

		// 添加新的
		manager.AddCrawledNote(cp, "note3")
		assert.Equal(t, []string{"note1", "note2", "note3"}, cp.CrawledNoteIDs)
	})

	t.Run("空的CrawledNoteIDs", func(t *testing.T) {
		cp := &Checkpoint{
			CrawlerType:    "search",
			Keyword:        "test",
			CrawledNoteIDs: nil,
		}

		manager.AddCrawledNote(cp, "note1")
		assert.NotNil(t, cp.CrawledNoteIDs)
		assert.Equal(t, []string{"note1"}, cp.CrawledNoteIDs)
	})
}

// TestCheckpointManager_IsNoteCrawled 测试检查笔记是否已爬
func TestCheckpointManager_IsNoteCrawled(t *testing.T) {
	manager, _, cleanup := setupTestCheckpointManager(t)
	defer cleanup()

	t.Run("笔记已爬取", func(t *testing.T) {
		cp := &Checkpoint{
			CrawlerType:    "search",
			Keyword:        "test",
			CrawledNoteIDs: []string{"note1", "note2", "note3"},
		}

		assert.True(t, manager.IsNoteCrawled(cp, "note1"))
		assert.True(t, manager.IsNoteCrawled(cp, "note2"))
		assert.True(t, manager.IsNoteCrawled(cp, "note3"))
	})

	t.Run("笔记未爬取", func(t *testing.T) {
		cp := &Checkpoint{
			CrawlerType:    "search",
			Keyword:        "test",
			CrawledNoteIDs: []string{"note1", "note2"},
		}

		assert.False(t, manager.IsNoteCrawled(cp, "note3"))
		assert.False(t, manager.IsNoteCrawled(cp, "note4"))
	})

	t.Run("nil checkpoint", func(t *testing.T) {
		assert.False(t, manager.IsNoteCrawled(nil, "note1"))
	})

	t.Run("空的CrawledNoteIDs", func(t *testing.T) {
		cp := &Checkpoint{
			CrawlerType:    "search",
			Keyword:        "test",
			CrawledNoteIDs: []string{},
		}

		assert.False(t, manager.IsNoteCrawled(cp, "note1"))
	})
}

// TestCheckpointManager_ConcurrentAccess 测试并发访问
func TestCheckpointManager_ConcurrentAccess(t *testing.T) {
	manager, _, cleanup := setupTestCheckpointManager(t)
	defer cleanup()

	t.Run("并发保存", func(t *testing.T) {
		const goroutines = 10
		done := make(chan bool, goroutines)

		for i := 0; i < goroutines; i++ {
			go func(id int) {
				cp := &Checkpoint{
					CrawlerType: "search",
					Keyword:     "并发测试",
					CurrentPage: id,
				}
				err := manager.Save(cp)
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < goroutines; i++ {
			<-done
		}

		// 验证至少有一个断点被保存
		loaded, err := manager.Load("search", "并发测试")
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
	})

	t.Run("并发读写", func(t *testing.T) {
		// 先保存一个断点
		cp := &Checkpoint{
			CrawlerType: "search",
			Keyword:     "读写测试",
			CurrentPage: 1,
		}
		err := manager.Save(cp)
		require.NoError(t, err)

		const goroutines = 20
		done := make(chan bool, goroutines)

		// 10个读,10个写
		for i := 0; i < goroutines/2; i++ {
			// 读操作
			go func() {
				_, err := manager.Load("search", "读写测试")
				assert.NoError(t, err)
				done <- true
			}()

			// 写操作
			go func(page int) {
				cp := &Checkpoint{
					CrawlerType: "search",
					Keyword:     "读写测试",
					CurrentPage: page,
				}
				err := manager.Save(cp)
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < goroutines; i++ {
			<-done
		}

		// 验证最终状态
		loaded, err := manager.Load("search", "读写测试")
		assert.NoError(t, err)
		assert.NotNil(t, loaded)
	})
}

// TestCheckpointResumeScenario 测试断点续爬场景
func TestCheckpointResumeScenario(t *testing.T) {
	manager, _, cleanup := setupTestCheckpointManager(t)
	defer cleanup()

	t.Run("完整的断点续爬流程", func(t *testing.T) {
		keyword := "完整流程测试"

		// 1. 第一次爬取 - 爬到第3页
		cp := &Checkpoint{
			CrawlerType:  "search",
			Keyword:      keyword,
			CurrentPage:  0,
			TotalCrawled: 0,
		}

		for page := 1; page <= 3; page++ {
			cp.CurrentPage = page
			cp.TotalCrawled += 20

			// 模拟爬取笔记
			for i := 0; i < 20; i++ {
				noteID := fmt.Sprintf("note_page%d_%d", page, i)
				manager.AddCrawledNote(cp, noteID)
			}

			// 保存断点
			err := manager.Save(cp)
			require.NoError(t, err)
		}

		// 2. 模拟中断,重新加载断点
		loaded, err := manager.Load("search", keyword)
		require.NoError(t, err)
		require.NotNil(t, loaded)

		// 验证断点状态
		assert.Equal(t, 3, loaded.CurrentPage, "应该从第3页恢复")
		assert.Equal(t, 60, loaded.TotalCrawled, "已爬取60条")
		assert.Len(t, loaded.CrawledNoteIDs, 60, "已爬取笔记ID应该有60个")

		// 3. 从断点恢复,继续爬取第4-5页
		for page := 4; page <= 5; page++ {
			loaded.CurrentPage = page
			loaded.TotalCrawled += 20

			// 模拟爬取新笔记
			for i := 0; i < 20; i++ {
				noteID := fmt.Sprintf("note_page%d_%d", page, i)

				// 检查是否已爬取(应该都是新的)
				assert.False(t, manager.IsNoteCrawled(loaded, noteID))

				manager.AddCrawledNote(loaded, noteID)
			}

			err := manager.Save(loaded)
			require.NoError(t, err)
		}

		// 4. 验证最终状态
		final, err := manager.Load("search", keyword)
		require.NoError(t, err)
		require.NotNil(t, final)

		assert.Equal(t, 5, final.CurrentPage, "最终页码应该是5")
		assert.Equal(t, 100, final.TotalCrawled, "总共爬取100条")
		assert.Len(t, final.CrawledNoteIDs, 100, "笔记ID总数应该是100")

		// 验证没有重复
		seen := make(map[string]bool)
		for _, noteID := range final.CrawledNoteIDs {
			assert.False(t, seen[noteID], "不应该有重复的笔记ID")
			seen[noteID] = true
		}
	})
}

// BenchmarkCheckpointManager_Save 性能测试 - 保存
func BenchmarkCheckpointManager_Save(b *testing.B) {
	tempDir := b.TempDir()
	manager := NewCheckpointManager("xhs", tempDir)

	cp := &Checkpoint{
		CrawlerType: "search",
		Keyword:     "benchmark",
		CurrentPage: 1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cp.CurrentPage = i
		manager.Save(cp)
	}
}

// BenchmarkCheckpointManager_Load 性能测试 - 加载
func BenchmarkCheckpointManager_Load(b *testing.B) {
	tempDir := b.TempDir()
	manager := NewCheckpointManager("xhs", tempDir)

	// 先保存一个断点
	cp := &Checkpoint{
		CrawlerType: "search",
		Keyword:     "benchmark",
		CurrentPage: 1,
	}
	manager.Save(cp)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Load("search", "benchmark")
	}
}
