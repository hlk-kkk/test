// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/tools/convert_test.go
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
	"testing"

	"github.com/stretchr/testify/assert"
)

// 测试 RecursiveConvertKeysToSnakeCase 函数
func TestRecursiveConvertKeysToSnakeCase(t *testing.T) {
	// 测试用例：单层 map
	input := map[string]interface{}{
		"FirstName": "John",
		"LastName":  "Doe",
	}
	expected := map[string]interface{}{
		"first_name": "John",
		"last_name":  "Doe",
	}
	result := RecursiveConvertKeysToSnakeCase(input)
	assert.Equal(t, expected, result, "单层 map 转换失败")

	// 测试用例：嵌套结构
	nestedInput := map[string]interface{}{
		"UserInfo": map[string]interface{}{
			"UserName": "johndoe",
			"UserAge":  30,
		},
	}
	nestedExpected := map[string]interface{}{
		"user_info": map[string]interface{}{
			"user_name": "johndoe",
			"user_age":  30,
		},
	}
	nestedResult := RecursiveConvertKeysToSnakeCase(nestedInput)
	assert.Equal(t, nestedExpected, nestedResult, "嵌套结构 map 转换失败")
}
