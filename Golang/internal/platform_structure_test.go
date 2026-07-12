// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/platform_structure_test.go
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

package internal_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestPlatformPackagesUseConsistentFileLayout(t *testing.T) {
	platformEntrypoints := map[string]string{
		"bilibili": "bilibili.go",
		"douyin":   "douyin.go",
		"kuaishou": "kuaishou.go",
		"tieba":    "tieba.go",
		"weibo":    "weibo.go",
		"xhs":      "xhs.go",
		"zhihu":    "zhihu.go",
	}

	commonFiles := []string{
		"handler.go",
		"model.go",
		"store.go",
		"store_file.go",
		"store_sqlite.go",
	}

	for platform, entrypoint := range platformEntrypoints {
		platform := platform
		entrypoint := entrypoint
		t.Run(platform, func(t *testing.T) {
			for _, filename := range append(commonFiles, entrypoint) {
				path := filepath.Join(platform, filename)
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("expected %s platform package file %s: %v", platform, path, err)
				}
			}
		})
	}
}

func TestPlatformFileStoreLivesInDedicatedFile(t *testing.T) {
	platforms := []string{"bilibili", "douyin", "kuaishou", "tieba", "weibo", "xhs", "zhihu"}

	for _, platform := range platforms {
		platform := platform
		t.Run(platform, func(t *testing.T) {
			storeFileBody, err := os.ReadFile(filepath.Join(platform, "store_file.go"))
			if err != nil {
				t.Fatalf("read store_file.go: %v", err)
			}
			if !strings.Contains(string(storeFileBody), "type FileStore struct") {
				t.Fatalf("%s/store_file.go should define FileStore", platform)
			}

			storeBody, err := os.ReadFile(filepath.Join(platform, "store.go"))
			if err != nil {
				t.Fatalf("read store.go: %v", err)
			}
			if strings.Contains(string(storeBody), "type FileStore struct") {
				t.Fatalf("%s/store.go should keep Store factory/DB store only; move FileStore to store_file.go", platform)
			}
		})
	}
}

func TestPlatformStoreFactoriesUseSharedSaveDataConstants(t *testing.T) {
	platforms := []string{"bilibili", "douyin", "kuaishou", "tieba", "weibo", "xhs", "zhihu"}

	for _, platform := range platforms {
		platform := platform
		t.Run(platform, func(t *testing.T) {
			bodyBytes, err := os.ReadFile(filepath.Join(platform, "store.go"))
			if err != nil {
				t.Fatalf("read store.go: %v", err)
			}
			body := string(bodyBytes)
			for _, rawLiteral := range []string{
				`case "", "db":`,
				`case "sqlite":`,
				`case "json":`,
				`case "csv":`,
				`NewFileStore(cfg.DataDir, cfg.CrawlerType, "json")`,
				`NewFileStore(cfg.DataDir, cfg.CrawlerType, "csv")`,
			} {
				if strings.Contains(body, rawLiteral) {
					t.Fatalf("%s/store.go should use constant.SaveDataTo* instead of raw literal %s", platform, rawLiteral)
				}
			}
		})
	}
}

func TestPlatformCrawlerTypeSupportMatrixIsDocumented(t *testing.T) {
	type crawlerType struct {
		constName string
		label     string
	}

	crawlerTypes := []crawlerType{
		{constName: "SearchCrawlerType", label: "search"},
		{constName: "DetailCrawlerType", label: "detail"},
		{constName: "CreatorCrawlerType", label: "creator"},
		{constName: "HomefeedCrawlerType", label: "homefeed"},
		{constName: "HotSearchCrawlerType", label: "hot_search"},
	}
	platformEntrypoints := map[string]string{
		"bilibili": "bilibili.go",
		"douyin":   "douyin.go",
		"kuaishou": "kuaishou.go",
		"tieba":    "tieba.go",
		"weibo":    "weibo.go",
		"xhs":      "xhs.go",
		"zhihu":    "zhihu.go",
	}

	docBytes, err := os.ReadFile(filepath.Join("..", "docs", "internal_architecture_review.md"))
	if err != nil {
		t.Fatalf("read internal architecture review doc: %v", err)
	}
	doc := string(docBytes)

	for platform, entrypoint := range platformEntrypoints {
		bodyBytes, err := os.ReadFile(filepath.Join(platform, entrypoint))
		if err != nil {
			t.Fatalf("read %s entrypoint: %v", platform, err)
		}
		body := string(bodyBytes)

		cells := []string{platform}
		for _, crawlerType := range crawlerTypes {
			if strings.Contains(body, "handlers[constant."+crawlerType.constName+"]") {
				cells = append(cells, "Y")
			} else {
				cells = append(cells, "-")
			}
		}

		expectedRow := "| " + strings.Join(cells, " | ") + " |"
		if !strings.Contains(doc, expectedRow) {
			t.Fatalf("docs/internal_architecture_review.md should document crawler type support row: %s", expectedRow)
		}
	}
}

func TestPlatformCheckpointManagersUseExplicitCompatibilityConstants(t *testing.T) {
	expectedCalls := map[string]string{
		"bilibili": `crawler.NewCheckpointManager(constant.BilibiliCheckpointPlatform, "data/checkpoints", constant.BilibiliLegacyCheckpointPlatform)`,
		"douyin":   `crawler.NewCheckpointManager(constant.DouyinCheckpointPlatform, "data/checkpoints")`,
		"kuaishou": `crawler.NewCheckpointManager(constant.KuaishouCheckpointPlatform, "data/checkpoints")`,
		"tieba":    `crawler.NewCheckpointManager(constant.TiebaCheckpointPlatform, "data/checkpoints")`,
		"weibo":    `crawler.NewCheckpointManager(constant.WeiboCheckpointPlatform, "data/checkpoints")`,
		"xhs":      `crawler.NewCheckpointManager(constant.XhsCheckpointPlatform, "data/checkpoints")`,
		"zhihu":    `crawler.NewCheckpointManager(constant.ZhihuCheckpointPlatform, "data/checkpoints")`,
	}
	entrypoints := map[string]string{
		"bilibili": "bilibili.go",
		"douyin":   "douyin.go",
		"kuaishou": "kuaishou.go",
		"tieba":    "tieba.go",
		"weibo":    "weibo.go",
		"xhs":      "xhs.go",
		"zhihu":    "zhihu.go",
	}

	for platform, expectedCall := range expectedCalls {
		bodyBytes, err := os.ReadFile(filepath.Join(platform, entrypoints[platform]))
		if err != nil {
			t.Fatalf("read %s entrypoint: %v", platform, err)
		}
		body := string(bodyBytes)
		if !strings.Contains(body, expectedCall) {
			t.Fatalf("%s should construct checkpoint manager with explicit compatibility constants: %s", platform, expectedCall)
		}
	}
}

func TestPlatformProcessorArchitectureMatrixIsDocumented(t *testing.T) {
	platforms := []string{"bilibili", "douyin", "kuaishou", "tieba", "weibo", "xhs", "zhihu"}
	docBytes, err := os.ReadFile(filepath.Join("..", "docs", "internal_architecture_review.md"))
	if err != nil {
		t.Fatalf("read internal architecture review doc: %v", err)
	}
	doc := string(docBytes)

	for _, platform := range platforms {
		_, processorErr := os.Stat(filepath.Join(platform, "processor.go"))
		hasProcessor := processorErr == nil
		if processorErr != nil && !os.IsNotExist(processorErr) {
			t.Fatalf("stat %s processor.go: %v", platform, processorErr)
		}

		entrypoint := platform + ".go"
		bodyBytes, err := os.ReadFile(filepath.Join(platform, entrypoint))
		if err != nil {
			t.Fatalf("read %s entrypoint: %v", platform, err)
		}
		hasWorkerPool := strings.Contains(string(bodyBytes), "worker.NewPool(")

		cells := []string{platform, boolCell(hasProcessor), boolCell(hasWorkerPool)}
		expectedRow := "| " + strings.Join(cells, " | ") + " |"
		if !strings.Contains(doc, expectedRow) {
			t.Fatalf("docs/internal_architecture_review.md should document processor architecture row: %s", expectedRow)
		}
	}
}

func TestProcessorArchitectureDecisionIsDocumented(t *testing.T) {
	docBytes, err := os.ReadFile(filepath.Join("..", "docs", "internal_architecture_review.md"))
	if err != nil {
		t.Fatalf("read internal architecture review doc: %v", err)
	}
	doc := string(docBytes)
	for _, required := range []string{
		"Decision: keep `processor.go` as an optional platform adapter for now.",
		"Do not require every platform package to add `processor.go` until a shared processor contract is extracted.",
	} {
		if !strings.Contains(doc, required) {
			t.Fatalf("docs/internal_architecture_review.md should document processor architecture decision: %s", required)
		}
	}
}

func TestPlatformStorageTableMatrixIsDocumented(t *testing.T) {
	platforms := []string{"bilibili", "douyin", "kuaishou", "tieba", "weibo", "xhs", "zhihu"}
	docBytes, err := os.ReadFile(filepath.Join("..", "docs", "internal_architecture_review.md"))
	if err != nil {
		t.Fatalf("read internal architecture review doc: %v", err)
	}
	doc := string(docBytes)
	tableNamePattern := regexp.MustCompile(`(?s)func \([^)]*\) TableName\(\) string \{\s*return "([^"]+)"\s*\}`)

	for _, platform := range platforms {
		bodyBytes, err := os.ReadFile(filepath.Join(platform, "model.go"))
		if err != nil {
			t.Fatalf("read %s model.go: %v", platform, err)
		}
		matches := tableNamePattern.FindAllStringSubmatch(string(bodyBytes), -1)
		if len(matches) == 0 {
			t.Fatalf("%s/model.go should define TableName methods", platform)
		}

		tableNames := make([]string, 0, len(matches))
		for _, match := range matches {
			tableNames = append(tableNames, match[1])
		}
		expectedRow := "| " + platform + " | `" + strings.Join(tableNames, "`, `") + "` |"
		if !strings.Contains(doc, expectedRow) {
			t.Fatalf("docs/internal_architecture_review.md should document storage table row: %s", expectedRow)
		}
	}
}

func TestPlatformPersistenceModelFieldParityIsDocumented(t *testing.T) {
	type modelPair struct {
		platform      string
		pythonFile    string
		pythonClass   string
		goPackage     string
		goStruct      string
		ignoredPython []string
	}

	pairs := []modelPair{
		{platform: "xhs", pythonFile: "m_xhs.py", pythonClass: "XhsNote", goPackage: "xhs", goStruct: "Note"},
		{platform: "xhs", pythonFile: "m_xhs.py", pythonClass: "XhsComment", goPackage: "xhs", goStruct: "NoteComment"},
		{platform: "xhs", pythonFile: "m_xhs.py", pythonClass: "XhsCreator", goPackage: "xhs", goStruct: "Creator"},
		{platform: "bilibili", pythonFile: "m_bilibili.py", pythonClass: "BilibiliVideo", goPackage: "bilibili", goStruct: "Video"},
		{platform: "bilibili", pythonFile: "m_bilibili.py", pythonClass: "BilibiliComment", goPackage: "bilibili", goStruct: "VideoComment"},
		{platform: "bilibili", pythonFile: "m_bilibili.py", pythonClass: "BilibiliUpInfo", goPackage: "bilibili", goStruct: "UpInfo"},
		{platform: "douyin", pythonFile: "m_douyin.py", pythonClass: "DouyinAweme", goPackage: "douyin", goStruct: "Aweme"},
		{platform: "douyin", pythonFile: "m_douyin.py", pythonClass: "DouyinAwemeComment", goPackage: "douyin", goStruct: "AwemeComment"},
		{platform: "douyin", pythonFile: "m_douyin.py", pythonClass: "DouyinCreator", goPackage: "douyin", goStruct: "Creator"},
		{platform: "kuaishou", pythonFile: "m_kuaishou.py", pythonClass: "KuaishouVideo", goPackage: "kuaishou", goStruct: "Video"},
		{platform: "kuaishou", pythonFile: "m_kuaishou.py", pythonClass: "KuaishouVideoComment", goPackage: "kuaishou", goStruct: "VideoComment"},
		{platform: "kuaishou", pythonFile: "m_kuaishou.py", pythonClass: "KuaishouCreator", goPackage: "kuaishou", goStruct: "Creator"},
		{platform: "kuaishou", pythonFile: "m_hot_search.py", pythonClass: "HotSearchItem", goPackage: "kuaishou", goStruct: "HotSearchItem"},
		{platform: "kuaishou", pythonFile: "m_hot_search.py", pythonClass: "HotSearchContentRelation", goPackage: "kuaishou", goStruct: "HotSearchContentRelation"},
		{platform: "weibo", pythonFile: "m_weibo.py", pythonClass: "WeiboNote", goPackage: "weibo", goStruct: "Note"},
		{platform: "weibo", pythonFile: "m_weibo.py", pythonClass: "WeiboComment", goPackage: "weibo", goStruct: "Comment", ignoredPython: []string{"sub_comments"}},
		{platform: "weibo", pythonFile: "m_weibo.py", pythonClass: "WeiboCreator", goPackage: "weibo", goStruct: "Creator"},
		{platform: "weibo", pythonFile: "m_hot_search.py", pythonClass: "HotSearchItem", goPackage: "weibo", goStruct: "HotSearchItem"},
		{platform: "weibo", pythonFile: "m_hot_search.py", pythonClass: "HotSearchContentRelation", goPackage: "weibo", goStruct: "HotSearchContentRelation"},
		{platform: "tieba", pythonFile: "m_baidu_tieba.py", pythonClass: "TiebaNote", goPackage: "tieba", goStruct: "Note"},
		{platform: "tieba", pythonFile: "m_baidu_tieba.py", pythonClass: "TiebaComment", goPackage: "tieba", goStruct: "Comment"},
		{platform: "tieba", pythonFile: "m_baidu_tieba.py", pythonClass: "TiebaCreator", goPackage: "tieba", goStruct: "Creator"},
		{platform: "zhihu", pythonFile: "m_zhihu.py", pythonClass: "ZhihuContent", goPackage: "zhihu", goStruct: "Content"},
		{platform: "zhihu", pythonFile: "m_zhihu.py", pythonClass: "ZhihuComment", goPackage: "zhihu", goStruct: "Comment"},
		{platform: "zhihu", pythonFile: "m_zhihu.py", pythonClass: "ZhihuCreator", goPackage: "zhihu", goStruct: "Creator"},
		{platform: "zhihu", pythonFile: "m_hot_search.py", pythonClass: "HotSearchItem", goPackage: "zhihu", goStruct: "HotSearchItem"},
		{platform: "zhihu", pythonFile: "m_hot_search.py", pythonClass: "HotSearchContentRelation", goPackage: "zhihu", goStruct: "HotSearchContentRelation"},
	}

	docBytes, err := os.ReadFile(filepath.Join("..", "docs", "internal_architecture_review.md"))
	if err != nil {
		t.Fatalf("read internal architecture review doc: %v", err)
	}
	doc := string(docBytes)

	missingRows := make([]string, 0)
	for _, pair := range pairs {
		pythonFields, err := pythonModelFields(filepath.Join("..", "..", "MediaCrawlerPro-Python", "model", pair.pythonFile), pair.pythonClass)
		if err != nil {
			t.Fatalf("parse Python fields for %s.%s: %v", pair.pythonFile, pair.pythonClass, err)
		}
		for _, ignored := range pair.ignoredPython {
			delete(pythonFields, ignored)
		}
		goFields, err := goModelColumns(filepath.Join(pair.goPackage, "model.go"), pair.goStruct)
		if err != nil {
			t.Fatalf("parse Go columns for %s.%s: %v", pair.goPackage, pair.goStruct, err)
		}
		delete(goFields, "id")

		pythonOnly := sortedSetDifference(pythonFields, goFields)
		goOnly := sortedSetDifference(goFields, pythonFields)
		expectedRow := "| " + pair.platform + " | " + pair.pythonClass + " -> " + pair.goStruct + " | " + markdownFieldList(pythonOnly) + " | " + markdownFieldList(goOnly) + " |"
		if !strings.Contains(doc, expectedRow) {
			missingRows = append(missingRows, expectedRow)
		}
	}
	if len(missingRows) > 0 {
		t.Fatalf("docs/internal_architecture_review.md should document field parity rows:\n%s", strings.Join(missingRows, "\n"))
	}
}

func boolCell(value bool) string {
	if value {
		return "Y"
	}
	return "-"
}

func pythonModelFields(path, className string) (map[string]struct{}, error) {
	bodyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fieldPattern := regexp.MustCompile(`^\s+([A-Za-z_][A-Za-z0-9_]*): .*Field\(`)
	lines := strings.Split(string(bodyBytes), "\n")
	inClass := false
	fields := make(map[string]struct{})
	for _, line := range lines {
		if strings.HasPrefix(line, "class ") {
			inClass = strings.HasPrefix(line, "class "+className+"(")
			continue
		}
		if !inClass {
			continue
		}
		matches := fieldPattern.FindStringSubmatch(line)
		if len(matches) == 2 {
			fields[matches[1]] = struct{}{}
		}
	}
	if len(fields) == 0 {
		return nil, os.ErrNotExist
	}
	return fields, nil
}

func goModelColumns(path, structName string) (map[string]struct{}, error) {
	bodyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	columnPattern := regexp.MustCompile("`gorm:\"[^\"]*column:([^;\"`]+)")
	lines := strings.Split(string(bodyBytes), "\n")
	inStruct := false
	fields := make(map[string]struct{})
	for _, line := range lines {
		if strings.HasPrefix(line, "type "+structName+" struct {") {
			inStruct = true
			continue
		}
		if !inStruct {
			continue
		}
		if strings.TrimSpace(line) == "}" {
			break
		}
		matches := columnPattern.FindStringSubmatch(line)
		if len(matches) == 2 {
			fields[matches[1]] = struct{}{}
		}
	}
	if len(fields) == 0 {
		return nil, os.ErrNotExist
	}
	return fields, nil
}

func sortedSetDifference(left, right map[string]struct{}) []string {
	values := make([]string, 0)
	for value := range left {
		if _, ok := right[value]; !ok {
			values = append(values, value)
		}
	}
	sort.Strings(values)
	return values
}

func markdownFieldList(fields []string) string {
	if len(fields) == 0 {
		return "-"
	}
	quoted := make([]string, 0, len(fields))
	for _, field := range fields {
		quoted = append(quoted, "`"+field+"`")
	}
	return strings.Join(quoted, ", ")
}
