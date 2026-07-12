// Copyright (c) 2025 relakkes@gmail.com
//
// This file is part of MediaCrawlerPro-Golang project.
// Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/blob/main/internal/weibo/store_mysql_test.go
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

package weibo

import (
	"context"
	"testing"

	"github.com/MediaCrawlerPro/MediaCrawlerPro-Golang/internal/e2e"
	"github.com/stretchr/testify/require"
	mysqlgorm "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestWeiboDBStoreHotSearchSharedTablesUpsertOnMySQL(t *testing.T) {
	mysqlServer := e2e.StartMySQL(t, e2e.MySQLOptions{Database: "media_crawler_pro"})
	db, err := gorm.Open(mysqlgorm.Open(mysqlServer.DSN()), &gorm.Config{})
	require.NoError(t, err)

	store := NewDBStore(db)
	require.NoError(t, store.AutoMigrate())
	ctx := context.Background()

	firstHotItem := testWeiboHotSearchItem("wb_hot_item_mysql", func(item *HotSearchItem) {
		item.Title = "first mysql hot"
		item.Keyword = "first keyword"
		item.HotValue = "100"
		item.RawJSON = `{"title":"first mysql hot"}`
		item.AddTs = 100
		item.LastModifyTs = 100
	})
	require.NoError(t, store.SaveHotSearchItem(ctx, firstHotItem))

	secondHotItem := testWeiboHotSearchItem("wb_hot_item_mysql", func(item *HotSearchItem) {
		item.Title = "second mysql hot"
		item.Keyword = "second keyword"
		item.HotValue = "200"
		item.RawJSON = `{"title":"second mysql hot"}`
		item.AddTs = 200
		item.LastModifyTs = 200
	})
	require.NoError(t, store.SaveHotSearchItem(ctx, secondHotItem))

	var hotItems []HotSearchItem
	require.NoError(t, db.Where("hot_item_id = ?", "wb_hot_item_mysql").Find(&hotItems).Error)
	require.Len(t, hotItems, 1)
	require.Equal(t, "second mysql hot", hotItems[0].Title)
	require.Equal(t, "second keyword", hotItems[0].Keyword)
	require.Equal(t, "200", hotItems[0].HotValue)
	require.Equal(t, `{"title":"second mysql hot"}`, hotItems[0].RawJSON)
	require.Equal(t, int64(100), hotItems[0].AddTs)
	require.Equal(t, int64(200), hotItems[0].LastModifyTs)

	firstRelation := testWeiboHotSearchRelation("wb_hot_rel_mysql", func(relation *HotSearchContentRelation) {
		relation.HotItemID = "wb_hot_item_mysql"
		relation.RankNo = 1
		relation.SourceKeyword = "first keyword"
		relation.AddTs = 100
		relation.LastModifyTs = 100
	})
	require.NoError(t, store.SaveHotSearchRelation(ctx, firstRelation))

	secondRelation := testWeiboHotSearchRelation("wb_hot_rel_mysql", func(relation *HotSearchContentRelation) {
		relation.HotItemID = "wb_hot_item_mysql"
		relation.RankNo = 2
		relation.SourceKeyword = "second keyword"
		relation.AddTs = 200
		relation.LastModifyTs = 200
	})
	require.NoError(t, store.SaveHotSearchRelation(ctx, secondRelation))

	var relations []HotSearchContentRelation
	require.NoError(t, db.Where("rel_id = ?", "wb_hot_rel_mysql").Find(&relations).Error)
	require.Len(t, relations, 1)
	require.Equal(t, "wb_hot_item_mysql", relations[0].HotItemID)
	require.Equal(t, 2, relations[0].RankNo)
	require.Equal(t, "second keyword", relations[0].SourceKeyword)
	require.Equal(t, int64(100), relations[0].AddTs)
	require.Equal(t, int64(200), relations[0].LastModifyTs)
}
