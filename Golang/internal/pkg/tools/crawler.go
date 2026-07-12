// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/tools/crawler.go
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
	"crypto/rand"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"
)

// EncodeQueryParams encodes a map of query parameters into a URL-encoded string.
func EncodeQueryParams(params map[string]any) string {
	values := url.Values{}
	for key, value := range params {
		values.Add(key, fmt.Sprintf("%v", value))
	}
	return values.Encode()
}

// EncodeQueryParamsStr encodes a map[string]string of query parameters into a URL-encoded string.
func EncodeQueryParamsStr(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	return values.Encode()
}

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Base36encode converts a big integer to a base36 string
func Base36encode(number *big.Int) string {
	if number.Cmp(big.NewInt(0)) == 0 {
		return "0"
	}

	var base36 strings.Builder
	zero := big.NewInt(0)
	base := big.NewInt(36)
	num := new(big.Int).Set(number)

	// Convert to base36
	for num.Cmp(zero) > 0 {
		mod := new(big.Int)
		num.DivMod(num, base, mod)
		base36.WriteByte(alphabet[mod.Int64()])
	}

	// Reverse the string
	bytes := []byte(base36.String())
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}

	return string(bytes)
}

// GetSearchID generates a unique search ID
func GetSearchID() string {
	// Get current timestamp in nanoseconds for more precision
	timestamp := time.Now().UnixNano()

	// Create big.Int for timestamp and shift left 64 bits
	e := big.NewInt(timestamp)
	e.Lsh(e, 64)

	// Generate cryptographically secure random number between 0 and 2147483646
	_max := big.NewInt(2147483647)
	t, _ := rand.Int(rand.Reader, _max)

	// Add the random number to the shifted timestamp
	e.Add(e, t)

	return Base36encode(e)
}
