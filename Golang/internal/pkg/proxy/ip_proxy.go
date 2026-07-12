// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/proxy/ip_proxy.go
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
	"errors"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models"
	"github.com/google/wire"
	"github.com/samber/lo"
	"math/rand"
	"time"
)

var ProviderSet = wire.NewSet(NewIpPool, config.GetConfig)

type IpPool struct {
	IpPoolCount      int                   // IP代理池数量
	EnableValidateIp bool                  // 是否开启IP验证
	IpProvider       Provider              // Ip代理Provider接口
	ProxyList        []*models.IpInfoModel // IP代理列表
}

func NewIpPool(conf *config.Config, ipProvider Provider) (*IpPool, error) {
	ipPool := &IpPool{
		IpPoolCount:      conf.ProxyConfig.IpProxyPoolCount,
		EnableValidateIp: conf.ProxyConfig.EnableValidateIpProxy,
		IpProvider:       ipProvider,
	}
	err := ipPool.LoadProxies()
	if err != nil {
		return nil, err
	}
	return ipPool, nil
}

// LoadProxies load ip proxies from special provider brand name
func (i *IpPool) LoadProxies() error {
	ipList, err := i.IpProvider.GetProxies()
	if err != nil {
		// todo record log
		return err
	}
	i.ProxyList = ipList
	return nil
}

// validateProxy send an validate ip proxy request
func (i *IpPool) validateProxy(proxy *models.IpInfoModel) bool {

	return false
}

// GetProxy random get proxy from ip proxy pool
func (i *IpPool) GetProxy() (*models.IpInfoModel, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomIndex := r.Intn(len(i.ProxyList))
	proxy := i.ProxyList[randomIndex]

	// ip取出来之后移除掉
	i.ProxyList = lo.DropByIndex(i.ProxyList, randomIndex)
	if i.EnableValidateIp {
		if !i.validateProxy(proxy) {
			return nil, errors.New("ip not available")
		}
	}
	return proxy, nil
}

// reloadProxies reload proxies
func (i *IpPool) reloadProxies() error {
	i.ProxyList = make([]*models.IpInfoModel, 0)
	return i.LoadProxies()
}

// MarkIpInvalid mark ip validates
func (i *IpPool) MarkIpInvalid(proxy *models.IpInfoModel) {
	i.IpProvider.MarkIpValidate(proxy)
	for index, p := range i.ProxyList {
		if p.Ip == proxy.Ip && p.User == proxy.User && p.Port == proxy.Port {
			i.ProxyList = lo.DropByIndex(i.ProxyList, index)
			break
		}
	}
}
