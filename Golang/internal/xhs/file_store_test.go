// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/xhs/file_store_test.go
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
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewStoreCreatesJSONStoreWithoutDB(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.Config{
		SaveDataOption: "json",
		CrawlerType:    "search",
		DataDir:        dataDir,
	}
	store, err := NewStore(cfg, nil)
	require.NoError(t, err)

	err = store.SaveNote(context.Background(), NewTestDataBuilder().BuildNote("note_json", func(note *Note) {
		note.Title = "JSON标题"
	}))

	require.NoError(t, err)
	jsonPath := filepath.Join(dataDir, "xhs", "json", "search_contents_"+time.Now().Format("2006-01-02")+".json")
	body, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var rows []map[string]any
	require.NoError(t, json.Unmarshal(body, &rows))
	require.Len(t, rows, 1)
	require.Equal(t, "note_json", rows[0]["note_id"])
	require.Equal(t, "JSON标题", rows[0]["title"])
}

func TestNewStoreCreatesCSVStoreWithoutDB(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.Config{
		SaveDataOption: "csv",
		CrawlerType:    "search",
		DataDir:        dataDir,
	}
	store, err := NewStore(cfg, nil)
	require.NoError(t, err)

	err = store.SaveNote(context.Background(), NewTestDataBuilder().BuildNote("note_csv", func(note *Note) {
		note.Title = "CSV标题"
	}))

	require.NoError(t, err)
	csvPath := filepath.Join(dataDir, "xhs", "1_search_contents_"+time.Now().Format("2006-01-02")+".csv")
	file, err := os.Open(csvPath)
	require.NoError(t, err)
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Contains(t, records[0], "note_id")
	require.Contains(t, records[0], "title")
	require.Contains(t, records[1], "note_csv")
	require.Contains(t, records[1], "CSV标题")
}

func TestNewCrawlerUsesConfiguredStoreFactory(t *testing.T) {
	cfg := &config.Config{
		SaveDataOption:    "json",
		CrawlerType:       "search",
		DataDir:           t.TempDir(),
		MaxConcurrencyNum: 1,
	}

	crawler := NewCrawler(CrawlerOptions{Config: cfg})

	require.IsType(t, &FileStore{}, crawler.store)
}

func TestNewStoreCreatesSQLiteStoreWithoutMySQL(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "media_crawler.db")
	cfg := &config.Config{
		SaveDataOption: "sqlite",
		SqliteDBPath:   dbPath,
		CrawlerType:    "search",
	}

	store, err := NewStore(cfg, nil)
	require.NoError(t, err)
	require.IsType(t, &SQLiteDBStore{}, store)

	ctx := context.Background()
	now := time.Now().Unix()
	note := &Note{
		UserID:         "user_sqlite",
		NoteID:         "note_sqlite",
		Title:          "sqlite title",
		AddTs:          now,
		LastModifyTs:   now,
		Time:           now,
		LastUpdateTime: now,
	}
	require.NoError(t, store.SaveNote(ctx, note))

	note.Title = "sqlite updated title"
	require.NoError(t, store.SaveNote(ctx, note))

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)
	var rows []Note
	require.NoError(t, db.Where("note_id = ?", "note_sqlite").Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "sqlite updated title", rows[0].Title)
}
