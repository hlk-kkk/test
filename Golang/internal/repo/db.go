// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/repo/db.go
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

package repo

import (
	"fmt"
	"sync"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB
var dbMu sync.Mutex

// NewDB new db
func NewDB() *gorm.DB {
	gConfig := config.GetConfig()
	mysqlConnectDsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		gConfig.DbConfig.RelationDbUser,
		gConfig.DbConfig.RelationDbPwd,
		gConfig.DbConfig.RelationDbHost,
		gConfig.DbConfig.RelationDbPort,
		gConfig.DbConfig.RelationDbName,
	)
	db, err := gorm.Open(mysql.Open(mysqlConnectDsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return db
}

// GetDB get db
func GetDB() *gorm.DB {
	dbMu.Lock()
	defer dbMu.Unlock()
	if db == nil {
		db = NewDB()
	}
	return db
}
