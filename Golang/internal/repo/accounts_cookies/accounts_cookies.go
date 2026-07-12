// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/repo/accounts_cookies/accounts_cookies.go
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

package accounts_cookies

import (
	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/pkg/tools"
	"gorm.io/gorm"
)

type CrawlerCookiesAccount struct {
	ID               int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	AccountName      string `json:"account_name" gorm:"column:account_name"`
	PlatformName     string `json:"platform_name" gorm:"column:platform_name"`
	Cookies          string `json:"cookies" gorm:"column:cookies"`
	CreateTime       string `json:"create_time" gorm:"column:create_time"`
	UpdateTime       string `json:"update_time" gorm:"column:update_time"`
	InvalidTimeStamp int64  `json:"invalid_time_stamp" gorm:"column:invalid_time_stamp"`
	Status           uint8  `json:"status" gorm:"column:status"`
}

// TableName set table name
func (CrawlerCookiesAccount) TableName() string {
	return "crawler_cookies_account"
}

type CookiesAccountRepo struct {
	db     *gorm.DB
	logger *tools.Logger
}

// NewCrawlerCookiesAccountRepo new a data repo
func NewCrawlerCookiesAccountRepo(db *gorm.DB, logger *tools.Logger) *CookiesAccountRepo {
	return &CookiesAccountRepo{
		db:     db,
		logger: logger,
	}
}

// ListCookiesAccountByStatus list all crawler cookies account
func (c *CookiesAccountRepo) ListCookiesAccountByStatus(status int) ([]*CrawlerCookiesAccount, error) {
	var accounts []*CrawlerCookiesAccount
	txDb := c.db.Model(&CrawlerCookiesAccount{}).Where("status = ?", status).Find(&accounts)
	if txDb.Error != nil {
		c.logger.Error("ListCookiesAccountByStatus failed", txDb.Error)
		return nil, txDb.Error
	}
	return accounts, nil
}

// ListCookiesAccountByPlatformAndStatus list crawler cookies account by platform and status
func (c *CookiesAccountRepo) ListCookiesAccountByPlatformAndStatus(platformName string, status int) ([]*CrawlerCookiesAccount, error) {
	var accounts []*CrawlerCookiesAccount
	txDb := c.db.Model(&CrawlerCookiesAccount{}).Where("platform_name = ? AND status = ?", platformName, status).Find(&accounts)
	if txDb.Error != nil {
		c.logger.Error("ListCookiesAccountByPlatformAndStatus failed", txDb.Error)
		return nil, txDb.Error
	}
	return accounts, nil
}
