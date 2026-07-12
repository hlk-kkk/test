// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/tools/convert.go
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

package tools

import (
	"github.com/iancoleman/strcase"
)

// RecursiveConvertKeysToSnakeCase 递归地将所有 key 转换成下划线格式
func RecursiveConvertKeysToSnakeCase(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		newMap := make(map[string]interface{})
		for key, value := range v {
			// 使用 strcase 将 key 转换为 snake_case
			newKey := strcase.ToSnake(key)
			newMap[newKey] = RecursiveConvertKeysToSnakeCase(value)
		}
		return newMap
	case []interface{}:
		// 递归处理 slice 中的每一个元素
		for i, item := range v {
			v[i] = RecursiveConvertKeysToSnakeCase(item)
		}
		return v
	default:
		return v
	}
}
