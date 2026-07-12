// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/xhs/store_file.go
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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FileStore struct {
	rootDir     string
	crawlerType string
	format      string
	fileIndex   int
	mu          sync.Mutex
}

func NewFileStore(rootDir, crawlerType, format string) *FileStore {
	if rootDir == "" {
		rootDir = "data"
	}
	if crawlerType == "" {
		crawlerType = "search"
	}
	csvDir := filepath.Join(rootDir, "xhs")
	return &FileStore{
		rootDir:     rootDir,
		crawlerType: crawlerType,
		format:      strings.ToLower(format),
		fileIndex:   calculateFileIndex(csvDir),
	}
}

func (s *FileStore) SaveNote(ctx context.Context, note *Note) error {
	return s.save(ctx, note, "contents")
}

func (s *FileStore) NoteExists(ctx context.Context, noteID string) (bool, error) {
	return false, nil
}

func (s *FileStore) SaveComment(ctx context.Context, comment *NoteComment) error {
	return s.save(ctx, comment, "comments")
}

func (s *FileStore) SaveCreator(ctx context.Context, creator *Creator) error {
	return s.save(ctx, creator, "creator")
}

func (s *FileStore) save(ctx context.Context, item any, storeType string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	headers, values, row, err := modelRecord(item)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.format {
	case "json":
		return s.saveJSON(row, storeType)
	case "csv":
		return s.saveCSV(headers, values, storeType)
	default:
		return fmt.Errorf("unsupported file store format: %s", s.format)
	}
}

func (s *FileStore) saveJSON(row map[string]any, storeType string) error {
	path := s.jsonPath(storeType)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	rows := make([]map[string]any, 0)
	if body, err := os.ReadFile(path); err == nil && len(body) > 0 {
		if err := json.Unmarshal(body, &rows); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	rows = append(rows, row)
	body, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0644)
}

func (s *FileStore) saveCSV(headers, values []string, storeType string) error {
	path := s.csvPath(storeType)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	needsHeader := true
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		needsHeader = false
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if needsHeader {
		if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
			return err
		}
	}

	writer := csv.NewWriter(file)
	if needsHeader {
		if err := writer.Write(headers); err != nil {
			return err
		}
	}
	if err := writer.Write(values); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func (s *FileStore) jsonPath(storeType string) string {
	return filepath.Join(s.rootDir, "xhs", "json", fmt.Sprintf("%s_%s_%s.json", s.crawlerType, storeType, currentDate()))
}

func (s *FileStore) csvPath(storeType string) string {
	return filepath.Join(s.rootDir, "xhs", fmt.Sprintf("%d_%s_%s_%s.csv", s.fileIndex, s.crawlerType, storeType, currentDate()))
}

func currentDate() string {
	return time.Now().Format("2006-01-02")
}

func calculateFileIndex(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 1
	}
	maxIndex := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		prefix := strings.SplitN(name, "_", 2)[0]
		index, err := strconv.Atoi(prefix)
		if err == nil && index > maxIndex {
			maxIndex = index
		}
	}
	return maxIndex + 1
}

func modelRecord(item any) ([]string, []string, map[string]any, error) {
	value := reflect.Indirect(reflect.ValueOf(item))
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return nil, nil, nil, errors.New("file store item must be a struct or struct pointer")
	}

	itemType := value.Type()
	headers := make([]string, 0, value.NumField())
	values := make([]string, 0, value.NumField())
	row := make(map[string]any, value.NumField())

	for i := 0; i < value.NumField(); i++ {
		field := itemType.Field(i)
		column := gormColumnName(field)
		if column == "" || column == "id" || column == "add_ts" {
			continue
		}
		fieldValue := value.Field(i)
		stringValue := fmt.Sprint(fieldValue.Interface())
		headers = append(headers, column)
		values = append(values, stringValue)
		if column == "time" || column == "last_update_time" {
			row[column] = stringValue
			continue
		}
		row[column] = fieldValue.Interface()
	}

	return headers, values, row, nil
}

func gormColumnName(field reflect.StructField) string {
	tag := field.Tag.Get("gorm")
	for _, part := range strings.Split(tag, ";") {
		if strings.HasPrefix(part, "column:") {
			return strings.TrimPrefix(part, "column:")
		}
	}
	return ""
}
