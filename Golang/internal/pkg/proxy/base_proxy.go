// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/proxy/base_proxy.go
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

package proxy

import (
	"encoding/json"
	"fmt"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/cache"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
)

type Provider interface {
	// GetProxies get proxies
	GetProxies() ([]*models.IpInfoModel, error)

	// MarkIpValidate mark ip validate
	MarkIpValidate(ip *models.IpInfoModel)
}

type IpCache struct {
	CacheClient cache.ICache
}

// SetIp set one ip to cache
func (i *IpCache) SetIp(ipKey string, ipValue string, ex int) error {
	return i.CacheClient.Set(ipKey, []byte(ipValue), ex)
}

// DeleteIp delete some ip
func (i *IpCache) DeleteIp(ipKey string) error {
	return i.CacheClient.Delete(ipKey)
}

// LoadAllIp load all ip from cache
func (i *IpCache) LoadAllIp(ProxyProviderBrandName string) ([]*models.IpInfoModel, error) {
	allIpList := make([]*models.IpInfoModel, 0)
	allIpKeys, err := i.CacheClient.Keys(fmt.Sprintf("%s_*", ProxyProviderBrandName))
	if err != nil {
		return allIpList, err
	}

	for _, ipKey := range allIpKeys {
		ipValue, err := i.CacheClient.Get(ipKey)
		if err != nil {
			// todo error log record
			return allIpList, err
		}

		ipValueInfo := models.IpInfoModel{}
		err = json.Unmarshal(ipValue, &ipValueInfo)
		if err != nil {
			// todo error log record
			return allIpList, err
		}

		ttl := i.CacheClient.Ttl(ipKey)
		if ttl <= 0 {
			continue
		}
		ipValueInfo.ExpiredTimeTs = tools.GetUnixTimeTs() + int64(ttl)
		allIpList = append(allIpList, &ipValueInfo)
	}
	return allIpList, nil
}
