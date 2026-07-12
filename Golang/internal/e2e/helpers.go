// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/e2e/helpers.go
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

package e2e

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type RunOptions struct {
	Args    []string
	Env     []string
	Timeout time.Duration
}

func RepoRoot(t testing.TB) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.mod from cwd")
		}
		dir = parent
	}
}

func BuildCLI(t testing.TB, repoRoot string) string {
	t.Helper()

	binaryPath := filepath.Join(t.TempDir(), "mcpro-go")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}
	return binaryPath
}

func (r *MockPlatformRunner) HostPort(t testing.TB) (string, string) {
	t.Helper()

	parsedURL, err := url.Parse(r.BaseURL)
	if err != nil {
		t.Fatalf("parse runner base URL: %v", err)
	}
	host, port, err := net.SplitHostPort(parsedURL.Host)
	if err != nil {
		t.Fatalf("split runner host port: %v", err)
	}
	return host, port
}

func (r *MockPlatformRunner) WriteConfig(t testing.TB, configYAML string) string {
	t.Helper()

	if err := os.MkdirAll(r.ConfigDir, 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	path := filepath.Join(r.ConfigDir, "config.yml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(configYAML)+"\n"), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	return path
}

func (r *MockPlatformRunner) RunCLI(t testing.TB, binaryPath string, opts RunOptions) []byte {
	t.Helper()

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	runCmd := exec.CommandContext(ctx, binaryPath, opts.Args...)
	runCmd.Dir = r.WorkDir
	runCmd.Env = append(os.Environ(),
		"RELATION_DB_HOST=127.0.0.1",
		"RELATION_DB_PORT=1",
		"MCPRO_DATA_DIR="+r.DataDir,
		"MCPRO_TEST_BLOCK_EXTERNAL_HTTP=1",
	)
	runCmd.Env = append(runCmd.Env, opts.Env...)

	output, err := runCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("crawler command timed out\n%s", output)
	}
	if err != nil {
		t.Fatalf("crawler command failed: %v\n%s", err, output)
	}
	return output
}

func ReadJSONRows(t testing.TB, path string, output []byte) []map[string]any {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read json rows from %s: %v\ncommand output:\n%s", path, err, output)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("unmarshal json rows from %s: %v\n%s", path, err, body)
	}
	return rows
}

func ReadCSVRecords(t testing.TB, path string, output []byte) [][]string {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open csv records from %s: %v\ncommand output:\n%s", path, err, output)
	}
	defer file.Close()
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("read csv records from %s: %v\ncommand output:\n%s", path, err, output)
	}
	return records
}

func CSVCellAt(records [][]string, row int, column string) string {
	for idx, header := range records[0] {
		if strings.TrimPrefix(header, "\ufeff") == column {
			return records[row][idx]
		}
	}
	return ""
}

func OpenSQLite(t testing.TB, dbPath string, output []byte) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db %s: %v\ncommand output:\n%s", dbPath, err, output)
	}
	return db
}

func ReadSQLiteRows(t testing.TB, dbPath, tableName, orderBy string, output []byte) []map[string]any {
	t.Helper()

	db := OpenSQLite(t, dbPath, output)
	query := fmt.Sprintf("SELECT * FROM %s", quoteSQLiteIdentifier(t, tableName))
	if orderBy != "" {
		query += " ORDER BY " + quoteSQLiteIdentifier(t, orderBy)
	}

	var rows []map[string]any
	if err := db.Raw(query).Scan(&rows).Error; err != nil {
		t.Fatalf("query sqlite rows from %s.%s: %v\ncommand output:\n%s", dbPath, tableName, err, output)
	}
	return rows
}

func SQLiteCell(rows []map[string]any, row int, column string) string {
	if row < 0 || row >= len(rows) {
		return ""
	}
	value, ok := rows[row][column]
	if !ok || value == nil {
		return ""
	}
	for {
		reflected := reflect.ValueOf(value)
		if reflected.Kind() != reflect.Ptr {
			break
		}
		if reflected.IsNil() {
			return ""
		}
		value = reflected.Elem().Interface()
	}
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}

var sqliteIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func quoteSQLiteIdentifier(t testing.TB, identifier string) string {
	t.Helper()
	if !sqliteIdentifierPattern.MatchString(identifier) {
		t.Fatalf("unsafe sqlite identifier: %q", identifier)
	}
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
