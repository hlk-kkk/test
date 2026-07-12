// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/pkg/account_pool/account_pool.go
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

package account_pool

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/constant"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/models"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/httpclient"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/proxy"
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/repo/accounts_cookies"
)

type AccountPool struct {
	platformName       string
	accountSaveType    string
	cookieBridgeURL    string
	accountList        []*models.AccountInfoModel
	accountCookiesRepo *accounts_cookies.CookiesAccountRepo
}

// NewAccountPool create account pool
func NewAccountPool(config *config.Config, accountCookiesRepo *accounts_cookies.CookiesAccountRepo) *AccountPool {
	config.ApplyDefaults()
	accountPool := &AccountPool{
		platformName:       config.Platform,
		accountSaveType:    config.AccountPoolSaveType,
		cookieBridgeURL:    config.CookieBridgeURL,
		accountCookiesRepo: accountCookiesRepo,
	}
	accountPool.initialize()
	return accountPool
}

func (a *AccountPool) initialize() {
	switch a.accountSaveType {
	case constant.AccountSaveTypeExcel:
		a.loadAccountFromExcel()
	case constant.AccountSaveTypeDb:
		a.loadAccountFromDb()
	case constant.AccountSaveTypeCookieBridge:
		a.loadAccountsFromCookieBridge()
	default:
		panic(fmt.Sprintf("不支持的账号池保存类型: %s", a.accountSaveType))
	}
}

func (a *AccountPool) loadAccountFromExcel() {
	// todo load account from excel
}

// loadAccountFromDb load account from db
func (a *AccountPool) loadAccountFromDb() {
	cookiesAccounts, err := a.accountCookiesRepo.ListCookiesAccountByPlatformAndStatus(a.platformName, constant.AccountStatusNormal)
	if err != nil {
		panic(fmt.Sprintf("加载账号池失败: %s", err.Error()))
	}
	for _, v := range cookiesAccounts {
		a.addCount(&models.AccountInfoModel{
			Id:                v.ID,
			AccountName:       v.AccountName,
			Cookies:           v.Cookies,
			PlatformName:      v.PlatformName,
			Status:            int(v.Status),
			InvalidTimeStamps: v.InvalidTimeStamp,
		})
	}
}

type cookieBridgeAccountsResponse struct {
	IsOk bool   `json:"isok"`
	Msg  string `json:"msg"`
	Data struct {
		Accounts []struct {
			ClientID  string `json:"client_id"`
			Platforms map[string]struct {
				HasCookies bool `json:"has_cookies"`
			} `json:"platforms"`
			Nicknames map[string]string `json:"nicknames"`
		} `json:"accounts"`
	} `json:"data"`
}

type cookieBridgeCookieResponse struct {
	IsOk bool   `json:"isok"`
	Msg  string `json:"msg"`
	Data struct {
		Cookies string `json:"cookies"`
	} `json:"data"`
}

func (a *AccountPool) loadAccountsFromCookieBridge() {
	bridgeURL := strings.TrimRight(a.cookieBridgeURL, "/")
	client := httpclient.NewStandardClient(10 * time.Second)

	var accountsResp cookieBridgeAccountsResponse
	if err := getJSON(client, bridgeURL+"/api/accounts", &accountsResp); err != nil {
		panic(fmt.Sprintf("加载CookieBridge账号失败: %s", err.Error()))
	}
	if !accountsResp.IsOk {
		panic(fmt.Sprintf("CookieBridge返回错误: %s", accountsResp.Msg))
	}

	accountID := int64(1)
	for _, item := range accountsResp.Data.Accounts {
		platformInfo, ok := item.Platforms[a.platformName]
		if !ok || !platformInfo.HasCookies {
			continue
		}

		cookieURL, err := url.Parse(bridgeURL + "/api/cookies/" + a.platformName)
		if err != nil {
			panic(err)
		}
		query := cookieURL.Query()
		query.Set("client_id", item.ClientID)
		cookieURL.RawQuery = query.Encode()

		var cookieResp cookieBridgeCookieResponse
		if err := getJSON(client, cookieURL.String(), &cookieResp); err != nil || !cookieResp.IsOk || cookieResp.Data.Cookies == "" {
			continue
		}

		accountName := item.Nicknames[a.platformName]
		if accountName == "" {
			accountName = item.ClientID
		}
		a.addCount(&models.AccountInfoModel{
			Id:           accountID,
			AccountName:  accountName,
			Cookies:      cookieResp.Data.Cookies,
			PlatformName: a.platformName,
			Status:       constant.AccountStatusNormal,
		})
		accountID++
	}
}

func getJSON(client *http.Client, requestURL string, target any) error {
	resp, err := client.Get(requestURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

// addCount add account count
func (a *AccountPool) addCount(accountInfoModel *models.AccountInfoModel) {
	a.accountList = append(a.accountList, accountInfoModel)
}

// GetActiveAccount get active account
func (a *AccountPool) GetActiveAccount() (*models.AccountInfoModel, error) {
	for len(a.accountList) > 0 {
		account := a.accountList[0]
		if account.Status == constant.AccountStatusNormal {
			return account, nil
		}
		a.accountList = a.accountList[1:]
	}
	return nil, errors.New("账号池中没有可用账号")
}

// UpdateAccountStatus get account list
func (a *AccountPool) UpdateAccountStatus(account *models.AccountInfoModel, status int) {
	for _, v := range a.accountList {
		if v.Id == account.Id {
			v.Status = status
			// todo update account status in db
			return
		}
	}
}

type AccountWithIpPool struct {
	AccountPool *AccountPool
	ProxyIpPool *proxy.IpPool
}

func NewAccountWithIpPool(accountPool *AccountPool) *AccountWithIpPool {
	return &AccountWithIpPool{
		AccountPool: accountPool,
	}
}

// GetAccountWithIpInfo get account with ip info
func (a *AccountWithIpPool) GetAccountWithIpInfo() (*models.AccountWithIpPoolModel, error) {
	account, err := a.AccountPool.GetActiveAccount()
	if err != nil {
		return nil, err
	}

	// todo get ip info
	return &models.AccountWithIpPoolModel{
		Account: account,
	}, nil
}

// MarkAccountInvalid mark account invalid
func (a *AccountWithIpPool) MarkAccountInvalid(account *models.AccountWithIpPoolModel) {
	a.AccountPool.UpdateAccountStatus(account.Account, constant.AccountStatusInvalid)
}

// MarkIpInvalid mark ip invalid
func (a *AccountWithIpPool) MarkIpInvalid(ipInfo *models.IpInfoModel) {
	if ipInfo != nil {
		a.ProxyIpPool.MarkIpInvalid(ipInfo)
	}
}
