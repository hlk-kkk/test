// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/tools/crawler_test.go
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
)

func TestEncodeQueryParamsEmptyParams(t *testing.T) {
	params := map[string]any{}
	expected := ""
	result := EncodeQueryParams(params)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestEncodeQueryParamsSingleParam(t *testing.T) {
	params := map[string]any{"key": "value"}
	expected := "key=value"
	result := EncodeQueryParams(params)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestEncodeQueryParamsMultipleParams(t *testing.T) {
	params := map[string]any{"key1": "value1", "key2": "value2"}
	expected := "key1=value1&key2=value2"
	result := EncodeQueryParams(params)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestEncodeQueryParamsSpecialCharacters(t *testing.T) {
	params := map[string]any{"key": "value with spaces", "key2": "value/with/slash"}
	expected := "key=value+with+spaces&key2=value%2Fwith%2Fslash"
	result := EncodeQueryParams(params)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestEncodeQueryParamsNumericValues(t *testing.T) {
	params := map[string]any{"key": 123, "key2": 45.67}
	expected := "key=123&key2=45.67"
	result := EncodeQueryParams(params)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGetSearchID(t *testing.T) {
	searchId := GetSearchID()
	t.Log(searchId)
}
